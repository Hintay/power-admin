package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"power-collector/pkg/client"
	"power-collector/pkg/config"
	"power-collector/pkg/database"
	"power-collector/pkg/pzem"
)

// CollectorService represents the main collector service
type CollectorService struct {
	config     *config.Config
	version    string
	apiClient  *client.APIClient
	pzemDevice *pzem.PZEM004T
	cacheDB    *database.CacheDB
	isRunning  bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex

	// Status tracking
	isRegistered bool
	isOnline     bool
	lastDataTime time.Time
	errorCount   int
}

// ServiceStatus represents the current status of the collector service
type ServiceStatus struct {
	IsRunning    bool             `json:"is_running"`
	IsRegistered bool             `json:"is_registered"`
	IsOnline     bool             `json:"is_online"`
	LastDataTime time.Time        `json:"last_data_time"`
	ErrorCount   int              `json:"error_count"`
	CacheStats   map[string]int64 `json:"cache_stats,omitempty"`
}

// NewCollectorService creates a new collector service instance
func NewCollectorService(cfg *config.Config, version string) (*CollectorService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &CollectorService{
		config:   cfg,
		version:  version,
		stopChan: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize components
	if err := service.initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize collector service: %w", err)
	}

	return service, nil
}

// Test performs a single data collection and prints the result to the console.
// This is intended for testing the connection to the PZEM device.
func (c *CollectorService) Test() error {
	log.Println("Performing a single data collection test...")

	// The pzemDevice is initialized in NewCollectorService.
	// We need to ensure it's closed after the test.
	defer func() {
		if err := c.pzemDevice.Close(); err != nil {
			log.Printf("Error closing PZEM device during test: %v", err)
		}
	}()

	// Read data from PZEM-004T
	powerData, err := c.pzemDevice.ReadDataWithRetry(3)
	if err != nil {
		return fmt.Errorf("failed to read data from PZEM-004T during test: %w", err)
	}

	// Validate data
	if !powerData.IsDataValid() {
		return fmt.Errorf("invalid data received from PZEM-004T during test: %v", powerData)
	}

	// Print the data
	fmt.Println("--- Test Collection Result ---")
	fmt.Printf("  Timestamp:   %s\n", powerData.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Voltage:     %.2f V\n", powerData.Voltage)
	fmt.Printf("  Current:     %.3f A\n", powerData.Current)
	fmt.Printf("  Power:       %.2f W\n", powerData.Power)
	fmt.Printf("  Energy:      %.3f kWh\n", powerData.Energy)
	fmt.Printf("  Frequency:   %.1f Hz\n", powerData.Frequency)
	fmt.Printf("  Power Factor: %.2f\n", powerData.PowerFactor)
	fmt.Println("------------------------------")
	log.Println("Test completed successfully.")

	return nil
}

// initialize initializes all required components
func (c *CollectorService) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize API client
	c.apiClient = client.NewAPIClient(
		c.config.Server.BaseURL,
		c.config.Server.APIPrefix,
		c.config.Server.Timeout*time.Second,
	)

	// Initialize PZEM-004T device
	pzemDevice, err := pzem.NewPZEM004T(c.config.Serial.Port, c.config.Serial.BaudRate, 0x01, c.config.Serial.Timeout*time.Second) // Assuming default address 0x01
	if err != nil {
		return fmt.Errorf("failed to initialize PZEM-004T device: %w", err)
	}
	c.pzemDevice = pzemDevice

	// Initialize cache database
	cacheDB, err := database.NewCacheDB(c.config.Data.CacheDB)
	if err != nil {
		return fmt.Errorf("failed to initialize cache database: %w", err)
	}
	c.cacheDB = cacheDB

	log.Println("Collector service initialized successfully")
	return nil
}

// Start starts the collector service
func (c *CollectorService) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return fmt.Errorf("collector service is already running")
	}

	// Perform registration if needed
	if err := c.ensureRegistration(); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Test server connection
	log.Println("Testing connection to the server...")
	if err := c.apiClient.TestConnection(); err != nil {
		return fmt.Errorf("server connection test failed: %w", err)
	}
	log.Println("Server connection successful.")

	c.isRunning = true
	log.Println("Starting collector service...")

	// Start background goroutines
	c.wg.Add(4)
	go c.dataCollectionLoop()
	go c.dataUploadLoop()
	go c.heartbeatLoop()
	go c.maintenanceLoop()

	log.Println("Collector service started successfully")
	return nil
}

