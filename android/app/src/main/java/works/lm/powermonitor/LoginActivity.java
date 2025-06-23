package works.lm.powermonitor;

import android.content.Intent;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;

import com.google.android.material.button.MaterialButton;
import com.google.android.material.textfield.TextInputEditText;

import works.lm.powermonitor.network.ApiClient;
import works.lm.powermonitor.utils.LanguageUtils;
import works.lm.powermonitor.network.ApiService;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;

/**
 * Login activity for user authentication
 */
public class LoginActivity extends AppCompatActivity {
    
    private static final String TAG = "LoginActivity";
    
    private TextInputEditText etUsername, etPassword;
    private MaterialButton btnLogin;
    private View progressBar, tvError;
    
    private ApiClient apiClient;
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // Apply saved language setting
        LanguageUtils.applySavedLanguage(this);
        
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_login);
        
        Log.d(TAG, "LoginActivity onCreate started");
        
        // Initialize API client
        apiClient = ApiClient.getInstance(this);
        
        // Check if already logged in
        if (apiClient.isLoggedIn()) {
            Log.d(TAG, "User already logged in, navigating to main");
            navigateToMain();
            return;
        }
        
        Log.d(TAG, "User not logged in, showing login form");
        initViews();
        setupClickListeners();
    }
    
    private void initViews() {
        etUsername = findViewById(R.id.etUsername);
        etPassword = findViewById(R.id.etPassword);
        btnLogin = findViewById(R.id.btnLogin);
        progressBar = findViewById(R.id.progressBar);
        tvError = findViewById(R.id.tvError);
        
        Log.d(TAG, "Views initialized successfully");
    }
    
    private void setupClickListeners() {
        btnLogin.setOnClickListener(v -> performLogin());
        Log.d(TAG, "Click listeners set up");
    }
    
    private void performLogin() {
        String username = etUsername.getText().toString().trim();
        String password = etPassword.getText().toString().trim();
        
        Log.d(TAG, "Attempting login for user: " + username);
        
        // Validate input
        if (username.isEmpty()) {
            etUsername.setError(getString(R.string.please_enter_username));
            etUsername.requestFocus();
            Log.w(TAG, "Login failed: empty username");
            return;
        }
        
        if (password.isEmpty()) {
            etPassword.setError(getString(R.string.please_enter_password));
            etPassword.requestFocus();
            Log.w(TAG, "Login failed: empty password");
            return;
        }
        
        // Hide error message
        tvError.setVisibility(View.GONE);
        
        // Show loading
        setLoadingState(true);
        
        // Create login request
        ApiService.LoginRequest request = new ApiService.LoginRequest(username, password);
        
        // Make API call
        Call<ApiService.ApiResponse<ApiService.LoginData>> call = apiClient.getApiService().login(request);
        Log.d(TAG, "Login API call initiated");
        
        call.enqueue(new Callback<ApiService.ApiResponse<ApiService.LoginData>>() {
            @Override
            public void onResponse(Call<ApiService.ApiResponse<ApiService.LoginData>> call, Response<ApiService.ApiResponse<ApiService.LoginData>> response) {
                Log.d(TAG, "Login response received with code: " + response.code());
                
                runOnUiThread(() -> {
                    setLoadingState(false);
                    
                    if (response.isSuccessful() && response.body() != null) {
                        ApiService.ApiResponse<ApiService.LoginData> apiResponse = response.body();
                        
                        if (apiResponse.success && apiResponse.data != null) {
                            Log.d(TAG, "Login successful, tokens received");
                            
                            // Save both access token and refresh token
                            String accessToken = apiResponse.data.access_token;
                            String refreshToken = apiResponse.data.refresh_token;
                            
                            if (accessToken != null && !accessToken.isEmpty()) {
                                apiClient.saveToken(accessToken);
                                Log.d(TAG, "Access token saved");
                            } else {
                                Log.e(TAG, "Access token is null or empty");
                            }
                            
                            if (refreshToken != null && !refreshToken.isEmpty()) {
                                apiClient.saveRefreshToken(refreshToken);
                                Log.d(TAG, "Refresh token saved");
                            } else {
                                Log.w(TAG, "Refresh token is null or empty");
                            }
                            
                            // Verify token was saved
                            String savedToken = apiClient.getStoredToken();
                            if (savedToken != null && !savedToken.isEmpty()) {
                                Log.d(TAG, "Token verification successful, navigating to main");
                                
                                // Show success message
                                Toast.makeText(LoginActivity.this, getString(R.string.login_success), Toast.LENGTH_SHORT).show();
                                
                                // Navigate to main activity
                                navigateToMain();
                            } else {
                                Log.e(TAG, "Token verification failed");
                                showError("Token save failed");
                            }
                        } else {
                            Log.w(TAG, "Login failed: " + apiResponse.message);
                            String errorMessage = apiResponse.error != null ? apiResponse.error : apiResponse.message;
                            showError(errorMessage != null ? errorMessage : getString(R.string.login_failed));
                        }
                    } else {
                        Log.w(TAG, "Login failed with response code: " + response.code());
                        String errorMessage = getString(R.string.login_failed);
                        if (response.errorBody() != null) {
                            try {
                                String errorBodyString = response.errorBody().string();
                                Log.e(TAG, "Error body: " + errorBodyString);
                                // You can parse the error body for more specific error messages
                            } catch (Exception e) {
                                Log.e(TAG, "Failed to read error body", e);
                            }
                        }
                        showError(errorMessage);
                    }
                });
            }
            
            @Override
            public void onFailure(Call<ApiService.ApiResponse<ApiService.LoginData>> call, Throwable t) {
                Log.e(TAG, "Login request failed", t);
                
                runOnUiThread(() -> {
                    setLoadingState(false);
                    showError(getString(R.string.network_error, t.getMessage()));
                });
            }
        });
    }
    
    private void setLoadingState(boolean loading) {
        btnLogin.setEnabled(!loading);
        progressBar.setVisibility(loading ? View.VISIBLE : View.GONE);
        etUsername.setEnabled(!loading);
        etPassword.setEnabled(!loading);
        
        Log.d(TAG, "Loading state set to: " + loading);
    }
    
    private void showError(String message) {
        tvError.setVisibility(View.VISIBLE);
        ((android.widget.TextView) tvError).setText(message);
        Log.e(TAG, "Error displayed: " + message);
    }
    
    private void navigateToMain() {
        Log.d(TAG, "Navigating to MainActivity");
        try {
            Intent intent = new Intent(this, MainActivity.class);
            intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
            startActivity(intent);
            finish();
            Log.d(TAG, "MainActivity started successfully");
        } catch (Exception e) {
            Log.e(TAG, "Failed to navigate to MainActivity", e);
            showError("Navigation failed: " + e.getMessage());
        }
    }
} 