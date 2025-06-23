package router

import (
	"Power-Monitor/api/admin"
	"Power-Monitor/api/client"
	"Power-Monitor/api/collector"
	"Power-Monitor/internal/middleware"
	"Power-Monitor/internal/realtime"
	"github.com/gin-gonic/gin"
	"github.com/uozi-tech/cosy/router"
)

// InitRouter initializes all routes
func InitRouter() {
	engine := router.GetEngine()

	// Add global middleware
	engine.Use(middleware.CORS())

	// API routes
	root := engine.Group("/api")
	{
		// Collector routes (for data collection devices)
		collectorGroup := root.Group("/collector")
		collectorGroup.Use(middleware.CollectorAuth())
		{
			collector.RegisterRoutes(collectorGroup)
		}

		// Admin routes (for management dashboard)
		adminGroup := root.Group("/admin")
		adminGroup.Use(middleware.JWTAuth())
		{
			admin.RegisterRoutes(adminGroup)
		}

		// Client routes (for web/mobile clients)
		clientGroup := root.Group("/client")
		clientGroup.Use(middleware.JWTAuth())
		{
			client.RegisterRoutes(clientGroup)
		}

		// Public auth routes
		authGroup := root.Group("/auth")
		{
			collector.RegisterAuthRoutes(authGroup)
			client.RegisterAuthRoutes(authGroup)
			authGuard := authGroup.Group("/", middleware.JWTAuth())
			{
				client.RegisterAuthGuardRoutes(authGuard)
			}
		}

		// Real-time communication routes
		realtimeGroup := root.Group("/realtime")
		realtimeGroup.Use(middleware.JWTAuth())
		{
			realtimeGroup.GET("/ws", realtime.HandleWebSocket)
		}

		// Health check
		root.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "power-monitor",
				"version": "1.0.0",
			})
		})
	}
}