// Stop stops the collector service gracefully
func (c *CollectorService) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return nil
	}

	log.Println("Stopping collector service...")
	c.isRunning = false

	// Signal all goroutines to stop
	close(c.stopChan)
	c.cancel()

	// Wait for all goroutines to finish
	c.wg.Wait()

	// Close resources
	if c.pzemDevice != nil {
		if err := c.pzemDevice.Close(); err != nil {
			log.Printf("Error closing PZEM device: %v", err)
		}
	}
	if c.cacheDB != nil {
		if err := c.cacheDB.Close(); err != nil {
			log.Printf("Error closing cache DB: %v", err)
		}
	}

	log.Println("Collector service stopped")
	return nil
}

// ensureRegistration ensures the collector is registered with the server
func (c *CollectorService) ensureRegistration() error {
	if c.config.Auth.Token == "" || c.config.Collector.ID == "" {
		return fmt.Errorf("auth token or collector ID is missing, please register first")
	}

	c.apiClient.SetToken(c.config.Auth.Token, c.config.Collector.ID)
	c.isRegistered = true
	log.Println("Authentication credentials set for API client.")

	return nil
}

// dataCollectionLoop handles periodic data collection from PZEM-004T
func (c *CollectorService) dataCollectionLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.Serial.SampleInterval * time.Second)
	defer ticker.Stop()

	log.Printf("Starting data collection loop (interval: %v)", c.config.Serial.SampleInterval*time.Second)

	for {
		select {
		case <-c.stopChan:
			log.Println("Data collection loop stopped")
			return
		case <-ticker.C:
			if err := c.collectData(); err != nil {
				c.handleError("data collection", err)
			}
		}
	}
}

// collectData collects data from PZEM-004T and attempts real-time upload or caches it.
func (c *CollectorService) collectData() error {
	// Read data from PZEM-004T with retries
	powerData, err := c.pzemDevice.ReadDataWithRetry(3)
	if err != nil {
		return fmt.Errorf("failed to read data from PZEM-004T: %w", err)
	}

	// Validate data
	if !powerData.IsDataValid() {
		return fmt.Errorf("invalid data received from PZEM-004T: %v", powerData)
	}

	// Attempt to upload data in real-time
	apiData := client.PowerDataRequest{
		Timestamp:   powerData.Timestamp,
		Voltage:     powerData.Voltage,
		Current:     powerData.Current,
		Power:       powerData.Power,
		Energy:      powerData.Energy,
		Frequency:   powerData.Frequency,
		PowerFactor: powerData.PowerFactor,
	}

	if err := c.apiClient.UploadData(apiData); err != nil {
		// If upload fails, write to cache
		log.Printf("Real-time upload failed: %v. Caching data instead.", err)
		c.isOnline = false // Mark as offline since we couldn't upload
		if cacheErr := c.cacheDB.StorePowerData(c.config.Collector.ID, powerData); cacheErr != nil {
			return fmt.Errorf("failed to cache power data after upload failure: %w", cacheErr)
		}
		log.Printf("Data collected and cached successfully: %s", powerData.String())
	} else {
		// If upload succeeds, mark as online
		c.isOnline = true
		log.Printf("Data collected and uploaded successfully in real-time: %s", powerData.String())
	}

	c.mu.Lock()
	c.lastDataTime = time.Now()
	c.mu.Unlock()

	return nil // Return nil because we handled the error by caching
}

// dataUploadLoop handles periodic data upload to server
func (c *CollectorService) dataUploadLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.Data.UploadInterval * time.Second)
	defer ticker.Stop()

	log.Printf("Starting data upload loop (interval: %v)", c.config.Data.UploadInterval*time.Second)

	for {
		select {
		case <-c.stopChan:
			log.Println("Data upload loop stopped")
			return
		case <-ticker.C:
			if c.config.Data.AutoUpload {
				if err := c.uploadCachedData(); err != nil {
					c.handleError("data upload", err)
				}
			}
		}
	}
}

