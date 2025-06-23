package model

import (
	"time"

	"gorm.io/gorm"
)

// GenerateAllModel returns all models for auto-migration
func GenerateAllModel() []interface{} {
	return []interface{}{
		User{},
		Collector{},
		RegistrationCode{},
		PowerData{},
		CollectorConfig{},
		AuthToken{},
	}
}

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total    int64 `json:"total"`
	Current  int   `json:"current"`
	PageSize int   `json:"pageSize"`
}

// ListResponse represents a standardized list response
type ListResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// StandardResponse represents a standardized API response
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
