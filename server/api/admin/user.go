package admin

import (
	"errors"
	"net/http"
	"strconv"

	"Power-Monitor/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerUserRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", getUsers)
		users.GET("/:id", getUserByID)
		users.POST("", createUser)
		users.PUT("/:id", updateUser)
		users.DELETE("/:id", deleteUser)
	}
}

func getUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var users []model.User
	var total int64

	query := model.DB.Model(&model.User{})

	// Search filter
	if search := c.Query("search"); search != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Role filter
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}

	// Count total
	query.Count(&total)

	// Get users with pagination
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	// Remove password from response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, model.ListResponse{
		Data: users,
		Pagination: model.Pagination{
			Total:    total,
			Current:  page,
			PageSize: pageSize,
		},
	})
}

func getUserByID(c *gin.Context) {
	id := c.Param("id")

	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		}
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func createUser(c *gin.Context) {
	var req model.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check if username already exists
	var existingUser model.User
	if err := model.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	// Check if email already exists
	if req.Email != "" {
		if err := model.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
			return
		}
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		Active:   true,
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := model.DB.Create(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User created successfully",
		"data":    user,
	})
}

func updateUser(c *gin.Context) {
	id := c.Param("id")

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		}
		return
	}

	// Update fields
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	user.Active = req.Active

	if err := model.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
		"data":    user,
	})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	// Prevent deleting the current user
	currentUserID := c.GetUint("user_id")
	userID, _ := strconv.Atoi(id)
	if uint(userID) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	if err := model.DB.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}
