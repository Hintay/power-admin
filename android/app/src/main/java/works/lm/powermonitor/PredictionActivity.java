package works.lm.powermonitor;

import android.os.Bundle;
import android.util.Log;
import android.view.MenuItem;
import android.widget.ArrayAdapter;
import android.widget.TextView;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;

import com.github.mikephil.charting.charts.BarChart;
import com.github.mikephil.charting.components.Description;
import com.github.mikephil.charting.components.XAxis;
import com.github.mikephil.charting.components.YAxis;
import com.github.mikephil.charting.data.BarData;
import com.github.mikephil.charting.data.BarDataSet;
import com.github.mikephil.charting.data.BarEntry;
import com.google.android.material.appbar.MaterialToolbar;
import com.google.android.material.button.MaterialButton;
import com.google.android.material.textfield.MaterialAutoCompleteTextView;

import java.util.ArrayList;
import java.util.Locale;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;
import works.lm.powermonitor.network.ApiClient;
import works.lm.powermonitor.network.ApiService;
import works.lm.powermonitor.utils.LanguageUtils;

/**
 * Activity for power consumption prediction and cost estimation
 */
public class PredictionActivity extends AppCompatActivity {
    
    private static final String TAG = "PredictionActivity";
    
    private TextView tvDailyConsumption, tvMonthlyConsumption;
    private TextView tvConfidenceLevel, tvPredictionAccuracy, tvCurrentConsumption, tvRemainingEnergy;
    private BarChart chartPrediction;
    private MaterialButton btnRefresh;
    private MaterialAutoCompleteTextView algorithmDropdown;
    
