package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	BaseModel
	Username    string    `gorm:"uniqueIndex;not null" json:"username" binding:"required"`
	Email       string    `gorm:"uniqueIndex" json:"email" binding:"email"`
	Password    string    `gorm:"not null" json:"-"`
	FullName    string    `json:"full_name"`
	Role        string    `gorm:"default:user" json:"role"` // admin, user
	Active      bool      `gorm:"default:true" json:"active"`
	LastLoginAt time.Time `json:"last_login_at"`
	Avatar      string    `json:"avatar"`
}

// UserCreateRequest represents request for creating a user
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// UserUpdateRequest represents request for updating a user
type UserUpdateRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// HashPassword hashes the user password
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password is correct
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// SafeUser returns user data without sensitive information
func (u *User) SafeUser() User {
	safeUser := *u
	safeUser.Password = ""
	return safeUser
}
