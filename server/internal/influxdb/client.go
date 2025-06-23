package influxdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// Client represents the InfluxDB v3 client wrapper
type Client struct {
	client   *influxdb3.Client
	database string
}

// PowerDataPoint represents a power measurement data point for InfluxDB
type PowerDataPoint struct {
	CollectorID string    `json:"collector_id"`
	Timestamp   time.Time `json:"timestamp"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Energy      float64   `json:"energy"`
	Frequency   float64   `json:"frequency"`
	PowerFactor float64   `json:"power_factor"`
}

var globalClient *Client

// Init initializes the InfluxDB v3 client with database creation
func Init(host string, port int, token, database string, timeout time.Duration, useSSL bool) error {
	// Construct URL
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, host, port)

	// Create client configuration
	config := influxdb3.ClientConfig{
		Host:     url,
		Token:    token,
		Database: database,
	}

	// Create client
	client, err := influxdb3.New(config)
	if err != nil {
		return fmt.Errorf("failed to create InfluxDB v3 client: %w", err)
	}

	// Create the wrapper client
	globalClient = &Client{
		client:   client,
		database: database,
	}

	// Check if database exists, create if it doesn't
	if err := globalClient.ensureDatabaseExists(); err != nil {
		return fmt.Errorf("failed to ensure database exists: %w", err)
	}

	return nil
}

// ensureDatabaseExists checks if the database exists, creates it if it doesn't
func (c *Client) ensureDatabaseExists() error {
	// Check if database exists by querying SHOW TABLES
	query := "SHOW TABLES"

	iterator, err := c.client.Query(context.Background(), query)
	if err != nil {
		// If query fails, it might be because database doesn't exist
		// Try to create the database
		if strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "not found") {
			return c.createDatabase()
		}
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	// If we can query tables, database exists
	// Read at least one result to ensure query succeeded
	for iterator.Next() {
		break // Database exists and is accessible
	}

	if err := iterator.Err(); err != nil {
		// If we get an error reading results, database might not exist
		if strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "not found") {
			return c.createDatabase()
		}
		return fmt.Errorf("failed to verify database: %w", err)
	}

	return nil
}

// createDatabase creates the database using SQL DDL
func (c *Client) createDatabase() error {
	// In InfluxDB v3, databases are created implicitly when you write data
	// However, we can also use the HTTP API to create them explicitly
	// For now, we'll create a simple table to ensure the database exists

	// Try to create a system table that will implicitly create the database
	query := `CREATE TABLE IF NOT EXISTS __system_init (
		time TIMESTAMP,
		initialized BOOLEAN
	)`

	_, err := c.client.Query(context.Background(), query)
	if err != nil {
		// If CREATE TABLE fails, it might not be supported
		// In that case, the database will be created when we first write data
		fmt.Printf("Note: Database '%s' will be created automatically on first write\n", c.database)
		return nil
	}

	fmt.Printf("Database '%s' created successfully\n", c.database)
	return nil
}

// GetClient returns the global InfluxDB client
func GetClient() *Client {
	return globalClient
}

// WritePowerData writes a single power data point to InfluxDB
func (c *Client) WritePowerData(data PowerDataPoint) error {
	if c.client == nil {
		return fmt.Errorf("InfluxDB client not initialized")
	}

	// Create point for line protocol
	point := influxdb3.NewPoint("power_data",
		map[string]string{
			"collector_id": data.CollectorID,
		},
		map[string]interface{}{
			"voltage":      data.Voltage,
			"current":      data.Current,
			"power":        data.Power,
			"energy":       data.Energy,
			"frequency":    data.Frequency,
			"power_factor": data.PowerFactor,
		},
		data.Timestamp)

	// Write the point
	err := c.client.WritePoints(context.Background(), []*influxdb3.Point{point})
	if err != nil {
		return fmt.Errorf("failed to write power data: %w", err)
	}

	return nil
}

// WritePowerDataBatch writes multiple power data points to InfluxDB in a batch
func (c *Client) WritePowerDataBatch(data []PowerDataPoint) error {
	if c.client == nil {
		return fmt.Errorf("InfluxDB client not initialized")
	}

	if len(data) == 0 {
		return nil
	}

	// Create points
	points := make([]*influxdb3.Point, len(data))
	for i, point := range data {
		points[i] = influxdb3.NewPoint("power_data",
			map[string]string{
				"collector_id": point.CollectorID,
			},
			map[string]interface{}{
				"voltage":      point.Voltage,
				"current":      point.Current,
				"power":        point.Power,
				"energy":       point.Energy,
				"frequency":    point.Frequency,
				"power_factor": point.PowerFactor,
			},
			point.Timestamp)
	}

	// Write batch
	err := c.client.WritePoints(context.Background(), points)
	if err != nil {
		return fmt.Errorf("failed to write power data batch: %w", err)
	}

	return nil
}

// QueryPowerData queries power data within a time range
func (c *Client) QueryPowerData(collectorID string, start, end time.Time) ([]PowerDataPoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("InfluxDB client not initialized")
	}

	// Build SQL query
	query := fmt.Sprintf(`
		SELECT time, collector_id, voltage, current, power, energy, frequency, power_factor 
		FROM power_data 
		WHERE time >= '%s' AND time <= '%s'`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339))

	if collectorID != "" {
		query += fmt.Sprintf(" AND collector_id = '%s'", collectorID)
	}

	query += " ORDER BY time ASC"

	// Execute query
	iterator, err := c.client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}

	var results []PowerDataPoint
	for iterator.Next() {
		value := iterator.Value()

		// Extract values from the map
		timestamp, _ := value["time"].(time.Time)
		collectorID, _ := value["collector_id"].(string)
		voltage, _ := value["voltage"].(float64)
		current, _ := value["current"].(float64)
		power, _ := value["power"].(float64)
		energy, _ := value["energy"].(float64)
		frequency, _ := value["frequency"].(float64)
		powerFactor, _ := value["power_factor"].(float64)

		point := PowerDataPoint{
			Timestamp:   timestamp,
			CollectorID: collectorID,
			Voltage:     voltage,
			Current:     current,
			Power:       power,
			Energy:      energy,
			Frequency:   frequency,
			PowerFactor: powerFactor,
		}
		results = append(results, point)
	}

	if err := iterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to read query results: %w", err)
	}

	return results, nil
}

// QueryLatestPowerData queries the latest power data for a specific collector
func (c *Client) QueryLatestPowerData(collectorID string) (*PowerDataPoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("InfluxDB client not initialized")
	}

	// Build SQL query for latest data
	query := fmt.Sprintf(`
		SELECT time, collector_id, voltage, current, power, energy, frequency, power_factor 
		FROM power_data 
		WHERE collector_id = '%s' 
		ORDER BY time DESC 
		LIMIT 1`, collectorID)

	// Execute query
	iterator, err := c.client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest data: %w", err)
	}

	for iterator.Next() {
		value := iterator.Value()

		// Extract values from the map
		timestamp, _ := value["time"].(time.Time)
		collectorID, _ := value["collector_id"].(string)
		voltage, _ := value["voltage"].(float64)
		current, _ := value["current"].(float64)
		power, _ := value["power"].(float64)
		energy, _ := value["energy"].(float64)
		frequency, _ := value["frequency"].(float64)
		powerFactor, _ := value["power_factor"].(float64)

		point := PowerDataPoint{
			Timestamp:   timestamp,
			CollectorID: collectorID,
			Voltage:     voltage,
			Current:     current,
			Power:       power,
			Energy:      energy,
			Frequency:   frequency,
			PowerFactor: powerFactor,
		}
		return &point, nil
	}

	if err := iterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to read latest data: %w", err)
	}

	return nil, fmt.Errorf("no data found for collector %s", collectorID)
}

// Close closes the InfluxDB client connection
func (c *Client) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

// Health checks the health of the InfluxDB v3 connection
func (c *Client) Health() (*HealthStatus, error) {
	if c == nil {
		return nil, fmt.Errorf("InfluxDB client not initialized")
	}

	// Test connection with a simple query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.Query(ctx, "SELECT 1")
	if err != nil {
		return &HealthStatus{Status: "fail"}, err
	}

	return &HealthStatus{Status: "pass"}, nil
}

// HealthStatus represents the health status of InfluxDB
type HealthStatus struct {
	Status string `json:"status"`
}

// parseInterval converts interval strings like "5m", "1h", "1d" to seconds
func parseInterval(interval string) int {
	interval = strings.ToLower(interval)

	if strings.HasSuffix(interval, "s") {
		// seconds
		return 1
	} else if strings.HasSuffix(interval, "m") {
		// minutes
		return 60
	} else if strings.HasSuffix(interval, "h") {
		// hours
		return 3600
	} else if strings.HasSuffix(interval, "d") {
		// days
		return 86400
	}

	// Default to 5 minutes
	return 300
}
