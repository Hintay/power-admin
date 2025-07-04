package client

import (
	"fmt"
	"math"
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
		analytics.GET("/prediction/:collectorId", getDailyEnergyPrediction)
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

	// Use GORM's standard query methods instead of raw SQL
	var powerDataList []model.PowerData
	model.DB.Where("collector_id IN ? AND timestamp >= ?", collectorIDs, startTime).
		Find(&powerDataList)

	// Group by period manually
	periodMap := make(map[string]struct {
		TotalPower     float64
		PowerSum       float64
		PowerCount     int64
		MaxPower       float64
		MinPower       float64
		EnergyConsumed float64
		DataPoints     int64
	})

	for _, data := range powerDataList {
		var period string
		if aggregateBy == "hour" {
			period = data.Timestamp.Format("2006-01-02 15:00:00")
		} else {
			period = data.Timestamp.Format("2006-01-02")
		}

		entry := periodMap[period]
		entry.TotalPower += data.Power
		entry.PowerSum += data.Power
		entry.PowerCount++
		if entry.DataPoints == 0 || data.Power > entry.MaxPower {
			entry.MaxPower = data.Power
		}
		if entry.DataPoints == 0 || data.Power < entry.MinPower {
			entry.MinPower = data.Power
		}
		entry.EnergyConsumed += data.Energy
		entry.DataPoints++
		periodMap[period] = entry
	}

	// Convert map to slice
	for period, entry := range periodMap {
		avgPower := entry.PowerSum / float64(entry.PowerCount)
		trends = append(trends, struct {
			Period         string  `json:"period"`
			TotalPower     float64 `json:"total_power"`
			AveragePower   float64 `json:"average_power"`
			MaxPower       float64 `json:"max_power"`
			MinPower       float64 `json:"min_power"`
			EnergyConsumed float64 `json:"energy_consumed"`
			DataPoints     int64   `json:"data_points"`
		}{
			Period:         period,
			TotalPower:     entry.TotalPower,
			AveragePower:   avgPower,
			MaxPower:       entry.MaxPower,
			MinPower:       entry.MinPower,
			EnergyConsumed: entry.EnergyConsumed,
			DataPoints:     entry.DataPoints,
		})
	}

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

	// Use GORM's standard query methods instead of raw SQL
	var powerDataList []model.PowerData
	model.DB.Where("collector_id IN ? AND timestamp >= ?", collectorIDs, startTime).
		Find(&powerDataList)

	// Group by date manually
	dailyMap := make(map[string]struct {
		EnergyUsed float64
		DataPoints int64
	})

	for _, data := range powerDataList {
		date := data.Timestamp.Format("2006-01-02")
		entry := dailyMap[date]
		entry.EnergyUsed += data.Energy / 1000.0 // Convert to kWh
		entry.DataPoints++
		dailyMap[date] = entry
	}

	// Convert map to slice
	for date, entry := range dailyMap {
		dailyCosts = append(dailyCosts, struct {
			Date       string  `json:"date"`
			EnergyUsed float64 `json:"energy_used_kwh"`
			Cost       float64 `json:"cost"`
			DataPoints int64   `json:"data_points"`
		}{
			Date:       date,
			EnergyUsed: entry.EnergyUsed,
			Cost:       0, // Will be calculated below
			DataPoints: entry.DataPoints,
		})
	}

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

// getDailyEnergyPrediction predicts daily energy consumption for a specific collector
// Path: GET /api/client/analytics/prediction/{collectorId}
// Minimum 10 minutes of data required for prediction
func getDailyEnergyPrediction(c *gin.Context) {
	userID := c.GetUint("user_id")
	collectorID := c.Param("collectorId")              // Required: collector ID from URL path
	algorithm := c.DefaultQuery("algorithm", "hybrid") // hybrid, linear, seasonal, moving_average

	// Validate collector ID parameter
	if collectorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Collector ID is required in URL path",
		})
		return
	}

	// Validate collector belongs to user and is active
	var collector model.Collector
	if err := model.DB.Where("id = ? AND user_id = ? AND is_active = ?", collectorID, userID, true).First(&collector).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Collector not found or not accessible",
		})
		return
	}

	collectorIDs := []string{collector.CollectorID}

	// Get today's start time
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Check if we have minimum 10 minutes of data today
	minDataTime := now.Add(-10 * time.Minute)
	var todayDataCount int64
	model.DB.Model(&model.PowerData{}).
		Where("collector_id IN ? AND timestamp >= ? AND timestamp >= ?",
			collectorIDs, todayStart, minDataTime).
		Count(&todayDataCount)

	// If no current day data, check for historical data availability
	var hasHistoricalData bool = false
	if todayDataCount == 0 {
		var historicalDataCount int64
		// Check for data in the past 30 days
		historicalStartTime := now.AddDate(0, 0, -30)
		model.DB.Model(&model.PowerData{}).
			Where("collector_id IN ? AND timestamp >= ? AND timestamp < ?",
				collectorIDs, historicalStartTime, todayStart).
			Count(&historicalDataCount)

		if historicalDataCount >= 24 { // At least 24 data points (roughly 1 day worth)
			hasHistoricalData = true
		}
	}

	// If neither current day nor historical data is available, return error
	if todayDataCount == 0 && !hasHistoricalData {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Insufficient data: no current day data and insufficient historical data for prediction",
		})
		return
	}

	// Get prediction based on selected algorithm
	var prediction *EnergyPrediction
	var err error

	// Pass information about data availability to prediction functions
	hasCurrentData := todayDataCount > 0

	// Debug information
	debugInfo := gin.H{
		"collector_ids":    collectorIDs,
		"today_data_count": todayDataCount,
		"has_current_data": hasCurrentData,
		"has_historical":   hasHistoricalData,
		"algorithm":        algorithm,
		"prediction_time":  now,
		"today_start":      todayStart,
	}

	switch algorithm {
	case "linear":
		prediction, err = predictLinearTrend(collectorIDs, todayStart, now, hasCurrentData)
		if err != nil {
			debugInfo["linear_error"] = err.Error()
		}
	case "seasonal":
		prediction, err = predictSeasonalPattern(collectorIDs, todayStart, now, hasCurrentData)
		if err != nil {
			debugInfo["seasonal_error"] = err.Error()
		}
	case "moving_average":
		prediction, err = predictMovingAverage(collectorIDs, todayStart, now, hasCurrentData)
		if err != nil {
			debugInfo["moving_average_error"] = err.Error()
		}
	default: // hybrid
		prediction, err = predictHybridMethod(collectorIDs, todayStart, now, hasCurrentData)
		if err != nil {
			debugInfo["hybrid_error"] = err.Error()
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Prediction failed: %v", err),
			"debug":   debugInfo,
		})
		return
	}

	// Get actual consumption so far today for comparison
	actualConsumption := getTodayActualConsumption(collectorIDs, todayStart, now)

	response := gin.H{
		"prediction":         prediction,
		"actual_consumption": actualConsumption,
		"algorithm_used":     algorithm,
		"data_points":        todayDataCount,
		"prediction_time":    now,
		"collectors":         collectorIDs,
		"has_current_data":   hasCurrentData,
		"debug":              debugInfo,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// EnergyPrediction represents the prediction result
type EnergyPrediction struct {
	TotalDailyEnergyKWh  float64               `json:"total_daily_energy_kwh"`
	RemainingEnergyKWh   float64               `json:"remaining_energy_kwh"`
	PredictedEndTime     time.Time             `json:"predicted_end_time"`
	ConfidenceLevel      float64               `json:"confidence_level"`
	PredictionAccuracy   string                `json:"prediction_accuracy"`
	HourlyPredictions    []HourlyPrediction    `json:"hourly_predictions"`
	CollectorPredictions []CollectorPrediction `json:"collector_predictions"`
	ModelMetrics         PredictionMetrics     `json:"model_metrics"`
	Recommendations      []string              `json:"recommendations"`
}

type HourlyPrediction struct {
	Hour               int     `json:"hour"`
	PredictedEnergyKWh float64 `json:"predicted_energy_kwh"`
	PredictedAvgPower  float64 `json:"predicted_avg_power"`
	ConfidenceInterval float64 `json:"confidence_interval"`
}

type CollectorPrediction struct {
	CollectorID        string  `json:"collector_id"`
	CollectorName      string  `json:"collector_name"`
	PredictedEnergyKWh float64 `json:"predicted_energy_kwh"`
	CurrentEnergyKWh   float64 `json:"current_energy_kwh"`
	PercentageOfTotal  float64 `json:"percentage_of_total"`
}

type PredictionMetrics struct {
	Algorithm          string  `json:"algorithm"`
	DataQuality        string  `json:"data_quality"`
	HistoricalAccuracy float64 `json:"historical_accuracy"`
	TrendStrength      float64 `json:"trend_strength"`
	SeasonalityScore   float64 `json:"seasonality_score"`
	NoiseLevel         float64 `json:"noise_level"`
}

// predictHybridMethod combines multiple prediction methods for better accuracy
func predictHybridMethod(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) (*EnergyPrediction, error) {
	// Get multiple predictions with detailed error tracking
	var errors []string

	linearPred, linearErr := predictLinearTrend(collectorIDs, todayStart, now, hasCurrentData)
	if linearErr != nil {
		errors = append(errors, fmt.Sprintf("Linear trend: %v", linearErr))
	}

	seasonalPred, seasonalErr := predictSeasonalPattern(collectorIDs, todayStart, now, hasCurrentData)
	if seasonalErr != nil {
		errors = append(errors, fmt.Sprintf("Seasonal pattern: %v", seasonalErr))
	}

	avgPred, avgErr := predictMovingAverage(collectorIDs, todayStart, now, hasCurrentData)
	if avgErr != nil {
		errors = append(errors, fmt.Sprintf("Moving average: %v", avgErr))
	}

	if linearPred == nil && seasonalPred == nil && avgPred == nil {
		// Try a simple fallback prediction based on any available data
		fallbackPred, fallbackErr := createFallbackPrediction(collectorIDs, todayStart, now, hasCurrentData)
		if fallbackPred != nil {
			fallbackPred.ModelMetrics.Algorithm = "fallback"
			fallbackPred.Recommendations = append(fallbackPred.Recommendations,
				"Using fallback prediction due to insufficient historical data. Accuracy may be limited.")
			return fallbackPred, nil
		}

		errorDetails := "All prediction methods failed: " + fmt.Sprintf("%v", errors)
		if fallbackErr != nil {
			errorDetails += fmt.Sprintf(". Fallback error: %v", fallbackErr)
		}
		return nil, fmt.Errorf("%s. Collector IDs: %v, Has current data: %v", errorDetails, collectorIDs, hasCurrentData)
	}

	// Weight predictions based on data quality and historical accuracy
	weights := calculatePredictionWeights(collectorIDs, todayStart, now, hasCurrentData)

	totalPrediction := 0.0
	totalWeight := 0.0
	confidence := 0.0

	if linearPred != nil {
		totalPrediction += linearPred.TotalDailyEnergyKWh * weights.Linear
		totalWeight += weights.Linear
		confidence += linearPred.ConfidenceLevel * weights.Linear
	}

	if seasonalPred != nil {
		totalPrediction += seasonalPred.TotalDailyEnergyKWh * weights.Seasonal
		totalWeight += weights.Seasonal
		confidence += seasonalPred.ConfidenceLevel * weights.Seasonal
	}

	if avgPred != nil {
		totalPrediction += avgPred.TotalDailyEnergyKWh * weights.MovingAverage
		totalWeight += weights.MovingAverage
		confidence += avgPred.ConfidenceLevel * weights.MovingAverage
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no valid predictions available")
	}

	finalPrediction := totalPrediction / totalWeight
	finalConfidence := confidence / totalWeight

	// Get current consumption
	currentConsumption := getTodayActualConsumption(collectorIDs, todayStart, now)

	// Generate hourly predictions
	hourlyPreds := generateHourlyPredictions(collectorIDs, todayStart, now, finalPrediction)

	// Generate collector-specific predictions
	collectorPreds := generateCollectorPredictions(collectorIDs, todayStart, now, finalPrediction)

	// Calculate metrics
	metrics := calculatePredictionMetrics(collectorIDs, todayStart, now, "hybrid")

	// Generate recommendations
	recommendations := generatePredictionRecommendations(finalPrediction, currentConsumption, now, hasCurrentData)

	return &EnergyPrediction{
		TotalDailyEnergyKWh:  finalPrediction,
		RemainingEnergyKWh:   finalPrediction - currentConsumption,
		PredictedEndTime:     time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
		ConfidenceLevel:      finalConfidence,
		PredictionAccuracy:   getPredictionAccuracyLabel(finalConfidence),
		HourlyPredictions:    hourlyPreds,
		CollectorPredictions: collectorPreds,
		ModelMetrics:         metrics,
		Recommendations:      recommendations,
	}, nil
}

// predictLinearTrend uses linear extrapolation based on current day's trend
func predictLinearTrend(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) (*EnergyPrediction, error) {
	// Get hourly consumption for today
	var hourlyData []struct {
		Hour   int     `json:"hour"`
		Energy float64 `json:"energy"`
	}

	// Validate input
	if len(collectorIDs) == 0 {
		return nil, fmt.Errorf("no collector IDs provided for linear trend prediction")
	}

	// Use GORM's standard query methods instead of raw SQL to ensure database compatibility
	var powerDataList []model.PowerData
	err := model.DB.Where("collector_id IN ? AND timestamp >= ? AND timestamp <= ?", collectorIDs, todayStart, now).
		Find(&powerDataList).Error

	if err != nil {
		return nil, fmt.Errorf("database error in linear trend prediction: %v", err)
	}

	// Group by hour manually
	hourlyMap := make(map[int]float64)
	for _, data := range powerDataList {
		hour := data.Timestamp.Hour()
		hourlyMap[hour] += data.Energy / 1000.0 // Convert to kWh
	}

	// Convert map to slice
	for hour, energy := range hourlyMap {
		hourlyData = append(hourlyData, struct {
			Hour   int     `json:"hour"`
			Energy float64 `json:"energy"`
		}{Hour: hour, Energy: energy})
	}

	if !hasCurrentData {
		// If no current data, use historical pattern to predict full day
		return predictSeasonalPattern(collectorIDs, todayStart, now, hasCurrentData)
	}

	if len(hourlyData) < 2 {
		return nil, fmt.Errorf("insufficient hourly data for linear trend prediction (found %d hours, need at least 2). Power data points: %d", len(hourlyData), len(powerDataList))
	}

	// Calculate linear trend
	slope, intercept := calculateLinearRegression(hourlyData)

	// Predict remaining hours
	currentHour := now.Hour()
	totalPrediction := 0.0

	// Add actual consumption so far
	for _, data := range hourlyData {
		totalPrediction += data.Energy
	}

	// Predict remaining hours
	for hour := currentHour + 1; hour < 24; hour++ {
		predicted := slope*float64(hour) + intercept
		if predicted < 0 {
			predicted = 0
		}
		totalPrediction += predicted
	}

	// Calculate confidence based on R-squared
	confidence := calculateRSquared(hourlyData, slope, intercept)

	return &EnergyPrediction{
		TotalDailyEnergyKWh: totalPrediction,
		ConfidenceLevel:     confidence * 100,
		PredictionAccuracy:  getPredictionAccuracyLabel(confidence * 100),
	}, nil
}

// predictSeasonalPattern uses historical data from similar days
func predictSeasonalPattern(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) (*EnergyPrediction, error) {
	// Validate input
	if len(collectorIDs) == 0 {
		return nil, fmt.Errorf("no collector IDs provided")
	}

	// Get historical data for the same day of week over the past 4 weeks
	var historicalDays []time.Time
	for i := 1; i <= 4; i++ {
		historicalDay := todayStart.AddDate(0, 0, -7*i)
		historicalDays = append(historicalDays, historicalDay)
	}

	var historicalData []struct {
		Date   time.Time `json:"date"`
		Energy float64   `json:"energy"`
	}

	// Track data collection details for debugging
	var dataCollectionDetails []string

	for _, date := range historicalDays {
		dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		dayEnd := dayStart.Add(24 * time.Hour)

		var dayEnergy float64
		err := model.DB.Model(&model.PowerData{}).
			Select("COALESCE(SUM(energy) / 1000.0, 0)").
			Where("collector_id IN ? AND timestamp >= ? AND timestamp < ?", collectorIDs, dayStart, dayEnd).
			Row().Scan(&dayEnergy)

		if err != nil {
			dataCollectionDetails = append(dataCollectionDetails, fmt.Sprintf("DB error for %s: %v", dayStart.Format("2006-01-02"), err))
			continue
		}

		dataCollectionDetails = append(dataCollectionDetails, fmt.Sprintf("Date %s: %.3f kWh", dayStart.Format("2006-01-02"), dayEnergy))

		if dayEnergy > 0 {
			historicalData = append(historicalData, struct {
				Date   time.Time `json:"date"`
				Energy float64   `json:"energy"`
			}{Date: date, Energy: dayEnergy})
		}
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data available for seasonal prediction. Data collection details: %v", dataCollectionDetails)
	}

	// Calculate average consumption for similar days
	totalHistorical := 0.0
	for _, data := range historicalData {
		totalHistorical += data.Energy
	}
	avgHistorical := totalHistorical / float64(len(historicalData))

	// Adjust prediction based on current consumption pattern if we have current data
	if hasCurrentData {
		currentConsumption := getTodayActualConsumption(collectorIDs, todayStart, now)
		timeElapsed := now.Sub(todayStart).Hours()
		timeRatio := timeElapsed / 24.0

		if timeRatio > 0 {
			expectedByNow := avgHistorical * timeRatio
			adjustmentFactor := currentConsumption / expectedByNow
			if adjustmentFactor > 0.5 && adjustmentFactor < 2.0 {
				avgHistorical *= adjustmentFactor
			}
		}
	}

	// Calculate confidence based on consistency of historical data
	variance := calculateVariance(historicalData)
	confidence := math.Max(0, 100-variance*10)

	return &EnergyPrediction{
		TotalDailyEnergyKWh: avgHistorical,
		ConfidenceLevel:     confidence,
		PredictionAccuracy:  getPredictionAccuracyLabel(confidence),
	}, nil
}

// predictMovingAverage uses moving average of recent days
func predictMovingAverage(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) (*EnergyPrediction, error) {
	// Validate input
	if len(collectorIDs) == 0 {
		return nil, fmt.Errorf("no collector IDs provided")
	}

	// Get past 7 days consumption
	var recentDays []float64
	var dataDetails []string

	for i := 1; i <= 7; i++ {
		dayStart := todayStart.AddDate(0, 0, -i)
		dayEnd := dayStart.Add(24 * time.Hour)

		var dayEnergy float64
		err := model.DB.Model(&model.PowerData{}).
			Select("COALESCE(SUM(energy) / 1000.0, 0)").
			Where("collector_id IN ? AND timestamp >= ? AND timestamp < ?", collectorIDs, dayStart, dayEnd).
			Row().Scan(&dayEnergy)

		if err != nil {
			dataDetails = append(dataDetails, fmt.Sprintf("DB error for day -%d (%s): %v", i, dayStart.Format("2006-01-02"), err))
			continue
		}

		dataDetails = append(dataDetails, fmt.Sprintf("Day -%d (%s): %.3f kWh", i, dayStart.Format("2006-01-02"), dayEnergy))

		if dayEnergy > 0 {
			recentDays = append(recentDays, dayEnergy)
		}
	}

	if len(recentDays) < 3 {
		return nil, fmt.Errorf("insufficient historical data for moving average prediction (found %d days, need at least 3). Data details: %v", len(recentDays), dataDetails)
	}

	// Calculate weighted moving average (recent days have higher weight)
	totalWeight := 0.0
	weightedSum := 0.0
	for i, energy := range recentDays {
		weight := float64(len(recentDays) - i) // More recent = higher weight
		weightedSum += energy * weight
		totalWeight += weight
	}

	avgConsumption := weightedSum / totalWeight

	// Adjust based on current day's pattern if we have current data
	if hasCurrentData {
		currentConsumption := getTodayActualConsumption(collectorIDs, todayStart, now)
		timeElapsed := now.Sub(todayStart).Hours()
		timeRatio := timeElapsed / 24.0

		if timeRatio > 0.1 {
			expectedByNow := avgConsumption * timeRatio
			adjustmentFactor := currentConsumption / expectedByNow
			if adjustmentFactor > 0.7 && adjustmentFactor < 1.5 {
				avgConsumption *= adjustmentFactor
			}
		}
	}

	// Calculate confidence based on data consistency
	variance := 0.0
	for _, energy := range recentDays {
		variance += math.Pow(energy-avgConsumption, 2)
	}
	variance /= float64(len(recentDays))
	stdDev := math.Sqrt(variance)

	confidence := math.Max(0, 100-(stdDev/avgConsumption)*100)

	return &EnergyPrediction{
		TotalDailyEnergyKWh: avgConsumption,
		ConfidenceLevel:     confidence,
		PredictionAccuracy:  getPredictionAccuracyLabel(confidence),
	}, nil
}

// Helper functions

type PredictionWeights struct {
	Linear        float64
	Seasonal      float64
	MovingAverage float64
}

func calculatePredictionWeights(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) PredictionWeights {
	// Default weights
	weights := PredictionWeights{
		Linear:        0.4,
		Seasonal:      0.3,
		MovingAverage: 0.3,
	}

	// If no current data, rely entirely on historical patterns
	if !hasCurrentData {
		weights.Linear = 0.0        // Can't do linear trend without current data
		weights.Seasonal = 0.6      // Prefer seasonal patterns
		weights.MovingAverage = 0.4 // Use moving average as backup
		return weights
	}

	// Adjust weights based on data availability and time of day
	timeElapsed := now.Sub(todayStart).Hours()

	if timeElapsed < 6 {
		// Early in the day, rely more on historical patterns
		weights.Seasonal = 0.5
		weights.MovingAverage = 0.4
		weights.Linear = 0.1
	} else if timeElapsed > 18 {
		// Late in the day, linear trend is more reliable
		weights.Linear = 0.6
		weights.Seasonal = 0.2
		weights.MovingAverage = 0.2
	}

	return weights
}

// createFallbackPrediction creates a simple prediction when all other methods fail
func createFallbackPrediction(collectorIDs []string, todayStart, now time.Time, hasCurrentData bool) (*EnergyPrediction, error) {
	if len(collectorIDs) == 0 {
		return nil, fmt.Errorf("no collector IDs provided for fallback prediction")
	}

	// Try to get any available data from the past 30 days
	var anyHistoricalData []model.PowerData
	err := model.DB.Where("collector_id IN ? AND timestamp >= ?", collectorIDs, todayStart.AddDate(0, 0, -30)).
		Limit(100). // Limit to avoid too much data
		Find(&anyHistoricalData).Error

	if err != nil {
		return nil, fmt.Errorf("database error in fallback prediction: %v", err)
	}

	if len(anyHistoricalData) == 0 {
		return nil, fmt.Errorf("no historical data found for fallback prediction")
	}

	// Calculate a simple average energy consumption per day
	var totalEnergy float64
	dayMap := make(map[string]float64)

	for _, data := range anyHistoricalData {
		date := data.Timestamp.Format("2006-01-02")
		dayMap[date] += data.Energy / 1000.0 // Convert to kWh
		totalEnergy += data.Energy / 1000.0
	}

	var avgDailyEnergy float64
	if len(dayMap) > 0 {
		var totalDaily float64
		for _, daily := range dayMap {
			totalDaily += daily
		}
		avgDailyEnergy = totalDaily / float64(len(dayMap))
	} else {
		avgDailyEnergy = 5.0 // Default 5 kWh if no data
	}

	// Ensure minimum reasonable prediction
	if avgDailyEnergy < 0.1 {
		avgDailyEnergy = 1.0 // Minimum 1 kWh per day
	}

	currentConsumption := getTodayActualConsumption(collectorIDs, todayStart, now)

	return &EnergyPrediction{
		TotalDailyEnergyKWh:  avgDailyEnergy,
		RemainingEnergyKWh:   math.Max(0, avgDailyEnergy-currentConsumption),
		PredictedEndTime:     time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
		ConfidenceLevel:      30.0, // Low confidence for fallback
		PredictionAccuracy:   "low",
		HourlyPredictions:    []HourlyPrediction{},
		CollectorPredictions: []CollectorPrediction{},
		ModelMetrics: PredictionMetrics{
			Algorithm:          "fallback",
			DataQuality:        "limited",
			HistoricalAccuracy: 30.0,
			TrendStrength:      0.1,
			SeasonalityScore:   0.1,
			NoiseLevel:         0.8,
		},
		Recommendations: []string{
			fmt.Sprintf("Fallback prediction based on %d historical data points from %d days", len(anyHistoricalData), len(dayMap)),
			"Consider collecting more data for better predictions",
		},
	}, nil
}

func getTodayActualConsumption(collectorIDs []string, todayStart, now time.Time) float64 {
	var consumption float64
	model.DB.Model(&model.PowerData{}).
		Select("COALESCE(SUM(energy) / 1000.0, 0)").
		Where("collector_id IN ? AND timestamp >= ? AND timestamp <= ?", collectorIDs, todayStart, now).
		Row().Scan(&consumption)
	return consumption
}

func generateHourlyPredictions(collectorIDs []string, todayStart, now time.Time, totalPrediction float64) []HourlyPrediction {
	var predictions []HourlyPrediction
	currentHour := now.Hour()

	// Get hourly pattern from historical data
	var hourlyPattern []struct {
		Hour           int     `json:"hour"`
		AvgEnergyRatio float64 `json:"avg_energy_ratio"`
	}

	// Use GORM's standard query methods instead of raw SQL
	historicalStartTime := todayStart.AddDate(0, 0, -7)

	// Get all historical data
	var historicalData []model.PowerData
	model.DB.Where("collector_id IN ? AND timestamp >= ?", collectorIDs, historicalStartTime).
		Find(&historicalData)

	// Calculate overall average
	var totalEnergy float64
	var totalCount int64
	for _, data := range historicalData {
		totalEnergy += data.Energy
		totalCount++
	}
	overallAvg := totalEnergy / float64(totalCount)

	// Group by hour and calculate ratios
	hourlyMap := make(map[int]struct {
		TotalEnergy float64
		Count       int64
	})

	for _, data := range historicalData {
		hour := data.Timestamp.Hour()
		entry := hourlyMap[hour]
		entry.TotalEnergy += data.Energy
		entry.Count++
		hourlyMap[hour] = entry
	}

	// Convert to ratio format
	for hour, entry := range hourlyMap {
		avgEnergy := entry.TotalEnergy / float64(entry.Count)
		ratio := avgEnergy / overallAvg
		hourlyPattern = append(hourlyPattern, struct {
			Hour           int     `json:"hour"`
			AvgEnergyRatio float64 `json:"avg_energy_ratio"`
		}{
			Hour:           hour,
			AvgEnergyRatio: ratio,
		})
	}

	// Create predictions for remaining hours
	for hour := currentHour + 1; hour < 24; hour++ {
		var ratio float64 = 1.0 / 24.0 // Default equal distribution

		// Find ratio from historical pattern
		for _, pattern := range hourlyPattern {
			if pattern.Hour == hour {
				ratio = pattern.AvgEnergyRatio / 24.0
				break
			}
		}

		predictions = append(predictions, HourlyPrediction{
			Hour:               hour,
			PredictedEnergyKWh: totalPrediction * ratio,
			PredictedAvgPower:  (totalPrediction * ratio) * 1000, // Convert to Wh
			ConfidenceInterval: 0.15,                             // ±15%
		})
	}

	return predictions
}

func generateCollectorPredictions(collectorIDs []string, todayStart, now time.Time, totalPrediction float64) []CollectorPrediction {
	var predictions []CollectorPrediction

	for _, collectorID := range collectorIDs {
		var collectorData struct {
			Name          string  `json:"name"`
			CurrentEnergy float64 `json:"current_energy"`
			HistoricalAvg float64 `json:"historical_avg"`
		}

		// Get collector info and current consumption
		model.DB.Model(&model.Collector{}).
			Select("name").
			Where("collector_id = ?", collectorID).
			Row().Scan(&collectorData.Name)

		model.DB.Model(&model.PowerData{}).
			Select("COALESCE(SUM(energy) / 1000.0, 0)").
			Where("collector_id = ? AND timestamp >= ? AND timestamp <= ?", collectorID, todayStart, now).
			Row().Scan(&collectorData.CurrentEnergy)

		// Get historical average for this collector using GORM methods
		var historicalPowerData []model.PowerData
		model.DB.Where("collector_id = ? AND timestamp >= ?", collectorID, todayStart.AddDate(0, 0, -30)).
			Find(&historicalPowerData)

		// Group by date and calculate daily totals
		dailyTotals := make(map[string]float64)
		for _, data := range historicalPowerData {
			date := data.Timestamp.Format("2006-01-02")
			dailyTotals[date] += data.Energy / 1000.0 // Convert to kWh
		}

		// Calculate average of daily totals
		if len(dailyTotals) > 0 {
			var totalDailyEnergy float64
			for _, daily := range dailyTotals {
				totalDailyEnergy += daily
			}
			collectorData.HistoricalAvg = totalDailyEnergy / float64(len(dailyTotals))
		} else {
			collectorData.HistoricalAvg = 0
		}

		// Calculate this collector's share of total prediction
		var collectorShare float64
		if collectorData.HistoricalAvg > 0 {
			collectorShare = collectorData.HistoricalAvg
		} else {
			collectorShare = totalPrediction / float64(len(collectorIDs))
		}

		predictions = append(predictions, CollectorPrediction{
			CollectorID:        collectorID,
			CollectorName:      collectorData.Name,
			PredictedEnergyKWh: collectorShare,
			CurrentEnergyKWh:   collectorData.CurrentEnergy,
			PercentageOfTotal:  (collectorShare / totalPrediction) * 100,
		})
	}

	return predictions
}

func calculatePredictionMetrics(collectorIDs []string, todayStart, now time.Time, algorithm string) PredictionMetrics {
	// Calculate basic metrics
	var dataPoints int64
	model.DB.Model(&model.PowerData{}).
		Where("collector_id IN ? AND timestamp >= ?", collectorIDs, todayStart.AddDate(0, 0, -7)).
		Count(&dataPoints)

	dataQuality := "good"
	if dataPoints < 100 {
		dataQuality = "limited"
	} else if dataPoints > 1000 {
		dataQuality = "excellent"
	}

	return PredictionMetrics{
		Algorithm:          algorithm,
		DataQuality:        dataQuality,
		HistoricalAccuracy: 85.0, // Would be calculated from past predictions
		TrendStrength:      0.7,
		SeasonalityScore:   0.6,
		NoiseLevel:         0.2,
	}
}

func generatePredictionRecommendations(predictedEnergy, currentEnergy float64, now time.Time, hasCurrentData bool) []string {
	var recommendations []string

	// If no current data, provide different recommendations
	if !hasCurrentData {
		recommendations = append(recommendations, "Prediction based on historical patterns only. Monitor actual consumption for accuracy.")

		if predictedEnergy > 50 {
			recommendations = append(recommendations, "High energy consumption predicted for today based on historical data. Consider load optimization.")
		}

		currentHour := now.Hour()
		if currentHour >= 18 && currentHour <= 22 {
			recommendations = append(recommendations, "Peak hours detected. Consider deferring non-critical loads.")
		}

		return recommendations
	}

	// With current data, provide detailed recommendations
	timeElapsed := time.Since(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())).Hours()
	expectedByNow := predictedEnergy * (timeElapsed / 24.0)

	if currentEnergy > expectedByNow*1.2 {
		recommendations = append(recommendations, "Energy consumption is 20% higher than predicted. Consider reducing non-essential loads.")
	} else if currentEnergy < expectedByNow*0.8 {
		recommendations = append(recommendations, "Energy consumption is lower than predicted. Good energy management!")
	}

	if predictedEnergy > 50 {
		recommendations = append(recommendations, "High energy consumption predicted for today. Consider load balancing.")
	}

	currentHour := now.Hour()
	if currentHour >= 18 && currentHour <= 22 {
		recommendations = append(recommendations, "Peak hours detected. Consider deferring non-critical loads.")
	}

	return recommendations
}

