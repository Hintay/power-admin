package works.lm.powermonitor;

import android.content.Context;
import android.content.Intent;
import android.graphics.Color;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.MenuItem;
import android.view.View;
import android.widget.ArrayAdapter;
import android.widget.Spinner;
import android.widget.TextView;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.appcompat.app.ActionBarDrawerToggle;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.view.GravityCompat;
import androidx.drawerlayout.widget.DrawerLayout;

import com.github.mikephil.charting.charts.LineChart;
import com.github.mikephil.charting.components.Description;
import com.github.mikephil.charting.components.MarkerView;
import com.github.mikephil.charting.components.XAxis;
import com.github.mikephil.charting.components.YAxis;
import com.github.mikephil.charting.data.Entry;
import com.github.mikephil.charting.data.LineData;
import com.github.mikephil.charting.data.LineDataSet;
import com.github.mikephil.charting.formatter.ValueFormatter;
import com.github.mikephil.charting.highlight.Highlight;
import com.github.mikephil.charting.utils.MPPointF;
import com.google.android.material.appbar.MaterialToolbar;
import com.google.android.material.button.MaterialButton;
import com.google.android.material.navigation.NavigationView;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;
import works.lm.powermonitor.model.Collector;
import works.lm.powermonitor.model.PowerData;
import works.lm.powermonitor.network.ApiClient;
import works.lm.powermonitor.network.WebSocketClient;
import works.lm.powermonitor.utils.LanguageUtils;
import works.lm.powermonitor.network.ApiService;

/**
 * Main activity for real-time power monitoring
 */
