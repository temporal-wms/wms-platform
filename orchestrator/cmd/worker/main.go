package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wms-platform/orchestrator/internal/activities"
	"github.com/wms-platform/orchestrator/internal/workflows"
	"github.com/wms-platform/shared/pkg/metrics"
	"github.com/wms-platform/shared/pkg/middleware"
	"github.com/wms-platform/shared/pkg/temporal"
	"go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/client"
)

func main() {
	// Setup JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting orchestrator worker")

	// Load configuration
	config := loadConfig()

	// Initialize Prometheus metrics
	metricsConfig := metrics.DefaultConfig("orchestrator")
	m := metrics.New(metricsConfig)
	logger.Info("Metrics initialized")

	// Initialize failure metrics helper
	failureMetrics := middleware.NewFailureMetrics(m)

	// Initialize Temporal client
	ctx := context.Background()
	temporalClient, err := temporal.NewClient(ctx, config.Temporal)
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Connected to Temporal", "hostPort", config.Temporal.HostPort, "namespace", config.Temporal.Namespace)

	// Initialize HTTP clients for service communication
	serviceClients := activities.NewServiceClients(&activities.ServiceClientsConfig{
		OrderServiceURL:         config.OrderServiceURL,
		InventoryServiceURL:     config.InventoryServiceURL,
		RoutingServiceURL:       config.RoutingServiceURL,
		PickingServiceURL:       config.PickingServiceURL,
		ConsolidationServiceURL: config.ConsolidationServiceURL,
		PackingServiceURL:       config.PackingServiceURL,
		ShippingServiceURL:      config.ShippingServiceURL,
		LaborServiceURL:         config.LaborServiceURL,
		WavingServiceURL:        config.WavingServiceURL,
		FacilityServiceURL:      config.FacilityServiceURL,
		UnitServiceURL:          config.UnitServiceURL,
		ProcessPathServiceURL:   config.ProcessPathServiceURL,
		BillingServiceURL:       config.BillingServiceURL,
		ChannelServiceURL:       config.ChannelServiceURL,
		SellerServiceURL:        config.SellerServiceURL,
	})

	// Create activities with service clients
	orderActivities := activities.NewOrderActivities(serviceClients, logger)
	inventoryActivities := activities.NewInventoryActivities(serviceClients, logger)
	routingActivities := activities.NewRoutingActivities(serviceClients, logger)
	pickingActivities := activities.NewPickingActivities(serviceClients, logger)
	consolidationActivities := activities.NewConsolidationActivities(serviceClients, logger)
	packingActivities := activities.NewPackingActivities(serviceClients, logger)
	shippingActivities := activities.NewShippingActivities(serviceClients, logger)
	reprocessingActivities := activities.NewReprocessingActivities(serviceClients, temporalClient.Client(), logger, failureMetrics)
	processPathActivities := activities.NewProcessPathActivities(serviceClients, logger)
	giftWrapActivities := activities.NewGiftWrapActivities(serviceClients)

	// Create Phase 2 & 3 enhancement activities
	laborActivities := activities.NewLaborActivities(serviceClients, logger)
	equipmentActivities := activities.NewEquipmentActivities(serviceClients, logger)
	routingOptimizerActivities := activities.NewRoutingOptimizerActivities(serviceClients)
	escalationActivities := activities.NewEscalationActivities(serviceClients)
	continuousOptimizationActivities := activities.NewContinuousOptimizationActivities(serviceClients)

	// Create Amazon-aligned fulfillment activities
	receivingActivities := activities.NewReceivingActivities()
	stowActivities := activities.NewStowActivities()
	slamActivities := activities.NewSLAMActivities()
	sortationActivities := activities.NewSortationActivities()

	// Create unit-level tracking activities
	unitActivities := activities.NewUnitActivities(serviceClients, logger)

	// Create worker
	workerOpts := temporal.DefaultWorkerOptions(temporal.TaskQueues.Orchestrator)
	w := temporalClient.NewWorker(workerOpts)

	// Register workflows
	w.RegisterWorkflow(workflows.OrderFulfillmentWorkflow)
	w.RegisterWorkflow(workflows.OrderCancellationWorkflow)
	w.RegisterWorkflow(workflows.OrderCancellationWorkflowWithAllocations)
	w.RegisterWorkflow(workflows.OrchestratedPickingWorkflow)
	w.RegisterWorkflow(workflows.ConsolidationWorkflow)
	w.RegisterWorkflow(workflows.PackingWorkflow)
	w.RegisterWorkflow(workflows.ShippingWorkflow)
	w.RegisterWorkflow(workflows.GiftWrapWorkflow)
	w.RegisterWorkflow(workflows.ReprocessingBatchWorkflow)
	w.RegisterWorkflow(workflows.ReprocessingOrchestrationWorkflow)
	w.RegisterWorkflow(workflows.StockShortageWorkflow)
	w.RegisterWorkflow(workflows.BackorderFulfillmentWorkflow)
	w.RegisterWorkflow(workflows.InboundFulfillmentWorkflow)
	w.RegisterWorkflow(workflows.SortationWorkflow)
	w.RegisterWorkflow(workflows.BatchSortationWorkflow)
	w.RegisterWorkflow(workflows.PlanningWorkflow)
	w.RegisterWorkflow(workflows.ContinuousOptimizationWorkflow)
	logger.Info("Registered workflows", "workflows", []string{
		"OrderFulfillmentWorkflow",
		"OrderCancellationWorkflow",
		"OrchestratedPickingWorkflow",
		"ConsolidationWorkflow",
		"PackingWorkflow",
		"ShippingWorkflow",
		"GiftWrapWorkflow",
		"ReprocessingBatchWorkflow",
		"ReprocessingOrchestrationWorkflow",
		"StockShortageWorkflow",
		"BackorderFulfillmentWorkflow",
		"InboundFulfillmentWorkflow",
		"SortationWorkflow",
		"BatchSortationWorkflow",
		"PlanningWorkflow",
		"ContinuousOptimizationWorkflow",
	})

	// Register activities
	w.RegisterActivity(orderActivities.ValidateOrder)
	w.RegisterActivity(orderActivities.CancelOrder)
	w.RegisterActivity(orderActivities.NotifyCustomerCancellation)
	w.RegisterActivity(orderActivities.AssignToWave)
	w.RegisterActivity(orderActivities.StartPicking)
	w.RegisterActivity(orderActivities.MarkConsolidated)
	w.RegisterActivity(orderActivities.MarkPacked)
	w.RegisterActivity(inventoryActivities.ReleaseInventoryReservation)
	w.RegisterActivity(inventoryActivities.ReserveInventory)
	w.RegisterActivity(inventoryActivities.ConfirmInventoryPick)
	w.RegisterActivity(inventoryActivities.GetReservationIDs)
	w.RegisterActivity(inventoryActivities.StageInventory)
	w.RegisterActivity(inventoryActivities.PackInventory)
	w.RegisterActivity(inventoryActivities.ShipInventory)
	w.RegisterActivity(inventoryActivities.ReturnInventoryToShelf)
	w.RegisterActivity(inventoryActivities.RecordStockShortage)
	w.RegisterActivity(routingActivities.CalculateRoute)
	w.RegisterActivity(routingActivities.CalculateMultiRoute)

	// Register picking activities
	w.RegisterActivity(pickingActivities.CreatePickTask)
	w.RegisterActivity(pickingActivities.AssignPickerToTask)

	// Register consolidation activities
	w.RegisterActivity(consolidationActivities.CreateConsolidationUnit)
	w.RegisterActivity(consolidationActivities.ConsolidateItems)
	w.RegisterActivity(consolidationActivities.VerifyConsolidation)
	w.RegisterActivity(consolidationActivities.CompleteConsolidation)

	// Register packing activities
	w.RegisterActivity(packingActivities.CreatePackTask)
	w.RegisterActivity(packingActivities.SelectPackagingMaterials)
	w.RegisterActivity(packingActivities.PackItems)
	w.RegisterActivity(packingActivities.WeighPackage)
	w.RegisterActivity(packingActivities.GenerateShippingLabel)
	w.RegisterActivity(packingActivities.ApplyLabelToPackage)
	w.RegisterActivity(packingActivities.SealPackage)

	// Register shipping activities
	w.RegisterActivity(shippingActivities.CreateShipment)
	w.RegisterActivity(shippingActivities.ScanPackage)
	w.RegisterActivity(shippingActivities.VerifyShippingLabel)
	w.RegisterActivity(shippingActivities.PlaceOnOutboundDock)
	w.RegisterActivity(shippingActivities.AddToCarrierManifest)
	w.RegisterActivity(shippingActivities.MarkOrderShipped)
	w.RegisterActivity(shippingActivities.NotifyCustomerShipped)

	// Register reprocessing activities
	w.RegisterActivity(reprocessingActivities.QueryFailedWorkflows)
	w.RegisterActivity(reprocessingActivities.ProcessFailedWorkflow)

	// Register process path activities
	w.RegisterActivity(processPathActivities.DetermineProcessPath)
	w.RegisterActivity(processPathActivities.FindCapableStation)
	w.RegisterActivity(processPathActivities.GetStation)
	w.RegisterActivity(processPathActivities.GetStationsByZone)
	w.RegisterActivity(processPathActivities.ReserveStationCapacity)
	w.RegisterActivity(processPathActivities.ReleaseStationCapacity)

	// Register Phase 2.2: Labor certification activities
	w.RegisterActivity(laborActivities.ValidateWorkerCertification)
	w.RegisterActivity(laborActivities.AssignCertifiedWorker)
	w.RegisterActivity(laborActivities.GetAvailableWorkers)

	// Register Phase 2.3: Equipment availability activities
	w.RegisterActivity(equipmentActivities.CheckEquipmentAvailability)
	w.RegisterActivity(equipmentActivities.ReserveEquipment)
	w.RegisterActivity(equipmentActivities.ReleaseEquipment)

	// Register Phase 3.1: Routing optimizer activities (ATROPS-like)
	w.RegisterActivity(routingOptimizerActivities.OptimizeStationSelection)
	w.RegisterActivity(routingOptimizerActivities.GetRoutingMetrics)
	w.RegisterActivity(routingOptimizerActivities.RerouteOrder)

	// Register Phase 3.2: Escalation activities
	w.RegisterActivity(escalationActivities.EscalateProcessPath)
	w.RegisterActivity(escalationActivities.DetermineEscalationTier)
	w.RegisterActivity(escalationActivities.FindFallbackStations)
	w.RegisterActivity(escalationActivities.DowngradeProcessPath)

	// Register Phase 3.3: Continuous optimization activities
	w.RegisterActivity(continuousOptimizationActivities.MonitorSystemHealth)
	w.RegisterActivity(continuousOptimizationActivities.RebalanceWaves)
	w.RegisterActivity(continuousOptimizationActivities.TriggerDynamicRerouting)
	w.RegisterActivity(continuousOptimizationActivities.PredictCapacityNeeds)

	// Register gift wrap activities
	w.RegisterActivity(giftWrapActivities.CreateGiftWrapTask)
	w.RegisterActivity(giftWrapActivities.AssignGiftWrapWorker)
	w.RegisterActivity(giftWrapActivities.CheckGiftWrapStatus)
	w.RegisterActivity(giftWrapActivities.ApplyGiftMessage)
	w.RegisterActivity(giftWrapActivities.CompleteGiftWrapTask)

	// Register receiving activities (Amazon-aligned inbound)
	w.RegisterActivity(receivingActivities.ValidateASN)
	w.RegisterActivity(receivingActivities.MarkShipmentArrived)
	w.RegisterActivity(receivingActivities.PerformQualityInspection)
	w.RegisterActivity(receivingActivities.CreatePutawayTasks)
	w.RegisterActivity(receivingActivities.ConfirmInventoryReceipt)
	w.RegisterActivity(receivingActivities.CompleteReceiving)
	w.RegisterActivity(receivingActivities.ProcessReceiving)

	// Register stow activities (chaotic storage)
	w.RegisterActivity(stowActivities.FindStorageLocation)
	w.RegisterActivity(stowActivities.AssignLocation)
	w.RegisterActivity(stowActivities.ExecuteStow)
	w.RegisterActivity(stowActivities.UpdateInventoryLocation)
	w.RegisterActivity(stowActivities.ProcessStow)

	// Register SLAM activities (Scan, Label, Apply, Manifest)
	// Note: ScanPackage is not registered here to avoid conflict with ShippingActivities.ScanPackage
	// SLAM's ScanPackage is called internally by ExecuteSLAM via Go method call
	w.RegisterActivity(slamActivities.GenerateLabel)
	w.RegisterActivity(slamActivities.ApplyLabel)
	w.RegisterActivity(slamActivities.AddToManifest)
	w.RegisterActivity(slamActivities.VerifyWeight)
	w.RegisterActivity(slamActivities.ExecuteSLAM)

	// Register sortation activities
	w.RegisterActivity(sortationActivities.CreateSortationBatch)
	w.RegisterActivity(sortationActivities.AddPackageToBatch)
	w.RegisterActivity(sortationActivities.AssignChute)
	w.RegisterActivity(sortationActivities.SortPackage)
	w.RegisterActivity(sortationActivities.CloseBatch)
	w.RegisterActivity(sortationActivities.DispatchBatch)
	w.RegisterActivity(sortationActivities.ProcessSortation)
	w.RegisterActivity(sortationActivities.NotifyCarrier)

	// Register unit-level tracking activities
	w.RegisterActivity(unitActivities.CreateUnits)
	w.RegisterActivity(unitActivities.ReserveUnits)
	w.RegisterActivity(unitActivities.ReleaseUnits)
	w.RegisterActivity(unitActivities.GetUnitsForOrder)
	w.RegisterActivity(unitActivities.ConfirmUnitPick)
	w.RegisterActivity(unitActivities.ConfirmUnitConsolidation)
	w.RegisterActivity(unitActivities.ConfirmUnitPacked)
	w.RegisterActivity(unitActivities.ConfirmUnitShipped)
	w.RegisterActivity(unitActivities.CreateUnitException)
	w.RegisterActivity(unitActivities.GetUnitAuditTrail)
	w.RegisterActivity(unitActivities.PersistProcessPath)
	w.RegisterActivity(unitActivities.GetProcessPath)

	logger.Info("Registered activities", "activities", []string{
		"ValidateOrder",
		"CancelOrder",
		"NotifyCustomerCancellation",
		"AssignToWave",
		"StartPicking",
		"MarkConsolidated",
		"MarkPacked",
		"ReleaseInventoryReservation",
		"ReserveInventory",
		"ConfirmInventoryPick",
		"RecordStockShortage",
		"CalculateRoute",
		"CalculateMultiRoute",
		"CreatePickTask",
		"AssignPickerToTask",
		"CreateConsolidationUnit",
		"ConsolidateItems",
		"VerifyConsolidation",
		"CompleteConsolidation",
		"CreatePackTask",
		"SelectPackagingMaterials",
		"PackItems",
		"WeighPackage",
		"GenerateShippingLabel",
		"ApplyLabelToPackage",
		"SealPackage",
		"CreateShipment",
		"ScanPackage",
		"VerifyShippingLabel",
		"PlaceOnOutboundDock",
		"AddToCarrierManifest",
		"MarkOrderShipped",
		"NotifyCustomerShipped",
		"QueryFailedWorkflows",
		"ProcessFailedWorkflow",
		"DetermineProcessPath",
		"FindCapableStation",
		"GetStation",
		"GetStationsByZone",
		"ReserveStationCapacity",
		"ReleaseStationCapacity",
		// Phase 2.2: Labor certification activities
		"ValidateWorkerCertification",
		"AssignCertifiedWorker",
		"GetAvailableWorkers",
		// Phase 2.3: Equipment availability activities
		"CheckEquipmentAvailability",
		"ReserveEquipment",
		"ReleaseEquipment",
		// Phase 3.1: Routing optimizer activities
		"OptimizeStationSelection",
		"GetRoutingMetrics",
		"RerouteOrder",
		// Phase 3.2: Escalation activities
		"EscalateProcessPath",
		"DetermineEscalationTier",
		"FindFallbackStations",
		"DowngradeProcessPath",
		// Phase 3.3: Continuous optimization activities
		"MonitorSystemHealth",
		"RebalanceWaves",
		"TriggerDynamicRerouting",
		"PredictCapacityNeeds",
		"CreateGiftWrapTask",
		"AssignGiftWrapWorker",
		"CheckGiftWrapStatus",
		"ApplyGiftMessage",
		"CompleteGiftWrapTask",
		// Receiving activities
		"ValidateASN",
		"MarkShipmentArrived",
		"PerformQualityInspection",
		"CreatePutawayTasks",
		"ConfirmInventoryReceipt",
		"CompleteReceiving",
		"ProcessReceiving",
		// Stow activities
		"FindStorageLocation",
		"AssignLocation",
		"ExecuteStow",
		"UpdateInventoryLocation",
		"ProcessStow",
		// SLAM activities
		"ScanPackage",
		"GenerateLabel",
		"ApplyLabel",
		"AddToManifest",
		"VerifyWeight",
		"ExecuteSLAM",
		// Sortation activities
		"CreateSortationBatch",
		"AddPackageToBatch",
		"AssignChute",
		"SortPackage",
		"CloseBatch",
		"DispatchBatch",
		"ProcessSortation",
		"NotifyCarrier",
		// Unit-level tracking activities
		"CreateUnits",
		"ReserveUnits",
		"ReleaseUnits",
		"GetUnitsForOrder",
		"ConfirmUnitPick",
		"ConfirmUnitConsolidation",
		"ConfirmUnitPacked",
		"ConfirmUnitShipped",
		"CreateUnitException",
		"GetUnitAuditTrail",
		"PersistProcessPath",
		"GetProcessPath",
	})

	// Create reprocessing schedule if enabled
	if getEnv("REPROCESSING_ENABLED", "true") == "true" {
		if err := createReprocessingSchedule(ctx, temporalClient.Client(), logger); err != nil {
			logger.Warn("Failed to create reprocessing schedule", "error", err)
			// Continue even if schedule creation fails - can be created manually
		}
	}

	// Start health server with signal bridge
	healthPort := getEnv("HEALTH_PORT", "8080")
	healthServer := startHealthServer(healthPort, temporalClient.Client(), m, logger)
	logger.Info("Health server started", "port", healthPort)

	// Start worker in background
	go func() {
		if err := w.Run(nil); err != nil {
			logger.Error("Worker failed", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("Worker started", "taskQueue", temporal.TaskQueues.Orchestrator)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down worker...")

	// Shutdown health server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(ctx); err != nil {
		logger.Error("Health server shutdown failed", "error", err)
	}

	w.Stop()
	logger.Info("Worker stopped")
}

func startHealthServer(port string, temporalClient temporalclient.Client, m *metrics.Metrics, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Ready check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", m.Handler())

	// Signal bridge endpoints for simulators
	mux.HandleFunc("/api/v1/signals/wave-assigned", createWaveAssignedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/pick-completed", createPickCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/tote-arrived", createToteArrivedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/consolidation-completed", createConsolidationCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/gift-wrap-completed", createGiftWrapCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/walling-completed", createWallingCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/packing-completed", createPackingCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/receiving-completed", createReceivingCompletedHandler(temporalClient, logger))
	mux.HandleFunc("/api/v1/signals/stow-completed", createStowCompletedHandler(temporalClient, logger))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health server failed", "error", err)
		}
	}()

	return server
}

// PickCompletedRequest represents the request body for the pick-completed signal
type PickCompletedRequest struct {
	OrderID     string       `json:"orderId"`
	TaskID      string       `json:"taskId"`
	PickedItems []PickedItem `json:"pickedItems"`
}

// PickedItem represents a picked item in the signal
type PickedItem struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	LocationID string `json:"locationId"`
	ToteID     string `json:"toteId"`
}

