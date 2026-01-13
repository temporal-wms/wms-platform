package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/logging"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/mongodb"

	"github.com/wms-platform/labor-service/internal/application"
	"github.com/wms-platform/labor-service/internal/domain"
)

const serviceName = "labor-service"

// Config holds application configuration
type Config struct {
	ServerAddr string
	MongoDB    *mongodb.Config
	Kafka      *kafka.Config
}

func loadConfig() *Config {
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8009"),
		MongoDB: &mongodb.Config{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "labor_db"),
			ConnectTimeout: 10 * time.Second,
			MaxPoolSize:    100,
			MinPoolSize:    10,
		},
		Kafka: &kafka.Config{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: "labor-service",
			ClientID:      "labor-service",
			BatchSize:     100,
			BatchTimeout:  10 * time.Millisecond,
			RequiredAcks:  -1,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createWorkerHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		var req struct {
			WorkerID   string `json:"workerId" binding:"required"`
			EmployeeID string `json:"employeeId" binding:"required"`
			Name       string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": req.WorkerID,
		})

		cmd := application.CreateWorkerCommand{
			WorkerID:   req.WorkerID,
			EmployeeID: req.EmployeeID,
			Name:       req.Name,
		}

		worker, err := service.CreateWorker(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusCreated, worker)
	}
}

func getWorkerHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		query := application.GetWorkerQuery{WorkerID: workerID}

		worker, err := service.GetWorker(c.Request.Context(), query)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startShiftHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			ShiftID   string `json:"shiftId" binding:"required"`
			ShiftType string `json:"shiftType" binding:"required"`
			Zone      string `json:"zone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"shift.id": req.ShiftID,
			"zone":     req.Zone,
		})

		cmd := application.StartShiftCommand{
			WorkerID:  workerID,
			ShiftID:   req.ShiftID,
			ShiftType: req.ShiftType,
			Zone:      req.Zone,
		}

		worker, err := service.StartShift(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func endShiftHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.EndShiftCommand{WorkerID: workerID}

		worker, err := service.EndShift(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startBreakHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			BreakType string `json:"breakType" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.StartBreakCommand{
			WorkerID:  workerID,
			BreakType: req.BreakType,
		}

		worker, err := service.StartBreak(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func endBreakHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.EndBreakCommand{WorkerID: workerID}

		worker, err := service.EndBreak(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func assignTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			TaskID   string `json:"taskId" binding:"required"`
			TaskType string `json:"taskType" binding:"required"`
			Priority int    `json:"priority"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		middleware.AddSpanAttributes(c, map[string]interface{}{
			"task.id":   req.TaskID,
			"task.type": req.TaskType,
		})

		cmd := application.AssignTaskCommand{
			WorkerID: workerID,
			TaskID:   req.TaskID,
			TaskType: domain.TaskType(req.TaskType),
			Priority: req.Priority,
		}

		worker, err := service.AssignTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func startTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		cmd := application.StartTaskCommand{WorkerID: workerID}

		worker, err := service.StartTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func completeTaskHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			ItemsProcessed int `json:"itemsProcessed"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.CompleteTaskCommand{
			WorkerID:       workerID,
			ItemsProcessed: req.ItemsProcessed,
		}

		worker, err := service.CompleteTask(c.Request.Context(), cmd)
		if err != nil {
			if appErr, ok := err.(*errors.AppError); ok {
				responder.RespondWithAppError(appErr)
			} else {
				responder.RespondInternalError(err)
			}
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func addSkillHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		workerID := c.Param("workerId")
		middleware.AddSpanAttributes(c, map[string]interface{}{
			"worker.id": workerID,
		})

		var req struct {
			TaskType  string `json:"taskType" binding:"required"`
			Level     int    `json:"level" binding:"required"`
			Certified bool   `json:"certified"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmd := application.AddSkillCommand{
			WorkerID:  workerID,
			TaskType:  domain.TaskType(req.TaskType),
			Level:     req.Level,
			Certified: req.Certified,
		}

		worker, err := service.AddSkill(c.Request.Context(), cmd)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, worker)
	}
}

func getByStatusHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		status := domain.WorkerStatus(c.Param("status"))

		query := application.GetByStatusQuery{Status: status}

		workers, err := service.GetByStatus(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func getByZoneHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Param("zone")

		query := application.GetByZoneQuery{Zone: zone}

		workers, err := service.GetByZone(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func getAvailableHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		zone := c.Query("zone")

		query := application.GetAvailableQuery{Zone: zone}

		workers, err := service.GetAvailable(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}

func listWorkersHandler(service *application.LaborApplicationService, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		responder := middleware.NewErrorResponder(c, logger.Logger)

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := application.ListWorkersQuery{
			Limit:  limit,
			Offset: offset,
		}

		workers, err := service.ListWorkers(c.Request.Context(), query)
		if err != nil {
			responder.RespondInternalError(err)
			return
		}

		c.JSON(http.StatusOK, workers)
	}
}
