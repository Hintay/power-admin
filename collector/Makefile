# Power Collector Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=power-collector
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_LINUX=$(BINARY_NAME)
BINARY_ARM=$(BINARY_NAME)-arm

# Version info
VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitHash=$(GIT_HASH)"

.PHONY: all build clean test coverage deps help

all: test build

# Build for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v .

# Build for Linux
build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_LINUX) -v .

# Build for Windows
build-windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_WINDOWS) -v .

# Build for ARM (Raspberry Pi)
build-arm:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(LDFLAGS) -o $(BINARY_ARM) -v .

# Build for all platforms
build-all: build-linux build-windows build-arm

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_LINUX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_ARM)
	rm -f *.log
	rm -f *.db

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run the application
run: build
	./$(BINARY_NAME) -config config.example.ini

# Install the binary
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

# Create systemd service file
systemd-service:
	@echo "Creating systemd service file..."
	@cat > power-collector.service << EOF
[Unit]
Description=Power Collector Service
After=network.target

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/opt/power-collector
ExecStart=/opt/power-collector/$(BINARY_NAME) -config /opt/power-collector/config.ini
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
	@echo "Service file created: power-collector.service"
	@echo "To install: sudo cp power-collector.service /etc/systemd/system/"
	@echo "To enable: sudo systemctl enable power-collector"
	@echo "To start: sudo systemctl start power-collector"

# Create deployment package
package: build-arm
	@echo "Creating deployment package..."
	mkdir -p dist/power-collector
	cp $(BINARY_ARM) dist/power-collector/$(BINARY_NAME)
	cp config.example.ini dist/power-collector/config.ini
	cp README.md dist/power-collector/ 2>/dev/null || echo "README.md not found, skipping"
	@echo "#!/bin/bash" > dist/power-collector/install.sh
	@echo "sudo cp power-collector /usr/local/bin/" >> dist/power-collector/install.sh
	@echo "sudo chmod +x /usr/local/bin/power-collector" >> dist/power-collector/install.sh
	@echo "echo 'Power Collector installed successfully'" >> dist/power-collector/install.sh
	chmod +x dist/power-collector/install.sh
	cd dist && tar -czf power-collector-$(VERSION)-arm.tar.gz power-collector/
	@echo "Package created: dist/power-collector-$(VERSION)-arm.tar.gz"

# Create Docker image
docker-build:
	docker build -t power-collector:$(VERSION) .
	docker tag power-collector:$(VERSION) power-collector:latest

# Development targets
dev-setup:
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/swaggo/swag/cmd/swag@latest

# Generate mock files for testing
mock:
	mockgen -source=pkg/pzem/pzem004t.go -destination=pkg/mocks/mock_pzem.go
	mockgen -source=pkg/client/api.go -destination=pkg/mocks/mock_client.go

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-linux   - Build for Linux AMD64"
	@echo "  build-windows - Build for Windows AMD64"
	@echo "  build-arm     - Build for ARM (Raspberry Pi)"
	@echo "  build-all     - Build for all platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  deps          - Download dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  run           - Build and run application"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  package       - Create deployment package for ARM"
	@echo "  docker-build  - Build Docker image"
	@echo "  systemd-service - Create systemd service file"
	@echo "  dev-setup     - Install development tools"
	@echo "  mock          - Generate mock files for testing"
	@echo "  help          - Show this help message" 