// WaveAssignedRequest represents the request body for the wave-assigned signal
type WaveAssignedRequest struct {
	OrderID string `json:"orderId"`
	WaveID  string `json:"waveId"`
}

// ToteArrivedRequest represents the request body for the tote-arrived signal (multi-route support)
type ToteArrivedRequest struct {
	OrderID    string `json:"orderId"`
	ToteID     string `json:"toteId"`
	RouteID    string `json:"routeId"`
	RouteIndex int    `json:"routeIndex"`
	ArrivedAt  string `json:"arrivedAt"`
}

// ConsolidationCompletedRequest represents the request body for the consolidation-completed signal
type ConsolidationCompletedRequest struct {
	OrderID           string                 `json:"orderId"`
	ConsolidationID   string                 `json:"consolidationId"`
	ConsolidatedItems []map[string]interface{} `json:"consolidatedItems"`
}

// GiftWrapCompletedRequest represents the request body for the gift-wrap-completed signal
type GiftWrapCompletedRequest struct {
	OrderID     string `json:"orderId"`
	StationID   string `json:"stationId"`
	WrapType    string `json:"wrapType"`
	GiftMessage string `json:"giftMessage"`
	CompletedAt string `json:"completedAt"`
}

// WallingCompletedRequest represents the request body for the walling-completed signal
type WallingCompletedRequest struct {
	OrderID     string                   `json:"orderId"`
	TaskID      string                   `json:"taskId"`
	RouteID     string                   `json:"routeId"`
	SortedItems []map[string]interface{} `json:"sortedItems"`
}

