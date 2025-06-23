package works.lm.powermonitor.utils;

import android.content.Context;
import android.content.SharedPreferences;
import android.content.res.Configuration;
import android.content.res.Resources;
import android.os.Build;

import java.util.Locale;

/**
 * Language utility class for managing app language settings
 */
public class LanguageUtils {
    private static final String PREF_NAME = "language_prefs";
    private static final String LANGUAGE_KEY = "selected_language";
    
    // Supported languages
    public static final String ENGLISH = "en";
    public static final String CHINESE = "zh";
    public static final String JAPANESE = "ja";
    
    /**
     * Set app language
     */
    public static void setAppLanguage(Context context, String languageCode) {
        saveLanguagePreference(context, languageCode);
        updateConfiguration(context, languageCode);
    }
    
    /**
     * Get saved language preference
     */
    public static String getSavedLanguage(Context context) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        return prefs.getString(LANGUAGE_KEY, getSystemLanguage());
    }
    
    /**
     * Save language preference
     */
    private static void saveLanguagePreference(Context context, String languageCode) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        prefs.edit().putString(LANGUAGE_KEY, languageCode).apply();
    }
    
    /**
     * Update app configuration with selected language
     */
    public static void updateConfiguration(Context context, String languageCode) {
        Locale locale = new Locale(languageCode);
        Locale.setDefault(locale);
        
        Resources resources = context.getResources();
        Configuration configuration = new Configuration(resources.getConfiguration());
        
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            configuration.setLocale(locale);
        } else {
            configuration.locale = locale;
        }
        
        resources.updateConfiguration(configuration, resources.getDisplayMetrics());
    }
    
    /**
     * Apply saved language on app start
     */
    public static void applySavedLanguage(Context context) {
        String savedLanguage = getSavedLanguage(context);
        updateConfiguration(context, savedLanguage);
    }
    
    /**
     * Get system default language
     */
    private static String getSystemLanguage() {
        String systemLang = Locale.getDefault().getLanguage();
        
        // Map system language to supported languages
        switch (systemLang) {
            case "zh":
                return CHINESE;
            case "ja":
                return JAPANESE;
            default:
                return ENGLISH;
        }
    }
    
    /**
     * Get language display name
     */
    public static String getLanguageDisplayName(String languageCode) {
        switch (languageCode) {
            case CHINESE:
                return "中文";
            case JAPANESE:
                return "日本語";
            case ENGLISH:
            default:
                return "English";
        }
    }
    
    /**
     * Get all supported languages
     */
    public static String[] getSupportedLanguages() {
        return new String[]{ENGLISH, CHINESE, JAPANESE};
    }
    
    /**
     * Get all supported language display names
     */
    public static String[] getSupportedLanguageNames() {
        return new String[]{"English", "中文", "日本語"};
    }
} 