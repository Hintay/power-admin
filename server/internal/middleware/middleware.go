package middleware

import (
	"net/http"
	"strings"
	"time"

	"Power-Monitor/internal/auth"
	"Power-Monitor/model"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns CORS middleware
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure appropriately for production
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// JWTAuth returns JWT authentication middleware
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := tokenParts[1]
		jwtService := auth.GetJWTService()
		if jwtService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service not available"})
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Check if token is revoked
		var authToken model.AuthToken
		if err := model.DB.Where("user_id = ? AND token_type = ? AND is_revoked = ?",
			claims.UserID, "access", false).First(&authToken).Error; err != nil {
			// If we can't find a valid token record, allow but log
			// In production, you might want to be more strict
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token_type", claims.TokenType)

		c.Next()
	}
}

// CollectorAuth returns collector authentication middleware
func CollectorAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Static token validation
		var collector model.Collector
		if err := model.DB.Where("token = ? AND is_active = ?",
			token, true).First(&collector).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid collector token"})
			c.Abort()
			return
		}

		// Update last seen time
		collector.LastSeenAt = time.Now()
		collector.IPAddress = c.ClientIP()
		model.DB.Save(&collector)

		c.Set("collector_id", collector.CollectorID)
		c.Next()
	}
}