// PackingCompletedRequest represents the packing completed signal payload
type PackingCompletedRequest struct {
	OrderID     string                 `json:"orderId"`
	TaskID      string                 `json:"taskId"`
	PackageInfo map[string]interface{} `json:"packageInfo"`
}

// ReceivingCompletedRequest represents the request body for the receiving-completed signal
type ReceivingCompletedRequest struct {
	ShipmentID    string                   `json:"shipmentId"`
	ReceivedItems []map[string]interface{} `json:"receivedItems"`
	TotalReceived int                      `json:"totalReceived"`
	TotalDamaged  int                      `json:"totalDamaged"`
}

// StowCompletedRequest represents the request body for the stow-completed signal
type StowCompletedRequest struct {
	ShipmentID  string                   `json:"shipmentId"`
	StowedItems []map[string]interface{} `json:"stowedItems"`
	CompletedAt string                   `json:"completedAt"`
}

// createWaveAssignedHandler creates a handler for the wave-assigned signal endpoint
func createWaveAssignedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		// Parse request
		var req WaveAssignedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		// Validate request
		if req.OrderID == "" || req.WaveID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId and waveId are required"}`))
			return
		}

		// Construct workflow ID (planning workflow follows pattern "planning-{orderId}")
		workflowID := fmt.Sprintf("planning-%s", req.OrderID)

		// Build signal payload matching workflow expectations
		signalPayload := map[string]interface{}{
			"waveId": req.WaveID,
		}

		// Send signal to Temporal workflow
		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"", // Run ID - empty to signal the latest run
			"waveAssigned",
			signalPayload,
		)
		if err != nil {
			logger.Error("Failed to signal workflow", "workflowId", workflowID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": workflowID,
			})
			return
		}

		logger.Info("Successfully signaled wave assignment",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"waveId", req.WaveID,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Wave assigned signal sent successfully",
		})
	}
}

