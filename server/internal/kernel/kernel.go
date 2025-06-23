package kernel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"time"

	"Power-Monitor/internal/auth"
	"Power-Monitor/internal/influxdb"
	"Power-Monitor/internal/realtime"
	"Power-Monitor/model"
	"Power-Monitor/settings"

	"github.com/uozi-tech/cosy"
	sqlite "github.com/uozi-tech/cosy-driver-sqlite"
	"github.com/uozi-tech/cosy/logger"
	cModel "github.com/uozi-tech/cosy/model"
	cSettings "github.com/uozi-tech/cosy/settings"
)

// Boot initializes the system kernel and core services
func Boot(ctx context.Context) {
	logger.Info("Starting Power Monitor kernel initialization...")

	// Initialize SQLite database first
	initDatabase(ctx)

	// Initialize InfluxDB client
	if settings.InfluxDBSettings.Enabled {
		initInfluxDB()
	}

	// Initialize authentication service
	initAuthService()

	// Initialize realtime service
	initRealtimeService(ctx)

	// Create default admin user if none exists
	createDefaultAdmin()

	// Start background services
	startBackgroundServices(ctx)

	logger.Info("Power Monitor kernel initialization completed")
}

// initDatabase initializes the SQLite database for storing metadata and configurations
func initDatabase(ctx context.Context) {
	logger.Info("Initializing SQLite database...")

	// Resolve cosy models
	cModel.ResolvedModels()

	// Initialize cosy database with SQLite driver
	dbPath := path.Dir(cSettings.ConfPath)
	db := cosy.InitDB(sqlite.Open(dbPath, settings.DatabaseSettings))

	// Set the global database instance
	model.SetDB(db)
}

// initInfluxDB initializes the InfluxDB v3 client
func initInfluxDB() {
	if !settings.InfluxDBSettings.Enabled {
		logger.Info("InfluxDB is disabled by settings")
		return
	}
	logger.Info("Initializing InfluxDB v3 client...")

	err := influxdb.Init(
		settings.InfluxDBSettings.Host,
		settings.InfluxDBSettings.Port,
		settings.InfluxDBSettings.Token,
		settings.InfluxDBSettings.Database,
		settings.InfluxDBSettings.Timeout,
		settings.InfluxDBSettings.UseSSL,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize InfluxDB v3: %v", err)
	}

	logger.Info("InfluxDB v3 client initialized successfully")
}

// initAuthService initializes the authentication service
func initAuthService() {
	logger.Info("Initializing authentication service...")

	auth.Init(
		settings.FrontendSettings.JwtSecret,
		settings.FrontendSettings.AccessTokenExpires,
		settings.FrontendSettings.RefreshTokenExpires,
	)

	logger.Info("Authentication service initialized successfully")
}

// initRealtimeService initializes the realtime communication service
func initRealtimeService(ctx context.Context) {
	logger.Info("Initializing realtime service...")

	realtime.Init(ctx)

	logger.Info("Realtime service initialized successfully")
}

// createDefaultAdmin creates a default admin user if no users exist
func createDefaultAdmin() {
	logger.Info("Checking for existing users...")

	var count int64
	if err := model.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		logger.Errorf("Failed to count users: %v", err)
		return
	}

	if count > 0 {
		logger.Info("Users already exist, skipping default admin creation")
		return
	}

	logger.Info("No users found, creating default admin user...")

	// Generate random password
	password := generateRandomPassword()

	admin := &model.User{
		Username: "admin",
		Email:    "admin@power-monitor.local",
		FullName: "System Administrator",
		Role:     "admin",
		Active:   true,
	}

	if err := admin.HashPassword(password); err != nil {
		logger.Errorf("Failed to hash admin password: %v", err)
		return
	}

	if err := model.DB.Create(admin).Error; err != nil {
		logger.Errorf("Failed to create admin user: %v", err)
		return
	}

	logger.Infof("Default admin user created successfully")
	logger.Infof("Username: admin")
	logger.Infof("Password: %s", password)
	logger.Infof("Please change the password after first login!")
}

// startBackgroundServices starts background services
func startBackgroundServices(ctx context.Context) {
	logger.Info("Starting background services...")

	// Start token cleanup service
	go startTokenCleanupService(ctx)

	// Start collector status check service
	go startCollectorStatusService(ctx)

	logger.Info("Background services started successfully")
}

// startTokenCleanupService starts the token cleanup service
func startTokenCleanupService(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupExpiredTokens()
		}
	}
}

// startCollectorStatusService starts the collector status monitoring service
func startCollectorStatusService(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateCollectorStatus()
		}
	}
}

// cleanupExpiredTokens removes expired tokens from the database
func cleanupExpiredTokens() {
	result := model.DB.Where("expires_at < ?", time.Now()).Delete(&model.AuthToken{})
	if result.Error != nil {
		logger.Errorf("Failed to cleanup expired tokens: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		logger.Infof("Cleaned up %d expired tokens", result.RowsAffected)
	}
}

// updateCollectorStatus updates collector status based on last seen time
func updateCollectorStatus() {
	// This is a placeholder for collector status monitoring
	// In a full implementation, you might want to check collector heartbeats
	// and update their status accordingly
	logger.Debug("Checking collector status...")
}

// generateRandomPassword generates a random password
func generateRandomPassword() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based password
		return fmt.Sprintf("admin%d", time.Now().Unix())
	}
	return hex.EncodeToString(bytes)[:16]
}
