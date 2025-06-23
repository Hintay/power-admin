package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService handles JWT token operations
type JWTService struct {
	secretKey              string
	accessTokenExpiration  time.Duration
	refreshTokenExpiration time.Duration
}

// Claims represents JWT claims
type Claims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // access, refresh, collector
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var jwtService *JWTService

// Init initializes the JWT service
func Init(secretKey string, accessExpiration, refreshExpiration time.Duration) {
	jwtService = &JWTService{
		secretKey:              secretKey,
		accessTokenExpiration:  accessExpiration,
		refreshTokenExpiration: refreshExpiration,
	}
}

// GetJWTService returns the global JWT service instance
func GetJWTService() *JWTService {
	return jwtService
}

// GenerateTokenPair generates access and refresh token pair for a user
func (j *JWTService) GenerateTokenPair(userID uint, username, role string) (*TokenPair, error) {
	if j == nil {
		return nil, fmt.Errorf("JWT service not initialized")
	}

	now := time.Now()

	// Generate access token
	accessClaims := &Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "power-monitor",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "power-monitor",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(j.accessTokenExpiration),
	}, nil
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	if j == nil {
		return nil, fmt.Errorf("JWT service not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new access token from a valid refresh token
func (j *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	// Generate new token pair
	return j.GenerateTokenPair(claims.UserID, claims.Username, claims.Role)
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token
		return fmt.Sprintf("token_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
	}
	return hex.EncodeToString(bytes)
}

// GenerateRegistrationCode generates a registration code for collectors
func GenerateRegistrationCode() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based code
		return fmt.Sprintf("REG_%d", time.Now().Unix())
	}
	return fmt.Sprintf("REG_%s", hex.EncodeToString(bytes)[:16])
}
