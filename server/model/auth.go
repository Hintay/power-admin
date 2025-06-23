package model

import (
	"time"
)

// AuthToken represents authentication tokens
type AuthToken struct {
	BaseModel
	UserID    uint      `gorm:"index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"-"`
	TokenType string    `gorm:"not null" json:"token_type"` // access, refresh, collector
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	IsRevoked bool      `gorm:"default:false" json:"is_revoked"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// IsValid checks if the token is valid (not expired and not revoked)
func (t *AuthToken) IsValid() bool {
	return !t.IsRevoked && time.Now().Before(t.ExpiresAt)
}

// IsExpired checks if the token is expired
func (t *AuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// Revoke marks the token as revoked
func (t *AuthToken) Revoke() {
	t.IsRevoked = true
}
