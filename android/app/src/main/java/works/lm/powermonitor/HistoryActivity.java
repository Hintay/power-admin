package works.lm.powermonitor;

import android.app.DatePickerDialog;
import android.content.Context;
import android.graphics.Canvas;
import android.os.AsyncTask;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.MenuItem;
import android.view.View;
import android.widget.CheckBox;
import android.widget.CompoundButton;
import android.widget.DatePicker;
import android.widget.TextView;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;
import androidx.core.content.ContextCompat;

import com.github.mikephil.charting.charts.LineChart;
import com.github.mikephil.charting.components.Description;
import com.github.mikephil.charting.components.Legend;
import com.github.mikephil.charting.components.MarkerView;
import com.github.mikephil.charting.components.XAxis;
import com.github.mikephil.charting.components.YAxis;
import com.github.mikephil.charting.data.Entry;
import com.github.mikephil.charting.data.LineData;
import com.github.mikephil.charting.data.LineDataSet;
import com.github.mikephil.charting.formatter.ValueFormatter;
import com.github.mikephil.charting.highlight.Highlight;
import com.github.mikephil.charting.interfaces.datasets.ILineDataSet;
import com.github.mikephil.charting.listener.ChartTouchListener;
import com.github.mikephil.charting.listener.OnChartGestureListener;
import com.github.mikephil.charting.utils.MPPointF;
import com.google.android.material.appbar.MaterialToolbar;
import com.google.android.material.button.MaterialButton;
import com.google.android.material.card.MaterialCardView;
import com.google.android.material.checkbox.MaterialCheckBox;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collections;
import java.util.Comparator;
import java.util.Date;
import java.util.List;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;
import works.lm.powermonitor.model.PowerData;
import works.lm.powermonitor.network.ApiClient;
import works.lm.powermonitor.network.ApiService;
import works.lm.powermonitor.utils.LanguageUtils;

/**
 * Activity for viewing historical power data with separate charts for each metric
 * Optimized for better performance and smooth scrolling
 * Features: Custom markers for data details and synchronized chart scrolling
 */
public class HistoryActivity extends AppCompatActivity implements OnChartGestureListener {
    
    private static final String TAG = "HistoryActivity";
    
    // Performance optimization constants
    private static final int MAX_DATA_POINTS = 500; // Reduced from 1000 for better performance
    private static final int CHART_ANIMATION_DURATION = 800; // Milliseconds
    private static final long UPDATE_DEBOUNCE_DELAY = 300; // Milliseconds
    
    // UI Components
    private LineChart chartPower, chartVoltage, chartCurrent, chartFrequency, chartEnergy;
    private MaterialCardView cardPowerChart, cardVoltageChart, cardCurrentChart, cardFrequencyChart, cardEnergyChart, cardEnergyStats;
    private MaterialButton btnStartDate, btnEndDate, btnLoadData;
    private MaterialButton btn24Hours, btn7Days, btn4Weeks, btn12Months;
    private MaterialCheckBox cbShowPower, cbShowVoltage, cbShowCurrent, cbShowFrequency, cbShowEnergy;
    private TextView tvStartEnergy, tvEndEnergy, tvEnergyConsumed;
    
    // Data and API
    private ApiClient apiClient;
    private String collectorId;
    private Date startDate, endDate;
    private SimpleDateFormat dateFormat;
    private List<PowerData> currentDataList;
    
    // Performance optimization
    private ExecutorService executorService;
    private Handler mainHandler;
    private Runnable updateChartsRunnable;
    
    // Chart synchronization
    private boolean isChartSyncEnabled = true;
    private LineChart currentSyncChart = null;
    private List<LineChart> allCharts = new ArrayList<>();
    
    // Chart configuration
    private static final int COLOR_POWER = 0xFFE91E63;     // Pink
    private static final int COLOR_VOLTAGE = 0xFF2196F3;   // Blue
    private static final int COLOR_CURRENT = 0xFF4CAF50;   // Green
    private static final int COLOR_FREQUENCY = 0xFFFF9800; // Orange
    private static final int COLOR_ENERGY = 0xFF9C27B0;    // Purple

    /**
     * Custom MarkerView to display detailed data when touching chart
     */
    public class CustomMarkerView extends MarkerView {
        
        private TextView tvTimestamp, tvValue, tvPower, tvVoltage, tvCurrent, tvFrequency, tvEnergy;
        private PowerDataType currentDataType;
        
