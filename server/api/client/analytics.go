package client

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"Power-Monitor/model"

	"github.com/gin-gonic/gin"
)

func registerAnalyticsRoutes(r *gin.RouterGroup) {
	analytics := r.Group("/analytics")
	{
		analytics.GET("/dashboard", getDashboard)
		analytics.GET("/energy-consumption", getEnergyConsumption)
		analytics.GET("/power-trends", getPowerTrends)
		analytics.GET("/cost-analysis", getCostAnalysis)
	}
}

func getDashboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Get user's collectors
	var collectors []model.Collector
	model.DB.Where("user_id = ? AND is_active = ?", userID, true).Find(&collectors)

	// Get summary statistics
	var summary struct {
		TotalCollectors  int     `json:"total_collectors"`
		OnlineCollectors int     `json:"online_collectors"`
		TotalPower       float64 `json:"total_power"`
		TotalEnergy      float64 `json:"total_energy"`
		AlertsCount      int     `json:"alerts_count"`
	}

	summary.TotalCollectors = len(collectors)

	// Calculate online collectors and power consumption
	now := time.Now()
	for _, collector := range collectors {
		if collector.IsOnline() {
			summary.OnlineCollectors++
		}

		// Get latest power data
		var latestData model.PowerData
		if err := model.DB.Where("collector_id = ?", collector.ID).
			Order("timestamp DESC").First(&latestData).Error; err == nil {
			summary.TotalPower += latestData.Power
			summary.TotalEnergy += latestData.Energy
		}
	}

	// Get recent data for charts (last 24 hours)
	yesterday := now.Add(-24 * time.Hour)
	var recentData []model.PowerData
	model.DB.Where("timestamp >= ?", yesterday).
		Order("timestamp ASC").
		Limit(100).
		Find(&recentData)

	response := gin.H{
		"summary":     summary,
		"collectors":  collectors,
		"recent_data": recentData,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func getEnergyConsumption(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "24h") // 24h, 7d, 30d

	// Parse period and calculate start time
	var startTime time.Time
	switch period {
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	// Get user's collectors
	var collectorIDs []string
	model.DB.Model(&model.Collector{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Pluck("collector_id", &collectorIDs)

	if len(collectorIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []interface{}{},
			"message": "No collectors found",
		})
		return
	}

	// Query energy consumption data
	var data []struct {
		CollectorID string    `json:"collector_id"`
		Timestamp   time.Time `json:"timestamp"`
		Energy      float64   `json:"energy"`
	}

	model.DB.Model(&model.PowerData{}).
		Select("collector_id, timestamp, energy").
		Where("collector_id IN ? AND timestamp >= ?", collectorIDs, startTime).
		Order("timestamp ASC").
		Find(&data)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"period":  period,
	})
}

