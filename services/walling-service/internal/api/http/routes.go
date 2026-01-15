package http

import (
	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/middleware"
)

// SetupRoutes configures all HTTP routes for the Walling service
func SetupRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 routes with tenant context required
	v1 := router.Group("/api/v1")
	v1.Use(middleware.RequireTenantAuth()) // All API routes require tenant headers
	{
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", handlers.CreateTask)
			tasks.GET("/pending", handlers.GetPendingTasksByPutWall)
			tasks.GET("/walliner/:wallinerId/active", handlers.GetActiveTaskByWalliner)
			tasks.GET("/:taskId", handlers.GetTask)
			tasks.POST("/:taskId/assign", handlers.AssignWalliner)
			tasks.POST("/:taskId/sort", handlers.SortItem)
			tasks.POST("/:taskId/complete", handlers.CompleteTask)
		}
	}
}