        public CustomMarkerView(Context context, int layoutResource, PowerDataType dataType) {
            super(context, layoutResource);
            this.currentDataType = dataType;
            
            tvTimestamp = findViewById(R.id.tvMarkerTimestamp);
            tvValue = findViewById(R.id.tvMarkerValue);
            tvPower = findViewById(R.id.tvMarkerPower);
            tvVoltage = findViewById(R.id.tvMarkerVoltage);
            tvCurrent = findViewById(R.id.tvMarkerCurrent);
            tvFrequency = findViewById(R.id.tvMarkerFrequency);
            tvEnergy = findViewById(R.id.tvMarkerEnergy);
        }
        
        @Override
        public void refreshContent(Entry e, Highlight highlight) {
            if (currentDataList == null || e.getX() < 0 || e.getX() >= currentDataList.size()) {
                return;
            }
            
            PowerData data = currentDataList.get((int) e.getX());
            SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault());
            
            // Display timestamp
            tvTimestamp.setText(formatter.format(data.getTimestamp()));
            
            // Display current metric value with emphasis
            String currentValue = "";
            switch (currentDataType) {
                case POWER:
                    currentValue = String.format("%.2f W", data.getPower());
                    break;
                case VOLTAGE:
                    currentValue = String.format("%.2f V", data.getVoltage());
                    break;
                case CURRENT:
                    currentValue = String.format("%.3f A", data.getCurrent());
                    break;
                case FREQUENCY:
                    currentValue = String.format("%.2f Hz", data.getFrequency());
                    break;
                case ENERGY:
                    currentValue = String.format("%.3f Wh", data.getEnergy());
                    break;
            }
            tvValue.setText(currentValue);
            
            // Display all metrics for reference
            tvPower.setText(String.format(getContext().getString(R.string.marker_power_label), data.getPower()));
            tvVoltage.setText(String.format(getContext().getString(R.string.marker_voltage_label), data.getVoltage()));
            tvCurrent.setText(String.format(getContext().getString(R.string.marker_current_label), data.getCurrent()));
            tvFrequency.setText(String.format(getContext().getString(R.string.marker_frequency_label), data.getFrequency()));
            tvEnergy.setText(String.format(getContext().getString(R.string.marker_energy_label), data.getEnergy()));
            
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
        setContentView(R.layout.activity_history);
        
        // Initialize performance optimization components
        executorService = Executors.newSingleThreadExecutor();
        mainHandler = new Handler(Looper.getMainLooper());
        
        // Initialize API client
        apiClient = ApiClient.getInstance(this);
        
        // Get collector ID from intent
        collectorId = getIntent().getStringExtra("collectorId");
        
        // Initialize date format
        dateFormat = new SimpleDateFormat("yyyy-MM-dd", Locale.getDefault());
        
        // Set default date range (last 7 days)
        Calendar calendar = Calendar.getInstance();
        endDate = calendar.getTime();
        calendar.add(Calendar.DAY_OF_MONTH, -7);
        startDate = calendar.getTime();
        
        initViews();
        setupToolbar();
        setupCharts();
        setupClickListeners();
        updateDateButtons();
        
        // Initialize chart synchronization
        initChartSync();
    }
    
    private void initChartSync() {
        allCharts.clear();
        allCharts.add(chartPower);
        allCharts.add(chartVoltage);
        allCharts.add(chartCurrent);
        allCharts.add(chartFrequency);
        allCharts.add(chartEnergy);
        
        // Set gesture listeners for all charts
        for (int i = 0; i < allCharts.size(); i++) {
            LineChart chart = allCharts.get(i);
            chart.setOnChartGestureListener(this);
            // Tag charts with their index for easier identification
            chart.setTag("chart_" + i);
        }
    }
    
    @Override
    protected void onDestroy() {
        super.onDestroy();
        // Clean up resources to prevent memory leaks
        if (executorService != null && !executorService.isShutdown()) {
            executorService.shutdown();
        }
        if (updateChartsRunnable != null) {
            mainHandler.removeCallbacks(updateChartsRunnable);
        }
        
        // Clear chart data to free memory
        clearAllCharts();
    }
    
    private void clearAllCharts() {
        if (chartPower != null) chartPower.clear();
        if (chartVoltage != null) chartVoltage.clear();
        if (chartCurrent != null) chartCurrent.clear();
        if (chartFrequency != null) chartFrequency.clear();
        if (chartEnergy != null) chartEnergy.clear();
    }
    
