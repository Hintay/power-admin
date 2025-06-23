package admin

import (
	"fmt"
	"net/http"
	"time"

	"Power-Monitor/internal/influxdb"
	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/gin-gonic/gin"
)

func registerSystemRoutes(r *gin.RouterGroup) {
	system := r.Group("/system")
	{
		system.GET("/stats", getSystemStats)
		system.GET("/health", getSystemHealth)
	}
}

func getSystemStats(c *gin.Context) {
	var stats struct {
		TotalUsers       int64 `json:"total_users"`
		TotalCollectors  int64 `json:"total_collectors"`
		ActiveCollectors int64 `json:"active_collectors"`
		TotalDataPoints  int64 `json:"total_data_points"`
		DataPointsToday  int64 `json:"data_points_today"`
	}

	model.DB.Model(&model.User{}).Count(&stats.TotalUsers)
	model.DB.Model(&model.Collector{}).Count(&stats.TotalCollectors)
	model.DB.Model(&model.Collector{}).Where("is_active = ?", true).Count(&stats.ActiveCollectors)
	model.DB.Model(&model.PowerData{}).Count(&stats.TotalDataPoints)

	today := time.Now().Truncate(24 * time.Hour)
	model.DB.Model(&model.PowerData{}).Where("timestamp >= ?", today).Count(&stats.DataPointsToday)

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// Global variable to track server start time
var serverStartTime = time.Now()

func getSystemHealth(c *gin.Context) {
	// Check database connection
	dbStatus := "connected"
	if sqlDB, err := model.DB.DB(); err != nil {
		dbStatus = "error"
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	// Check InfluxDB connection
	influxStatus := "disabled"
	if settings.InfluxDBSettings.Enabled {
		influxStatus = "disconnected"
		influxClient := influxdb.GetClient()
		if influxClient != nil {
			// Test InfluxDB health
			if health, err := influxClient.Health(); err == nil && health.Status == "pass" {
				influxStatus = "connected"
			} else {
				influxStatus = "error"
			}
		}
	}

	// Calculate actual uptime
	uptime := time.Since(serverStartTime)
	uptimeStr := formatDuration(uptime)

	health := gin.H{
		"status":         determineOverallStatus(dbStatus, influxStatus),
		"database":       dbStatus,
		"influxdb":       influxStatus,
		"realtime":       "running",
		"timestamp":      time.Now(),
		"uptime":         uptimeStr,
		"uptime_seconds": int64(uptime.Seconds()),
	}

	c.JSON(http.StatusOK, gin.H{"data": health})
}

// Helper function to determine overall system status
func determineOverallStatus(dbStatus, influxStatus string) string {
	if dbStatus == "connected" && (influxStatus == "connected" || influxStatus == "disconnected" || influxStatus == "disabled") {
		return "healthy"
	}
	if dbStatus == "connected" && influxStatus == "error" {
		return "degraded"
	}
	return "unhealthy"
}

// Helper function to format duration to human-readable string
func formatDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
