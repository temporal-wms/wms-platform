package http

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all process path routes
func RegisterRoutes(router *gin.Engine, handlers *Handlers) {
	api := router.Group("/api/v1/process-paths")
	{
		api.POST("/determine", handlers.DetermineProcessPath())
		api.GET("/:pathId", handlers.GetProcessPath())
		api.GET("/order/:orderId", handlers.GetProcessPathByOrder())
		api.PUT("/:pathId/station", handlers.AssignStation())
	}
}
