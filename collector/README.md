# Power Collector

The power data acquisition client program connects to the PZEM-004T power measurement module through a serial port to collect data such as voltage, current, and power, and upload it to the server.

## Features

- 🔌 **Serial Communication**: Communicates with the PZEM-004T module via serial port, supporting the Modbus protocol.
- 📊 **Data Collection**: Collects power data (voltage, current, power, energy, frequency, power factor) every 15 seconds.
- 🌐 **Network Upload**: Uploads data to the server in real-time, supporting HTTP REST API.
- 💾 **Local Cache**: Automatically caches data when the network is disconnected and sends it upon recovery.
- 🔐 **Secure Authentication**: Supports static token authentication and registration code registration.
- 🔧 **Configuration Management**: Uses an INI configuration file, supporting various parameter settings.
- 📋 **Logging**: Detailed logging with support for log rotation.
- 🔄 **Automatic Retry**: Automatically retries on network failure, with exponential backoff.
- 💓 **Heartbeat**: Periodically sends heartbeat signals to maintain the connection.
- 🧹 **Data Cleanup**: Automatically cleans up expired local cache data.

## System Requirements

- **Operating System**: Linux (Raspberry Pi recommended), Windows, macOS
- **Hardware**: PZEM-004T power measurement module
- **Connection**: USB to serial module or Raspberry Pi GPIO serial port
- **Network**: HTTP network connection (optional, supports offline mode)

## Quick Start

### 1. Download and Install

```bash
# Clone the source code
git clone <repository-url>
cd collector

# Install dependencies
go mod tidy

# Build the program
make build
```

### 2. Configuration

Copy the example configuration file and modify it:

```bash
cp config.example.ini config.ini
```

Edit the `config.ini` file to set your parameters:

```ini
[collector]
id = your-collector-id
name = Your Collector Name
description = Power monitoring for your location
location = Your Location

[serial]
port = /dev/ttyUSB0  # Linux: /dev/ttyUSB0, Windows: COM1
baud_rate = 9600
sample_interval = 15
timeout = 2

[server]
base_url = http://your-server:8080
api_prefix = /api
timeout = 30
retry_interval = 60
max_retries = 5

[auth]
token = your-auth-token
registration_code = your-registration-code
```

### 3. Run the Program

```bash
# Run directly
./power-collector -config config.ini

# Or use Make command
make run

# Check version information
./power-collector -version
```

## Configuration Details

### [collector]

- `id`: Unique collector ID (required)
- `name`: Friendly name (required)
- `description`: Description
- `location`: Location information

### [serial]

- `port`: Serial device path
- `baud_rate`: Baud rate (default 9600)
- `sample_interval`: Sampling interval in seconds (e.g., `15s`)
- `timeout`: Serial port timeout in seconds (e.g., `2s`)

### [server]

- `base_url`: Server base URL
- `api_prefix`: API prefix path
- `timeout`: Request timeout in seconds (e.g., `30s`)
- `retry_interval`: Retry interval in seconds (e.g., `60s`)
- `max_retries`: Maximum number of retries

### [auth]

- `token`: Static authentication token
- `registration_code`: Registration code (for initial registration)

### [data]

- `cache_db`: Local cache database path
- `max_cache_size`: Maximum number of cache records
- `batch_size`: Batch upload size
- `upload_interval`: Upload interval in seconds (e.g., `60s`)
- `auto_upload`: Whether to upload automatically when the network is available

### [logging]

- `level`: Log level (debug/info/warn/error)
- `file`: Log file path
- `max_size`: Maximum log file size (MB)
- `max_backups`: Number of log files to keep
- `max_age`: Log file retention period (days)

## Development and Build

### Build Commands

```bash
# Build for the current platform
make build

# Build for ARM (Raspberry Pi)
make build-arm

# Build for all platforms
make build-all

# Clean build files
make clean
```

### Testing

```bash
# Run tests
make test

# Run tests and generate a coverage report
make coverage

# Run linter
make lint

# Format code
make fmt
```

### Deployment

```bash
# Create a deployment package
make package

# Install to system
make install

# Create systemd service file
make systemd-service
```

## Hardware Connection

### PZEM-004T Connection Diagram

```
PZEM-004T    USB-to-Serial    Raspberry Pi/PC
---------    -----------    ---------------
5V      →    5V             →    5V/VCC
GND     →    GND            →    GND
TX      →    RX             →    GPIO15 (RXD)
RX      →    TX             →    GPIO14 (TXD)
```

### Raspberry Pi Serial Configuration

You need to enable the serial port on a Raspberry Pi:

```bash
# Edit the config file
sudo nano /boot/config.txt

# Add the following lines
enable_uart=1
dtoverlay=disable-bt

# Reboot to apply
sudo reboot
```

## API

The collector communicates with the server using the following APIs:

### Endpoints
```http
POST /api/auth/collector/register    # Register the collector
POST /api/collector/heartbeat         # Send heartbeat to the server
POST /api/collector/data              # Upload single data point
POST /api/collector/data/batch        # Upload data in batch
GET /api/collector/config           # Get remote configuration
```

## Troubleshooting

### Common Issues

1. **Serial Port Permission Issues**
   ```bash
   # Add user to the dialout group
   sudo usermod -a -G dialout $USER
   # Re-login to apply
   ```

2. **Serial Port is Busy**
   ```bash
   # Check serial port usage
   sudo lsof /dev/ttyUSB0
   ```

3. **Network Connection Issues**
   - Check firewall settings.
   - Verify server address and port.

4. **Authentication Failure**
   - Ensure the token or registration code is correct.

## Logging

```bash
# View real-time logs
tail -f collector.log

# View systemd service logs
sudo journalctl -u power-collector -f
```

## System Service

### Create systemd service

```bash
# Create service file
make systemd-service

# Install service
sudo cp power-collector.service /etc/systemd/system/
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable power-collector
sudo systemctl start power-collector

# View service status
sudo systemctl status power-collector
```
