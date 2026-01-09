package idempotency

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// PhaseManager handles atomic phases for complex multi-step operations
// It allows tracking progress through operations that involve multiple external calls
// and supports recovery from failures at any point
type PhaseManager struct {
	keyID      string
	repository KeyRepository
}

// NewPhaseManager creates a phase manager from Gin context
// Returns nil if no idempotency key is present in the context
func NewPhaseManager(c *gin.Context, repository KeyRepository) *PhaseManager {
	keyID, exists := c.Get(ContextKeyIDempotencyKeyID)
	if !exists {
		return nil
	}

	keyIDStr, ok := keyID.(string)
	if !ok || keyIDStr == "" {
		return nil
	}

	return &PhaseManager{
		keyID:      keyIDStr,
		repository: repository,
	}
}

// NewPhaseManagerFromContext creates a phase manager from a regular context
// This is useful for operations outside of HTTP handlers
func NewPhaseManagerFromContext(ctx context.Context, keyID string, repository KeyRepository) *PhaseManager {
	if keyID == "" {
		return nil
	}

	return &PhaseManager{
		keyID:      keyID,
		repository: repository,
	}
}

// Checkpoint marks a phase as complete
// This allows the operation to be recovered from this point if it fails later
func (pm *PhaseManager) Checkpoint(ctx context.Context, phase string) error {
	if pm == nil || pm.keyID == "" {
		// No idempotency key, skip checkpoint
		return nil
	}

	slog.Debug("Setting recovery checkpoint",
		"keyId", pm.keyID,
		"phase", phase,
	)

	err := pm.repository.UpdateRecoveryPoint(ctx, pm.keyID, phase)
	if err != nil {
		slog.Error("Failed to set recovery checkpoint",
			"error", err,
			"keyId", pm.keyID,
			"phase", phase,
		)
		return err
	}

	slog.Info("Recovery checkpoint set",
		"keyId", pm.keyID,
		"phase", phase,
	)

	return nil
}

// GetRecoveryPoint retrieves the current recovery point
// Returns empty string if no checkpoint has been set
func (pm *PhaseManager) GetRecoveryPoint(ctx context.Context) (string, error) {
	if pm == nil || pm.keyID == "" {
		return "", nil
	}

	key, err := pm.repository.GetByID(ctx, pm.keyID)
	if err != nil {
		return "", err
	}

	return key.RecoveryPoint, nil
}

// ShouldSkipPhase returns true if the phase has already been completed
// This is used to skip phases when recovering from a failure
func (pm *PhaseManager) ShouldSkipPhase(ctx context.Context, phase string) (bool, error) {
	if pm == nil || pm.keyID == "" {
		return false, nil
	}

	recoveryPoint, err := pm.GetRecoveryPoint(ctx)
	if err != nil {
		return false, err
	}

	// If recovery point is set and matches or is after this phase, skip it
	// This requires phases to be ordered/comparable
	return recoveryPoint == phase, nil
}

// Example usage:
//
// func CreateOrderHandler(c *gin.Context) {
//     phases := idempotency.NewPhaseManager(c, repo)
//
//     // Phase 1: Validate inventory
//     if skip, _ := phases.ShouldSkipPhase(ctx, "inventory_validated"); !skip {
//         if err := validateInventory(order); err != nil {
//             return err
//         }
//         phases.Checkpoint(ctx, "inventory_validated")
//     }
//
//     // Phase 2: Reserve items
//     if skip, _ := phases.ShouldSkipPhase(ctx, "items_reserved"); !skip {
//         if err := reserveItems(order); err != nil {
//             return err
//         }
//         phases.Checkpoint(ctx, "items_reserved")
//     }
//
//     // Phase 3: Create order
//     if skip, _ := phases.ShouldSkipPhase(ctx, "order_created"); !skip {
//         if err := createOrder(order); err != nil {
//             return err
//         }
//         phases.Checkpoint(ctx, "order_created")
//     }
//
//     return nil
// }
