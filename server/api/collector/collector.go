package collector

import (
	"errors"
	"net/http"
	"time"

	"Power-Monitor/internal/auth"
	"Power-Monitor/internal/influxdb"
	"Power-Monitor/internal/realtime"
	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/uozi-tech/cosy/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers collector API routes
func RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/data", uploadPowerData)
	r.POST("/data/batch", uploadPowerDataBatch)
	r.GET("/config", getCollectorConfig)
	r.POST("/heartbeat", heartbeat)
}

// RegisterAuthRoutes registers authentication routes for collectors
func RegisterAuthRoutes(r *gin.RouterGroup) {
	collector := r.Group("/collector")
	collector.POST("/register", registerCollector)
}

// uploadPowerData handles single power data upload from collector
func uploadPowerData(c *gin.Context) {
	collectorID := c.GetString("collector_id")
	if collectorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid collector token"})
		return
	}

	var req model.PowerDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Save to database for quick access
	powerData := &model.PowerData{
		CollectorID: collectorID,
		Timestamp:   req.Timestamp,
		Voltage:     req.Voltage,
		Current:     req.Current,
		Power:       req.Power,
		Energy:      req.Energy,
		Frequency:   req.Frequency,
		PowerFactor: req.PowerFactor,
	}

	if err := model.DB.Create(powerData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}

	// Save to InfluxDB for time-series analytics
	if settings.InfluxDBSettings.Enabled {
		influxClient := influxdb.GetClient()
		if influxClient != nil {
			influxData := influxdb.PowerDataPoint{
				CollectorID: collectorID,
				Timestamp:   req.Timestamp,
				Voltage:     req.Voltage,
				Current:     req.Current,
				Power:       req.Power,
				Energy:      req.Energy,
				Frequency:   req.Frequency,
				PowerFactor: req.PowerFactor,
			}

			if err := influxClient.WritePowerData(influxData); err != nil {
				// Log error but don't fail the request
				logger.Errorf("Failed to write to InfluxDB: %v", err)
			}
		}
	}

	// Update collector last seen time
	updateCollectorLastSeen(collectorID, c.ClientIP())

	// Broadcast real-time data
	realtimeData := realtime.PowerDataMessage{
		CollectorID: collectorID,
		Timestamp:   req.Timestamp,
		Voltage:     req.Voltage,
		Current:     req.Current,
		Power:       req.Power,
		Energy:      req.Energy,
		Frequency:   req.Frequency,
		PowerFactor: req.PowerFactor,
	}
	realtime.BroadcastPowerData(realtimeData)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data uploaded successfully",
	})
}

// uploadPowerDataBatch handles batch power data upload from collector
func uploadPowerDataBatch(c *gin.Context) {
	collectorID := c.GetString("collector_id")
	if collectorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid collector token"})
		return
	}

	var req model.PowerDataUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate collector ID matches token
	if req.CollectorID != collectorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Collector ID mismatch"})
		return
	}

	// Prepare data for batch insert
	var powerDataList []model.PowerData
	for _, data := range req.Data {
		powerData := model.PowerData{
			CollectorID: collectorID,
			Timestamp:   data.Timestamp,
			Voltage:     data.Voltage,
			Current:     data.Current,
			Power:       data.Power,
			Energy:      data.Energy,
			Frequency:   data.Frequency,
			PowerFactor: data.PowerFactor,
		}
		powerDataList = append(powerDataList, powerData)
	}

	// Batch insert to the database
	if err := model.DB.CreateInBatches(powerDataList, 100).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save batch data"})
		return
	}

	// Batch insert to InfluxDB
	if settings.InfluxDBSettings.Enabled {
		influxClient := influxdb.GetClient()
		if influxClient != nil {
			var influxDataList []influxdb.PowerDataPoint
			for _, data := range req.Data {
				influxData := influxdb.PowerDataPoint{
					CollectorID: collectorID,
					Timestamp:   data.Timestamp,
					Voltage:     data.Voltage,
					Current:     data.Current,
					Power:       data.Power,
					Energy:      data.Energy,
					Frequency:   data.Frequency,
					PowerFactor: data.PowerFactor,
				}
				influxDataList = append(influxDataList, influxData)
			}
			if err := influxClient.WritePowerDataBatch(influxDataList); err != nil {
				// Log error but don't fail the request
				logger.Errorf("Failed to write batch to InfluxDB: %v", err)
			}
		}
	}

	// Update collector last seen time
	updateCollectorLastSeen(collectorID, c.ClientIP())

	// Broadcast the latest data point
	if len(req.Data) > 0 {
		latestData := req.Data[len(req.Data)-1]
		realtimeData := realtime.PowerDataMessage{
			CollectorID: collectorID,
			Timestamp:   latestData.Timestamp,
			Voltage:     latestData.Voltage,
			Current:     latestData.Current,
			Power:       latestData.Power,
			Energy:      latestData.Energy,
			Frequency:   latestData.Frequency,
			PowerFactor: latestData.PowerFactor,
		}
		realtime.BroadcastPowerData(realtimeData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Batch data uploaded successfully",
		"count":   len(req.Data),
	})
}

// getCollectorConfig returns configuration for the collector
func getCollectorConfig(c *gin.Context) {
	collectorID := c.GetString("collector_id")
	if collectorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid collector token"})
		return
	}

	var config model.CollectorConfig
	if err := model.DB.Where("collector_id = ?", collectorID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default config
			config = model.CollectorConfig{
				CollectorID:      collectorID,
				SampleInterval:   15,
				UploadInterval:   60,
				MaxCacheSize:     1000,
				AutoUpload:       true,
				CompressionLevel: 6,
			}
			model.DB.Create(&config)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// heartbeat handles collector heartbeat
func heartbeat(c *gin.Context) {
	collectorID := c.GetString("collector_id")
	if collectorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid collector token"})
		return
	}

	updateCollectorLastSeen(collectorID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Heartbeat received",
		"timestamp": time.Now(),
	})
}

// registerCollector handles collector registration
func registerCollector(c *gin.Context) {
	var req model.CollectorRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate registration code
	var regCode model.RegistrationCode
	if err := model.DB.Where("code = ? AND is_used = ? AND expires_at > ?",
		req.RegistrationCode, false, time.Now()).First(&regCode).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired registration code"})
		return
	}

	// Check if collector already exists
	var existingCollector model.Collector
	if err := model.DB.Where("collector_id = ?", req.CollectorID).First(&existingCollector).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collector already registered"})
		return
	}

	// Generate collector token (static token)
	token := auth.GenerateSecureToken()

	// Create collector
	collector := &model.Collector{
		CollectorID: req.CollectorID,
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		IsActive:    true,
		LastSeenAt:  time.Now(),
		Token:       token,
		Version:     req.Version,
		IPAddress:   c.ClientIP(),
		UserID:      regCode.UserID,
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

	if err := model.DB.Create(config).Error; err != nil {
		// Not critical, just log
		logger.Errorf("Failed to create collector config: %v", err)
	}

	// Mark registration code as used
	regCode.IsUsed = true
	regCode.UsedBy = req.CollectorID
	model.DB.Save(&regCode)

	response := model.CollectorRegisterResponse{
		Token:  token,
		Config: *config,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Collector registered successfully",
		"data":    response,
	})
}

// updateCollectorLastSeen updates the collector's last seen time and IP
func updateCollectorLastSeen(collectorID, ipAddress string) {
	model.DB.Model(&model.Collector{}).
		Where("collector_id = ?", collectorID).
		Updates(map[string]interface{}{
			"last_seen_at": time.Now(),
			"ip_address":   ipAddress,
		})
}
