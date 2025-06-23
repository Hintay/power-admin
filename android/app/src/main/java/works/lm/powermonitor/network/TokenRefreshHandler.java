package works.lm.powermonitor.network;

/**
 * Interface for handling token refresh events
 */
public interface TokenRefreshHandler {
    
    /**
     * Called when token refresh is successful
     */
    void onTokenRefreshSuccess();
    
    /**
     * Called when token refresh fails
     * This usually means the user needs to log in again
     */
    void onTokenRefreshFailed();
    
    /**
     * Called when no refresh token is available
     * This means the user was never logged in or tokens were cleared
     */
    void onNoRefreshToken();
} 