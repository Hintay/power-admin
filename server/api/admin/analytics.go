package admin

import (
	"net/http"
	"time"

	"Power-Monitor/model"

	"github.com/gin-gonic/gin"
)

func registerAnalyticsRoutes(r *gin.RouterGroup) {
	analytics := r.Group("/analytics")
	{
		analytics.GET("/dashboard", getDashboardData)
		analytics.GET("/power-data", getPowerDataAnalytics)
		analytics.GET("/collectors/:id/data", getCollectorData)
	}
}

func getDashboardData(c *gin.Context) {
	// Admin-only: Get system-wide overview statistics for all users and collectors
	var stats struct {
		TotalUsers         int64   `json:"total_users"`
		TotalCollectors    int64   `json:"total_collectors"`
		ActiveCollectors   int64   `json:"active_collectors"`
		TotalDataPoints    int64   `json:"total_data_points"`
		DataPointsToday    int64   `json:"data_points_today"`
		DataPointsThisWeek int64   `json:"data_points_this_week"`
		AveragePower       float64 `json:"average_power"`
		TotalEnergy        float64 `json:"total_energy"`
	}

	model.DB.Model(&model.User{}).Count(&stats.TotalUsers)
	model.DB.Model(&model.Collector{}).Count(&stats.TotalCollectors)
	model.DB.Model(&model.Collector{}).Where("is_active = ?", true).Count(&stats.ActiveCollectors)
	model.DB.Model(&model.PowerData{}).Count(&stats.TotalDataPoints)

	// Time-based queries
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	model.DB.Model(&model.PowerData{}).Where("timestamp >= ?", today).Count(&stats.DataPointsToday)
	model.DB.Model(&model.PowerData{}).Where("timestamp >= ?", weekStart).Count(&stats.DataPointsThisWeek)

	// Calculate average power and total energy
	model.DB.Model(&model.PowerData{}).
		Select("AVG(power), SUM(energy)").
		Row().Scan(&stats.AveragePower, &stats.TotalEnergy)

	// Get recent activity data (last 24 hours)
	var recentActivity []struct {
		Hour       int     `json:"hour"`
		DataPoints int64   `json:"data_points"`
		AvgPower   float64 `json:"avg_power"`
	}

	model.DB.Raw(`
		SELECT 
			EXTRACT(hour FROM timestamp) as hour,
			COUNT(*) as data_points,
			AVG(power) as avg_power
		FROM power_data 
		WHERE timestamp >= ? 
		GROUP BY EXTRACT(hour FROM timestamp)
		ORDER BY hour
	`, today).Scan(&recentActivity)

	// Get top collectors by data points
	var topCollectors []struct {
		CollectorID string  `json:"collector_id"`
		Name        string  `json:"name"`
		DataPoints  int64   `json:"data_points"`
		AvgPower    float64 `json:"avg_power"`
	}

	model.DB.Raw(`
		SELECT 
			c.collector_id,
			c.name,
			COUNT(pd.id) as data_points,
			AVG(pd.power) as avg_power
		FROM collectors c
		LEFT JOIN power_data pd ON c.collector_id = pd.collector_id
		WHERE c.is_active = true
		GROUP BY c.collector_id, c.name
		ORDER BY data_points DESC
		LIMIT 5
	`).Scan(&topCollectors)

	response := gin.H{
		"overview":        stats,
		"recent_activity": recentActivity,
		"top_collectors":  topCollectors,
		"timestamp":       now,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func getPowerDataAnalytics(c *gin.Context) {
	// Admin-only: Get power data analytics for any collector (system-wide)
	// Note: Client version with user permissions is available in client/data.go
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

	// Build query conditions
	query := model.DB.Model(&model.PowerData{}).Where("timestamp >= ?", startTime)
	if collectorID != "" {
		query = query.Where("collector_id = ?", collectorID)
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
			WHERE timestamp >= ?`
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
			WHERE timestamp >= ?`
	}

	args := []interface{}{startTime}
	if collectorID != "" {
		rawQuery += " AND collector_id = ?"
		args = append(args, collectorID)
	}

	rawQuery += " GROUP BY period ORDER BY period"

	model.DB.Raw(rawQuery, args...).Scan(&powerTrends)

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

	query.Select("COUNT(*), AVG(power), MAX(power), MIN(power), SUM(energy)").
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

func getCollectorData(c *gin.Context) {
	// Admin-only: Get any collector's data without user permission checks
	// Note: Client version with user permissions is available in client/data.go
	collectorID := c.Param("id")
	dataType := c.DefaultQuery("type", "overview") // overview, realtime, history
	period := c.DefaultQuery("period", "24h")

	// Get collector info (admin can access any collector)
	var collector model.Collector
	if err := model.DB.Where("id = ?", collectorID).First(&collector).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collector not found"})
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
