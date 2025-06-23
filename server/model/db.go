package model

import (
	"gorm.io/gorm"
)

// DB is the global database instance
var DB *gorm.DB

// SetDB sets the global database instance
func SetDB(db *gorm.DB) {
	DB = db
}
