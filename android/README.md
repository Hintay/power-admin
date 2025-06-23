# Power Monitor Android App

An Android application for real-time power monitoring and data visualization, designed to work with the Power Monitor server system.

## Features

### 🔌 Real-time Monitoring
- **Live Data Display**: Real-time voltage, current, power, and energy consumption monitoring
- **Interactive Charts**: Touch-enabled charts with detailed data markers
- **Multiple Data Views**: Switch between power, voltage, current, and energy visualizations
- **Collector Management**: Support for multiple power monitoring devices

### 📊 Data Visualization
- **Real-time Charts**: Live updating line charts with up to 50 data points
- **Historical Data**: Browse and analyze past power consumption data
- **Custom Time Ranges**: Select specific date ranges for historical analysis
- **Interactive Markers**: Tap chart points to view detailed measurements

### 🔮 Power Prediction
- **Consumption Forecasting**: Daily and monthly power consumption predictions
- **Multiple Algorithms**: Support for Hybrid, Linear, Seasonal, and Moving Average algorithms
- **Confidence Levels**: Prediction accuracy indicators
- **Usage Recommendations**: Smart suggestions based on consumption patterns

### 🌐 Connectivity
- **WebSocket Integration**: Real-time data streaming from power monitoring devices
- **REST API**: Secure communication with the Power Monitor server
- **Token Authentication**: JWT-based authentication with automatic token refresh
- **Offline Support**: Display last known data when connection is unavailable

### 🎨 User Experience
- **Material Design**: Modern Android UI following Material Design guidelines
- **Multi-language Support**: English, Chinese, and Japanese language options
- **Dark Theme Support**: Automatic dark/light theme switching
- **Responsive Layout**: Optimized for various screen sizes

## Technical Specifications

### Requirements
- **Minimum Android Version**: API 23 (Android 6.0)
- **Target Android Version**: API 35 (Android 15)
- **Java Version**: Java 11
- **Network**: Internet connection required for real-time data

### Dependencies
- **UI Framework**: AndroidX Material Components
- **Networking**: Retrofit2 + OkHttp3 for REST API
- **WebSocket**: Java-WebSocket for real-time communication
- **Charts**: MPAndroidChart for data visualization
- **Security**: AndroidX Security Crypto for secure storage

## Project Structure

```
android/
├── app/
│   ├── src/main/
│   │   ├── java/works/lm/powermonitor/
│   │   │   ├── MainActivity.java          # Main real-time monitoring screen
│   │   │   ├── LoginActivity.java         # User authentication
│   │   │   ├── HistoryActivity.java       # Historical data analysis
│   │   │   ├── PredictionActivity.java    # Power consumption forecasting
│   │   │   ├── AnalyticsActivity.java     # Data analytics (placeholder)
│   │   │   ├── SettingsActivity.java      # App settings (placeholder)
│   │   │   ├── model/                     # Data models
│   │   │   │   ├── PowerData.java         # Power measurement data
│   │   │   │   ├── Collector.java         # Power monitoring device info
│   │   │   │   └── User.java              # User authentication data
│   │   │   ├── network/                   # Network communication
│   │   │   │   ├── ApiClient.java         # REST API client
│   │   │   │   ├── ApiService.java        # API endpoints definition
│   │   │   │   ├── WebSocketClient.java   # Real-time data streaming
│   │   │   │   ├── NetworkConfig.java     # Network configuration
│   │   │   │   └── TokenRefreshHandler.java # JWT token management
│   │   │   └── utils/
│   │   │       └── LanguageUtils.java     # Multi-language support
│   │   ├── res/                           # Resources
│   │   │   ├── layout/                    # UI layouts
│   │   │   ├── values/                    # Strings, colors, themes
│   │   │   ├── values-zh/                 # Chinese translations
│   │   │   ├── values-ja/                 # Japanese translations
│   │   │   └── drawable/                  # Icons and graphics
│   │   └── AndroidManifest.xml
│   └── build.gradle.kts                   # App build configuration
├── gradle/                                # Gradle wrapper
├── build.gradle.kts                       # Project build configuration
└── README.md                              # This file
```

## Installation

### Prerequisites
1. **Android Studio**: Latest stable version
2. **Android SDK**: API level 23-35
3. **Power Monitor Server**: Running server instance (see [server documentation](../server/README.md))

### Setup Steps

1. **Clone the Repository**
   ```bash
   git clone <repository-url>
   cd power-admin/android
   ```

