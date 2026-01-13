package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wms-platform/wes-service/internal/domain"
)

func TestRouteEvents(t *testing.T) {
	now := time.Now()

	created := &domain.RouteCreatedEvent{
		RouteID:        "ROUTE-1",
		OrderID:        "ORDER-1",
		WaveID:         "WAVE-1",
		PathTemplateID: "TPL",
		PathType:       "pick_pack",
		StageCount:     2,
		CreatedAt:      now,
	}
	require.Equal(t, "wms.wes.route-created", created.EventType())
	require.Equal(t, now, created.OccurredAt())

	completed := &domain.RouteCompletedEvent{
		RouteID:     "ROUTE-1",
		OrderID:     "ORDER-1",
		PathType:    "pick_pack",
		StageCount:  2,
		CompletedAt: now,
	}
	require.Equal(t, "wms.wes.route-completed", completed.EventType())
	require.Equal(t, now, completed.OccurredAt())
}

func TestStageLifecycleEvents(t *testing.T) {
	ts := time.Now().UnixMilli()

	assigned := &domain.StageAssignedEvent{
		RouteID:   "ROUTE-ASSIGN",
		OrderID:   "ORDER-ASSIGN",
		StageType: "picking",
		WorkerID:  "WORKER-1",
		TaskID:    "TASK-1",
		Timestamp: ts,
	}
	require.Equal(t, "wms.wes.stage-assigned", assigned.EventType())
	require.Equal(t, ts, assigned.OccurredAt().UnixMilli())

	started := &domain.StageStartedEvent{
		RouteID:   "ROUTE-START",
		OrderID:   "ORDER-START",
		StageType: "picking",
		TaskID:    "TASK-START",
		WorkerID:  "WORKER-1",
		Timestamp: ts,
	}
	require.Equal(t, "wms.wes.stage-started", started.EventType())
	require.Equal(t, ts, started.OccurredAt().UnixMilli())

	completed := &domain.StageCompletedEvent{
		RouteID:   "ROUTE-COMPLETE",
		OrderID:   "ORDER-COMPLETE",
		StageType: "picking",
		TaskID:    "TASK-DONE",
		WorkerID:  "WORKER-1",
		Timestamp: ts,
	}
	require.Equal(t, "wms.wes.stage-completed", completed.EventType())
	require.Equal(t, ts, completed.OccurredAt().UnixMilli())

	failed := &domain.StageFailedEvent{
		RouteID:   "ROUTE-FAIL",
		OrderID:   "ORDER-FAIL",
		StageType: "picking",
		TaskID:    "TASK-FAIL",
		Error:     "boom",
		Timestamp: ts,
	}
	require.Equal(t, "wms.wes.stage-failed", failed.EventType())
	require.Equal(t, ts, failed.OccurredAt().UnixMilli())
}

func TestTemplateCreatedEvent(t *testing.T) {
	now := time.Now()
	event := &domain.TemplateCreatedEvent{
		TemplateID: "TPL-NEW",
		PathType:   "pick_pack",
		Name:       "Test Template",
		StageCount: 2,
		CreatedAt:  now,
	}

	require.Equal(t, "wms.wes.template-created", event.EventType())
	require.Equal(t, now, event.OccurredAt())
}