// createPickCompletedHandler creates a handler for the pick-completed signal endpoint
func createPickCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		// Parse request
		var req PickCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		// Validate request
		if req.OrderID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId is required"}`))
			return
		}

		// Construct workflow ID (picking workflow follows pattern "picking-{orderId}")
		workflowID := fmt.Sprintf("picking-%s", req.OrderID)

		// Build signal payload matching workflow expectations
		signalPayload := map[string]interface{}{
			"taskId":      req.TaskID,
			"pickedItems": req.PickedItems,
		}

	// Send signal to Temporal workflow
	ctx := r.Context()
	err = temporalClient.SignalWorkflow(
		ctx,
		workflowID,
		"", // Run ID - empty to signal the latest run
		"pickCompleted",
		signalPayload,
	)
	if err != nil {
		logger.Error("Failed to signal picking workflow",
			"orderId", req.OrderID,
			"workflowId", workflowID,
			"error", err,
		)

		// If picking workflow not found, try WES workflow
		wesWorkflowID := fmt.Sprintf("wes-%s", req.OrderID)
		err = temporalClient.SignalWorkflow(ctx, wesWorkflowID, "", "pickCompleted", signalPayload)
		if err != nil {
			logger.Error("Failed to signal WES workflow for picking",
				"orderId", req.OrderID,
				"workflowId", wesWorkflowID,
				"error", err,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": wesWorkflowID,
			})
			return
		}
		workflowID = wesWorkflowID
	}

		logger.Info("Successfully signaled workflow",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"taskId", req.TaskID,
			"itemsCount", len(req.PickedItems),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Signal sent successfully",
		})
	}
}

