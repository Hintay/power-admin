package works.lm.powermonitor.network;

import android.content.Context;
import android.util.Log;

import org.java_websocket.handshake.ServerHandshake;

import java.net.URI;
import java.util.HashMap;
import java.util.Map;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import works.lm.powermonitor.model.PowerData;

/**
 * WebSocket client for real-time data streaming
 * Updated to match server-side implementation
 */
public class WebSocketClient extends org.java_websocket.client.WebSocketClient {
    private static final String TAG = "WebSocketClient";
    
    private Context context;
    private OnDataReceivedListener dataListener;
    private String collectorId;
    private boolean isConnected = false;
    private String authToken;
    
    public interface OnDataReceivedListener {
        void onPowerDataReceived(PowerData data);
        void onConnectionStatusChanged(boolean connected);
        void onError(String error);
        void onCollectorStatusChanged(String collectorId, boolean isOnline);
    }
    
    // Server message structure
    private static class RealtimeMessage {
        public String type;
        public String event;
        public Object data;
        public String timestamp;
        public String collector_id;
        public int user_id;
    }
    
    // Server power data structure
    private static class PowerDataMessage {
        public String collector_id;
        public String timestamp;
        public double voltage;
        public double current;
        public double power;
        public double energy;
        public double frequency;
        public double power_factor;
    }
    
    public WebSocketClient(Context context, String token, OnDataReceivedListener listener) {
        super(createUri(), createHeaders(token));
        this.context = context;
        this.dataListener = listener;
        this.authToken = token;
        
        // Set connection timeout
        setConnectionLostTimeout(30);
    }
    
    private static Map<String, String> createHeaders(String token) {
        Map<String, String> headers = new HashMap<>();
        headers.put("Authorization", "Bearer " + token);
        return headers;
    }
    
    private static URI createUri() {
        try {
            return new URI(NetworkConfig.getWebSocketUrl());
        } catch (Exception e) {
            Log.e(TAG, "Error creating WebSocket URI", e);
            return null;
        }
    }
    
    @Override
    public void onOpen(ServerHandshake handshake) {
        Log.d(TAG, "WebSocket connected to: " + NetworkConfig.getWebSocketUrl());
        isConnected = true;
        if (dataListener != null) {
            dataListener.onConnectionStatusChanged(true);
        }
    }
    
    @Override
    public void onMessage(String message) {
        Log.d(TAG, "Received message: " + message);
        try {
            Gson gson = new Gson();
            RealtimeMessage realtimeMsg = gson.fromJson(message, RealtimeMessage.class);
            
            if (realtimeMsg == null) {
                Log.w(TAG, "Received null message");
                return;
            }
            
            Log.d(TAG, "Message type: " + realtimeMsg.type + ", event: " + realtimeMsg.event);
            
            switch (realtimeMsg.type) {
                case "system":
                    handleSystemMessage(realtimeMsg);
                    break;
                    
                case "data":
                    if ("power_data".equals(realtimeMsg.event)) {
                        handlePowerDataMessage(realtimeMsg);
                    }
                    break;
                    
                case "status":
                    if ("collector_status".equals(realtimeMsg.event)) {
                        handleCollectorStatusMessage(realtimeMsg);
                    }
                    break;
                    
                case "alert":
                    handleAlertMessage(realtimeMsg);
                    break;
                    
                default:
                    Log.w(TAG, "Unknown message type: " + realtimeMsg.type);
                    break;
            }
        } catch (Exception e) {
            Log.e(TAG, "Error parsing message", e);
            if (dataListener != null) {
                dataListener.onError("Failed to parse data: " + e.getMessage());
            }
        }
    }
    
    private void handleSystemMessage(RealtimeMessage message) {
        Log.d(TAG, "System message - event: " + message.event);
        if ("connected".equals(message.event)) {
            Log.i(TAG, "WebSocket system confirmed connection");
            // If we have a collector ID, subscribe to it
            if (collectorId != null) {
                subscribeToCollector(collectorId);
            }
        }
    }
    
