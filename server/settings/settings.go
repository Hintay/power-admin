package settings

import (
	"github.com/elliotchance/orderedmap/v3"
	"github.com/uozi-tech/cosy/settings"
)

var sections = orderedmap.NewOrderedMap[string, any]()

func init() {
	sections.Set("collector", CollectorSettings)
	sections.Set("database", DatabaseSettings)
	sections.Set("influxdb", InfluxDBSettings)
	sections.Set("auth", AuthSettings)
	sections.Set("frontend", FrontendSettings)

	for k, v := range sections.AllFromFront() {
		settings.Register(k, v)
	}
	settings.WithoutRedis()
	settings.WithoutSonyflake()
}

// Init initializes all settings
func Init(confPath string) {
	// Initialize cosy settings first
	settings.Init(confPath)

	// Set Default Port
	if settings.ServerSettings.Port == 0 {
		settings.ServerSettings.Port = 9000
	}
}
