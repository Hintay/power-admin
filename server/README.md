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
JwtSecret = your-secure-jwt-secret-key
RefreshTokenExpires = 7d
AccessTokenExpires = 24h

[server]
Host    = 0.0.0.0
Port    = 8080
RunMode = release

[influxdb]
URL = http://localhost:8086
Token = your-influxdb-token
Organization = power-monitor
Bucket = power-data
```

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
./power-monitor -c app.ini
```

### 6. First Run
On the first start, the system will automatically create an administrator account. Check the logs to get the default password:
```
Username: admin
Password: <randomly-generated-password>
```

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

A summary of the main API endpoints is provided below. For full details, please refer to the API documentation.

#### Client API (`/root/v1/client`)
- `POST /auth/login`: User login
- `POST /auth/refresh`: Refresh token
- `POST /auth/logout`: User logout
- `GET /auth/profile`: Get user profile
- `PUT /auth/profile`: Update user profile
- `POST /auth/change-password`: Change user password
- `GET /data/collectors`: Get list of user's collectors
- `GET /data/collectors/:id`: Get collector details
- `GET /data/collectors/:id/status`: Get collector status
- `GET /data/collectors/:id/latest`: Get latest data for a collector
- `GET /data/collectors/:id/history`: Get historical data for a collector
- `GET /data/collectors/:id/statistics`: Get statistics for a collector
- `GET /data/analytics`: Get overall power data analytics

#### Admin API (`/root/v1/admin`)
- `GET /collectors`: Get list of collectors
- `POST /collectors`: Create a new collector
- `GET /collectors/:id`: Get collector details
- `PUT /collectors/:id`: Update a collector
- `DELETE /collectors/:id`: Delete a collector
- `GET /collectors/:id/status`: Get collector status
- `POST /collectors/:id/config`: Update collector configuration
- `GET /registration-codes`: Get list of registration codes
- `POST /registration-codes`: Create a new registration code
- `DELETE /registration-codes/:id`: Delete a registration code
- ... and more for user management, system settings, etc.

#### Collector API (`/root/v1/collector`)
- `POST /data`: Upload a single data point
- `POST /data/batch`: Upload a batch of data points
- `GET /config`: Get collector configuration
- `POST /heartbeat`: Send a heartbeat signal

#### Real-time Communication
- `GET /realtime/ws`: WebSocket connection
- `GET /realtime/sse`: Server-Sent Events

## Command-line Tool (CLI)

The server includes a powerful command-line interface (CLI) for management tasks.

### Basic Usage
```bash
# Show help
./power-monitor --help

# Start the server (default action)
./power-monitor serve --config /path/to/app.ini
```

### User Management
```bash
# Create a user
./power-monitor user create -u <username> -e <email> -p <password> -r <role>

# List users
./power-monitor user list

# Reset user password
./power-monitor user reset-password -u <username>
```

### Collector Management
```bash
# Add a new collector
./power-monitor collector add -i <collector-id> -n <name> -u <owner-username>

# List collectors
./power-monitor collector list

# Configure a collector
./power-monitor collector config -i <collector-id> --sample-interval 30
```

### Registration Code Management
```bash
# Generate a registration code
./power-monitor regcode generate -d "New collector" -e 48 # expires in 48 hours

# List registration codes
./power-monitor regcode list
```
For more commands and options, use the `--help` flag on each command.

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