func calculateLinearRegression(data []struct {
	Hour   int     `json:"hour"`
	Energy float64 `json:"energy"`
}) (slope, intercept float64) {
	n := float64(len(data))
	if n < 2 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for _, point := range data {
		x := float64(point.Hour)
		y := point.Energy
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

func calculateRSquared(data []struct {
	Hour   int     `json:"hour"`
	Energy float64 `json:"energy"`
}, slope, intercept float64) float64 {
	if len(data) < 2 {
		return 0
	}

	// Calculate mean of y values
	var sumY float64
	for _, point := range data {
		sumY += point.Energy
	}
	meanY := sumY / float64(len(data))

	// Calculate sum of squares
	var ssRes, ssTot float64
	for _, point := range data {
		predicted := slope*float64(point.Hour) + intercept
		ssRes += math.Pow(point.Energy-predicted, 2)
		ssTot += math.Pow(point.Energy-meanY, 2)
	}

	if ssTot == 0 {
		return 1
	}

	return 1 - (ssRes / ssTot)
}

func calculateVariance(data []struct {
	Date   time.Time `json:"date"`
	Energy float64   `json:"energy"`
}) float64 {
	if len(data) < 2 {
		return 0
	}

	var sum float64
	for _, point := range data {
		sum += point.Energy
	}
	mean := sum / float64(len(data))

	var variance float64
	for _, point := range data {
		variance += math.Pow(point.Energy-mean, 2)
	}

	return variance / float64(len(data))
}

func getPredictionAccuracyLabel(confidence float64) string {
	if confidence >= 90 {
		return "very_high"
	} else if confidence >= 75 {
		return "high"
	} else if confidence >= 60 {
		return "medium"
	} else if confidence >= 40 {
		return "low"
	}
	return "very_low"
}
