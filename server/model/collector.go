package model

import (
	"time"
)

// Collector represents a power data collector device
type Collector struct {
	BaseModel
	CollectorID string    `gorm:"uniqueIndex;not null" json:"collector_id" binding:"required"`
	Name        string    `gorm:"not null" json:"name" binding:"required"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	Token       string    `gorm:"uniqueIndex" json:"-"`
	Version     string    `json:"version"`
	IPAddress   string    `json:"ip_address"`
	UserID      uint      `json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// CollectorConfig represents configuration for a collector
type CollectorConfig struct {
	BaseModel
	CollectorID      string    `gorm:"not null" json:"collector_id"`
	SampleInterval   int       `gorm:"default:15" json:"sample_interval"`  // seconds
	UploadInterval   int       `gorm:"default:60" json:"upload_interval"`  // seconds
	MaxCacheSize     int       `gorm:"default:1000" json:"max_cache_size"` // number of records
	AutoUpload       bool      `gorm:"default:true" json:"auto_upload"`
	CompressionLevel int       `gorm:"default:6" json:"compression_level"`
	Collector        Collector `gorm:"foreignKey:CollectorID;references:CollectorID" json:"collector,omitempty"`
}

// RegistrationCode represents a code for registering new collectors
type RegistrationCode struct {
	BaseModel
	Code        string    `gorm:"uniqueIndex;not null" json:"code"`
	Description string    `json:"description"`
	UsedBy      string    `json:"used_by"`
	IsUsed      bool      `gorm:"default:false" json:"is_used"`
	ExpiresAt   time.Time `json:"expires_at"`
	UserID      uint      `json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PowerData represents power measurement data (cached in database for quick access)
type PowerData struct {
	BaseModel
	CollectorID string    `gorm:"index;not null" json:"collector_id"`
	Timestamp   time.Time `gorm:"index;not null" json:"timestamp"`
	Voltage     float64   `json:"voltage"`   // Volts
	Current     float64   `json:"current"`   // Amperes
	Power       float64   `json:"power"`     // Watts
	Energy      float64   `json:"energy"`    // kWh
	Frequency   float64   `json:"frequency"` // Hz
	PowerFactor float64   `json:"power_factor"`
	Collector   Collector `gorm:"foreignKey:CollectorID;references:CollectorID" json:"collector,omitempty"`
}

// CollectorCreateRequest represents request for creating a collector
type CollectorCreateRequest struct {
	CollectorID string `json:"collector_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

// CollectorUpdateRequest represents request for updating a collector
type CollectorUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	IsActive    bool   `json:"is_active"`
}

// CollectorRegisterRequest represents request for registering a collector
type CollectorRegisterRequest struct {
	RegistrationCode string `json:"registration_code" binding:"required"`
	CollectorID      string `json:"collector_id" binding:"required"`
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	Location         string `json:"location"`
	Version          string `json:"version"`
}

// CollectorRegisterResponse represents response for collector registration
type CollectorRegisterResponse struct {
	Token  string          `json:"token"`
	Config CollectorConfig `json:"config"`
}

// CollectorStatusResponse represents collector status
type CollectorStatusResponse struct {
	Collector    Collector `json:"collector"`
	IsOnline     bool      `json:"is_online"`
	LastDataTime time.Time `json:"last_data_time"`
	DataCount    int64     `json:"data_count"`
}

// PowerDataUploadRequest represents bulk power data upload
type PowerDataUploadRequest struct {
	CollectorID string             `json:"collector_id" binding:"required"`
	Data        []PowerDataRequest `json:"data" binding:"required"`
}

// PowerDataRequest represents single power data measurement
type PowerDataRequest struct {
	Timestamp   time.Time `json:"timestamp" binding:"required"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Energy      float64   `json:"energy"`
	Frequency   float64   `json:"frequency"`
	PowerFactor float64   `json:"power_factor"`
}

// IsOnline checks if collector is online (last seen within 5 minutes)
func (c *Collector) IsOnline() bool {
	return time.Since(c.LastSeenAt) < 5*time.Minute
}