func getPowerTrends(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "7d")      // 24h, 7d, 30d
	trendType := c.DefaultQuery("type", "hourly") // hourly, daily, weekly

	// Parse period and calculate start time
	var startTime time.Time
	var aggregateBy string
	switch period {
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
		aggregateBy = "hour"
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
		aggregateBy = "day"
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
		aggregateBy = "day"
	default:
		startTime = time.Now().Add(-7 * 24 * time.Hour)
		aggregateBy = "day"
	}

	// Get user's collectors
	var collectorIDs []string
	model.DB.Model(&model.Collector{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Pluck("collector_id", &collectorIDs)

	if len(collectorIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"trends":  []interface{}{},
				"message": "No active collectors found",
			},
		})
		return
	}

	// Get power trend data
	var trends []struct {
		Period         string  `json:"period"`
		TotalPower     float64 `json:"total_power"`
		AveragePower   float64 `json:"average_power"`
		MaxPower       float64 `json:"max_power"`
		MinPower       float64 `json:"min_power"`
		EnergyConsumed float64 `json:"energy_consumed"`
		DataPoints     int64   `json:"data_points"`
	}

	var rawQuery string
	if aggregateBy == "hour" {
		rawQuery = `
			SELECT 
				TO_CHAR(timestamp, 'YYYY-MM-DD HH24:00:00') as period,
				SUM(power) as total_power,
				AVG(power) as average_power,
				MAX(power) as max_power,
				MIN(power) as min_power,
				SUM(energy) as energy_consumed,
				COUNT(*) as data_points
			FROM power_data 
			WHERE collector_id = ANY($1) AND timestamp >= $2
			GROUP BY TO_CHAR(timestamp, 'YYYY-MM-DD HH24:00:00')
			ORDER BY period`
	} else {
		rawQuery = `
			SELECT 
				TO_CHAR(timestamp, 'YYYY-MM-DD') as period,
				SUM(power) as total_power,
				AVG(power) as average_power,
				MAX(power) as max_power,
				MIN(power) as min_power,
				SUM(energy) as energy_consumed,
				COUNT(*) as data_points
			FROM power_data 
			WHERE collector_id = ANY($1) AND timestamp >= $2
			GROUP BY TO_CHAR(timestamp, 'YYYY-MM-DD')
			ORDER BY period`
	}

	model.DB.Raw(rawQuery, collectorIDs, startTime).Scan(&trends)

	// Calculate trend analysis
	var analysis struct {
		TrendDirection string  `json:"trend_direction"` // increasing, decreasing, stable
		PercentChange  float64 `json:"percent_change"`
		PeakPeriod     string  `json:"peak_period"`
		LowestPeriod   string  `json:"lowest_period"`
		AverageGrowth  float64 `json:"average_growth"`
	}

	if len(trends) >= 2 {
		firstWeek := trends[:len(trends)/2]
		secondWeek := trends[len(trends)/2:]

		var firstAvg, secondAvg float64
		for _, t := range firstWeek {
			firstAvg += t.AveragePower
		}
		firstAvg /= float64(len(firstWeek))

		for _, t := range secondWeek {
			secondAvg += t.AveragePower
		}
		secondAvg /= float64(len(secondWeek))

		if secondAvg > firstAvg*1.05 {
			analysis.TrendDirection = "increasing"
		} else if secondAvg < firstAvg*0.95 {
			analysis.TrendDirection = "decreasing"
		} else {
			analysis.TrendDirection = "stable"
		}

		analysis.PercentChange = ((secondAvg - firstAvg) / firstAvg) * 100

		// Find peak and lowest periods
		maxPower := trends[0].AveragePower
		minPower := trends[0].AveragePower
		for _, trend := range trends {
			if trend.AveragePower > maxPower {
				maxPower = trend.AveragePower
				analysis.PeakPeriod = trend.Period
			}
			if trend.AveragePower < minPower {
				minPower = trend.AveragePower
				analysis.LowestPeriod = trend.Period
			}
		}
	}

	response := gin.H{
		"trends":   trends,
		"analysis": analysis,
		"period":   period,
		"type":     trendType,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func getCostAnalysis(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "30d") // 30d, 90d, 1y
	currency := c.DefaultQuery("currency", "USD")

	// Get electricity rate from query params or use default
	electricityRateStr := c.DefaultQuery("rate", "0.12") // $0.12 per kWh default
	electricityRate, _ := strconv.ParseFloat(electricityRateStr, 64)

	// Parse period and calculate start time
	var startTime time.Time
	switch period {
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	case "90d":
		startTime = time.Now().Add(-90 * 24 * time.Hour)
	case "1y":
		startTime = time.Now().Add(-365 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	}

	// Get user's collectors
	var collectorIDs []string
	model.DB.Model(&model.Collector{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Pluck("collector_id", &collectorIDs)

	if len(collectorIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"cost_breakdown": []interface{}{},
				"message":        "No active collectors found",
			},
		})
		return
	}

	// Get energy consumption by collector
	var collectorCosts []struct {
		CollectorID    string  `json:"collector_id"`
		CollectorName  string  `json:"collector_name"`
		TotalEnergy    float64 `json:"total_energy_kwh"`
		TotalCost      float64 `json:"total_cost"`
		AveragePower   float64 `json:"average_power"`
		OperatingHours float64 `json:"operating_hours"`
		Percentage     float64 `json:"percentage"`
	}

	model.DB.Raw(`
		SELECT 
			c.collector_id,
			c.name as collector_name,
			COALESCE(SUM(pd.energy) / 1000, 0) as total_energy_kwh,
			COALESCE(AVG(pd.power), 0) as average_power,
			COALESCE(COUNT(pd.id) * (SELECT sample_interval FROM collector_configs WHERE collector_id = c.collector_id LIMIT 1) / 3600.0, 0) as operating_hours
		FROM collectors c
		LEFT JOIN power_data pd ON c.collector_id = pd.collector_id AND pd.timestamp >= ?
		WHERE c.user_id = ? AND c.is_active = true
		GROUP BY c.collector_id, c.name
	`, startTime, userID).Scan(&collectorCosts)

	// Calculate costs and percentages
	var totalEnergy, totalCost float64
	for i := range collectorCosts {
		collectorCosts[i].TotalCost = collectorCosts[i].TotalEnergy * electricityRate
		totalEnergy += collectorCosts[i].TotalEnergy
		totalCost += collectorCosts[i].TotalCost
	}

	for i := range collectorCosts {
		if totalEnergy > 0 {
			collectorCosts[i].Percentage = (collectorCosts[i].TotalEnergy / totalEnergy) * 100
		}
	}

	// Get daily cost breakdown
	var dailyCosts []struct {
		Date       string  `json:"date"`
		EnergyUsed float64 `json:"energy_used_kwh"`
		Cost       float64 `json:"cost"`
		DataPoints int64   `json:"data_points"`
	}

	model.DB.Raw(`
		SELECT 
			TO_CHAR(timestamp, 'YYYY-MM-DD') as date,
			COALESCE(SUM(energy) / 1000, 0) as energy_used_kwh,
			COUNT(*) as data_points
		FROM power_data 
		WHERE collector_id = ANY($1) AND timestamp >= $2
		GROUP BY TO_CHAR(timestamp, 'YYYY-MM-DD')
		ORDER BY date
	`, collectorIDs, startTime).Scan(&dailyCosts)

	for i := range dailyCosts {
		dailyCosts[i].Cost = dailyCosts[i].EnergyUsed * electricityRate
	}

	// Calculate projections and savings estimates
	avgDailyCost := totalCost / float64(len(dailyCosts))
	if len(dailyCosts) == 0 {
		avgDailyCost = 0
	}

	projections := gin.H{
		"monthly_projection": avgDailyCost * 30,
		"yearly_projection":  avgDailyCost * 365,
		"potential_savings":  calculatePotentialSavings(collectorCosts),
	}

	// Cost analysis summary
	summary := gin.H{
		"total_energy_kwh":      totalEnergy,
		"total_cost":            totalCost,
		"average_daily_cost":    avgDailyCost,
		"electricity_rate":      electricityRate,
		"currency":              currency,
		"period":                period,
		"most_expensive_device": getMostExpensiveDevice(collectorCosts),
		"cost_per_kwh":          electricityRate,
	}

	response := gin.H{
		"summary":         summary,
		"collector_costs": collectorCosts,
		"daily_costs":     dailyCosts,
		"projections":     projections,
		"recommendations": generateCostRecommendations(collectorCosts, avgDailyCost),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// Helper function to calculate potential savings
func calculatePotentialSavings(collectorCosts []struct {
	CollectorID    string  `json:"collector_id"`
	CollectorName  string  `json:"collector_name"`
	TotalEnergy    float64 `json:"total_energy_kwh"`
	TotalCost      float64 `json:"total_cost"`
	AveragePower   float64 `json:"average_power"`
	OperatingHours float64 `json:"operating_hours"`
	Percentage     float64 `json:"percentage"`
}) float64 {
	// Simple heuristic: assume 10-20% savings possible through optimization
	var totalCost float64
	for _, cost := range collectorCosts {
		totalCost += cost.TotalCost
	}
	return totalCost * 0.15 // 15% potential savings
}

// Helper function to get most expensive device
func getMostExpensiveDevice(collectorCosts []struct {
	CollectorID    string  `json:"collector_id"`
	CollectorName  string  `json:"collector_name"`
	TotalEnergy    float64 `json:"total_energy_kwh"`
	TotalCost      float64 `json:"total_cost"`
	AveragePower   float64 `json:"average_power"`
	OperatingHours float64 `json:"operating_hours"`
	Percentage     float64 `json:"percentage"`
}) gin.H {
	if len(collectorCosts) == 0 {
		return gin.H{}
	}

	maxCost := collectorCosts[0]
	for _, cost := range collectorCosts {
		if cost.TotalCost > maxCost.TotalCost {
			maxCost = cost
		}
	}

	return gin.H{
		"collector_id":   maxCost.CollectorID,
		"collector_name": maxCost.CollectorName,
		"total_cost":     maxCost.TotalCost,
		"percentage":     maxCost.Percentage,
	}
}

// Helper function to generate cost recommendations
func generateCostRecommendations(collectorCosts []struct {
	CollectorID    string  `json:"collector_id"`
	CollectorName  string  `json:"collector_name"`
	TotalEnergy    float64 `json:"total_energy_kwh"`
	TotalCost      float64 `json:"total_cost"`
	AveragePower   float64 `json:"average_power"`
	OperatingHours float64 `json:"operating_hours"`
	Percentage     float64 `json:"percentage"`
}, avgDailyCost float64) []gin.H {
	var recommendations []gin.H

	// High cost device recommendation
	for _, cost := range collectorCosts {
		if cost.Percentage > 30 {
			recommendations = append(recommendations, gin.H{
				"type":        "high_usage",
				"title":       "High Energy Consumer Detected",
				"description": fmt.Sprintf("Device '%s' accounts for %.1f%% of your energy consumption", cost.CollectorName, cost.Percentage),
				"suggestion":  "Consider optimizing usage patterns or upgrading to more efficient equipment",
				"priority":    "high",
			})
		}
	}

	// Daily cost recommendation
	if avgDailyCost > 5.0 {
		recommendations = append(recommendations, gin.H{
			"type":        "cost_reduction",
			"title":       "High Daily Energy Costs",
			"description": fmt.Sprintf("Your average daily cost is $%.2f", avgDailyCost),
			"suggestion":  "Consider implementing energy-saving measures or time-of-use optimization",
			"priority":    "medium",
		})
	}

	// Always-on devices recommendation
	for _, cost := range collectorCosts {
		if cost.OperatingHours > 20 {
			recommendations = append(recommendations, gin.H{
				"type":        "always_on",
				"title":       "Always-On Device Detected",
				"description": fmt.Sprintf("Device '%s' operates %.1f hours daily", cost.CollectorName, cost.OperatingHours),
				"suggestion":  "Check if this device needs to run continuously or can be scheduled",
				"priority":    "low",
			})
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, gin.H{
			"type":        "efficiency",
			"title":       "Efficient Energy Usage",
			"description": "Your energy consumption patterns look good",
			"suggestion":  "Continue monitoring for optimization opportunities",
			"priority":    "info",
		})
	}

	return recommendations
}
