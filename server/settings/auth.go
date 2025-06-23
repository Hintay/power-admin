package settings

type Auth struct {
	IPWhiteList         string `ini:"IPWhiteList"`
	BanThresholdMinutes int    `ini:"BanThresholdMinutes"`
	MaxAttempts         int    `ini:"MaxAttempts"`
}

var AuthSettings = &Auth{
	IPWhiteList:         "",
	BanThresholdMinutes: 10,
	MaxAttempts:         10,
}
