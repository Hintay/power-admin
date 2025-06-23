package works.lm.powermonitor;

import android.os.Bundle;
import android.view.MenuItem;

import androidx.appcompat.app.AppCompatActivity;

import works.lm.powermonitor.utils.LanguageUtils;

import com.google.android.material.appbar.MaterialToolbar;

/**
 * Activity for app settings (placeholder)
 */
public class SettingsActivity extends AppCompatActivity {
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // Apply saved language setting
        LanguageUtils.applySavedLanguage(this);
        
        super.onCreate(savedInstanceState);
        
        // Simple layout for now
        setContentView(R.layout.activity_placeholder);
        
        MaterialToolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        
        if (getSupportActionBar() != null) {
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
            getSupportActionBar().setTitle(getString(R.string.settings));
        }
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