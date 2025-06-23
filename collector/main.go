package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"power-collector/pkg/client"
	"power-collector/pkg/collector"
	"power-collector/pkg/config"

	"github.com/google/uuid"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	gitHash   = "unknown"
)

func main() {
	// Command line flags
	var (
		configFile  = flag.String("config", "config.ini", "Configuration file path")
		showVersion = flag.Bool("version", false, "Show version information")
		testMode    = flag.Bool("test", false, "Run in test mode to collect data once and print")
	)
	flag.Parse()

	// Show version information
	if *showVersion {
		fmt.Printf("Power Collector v%s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Hash: %s\n", gitHash)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// If test mode is enabled, run a single collection and exit
	if *testMode {
		log.Println("Running in test mode...")
		service, err := collector.NewCollectorService(cfg, version)
		if err != nil {
			log.Fatalf("Failed to create collector service for testing: %v", err)
		}
		if err := service.Test(); err != nil {
			log.Fatalf("Test failed: %v", err)
		}
		return
	}

	// If no token, but has registration code, register with server
	if cfg.Auth.Token == "" && cfg.Auth.RegistrationCode != "" {
		log.Println("No token found, attempting to register with server using registration code...")

		// Generate collector ID if not provided
		if cfg.Collector.ID == "" {
			cfg.Collector.ID = uuid.New().String()
			log.Printf("Generated new collector ID: %s", cfg.Collector.ID)
		}

		apiClient := client.NewAPIClient(cfg.Server.BaseURL, cfg.Server.APIPrefix, cfg.Server.Timeout*time.Second)
		req := client.RegisterRequest{
			RegistrationCode: cfg.Auth.RegistrationCode,
			CollectorID:      cfg.Collector.ID,
			Name:             cfg.Collector.Name,
			Description:      cfg.Collector.Description,
			Location:         cfg.Collector.Location,
			Version:          version,
		}

		resp, err := apiClient.Register(req)
		if err != nil {
			log.Fatalf("Failed to register collector: %v", err)
		}

		log.Println("Collector registered successfully.")

		// Update config with data from server
		cfg.Auth.Token = resp.Data.Token
		if resp.Data.Config.CollectorID != "" {
			cfg.Collector.ID = resp.Data.Config.CollectorID
		}
		if resp.Data.Config.SampleInterval > 0 {
			cfg.Serial.SampleInterval = time.Duration(resp.Data.Config.SampleInterval)
		}
		if resp.Data.Config.UploadInterval > 0 {
			cfg.Data.UploadInterval = time.Duration(resp.Data.Config.UploadInterval)
		}
		if resp.Data.Config.MaxCacheSize > 0 {
			cfg.Data.MaxCacheSize = resp.Data.Config.MaxCacheSize
		}
		cfg.Data.AutoUpload = resp.Data.Config.AutoUpload

		// Save updated config
		if err := config.SaveConfig(cfg, *configFile); err != nil {
			log.Fatalf("Failed to save updated configuration: %v", err)
		}
		log.Printf("Configuration updated and saved to %s", *configFile)
	}

	log.Printf("Starting Power Collector v%s", version)
	log.Printf("Collector ID: %s", cfg.Collector.ID)
	log.Printf("Collector Name: %s", cfg.Collector.Name)

	// Create collector service
	service, err := collector.NewCollectorService(cfg, version)
	if err != nil {
		log.Fatalf("Failed to create collector service: %v", err)
	}

	// Start the service
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start collector service: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	log.Println("Collector is running. Press Ctrl+C to stop.")
	sig := <-sigChan

	log.Printf("Received signal %v, stopping collector...", sig)

	// Stop the service gracefully with a timeout
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := service.Stop(); err != nil {
			log.Printf("Error stopping collector service: %v", err)
		}
	}()

	select {
	case <-done:
		log.Println("Collector stopped gracefully.")
	case <-stopCtx.Done():
		log.Println("Graceful shutdown timed out after 10 seconds. Forcing exit.")
	}
}
