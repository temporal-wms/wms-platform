package idempotency

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/mongo"
)

// InitializeIndexes creates all required indexes for idempotency functionality
// This should be called during service startup before processing any requests
func InitializeIndexes(ctx context.Context, db *mongo.Database) error {
	slog.Info("Initializing idempotency indexes...")

	// Initialize key repository indexes
	keyRepo := NewMongoKeyRepository(db)
	if err := keyRepo.EnsureIndexes(ctx); err != nil {
		slog.Error("Failed to create idempotency_keys indexes", "error", err)
		return err
	}
	slog.Info("Created indexes for idempotency_keys collection")

	// Initialize message repository indexes
	msgRepo := NewMongoMessageRepository(db)
	if err := msgRepo.EnsureIndexes(ctx); err != nil {
		slog.Error("Failed to create processed_messages indexes", "error", err)
		return err
	}
	slog.Info("Created indexes for processed_messages collection")

	slog.Info("All idempotency indexes initialized successfully")
	return nil
}

// InitializeKeyIndexes creates indexes for the idempotency_keys collection only
func InitializeKeyIndexes(ctx context.Context, db *mongo.Database) error {
	slog.Info("Initializing idempotency_keys indexes...")

	keyRepo := NewMongoKeyRepository(db)
	if err := keyRepo.EnsureIndexes(ctx); err != nil {
		slog.Error("Failed to create idempotency_keys indexes", "error", err)
		return err
	}

	slog.Info("Idempotency_keys indexes initialized successfully")
	return nil
}

// InitializeMessageIndexes creates indexes for the processed_messages collection only
func InitializeMessageIndexes(ctx context.Context, db *mongo.Database) error {
	slog.Info("Initializing processed_messages indexes...")

	msgRepo := NewMongoMessageRepository(db)
	if err := msgRepo.EnsureIndexes(ctx); err != nil {
		slog.Error("Failed to create processed_messages indexes", "error", err)
		return err
	}

	slog.Info("Processed_messages indexes initialized successfully")
	return nil
}
