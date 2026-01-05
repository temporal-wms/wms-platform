package http

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all HTTP routes for the WES service
func SetupRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Execution plans
		executionPlans := v1.Group("/execution-plans")
		{
			executionPlans.POST("/resolve", handlers.ResolveExecutionPlan)
		}

		// Task routes
		routes := v1.Group("/routes")
		{
			routes.POST("", handlers.CreateTaskRoute)
			routes.GET("/:routeId", handlers.GetTaskRoute)
			routes.GET("/order/:orderId", handlers.GetTaskRouteByOrder)

			// Stage operations on current stage
			routes.POST("/:routeId/stages/current/assign", handlers.AssignWorkerToStage)
			routes.POST("/:routeId/stages/current/start", handlers.StartStage)
			routes.POST("/:routeId/stages/current/complete", handlers.CompleteStage)
			routes.POST("/:routeId/stages/current/fail", handlers.FailStage)
		}

		// Templates
		templates := v1.Group("/templates")
		{
			templates.GET("", handlers.ListTemplates)
			templates.GET("/:templateId", handlers.GetTemplate)
		}
	}
}
