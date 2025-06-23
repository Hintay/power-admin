package works.lm.powermonitor.network;

import works.lm.powermonitor.model.Collector;
import works.lm.powermonitor.model.PowerData;
import works.lm.powermonitor.model.User;

import java.util.List;
import java.util.Map;

import retrofit2.Call;
import retrofit2.http.*;

/**
 * API service interface for REST API calls
 * Updated to match server-side API definitions
 */
public interface ApiService {

    // Prediction algorithm constants
    String ALGORITHM_HYBRID = "hybrid";
    String ALGORITHM_LINEAR = "linear";
    String ALGORITHM_SEASONAL = "seasonal";
    String ALGORITHM_MOVING_AVERAGE = "moving_average";

    // Authentication endpoints
    @POST("api/auth/login")
    Call<ApiResponse<LoginData>> login(@Body LoginRequest request);

    @POST("api/auth/refresh")
    Call<ApiResponse<LoginData>> refreshToken(@Body RefreshRequest request);

    @POST("api/auth/logout")
    Call<ApiResponse<Void>> logout(@Header("Authorization") String token);

    // Collector endpoints - updated paths to match server
    @GET("api/client/data/collectors")
    Call<ApiResponse<List<Collector>>> getCollectors(@Header("Authorization") String token);

    @GET("api/client/data/collectors/{id}")
    Call<ApiResponse<CollectorInfoData>> getCollector(@Header("Authorization") String token, @Path("id") String collectorId);

    @GET("api/client/data/collectors/{id}/status")
    Call<ApiResponse<CollectorStatusData>> getCollectorStatus(@Header("Authorization") String token, @Path("id") String collectorId);

    // Real-time data endpoints - updated paths
    @GET("api/client/data/collectors/{id}/latest")
    Call<ApiResponse<PowerData>> getRealtimeData(@Header("Authorization") String token, @Path("id") String collectorId);

    // Historical data endpoints - updated to match server parameters
    @GET("api/client/data/collectors/{id}/history")
    Call<ApiResponse<List<PowerData>>> getHistoryData(
            @Header("Authorization") String token,
            @Path("id") String collectorId,
            @Query("start") String startTime,
            @Query("end") String endTime,
            @Query("limit") Integer limit
    );

    // Analytics endpoints - updated path and response structure
    @GET("api/client/analytics/prediction/{collectorId}")
    Call<ApiResponse<PredictionData>> getPrediction(
            @Header("Authorization") String token,
            @Path("collectorId") String collectorId,
            @Query("algorithm") String algorithm
    );

    // Convenience method for default hybrid algorithm
    default Call<ApiResponse<PredictionData>> getPrediction(String token, String collectorId) {
        return getPrediction(token, collectorId, ALGORITHM_HYBRID);
    }

    // Additional analytics endpoints that exist on server
    @GET("api/client/analytics/dashboard")
    Call<ApiResponse<DashboardData>> getDashboard(@Header("Authorization") String token);

    @GET("api/client/analytics/energy-consumption")
    Call<ApiResponse<List<EnergyConsumptionData>>> getEnergyConsumption(
            @Header("Authorization") String token,
            @Query("period") String period
    );

    // Inner classes for request/response models - updated to match server
    class LoginRequest {
        public String username;
        public String password;

        public LoginRequest(String username, String password) {
            this.username = username;
            this.password = password;
        }
    }

    class RefreshRequest {
        public String refresh_token;

        public RefreshRequest(String refreshToken) {
            this.refresh_token = refreshToken;
        }
    }

    // Updated API response wrapper to match server format
    class ApiResponse<T> {
        public boolean success;
        public String message;
        public T data;
        public String error;
    }

    class LoginData {
        public User user;
        public String access_token;
        public String refresh_token;
    }

    // Updated collector info response structure
    class CollectorInfoData {
        public Collector collector;
        public CollectorConfig config;
    }

    class CollectorConfig {
        public String collector_id;
        public int sample_interval;
        public int upload_interval;
        public int max_cache_size;
        public boolean auto_upload;
        public int compression_level;
    }

    // Updated collector status response
    class CollectorStatusData {
        public Collector collector;
        public boolean is_online;
        public String last_data_time;
        public long data_count;
    }

    // Dashboard data structure
    class DashboardData {
        public DashboardSummary summary;
        public List<Collector> collectors;
        public List<PowerData> recent_data;
    }

    class DashboardSummary {
        public int total_collectors;
        public int online_collectors;
        public double total_power;
        public double total_energy;
        public int alerts_count;
    }

    // Energy consumption data
    class EnergyConsumptionData {
        public String collector_id;
        public String timestamp;
        public double energy;
    }

    // Updated prediction response structure to match server
    class PredictionData {
        public EnergyPrediction prediction;
        public double actualConsumption;
        public String algorithmUsed;
        public long dataPoints;
        public String predictionTime;
        public List<String> collectors;
    }

    class EnergyPrediction {
        public double total_daily_energy_kwh;
        public double remaining_energy_kwh;
        public String predicted_end_time;
        public double confidence_level;
        public String prediction_accuracy;
        public List<HourlyPrediction> hourly_predictions;
        public List<CollectorPrediction> collector_predictions;
        public PredictionMetrics model_metrics;
        public List<String> recommendations;
    }

    class HourlyPrediction {
        public int hour;
        public double predicted_energy_kwh;
        public double predicted_avg_power;
        public double confidence_interval;
    }

    class CollectorPrediction {
        public String collector_id;
        public String collector_name;
        public double predicted_energy_kwh;
        public double current_energy_kwh;
        public double percentage_of_total;
    }

    class PredictionMetrics {
        public String algorithm;
        public String data_quality;
        public double historical_accuracy;
        public double trend_strength;
        public double seasonality_score;
        public double noise_level;
    }
} 