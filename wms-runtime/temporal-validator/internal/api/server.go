package api

import (
	"github.com/wms-platform/wms-runtime/temporal-validator/internal/workflow"
)

// Server represents the API server
type Server struct {
	stateMonitor  *workflow.StateMonitor
	signalTracker *workflow.SignalTracker
}

// NewServer creates a new API server
func NewServer(stateMonitor *workflow.StateMonitor, signalTracker *workflow.SignalTracker) *Server {
	return &Server{
		stateMonitor:  stateMonitor,
		signalTracker: signalTracker,
	}
}