// uploadCachedData uploads cached data to server
func (c *CollectorService) uploadCachedData() error {
	// Get unuploaded data from cache
	cachedData, err := c.cacheDB.GetUnuploadedData(c.config.Data.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get unuploaded data from cache: %w", err)
	}

	if len(cachedData) == 0 {
		log.Println("No new data to upload.")
		return nil
	}

	log.Printf("Found %d records to upload.", len(cachedData))

	// Convert data to API format
	var apiData []client.PowerDataRequest
	var uploadedIDs []uint
	for _, item := range cachedData {
		apiData = append(apiData, client.PowerDataRequest{
			Timestamp:   item.Timestamp,
			Voltage:     item.Voltage,
			Current:     item.Current,
			Power:       item.Power,
			Energy:      item.Energy,
			Frequency:   item.Frequency,
			PowerFactor: item.PowerFactor,
		})
		uploadedIDs = append(uploadedIDs, item.ID)
	}

	// Upload batch data
	if err := c.apiClient.UploadBatchData(apiData); err != nil {
		c.isOnline = false
		return fmt.Errorf("failed to upload batch data: %w", err)
	}

	// Mark data as uploaded
	if err := c.cacheDB.MarkAsUploaded(uploadedIDs); err != nil {
		// This is a non-critical error, we log it but don't fail the entire upload
		log.Printf("Warning: failed to mark data as uploaded: %v", err)
	}

	c.isOnline = true
	log.Printf("Successfully uploaded %d data records.", len(apiData))
	return nil
}

// heartbeatLoop sends periodic heartbeat to server
func (c *CollectorService) heartbeatLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // Heartbeat every 5 minutes
	defer ticker.Stop()

	log.Println("Starting heartbeat loop")

	for {
		select {
		case <-c.stopChan:
			log.Println("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if c.isRegistered {
				if err := c.sendHeartbeat(); err != nil {
					c.handleError("heartbeat", err)
				}
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the server
func (c *CollectorService) sendHeartbeat() error {
	var status string
	if c.IsHealthy() {
		status = "ok"
	} else {
		status = "error"
	}

	err := c.apiClient.SendHeartbeat(status, c.version)
	if err != nil {
		c.isOnline = false
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	c.isOnline = true
	log.Println("Heartbeat sent successfully")
	return nil
}

// maintenanceLoop handles periodic maintenance tasks
func (c *CollectorService) maintenanceLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Maintenance every hour
	defer ticker.Stop()

	log.Println("Starting maintenance loop")

	for {
		select {
		case <-c.stopChan:
			log.Println("Maintenance loop stopped")
			return
		case <-ticker.C:
			c.performMaintenance()
		}
	}
}

// performMaintenance performs routine maintenance tasks
func (c *CollectorService) performMaintenance() {
	// Reset error count if everything is working fine
	if c.isOnline && time.Since(c.lastDataTime) < c.config.Serial.SampleInterval*time.Second*2 {
		c.errorCount = 0
	}

	// Cleanup old data from cache
	if err := c.cacheDB.CleanupOldData(7 * 24 * time.Hour); err != nil { // Cleanup data older than 7 days
		c.handleError("cache cleanup", err)
	}

	log.Println("Maintenance completed")
}

// handleError handles errors and implements error recovery
func (c *CollectorService) handleError(operation string, err error) {
	c.errorCount++
	log.Printf("Error in %s: %v (error count: %d)", operation, err, c.errorCount)

	// Implement exponential backoff for critical errors
	if c.errorCount > 10 {
		log.Printf("Too many errors, sleeping for 1 minute")
		time.Sleep(1 * time.Minute)
		c.errorCount = 5 // Reset to moderate level
	}
}

// GetStatus returns the current status of the collector service
func (c *CollectorService) GetStatus() ServiceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheStats, err := c.cacheDB.GetCacheStats()
	if err != nil {
		log.Printf("Warning: could not get cache stats: %v", err)
	}

	status := ServiceStatus{
		IsRunning:    c.isRunning,
		IsRegistered: c.isRegistered,
		IsOnline:     c.isOnline,
		LastDataTime: c.lastDataTime,
		ErrorCount:   c.errorCount,
		CacheStats:   cacheStats,
	}

	return status
}

// IsHealthy returns true if the collector service is healthy
func (c *CollectorService) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isRunning || !c.isRegistered {
		return false
	}

	// Check if data collection is working
	if time.Since(c.lastDataTime) > c.config.Serial.SampleInterval*time.Second*3 {
		return false
	}

	// Check error count
	if c.errorCount > 20 {
		return false
	}

	return true
}
