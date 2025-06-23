package client

import (
	"net/http"
	"strconv"
	"time"

	"Power-Monitor/internal/influxdb"
	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerDataRoutes(r *gin.RouterGroup) {
	data := r.Group("/data")
	{
		data.GET("/collectors", getUserCollectors)
		data.GET("/collectors/:id", getCollectorInfo)
		data.GET("/collectors/:id/status", getCollectorStatus)
		data.GET("/collectors/:id/latest", getLatestData)
		data.GET("/collectors/:id/history", getHistoryData)
		data.GET("/collectors/:id/statistics", getDataStatistics)
		data.GET("/collectors/:id/data", getCollectorDataView)
		data.GET("/analytics", getPowerDataAnalytics)
	}
}

func getUserCollectors(c *gin.Context) {
	userID := c.GetUint("user_id")

	var collectors []model.Collector
	if err := model.DB.Where("user_id = ? AND is_active = ?", userID, true).Find(&collectors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collectors"})
		return
	}

	// Hide sensitive information
	for i := range collectors {
		collectors[i].Token = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    collectors,
	})
}

func getCollectorInfo(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")

	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	// Get configuration
	var config model.CollectorConfig
	model.DB.Where("collector_id = ?", collector.ID).First(&config)

	collector.Token = ""

	response := gin.H{
		"collector": collector,
		"config":    config,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func getCollectorStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")

	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
		Where("collector_id = ?", collector.ID).
		Count(&dataCount)

	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ?", collector.ID).
		Select("MAX(timestamp)").
		Scan(&lastDataTime)

	status := model.CollectorStatusResponse{
		Collector:    collector,
		IsOnline:     collector.IsOnline(),
		LastDataTime: lastDataTime,
		DataCount:    dataCount,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func getLatestData(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")

	// Verify collector belongs to user
	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		return
	}

	// Get latest data from database
	var latestData model.PowerData
	if err := model.DB.Where("collector_id = ?", collector.ID).
		Order("timestamp DESC").First(&latestData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    nil,
				"message": "No data available",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get latest data"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    latestData,
	})
}

func getHistoryData(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")

	// Verify collector belongs to user
	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		return
	}

	// Parse query parameters
	startTimeStr := c.Query("start")
	endTimeStr := c.Query("end")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format"})
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format"})
			return
		}
	} else {
		endTime = time.Now()
	}

	// Use InfluxDB for time-series data if available
	if settings.InfluxDBSettings.Enabled {
		influxClient := influxdb.GetClient()
		if influxClient != nil {
			data, err := influxClient.QueryPowerData(collector.CollectorID, startTime, endTime)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    data,
					"source":  "influxdb",
				})
				return
			}
			// Fall back to database if InfluxDB fails
		}
	}

	// Query from database
	var data []model.PowerData
	query := model.DB.Where("collector_id = ? AND timestamp BETWEEN ? AND ?",
		collector.ID, startTime, endTime).
		Order("timestamp ASC").
		Limit(limit)

	if err := query.Find(&data).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"source":  "database",
		"count":   len(data),
	})
}

func getDataStatistics(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")

	// Verify collector belongs to user
	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		return
	}

	// Calculate statistics
	var stats struct {
		TotalDataPoints int64     `json:"total_data_points"`
		AvgPower        float64   `json:"avg_power"`
		MaxPower        float64   `json:"max_power"`
		MinPower        float64   `json:"min_power"`
		TotalEnergy     float64   `json:"total_energy"`
		LastUpdated     time.Time `json:"last_updated"`
	}

	// Get basic stats from database
	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ?", collector.ID).
		Count(&stats.TotalDataPoints)

	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ?", collector.ID).
		Select("AVG(power), MAX(power), MIN(power), MAX(energy), MAX(timestamp)").
		Row().Scan(&stats.AvgPower, &stats.MaxPower, &stats.MinPower, &stats.TotalEnergy, &stats.LastUpdated)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func getCollectorDataView(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("id")
	dataType := c.DefaultQuery("type", "overview") // overview, realtime, history
	period := c.DefaultQuery("period", "24h")

	// Verify collector belongs to user
	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ?", collectorID, userID).First(&collector).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collector"})
		}
		return
	}

	switch dataType {
	case "overview":
		// Get collector overview with statistics
		var stats struct {
			TotalDataPoints int64     `json:"total_data_points"`
			FirstDataTime   time.Time `json:"first_data_time"`
			LastDataTime    time.Time `json:"last_data_time"`
			AvgPower        float64   `json:"avg_power"`
			MaxPower        float64   `json:"max_power"`
			TotalEnergy     float64   `json:"total_energy"`
			IsOnline        bool      `json:"is_online"`
		}

		model.DB.Model(&model.PowerData{}).
			Where("collector_id = ?", collector.ID).
			Count(&stats.TotalDataPoints)

		model.DB.Model(&model.PowerData{}).
			Where("collector_id = ?", collector.ID).
			Select("MIN(timestamp), MAX(timestamp), AVG(power), MAX(power), SUM(energy)").
			Row().Scan(&stats.FirstDataTime, &stats.LastDataTime, &stats.AvgPower, &stats.MaxPower, &stats.TotalEnergy)

		stats.IsOnline = collector.IsOnline()

		// Get recent power trend (last 10 readings)
		var recentData []model.PowerData
		model.DB.Where("collector_id = ?", collector.ID).
			Order("timestamp DESC").
			Limit(10).
			Find(&recentData)

		response := gin.H{
			"collector":   collector,
			"statistics":  stats,
			"recent_data": recentData,
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    response,
		})

	case "realtime":
		// Get latest real-time data
		var latestData model.PowerData
		if err := model.DB.Where("collector_id = ?", collector.ID).
			Order("timestamp DESC").First(&latestData).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No data found for collector"})
			return
		}

		response := gin.H{
			"collector":   collector,
			"latest_data": latestData,
			"timestamp":   time.Now(),
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    response,
		})

	case "history":
		// Get historical data based on period
		var startTime time.Time
		switch period {
		case "1h":
			startTime = time.Now().Add(-1 * time.Hour)
		case "24h":
			startTime = time.Now().Add(-24 * time.Hour)
		case "7d":
			startTime = time.Now().Add(-7 * 24 * time.Hour)
		case "30d":
			startTime = time.Now().Add(-30 * 24 * time.Hour)
		default:
			startTime = time.Now().Add(-24 * time.Hour)
		}

		var historyData []model.PowerData
		limit := 1000 // Limit to prevent too much data

		model.DB.Where("collector_id = ? AND timestamp >= ?", collector.ID, startTime).
			Order("timestamp ASC").
			Limit(limit).
			Find(&historyData)

		response := gin.H{
			"collector": collector,
			"data":      historyData,
			"period":    period,
			"count":     len(historyData),
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    response,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data type"})
	}
}

