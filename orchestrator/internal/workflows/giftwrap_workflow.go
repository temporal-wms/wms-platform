package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// GiftWrapInput represents input for the gift wrap workflow
type GiftWrapInput struct {
	OrderID     string           `json:"orderId"`
	WaveID      string           `json:"waveId"`
	Items       []GiftWrapItem   `json:"items"`
	WrapDetails GiftWrapDetails  `json:"wrapDetails"`
	StationID   string           `json:"stationId,omitempty"`
}

// GiftWrapItem represents an item to be gift wrapped
type GiftWrapItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// GiftWrapDetails contains gift wrap configuration
type GiftWrapDetails struct {
	WrapType    string `json:"wrapType"`
	GiftMessage string `json:"giftMessage"`
	HidePrice   bool   `json:"hidePrice"`
}

// GiftWrapResult represents the result of gift wrap processing
type GiftWrapResult struct {
	TaskID      string    `json:"taskId"`
	OrderID     string    `json:"orderId"`
	StationID   string    `json:"stationId"`
	WorkerID    string    `json:"workerId"`
	CompletedAt time.Time `json:"completedAt"`
	Success     bool      `json:"success"`
}

// GiftWrapWorkflow coordinates gift wrap processing for orders
func GiftWrapWorkflow(ctx workflow.Context, input map[string]interface{}) (*GiftWrapResult, error) {
	logger := workflow.GetLogger(ctx)

	orderID, _ := input["orderId"].(string)
	waveID, _ := input["waveId"].(string)
	stationID, _ := input["stationId"].(string)

	logger.Info("Starting gift wrap workflow",
		"orderId", orderID,
		"waveId", waveID,
		"stationId", stationID,
	)

	result := &GiftWrapResult{
		OrderID: orderID,
	}

	// Activity options for gift wrap operations
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Find capable station if not provided
	if stationID == "" {
		logger.Info("Finding gift wrap capable station", "orderId", orderID)
		var foundStation map[string]interface{}
		err := workflow.ExecuteActivity(ctx, "FindCapableStation", map[string]interface{}{
			"requirements": []string{"gift_wrap"},
			"stationType":  "packing",
		}).Get(ctx, &foundStation)
		if err != nil {
			return nil, fmt.Errorf("failed to find gift wrap station: %w", err)
		}
		if id, ok := foundStation["stationId"].(string); ok {
			stationID = id
		}
	}
	result.StationID = stationID

	// Step 2: Create gift wrap task
	logger.Info("Creating gift wrap task", "orderId", orderID, "stationId", stationID)
	var taskID string
	err := workflow.ExecuteActivity(ctx, "CreateGiftWrapTask", map[string]interface{}{
		"orderId":     orderID,
		"waveId":      waveID,
		"stationId":   stationID,
		"items":       input["items"],
		"wrapDetails": input["wrapDetails"],
	}).Get(ctx, &taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to create gift wrap task: %w", err)
	}
	result.TaskID = taskID

	// Step 3: Assign worker with gift wrap certification
	logger.Info("Assigning gift wrap worker", "orderId", orderID, "taskId", taskID)
	var workerID string
	err = workflow.ExecuteActivity(ctx, "AssignGiftWrapWorker", map[string]interface{}{
		"taskId":    taskID,
		"stationId": stationID,
	}).Get(ctx, &workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign gift wrap worker: %w", err)
	}
	result.WorkerID = workerID

	// Step 4: Wait for gift wrap to be applied (with timeout)
	logger.Info("Waiting for gift wrap application", "orderId", orderID, "taskId", taskID)
	giftWrapTimeout := 20 * time.Minute
	giftWrapCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	// Create a timer for timeout
	timerFuture := workflow.NewTimer(giftWrapCtx, giftWrapTimeout)

	// Wait for gift wrap completion signal or timeout
	var giftWrapComplete bool
	signalCh := workflow.GetSignalChannel(ctx, "gift-wrap-completed")

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
		var signalData map[string]interface{}
		c.Receive(ctx, &signalData)
		giftWrapComplete = true
		logger.Info("Gift wrap completed signal received", "orderId", orderID, "taskId", taskID)
	})
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		logger.Warn("Gift wrap timeout reached", "orderId", orderID, "taskId", taskID)
	})
	selector.Select(ctx)

	if !giftWrapComplete {
		// Poll for completion if signal wasn't received
		logger.Info("Checking gift wrap task status", "orderId", orderID, "taskId", taskID)
		err = workflow.ExecuteActivity(ctx, "CheckGiftWrapStatus", taskID).Get(ctx, &giftWrapComplete)
		if err != nil {
			return nil, fmt.Errorf("failed to check gift wrap status: %w", err)
		}
		if !giftWrapComplete {
			return nil, fmt.Errorf("gift wrap timed out for order %s", orderID)
		}
	}

	// Step 5: Apply gift message if present
	wrapDetails, hasDetails := input["wrapDetails"].(map[string]interface{})
	if hasDetails {
		if message, ok := wrapDetails["giftMessage"].(string); ok && message != "" {
			logger.Info("Applying gift message", "orderId", orderID, "taskId", taskID)
			err = workflow.ExecuteActivity(ctx, "ApplyGiftMessage", map[string]interface{}{
				"taskId":  taskID,
				"message": message,
			}).Get(ctx, nil)
			if err != nil {
				logger.Warn("Failed to apply gift message, continuing", "orderId", orderID, "error", err)
				// Non-fatal: continue processing
			}
		}
	}

	// Step 6: Complete gift wrap task
	logger.Info("Completing gift wrap task", "orderId", orderID, "taskId", taskID)
	err = workflow.ExecuteActivity(ctx, "CompleteGiftWrapTask", taskID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to complete gift wrap task: %w", err)
	}

	result.Success = true
	result.CompletedAt = workflow.Now(ctx)

	logger.Info("Gift wrap workflow completed successfully",
		"orderId", orderID,
		"taskId", taskID,
		"stationId", stationID,
		"workerId", workerID,
	)

	return result, nil
}