// createToteArrivedHandler creates a handler for the tote-arrived signal endpoint (multi-route support)
func createToteArrivedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req ToteArrivedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.OrderID == "" || req.ToteID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId and toteId are required"}`))
			return
		}

		// Signal the consolidation workflow for the order
		workflowID := fmt.Sprintf("consolidation-%s", req.OrderID)

		signalPayload := map[string]interface{}{
			"toteId":     req.ToteID,
			"routeId":    req.RouteID,
			"routeIndex": req.RouteIndex,
			"arrivedAt":  req.ArrivedAt,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"toteArrived",
			signalPayload,
		)
		if err != nil {
			logger.Error("Failed to signal workflow", "workflowId", workflowID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": workflowID,
			})
			return
		}

		logger.Info("Successfully signaled tote arrival",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"toteId", req.ToteID,
			"routeId", req.RouteID,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Tote arrival signal sent successfully",
		})
	}
}

// createConsolidationCompletedHandler creates a handler for the consolidation-completed signal endpoint
func createConsolidationCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req ConsolidationCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.OrderID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId is required"}`))
			return
		}

		// Signal the order fulfillment workflow
		workflowID := fmt.Sprintf("order-fulfillment-%s", req.OrderID)

		signalPayload := map[string]interface{}{
			"consolidationId":   req.ConsolidationID,
			"consolidatedItems": req.ConsolidatedItems,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"consolidationCompleted",
			signalPayload,
		)
		if err != nil {
			logger.Error("Failed to signal workflow", "workflowId", workflowID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": workflowID,
			})
			return
		}

		logger.Info("Successfully signaled consolidation completed",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"consolidationId", req.ConsolidationID,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Consolidation completed signal sent successfully",
		})
	}
}

