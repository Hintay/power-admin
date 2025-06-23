package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	configContent := `
[collector]
id = test-collector
name = Test Collector
description = Test Description
location = Test Location
version = 1.0.0

[serial]
port = /dev/ttyUSB0
baud_rate = 9600
sample_interval = 15
timeout = 2

[server]
base_url = http://localhost:8080
api_prefix = /root/v1
timeout = 30
retry_interval = 60
max_retries = 5

[auth]
token = test-token
registration_code = REG-TEST-001

[data]
cache_db = ./test_cache.db
max_cache_size = 1000
batch_size = 50
upload_interval = 30
auto_upload = true
enable_compression = true

[logging]
level = info
file = ./test.log
max_size = 10
max_backups = 3
max_age = 30
`

	// Write test config file
	tempFile, err := os.CreateTemp("", "test_config_*.ini")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tempFile.Close()

	// Test loading configuration
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration values
	if cfg.Collector.ID != "test-collector" {
		t.Errorf("Expected collector ID 'test-collector', got '%s'", cfg.Collector.ID)
	}

	if cfg.Collector.Name != "Test Collector" {
		t.Errorf("Expected collector name 'Test Collector', got '%s'", cfg.Collector.Name)
	}

	if cfg.Serial.Port != "/dev/ttyUSB0" {
		t.Errorf("Expected serial port '/dev/ttyUSB0', got '%s'", cfg.Serial.Port)
	}

	if cfg.Serial.BaudRate != 9600 {
		t.Errorf("Expected baud rate 9600, got %d", cfg.Serial.BaudRate)
	}

	if cfg.Serial.SampleInterval != 15 {
		t.Errorf("Expected sample interval 15, got %v", cfg.Serial.SampleInterval)
	}

	if cfg.Serial.Timeout != 2 {
		t.Errorf("Expected serial timeout 2, got %v", cfg.Serial.Timeout)
	}

	if cfg.Server.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected base URL 'http://localhost:8080', got '%s'", cfg.Server.BaseURL)
	}

	if cfg.Server.Timeout != 30 {
		t.Errorf("Expected server timeout 30, got %v", cfg.Server.Timeout)
	}

	if cfg.Server.RetryInterval != 60 {
		t.Errorf("Expected server retry interval 60, got %v", cfg.Server.RetryInterval)
	}

	if cfg.Auth.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", cfg.Auth.Token)
	}

	if cfg.Data.MaxCacheSize != 1000 {
		t.Errorf("Expected max cache size 1000, got %d", cfg.Data.MaxCacheSize)
	}

	if cfg.Data.UploadInterval != 30 {
		t.Errorf("Expected upload interval 30, got %v", cfg.Data.UploadInterval)
	}

	if !cfg.Data.AutoUpload {
		t.Error("Expected auto upload to be true")
	}
	globalConfig = nil
}

func TestLoadConfigWithMissingFile(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.ini")
	if err == nil {
		t.Error("Expected error when loading nonexistent config file")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: &Config{
				Collector: CollectorConfig{
					ID:   "test-id",
					Name: "test-name",
				},
				Serial: SerialConfig{
					Port:     "/dev/ttyUSB0",
					BaudRate: 9600,
				},
				Server: ServerConfig{
					BaseURL: "http://localhost:8080",
				},
				Auth: AuthConfig{
					Token: "test-token",
				},
			},
			expectError: false,
		},
		{
			name: "Missing collector ID is not an error",
			config: &Config{
				Collector: CollectorConfig{
					Name: "test-name",
				},
				Serial: SerialConfig{
					Port:     "/dev/ttyUSB0",
					BaudRate: 9600,
				},
				Server: ServerConfig{
					BaseURL: "http://localhost:8080",
				},
				Auth: AuthConfig{
					Token: "test-token",
				},
			},
			expectError: false,
		},
		{
			name: "Missing collector name",
			config: &Config{
				Collector: CollectorConfig{
					ID: "test-id",
				},
				Serial: SerialConfig{
					Port:     "/dev/ttyUSB0",
					BaudRate: 9600,
				},
				Server: ServerConfig{
					BaseURL: "http://localhost:8080",
				},
				Auth: AuthConfig{
					Token: "test-token",
				},
			},
			expectError: true,
		},
		{
			name: "Invalid baud rate",
			config: &Config{
				Collector: CollectorConfig{
					ID:   "test-id",
					Name: "test-name",
				},
				Serial: SerialConfig{
					Port:     "/dev/ttyUSB0",
					BaudRate: -1,
				},
				Server: ServerConfig{
					BaseURL: "http://localhost:8080",
				},
				Auth: AuthConfig{
					Token: "test-token",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	// Test when no config is loaded
	globalConfig = nil
	cfg := GetConfig()
	if cfg != nil {
		t.Error("Expected nil config when none is loaded")
	}

	// Load a config and test GetConfig
	testConfig := &Config{
		Collector: CollectorConfig{
			ID: "test-global",
		},
	}
	globalConfig = testConfig

	cfg = GetConfig()
	if cfg == nil {
		t.Error("Expected config to be returned")
	}
	if cfg.Collector.ID != "test-global" {
		t.Errorf("Expected collector ID 'test-global', got '%s'", cfg.Collector.ID)
	}

	// Reset global config
	globalConfig = nil
}
