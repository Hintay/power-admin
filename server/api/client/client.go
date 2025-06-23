package client

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers client API routes
func RegisterRoutes(r *gin.RouterGroup) {
	// Data access
	registerDataRoutes(r)

	// Analytics
	registerAnalyticsRoutes(r)
}
