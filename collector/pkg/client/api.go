package client

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// APIClient represents the HTTP client for server communication
type APIClient struct {
	client      *resty.Client
	baseURL     string
	prefix      string
	token       string
	collectorID string
}

// PowerDataRequest represents a single power data measurement for API
type PowerDataRequest struct {
	Timestamp   time.Time `json:"timestamp"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Energy      float64   `json:"energy"`
	Frequency   float64   `json:"frequency"`
	PowerFactor float64   `json:"power_factor"`
}

// PowerDataUploadRequest represents bulk power data upload
type PowerDataUploadRequest struct {
	CollectorID string             `json:"collector_id"`
	Data        []PowerDataRequest `json:"data"`
}

// RegisterRequest represents collector registration request
type RegisterRequest struct {
	RegistrationCode string `json:"registration_code"`
	CollectorID      string `json:"collector_id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Location         string `json:"location"`
	Version          string `json:"version"`
}

// RegisterResponse represents collector registration response
type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Token        string `json:"token"`
		TokenExpires string `json:"token_expires"`
		Config       struct {
			CollectorID      string `json:"collector_id"`
			SampleInterval   int    `json:"sample_interval"`
			UploadInterval   int    `json:"upload_interval"`
			MaxCacheSize     int    `json:"max_cache_size"`
			AutoUpload       bool   `json:"auto_upload"`
			CompressionLevel int    `json:"compression_level"`
		} `json:"config"`
	} `json:"data"`
}

// HeartbeatRequest represents heartbeat request
type HeartbeatRequest struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// APIResponse represents standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Count   int         `json:"count,omitempty"`
}

// NewAPIClient creates a new API client instance
func NewAPIClient(baseURL, apiPrefix string, timeout time.Duration) *APIClient {
	client := resty.New()
	client.SetTimeout(timeout)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", "PowerCollector/1.0.0")

	// Add retry mechanism
	client.SetRetryCount(3)
	client.SetRetryWaitTime(5 * time.Second)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return r.StatusCode() >= 500 || err != nil
	})

	return &APIClient{
		client:  client,
		baseURL: baseURL,
		prefix:  apiPrefix,
	}
}

// SetToken sets the token for API requests
func (a *APIClient) SetToken(token, collectorID string) {
	a.token = token
	a.collectorID = collectorID
	a.client.SetHeader("Authorization", "Bearer "+token)
}

// Register registers the collector with the server using registration code
func (a *APIClient) Register(req RegisterRequest) (*RegisterResponse, error) {
	var response RegisterResponse

	resp, err := a.client.R().
		SetBody(req).
		SetResult(&response).
		Post(a.buildURL("/auth/collector/register"))

	if err != nil {
		return nil, fmt.Errorf("registration request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	if !response.Success {
		return nil, fmt.Errorf("registration failed: %s", response.Message)
	}

	return &response, nil
}

// UploadData uploads single power data measurement
func (a *APIClient) UploadData(data PowerDataRequest) error {
	var response APIResponse

	resp, err := a.client.R().
		SetBody(data).
		SetResult(&response).
		Post(a.buildURL("/collector/data"))

	if err != nil {
		return fmt.Errorf("data upload request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("data upload failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	if !response.Success {
		return fmt.Errorf("data upload failed: %s", response.Message)
	}

	return nil
}

// UploadBatchData uploads multiple power data measurements
func (a *APIClient) UploadBatchData(data []PowerDataRequest) error {
	if len(data) == 0 {
		return nil
	}

	request := PowerDataUploadRequest{
		CollectorID: a.collectorID,
		Data:        data,
	}

	var response APIResponse

	resp, err := a.client.R().
		SetBody(request).
		SetResult(&response).
		Post(a.buildURL("/collector/data/batch"))

	if err != nil {
		return fmt.Errorf("batch upload request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("batch upload failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	if !response.Success {
		return fmt.Errorf("batch upload failed: %s", response.Message)
	}

	return nil
}

// SendHeartbeat sends heartbeat to maintain connection
func (a *APIClient) SendHeartbeat(status, version string) error {
	request := HeartbeatRequest{
		Status:  status,
		Version: version,
	}

	var response APIResponse

	resp, err := a.client.R().
		SetBody(request).
		SetResult(&response).
		Post(a.buildURL("/collector/heartbeat"))

	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	if !response.Success {
		return fmt.Errorf("heartbeat failed: %s", response.Message)
	}

	return nil
}

// GetConfig retrieves collector configuration from server
func (a *APIClient) GetConfig() (map[string]interface{}, error) {
	var response APIResponse

	resp, err := a.client.R().
		SetResult(&response).
		Get(a.buildURL("/collector/config"))

	if err != nil {
		return nil, fmt.Errorf("get config request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("get config failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	if !response.Success {
		return nil, fmt.Errorf("get config failed: %s", response.Message)
	}

	if config, ok := response.Data.(map[string]interface{}); ok {
		return config, nil
	}

	return nil, fmt.Errorf("get config failed: invalid data format")
}

// TestConnection tests the connection to the server
func (a *APIClient) TestConnection() error {
	resp, err := a.client.R().
		Get(a.buildURL("/collector/config"))

	if err != nil {
		return fmt.Errorf("test connection request failed: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("connection test failed with status %d", resp.StatusCode())
	}

	return nil
}

// buildURL constructs the full API URL
func (a *APIClient) buildURL(endpoint string) string {
	return a.baseURL + a.prefix + endpoint
}

// IsNetworkAvailable checks if the server is reachable
func (a *APIClient) IsNetworkAvailable() bool {
	err := a.TestConnection()
	return err == nil
}

// ConvertPZEMDataToAPI converts PZEM data to API request format
func ConvertPZEMDataToAPI(pzemData interface{}) (*PowerDataRequest, error) {
	// Handle conversion from PZEM data structure to API format
	dataMap, ok := pzemData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format")
	}

	return &PowerDataRequest{
		Timestamp:   parseTimestamp(dataMap["timestamp"]),
		Voltage:     parseFloat64(dataMap["voltage"]),
		Current:     parseFloat64(dataMap["current"]),
		Power:       parseFloat64(dataMap["power"]),
		Energy:      parseFloat64(dataMap["energy"]),
		Frequency:   parseFloat64(dataMap["frequency"]),
		PowerFactor: parseFloat64(dataMap["power_factor"]),
	}, nil
}

// Helper functions
func parseTimestamp(v interface{}) time.Time {
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	case time.Time:
		return val
	}
	return time.Now()
}

func parseFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0.0
}
