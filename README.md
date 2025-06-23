# Power Monitor

Power Monitor is a power monitoring system designed to collect, store, and visualize power consumption data. It consists of two main components: a `collector` and a `server`.

## Components

- **Collector**: A service that reads data from power monitoring devices (e.g., PZEM-004T) and sends it to the server.
- **Server**: A backend service that receives data from collectors, stores it in a time-series database (InfluxDB), and provides a web interface and API for data visualization and management.

## Documentation

- [Collector Documentation](./collector/README.md)
- [Server Documentation](./server/README.md)

## Getting Started

Please refer to the documentation for each component for installation and configuration instructions. 