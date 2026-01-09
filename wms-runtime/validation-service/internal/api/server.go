package api

import (
	"github.com/wms-platform/wms-runtime/validation-service/internal/eventcapture"
	"github.com/wms-platform/wms-runtime/validation-service/internal/validation"
)

// Server represents the API server
type Server struct {
	eventStore         *eventcapture.EventStore
	eventValidator     *validation.EventValidator
	sequenceValidator  *validation.SequenceValidator
	correlationTracker *validation.CorrelationTracker
}

// NewServer creates a new API server
func NewServer(
	eventStore *eventcapture.EventStore,
	eventValidator *validation.EventValidator,
	sequenceValidator *validation.SequenceValidator,
	correlationTracker *validation.CorrelationTracker,
) *Server {
	return &Server{
		eventStore:         eventStore,
		eventValidator:     eventValidator,
		sequenceValidator:  sequenceValidator,
		correlationTracker: correlationTracker,
	}
}
