package pzem

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

// PowerData represents the power measurement data from PZEM-004T
type PowerData struct {
	Timestamp   time.Time `json:"timestamp"`
	Voltage     float64   `json:"voltage"`      // Volts
	Current     float64   `json:"current"`      // Amperes
	Power       float64   `json:"power"`        // Watts
	Energy      float64   `json:"energy"`       // Wh
	Frequency   float64   `json:"frequency"`    // Hz
	PowerFactor float64   `json:"power_factor"` // Power Factor
	Alarm       bool      `json:"alarm"`        // Alarm status
}

// PZEM004T represents the PZEM-004T device
type PZEM004T struct {
	port    *serial.Port
	address uint8
}

// NewPZEM004T creates a new PZEM-004T instance
func NewPZEM004T(portName string, baudRate int, address uint8, timeout time.Duration) (*PZEM004T, error) {
	config := &serial.Config{
		Name:        portName,
		Baud:        baudRate,
		ReadTimeout: timeout,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %w", err)
	}

	return &PZEM004T{
		port:    port,
		address: address,
	}, nil
}

// Close closes the serial port connection
func (p *PZEM004T) Close() error {
	if p.port != nil {
		return p.port.Close()
	}
	return nil
}

// ReadData reads power data from PZEM-004T module
func (p *PZEM004T) ReadData() (*PowerData, error) {
	// Build read command
	cmd := p.buildReadCommand()

	// Send command
	_, err := p.port.Write(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Read response (expect 25 bytes)
	response := make([]byte, 25)
	n, err := p.port.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if n < 25 {
		return nil, fmt.Errorf("insufficient data received: got %d bytes, expected 25", n)
	}

	// Parse response
	data, err := p.parseResponse(response[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return data, nil
}

// buildReadCommand constructs the command frame to read PZEM-004T data
// Command format: [address, 0x04, 0x00, 0x00, 0x00, 0x0A, CRC_Low, CRC_High]
func (p *PZEM004T) buildReadCommand() []byte {
	frame := []byte{p.address, 0x04, 0x00, 0x00, 0x00, 0x0A}
	crc := p.calculateCRC16(frame)

	// Append CRC (little-endian)
	frame = append(frame, byte(crc&0xFF))      // CRC low byte
	frame = append(frame, byte((crc>>8)&0xFF)) // CRC high byte

	return frame
}

// calculateCRC16 calculates CRC-16 (Modbus) checksum
func (p *PZEM004T) calculateCRC16(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}

// verifyCRC verifies the CRC of received data frame
func (p *PZEM004T) verifyCRC(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// Extract received CRC (last 2 bytes, little-endian)
	receivedCRC := uint16(data[len(data)-2]) + (uint16(data[len(data)-1]) << 8)

	// Calculate CRC for data without CRC bytes
	calculatedCRC := p.calculateCRC16(data[:len(data)-2])

	return receivedCRC == calculatedCRC
}

// parseResponse parses the PZEM-004T response data
func (p *PZEM004T) parseResponse(data []byte) (*PowerData, error) {
	if len(data) < 25 {
		return nil, fmt.Errorf("insufficient data length: %d", len(data))
	}

	// Verify CRC
	if !p.verifyCRC(data) {
		return nil, fmt.Errorf("CRC verification failed")
	}

	// Parse data according to PZEM-004T protocol
	// Data frame structure:
	// [Address][Function Code][Byte Count][Data...][CRC_Low][CRC_High]
	// Byte count = 20, Data = 20 bytes (voltage, current, power, energy, frequency, power factor, alarm)

	// Extract data fields (big-endian format)
	voltage := uint16(data[3])<<8 + uint16(data[4])
	current := uint32(data[7])<<24 + uint32(data[8])<<16 +
		uint32(data[5])<<8 + uint32(data[6])
	power := uint32(data[11])<<24 + uint32(data[12])<<16 +
		uint32(data[9])<<8 + uint32(data[10])
	energy := uint32(data[15])<<24 + uint32(data[16])<<16 +
		uint32(data[13])<<8 + uint32(data[14])
	frequency := uint16(data[17])<<8 + uint16(data[18])
	powerFactor := uint16(data[19])<<8 + uint16(data[20])
	alarm := data[21]

	return &PowerData{
		Timestamp:   time.Now(),
		Voltage:     float64(voltage) / 10.0,      // 0.1V resolution
		Current:     float64(current) / 1000.0,    // 0.001A resolution
		Power:       float64(power) / 10.0,        // 0.1W resolution
		Energy:      float64(energy),              // 1Wh resolution
		Frequency:   float64(frequency) / 10.0,    // 0.1Hz resolution
		PowerFactor: float64(powerFactor) / 100.0, // 0.01 resolution
		Alarm:       alarm != 0,
	}, nil
}

// TestConnection tests the connection to PZEM-004T module
func (p *PZEM004T) TestConnection() error {
	_, err := p.ReadData()
	return err
}

// SetAddress sets the PZEM-004T device address
func (p *PZEM004T) SetAddress(newAddress uint8) {
	p.address = newAddress
}

// GetAddress returns the current device address
func (p *PZEM004T) GetAddress() uint8 {
	return p.address
}

// ReadDataWithRetry reads data with retry mechanism
func (p *PZEM004T) ReadDataWithRetry(maxRetries int) (*PowerData, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		data, err := p.ReadData()
		if err == nil {
			return data, nil
		}

		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(100 * time.Millisecond) // Wait before retry
		}
	}

	return nil, fmt.Errorf("failed after %d retries, last error: %w", maxRetries, lastErr)
}

// IsDataValid validates if the power data is within expected ranges
func (data *PowerData) IsDataValid() bool {
	// Basic validation ranges for typical household/industrial use
	if data.Voltage < 0 || data.Voltage > 300 { // 0-300V
		return false
	}

	if data.Current < 0 || data.Current > 100 { // 0-100A
		return false
	}

	if data.Power < 0 || data.Power > 30000 { // 0-30kW
		return false
	}

	if data.Energy < 0 { // Energy should not be negative
		return false
	}

	if data.Frequency < 45 || data.Frequency > 65 { // 45-65Hz (typical power frequency range)
		return false
	}

	if data.PowerFactor < 0 || data.PowerFactor > 1 { // 0-1 power factor range
		return false
	}

	return true
}

// String returns a string representation of the power data
func (data *PowerData) String() string {
	return fmt.Sprintf(
		"Voltage: %.1fV, Current: %.3fA, Power: %.1fW, Energy: %.0fWh, "+
			"Frequency: %.1fHz, PowerFactor: %.2f, Alarm: %t",
		data.Voltage, data.Current, data.Power, data.Energy,
		data.Frequency, data.PowerFactor, data.Alarm,
	)
}
