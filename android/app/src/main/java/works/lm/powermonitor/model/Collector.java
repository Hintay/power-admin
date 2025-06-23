package works.lm.powermonitor.model;

import java.util.Date;

/**
 * Collector model representing data collection endpoints
 */
public class Collector {
    private int id;                    // Database primary key (from BaseModel)
    private String collector_id;       // Actual collector identifier (UUID)
    private String name;
    private String description;
    private String location;
    private boolean is_active;         // Match server field name
    private Date last_seen_at;         // Match server field name
    private int collectInterval;       // Collection interval in seconds (local field)
    private double pricePerKwh;        // Electricity price per kWh (local field)

    public Collector() {}

    public Collector(int id, String collector_id, String name, String description, String location,
                     boolean is_active, Date last_seen_at, int collectInterval, double pricePerKwh) {
        this.id = id;
        this.collector_id = collector_id;
        this.name = name;
        this.description = description;
        this.location = location;
        this.is_active = is_active;
        this.last_seen_at = last_seen_at;
        this.collectInterval = collectInterval;
        this.pricePerKwh = pricePerKwh;
    }

    // Getters and Setters
    public int getId() { return id; }
    public void setId(int id) { this.id = id; }

    public String getCollectorId() { return collector_id; }
    public void setCollectorId(String collector_id) { this.collector_id = collector_id; }

    // Backward compatibility - return collector_id for WebSocket usage
    public String getCollectorIdForWebSocket() { return collector_id; }

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public String getLocation() { return location; }
    public void setLocation(String location) { this.location = location; }

    public boolean isActive() { return is_active; }
    public void setActive(boolean is_active) { this.is_active = is_active; }

    // Backward compatibility
    public boolean isOnline() { return is_active; }
    public void setOnline(boolean online) { this.is_active = online; }

    public Date getLastSeenAt() { return last_seen_at; }
    public void setLastSeenAt(Date last_seen_at) { this.last_seen_at = last_seen_at; }

    // Backward compatibility
    public Date getLastSeen() { return last_seen_at; }
    public void setLastSeen(Date lastSeen) { this.last_seen_at = lastSeen; }

    public int getCollectInterval() { return collectInterval; }
    public void setCollectInterval(int collectInterval) { this.collectInterval = collectInterval; }

    public double getPricePerKwh() { return pricePerKwh; }
    public void setPricePerKwh(double pricePerKwh) { this.pricePerKwh = pricePerKwh; }

    @Override
    public String toString() {
        return name + " (" + collector_id + ")";
    }
} 