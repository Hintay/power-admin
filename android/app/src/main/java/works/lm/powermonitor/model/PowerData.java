package works.lm.powermonitor.model;

import java.util.Date;

/**
 * Power data model representing electrical measurements from PZEM-004
 */
public class PowerData {
    private String collectorId;
    private String collectorName;
    private Date timestamp;
    private double voltage;         // Voltage in V
    private double current;         // Current in A
    private double power;           // Power in W
    private double energy;          // Energy in Wh
    private double frequency;       // Frequency in Hz
    private double powerFactor;     // Power factor

    // Constructors
    public PowerData() {}

    public PowerData(String collectorId, String collectorName, Date timestamp,
                     double voltage, double current, double power, double energy,
                     double frequency, double powerFactor) {
        this.collectorId = collectorId;
        this.collectorName = collectorName;
        this.timestamp = timestamp;
        this.voltage = voltage;
        this.current = current;
        this.power = power;
        this.energy = energy;
        this.frequency = frequency;
        this.powerFactor = powerFactor;
    }

    // Getters and Setters
    public String getCollectorId() { return collectorId; }
    public void setCollectorId(String collectorId) { this.collectorId = collectorId; }

    public String getCollectorName() { return collectorName; }
    public void setCollectorName(String collectorName) { this.collectorName = collectorName; }

    public Date getTimestamp() { return timestamp; }
    public void setTimestamp(Date timestamp) { this.timestamp = timestamp; }

    public double getVoltage() { return voltage; }
    public void setVoltage(double voltage) { this.voltage = voltage; }

    public double getCurrent() { return current; }
    public void setCurrent(double current) { this.current = current; }

    public double getPower() { return power; }
    public void setPower(double power) { this.power = power; }

    public double getEnergy() { return energy; }
    public void setEnergy(double energy) { this.energy = energy; }

    public double getFrequency() { return frequency; }
    public void setFrequency(double frequency) { this.frequency = frequency; }

    public double getPowerFactor() { return powerFactor; }
    public void setPowerFactor(double powerFactor) { this.powerFactor = powerFactor; }

    /**
     * Calculate estimated cost based on energy consumption (Wh) and price per kWh
     * Note: Energy is converted from Wh to kWh for cost calculation
     */
    public double calculateCost(double pricePerKwh) {
        return (energy / 1000.0) * pricePerKwh;
    }

    @Override
    public String toString() {
        return "PowerData{" +
                "collectorId='" + collectorId + '\'' +
                ", collectorName='" + collectorName + '\'' +
                ", timestamp=" + timestamp +
                ", voltage=" + voltage +
                ", current=" + current +
                ", power=" + power +
                ", energy=" + energy +
                ", frequency=" + frequency +
                ", powerFactor=" + powerFactor +
                '}';
    }
} 