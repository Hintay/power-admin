package settings

import "time"

type InfluxDB struct {
	Enabled  bool          `ini:"Enabled"`
	Host     string        `ini:"Host"`
	Port     int           `ini:"Port"`
	Token    string        `ini:"Token"`
	Database string        `ini:"Database"`
	Timeout  time.Duration `ini:"Timeout"`
	UseSSL   bool          `ini:"UseSSL"`
}

var InfluxDBSettings = &InfluxDB{
	Enabled:  true,
	Host:     "localhost",
	Port:     8086,
	Token:    "",
	Database: "power-data",
	Timeout:  30 * time.Second,
	UseSSL:   false,
}