2. **Configure Network Settings**
   - Open `app/src/main/java/works/lm/powermonitor/network/NetworkConfig.java`
   - Update the server address to match your Power Monitor server:
   ```java
   // Modify the default server configuration
   private static final String DEFAULT_HOST = "your-server-ip";
   private static final int DEFAULT_PORT = 8080;
   ```
   
   Or use the dynamic configuration methods:
   ```java
   // Set server address programmatically
   NetworkConfig.setServerAddress("your-server-ip", 8080);
   ```

3. **Build and Install**
   ```bash
   ./gradlew assembleDebug
   adb install app/build/outputs/apk/debug/app-debug.apk
   ```

   Or use Android Studio:
   - Open the project in Android Studio
   - Click "Run" or press Shift+F10

## Configuration

### Server Connection
The app requires a running Power Monitor server. Configure the connection in `NetworkConfig.java`:

```java
// Default server configuration (modify these values)
private static final String DEFAULT_HOST = "192.168.50.92";  // Change to your server IP
private static final int DEFAULT_PORT = 8080;                // Change to your server port

// The app will automatically generate URLs:
// API Base URL: http://192.168.50.92:8080/
// WebSocket URL: ws://192.168.50.92:8080/api/realtime/ws
```

You can also change the server address dynamically:
```java
// Change server address at runtime
NetworkConfig.setServerAddress("192.168.1.100", 8080);

// Or set from complete URL
NetworkConfig.setServerFromUrl("http://192.168.1.100:8080/");
```

### Authentication
The app uses JWT token-based authentication. Users must log in with valid credentials from the Power Monitor server.

## Usage

### Getting Started
1. **Login**: Enter your username and password on the login screen
2. **Select Collector**: Choose a power monitoring device from the dropdown
3. **Monitor Data**: View real-time power consumption data
4. **Switch Views**: Tap on data cards (Voltage, Current, Power, Energy) to change chart view
5. **Access History**: Use the navigation menu to view historical data
6. **Predictions**: Check power consumption forecasts and recommendations

### Chart Interaction
- **Real-time Data**: Charts update automatically when new data arrives
- **Touch Markers**: Tap on chart points to see detailed measurements
- **Data Switching**: Tap voltage, current, power, or energy cards to switch chart display
- **Zoom and Pan**: Pinch to zoom and pan to explore chart data

### Navigation
Use the navigation drawer (hamburger menu) to access:
- **Real-time Monitoring**: Live power data dashboard
- **Historical Data**: Browse past consumption data
- **Data Analytics**: Advanced analytics (coming soon)
- **Power Forecast**: Consumption predictions and recommendations
- **App Settings**: Language and preferences (coming soon)
- **Logout**: Sign out of the application

## API Integration

The app integrates with the Power Monitor server through:

### REST API Endpoints
- `POST /auth/login` - User authentication
- `GET /client/collectors` - List available collectors
- `GET /client/data/{collectorId}/latest` - Get latest power data
- `GET /client/data/{collectorId}/history` - Get historical data
- `GET /client/analytics/{collectorId}/prediction` - Get power predictions

### WebSocket Connection
- Real-time data streaming from `/api/realtime/ws` endpoint
- Automatic reconnection on connection loss
- Collector status monitoring

## Development

### Building from Source
```bash
# Debug build
./gradlew assembleDebug

# Release build
./gradlew assembleRelease

# Run tests
./gradlew test
```

### Code Style
The project follows Android development best practices:
- Material Design guidelines
- MVP architecture pattern
- Async/await for network operations
- Proper error handling and user feedback

## Troubleshooting

### Common Issues

**Connection Failed**
- Verify server address in `NetworkConfig.java` (default: 192.168.50.92:8080)
- Check if Power Monitor server is running
- Ensure device/emulator has network access
- Use `NetworkConfig.setServerAddress()` to change server dynamically

**Login Issues**
- Verify credentials with server administrator
- Check server authentication configuration

**No Real-time Data**
- Ensure collector is online and sending data
- Check WebSocket connection status
- Verify collector selection

**Chart Performance**
- App limits real-time chart to 50 data points for optimal performance
- Historical data charts may be slower with large datasets

### Logs
Enable debug logging to troubleshoot issues:
```bash
adb logcat | grep PowerMonitor
```

## License

This project is part of the Power Monitor system. Please refer to the main project license.

## Support

For issues and questions:
- Check the main [Power Monitor documentation](../README.md)
- Review server setup in [server documentation](../server/README.md)
- Check collector configuration in [collector documentation](../collector/README.md) 