    private void initViews() {
        // Charts
        chartPower = findViewById(R.id.chartPower);
        chartVoltage = findViewById(R.id.chartVoltage);
        chartCurrent = findViewById(R.id.chartCurrent);
        chartFrequency = findViewById(R.id.chartFrequency);
        chartEnergy = findViewById(R.id.chartEnergy);
        
        // Chart cards
        cardPowerChart = findViewById(R.id.cardPowerChart);
        cardVoltageChart = findViewById(R.id.cardVoltageChart);
        cardCurrentChart = findViewById(R.id.cardCurrentChart);
        cardFrequencyChart = findViewById(R.id.cardFrequencyChart);
        cardEnergyChart = findViewById(R.id.cardEnergyChart);
        cardEnergyStats = findViewById(R.id.cardEnergyStats);
        
        // Buttons
        btnStartDate = findViewById(R.id.btnStartDate);
        btnEndDate = findViewById(R.id.btnEndDate);
        btnLoadData = findViewById(R.id.btnLoadData);
        
        // Quick time range buttons
        btn24Hours = findViewById(R.id.btn24Hours);
        btn7Days = findViewById(R.id.btn7Days);
        btn4Weeks = findViewById(R.id.btn4Weeks);
        btn12Months = findViewById(R.id.btn12Months);
        
        // Chart display checkboxes
        cbShowPower = findViewById(R.id.cbShowPower);
        cbShowVoltage = findViewById(R.id.cbShowVoltage);
        cbShowCurrent = findViewById(R.id.cbShowCurrent);
        cbShowFrequency = findViewById(R.id.cbShowFrequency);
        cbShowEnergy = findViewById(R.id.cbShowEnergy);
        
        // Energy stats text views
        tvStartEnergy = findViewById(R.id.tvStartEnergy);
        tvEndEnergy = findViewById(R.id.tvEndEnergy);
        tvEnergyConsumed = findViewById(R.id.tvEnergyConsumed);
    }
    
    private void setupToolbar() {
        MaterialToolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        
        if (getSupportActionBar() != null) {
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
            getSupportActionBar().setTitle(getString(R.string.historical_data_title));
        }
    }
    
    private void setupCharts() {
        setupSingleChart(chartPower, COLOR_POWER);
        setupSingleChart(chartVoltage, COLOR_VOLTAGE);
        setupSingleChart(chartCurrent, COLOR_CURRENT);
        setupSingleChart(chartFrequency, COLOR_FREQUENCY);
        setupSingleChart(chartEnergy, COLOR_ENERGY);
    }
    
    private void setupSingleChart(LineChart chart, int color) {
        // Configure chart for better performance
        Description description = new Description();
        description.setText("");
        chart.setDescription(description);
        chart.setTouchEnabled(true);
        chart.setDragEnabled(true);
        chart.setScaleEnabled(true);
        chart.setPinchZoom(true);
        chart.setDrawGridBackground(false);
        
        // Performance optimization settings
        chart.setDrawBorders(false);
        chart.setHardwareAccelerationEnabled(true); // Enable hardware acceleration
        chart.setDrawMarkers(true); // Enable markers for data details
        chart.setAutoScaleMinMaxEnabled(true);
        
        // Enable highlighting for touch interaction
        chart.setHighlightPerTapEnabled(true);
        chart.setHighlightPerDragEnabled(true);
        chart.setMaxHighlightDistance(300f); // Increase touch radius for easier interaction
        
        // Configure X axis
        XAxis xAxis = chart.getXAxis();
        xAxis.setPosition(XAxis.XAxisPosition.BOTTOM);
        xAxis.setDrawGridLines(true);
        xAxis.setGridColor(ContextCompat.getColor(this, R.color.chart_grid));
        xAxis.setGranularity(1f);
        xAxis.setLabelCount(6); // Limit label count for better performance
        xAxis.setAvoidFirstLastClipping(true);
        xAxis.setValueFormatter(new ValueFormatter() {
            @Override
            public String getFormattedValue(float value) {
                if (currentDataList == null || currentDataList.isEmpty() || value < 0 || value >= currentDataList.size()) {
                    return "";
                }
                PowerData data = currentDataList.get((int) value);
                SimpleDateFormat formatter = new SimpleDateFormat("MM-dd HH:mm", Locale.getDefault());
                return formatter.format(data.getTimestamp());
            }
        });
        
        // Configure Y axis
        YAxis leftAxis = chart.getAxisLeft();
        leftAxis.setDrawGridLines(true);
        leftAxis.setGridColor(ContextCompat.getColor(this, R.color.chart_grid));
        leftAxis.setAxisMinimum(0f);
        leftAxis.setTextColor(ContextCompat.getColor(this, R.color.text_primary));
        leftAxis.setLabelCount(5); // Limit label count
        
        YAxis rightAxis = chart.getAxisRight();
        rightAxis.setEnabled(false);
        
        // Configure legend
        Legend legend = chart.getLegend();
        legend.setEnabled(false); // Disable legend for individual charts
        
        // Set viewport to show reasonable amount of data
        chart.setVisibleXRangeMaximum(100); // Show max 100 data points at once
    }
    