public class MainActivity extends AppCompatActivity 
        implements NavigationView.OnNavigationItemSelectedListener,
                   WebSocketClient.OnDataReceivedListener {

    private static final String TAG = "MainActivity";
    private static final int CHART_MAX_ENTRIES = 50;

    // Chart data types
    public enum ChartDataType {
        POWER(R.string.power_w_unit, R.color.chart_power, "W"),
        VOLTAGE(R.string.voltage_v_unit, R.color.chart_voltage, "V"),
        CURRENT(R.string.current_a_unit, R.color.chart_current, "A"),
        ENERGY(R.string.energy_kwh_unit, R.color.chart_energy, "Wh");

        private final int labelResId;
        private final int colorResId;
        private final String unit;

        ChartDataType(int labelResId, int colorResId, String unit) {
            this.labelResId = labelResId;
            this.colorResId = colorResId;
            this.unit = unit;
        }

        public int getLabelResId() { return labelResId; }
        public int getColorResId() { return colorResId; }
        public String getUnit() { return unit; }
    }

    // UI components
    private DrawerLayout drawerLayout;
    private Spinner spinnerCollectors;
    private TextView tvConnectionStatus;
    private TextView tvDataType, tvDataTimestamp;
    private TextView tvVoltage, tvCurrent, tvPower, tvEnergy;
    private TextView tvChartTitle;
    private LineChart chartPower;
    private MaterialButton btnHistory, btnPrediction;
    private View hintCard;
    private View btnCloseHint;

    // Data card containers for click handling
    private View cardVoltage, cardCurrent, cardPower, cardEnergy;
    
    // Selection indicators
    private View ivVoltageSelected, ivCurrentSelected, ivPowerSelected, ivEnergySelected;

    // Data and network
    private ApiClient apiClient;
    private WebSocketClient webSocketClient;
    private List<Collector> collectors;
    private ArrayAdapter<Collector> collectorsAdapter;
    private String currentCollectorId;           // UUID for WebSocket
    private int currentCollectorDbId;            // Database ID for API calls

    // Chart data
    private ArrayList<Entry> powerEntries;
    private ArrayList<PowerData> powerDataList;  // Store PowerData objects for marker display
    private LineDataSet powerDataSet;
    private LineData lineData;
    private ChartDataType currentChartType = ChartDataType.POWER; // Default to power chart

    // Data state management
    private boolean isRealTimeData = false;
    private PowerData lastKnownData = null;

    // UI update handler and async processing
    private Handler uiHandler;
    private ExecutorService executorService;

    /**
     * Custom MarkerView to display detailed data when touching chart point
     */
    public class PowerMarkerView extends MarkerView {
        
        private TextView tvTimestamp, tvValue, tvPower, tvVoltage, tvCurrent, tvEnergy;
        
        public PowerMarkerView(Context context, int layoutResource) {
            super(context, layoutResource);
            
            tvTimestamp = findViewById(R.id.tvMarkerTimestamp);
            tvValue = findViewById(R.id.tvMarkerValue);
            tvPower = findViewById(R.id.tvMarkerPower);
            tvVoltage = findViewById(R.id.tvMarkerVoltage);
            tvCurrent = findViewById(R.id.tvMarkerCurrent);
            tvEnergy = findViewById(R.id.tvMarkerEnergy);
        }
        
        @Override
        public void refreshContent(Entry e, Highlight highlight) {
            if (powerDataList == null || e.getX() < 0 || e.getX() >= powerDataList.size()) {
                return;
            }
            
            PowerData data = powerDataList.get((int) e.getX());
            SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault());
            
            // Display timestamp with null check
            if (data.getTimestamp() != null) {
                tvTimestamp.setText(formatter.format(data.getTimestamp()));
            } else {
                tvTimestamp.setText(getString(R.string.no_timestamp_available));
            }
            
            // Display current metric value with emphasis
            String currentValue = "";
            switch (currentChartType) {
                case POWER:
                    currentValue = String.format("%.1f W", data.getPower());
                    break;
                case VOLTAGE:
                    currentValue = String.format("%.1f V", data.getVoltage());
                    break;
                case CURRENT:
                    currentValue = String.format("%.2f A", data.getCurrent());
                    break;
                case ENERGY:
                    currentValue = String.format("%.0f Wh", data.getEnergy());
                    break;
            }
            tvValue.setText(currentValue);
            
            // Display all metrics for reference
            tvPower.setText(String.format(getString(R.string.power_format), data.getPower()));
            tvVoltage.setText(String.format(getString(R.string.voltage_format), data.getVoltage()));
            tvCurrent.setText(String.format(getString(R.string.current_format), data.getCurrent()));
            tvEnergy.setText(String.format(getString(R.string.energy_format), data.getEnergy()));
            
            super.refreshContent(e, highlight);
        }
        
        @Override
        public MPPointF getOffset() {
            return new MPPointF(-(getWidth() / 2), -getHeight());
        }
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // Apply saved language setting
        LanguageUtils.applySavedLanguage(this);
        
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        // Initialize API client
        apiClient = ApiClient.getInstance(this);
        
        // Check if user is logged in
        if (!apiClient.isLoggedIn()) {
            navigateToLogin();
            return;
        }

        // Initialize handlers and executors
        uiHandler = new Handler(Looper.getMainLooper());
        executorService = Executors.newSingleThreadExecutor();

        // Initialize UI
        initViews();
        setupToolbar();
        setupNavigationDrawer();
        setupSpinner();
        setupChart();
        setupClickListeners();

        // Initialize card selection state (default to power)
        initializeDefaultCardSelection();

        // Load collectors
        loadCollectors();
    }

    /**
     * Initialize default card selection state (power selected by default)
     */
    private void initializeDefaultCardSelection() {
        // Ensure we start with power chart type
        currentChartType = ChartDataType.POWER;
        
        // Update chart appearance for power
        updateChartForDataType();
        
        // Update chart title
        updateChartTitle();
        
        // Update visual selection state
        updateCardSelectionState();
    }

    private void initViews() {
        drawerLayout = findViewById(R.id.main);
        spinnerCollectors = findViewById(R.id.spinnerCollectors);
        tvConnectionStatus = findViewById(R.id.tvConnectionStatus);
        tvDataType = findViewById(R.id.tvDataType);
        tvDataTimestamp = findViewById(R.id.tvDataTimestamp);
        tvVoltage = findViewById(R.id.tvVoltage);
        tvCurrent = findViewById(R.id.tvCurrent);
        tvPower = findViewById(R.id.tvPower);
        tvEnergy = findViewById(R.id.tvEnergy);
        tvChartTitle = findViewById(R.id.tvChartTitle);
        chartPower = findViewById(R.id.chartPower);
        btnHistory = findViewById(R.id.btnHistory);
        btnPrediction = findViewById(R.id.btnPrediction);
        hintCard = findViewById(R.id.hintCard);
        btnCloseHint = findViewById(R.id.btnCloseHint);
        cardVoltage = findViewById(R.id.cardVoltage);
        cardCurrent = findViewById(R.id.cardCurrent);
        cardPower = findViewById(R.id.cardPower);
        cardEnergy = findViewById(R.id.cardEnergy);
        
        // Initialize selection indicators
        ivVoltageSelected = findViewById(R.id.ivVoltageSelected);
        ivCurrentSelected = findViewById(R.id.ivCurrentSelected);
        ivPowerSelected = findViewById(R.id.ivPowerSelected);
        ivEnergySelected = findViewById(R.id.ivEnergySelected);
    }

    private void setupToolbar() {
        MaterialToolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
    }

    private void setupNavigationDrawer() {
        NavigationView navigationView = findViewById(R.id.navigationView);
        navigationView.setNavigationItemSelectedListener(this);

        ActionBarDrawerToggle toggle = new ActionBarDrawerToggle(
                this, drawerLayout, R.string.navigation_drawer_open, R.string.navigation_drawer_close);
        drawerLayout.addDrawerListener(toggle);
        toggle.syncState();

        if (getSupportActionBar() != null) {
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
            getSupportActionBar().setHomeButtonEnabled(true);
        }
    }

    private void setupSpinner() {
        collectors = new ArrayList<>();
        collectorsAdapter = new ArrayAdapter<>(this, android.R.layout.simple_spinner_item, collectors);
        collectorsAdapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item);
        spinnerCollectors.setAdapter(collectorsAdapter);

        spinnerCollectors.setOnItemSelectedListener(new android.widget.AdapterView.OnItemSelectedListener() {
            @Override
            public void onItemSelected(android.widget.AdapterView<?> parent, android.view.View view, int position, long id) {
                if (position >= 0 && position < collectors.size()) {
                    Collector selectedCollector = collectors.get(position);
                    // Pass the collector object to get both IDs
                    switchToCollector(selectedCollector);
                }
            }

            @Override
            public void onNothingSelected(android.widget.AdapterView<?> parent) {}
        });
    }

    private void setupChart() {
        powerEntries = new ArrayList<>();
        powerDataList = new ArrayList<>();
        powerDataSet = new LineDataSet(powerEntries, getString(currentChartType.getLabelResId()));
        powerDataSet.setColor(getResources().getColor(currentChartType.getColorResId(), null));
        powerDataSet.setCircleColor(getResources().getColor(currentChartType.getColorResId(), null));
        powerDataSet.setLineWidth(2f);
        powerDataSet.setCircleRadius(3f);
        powerDataSet.setDrawFilled(false);
        powerDataSet.setValueTextSize(9f);
        powerDataSet.setDrawValues(false);

        lineData = new LineData(powerDataSet);
        chartPower.setData(lineData);

        // Configure chart
        Description description = new Description();
        description.setText("");
        chartPower.setDescription(description);
        chartPower.setTouchEnabled(true);
        chartPower.setDragEnabled(true);
        chartPower.setScaleEnabled(true);
        chartPower.setPinchZoom(true);

        // Configure X axis to show time
        XAxis xAxis = chartPower.getXAxis();
        xAxis.setPosition(XAxis.XAxisPosition.BOTTOM);
        xAxis.setDrawGridLines(true);
        xAxis.setGranularity(1f);
        xAxis.setValueFormatter(new ValueFormatter() {
            @Override
            public String getFormattedValue(float value) {
                if (powerDataList != null && value >= 0 && value < powerDataList.size()) {
                    PowerData data = powerDataList.get((int) value);
                    // Check if timestamp is not null before formatting
                    if (data.getTimestamp() != null) {
                        SimpleDateFormat timeFormat = new SimpleDateFormat("HH:mm:ss", Locale.getDefault());
                        return timeFormat.format(data.getTimestamp());
                    } else {
                        // Return a default time format or current time
                        return String.format(Locale.getDefault(), "%02d:%02d:%02d", 
                                (int)value / 3600, ((int)value % 3600) / 60, (int)value % 60);
                    }
                }
                return "";
            }
        });

        // Configure Y axis
        YAxis leftAxis = chartPower.getAxisLeft();
        leftAxis.setDrawGridLines(true);
        leftAxis.setAxisMinimum(0f);

        YAxis rightAxis = chartPower.getAxisRight();
        rightAxis.setEnabled(false);

        // Set custom marker view for displaying detailed data
        PowerMarkerView markerView = new PowerMarkerView(this, R.layout.custom_marker_view);
        chartPower.setMarker(markerView);

        chartPower.invalidate();
    }

    private void setupClickListeners() {
        // Navigation buttons
        btnHistory.setOnClickListener(v -> {
            // Navigate to history activity
            Intent intent = new Intent(this, HistoryActivity.class);
            if (currentCollectorDbId != 0) {
                intent.putExtra("collectorId", String.valueOf(currentCollectorDbId));
                intent.putExtra("collectorUuid", currentCollectorId);
            }
            startActivity(intent);
        });

        btnPrediction.setOnClickListener(v -> {
            // Navigate to prediction activity
            Intent intent = new Intent(this, PredictionActivity.class);
            if (currentCollectorDbId != 0) {
                intent.putExtra("collectorId", String.valueOf(currentCollectorDbId));
                intent.putExtra("collectorUuid", currentCollectorId);
            }
            startActivity(intent);
        });

        // Data card click listeners for chart switching
        cardVoltage.setOnClickListener(v -> switchChartType(ChartDataType.VOLTAGE));
        cardCurrent.setOnClickListener(v -> switchChartType(ChartDataType.CURRENT));
        cardPower.setOnClickListener(v -> switchChartType(ChartDataType.POWER));
        cardEnergy.setOnClickListener(v -> switchChartType(ChartDataType.ENERGY));

        // Hint card close button
        btnCloseHint.setOnClickListener(v -> hideHintCard());
    }

    /**
     * Hide the hint card with animation
     */
    private void hideHintCard() {
        if (hintCard != null) {
            // Animate hide
            hintCard.animate()
                    .alpha(0f)
                    .scaleY(0f)
                    .setDuration(300)
                    .withEndAction(() -> hintCard.setVisibility(View.GONE))
                    .start();
        }
    }

    /**
     * Switch chart to display different data type
     */
    private void switchChartType(ChartDataType newType) {
        if (currentChartType == newType) {
            return; // Already showing this type
        }

        currentChartType = newType;
        
        // Update chart appearance
        updateChartForDataType();
        
        // Update chart title
        updateChartTitle();
        
        // Update card selection state
        updateCardSelectionState();
        
        // Rebuild chart data if we have existing data
        rebuildChartDataAsync();
        
        // Show feedback
        String message = getString(R.string.chart_switched_to, getString(currentChartType.getLabelResId()));
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show();
    }

    /**
     * Update chart appearance for current data type
     */
    private void updateChartForDataType() {
        int color = getResources().getColor(currentChartType.getColorResId(), null);
        
        powerDataSet.setLabel(getString(currentChartType.getLabelResId()));
        powerDataSet.setColor(color);
        powerDataSet.setCircleColor(color);
        
        powerDataSet.notifyDataSetChanged();
        lineData.notifyDataChanged();
        chartPower.notifyDataSetChanged();
        chartPower.invalidate();
    }

    /**
     * Update chart title based on current data type
     */
    private void updateChartTitle() {
        String title = "";
        switch (currentChartType) {
            case POWER:
                title = getString(R.string.realtime_power_curve);
                break;
            case VOLTAGE:
                title = getString(R.string.realtime_voltage_curve);
                break;
            case CURRENT:
                title = getString(R.string.realtime_current_curve);
                break;
            case ENERGY:
                title = getString(R.string.realtime_energy_curve);
                break;
        }
        tvChartTitle.setText(title);
    }

    /**
     * Update visual selection state of data cards
     */
    private void updateCardSelectionState() {
        // Reset all cards to normal state
        resetCardState(cardVoltage);
        resetCardState(cardCurrent);
        resetCardState(cardPower);
        resetCardState(cardEnergy);
        
        // Hide all selection indicators
        hideAllSelectionIndicators();
        
        // Highlight selected card and show its indicator
        View selectedCard = null;
        View selectedIndicator = null;
        
        switch (currentChartType) {
            case VOLTAGE:
                selectedCard = cardVoltage;
                selectedIndicator = ivVoltageSelected;
                break;
            case CURRENT:
                selectedCard = cardCurrent;
                selectedIndicator = ivCurrentSelected;
                break;
            case POWER:
                selectedCard = cardPower;
                selectedIndicator = ivPowerSelected;
                break;
            case ENERGY:
                selectedCard = cardEnergy;
                selectedIndicator = ivEnergySelected;
                break;
        }
        
        if (selectedCard != null) {
            highlightCard(selectedCard);
        }
        
        if (selectedIndicator != null) {
            showSelectionIndicator(selectedIndicator);
        }
    }

    /**
     * Hide all selection indicators
     */
    private void hideAllSelectionIndicators() {
        ivVoltageSelected.setVisibility(View.GONE);
        ivCurrentSelected.setVisibility(View.GONE);
        ivPowerSelected.setVisibility(View.GONE);
        ivEnergySelected.setVisibility(View.GONE);
    }

    /**
     * Show selection indicator with animation
     */
    private void showSelectionIndicator(View indicator) {
        indicator.setVisibility(View.VISIBLE);
        indicator.setAlpha(0f);
        indicator.setScaleX(0.5f);
        indicator.setScaleY(0.5f);
        
        indicator.animate()
                .alpha(1f)
                .scaleX(1f)
                .scaleY(1f)
                .setDuration(300)
                .start();
    }

    /**
     * Reset card to normal visual state
     */
    private void resetCardState(View card) {
        // Reset appearance with animation
        card.animate()
                .alpha(0.8f)
                .scaleX(1.0f)
                .scaleY(1.0f)
                .setDuration(200)
                .start();
        
        // Reset background and elevation
        card.setBackground(getResources().getDrawable(R.drawable.card_normal_background, null));
        if (card instanceof com.google.android.material.card.MaterialCardView) {
            ((com.google.android.material.card.MaterialCardView) card).setCardElevation(4f);
        }
    }

    /**
     * Highlight selected card with enhanced visual effects
     */
    private void highlightCard(View card) {
        // Enhanced highlight with animation
        card.animate()
                .alpha(1.0f)
                .scaleX(1.08f)
                .scaleY(1.08f)
                .setDuration(200)
                .start();
        
        // Enhanced background and elevation
        card.setBackground(getResources().getDrawable(R.drawable.card_selected_background, null));
        if (card instanceof com.google.android.material.card.MaterialCardView) {
            ((com.google.android.material.card.MaterialCardView) card).setCardElevation(8f);
        }
        
        // Add subtle vibration feedback if available
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
            android.os.VibrationEffect effect = android.os.VibrationEffect.createOneShot(50, android.os.VibrationEffect.DEFAULT_AMPLITUDE);
            android.os.Vibrator vibrator = (android.os.Vibrator) getSystemService(android.content.Context.VIBRATOR_SERVICE);
            if (vibrator != null && vibrator.hasVibrator()) {
                vibrator.vibrate(effect);
            }
        }
    }

    /**
     * Rebuild chart data for current data type asynchronously
     */
    private void rebuildChartDataAsync() {
        if (powerDataList.isEmpty()) {
            return;
        }

        // Check if executorService is available and not shutdown
        if (executorService == null || executorService.isShutdown() || executorService.isTerminated()) {
            Log.w(TAG, "ExecutorService is not available, skipping chart rebuild");
            return;
        }

        try {
            executorService.execute(() -> {
                // Check if activity is still valid
                if (isFinishing() || isDestroyed()) {
                    return;
                }

                List<Entry> newEntries = new ArrayList<>();
                
                // Convert data based on current chart type
                for (int i = 0; i < powerDataList.size(); i++) {
                    PowerData data = powerDataList.get(i);
                    float value = 0f;
                    
                    switch (currentChartType) {
                        case POWER:
                            value = (float) data.getPower();
                            break;
                        case VOLTAGE:
                            value = (float) data.getVoltage();
                            break;
                        case CURRENT:
                            value = (float) data.getCurrent();
                            break;
                        case ENERGY:
                            value = (float) data.getEnergy();
                            break;
                    }
                    
                    newEntries.add(new Entry(i, value));
                }
                
                // Update UI on main thread
                uiHandler.post(() -> {
                    // Double-check if activity is still valid before updating UI
                    if (isFinishing() || isDestroyed()) {
                        return;
                    }

                    powerEntries.clear();
                    powerEntries.addAll(newEntries);
                    
                    powerDataSet.notifyDataSetChanged();
                    lineData.notifyDataChanged();
                    chartPower.notifyDataSetChanged();
                    chartPower.invalidate();
                    
                    // Auto-scale chart
                    chartPower.fitScreen();
                });
            });
        } catch (Exception e) {
            Log.e(TAG, "Error executing chart rebuild task", e);
        }
    }

    private void loadCollectors() {
        Call<ApiService.ApiResponse<List<Collector>>> call = apiClient.getApiService().getCollectors("Bearer " + apiClient.getStoredToken());
        call.enqueue(new Callback<ApiService.ApiResponse<List<Collector>>>() {
            @Override
            public void onResponse(Call<ApiService.ApiResponse<List<Collector>>> call, Response<ApiService.ApiResponse<List<Collector>>> response) {
                if (isFinishing() || isDestroyed()) {
                    return;
                }
                
                if (response.isSuccessful() && response.body() != null) {
                    ApiService.ApiResponse<List<Collector>> apiResponse = response.body();
                    
                    if (apiResponse.success && apiResponse.data != null) {
                        collectors.clear();
                        collectors.addAll(apiResponse.data);
                        collectorsAdapter.notifyDataSetChanged();

                        // Select first collector if available
                        if (!collectors.isEmpty()) {
                            spinnerCollectors.setSelection(0);
                            switchToCollector(collectors.get(0));
                        }
                    } else {
                        String errorMessage = apiResponse.error != null ? apiResponse.error : getString(R.string.load_collectors_failed);
                        Toast.makeText(MainActivity.this, errorMessage, Toast.LENGTH_SHORT).show();
                    }
                } else {
                    Toast.makeText(MainActivity.this, getString(R.string.load_collectors_failed), Toast.LENGTH_SHORT).show();
                }
            }

            @Override
            public void onFailure(Call<ApiService.ApiResponse<List<Collector>>> call, Throwable t) {
                if (isFinishing() || isDestroyed()) {
                    return;
                }
                
                Log.e(TAG, "Error loading collectors", t);
                Toast.makeText(MainActivity.this, getString(R.string.network_error, t.getMessage()), Toast.LENGTH_SHORT).show();
            }
        });
    }

    private void switchToCollector(Collector collector) {
        if (collector.getCollectorId() != null && collector.getCollectorId().equals(currentCollectorId)) {
            return;
        }

        // Disconnect from previous collector
        disconnectWebSocket();

        // Store both IDs for different purposes
        currentCollectorId = collector.getCollectorId();     // UUID for WebSocket
        currentCollectorDbId = collector.getId();            // Database ID for API calls

        // Clear chart data
        clearChartDataAsync();

        // Reset data state to historical mode
        isRealTimeData = false;
        lastKnownData = null;
        
        // Load last known data first, which will show historical data status
        loadLastKnownData();

        // Connect to new collector (will keep showing historical data until real-time data arrives)
        connectWebSocket();
    }

    private void clearChartDataAsync() {
        // Check if executorService is available and not shutdown
        if (executorService == null || executorService.isShutdown() || executorService.isTerminated()) {
            Log.w(TAG, "ExecutorService is not available, skipping chart clear");
            return;
        }

        try {
            executorService.execute(() -> {
                // Check if activity is still valid
                if (isFinishing() || isDestroyed()) {
                    return;
                }

                // Clear data in background thread
                List<Entry> newPowerEntries = new ArrayList<>();
                List<PowerData> newPowerDataList = new ArrayList<>();
                
                // Update UI on main thread
                uiHandler.post(() -> {
                    // Double-check if activity is still valid before updating UI
                    if (isFinishing() || isDestroyed()) {
                        return;
                    }

                    powerEntries.clear();
                    powerEntries.addAll(newPowerEntries);
                    powerDataList.clear();
                    powerDataList.addAll(newPowerDataList);
                    
                    chartPower.notifyDataSetChanged();
                    chartPower.invalidate();
                });
            });
        } catch (Exception e) {
            Log.e(TAG, "Error executing chart clear task", e);
        }
    }

    private void connectWebSocket() {
        String token = apiClient.getStoredToken();
        if (token != null && currentCollectorId != null) {
            webSocketClient = new WebSocketClient(this, token, this);
            webSocketClient.connect();
            // Keep showing historical data until real-time data arrives
            // isRealTimeData remains false until onPowerDataReceived is called
        }
    }

    private void disconnectWebSocket() {
        if (webSocketClient != null) {
            webSocketClient.unsubscribeFromCollector();
            webSocketClient.close();
            webSocketClient = null;
        }
    }

    /**
     * Load the last known data for the current collector
     */
    private void loadLastKnownData() {
        if (currentCollectorDbId == 0) {
            return;
        }

        Call<ApiService.ApiResponse<PowerData>> call = apiClient.getApiService()
                .getRealtimeData("Bearer " + apiClient.getStoredToken(), String.valueOf(currentCollectorDbId));
        
        call.enqueue(new Callback<ApiService.ApiResponse<PowerData>>() {
            @Override
            public void onResponse(Call<ApiService.ApiResponse<PowerData>> call, 
                                 Response<ApiService.ApiResponse<PowerData>> response) {
                if (response.isSuccessful() && response.body() != null) {
                    ApiService.ApiResponse<PowerData> apiResponse = response.body();
                    
                    if (apiResponse.success && apiResponse.data != null) {
                        lastKnownData = apiResponse.data;
                        // Always show as historical data since this is loaded from API, not real-time
                        uiHandler.post(() -> {
                            if (!isFinishing() && !isDestroyed()) {
                                updateUIWithPowerData(lastKnownData, false);
                            }
                        });
                    } else {
                        Log.w(TAG, "No last known data available: " + apiResponse.error);
                        uiHandler.post(() -> {
                            if (!isFinishing() && !isDestroyed()) {
                                showNoDataAvailable();
                            }
                        });
                    }
                } else {
                    Log.e(TAG, "Failed to load last known data: " + response.message());
                    uiHandler.post(() -> {
                        if (!isFinishing() && !isDestroyed()) {
                            showNoDataAvailable();
                        }
                    });
                }
            }

            @Override
            public void onFailure(Call<ApiService.ApiResponse<PowerData>> call, Throwable t) {
                Log.e(TAG, "Error loading last known data", t);
                uiHandler.post(() -> {
                    if (!isFinishing() && !isDestroyed()) {
                        showNoDataAvailable();
                    }
                });
            }
        });
    }

    /**
     * Show UI when no data is available
     */
    private void showNoDataAvailable() {
        tvDataType.setText(getString(R.string.no_historical_data));
        tvDataTimestamp.setText("");
        // Keep the UI showing zero values
    }

    // WebSocket callbacks
    @Override
    public void onPowerDataReceived(PowerData data) {
        // Check if activity is still valid
        if (isFinishing() || isDestroyed()) {
            return;
        }

        // Ensure timestamp is not null - set current time if needed
        if (data.getTimestamp() == null) {
            data.setTimestamp(new Date());
        }
        
        // Only switch to real-time mode when we actually receive real-time data
        isRealTimeData = true;
        uiHandler.post(() -> {
            // Double-check if activity is still valid before updating UI
            if (isFinishing() || isDestroyed()) {
                return;
            }
            updateUIWithPowerData(data, true);
        });
    }

    @Override
    public void onConnectionStatusChanged(boolean connected) {
        // Check if activity is still valid
        if (isFinishing() || isDestroyed()) {
            return;
        }

        uiHandler.post(() -> {
            // Double-check if activity is still valid before updating UI
            if (isFinishing() || isDestroyed()) {
                return;
            }

            tvConnectionStatus.setText(connected ? getString(R.string.connected) : getString(R.string.disconnected));
            tvConnectionStatus.setTextColor(getResources().getColor(
                    connected ? R.color.success : R.color.error, null));

            if (connected && currentCollectorId != null) {
                webSocketClient.subscribeToCollector(currentCollectorId);
                // Don't change data type display here - let it be updated when real data arrives
            } else if (!connected) {
                // Connection lost, switch to historical data mode
                isRealTimeData = false;
                if (lastKnownData != null) {
                    updateUIWithPowerData(lastKnownData, false);
                } else {
                    loadLastKnownData();
                }
            }
        });
    }

    @Override
    public void onError(String error) {
        // Check if activity is still valid
        if (isFinishing() || isDestroyed()) {
            return;
        }

        uiHandler.post(() -> {
            // Double-check if activity is still valid before updating UI
            if (isFinishing() || isDestroyed()) {
                return;
            }

            Log.e(TAG, "WebSocket error: " + error);
            Toast.makeText(this, getString(R.string.connection_error, error), Toast.LENGTH_SHORT).show();
        });
    }

    @Override
    public void onCollectorStatusChanged(String collectorId, boolean isOnline) {
        // Check if activity is still valid
        if (isFinishing() || isDestroyed()) {
            return;
        }

        uiHandler.post(() -> {
            // Double-check if activity is still valid before updating UI
            if (isFinishing() || isDestroyed()) {
                return;
            }

            Log.d(TAG, "Collector " + collectorId + " status changed to: " + (isOnline ? "online" : "offline"));
            // Update UI to show collector status if needed
            if (collectorId.equals(currentCollectorId)) {
                // Update current collector status indicator
                // You can add a status indicator in the UI for this
            }
        });
    }

    private void updateUIWithPowerData(PowerData data) {
        updateUIWithPowerData(data, isRealTimeData);
    }

    private void updateUIWithPowerData(PowerData data, boolean isRealTime) {
        // Update text displays
        tvVoltage.setText(String.format(Locale.getDefault(), "%.1f V", data.getVoltage()));
        tvCurrent.setText(String.format(Locale.getDefault(), "%.2f A", data.getCurrent()));
        tvPower.setText(String.format(Locale.getDefault(), "%.1f W", data.getPower()));
        tvEnergy.setText(String.format(Locale.getDefault(), "%.0f Wh", data.getEnergy())); // 不显示小数点

        // Update data type and timestamp display based on actual data being shown
        updateDataTypeDisplay(data, isRealTime);

        // Update chart only for real-time data to avoid confusing historical points
        if (isRealTime) {
            updateChartAsync(data);
        }
    }

    /**
     * Update chart asynchronously to prevent UI lag
     */
    private void updateChartAsync(PowerData data) {
        // Check if executorService is available and not shutdown
        if (executorService == null || executorService.isShutdown() || executorService.isTerminated()) {
            Log.w(TAG, "ExecutorService is not available, skipping chart update");
            return;
        }

        try {
            executorService.execute(() -> {
                // Check if activity is still valid
                if (isFinishing() || isDestroyed()) {
                    return;
                }

                // Process chart data in background thread
                int newIndex = powerDataList.size();
                
                // Get value based on current chart type
                float value = 0f;
                switch (currentChartType) {
                    case POWER:
                        value = (float) data.getPower();
                        break;
                    case VOLTAGE:
                        value = (float) data.getVoltage();
                        break;
                    case CURRENT:
                        value = (float) data.getCurrent();
                        break;
                    case ENERGY:
                        value = (float) data.getEnergy();
                        break;
                }
                
                Entry newEntry = new Entry(newIndex, value);
                
                // Create new lists to avoid concurrent modification
                List<Entry> newPowerEntries = new ArrayList<>(powerEntries);
                List<PowerData> newPowerDataList = new ArrayList<>(powerDataList);
                
                newPowerEntries.add(newEntry);
                newPowerDataList.add(data);

                // Keep only recent entries
                if (newPowerEntries.size() > CHART_MAX_ENTRIES) {
                    newPowerEntries.remove(0);
                    newPowerDataList.remove(0);
                    
                    // Adjust x values
                    for (int i = 0; i < newPowerEntries.size(); i++) {
                        newPowerEntries.get(i).setX(i);
                    }
                }

                // Update UI on main thread
                uiHandler.post(() -> {
                    // Double-check if activity is still valid before updating UI
                    if (isFinishing() || isDestroyed()) {
                        return;
                    }

                    powerEntries.clear();
                    powerEntries.addAll(newPowerEntries);
                    powerDataList.clear();
                    powerDataList.addAll(newPowerDataList);

                    powerDataSet.notifyDataSetChanged();
                    lineData.notifyDataChanged();
                    chartPower.notifyDataSetChanged();
                    chartPower.invalidate();

                    // Auto-scale chart
                    chartPower.fitScreen();
                });
            });
        } catch (Exception e) {
            Log.e(TAG, "Error executing chart update task", e);
        }
    }

    /**
     * Update data type display based on the actual data being shown
     */
    private void updateDataTypeDisplay(PowerData data, boolean isRealTime) {
        if (isRealTime) {
            tvDataType.setText(getString(R.string.realtime_data));
            tvDataTimestamp.setText("");
        } else {
            tvDataType.setText(getString(R.string.last_known_data));
            if (data != null && data.getTimestamp() != null) {
                SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault());
                String timestampStr = dateFormat.format(data.getTimestamp());
                tvDataTimestamp.setText(getString(R.string.data_timestamp, timestampStr));
            } else {
                tvDataTimestamp.setText(getString(R.string.no_timestamp_available));
            }
        }
    }

    // Navigation drawer menu handling
    @Override
    public boolean onNavigationItemSelected(@NonNull MenuItem item) {
        int id = item.getItemId();

        if (id == R.id.nav_dashboard) {
            // Already on dashboard
        } else if (id == R.id.nav_history) {
            Intent intent = new Intent(this, HistoryActivity.class);
            if (currentCollectorDbId != 0) {
                intent.putExtra("collectorId", String.valueOf(currentCollectorDbId));
                intent.putExtra("collectorUuid", currentCollectorId);
            }
            startActivity(intent);
        } else if (id == R.id.nav_analytics) {
            Intent intent = new Intent(this, AnalyticsActivity.class);
            if (currentCollectorDbId != 0) {
                intent.putExtra("collectorId", String.valueOf(currentCollectorDbId));
                intent.putExtra("collectorUuid", currentCollectorId);
            }
            startActivity(intent);
        } else if (id == R.id.nav_prediction) {
            Intent intent = new Intent(this, PredictionActivity.class);
            if (currentCollectorDbId != 0) {
                intent.putExtra("collectorId", String.valueOf(currentCollectorDbId));
                intent.putExtra("collectorUuid", currentCollectorId);
            }
            startActivity(intent);
        } else if (id == R.id.nav_settings) {
            Intent intent = new Intent(this, SettingsActivity.class);
            startActivity(intent);
        } else if (id == R.id.nav_logout) {
            logout();
        }

        drawerLayout.closeDrawer(GravityCompat.START);
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(@NonNull MenuItem item) {
        if (item.getItemId() == android.R.id.home) {
            drawerLayout.openDrawer(GravityCompat.START);
            return true;
        }
        return super.onOptionsItemSelected(item);
    }

    @Override
    public void onBackPressed() {
        if (drawerLayout.isDrawerOpen(GravityCompat.START)) {
            drawerLayout.closeDrawer(GravityCompat.START);
        } else {
            super.onBackPressed();
        }
    }

    private void logout() {
        // Clear all tokens (access token and refresh token)
        apiClient.clearTokens();
        
        // Disconnect WebSocket
        disconnectWebSocket();
        
        // Navigate to login
        navigateToLogin();
    }

    private void navigateToLogin() {
        Intent intent = new Intent(this, LoginActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
        startActivity(intent);
        finish();
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        disconnectWebSocket();
        
        // Clean up executor service
        if (executorService != null) {
            executorService.shutdown();
            try {
                // Wait a bit for tasks to complete
                if (!executorService.awaitTermination(2, java.util.concurrent.TimeUnit.SECONDS)) {
                    executorService.shutdownNow();
                }
            } catch (InterruptedException e) {
                // Force shutdown if interrupted
                executorService.shutdownNow();
                Thread.currentThread().interrupt();
            }
        }
    }

    @Override
    protected void onPause() {
        super.onPause();
        // Don't disconnect WebSocket on pause to avoid unnecessary reconnections
        // Only disconnect on destroy to maintain connection during brief pauses
    }

    @Override
    protected void onResume() {
        super.onResume();
        // Only connect if we don't already have a connection
        if (currentCollectorId != null && currentCollectorDbId != 0 && 
            (webSocketClient == null || !webSocketClient.isConnected())) {
            connectWebSocket();
        }
    }
}