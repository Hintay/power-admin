# Power Monitor Server

The Power Monitor server is a complete IoT power data acquisition and visualization system. It supports multi-client concurrent data collection, real-time data push, historical data analysis, and visualization.

## 🚀 Features

### Core Functionality
- **Multi-client Data Collection**: Supports devices like Raspberry Pi connected to PZEM-004 power measurement modules via serial port.
- **Real-time Data Transmission**: Supports WebSocket for real-time data push.
- **Data Persistence**: Uses a dual storage strategy with SQLite and InfluxDB.
- **User and Permission Management**: Supports multiple users and role-based access control.
- **Device Management**: Supports collector registration, configuration, and status monitoring.
- **API Separation**: Collector, admin backend, and client APIs are completely separate.

### System Architecture
- **Collector**: Implemented in Golang, supports serial data acquisition and network transmission.
- **Server**: A REST API service implemented in Golang with the Gin framework.
- **Admin Frontend**: A management interface built with Vue and Ant Design Vue.
- **Client**: Android application and Web client.

## 📋 System Requirements

### Server Requirements
- Go 1.21+
- SQLite 3.x
- InfluxDB 2.x
- Memory: 512MB+
- Storage: 1GB+

### Collector Requirements
- Go 1.21+
- Serial port support
- Network connection
- PZEM-004 power measurement module

## 🛠️ Installation and Deployment

### 1. Clone the Project
```bash
git clone <repository-url>
cd power-monitor/server
```

### 2. Install Dependencies
```bash
# Tidy and download Go module dependencies
go mod tidy
go mod download
```

### 3. Configuration
Copy the configuration file template and modify the settings:
```bash
cp app.example.ini app.ini
```

Edit the `app.ini` configuration file:
```ini
[app]
PageSize  = 20
JwtSecret = your-jwt-secret-key-here
RefreshTokenExpires = 7d
AccessTokenExpires = 24h

[server]
Host    = 0.0.0.0
Port    = 8080
RunMode = release  # debug, release, test
EnableHTTPS = false

[database]
Name = power_monitor.db

[influxdb]
Host = localhost
Port = 8086
Token = your-influxdb-token
Database = power-data
Timeout = 30s
UseSSL = false

[auth]
IPWhiteList         =
BanThresholdMinutes = 10
MaxAttempts         = 10

[collector]
TokenExpires = 30d
RegistrationCodeExpires = 7d

[realtime]
EnableWebSocket = true
EnableSSE = true
WebSocketPath = /ws
SSEPath = /sse
MaxConnections = 1000

[crypto]
Secret = your-crypto-secret-key

[logs]
Level = info
MaxSize = 100
MaxBackups = 5
MaxAge = 28
Compress = true

[rate_limit]
RequestsPerMinute = 60
BurstSize = 10
```

**Configuration Options:**
- `[app]`: Application basic settings, including JWT secret and token expiration times
- `[server]`: Server settings, including listen address, port, and run mode
- `[database]`: SQLite database file path
- `[influxdb]`: InfluxDB time-series database connection settings
- `[auth]`: Authentication settings, including IP whitelist and login attempt limits
- `[collector]`: Collector settings, including token and registration code expiration times
- `[realtime]`: Real-time communication settings, WebSocket and SSE related configurations
- `[crypto]`: Encryption key settings
- `[logs]`: Log management settings
- `[rate_limit]`: API access rate limiting

### 4. Start InfluxDB
You can use Docker to start an InfluxDB instance:
```bash
# Start InfluxDB using Docker
docker run -d \
  --name influxdb \
  -p 8086:8086 \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=your-password \
  -e DOCKER_INFLUXDB_INIT_ORG=power-monitor \
  -e DOCKER_INFLUXDB_INIT_BUCKET=power-data \
  influxdb:2.7
```

### 5. Compile and Run
```bash
# Compile the application
go build -o power-monitor

# Run the server
./power-monitor --config app.ini

# Or use the default configuration file (app.ini)
./power-monitor
```