    private void setupClickListeners() {
        // Custom date selection
        btnStartDate.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                showDatePicker(true);
            }
        });
        btnEndDate.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                showDatePicker(false);
            }
        });
        btnLoadData.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                loadHistoryData();
            }
        });
        
        // Quick time range buttons
        btn24Hours.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                setQuickTimeRange(Calendar.HOUR_OF_DAY, -24);
            }
        });
        btn7Days.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                setQuickTimeRange(Calendar.DAY_OF_MONTH, -7);
            }
        });
        btn4Weeks.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                setQuickTimeRange(Calendar.WEEK_OF_YEAR, -4);
            }
        });
        btn12Months.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                setQuickTimeRange(Calendar.MONTH, -12);
            }
        });
        
        // Chart display checkboxes
        cbShowPower.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                updateChartsVisibility();
            }
        });
        cbShowVoltage.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                updateChartsVisibility();
            }
        });
        cbShowCurrent.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                updateChartsVisibility();
            }
        });
        cbShowFrequency.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                updateChartsVisibility();
            }
        });
        cbShowEnergy.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                updateChartsVisibility();
            }
        });
    }
    
    private void setQuickTimeRange(int field, int amount) {
        Calendar calendar = Calendar.getInstance();
        endDate = calendar.getTime();
        calendar.add(field, amount);
        startDate = calendar.getTime();
        
        updateDateButtons();
        resetQuickButtonsBackground();
        
        // Highlight selected button
        MaterialButton selectedButton = null;
        if (field == Calendar.HOUR_OF_DAY && amount == -24) selectedButton = btn24Hours;
        else if (field == Calendar.DAY_OF_MONTH && amount == -7) selectedButton = btn7Days;
        else if (field == Calendar.WEEK_OF_YEAR && amount == -4) selectedButton = btn4Weeks;
        else if (field == Calendar.MONTH && amount == -12) selectedButton = btn12Months;
        
        if (selectedButton != null) {
            selectedButton.setBackgroundColor(ContextCompat.getColor(this, R.color.primary_light));
        }
        
        // Auto load data
        loadHistoryData();
    }
    
    private void resetQuickButtonsBackground() {
        int defaultColor = ContextCompat.getColor(this, android.R.color.transparent);
        btn24Hours.setBackgroundColor(defaultColor);
        btn7Days.setBackgroundColor(defaultColor);
        btn4Weeks.setBackgroundColor(defaultColor);
        btn12Months.setBackgroundColor(defaultColor);
    }
    
    private void showDatePicker(boolean isStartDate) {
        Calendar calendar = Calendar.getInstance();
        calendar.setTime(isStartDate ? startDate : endDate);
        
        DatePickerDialog datePickerDialog = new DatePickerDialog(
                this,
                new DatePickerDialog.OnDateSetListener() {
                    @Override
                    public void onDateSet(DatePicker view, int year, int month, int dayOfMonth) {
                        Calendar selectedCalendar = Calendar.getInstance();
                        selectedCalendar.set(year, month, dayOfMonth);
                        
                        if (isStartDate) {
                            startDate = selectedCalendar.getTime();
                        } else {
                            endDate = selectedCalendar.getTime();
                        }
                        
                        updateDateButtons();
                        resetQuickButtonsBackground(); // Reset when custom date is selected
                    }
                },
                calendar.get(Calendar.YEAR),
                calendar.get(Calendar.MONTH),
                calendar.get(Calendar.DAY_OF_MONTH)
        );
        
        datePickerDialog.show();
    }
    
    private void updateDateButtons() {
        btnStartDate.setText(getString(R.string.start_date_label, dateFormat.format(startDate)));
        btnEndDate.setText(getString(R.string.end_date_label, dateFormat.format(endDate)));
    }
    
    private void updateChartsVisibility() {
        // Use debounced update to prevent rapid UI changes
        if (updateChartsRunnable != null) {
            mainHandler.removeCallbacks(updateChartsRunnable);
        }
        
        updateChartsRunnable = new Runnable() {
            @Override
            public void run() {
                cardPowerChart.setVisibility(cbShowPower.isChecked() ? View.VISIBLE : View.GONE);
                cardVoltageChart.setVisibility(cbShowVoltage.isChecked() ? View.VISIBLE : View.GONE);
                cardCurrentChart.setVisibility(cbShowCurrent.isChecked() ? View.VISIBLE : View.GONE);
                cardFrequencyChart.setVisibility(cbShowFrequency.isChecked() ? View.VISIBLE : View.GONE);
                cardEnergyChart.setVisibility(cbShowEnergy.isChecked() ? View.VISIBLE : View.GONE);
                
                // Update charts if data is available
                if (currentDataList != null && !currentDataList.isEmpty()) {
                    updateVisibleChartsAsync();
                }
            }
        };
        
        mainHandler.postDelayed(updateChartsRunnable, UPDATE_DEBOUNCE_DELAY);
    }
    
    private void loadHistoryData() {
        if (collectorId == null) {
            Toast.makeText(this, getString(R.string.no_collector_selected), Toast.LENGTH_SHORT).show();
            return;
        }
        
        // Set loading state
        btnLoadData.setEnabled(false);
        btnLoadData.setText(getString(R.string.loading));
        
        // Format dates for API
        String startDateStr = dateFormat.format(startDate) + "T00:00:00Z";
        String endDateStr = dateFormat.format(endDate) + "T23:59:59Z";
        
        Call<ApiService.ApiResponse<List<PowerData>>> call = apiClient.getApiService().getHistoryData(
                "Bearer " + apiClient.getStoredToken(),
                collectorId,
                startDateStr,
                endDateStr,
                MAX_DATA_POINTS // Use optimized limit
        );
        
        call.enqueue(new Callback<ApiService.ApiResponse<List<PowerData>>>() {
            @Override
            public void onResponse(Call<ApiService.ApiResponse<List<PowerData>>> call, Response<ApiService.ApiResponse<List<PowerData>>> response) {
                btnLoadData.setEnabled(true);
                btnLoadData.setText(getString(R.string.load_data));
                
                if (response.isSuccessful() && response.body() != null) {
                    ApiService.ApiResponse<List<PowerData>> apiResponse = response.body();
                    
                    if (apiResponse.success && apiResponse.data != null) {
                        // Process data asynchronously to avoid blocking UI
                        processDataAsync(apiResponse.data);
                        Toast.makeText(HistoryActivity.this, getString(R.string.loaded_records, apiResponse.data.size()), Toast.LENGTH_SHORT).show();
                    } else {
                        String errorMessage = apiResponse.error != null ? apiResponse.error : getString(R.string.load_history_failed);
                        Toast.makeText(HistoryActivity.this, errorMessage, Toast.LENGTH_SHORT).show();
                    }
                } else {
                    Toast.makeText(HistoryActivity.this, getString(R.string.load_history_failed), Toast.LENGTH_SHORT).show();
                }
            }
            
            @Override
            public void onFailure(Call<ApiService.ApiResponse<List<PowerData>>> call, Throwable t) {
                btnLoadData.setEnabled(true);
                btnLoadData.setText(getString(R.string.load_data));
                Log.e(TAG, "Error loading history data", t);
                Toast.makeText(HistoryActivity.this, getString(R.string.network_error, t.getMessage()), Toast.LENGTH_SHORT).show();
            }
        });
    }
    
    /**
     * Process data asynchronously to avoid blocking UI thread
     */
    private void processDataAsync(List<PowerData> rawData) {
        executorService.execute(new Runnable() {
            @Override
            public void run() {
                try {
                    // Sort and process data in background thread
                    List<PowerData> processedData = optimizeDataForCharts(rawData);
                    
                    // Update UI on main thread
                    mainHandler.post(new Runnable() {
                        @Override
                        public void run() {
                            currentDataList = processedData;
                            updateAllChartsAsync();
                        }
                    });
                } catch (Exception e) {
                    Log.e(TAG, "Error processing data", e);
                    mainHandler.post(new Runnable() {
                        @Override
                        public void run() {
                            Toast.makeText(HistoryActivity.this, getString(R.string.data_processing_error), Toast.LENGTH_SHORT).show();
                        }
                    });
                }
            }
        });
    }
    
    /**
     * Optimize data for better chart performance
     */
    private List<PowerData> optimizeDataForCharts(List<PowerData> rawData) {
        if (rawData == null || rawData.isEmpty()) {
            return new ArrayList<>();
        }
        
        // Sort data by timestamp (more efficient than Collections.sort for large datasets)
        List<PowerData> sortedData = new ArrayList<>(rawData);
        Collections.sort(sortedData, new Comparator<PowerData>() {
            @Override
            public int compare(PowerData d1, PowerData d2) {
                return d1.getTimestamp().compareTo(d2.getTimestamp());
            }
        });
        
        // If data is too large, sample it intelligently
        if (sortedData.size() > MAX_DATA_POINTS) {
            return sampleData(sortedData, MAX_DATA_POINTS);
        }
        
        return sortedData;
    }
    
    /**
     * Intelligent data sampling to maintain chart quality while reducing data points
     */
    private List<PowerData> sampleData(List<PowerData> data, int targetSize) {
        if (data.size() <= targetSize) {
            return data;
        }
        
        List<PowerData> sampledData = new ArrayList<>();
        double step = (double) data.size() / targetSize;
        
        for (int i = 0; i < targetSize; i++) {
            int index = (int) (i * step);
            if (index < data.size()) {
                sampledData.add(data.get(index));
            }
        }
        
        // Always include the last data point
        if (!sampledData.isEmpty() && !sampledData.get(sampledData.size() - 1).equals(data.get(data.size() - 1))) {
            sampledData.add(data.get(data.size() - 1));
        }
        
        return sampledData;
    }
    
    private void updateAllChartsAsync() {
        if (currentDataList == null || currentDataList.isEmpty()) {
            mainHandler.post(new Runnable() {
                @Override
                public void run() {
                    Toast.makeText(HistoryActivity.this, getString(R.string.no_data_in_range), Toast.LENGTH_SHORT).show();
                }
            });
            return;
        }
        
        // Calculate energy statistics on main thread (quick operation)
        calculateEnergyStatistics();
        
        // Update charts asynchronously
        updateVisibleChartsAsync();
    }
    
    private void updateVisibleChartsAsync() {
        executorService.execute(new Runnable() {
            @Override
            public void run() {
                try {
                    // Prepare chart data in background
                    final List<ChartUpdateData> chartUpdates = new ArrayList<>();
                    
                    if (cbShowPower.isChecked()) {
                        chartUpdates.add(new ChartUpdateData(chartPower, PowerDataType.POWER, COLOR_POWER));
                    }
                    if (cbShowVoltage.isChecked()) {
                        chartUpdates.add(new ChartUpdateData(chartVoltage, PowerDataType.VOLTAGE, COLOR_VOLTAGE));
                    }
                    if (cbShowCurrent.isChecked()) {
                        chartUpdates.add(new ChartUpdateData(chartCurrent, PowerDataType.CURRENT, COLOR_CURRENT));
                    }
                    if (cbShowFrequency.isChecked()) {
                        chartUpdates.add(new ChartUpdateData(chartFrequency, PowerDataType.FREQUENCY, COLOR_FREQUENCY));
                    }
                    if (cbShowEnergy.isChecked()) {
                        chartUpdates.add(new ChartUpdateData(chartEnergy, PowerDataType.ENERGY, COLOR_ENERGY));
                    }
                    
                    // Update charts on main thread
                    mainHandler.post(new Runnable() {
                        @Override
                        public void run() {
                            for (ChartUpdateData update : chartUpdates) {
                                updateChartOptimized(update.chart, update.dataType, update.color);
                            }
                        }
                    });
                    
                } catch (Exception e) {
                    Log.e(TAG, "Error updating charts", e);
                }
            }
        });
    }
    
    // Helper class for chart update data
    private static class ChartUpdateData {
        final LineChart chart;
        final PowerDataType dataType;
        final int color;
        
        ChartUpdateData(LineChart chart, PowerDataType dataType, int color) {
            this.chart = chart;
            this.dataType = dataType;
            this.color = color;
        }
    }
    
    private void calculateEnergyStatistics() {
        if (currentDataList == null || currentDataList.isEmpty()) {
            return;
        }
        
        // Sort data by timestamp to ensure correct order (compatible with older Android versions)
        Collections.sort(currentDataList, new Comparator<PowerData>() {
            @Override
            public int compare(PowerData d1, PowerData d2) {
                return d1.getTimestamp().compareTo(d2.getTimestamp());
            }
        });
        
        // Get first and last energy readings
        PowerData firstData = currentDataList.get(0);
        PowerData lastData = currentDataList.get(currentDataList.size() - 1);
        
        double startEnergy = firstData.getEnergy();
        double endEnergy = lastData.getEnergy();
        double consumedEnergy = Math.max(0, endEnergy - startEnergy); // Ensure non-negative
        
        // Update UI with formatted values
        tvStartEnergy.setText(getString(R.string.energy_unit, startEnergy));
        tvEndEnergy.setText(getString(R.string.energy_unit, endEnergy));
        tvEnergyConsumed.setText(getString(R.string.energy_unit, consumedEnergy));
        
        // Show energy statistics card when data is available
        cardEnergyStats.setVisibility(View.VISIBLE);
        
        // Log energy consumption for debugging
        Log.d(TAG, String.format("Energy consumption: %.3f Wh (from %.3f to %.3f)", 
                consumedEnergy, startEnergy, endEnergy));
    }
    
    // Enum for different data types
    private enum PowerDataType {
        POWER, VOLTAGE, CURRENT, FREQUENCY, ENERGY
    }
    
    private void updateChartOptimized(LineChart chart, PowerDataType dataType, int color) {
        if (currentDataList == null || currentDataList.isEmpty()) {
            return;
        }
        
        ArrayList<Entry> entries = new ArrayList<>();
        String label = "";
        
        // Pre-allocate list size for better performance
        entries.ensureCapacity(currentDataList.size());
        
        for (int i = 0; i < currentDataList.size(); i++) {
            PowerData data = currentDataList.get(i);
            float value = 0f;
            
            switch (dataType) {
                case POWER:
                    value = (float) data.getPower();
                    label = getString(R.string.power_w_unit);
                    break;
                case VOLTAGE:
                    value = (float) data.getVoltage();
                    label = getString(R.string.voltage_v_unit);
                    break;
                case CURRENT:
                    value = (float) data.getCurrent();
                    label = getString(R.string.current_a_unit);
                    break;
                case FREQUENCY:
                    value = (float) data.getFrequency();
                    label = getString(R.string.frequency_hz_unit);
                    break;
                case ENERGY:
                    value = (float) data.getEnergy();
                    label = getString(R.string.energy_kwh_unit);
                    break;
            }
            
            entries.add(new Entry(i, value));
        }
        
        // Create optimized data set
        LineDataSet dataSet = new LineDataSet(entries, label);
        dataSet.setColor(color);
        dataSet.setCircleColor(color);
        dataSet.setLineWidth(2f); // Slightly thinner for better performance
        dataSet.setCircleRadius(2f); // Smaller circles for better performance
        dataSet.setDrawValues(false);
        dataSet.setDrawFilled(false); // Disable fill for better performance on large datasets
        dataSet.setDrawCircles(currentDataList.size() <= 50); // Only show circles for small datasets
        dataSet.setMode(LineDataSet.Mode.LINEAR); // Use linear mode for better performance
        
        // Enable highlighting for touch interaction
        dataSet.setHighlightEnabled(true);
        dataSet.setHighlightLineWidth(1f);
        dataSet.enableDashedHighlightLine(10f, 5f, 0f);
        
        // Create line data and update chart with animation
        LineData lineData = new LineData(dataSet);
        chart.setData(lineData);
        
        // Set custom marker for this chart
        CustomMarkerView marker = new CustomMarkerView(this, R.layout.custom_marker_view, dataType);
        marker.setChartView(chart);
        chart.setMarker(marker);
        
        chart.animateY(CHART_ANIMATION_DURATION); // Smooth animation
        chart.notifyDataSetChanged();
        chart.invalidate();
    }
    
    // Chart gesture listener implementations for synchronization
    @Override
    public void onChartGestureStart(android.view.MotionEvent me, ChartTouchListener.ChartGesture lastPerformedGesture) {
        // The chart that triggered this event will be set as the sync source
        currentSyncChart = findChartFromMotionEvent(me);
    }
    
    /**
     * Find which chart triggered the motion event
     */
    private LineChart findChartFromMotionEvent(android.view.MotionEvent me) {
        float x = me.getX();
        float y = me.getY();
        
        // Check each chart to see if the touch point is within its bounds
        for (LineChart chart : allCharts) {
            if (chart.getVisibility() == View.VISIBLE && chart.getData() != null) {
                // Get chart's position relative to its parent
                int[] location = new int[2];
                chart.getLocationInWindow(location);
                
                // Convert to raw coordinates
                float rawX = me.getRawX();
                float rawY = me.getRawY();
                
                if (rawX >= location[0] && rawX <= location[0] + chart.getWidth() &&
                    rawY >= location[1] && rawY <= location[1] + chart.getHeight()) {
                    return chart;
                }
            }
        }
        
        // Fallback to first visible chart with data
        for (LineChart chart : allCharts) {
            if (chart.getVisibility() == View.VISIBLE && chart.getData() != null) {
                return chart;
            }
        }
        
        return null;
    }
    
    @Override
    public void onChartGestureEnd(android.view.MotionEvent me, ChartTouchListener.ChartGesture lastPerformedGesture) {
        currentSyncChart = null;
    }
    
    @Override
    public void onChartLongPressed(android.view.MotionEvent me) {
        // Handle long press if needed
    }
    
    @Override
    public void onChartDoubleTapped(android.view.MotionEvent me) {
        // Reset zoom on double tap
        if (currentSyncChart != null) {
            syncChartAction(new ChartAction() {
                @Override
                public void execute(LineChart chart) {
                    chart.fitScreen();
                }
            });
        }
    }
    
    @Override
    public void onChartSingleTapped(android.view.MotionEvent me) {
        // Handle single tap if needed
    }
    
    @Override
    public void onChartFling(android.view.MotionEvent me1, android.view.MotionEvent me2, float velocityX, float velocityY) {
        // Handle fling for momentum scrolling sync
        if (currentSyncChart != null && isChartSyncEnabled) {
            syncChartViewPort();
        }
    }
    
    @Override
    public void onChartScale(android.view.MotionEvent me, float scaleX, float scaleY) {
        // Sync zoom level across charts
        if (currentSyncChart != null && isChartSyncEnabled) {
            syncChartViewPort();
        }
    }
    
    @Override
    public void onChartTranslate(android.view.MotionEvent me, float dX, float dY) {
        // Sync pan/scroll across charts
        if (currentSyncChart != null && isChartSyncEnabled) {
            syncChartViewPort();
        }
    }
    
    /**
     * Synchronize viewport across all visible charts
     */
    private void syncChartViewPort() {
        if (currentSyncChart == null || !isChartSyncEnabled) {
            return;
        }
        
        try {
            // Get current viewport state
            final float lowestVisibleX = currentSyncChart.getLowestVisibleX();
            final float highestVisibleX = currentSyncChart.getHighestVisibleX();
            
            // Temporarily disable chart sync to prevent recursive calls
            isChartSyncEnabled = false;
            
            syncChartAction(new ChartAction() {
                @Override
                public void execute(LineChart chart) {
                    if (!chart.equals(currentSyncChart) && chart.getVisibility() == View.VISIBLE && chart.getData() != null) {
                        try {
                            // Simple sync by setting visible X range
                            chart.setVisibleXRangeMinimum(1f);
                            chart.setVisibleXRangeMaximum(highestVisibleX - lowestVisibleX + 10f);
                            chart.moveViewToX((lowestVisibleX + highestVisibleX) / 2f);
                            chart.invalidate();
                        } catch (Exception e) {
                            Log.w(TAG, "Error syncing chart: " + e.getMessage());
                        }
                    }
                }
            });
            
        } finally {
            // Re-enable chart sync after a short delay
            mainHandler.postDelayed(new Runnable() {
                @Override
                public void run() {
                    isChartSyncEnabled = true;
                }
            }, 100);
        }
    }
    
    /**
     * Execute action on all charts
     */
    private void syncChartAction(ChartAction action) {
        for (LineChart chart : allCharts) {
            if (chart != null && chart.getData() != null) {
                action.execute(chart);
            }
        }
    }
    
    /**
     * Interface for chart actions
     */
    private interface ChartAction {
        void execute(LineChart chart);
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