// createGiftWrapCompletedHandler creates a handler for the gift-wrap-completed signal endpoint
func createGiftWrapCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req GiftWrapCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.OrderID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId is required"}`))
			return
		}

		// Signal the gift wrap workflow
		workflowID := fmt.Sprintf("giftwrap-%s", req.OrderID)

		signalPayload := map[string]interface{}{
			"stationId":   req.StationID,
			"wrapType":    req.WrapType,
			"giftMessage": req.GiftMessage,
			"completedAt": req.CompletedAt,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"giftWrapCompleted",
			signalPayload,
		)
		if err != nil {
			logger.Error("Failed to signal workflow", "workflowId", workflowID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": workflowID,
			})
			return
		}

		logger.Info("Successfully signaled gift wrap completed",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"stationId", req.StationID,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Gift wrap completed signal sent successfully",
		})
	}
}

// createWallingCompletedHandler creates a handler for the walling-completed signal endpoint
func createWallingCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req WallingCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.OrderID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"orderId is required"}`))
			return
		}

		// Signal the WES execution workflow
		workflowID := fmt.Sprintf("wes-execution-%s", req.OrderID)

		signalPayload := map[string]interface{}{
			"taskId":      req.TaskID,
			"routeId":     req.RouteID,
			"sortedItems": req.SortedItems,
			"success":     true,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"wallingCompleted",
			signalPayload,
		)
		if err != nil {
			logger.Error("Failed to signal workflow", "workflowId", workflowID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error":      err.Error(),
				"workflowId": workflowID,
			})
			return
		}

		logger.Info("Successfully signaled walling completed",
			"workflowId", workflowID,
			"orderId", req.OrderID,
			"taskId", req.TaskID,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Walling completed signal sent successfully",
		})
	}
}

// createPackingCompletedHandler creates a handler for packing completed signals
func createPackingCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PackingCompletedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode packing completed request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.OrderID == "" || req.TaskID == "" {
			logger.Error("Missing required fields in packing completed request")
			http.Error(w, "Missing orderId or taskId", http.StatusBadRequest)
			return
		}

		// Determine workflow ID - packing can be standalone or part of WES
		// Try packing workflow first, then fall back to WES execution workflow
		workflowID := fmt.Sprintf("packing-%s", req.OrderID)

		// Prepare signal payload
		signalPayload := map[string]interface{}{
			"taskId":      req.TaskID,
			"packageInfo": req.PackageInfo,
		}

		// Send signal to workflow
		ctx := context.Background()
		err := temporalClient.SignalWorkflow(ctx, workflowID, "", "packingCompleted", signalPayload)
		if err != nil {
			logger.Error("Failed to signal packing workflow",
				"orderId", req.OrderID,
				"workflowId", workflowID,
				"error", err,
			)

			// If packing workflow not found, try WES execution workflow
			wesWorkflowID := fmt.Sprintf("wes-%s", req.OrderID)
			err = temporalClient.SignalWorkflow(ctx, wesWorkflowID, "", "packingCompleted", signalPayload)
			if err != nil {
				logger.Error("Failed to signal WES workflow for packing",
					"orderId", req.OrderID,
					"workflowId", wesWorkflowID,
					"error", err,
				)
				http.Error(w, fmt.Sprintf("Failed to send signal: %v", err), http.StatusInternalServerError)
				return
			}
			workflowID = wesWorkflowID
		}

		logger.Info("Packing completed signal sent",
			"orderId", req.OrderID,
			"taskId", req.TaskID,
			"workflowId", workflowID,
		)

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Signal sent successfully",
		})
	}
}

// createReceivingCompletedHandler creates a handler for the receiving-completed signal endpoint
func createReceivingCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req ReceivingCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.ShipmentID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"shipmentId is required"}`))
			return
		}

		// Try to signal the inbound fulfillment workflow
		workflowID := fmt.Sprintf("inbound-fulfillment-%s", req.ShipmentID)

		signalPayload := map[string]interface{}{
			"shipmentId":    req.ShipmentID,
			"receivedItems": req.ReceivedItems,
			"totalReceived": req.TotalReceived,
			"totalDamaged":  req.TotalDamaged,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"receivingCompleted",
			signalPayload,
		)
		if err != nil {
			// Log the error but return success anyway since the receiving was completed
			// The workflow might not exist if this is a standalone receiving operation
			logger.Warn("Failed to signal inbound fulfillment workflow (may not exist)",
				"workflowId", workflowID,
				"shipmentId", req.ShipmentID,
				"error", err,
			)

			// Return success - the receiving completed event is acknowledged
			// even if there's no workflow waiting for it
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    true,
				"workflowId": workflowID,
				"message":    "Receiving completed acknowledged (no active workflow)",
			})
			return
		}

		logger.Info("Successfully signaled receiving completed",
			"workflowId", workflowID,
			"shipmentId", req.ShipmentID,
			"totalReceived", req.TotalReceived,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Receiving completed signal sent successfully",
		})
	}
}

