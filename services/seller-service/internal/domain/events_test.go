package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSellerCreatedEvent_EventType(t *testing.T) {
	event := &SellerCreatedEvent{
		SellerID:    "SLR-001",
		TenantID:    "TNT-001",
		CompanyName: "Test Corp",
		CreatedAt:   time.Now().UTC(),
	}

	assert.Equal(t, "seller.created", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestSellerCreatedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &SellerCreatedEvent{
		SellerID:    "SLR-001",
		TenantID:    "TNT-001",
		CompanyName: "Test Corp",
		CreatedAt:   now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestSellerActivatedEvent_EventType(t *testing.T) {
	event := &SellerActivatedEvent{
		SellerID:    "SLR-001",
		ActivatedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.activated", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestSellerActivatedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &SellerActivatedEvent{
		SellerID:    "SLR-001",
		ActivatedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestSellerSuspendedEvent_EventType(t *testing.T) {
	event := &SellerSuspendedEvent{
		SellerID:    "SLR-001",
		Reason:      "Test reason",
		SuspendedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.suspended", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestSellerSuspendedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &SellerSuspendedEvent{
		SellerID:    "SLR-001",
		Reason:      "Test reason",
		SuspendedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestSellerClosedEvent_EventType(t *testing.T) {
	event := &SellerClosedEvent{
		SellerID: "SLR-001",
		Reason:   "Contract ended",
		ClosedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.closed", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestSellerClosedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &SellerClosedEvent{
		SellerID: "SLR-001",
		Reason:   "Contract ended",
		ClosedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestFacilityAssignedEvent_EventType(t *testing.T) {
	event := &FacilityAssignedEvent{
		SellerID:   "SLR-001",
		FacilityID: "FAC-001",
		AssignedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.facility_assigned", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestFacilityAssignedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &FacilityAssignedEvent{
		SellerID:   "SLR-001",
		FacilityID: "FAC-001",
		AssignedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestChannelConnectedEvent_EventType(t *testing.T) {
	event := &ChannelConnectedEvent{
		SellerID:    "SLR-001",
		ChannelID:   "CH-001",
		ChannelType: "shopify",
		ConnectedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.channel_connected", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestChannelConnectedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &ChannelConnectedEvent{
		SellerID:    "SLR-001",
		ChannelID:   "CH-001",
		ChannelType: "shopify",
		ConnectedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestFeeScheduleUpdatedEvent_EventType(t *testing.T) {
	event := &FeeScheduleUpdatedEvent{
		SellerID:  "SLR-001",
		UpdatedAt: time.Now().UTC(),
	}

	assert.Equal(t, "seller.fee_schedule_updated", event.EventType())
	assert.NotEmpty(t, event.EventType())
}

func TestFeeScheduleUpdatedEvent_OccurredAt(t *testing.T) {
	now := time.Now().UTC()
	event := &FeeScheduleUpdatedEvent{
		SellerID:  "SLR-001",
		UpdatedAt: now,
	}

	assert.Equal(t, now, event.OccurredAt())
}

func TestPagination_DefaultPagination(t *testing.T) {
	pag := DefaultPagination()

	assert.Equal(t, int64(1), pag.Page)
	assert.Equal(t, int64(20), pag.PageSize)
}

func TestPagination_Skip(t *testing.T) {
	tests := []struct {
		name     string
		page     int64
		pageSize int64
		want     int64
	}{
		{
			name:     "Page 1, size 20",
			page:     1,
			pageSize: 20,
			want:     0,
		},
		{
			name:     "Page 2, size 20",
			page:     2,
			pageSize: 20,
			want:     20,
		},
		{
			name:     "Page 3, size 10",
			page:     3,
			pageSize: 10,
			want:     20,
		},
		{
			name:     "Page 0, size 20",
			page:     0,
			pageSize: 20,
			want:     -20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pag := Pagination{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}
			assert.Equal(t, tt.want, pag.Skip())
		})
	}
}

func TestPagination_Limit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int64
		want     int64
	}{
		{
			name:     "Size 10",
			pageSize: 10,
			want:     10,
		},
		{
			name:     "Size 20",
			pageSize: 20,
			want:     20,
		},
		{
			name:     "Size 50",
			pageSize: 50,
			want:     50,
		},
		{
			name:     "Size 100",
			pageSize: 100,
			want:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pag := Pagination{
				PageSize: tt.pageSize,
			}
			assert.Equal(t, tt.want, pag.Limit())
		})
	}
}
