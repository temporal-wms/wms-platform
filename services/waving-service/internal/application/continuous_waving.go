package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wms-platform/waving-service/internal/domain"
)

// ContinuousWavingService handles waveless/continuous order release
type ContinuousWavingService struct {
	waveRepo       domain.WaveRepository
	orderService   domain.OrderService
	eventPublisher domain.EventPublisher
	config         ContinuousWavingConfig
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
}

// ContinuousWavingConfig configuration for continuous waving
type ContinuousWavingConfig struct {
	// ReleaseInterval is how often to check for orders to release
	ReleaseInterval time.Duration `json:"releaseInterval"`

	// BatchSize is the maximum number of orders to release at once
	BatchSize int `json:"batchSize"`

	// MinOrdersForRelease is the minimum orders needed before releasing
	MinOrdersForRelease int `json:"minOrdersForRelease"`

	// MaxWaitTime is the maximum time an order can wait before forced release
	MaxWaitTime time.Duration `json:"maxWaitTime"`

	// PriorityThreshold orders with priority <= this are immediately released
	PriorityThreshold int `json:"priorityThreshold"`

	// Zone to process (empty for all zones)
	Zone string `json:"zone,omitempty"`
}

// DefaultContinuousWavingConfig returns default configuration
func DefaultContinuousWavingConfig() ContinuousWavingConfig {
	return ContinuousWavingConfig{
		ReleaseInterval:     1 * time.Minute,
		BatchSize:           50,
		MinOrdersForRelease: 5,
		MaxWaitTime:         15 * time.Minute,
		PriorityThreshold:   2, // Same-day and next-day
		Zone:                "",
	}
}

// NewContinuousWavingService creates a new continuous waving service
func NewContinuousWavingService(
	waveRepo domain.WaveRepository,
	orderService domain.OrderService,
	eventPublisher domain.EventPublisher,
	config ContinuousWavingConfig,
) *ContinuousWavingService {
	return &ContinuousWavingService{
		waveRepo:       waveRepo,
		orderService:   orderService,
		eventPublisher: eventPublisher,
		config:         config,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the continuous waving process
func (s *ContinuousWavingService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("continuous waving service is already running")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx)
	return nil
}

// Stop stops the continuous waving process
func (s *ContinuousWavingService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopChan)
		s.running = false
	}
}

// IsRunning returns whether the service is running
func (s *ContinuousWavingService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main loop for continuous waving
func (s *ContinuousWavingService) run(ctx context.Context) {
	ticker := time.NewTicker(s.config.ReleaseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.processOrders(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Error in continuous waving: %v\n", err)
			}
		}
	}
}

// processOrders checks and releases orders
func (s *ContinuousWavingService) processOrders(ctx context.Context) error {
	// Get orders ready for waving
	filter := domain.OrderFilter{
		Zone:  []string{s.config.Zone},
		Limit: s.config.BatchSize,
	}

	orders, err := s.orderService.GetOrdersReadyForWaving(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get orders: %w", err)
	}

	if len(orders) == 0 {
		return nil
	}

	// Categorize orders
	var immediateOrders []domain.WaveOrder
	var batchOrders []domain.WaveOrder

	for _, order := range orders {
		priorityValue := getPriorityValue(order.Priority)
		if priorityValue <= s.config.PriorityThreshold {
			immediateOrders = append(immediateOrders, order)
		} else {
			batchOrders = append(batchOrders, order)
		}
	}

	// Release immediate orders right away
	if len(immediateOrders) > 0 {
		if err := s.releaseOrders(ctx, immediateOrders, "immediate"); err != nil {
			return fmt.Errorf("failed to release immediate orders: %w", err)
		}
	}

	// Release batch orders if threshold is met
	if len(batchOrders) >= s.config.MinOrdersForRelease {
		if err := s.releaseOrders(ctx, batchOrders, "batch"); err != nil {
			return fmt.Errorf("failed to release batch orders: %w", err)
		}
	}

	return nil
}

// releaseOrders creates a micro-wave and releases orders
func (s *ContinuousWavingService) releaseOrders(ctx context.Context, orders []domain.WaveOrder, releaseType string) error {
	// Create a micro-wave for these orders
	waveID := fmt.Sprintf("WV-CONT-%s-%d", releaseType, time.Now().UnixNano()%100000)

	config := domain.WaveConfiguration{
		MaxOrders:   len(orders),
		AutoRelease: true,
	}

	wave, err := domain.NewWave(waveID, domain.WaveTypeDigital, domain.FulfillmentModeWaveless, config)
	if err != nil {
		return err
	}

	// Add orders to wave
	for _, order := range orders {
		if err := wave.AddOrder(order); err != nil {
			continue // Skip orders that can't be added
		}
	}

	if wave.GetOrderCount() == 0 {
		return nil
	}

	// Schedule and immediately release
	now := time.Now()
	if err := wave.Schedule(now, now.Add(2*time.Hour)); err != nil {
		return err
	}

	if err := wave.Release(); err != nil {
		return err
	}

	// Save wave
	if err := s.waveRepo.Save(ctx, wave); err != nil {
		return err
	}

	// Notify order service of wave assignments
	for _, order := range wave.Orders {
		if err := s.orderService.NotifyWaveAssignment(ctx, order.OrderID, wave.WaveID, now); err != nil {
			// Log but continue
			fmt.Printf("Failed to notify wave assignment for order %s: %v\n", order.OrderID, err)
		}
	}

	// Publish events
	if err := s.eventPublisher.PublishAll(ctx, wave.GetDomainEvents()); err != nil {
		return err
	}

	return nil
}

// ProcessSingleOrder immediately processes a single high-priority order
func (s *ContinuousWavingService) ProcessSingleOrder(ctx context.Context, order domain.WaveOrder) error {
	waveID := fmt.Sprintf("WV-SINGLE-%d", time.Now().UnixNano()%100000)

	config := domain.WaveConfiguration{
		MaxOrders:   1,
		AutoRelease: true,
	}

	wave, err := domain.NewWave(waveID, domain.WaveTypePriority, domain.FulfillmentModeWaveless, config)
	if err != nil {
		return err
	}

	if err := wave.AddOrder(order); err != nil {
		return err
	}

	now := time.Now()
	if err := wave.Schedule(now, now.Add(1*time.Hour)); err != nil {
		return err
	}

	if err := wave.Release(); err != nil {
		return err
	}

	if err := s.waveRepo.Save(ctx, wave); err != nil {
		return err
	}

	// Notify order service
	if err := s.orderService.NotifyWaveAssignment(ctx, order.OrderID, wave.WaveID, now); err != nil {
		return err
	}

	// Publish events
	return s.eventPublisher.PublishAll(ctx, wave.GetDomainEvents())
}

// getPriorityValue converts priority string to numeric value
func getPriorityValue(priority string) int {
	switch priority {
	case "same_day":
		return 1
	case "next_day":
		return 2
	default:
		return 3
	}
}
