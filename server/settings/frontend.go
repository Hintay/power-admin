package settings

import "time"

type Frontend struct {
	JwtSecret           string        `ini:"JwtSecret"`
	RefreshTokenExpires time.Duration `ini:"RefreshTokenExpires"`
	AccessTokenExpires  time.Duration `ini:"AccessTokenExpires"`
}

var FrontendSettings = &Frontend{
	JwtSecret:           "",
	RefreshTokenExpires: 24 * time.Hour,
	AccessTokenExpires:  7 * 24 * time.Hour,
}
