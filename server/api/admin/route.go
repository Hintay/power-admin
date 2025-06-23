package admin

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers admin API routes
func RegisterRoutes(r *gin.RouterGroup) {
	// User management
	registerUserRoutes(r)

	// Collector management
	registerCollectorRoutes(r)

	// System management
	registerSystemRoutes(r)

	// Data analytics
	registerAnalyticsRoutes(r)
}