### 6. First Run
On first startup, you can use the CLI tool to create an administrator account:
```bash
# Create administrator account
./power-monitor user create -u admin -e admin@example.com -r admin

# Or create the first user via API after starting the server
```

**Note**: The system has no default accounts, you need to manually create the first administrator user.

## 📚 API Documentation

### Authentication
All APIs use JWT Bearer Token for authentication:
```
Authorization: Bearer <your-jwt-token>
```
Collectors use a dedicated Collector Token:
```
Authorization: Collector <your-collector-token>
```

### Main API Endpoints

The system API follows RESTful design principles, with all endpoints prefixed with `/api`. Below is a detailed description of the main API endpoints:

#### Authentication API (`/api/auth`)
- `POST /login`: User login
- `POST /refresh`: Refresh access token
- `POST /logout`: User logout (requires JWT authentication)
- `GET /profile`: Get user profile (requires JWT authentication)
- `PUT /profile`: Update user profile (requires JWT authentication)
- `POST /change-password`: Change password (requires JWT authentication)

#### Admin API (`/api/admin`) - Requires JWT Authentication
**User Management**
- `GET /users`: Get user list (supports pagination and search)
- `GET /users/:id`: Get user details
- `POST /users`: Create new user
- `PUT /users/:id`: Update user information
- `DELETE /users/:id`: Delete user

**Collector Management**
- `GET /collectors`: Get collector list (supports pagination and search)
- `GET /collectors/:id`: Get collector details
- `POST /collectors`: Create new collector
- `PUT /collectors/:id`: Update collector information
- `DELETE /collectors/:id`: Delete collector
- `GET /collectors/:id/status`: Get collector status
- `POST /collectors/:id/config`: Update collector configuration

**Registration Code Management**
- `GET /registration-codes`: Get registration code list
- `POST /registration-codes`: Create new registration code
- `DELETE /registration-codes/:id`: Delete registration code

**System Management**
- `GET /system/stats`: Get system statistics
- `GET /system/health`: Get system health status

**Data Analytics (Admin Level)**
- `GET /analytics/dashboard`: Get admin dashboard data
- `GET /analytics/power-data`: Get power data analytics (supports period and collector_id parameters)
- `GET /analytics/collectors/:id/data`: Get specified collector data (supports type and period parameters)

#### Client API (`/api/client`) - Requires JWT Authentication
**Data Access**
- `GET /data/collectors`: Get user's collector list
- `GET /data/collectors/:id`: Get collector details
- `GET /data/collectors/:id/status`: Get collector status
- `GET /data/collectors/:id/latest`: Get latest data
- `GET /data/collectors/:id/history`: Get historical data (supports start, end, limit parameters)
- `GET /data/collectors/:id/statistics`: Get statistics data
- `GET /data/collectors/:id/data`: Get collector data view (supports type and period parameters)
- `GET /data/analytics`: Get power data analytics

**User Analytics Features**
- `GET /analytics/dashboard`: Get user dashboard
- `GET /analytics/energy-consumption`: Get energy consumption analysis (supports period parameter)
- `GET /analytics/power-trends`: Get power trend analysis (supports period and type parameters)
- `GET /analytics/cost-analysis`: Get cost analysis (supports period, currency, rate parameters)
- `GET /analytics/prediction/:collectorId`: Get daily energy prediction

#### Collector API (`/api/collector`) - Requires Collector Token Authentication
**Data Upload**
- `POST /data`: Upload single data point
- `POST /data/batch`: Batch upload data points

**Configuration and Status**
- `GET /config`: Get collector configuration
- `POST /heartbeat`: Send heartbeat signal

**Collector Registration**
- `POST /register`: Collector registration (requires registration code)

#### Real-time Communication
- `GET /api/realtime/ws`: WebSocket connection (requires JWT authentication)

#### System Health Check
- `GET /api/health`: System health check (no authentication required)

## Command-line Tool (CLI)

