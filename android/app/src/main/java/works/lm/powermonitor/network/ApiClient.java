package works.lm.powermonitor.network;

import android.content.Context;
import android.content.SharedPreferences;
import android.util.Log;

import java.io.IOException;
import java.util.concurrent.TimeUnit;

import okhttp3.Interceptor;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import okhttp3.logging.HttpLoggingInterceptor;
import retrofit2.Retrofit;
import retrofit2.converter.gson.GsonConverterFactory;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;

/**
 * Singleton API client for managing HTTP requests and authentication
 * 
 * This class provides a centralized HTTP client with automatic token management,
 * request/response logging, and error handling. It uses the unified NetworkConfig
 * for server address management, allowing dynamic server configuration.
 * 
 * Features:
 * - Automatic JWT token refresh
 * - Request/response logging
 * - Centralized error handling
 * - Thread-safe token management
 * - Unified server configuration via NetworkConfig
 * 
 * The client automatically handles:
 * - Authorization headers
 * - Token expiration and refresh
 * - Network timeouts
 * - SSL/TLS configuration
 */
public class ApiClient {
    private static final String TAG = "ApiClient";
    
    // Use unified network configuration
    public static String getBaseUrl() {
        return NetworkConfig.getApiBaseUrl();
    }
    
    private static final String PREF_NAME = "power_monitor_prefs";
    private static final String TOKEN_KEY = "access_token";
    private static final String REFRESH_TOKEN_KEY = "refresh_token";
    
    private static ApiClient instance;
    private ApiService apiService;
    private Context context;
    private final Object tokenLock = new Object(); // For thread safety during token refresh
    private TokenRefreshHandler tokenRefreshHandler;
    
    private ApiClient(Context context) {
        this.context = context.getApplicationContext();
        setupRetrofit();
    }
    
    public static synchronized ApiClient getInstance(Context context) {
        if (instance == null) {
            instance = new ApiClient(context);
        }
        return instance;
    }
    