// createStowCompletedHandler creates a handler for the stow-completed signal endpoint
func createStowCompletedHandler(temporalClient temporalclient.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"failed to read request body"}`))
			return
		}
		defer r.Body.Close()

		var req StowCompletedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid JSON"}`))
			return
		}

		if req.ShipmentID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"shipmentId is required"}`))
			return
		}

		// Try to signal the inbound fulfillment workflow
		workflowID := fmt.Sprintf("inbound-fulfillment-%s", req.ShipmentID)

		signalPayload := map[string]interface{}{
			"shipmentId":  req.ShipmentID,
			"stowedItems": req.StowedItems,
			"completedAt": req.CompletedAt,
		}

		err = temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"stowCompleted",
			signalPayload,
		)
		if err != nil {
			// Log the error but return success anyway since the stow was completed
			// The workflow might not exist if this is a standalone stow operation
			logger.Warn("Failed to signal inbound fulfillment workflow for stow (may not exist)",
				"workflowId", workflowID,
				"shipmentId", req.ShipmentID,
				"error", err,
			)

			// Return success - the stow completed event is acknowledged
			// even if there's no workflow waiting for it
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    true,
				"workflowId": workflowID,
				"message":    "Stow completed acknowledged (no active workflow)",
			})
			return
		}

		logger.Info("Successfully signaled stow completed",
			"workflowId", workflowID,
			"shipmentId", req.ShipmentID,
			"stowedCount", len(req.StowedItems),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"workflowId": workflowID,
			"message":    "Stow completed signal sent successfully",
		})
	}
}

// Config holds application configuration
type Config struct {
	Temporal                *temporal.Config
	OrderServiceURL         string
	InventoryServiceURL     string
	RoutingServiceURL       string
	PickingServiceURL       string
	ConsolidationServiceURL string
	PackingServiceURL       string
	ShippingServiceURL      string
	LaborServiceURL         string
	WavingServiceURL        string
	FacilityServiceURL      string
	ReceivingServiceURL     string
	StowServiceURL          string
	SortationServiceURL     string
	UnitServiceURL          string
	ProcessPathServiceURL   string
	BillingServiceURL       string
	ChannelServiceURL       string
	SellerServiceURL        string
}

func loadConfig() *Config {
	return &Config{
		Temporal: &temporal.Config{
			HostPort:  getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			Identity:  "orchestrator-worker",
		},
		OrderServiceURL:         getEnv("ORDER_SERVICE_URL", "http://localhost:8001"),
		InventoryServiceURL:     getEnv("INVENTORY_SERVICE_URL", "http://localhost:8008"),
		RoutingServiceURL:       getEnv("ROUTING_SERVICE_URL", "http://localhost:8003"),
		PickingServiceURL:       getEnv("PICKING_SERVICE_URL", "http://localhost:8004"),
		ConsolidationServiceURL: getEnv("CONSOLIDATION_SERVICE_URL", "http://localhost:8005"),
		PackingServiceURL:       getEnv("PACKING_SERVICE_URL", "http://localhost:8006"),
		ShippingServiceURL:      getEnv("SHIPPING_SERVICE_URL", "http://localhost:8007"),
		LaborServiceURL:         getEnv("LABOR_SERVICE_URL", "http://localhost:8009"),
		WavingServiceURL:        getEnv("WAVING_SERVICE_URL", "http://localhost:8002"),
		FacilityServiceURL:      getEnv("FACILITY_SERVICE_URL", "http://localhost:8011"),
		ReceivingServiceURL:     getEnv("RECEIVING_SERVICE_URL", "http://localhost:8010"),
		StowServiceURL:          getEnv("STOW_SERVICE_URL", "http://localhost:8012"),
		SortationServiceURL:     getEnv("SORTATION_SERVICE_URL", "http://localhost:8013"),
		UnitServiceURL:          getEnv("UNIT_SERVICE_URL", "http://localhost:8014"),
		ProcessPathServiceURL:   getEnv("PROCESS_PATH_SERVICE_URL", "http://localhost:8015"),
		BillingServiceURL:       getEnv("BILLING_SERVICE_URL", "http://localhost:8018"),
		ChannelServiceURL:       getEnv("CHANNEL_SERVICE_URL", "http://localhost:8019"),
		SellerServiceURL:        getEnv("SELLER_SERVICE_URL", "http://localhost:8020"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createReprocessingSchedule creates a Temporal schedule for the reprocessing batch workflow
func createReprocessingSchedule(ctx context.Context, temporalClient client.Client, logger *slog.Logger) error {
	scheduleID := workflows.ReprocessingScheduleID

	// Check if schedule already exists
	scheduleHandle := temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	_, err := scheduleHandle.Describe(ctx)
	if err == nil {
		logger.Info("Reprocessing schedule already exists", "scheduleId", scheduleID)
		return nil
	}

	// Schedule doesn't exist or error checking - try to create it
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "NotFound") {
		// Unexpected error
		return fmt.Errorf("failed to check schedule: %w", err)
	}

	// Create the schedule
	_, err = temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: workflows.ReprocessingBatchInterval,
				},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "reprocessing-batch",
			Workflow:  workflows.ReprocessingBatchWorkflow,
			TaskQueue: temporal.TaskQueues.Orchestrator,
			Args:      []interface{}{workflows.ReprocessingBatchInput{}},
		},
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP, // Skip if previous run still executing
		Paused:  false,
	})

	if err != nil {
		// Check if schedule already exists (race condition)
		if strings.Contains(err.Error(), "already exists") {
			logger.Info("Reprocessing schedule already exists (created by another worker)", "scheduleId", scheduleID)
			return nil
		}
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	logger.Info("Created reprocessing schedule",
		"scheduleId", scheduleID,
		"interval", workflows.ReprocessingBatchInterval.String(),
	)

	return nil
}
