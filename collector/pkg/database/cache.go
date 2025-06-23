package database

import (
	"fmt"
	"time"

	"power-collector/pkg/pzem"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PowerDataCache represents cached power data for offline storage
type PowerDataCache struct {
	ID          uint      `gorm:"primaryKey"`
	CollectorID string    `gorm:"index;not null"`
	Timestamp   time.Time `gorm:"not null"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Energy      float64   `json:"energy"`
	Frequency   float64   `json:"frequency"`
	PowerFactor float64   `json:"power_factor"`
	Uploaded    bool      `gorm:"default:false;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CacheDB represents the local cache database
type CacheDB struct {
	db *gorm.DB
}

// NewCacheDB creates a new cache database instance
func NewCacheDB(dbPath string) (*CacheDB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Reduce log noise
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&PowerDataCache{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	return &CacheDB{db: db}, nil
}

// Close closes the database connection
func (c *CacheDB) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// StorePowerData stores power data to cache
func (c *CacheDB) StorePowerData(collectorID string, data interface{}) error {
	// Handle different data types (from PZEM or API response)
	var cache PowerDataCache

	switch v := data.(type) {
	case *pzem.PowerData:
		cache = PowerDataCache{
			CollectorID: collectorID,
			Timestamp:   v.Timestamp,
			Voltage:     v.Voltage,
			Current:     v.Current,
			Power:       v.Power,
			Energy:      v.Energy,
			Frequency:   v.Frequency,
			PowerFactor: v.PowerFactor,
			Uploaded:    false,
		}
	case map[string]interface{}:
		// From JSON/API response
		cache = PowerDataCache{
			CollectorID: collectorID,
			Timestamp:   parseTimestamp(v["timestamp"]),
			Voltage:     parseFloat64(v["voltage"]),
			Current:     parseFloat64(v["current"]),
			Power:       parseFloat64(v["power"]),
			Energy:      parseFloat64(v["energy"]),
			Frequency:   parseFloat64(v["frequency"]),
			PowerFactor: parseFloat64(v["power_factor"]),
			Uploaded:    false,
		}
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}

	if err := c.db.Create(&cache).Error; err != nil {
		return fmt.Errorf("failed to store cache data: %w", err)
	}

	return nil
}

// GetUnuploadedData retrieves data that hasn't been uploaded yet
func (c *CacheDB) GetUnuploadedData(limit int) ([]PowerDataCache, error) {
	var data []PowerDataCache
	err := c.db.Where("uploaded = ?", false).
		Order("timestamp ASC").
		Limit(limit).
		Find(&data).Error

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve unuploaded data: %w", err)
	}

	return data, nil
}

// MarkAsUploaded marks data records as uploaded
func (c *CacheDB) MarkAsUploaded(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	err := c.db.Model(&PowerDataCache{}).
		Where("id IN ?", ids).
		Update("uploaded", true).Error

	if err != nil {
		return fmt.Errorf("failed to mark data as uploaded: %w", err)
	}

	return nil
}

// CleanupOldData removes old uploaded data to prevent database growth
func (c *CacheDB) CleanupOldData(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	result := c.db.Where("uploaded = ? AND updated_at < ?", true, cutoff).
		Delete(&PowerDataCache{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old data: %w", result.Error)
	}

	return nil
}

// GetCacheStats returns statistics about cached data
func (c *CacheDB) GetCacheStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total records
	var total int64
	if err := c.db.Model(&PowerDataCache{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total records: %w", err)
	}
	stats["total"] = total

	// Unuploaded records
	var unuploaded int64
	if err := c.db.Model(&PowerDataCache{}).Where("uploaded = ?", false).Count(&unuploaded).Error; err != nil {
		return nil, fmt.Errorf("failed to count unuploaded records: %w", err)
	}
	stats["unuploaded"] = unuploaded

	// Uploaded records
	stats["uploaded"] = total - unuploaded

	return stats, nil
}

// GetLatestData returns the most recent cached data
func (c *CacheDB) GetLatestData(collectorID string, limit int) ([]PowerDataCache, error) {
	var data []PowerDataCache
	err := c.db.Where("collector_id = ?", collectorID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&data).Error

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve latest data: %w", err)
	}

	return data, nil
}

// ToAPIFormat converts cache data to API request format
func (data *PowerDataCache) ToAPIFormat() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":    data.Timestamp.Format(time.RFC3339),
		"voltage":      data.Voltage,
		"current":      data.Current,
		"power":        data.Power,
		"energy":       data.Energy,
		"frequency":    data.Frequency,
		"power_factor": data.PowerFactor,
	}
}

// Helper functions
func parseTimestamp(v interface{}) time.Time {
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	case time.Time:
		return val
	}
	return time.Now()
}

func parseFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0.0
}
