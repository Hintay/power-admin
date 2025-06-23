package settings

import "time"

type Collector struct {
	TokenExpires            time.Duration `ini:"TokenExpires"`
	RegistrationCodeExpires time.Duration `ini:"RegistrationCodeExpires"`
}

var CollectorSettings = &Collector{
	TokenExpires:            30 * 24 * time.Hour,
	RegistrationCodeExpires: 7 * 24 * time.Hour,
}