    private ApiClient apiClient;
    private String collectorId;
    private String selectedAlgorithm = ApiService.ALGORITHM_HYBRID;
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // Apply saved language setting
        LanguageUtils.applySavedLanguage(this);
        
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_prediction);
        
        // Initialize API client
        apiClient = ApiClient.getInstance(this);
        
        // Get collector ID from intent
        collectorId = getIntent().getStringExtra("collectorId");
        
        initViews();
        setupToolbar();
        setupChart();
        setupClickListeners();
        
        // Load prediction data
        loadPredictionData();
    }
    
    private void initViews() {
        tvDailyConsumption = findViewById(R.id.tvDailyConsumption);
        tvMonthlyConsumption = findViewById(R.id.tvMonthlyConsumption);
        tvConfidenceLevel = findViewById(R.id.tvConfidenceLevel);
        tvPredictionAccuracy = findViewById(R.id.tvPredictionAccuracy);
        tvCurrentConsumption = findViewById(R.id.tvCurrentConsumption);
        tvRemainingEnergy = findViewById(R.id.tvRemainingEnergy);
        chartPrediction = findViewById(R.id.chartPrediction);
        btnRefresh = findViewById(R.id.btnRefresh);
        algorithmDropdown = findViewById(R.id.algorithmDropdown);
    }
    
    private void setupToolbar() {
        MaterialToolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        
        if (getSupportActionBar() != null) {
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
            getSupportActionBar().setTitle(getString(R.string.power_prediction_title));
        }
    }
    
    private void setupChart() {
        // Configure chart
        Description description = new Description();
        description.setText("Hourly Energy Prediction");
        chartPrediction.setDescription(description);
        chartPrediction.setTouchEnabled(true);
        chartPrediction.setDragEnabled(true);
        chartPrediction.setScaleEnabled(true);
        chartPrediction.setPinchZoom(true);
        
        // Configure X axis
        XAxis xAxis = chartPrediction.getXAxis();
        xAxis.setPosition(XAxis.XAxisPosition.BOTTOM);
        xAxis.setDrawGridLines(true);
        xAxis.setGranularity(1f);
        
        // Configure Y axis
        YAxis leftAxis = chartPrediction.getAxisLeft();
        leftAxis.setDrawGridLines(true);
        leftAxis.setAxisMinimum(0f);
        
        YAxis rightAxis = chartPrediction.getAxisRight();
        rightAxis.setEnabled(false);
    }
    
    private void setupClickListeners() {
        btnRefresh.setOnClickListener(v -> loadPredictionData());
        
        // Setup algorithm dropdown
        String[] algorithms = {
            getString(R.string.algorithm_hybrid),
            getString(R.string.algorithm_linear),
            getString(R.string.algorithm_seasonal),
            getString(R.string.algorithm_moving_avg)
        };
        
        ArrayAdapter<String> adapter = new ArrayAdapter<>(this,
                android.R.layout.simple_dropdown_item_1line, algorithms);
        algorithmDropdown.setAdapter(adapter);
        
        algorithmDropdown.setOnItemClickListener((parent, view, position, id) -> {
            switch (position) {
                case 0:
                    selectedAlgorithm = ApiService.ALGORITHM_HYBRID;
                    break;
                case 1:
                    selectedAlgorithm = ApiService.ALGORITHM_LINEAR;
                    break;
                case 2:
                    selectedAlgorithm = ApiService.ALGORITHM_SEASONAL;
                    break;
                case 3:
                    selectedAlgorithm = ApiService.ALGORITHM_MOVING_AVERAGE;
                    break;
            }
            loadPredictionData();
        });
    }
    
    private void loadPredictionData() {
        if (collectorId == null) {
            Toast.makeText(this, getString(R.string.no_collector_selected), Toast.LENGTH_SHORT).show();
            return;
        }
        
        // Set loading state
        btnRefresh.setEnabled(false);
        btnRefresh.setText(getString(R.string.refreshing));
        
        Call<ApiService.ApiResponse<ApiService.PredictionData>> call = apiClient.getApiService().getPrediction(
                "Bearer " + apiClient.getStoredToken(),
                collectorId,
                selectedAlgorithm
        );
        
        call.enqueue(new Callback<ApiService.ApiResponse<ApiService.PredictionData>>() {
            @Override
            public void onResponse(Call<ApiService.ApiResponse<ApiService.PredictionData>> call, Response<ApiService.ApiResponse<ApiService.PredictionData>> response) {
                btnRefresh.setEnabled(true);
                btnRefresh.setText(getString(R.string.refresh_data));
                
                if (response.isSuccessful() && response.body() != null) {
                    ApiService.ApiResponse<ApiService.PredictionData> apiResponse = response.body();
                    if (apiResponse.success && apiResponse.data != null) {
                        updateUI(apiResponse.data);
                    } else {
                        String errorMsg = apiResponse.error != null ? apiResponse.error : getString(R.string.load_prediction_failed);
                        Toast.makeText(PredictionActivity.this, errorMsg, Toast.LENGTH_LONG).show();
                    }
                } else {
                    Toast.makeText(PredictionActivity.this, getString(R.string.load_prediction_failed), Toast.LENGTH_SHORT).show();
                }
            }
            
            @Override
            public void onFailure(Call<ApiService.ApiResponse<ApiService.PredictionData>> call, Throwable t) {
                btnRefresh.setEnabled(true);
                btnRefresh.setText(getString(R.string.refresh_data));
                Log.e(TAG, "Error loading prediction data", t);
                Toast.makeText(PredictionActivity.this, getString(R.string.network_error, t.getMessage()), Toast.LENGTH_SHORT).show();
            }
        });
    }
    
    private void updateUI(ApiService.PredictionData data) {
        if (data.prediction == null) {
            Toast.makeText(this, getString(R.string.no_prediction_data), Toast.LENGTH_SHORT).show();
            return;
        }
        
        ApiService.EnergyPrediction prediction = data.prediction;
        
        // Update energy consumption cards
        tvDailyConsumption.setText(String.format(Locale.getDefault(), "%.2f Wh", prediction.total_daily_energy_kwh));
        tvCurrentConsumption.setText(String.format(Locale.getDefault(), "%.2f Wh", data.actualConsumption));
        tvRemainingEnergy.setText(String.format(Locale.getDefault(), "%.2f Wh", prediction.remaining_energy_kwh));
        
        // Update prediction quality indicators
        tvConfidenceLevel.setText(String.format(Locale.getDefault(), "%.1f%%", prediction.confidence_level));
        tvPredictionAccuracy.setText(getAccuracyText(prediction.prediction_accuracy));
        
        // Update hourly prediction chart
        if (prediction.hourly_predictions != null && !prediction.hourly_predictions.isEmpty()) {
            updateChart(prediction.hourly_predictions);
        }
        
        // Show recommendations if available
        if (prediction.recommendations != null && !prediction.recommendations.isEmpty()) {
            showRecommendations(prediction.recommendations);
        }
        
        Toast.makeText(this, getString(R.string.prediction_data_updated), Toast.LENGTH_SHORT).show();
    }
    
    private void updateChart(java.util.List<ApiService.HourlyPrediction> hourlyPredictions) {
        ArrayList<BarEntry> powerEntries = new ArrayList<>();
        
        for (int i = 0; i < hourlyPredictions.size(); i++) {
            ApiService.HourlyPrediction prediction = hourlyPredictions.get(i);
            // Use predicted_energy_kwh from new API structure
            powerEntries.add(new BarEntry(prediction.hour, (float) (prediction.predicted_energy_kwh * 1000))); // Convert to Wh for display
        }
        
        BarDataSet dataSet = new BarDataSet(powerEntries, getString(R.string.predicted_energy_unit));
        dataSet.setColor(getResources().getColor(R.color.chart_power, null));
        dataSet.setValueTextSize(9f);
        dataSet.setDrawValues(false);
        
        BarData barData = new BarData(dataSet);
        chartPrediction.setData(barData);
        chartPrediction.invalidate();
    }

    /**
     * Convert prediction accuracy code to localized text
     */
    private String getAccuracyText(String accuracy) {
        switch (accuracy) {
            case "very_high":
                return getString(R.string.accuracy_very_high);
            case "high":
                return getString(R.string.accuracy_high);
            case "medium":
                return getString(R.string.accuracy_medium);
            case "low":
                return getString(R.string.accuracy_low);
            case "very_low":
                return getString(R.string.accuracy_very_low);
            default:
                return getString(R.string.accuracy_unknown);
        }
    }

    /**
     * Show prediction recommendations to user
     */
    private void showRecommendations(java.util.List<String> recommendations) {
        if (recommendations.isEmpty()) return;
        
        StringBuilder message = new StringBuilder(getString(R.string.recommendations_title));
        message.append("\n");
        
        for (int i = 0; i < recommendations.size() && i < 3; i++) { // Show max 3 recommendations
            message.append("• ").append(recommendations.get(i)).append("\n");
        }
        
        Toast.makeText(this, message.toString(), Toast.LENGTH_LONG).show();
    }
    
    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        if (item.getItemId() == android.R.id.home) {
            finish();
            return true;
        }
        return super.onOptionsItemSelected(item);
    }
} 