func getPowerDataAnalytics(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "24h") // 24h, 7d, 30d
	collectorID := c.Query("collector_id")

	// Parse period and calculate start time
	var startTime time.Time
	var groupBy string
	switch period {
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
		groupBy = "hour"
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
		groupBy = "day"
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
		groupBy = "day"
	default:
		startTime = time.Now().Add(-24 * time.Hour)
		groupBy = "hour"
	}

	// Get user's collectors
	var collectorIDs []string
	query := model.DB.Model(&model.Collector{}).Where("user_id = ? AND is_active = ?", userID, true)

	// If specific collector requested, verify it belongs to user
	if collectorID != "" {
		query = query.Where("collector_id = ?", collectorID)
	}

	query.Pluck("collector_id", &collectorIDs)

	if len(collectorIDs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No collectors found or collector doesn't belong to user"})
		return
	}

	// Get aggregated power data
	var powerTrends []struct {
		Period      string  `json:"period"`
		AvgPower    float64 `json:"avg_power"`
		MaxPower    float64 `json:"max_power"`
		MinPower    float64 `json:"min_power"`
		TotalEnergy float64 `json:"total_energy"`
		DataPoints  int64   `json:"data_points"`
	}

	var rawQuery string
	if groupBy == "hour" {
		rawQuery = `
			SELECT 
				TO_CHAR(timestamp, 'YYYY-MM-DD HH24:00:00') as period,
				AVG(power) as avg_power,
				MAX(power) as max_power,
				MIN(power) as min_power,
				SUM(energy) as total_energy,
				COUNT(*) as data_points
			FROM power_data 
			WHERE collector_id = ANY($1) AND timestamp >= $2
			GROUP BY TO_CHAR(timestamp, 'YYYY-MM-DD HH24:00:00')
			ORDER BY period`
	} else {
		rawQuery = `
			SELECT 
				TO_CHAR(timestamp, 'YYYY-MM-DD') as period,
				AVG(power) as avg_power,
				MAX(power) as max_power,
				MIN(power) as min_power,
				SUM(energy) as total_energy,
				COUNT(*) as data_points
			FROM power_data 
			WHERE collector_id = ANY($1) AND timestamp >= $2
			GROUP BY TO_CHAR(timestamp, 'YYYY-MM-DD')
			ORDER BY period`
	}

	model.DB.Raw(rawQuery, collectorIDs, startTime).Scan(&powerTrends)

	// Calculate summary statistics
	var summary struct {
		TotalDataPoints int64   `json:"total_data_points"`
		AvgPower        float64 `json:"avg_power"`
		MaxPower        float64 `json:"max_power"`
		MinPower        float64 `json:"min_power"`
		TotalEnergy     float64 `json:"total_energy"`
		PeakHour        string  `json:"peak_hour"`
		LowestHour      string  `json:"lowest_hour"`
	}

	// Get summary stats for user's collectors only
	model.DB.Model(&model.PowerData{}).
		Where("collector_id = ANY(?) AND timestamp >= ?", collectorIDs, startTime).
		Select("COUNT(*), AVG(power), MAX(power), MIN(power), SUM(energy)").
		Row().Scan(&summary.TotalDataPoints, &summary.AvgPower, &summary.MaxPower, &summary.MinPower, &summary.TotalEnergy)

	// Find peak and lowest consumption periods
	if len(powerTrends) > 0 {
		maxAvg := powerTrends[0].AvgPower
		minAvg := powerTrends[0].AvgPower
		summary.PeakHour = powerTrends[0].Period
		summary.LowestHour = powerTrends[0].Period

		for _, trend := range powerTrends {
			if trend.AvgPower > maxAvg {
				maxAvg = trend.AvgPower
				summary.PeakHour = trend.Period
			}
			if trend.AvgPower < minAvg {
				minAvg = trend.AvgPower
				summary.LowestHour = trend.Period
			}
		}
	}

	response := gin.H{
		"summary":      summary,
		"trends":       powerTrends,
		"period":       period,
		"collector_id": collectorID,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}
