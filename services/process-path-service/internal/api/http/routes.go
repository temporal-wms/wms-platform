package http

import (
	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/middleware"
)

// RegisterRoutes registers all process path routes
func RegisterRoutes(router *gin.Engine, handlers *Handlers) {
	// Process path management routes with tenant context required
	processPathAPI := router.Group("/api/v1/process-paths")
	processPathAPI.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		processPathAPI.POST("/determine", handlers.DetermineProcessPath())
		processPathAPI.GET("/:pathId", handlers.GetProcessPath())
		processPathAPI.GET("/order/:orderId", handlers.GetProcessPathByOrder())
		processPathAPI.PUT("/:pathId/station", handlers.AssignStation())
		processPathAPI.POST("/:pathId/escalate", handlers.EscalateProcessPath())
		processPathAPI.POST("/:pathId/downgrade", handlers.DowngradeProcessPath())
	}

	// Routing optimization routes (Phase 3.1 & 3.3) with tenant context required
	routingAPI := router.Group("/api/v1/routing")
	routingAPI.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		routingAPI.POST("/optimize", handlers.OptimizeRouting())
		routingAPI.GET("/metrics", handlers.GetRoutingMetrics())
		routingAPI.POST("/reroute", handlers.RerouteOrder())
	}
}
