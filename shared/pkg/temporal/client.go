package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Config holds Temporal client configuration
type Config struct {
	HostPort  string
	Namespace string
	Identity  string

	// TLS settings
	TLSEnabled  bool
	TLSCertPath string
	TLSKeyPath  string
	TLSCAPath   string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		HostPort:   "localhost:7233",
		Namespace:  "default",
		Identity:   "wms-worker",
		TLSEnabled: false,
	}
}

// TaskQueues contains all WMS Temporal task queue names
var TaskQueues = struct {
	OrderManagement string
	Waving          string
	Routing         string
	Picking         string
	Consolidation   string
	Packing         string
	Shipping        string
	Inventory       string
	Labor           string
	Orchestrator    string
}{
	OrderManagement: "order-management-queue",
	Waving:          "waving-queue",
	Routing:         "routing-queue",
	Picking:         "picking-queue",
	Consolidation:   "consolidation-queue",
	Packing:         "packing-queue",
	Shipping:        "shipping-queue",
	Inventory:       "inventory-queue",
	Labor:           "labor-queue",
	Orchestrator:    "orchestrator-queue",
}

// WorkflowNames contains all WMS workflow names
var WorkflowNames = struct {
	OrderFulfillment   string
	OrderValidation    string
	OrderCancellation  string
	WavePlanning       string
	ContinuousWaving   string
	RouteCalculation   string
	Picking            string
	Consolidation      string
	Packing            string
	ShipmentProcessing string
	CarrierSelection   string
	Manifesting        string
	InventoryReceiving string
	CycleCount         string
	Replenishment      string
	LaborScheduling    string
	TaskAssignment     string
}{
	OrderFulfillment:   "OrderFulfillmentWorkflow",
	OrderValidation:    "OrderValidationWorkflow",
	OrderCancellation:  "OrderCancellationWorkflow",
	WavePlanning:       "WavePlanningWorkflow",
	ContinuousWaving:   "ContinuousWavingWorkflow",
	RouteCalculation:   "RouteCalculationWorkflow",
	Picking:            "PickingWorkflow",
	Consolidation:      "ConsolidationWorkflow",
	Packing:            "PackingWorkflow",
	ShipmentProcessing: "ShipmentProcessingWorkflow",
	CarrierSelection:   "CarrierSelectionWorkflow",
	Manifesting:        "ManifestingWorkflow",
	InventoryReceiving: "InventoryReceivingWorkflow",
	CycleCount:         "CycleCountWorkflow",
	Replenishment:      "ReplenishmentWorkflow",
	LaborScheduling:    "LaborSchedulingWorkflow",
	TaskAssignment:     "TaskAssignmentWorkflow",
}

// Client wraps the Temporal client with WMS-specific functionality
type Client struct {
	client client.Client
	config *Config
}

// NewClient creates a new Temporal client
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	options := client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
		Identity:  config.Identity,
	}

	c, err := client.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return &Client{
		client: c,
		config: config,
	}, nil
}

// Client returns the underlying Temporal client
func (c *Client) Client() client.Client {
	return c.client
}

// Close closes the client connection
func (c *Client) Close() {
	c.client.Close()
}

// StartWorkflow starts a workflow execution
func (c *Client) StartWorkflow(
	ctx context.Context,
	workflowID string,
	taskQueue string,
	workflowName string,
	args ...interface{},
) (client.WorkflowRun, error) {
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}

	return c.client.ExecuteWorkflow(ctx, options, workflowName, args...)
}

// StartWorkflowWithOptions starts a workflow with custom options
func (c *Client) StartWorkflowWithOptions(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflowName string,
	args ...interface{},
) (client.WorkflowRun, error) {
	return c.client.ExecuteWorkflow(ctx, options, workflowName, args...)
}

// SignalWorkflow sends a signal to a running workflow
func (c *Client) SignalWorkflow(
	ctx context.Context,
	workflowID string,
	runID string,
	signalName string,
	arg interface{},
) error {
	return c.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)
}

// QueryWorkflow queries a workflow
func (c *Client) QueryWorkflow(
	ctx context.Context,
	workflowID string,
	runID string,
	queryType string,
	args ...interface{},
) (interface{}, error) {
	resp, err := c.client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := resp.Get(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// CancelWorkflow cancels a running workflow
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	return c.client.CancelWorkflow(ctx, workflowID, runID)
}

// TerminateWorkflow terminates a running workflow
func (c *Client) TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string) error {
	return c.client.TerminateWorkflow(ctx, workflowID, runID, reason)
}

// GetWorkflowHistory returns the history of a workflow
func (c *Client) GetWorkflowHistory(
	ctx context.Context,
	workflowID string,
	runID string,
) client.HistoryEventIterator {
	return c.client.GetWorkflowHistory(ctx, workflowID, runID, false, 0)
}

// WorkerOptions contains options for creating a worker
type WorkerOptions struct {
	TaskQueue                    string
	MaxConcurrentActivityPollers int
	MaxConcurrentWorkflowPollers int
	MaxConcurrentActivities      int
	MaxConcurrentWorkflows       int
}

// DefaultWorkerOptions returns default worker options
func DefaultWorkerOptions(taskQueue string) *WorkerOptions {
	return &WorkerOptions{
		TaskQueue:                    taskQueue,
		MaxConcurrentActivityPollers: 4,
		MaxConcurrentWorkflowPollers: 4,
		MaxConcurrentActivities:      100,
		MaxConcurrentWorkflows:       100,
	}
}

// NewWorker creates a new Temporal worker
func (c *Client) NewWorker(opts *WorkerOptions) worker.Worker {
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     opts.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: opts.MaxConcurrentWorkflows,
		MaxConcurrentActivityTaskPollers:       opts.MaxConcurrentActivityPollers,
		MaxConcurrentWorkflowTaskPollers:       opts.MaxConcurrentWorkflowPollers,
	}

	return worker.New(c.client, opts.TaskQueue, workerOpts)
}

// DefaultActivityOptions returns default activity options
func DefaultActivityOptions() ActivityOptions {
	return ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
}

// ActivityOptions represents activity execution options
type ActivityOptions struct {
	StartToCloseTimeout    time.Duration
	ScheduleToCloseTimeout time.Duration
	HeartbeatTimeout       time.Duration
	RetryPolicy            RetryPolicy
}

// RetryPolicy represents a retry policy for activities
type RetryPolicy struct {
	InitialInterval    time.Duration
	BackoffCoefficient float64
	MaximumInterval    time.Duration
	MaximumAttempts    int32
}

// ChildWorkflowOptions represents options for child workflows
type ChildWorkflowOptions struct {
	WorkflowID         string
	TaskQueue          string
	ExecutionTimeout   time.Duration
	RetryPolicy        *RetryPolicy
	ParentClosePolicy  string // TERMINATE, ABANDON, REQUEST_CANCEL
}

// DefaultChildWorkflowOptions returns default child workflow options
func DefaultChildWorkflowOptions(workflowID string, taskQueue string) *ChildWorkflowOptions {
	return &ChildWorkflowOptions{
		WorkflowID:        workflowID,
		TaskQueue:         taskQueue,
		ExecutionTimeout:  24 * time.Hour,
		ParentClosePolicy: "TERMINATE",
	}
}
