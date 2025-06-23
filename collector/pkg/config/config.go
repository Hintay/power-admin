package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/ini.v1"
)

// Config represents the application configuration
type Config struct {
	Collector CollectorConfig `ini:"collector"`
	Serial    SerialConfig    `ini:"serial"`
	Server    ServerConfig    `ini:"server"`
	Auth      AuthConfig      `ini:"auth"`
	Data      DataConfig      `ini:"data"`
	Logging   LoggingConfig   `ini:"logging"`
}

// CollectorConfig represents collector-specific configuration
type CollectorConfig struct {
	ID          string `ini:"id"`
	Name        string `ini:"name"`
	Description string `ini:"description"`
	Location    string `ini:"location"`
}

// SerialConfig represents serial port configuration
type SerialConfig struct {
	Port           string        `ini:"port"`
	BaudRate       int           `ini:"baud_rate"`
	SampleInterval time.Duration `ini:"sample_interval"`
	Timeout        time.Duration `ini:"timeout"`
}

// ServerConfig represents server connection configuration
type ServerConfig struct {
	BaseURL       string        `ini:"base_url"`
	APIPrefix     string        `ini:"api_prefix"`
	Timeout       time.Duration `ini:"timeout"`
	RetryInterval time.Duration `ini:"retry_interval"`
	MaxRetries    int           `ini:"max_retries"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Token            string `ini:"token"`
	RegistrationCode string `ini:"registration_code"`
}

// DataConfig represents data handling configuration
type DataConfig struct {
	CacheDB           string        `ini:"cache_db"`
	MaxCacheSize      int           `ini:"max_cache_size"`
	BatchSize         int           `ini:"batch_size"`
	UploadInterval    time.Duration `ini:"upload_interval"`
	AutoUpload        bool          `ini:"auto_upload"`
	EnableCompression bool          `ini:"enable_compression"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `ini:"level"`
	File       string `ini:"file"`
	MaxSize    int    `ini:"max_size"`
	MaxBackups int    `ini:"max_backups"`
	MaxAge     int    `ini:"max_age"`
}

var globalConfig *Config

// LoadConfig loads configuration from file
func LoadConfig(configFile string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configFile)
	}

	cfg, err := ini.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	config := &Config{}
	if err := cfg.MapTo(config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	globalConfig = config
	return config, nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return globalConfig
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	if config.Collector.Name == "" {
		return fmt.Errorf("collector name is required")
	}

	if config.Serial.Port == "" {
		return fmt.Errorf("serial port is required")
	}

	if config.Serial.BaudRate <= 0 {
		return fmt.Errorf("invalid baud rate: %d", config.Serial.BaudRate)
	}

	if config.Server.BaseURL == "" {
		return fmt.Errorf("server base URL is required")
	}

	if config.Auth.Token == "" && config.Auth.RegistrationCode == "" {
		return fmt.Errorf("either token or registration code is required")
	}

	if config.Data.MaxCacheSize <= 0 {
		config.Data.MaxCacheSize = 10000
	}

	if config.Data.BatchSize <= 0 {
		config.Data.BatchSize = 100
	}

	return nil
}

// SaveConfig saves current configuration to file
func SaveConfig(config *Config, configFile string) error {
	cfg := ini.Empty()

	if err := cfg.ReflectFrom(config); err != nil {
		return fmt.Errorf("failed to convert config to ini: %w", err)
	}

	if err := cfg.SaveTo(configFile); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}
