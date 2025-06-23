package admin

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"Power-Monitor/internal/auth"
	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerCollectorRoutes(r *gin.RouterGroup) {
	// Collector management routes
	collectors := r.Group("/collectors")
	{
		collectors.GET("", getCollectors)
		collectors.GET("/:id", getCollectorByID)
		collectors.POST("", createCollector)
		collectors.PUT("/:id", updateCollector)
		collectors.DELETE("/:id", deleteCollector)
		collectors.GET("/:id/status", getCollectorStatus)
		collectors.POST("/:id/config", updateCollectorConfig)
	}

	// Registration codes
	regCodes := r.Group("/registration-codes")
	{
		regCodes.GET("", getRegistrationCodes)
		regCodes.POST("", createRegistrationCode)
		regCodes.DELETE("/:id", deleteRegistrationCode)
	}
}

func getCollectors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var collectors []model.Collector
	var total int64

	query := model.DB.Model(&model.Collector{}).Preload("User")

	// Search filter
	if search := c.Query("search"); search != "" {
		query = query.Where("collector_id LIKE ? OR name LIKE ? OR location LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Status filter
	if status := c.Query("status"); status != "" {
		isActive := status == "active"
		query = query.Where("is_active = ?", isActive)
	}

	// Count total
	query.Count(&total)

	// Get collectors with pagination
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&collectors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collectors"})
		return
	}

	// Hide tokens and add online status
	for i := range collectors {
		collectors[i].Token = ""
		// Add online status based on last seen time
		// This could be enhanced with real-time status tracking
	}

	c.JSON(http.StatusOK, model.ListResponse{
		Data: collectors,
		Pagination: model.Pagination{
			Total:    total,
			Current:  page,
			PageSize: pageSize,
		},
	})
}

func getCollectorByID(c *gin.Context) {
	id := c.Param("id")

	var collector model.Collector
	if err := model.DB.Preload("User").First(&collector, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	collector.Token = ""
	c.JSON(http.StatusOK, gin.H{"data": collector})
}

func createCollector(c *gin.Context) {
	var req model.CollectorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check if collector ID already exists
	var existingCollector model.Collector
	if err := model.DB.Where("collector_id = ?", req.CollectorID).First(&existingCollector).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collector ID already exists"})
		return
	}

	// Generate collector token (static token)
	token := auth.GenerateSecureToken()

	currentUserID := c.GetUint("user_id")
	collector := &model.Collector{
		CollectorID: req.CollectorID,
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		IsActive:    true,
		Token:       token,
		UserID:      currentUserID,
	}

	if err := model.DB.Create(collector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collector"})
		return
	}

	// Create default configuration
	config := &model.CollectorConfig{
		CollectorID:      req.CollectorID,
		SampleInterval:   15,
		UploadInterval:   60,
		MaxCacheSize:     1000,
		AutoUpload:       true,
		CompressionLevel: 6,
	}
	model.DB.Create(config)

	collector.Token = ""
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Collector created successfully",
		"data":    collector,
	})
}

func updateCollector(c *gin.Context) {
	id := c.Param("id")

	var req model.CollectorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var collector model.Collector
	if err := model.DB.First(&collector, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	// Update fields
	if req.Name != "" {
		collector.Name = req.Name
	}
	if req.Description != "" {
		collector.Description = req.Description
	}
	if req.Location != "" {
		collector.Location = req.Location
	}
	collector.IsActive = req.IsActive

	if err := model.DB.Save(&collector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collector"})
		return
	}

	collector.Token = ""
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Collector updated successfully",
		"data":    collector,
	})
}

func deleteCollector(c *gin.Context) {
	id := c.Param("id")

	// Get collector info before deletion
	var collector model.Collector
	if err := model.DB.First(&collector, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	// Handle existing power data based on configuration
	archiveData := c.DefaultQuery("archive_data", "false") == "true"

	if archiveData {
		// Archive data by marking it as archived instead of deleting
		if err := model.DB.Model(&model.PowerData{}).
			Where("collector_id = ?", collector.CollectorID).
			Update("archived", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive power data"})
			return
		}
	} else {
		// Delete all associated power data
		if err := model.DB.Where("collector_id = ?", collector.CollectorID).
			Delete(&model.PowerData{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete power data"})
			return
		}
	}

	// Delete collector configuration
	model.DB.Where("collector_id = ?", collector.CollectorID).Delete(&model.CollectorConfig{})

	// Delete the collector
	if err := model.DB.Delete(&collector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collector"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Collector deleted successfully",
		"data": gin.H{
			"archived_data": archiveData,
		},
	})
}

func getCollectorStatus(c *gin.Context) {
	id := c.Param("id")

	var collector model.Collector
	if err := model.DB.First(&collector, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	// Get latest data timestamp
	var lastDataTime time.Time
	var dataCount int64

	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ?", collector.CollectorID).
		Count(&dataCount)

	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ?", collector.CollectorID).
		Select("MAX(timestamp)").
		Scan(&lastDataTime)

	status := model.CollectorStatusResponse{
		Collector:    collector,
		IsOnline:     collector.IsOnline(),
		LastDataTime: lastDataTime,
		DataCount:    dataCount,
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

func updateCollectorConfig(c *gin.Context) {
	id := c.Param("id")

	var collector model.Collector
	if err := model.DB.First(&collector, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	var req model.CollectorConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	req.CollectorID = collector.CollectorID

	// Update or create config
	var config model.CollectorConfig
	if err := model.DB.Where("collector_id = ?", collector.CollectorID).First(&config).Error; err != nil {
		// Create new config
		if err := model.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config"})
			return
		}
		config = req
	} else {
		// Update existing config
		config.SampleInterval = req.SampleInterval
		config.UploadInterval = req.UploadInterval
		config.MaxCacheSize = req.MaxCacheSize
		config.AutoUpload = req.AutoUpload
		config.CompressionLevel = req.CompressionLevel

		if err := model.DB.Save(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Collector config updated successfully",
		"data":    config,
	})
}

// Registration Code Management

func getRegistrationCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var codes []model.RegistrationCode
	var total int64

	query := model.DB.Model(&model.RegistrationCode{}).Preload("User")

	// Filter by usage status
	if used := c.Query("used"); used != "" {
		isUsed := used == "true"
		query = query.Where("is_used = ?", isUsed)
	}

	// Count total
	query.Count(&total)

	// Get codes with pagination
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get registration codes"})
		return
	}

	c.JSON(http.StatusOK, model.ListResponse{
		Data: codes,
		Pagination: model.Pagination{
			Total:    total,
			Current:  page,
			PageSize: pageSize,
		},
	})
}

func createRegistrationCode(c *gin.Context) {
	var req struct {
		Description string `json:"description"`
		ExpiresAt   string `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	expiresAt := time.Now().Add(settings.CollectorSettings.RegistrationCodeExpires)
	if req.ExpiresAt != "" {
		if parsedTime, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			expiresAt = parsedTime
		}
	}

	currentUserID := c.GetUint("user_id")
	regCode := &model.RegistrationCode{
		Code:        auth.GenerateRegistrationCode(),
		Description: req.Description,
		ExpiresAt:   expiresAt,
		UserID:      currentUserID,
	}

	if err := model.DB.Create(regCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create registration code"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Registration code created successfully",
		"data":    regCode,
	})
}

func deleteRegistrationCode(c *gin.Context) {
	id := c.Param("id")

	if err := model.DB.Delete(&model.RegistrationCode{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete registration code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Registration code deleted successfully",
	})
}