    private void setupRetrofit() {
        // Create Gson with custom date format
        Gson gson = new GsonBuilder()
                .setDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'")
                .create();
        
        // Create HTTP logging interceptor
        HttpLoggingInterceptor loggingInterceptor = new HttpLoggingInterceptor();
        loggingInterceptor.setLevel(HttpLoggingInterceptor.Level.BODY);
        
        // Create auth interceptor with token refresh capability
        Interceptor authInterceptor = chain -> {
            Request original = chain.request();

            // Skip auth for login and refresh endpoints
            String path = original.url().encodedPath();
            if (path.contains("/auth/login") || path.contains("/auth/refresh")) {
                return chain.proceed(original);
            }

            String token = getStoredToken();
            if (token != null && !token.isEmpty()) {
                Request.Builder requestBuilder = original.newBuilder()
                        .header("Authorization", "Bearer " + token);
                Request request = requestBuilder.build();

                Response response = chain.proceed(request);

                // Check if token is expired (401 Unauthorized)
                if (response.code() == 401) {
                    response.close(); // Close the original response

                    // Try to refresh token
                    synchronized (tokenLock) {
                        String refreshedToken = refreshAccessToken();
                        if (refreshedToken != null) {
                            // Retry the request with new token
                            Request newRequest = original.newBuilder()
                                    .header("Authorization", "Bearer " + refreshedToken)
                                    .build();
                            return chain.proceed(newRequest);
                        }
                    }
                }

                return response;
            }

            return chain.proceed(original);
        };
        
        // Create OkHttp client
        OkHttpClient client = new OkHttpClient.Builder()
                .addInterceptor(authInterceptor)
                .addInterceptor(loggingInterceptor)
                .connectTimeout(30, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .writeTimeout(30, TimeUnit.SECONDS)
                .build();
        
        // Create Retrofit instance
        String baseUrl = getBaseUrl();
        Log.d(TAG, "Initializing API client with base URL: " + baseUrl);
        
        Retrofit retrofit = new Retrofit.Builder()
                .baseUrl(baseUrl)
                .client(client)
                .addConverterFactory(GsonConverterFactory.create(gson))
                .build();
        
        apiService = retrofit.create(ApiService.class);
    }
    
    /**
     * Refresh access token using stored refresh token
     * @return new access token or null if refresh failed
     */
    private String refreshAccessToken() {
        try {
            String refreshToken = getStoredRefreshToken();
            if (refreshToken == null || refreshToken.isEmpty()) {
                Log.d(TAG, "No refresh token available");
                if (tokenRefreshHandler != null) {
                    tokenRefreshHandler.onNoRefreshToken();
                }
                return null;
            }
            
            Log.d(TAG, "Attempting to refresh access token");
            
            // Create refresh request
            ApiService.RefreshRequest refreshRequest = new ApiService.RefreshRequest(refreshToken);
            
            // Call refresh endpoint
            retrofit2.Response<ApiService.ApiResponse<ApiService.LoginData>> response = apiService.refreshToken(refreshRequest).execute();
            
            if (response.isSuccessful() && response.body() != null) {
                ApiService.ApiResponse<ApiService.LoginData> apiResponse = response.body();
                
                if (apiResponse.success && apiResponse.data != null) {
                    // Save new tokens
                    String newAccessToken = apiResponse.data.access_token;
                    String newRefreshToken = apiResponse.data.refresh_token;
                    
                    if (newAccessToken != null) {
                        saveToken(newAccessToken);
                    }
                    if (newRefreshToken != null) {
                        saveRefreshToken(newRefreshToken);
                    }
                    
                    Log.d(TAG, "Token refresh successful");
                    if (tokenRefreshHandler != null) {
                        tokenRefreshHandler.onTokenRefreshSuccess();
                    }
                    
                    return newAccessToken;
                } else {
                    Log.w(TAG, "Token refresh failed: " + apiResponse.message);
                    // Refresh token is invalid, clear all tokens
                    clearTokens();
                    if (tokenRefreshHandler != null) {
                        tokenRefreshHandler.onTokenRefreshFailed();
                    }
                    return null;
                }
            } else {
                Log.w(TAG, "Token refresh failed with response code: " + response.code());
                // Refresh token is invalid, clear all tokens
                clearTokens();
                if (tokenRefreshHandler != null) {
                    tokenRefreshHandler.onTokenRefreshFailed();
                }
                return null;
            }
        } catch (Exception e) {
            Log.e(TAG, "Error during token refresh", e);
            if (tokenRefreshHandler != null) {
                tokenRefreshHandler.onTokenRefreshFailed();
            }
            return null;
        }
    }
    
    public ApiService getApiService() {
        return apiService;
    }
    
    public void setTokenRefreshHandler(TokenRefreshHandler handler) {
        this.tokenRefreshHandler = handler;
    }
    
    public void saveToken(String token) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit().putString(TOKEN_KEY, token).apply();
    }
    
    public void saveRefreshToken(String refreshToken) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit().putString(REFRESH_TOKEN_KEY, refreshToken).apply();
    }
    
    public String getStoredToken() {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        return prefs.getString(TOKEN_KEY, null);
    }
    
    public String getStoredRefreshToken() {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        return prefs.getString(REFRESH_TOKEN_KEY, null);
    }
    
    public void clearToken() {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit().remove(TOKEN_KEY).apply();
    }
    
    public void clearRefreshToken() {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit().remove(REFRESH_TOKEN_KEY).apply();
    }
    
    public void clearTokens() {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit()
                .remove(TOKEN_KEY)
                .remove(REFRESH_TOKEN_KEY)
                .apply();
    }
    
    public boolean isLoggedIn() {
        String token = getStoredToken();
        String refreshToken = getStoredRefreshToken();
        return (token != null && !token.isEmpty()) || (refreshToken != null && !refreshToken.isEmpty());
    }
    
    /**
     * Get complete WebSocket URL (using unified configuration)
     */
    public static String getWebSocketUrl() {
        return NetworkConfig.getWebSocketUrl();
    }
} 