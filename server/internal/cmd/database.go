package cmd

import (
	"path"

	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/uozi-tech/cosy"
	sqlite "github.com/uozi-tech/cosy-driver-sqlite"
	cModel "github.com/uozi-tech/cosy/model"
	"gorm.io/gorm"
)

// initDatabase initializes the database connection
func initDatabase(confPath string) (*gorm.DB, error) {
	settings.Init(confPath)
	cosy.RegisterModels(model.GenerateAllModel()...)

	// Initialize database similar to kernel
	cModel.ResolvedModels()
	db := cosy.InitDB(sqlite.Open(path.Dir(confPath), settings.DatabaseSettings))
	model.SetDB(db)

	return db, nil
}
