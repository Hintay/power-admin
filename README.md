# Power Monitor

Power Monitor is a comprehensive power monitoring system designed to collect, store, and visualize power consumption data. It consists of three main components: a `collector`, a `server`, and an `Android app`.

## Components

- **Collector**: A service that reads data from power monitoring devices (PZEM-004T) and sends it to the server.
- **Server**: A backend service that receives data from collectors, stores it in a time-series database (InfluxDB), and provides a web interface and API for data visualization and management.
- **Android App**: A mobile application for real-time power monitoring, historical data analysis, and consumption forecasting with an intuitive user interface.

## Documentation

- [Collector Documentation](./collector/README.md)
- [Server Documentation](./server/README.md)
- [Android App Documentation](./android/README.md)

## Getting Started

Please refer to the documentation for each component for installation and configuration instructions. 