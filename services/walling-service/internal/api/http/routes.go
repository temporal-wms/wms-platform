package http

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all HTTP routes for the Walling service
func SetupRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 routes
	v1 := router.Group("/api/v1")
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