The server includes a powerful command-line interface (CLI) for management tasks.

### Basic Usage
```bash
# Show help information
./power-monitor --help

# Start the server (default action)
./power-monitor serve --config /path/to/app.ini

# Or start directly (serve is the default command)
./power-monitor --config /path/to/app.ini
```

### User Management
```bash
# Create user
./power-monitor user create -u <username> -e <email> [-p <password>] [-f <fullname>] [-r <role>]
# Example:
./power-monitor user create -u admin -e admin@example.com -r admin

# List all users
./power-monitor user list

# Update user information
./power-monitor user update -u <username> [--email <new-email>] [--fullname <new-fullname>] [--role <new-role>] [--active/--inactive]

# Reset user password
./power-monitor user reset-password -u <username> [-p <new-password>]

# Delete user
./power-monitor user delete -u <username> [--force]
```

### Collector Management
```bash
# Add new collector
./power-monitor collector add -n <name> [-i <collector-id>] [-d <description>] [-l <location>] [-u <owner-username>]
# Example:
./power-monitor collector add -n "Office Collector" -l "Building 1 Floor 1" -u admin

# List all collectors
./power-monitor collector list

# Update collector information
./power-monitor collector update -i <collector-id> [--name <new-name>] [--description <new-description>] [--location <new-location>] [--active/--inactive]

# Configure collector parameters
./power-monitor collector config -i <collector-id> [options]
# Configuration options:
#   --sample-interval <seconds>     # Sampling interval
#   --upload-interval <seconds>     # Upload interval
#   --max-cache-size <count>        # Maximum cache size
#   --auto-upload                   # Enable auto upload
#   --no-auto-upload               # Disable auto upload
#   --compression-level <0-9>       # Compression level

# View collector status
./power-monitor collector status [-i <collector-id>]  # Show all collectors if ID not specified

# Delete collector
./power-monitor collector delete -i <collector-id> [--force]
```

### Registration Code Management
```bash
# Generate registration code
./power-monitor regcode generate -u <creator-username> [-d <description>] [-e <expire-hours>] [-c <custom-code>]
# Example:
./power-monitor regcode generate -u admin -d "New collector registration" -e 48

# List all registration codes
./power-monitor regcode list

# Revoke registration code
./power-monitor regcode revoke -c <registration-code>
```

### Command Parameters
**Global Parameters:**
- `--config` / `-c`: Configuration file path (default: app.ini)
- `--help`: Show help information
- `--version`: Show version information

**User Roles:**
- `admin`: Administrator with full permissions
- `user`: Regular user, can only manage their own collectors

**Notes:**
- Password must be at least 6 characters long
- If password is not provided in command line, system will prompt for input
- Use `--force` parameter to skip confirmation prompts
- Collector ID will be auto-generated as UUID if not specified

For more commands and options, use the `--help` parameter for detailed help.

## 🔧 Development Guide

### Project Structure
```
server/
├── main.go                 # Main application entry point
├── app.example.ini         # Configuration file template
├── go.mod                  # Go module file
├── api/                    # API controllers
│   ├── admin/              # Admin API
│   ├── client/             # Client API
│   └── collector/          # Collector API
├── internal/               # Internal modules
│   ├── auth/               # Authentication service
│   ├── influxdb/           # InfluxDB client
│   ├── realtime/           # Real-time communication
│   ├── middleware/         # Middlewares
│   ├── cmd/                # Command-line handler
│   ├── kernel/             # System kernel
│   └── migrate/            # Database migrations
├── model/                  # Data models
├── router/                 # Router configuration
└── settings/               # Configuration management
```

### Adding a New API Endpoint
1. Add the handler function in the appropriate API package.
2. Register the route in the `RegisterRoutes` function.
3. Add necessary middleware and permission checks.

### Adding a New Data Model
1. Create the model file in the `model/` directory.
2. Register it in the `GenerateAllModel` function in `model/model.go`.
3. Add a migration script in `internal/migrate/migrate.go`. 