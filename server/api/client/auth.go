package client

import (
	"net/http"
	"time"

	"Power-Monitor/internal/auth"
	"Power-Monitor/model"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers authentication routes for both admin and client
func RegisterAuthRoutes(r *gin.RouterGroup) {
	r.POST("/login", login)
	r.POST("/refresh", refreshToken)
}

// RegisterAuthGuardRoutes registers user management routes that require JWT authentication
func RegisterAuthGuardRoutes(r *gin.RouterGroup) {
	r.POST("/logout", logout)
	r.GET("/profile", getProfile)
	r.PUT("/profile", updateProfile)
	r.POST("/change-password", changePassword)
}

func login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Find user by username
	var user model.User
	if err := model.DB.Where("username = ? AND active = ?", req.Username, true).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate token pair
	jwtService := auth.GetJWTService()
	tokenPair, err := jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	// Save refresh token
	refreshToken := &model.AuthToken{
		UserID:    user.ID,
		Token:     auth.GenerateSecureToken(),
		TokenType: "refresh",
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7), // 7 days
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	model.DB.Create(refreshToken)

	// Update last login time
	user.LastLoginAt = time.Now()
	model.DB.Save(&user)

	response := model.LoginResponse{
		User:         user.SafeUser(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"data":    response,
	})
}

func refreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	jwtService := auth.GetJWTService()
	tokenPair, err := jwtService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tokenPair,
	})
}

func logout(c *gin.Context) {
	// Revoke all tokens for this user
	userID := c.GetUint("user_id")
	model.DB.Model(&model.AuthToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

func getProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user.SafeUser(),
	})
}

func updateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update allowed fields (role cannot be changed by user)
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}

	if err := model.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile updated successfully",
		"data":    user.SafeUser(),
	})
}

func changePassword(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check current password
	if !user.CheckPassword(req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	if err := user.HashPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := model.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}
