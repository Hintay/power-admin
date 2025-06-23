package works.lm.powermonitor.network;

/**
 * Unified network configuration management class
 * Manages API server and WebSocket server address configuration
 * 
 * This class provides centralized management for all network endpoints,
 * allowing dynamic configuration of server addresses without code changes.
 * Both HTTP API and WebSocket connections use the same base server configuration.
 * 
 * Features:
 * - Centralized server address management
 * - Dynamic configuration support
 * - Input validation for host and port
 * - Default configuration fallback
 * - URL parsing utilities
 * 
 * Usage examples:
 * // Get current server address
 * String serverUrl = NetworkConfig.getApiBaseUrl();
 * 
 * // Change server address
 * NetworkConfig.setServerAddress("192.168.1.100", 8080);
 * 
 * // Reset to default address
 * NetworkConfig.resetToDefault();
 * 
 * // Validate configuration
 * if (NetworkConfig.isValidHost("192.168.1.100") && NetworkConfig.isValidPort(8080)) {
 *     NetworkConfig.setServerAddress("192.168.1.100", 8080);
 * }
 */
public class NetworkConfig {
    
    // Default server configuration
    private static final String DEFAULT_HOST = "192.168.50.92";
    private static final int DEFAULT_PORT = 8080;
    
    // Current server configuration (can be dynamically modified)
    private static String currentHost = DEFAULT_HOST;
    private static int currentPort = DEFAULT_PORT;
    
    /**
     * Get HTTP API base URL
     */
    public static String getApiBaseUrl() {
        return String.format("http://%s:%d/", currentHost, currentPort);
    }
    
    /**
     * Get WebSocket base URL
     */
    public static String getWebSocketBaseUrl() {
        return String.format("ws://%s:%d/", currentHost, currentPort);
    }
    
    /**
     * Get complete WebSocket connection URL
     */
    public static String getWebSocketUrl() {
        return getWebSocketBaseUrl() + "api/realtime/ws";
    }
    
    /**
     * Set server address
     * @param host Server host address
     * @param port Server port
     */
    public static void setServerAddress(String host, int port) {
        currentHost = host;
        currentPort = port;
    }
    
    /**
     * Set server address from complete URL
     * @param fullUrl Complete server URL (e.g., "http://192.168.1.100:8080/")
     */
    public static boolean setServerFromUrl(String fullUrl) {
        try {
            // Simple URL parsing
            fullUrl = fullUrl.trim();
            if (fullUrl.startsWith("http://")) {
                fullUrl = fullUrl.substring(7);
            }
            if (fullUrl.endsWith("/")) {
                fullUrl = fullUrl.substring(0, fullUrl.length() - 1);
            }
            
            String[] parts = fullUrl.split(":");
            if (parts.length == 2) {
                String host = parts[0];
                int port = Integer.parseInt(parts[1]);
                setServerAddress(host, port);
                return true;
            }
        } catch (Exception e) {
            // Parsing failed, keep original configuration
        }
        return false;
    }
    
    /**
     * Reset to default server address
     */
    public static void resetToDefault() {
        currentHost = DEFAULT_HOST;
        currentPort = DEFAULT_PORT;
    }
    
    /**
     * Get current server host address
     */
    public static String getCurrentHost() {
        return currentHost;
    }
    
    /**
     * Get current server port
     */
    public static int getCurrentPort() {
        return currentPort;
    }
    
    /**
     * Get server address display string
     */
    public static String getServerDisplayString() {
        return currentHost + ":" + currentPort;
    }
    
    /**
     * Check if server configuration is default configuration
     */
    public static boolean isDefaultServer() {
        return DEFAULT_HOST.equals(currentHost) && DEFAULT_PORT == currentPort;
    }
    
    /**
     * Validate if host address format is correct
     */
    public static boolean isValidHost(String host) {
        if (host == null || host.trim().isEmpty()) {
            return false;
        }
        // Simple IP address or domain name validation
        return host.matches("^[a-zA-Z0-9]([a-zA-Z0-9\\-\\.]*[a-zA-Z0-9])?$");
    }
    
    /**
     * Validate if port number is correct
     */
    public static boolean isValidPort(int port) {
        return port > 0 && port <= 65535;
    }
} 