    private void handlePowerDataMessage(RealtimeMessage message) {
        try {
            Gson gson = new Gson();
            
            // Convert data object to PowerDataMessage
            JsonObject dataObj = gson.toJsonTree(message.data).getAsJsonObject();
            PowerDataMessage powerDataMsg = gson.fromJson(dataObj, PowerDataMessage.class);
            
            // Only process if it's for the collector we're subscribed to
            if (collectorId == null || collectorId.equals(powerDataMsg.collector_id)) {
                // Convert to our PowerData model
                PowerData powerData = new PowerData();
                powerData.setCollectorId(powerDataMsg.collector_id);
                powerData.setVoltage(powerDataMsg.voltage);
                powerData.setCurrent(powerDataMsg.current);
                powerData.setPower(powerDataMsg.power);
                powerData.setEnergy(powerDataMsg.energy);
                powerData.setFrequency(powerDataMsg.frequency);
                powerData.setPowerFactor(powerDataMsg.power_factor);
                
                // Parse timestamp if needed
                // powerData.setTimestamp(parseTimestamp(powerDataMsg.timestamp));
                
                if (dataListener != null) {
                    dataListener.onPowerDataReceived(powerData);
                }
                
                Log.d(TAG, "Power data received for collector: " + powerDataMsg.collector_id);
            } else {
                Log.d(TAG, "Ignoring power data for different collector: " + powerDataMsg.collector_id);
            }
        } catch (Exception e) {
            Log.e(TAG, "Error processing power data message", e);
        }
    }
    
    private void handleCollectorStatusMessage(RealtimeMessage message) {
        try {
            Gson gson = new Gson();
            JsonObject dataObj = gson.toJsonTree(message.data).getAsJsonObject();
            
            String collectorIdFromMsg = dataObj.get("collector_id").getAsString();
            boolean isOnline = dataObj.get("online").getAsBoolean();
            
            Log.d(TAG, "Collector status change: " + collectorIdFromMsg + " -> " + (isOnline ? "online" : "offline"));
            
            if (dataListener != null) {
                dataListener.onCollectorStatusChanged(collectorIdFromMsg, isOnline);
            }
        } catch (Exception e) {
            Log.e(TAG, "Error processing collector status message", e);
        }
    }
    
    private void handleAlertMessage(RealtimeMessage message) {
        Log.i(TAG, "Alert received - event: " + message.event);
        // Handle alerts if needed
    }
    
    @Override
    public void onClose(int code, String reason, boolean remote) {
        Log.d(TAG, "WebSocket closed: " + reason + " (code: " + code + ", remote: " + remote + ")");
        isConnected = false;
        if (dataListener != null) {
            dataListener.onConnectionStatusChanged(false);
        }
    }
    
    @Override
    public void onError(Exception ex) {
        Log.e(TAG, "WebSocket error", ex);
        isConnected = false;
        if (dataListener != null) {
            dataListener.onError("Connection error: " + ex.getMessage());
            dataListener.onConnectionStatusChanged(false);
        }
    }
    
    /**
     * Subscribe to a specific collector's data
     */
    public void subscribeToCollector(String collectorId) {
        this.collectorId = collectorId;
        Log.d(TAG, "Subscribed to collector: " + collectorId + " (connected: " + isConnected + ")");
        
        // Note: Server doesn't handle subscription messages currently,
        // so we just store the collector ID to filter incoming messages
        if (isConnected) {
            Log.d(TAG, "WebSocket is connected, ready to receive data for collector: " + collectorId);
        }
    }
    
    /**
     * Unsubscribe from current collector's data
     */
    public void unsubscribeFromCollector() {
        if (collectorId != null) {
            Log.d(TAG, "Unsubscribed from collector: " + collectorId);
            this.collectorId = null;
        }
    }
    
    public boolean isConnected() {
        return isConnected && !isClosed();
    }
    
    public String getCurrentCollectorId() {
        return collectorId;
    }
    
    /**
     * Reconnect to the WebSocket server
     */
    public void reconnect() {
        if (!isConnected() && isClosed()) {
            Log.d(TAG, "Attempting to reconnect...");
            new Thread(() -> {
                try {
                    reconnectBlocking();
                } catch (InterruptedException e) {
                    Log.e(TAG, "Reconnection interrupted", e);
                }
            }).start();
        }
    }
    
    /**
     * Send a ping to keep connection alive
     */
    public void sendPing() {
        if (isConnected()) {
            try {
                super.sendPing();
            } catch (Exception e) {
                Log.e(TAG, "Error sending ping", e);
            }
        }
    }
} 