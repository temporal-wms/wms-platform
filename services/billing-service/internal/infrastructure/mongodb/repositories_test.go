package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wms-platform/services/billing-service/internal/domain"
)

func TestBillableActivityBuildFilter(t *testing.T) {
	repo := &BillableActivityRepository{}
	sellerID := "SLR-001"
	activityType := domain.ActivityTypePick
	invoiced := true

	filter := domain.ActivityFilter{
		SellerID: &sellerID,
		Type:     &activityType,
		Invoiced: &invoiced,
	}

	mongoFilter := repo.buildFilter(filter)
	assert.Equal(t, sellerID, mongoFilter["sellerId"])
	assert.Equal(t, activityType, mongoFilter["type"])
	assert.Equal(t, invoiced, mongoFilter["invoiced"])
}

func TestInvoiceBuildFilter(t *testing.T) {
	repo := &InvoiceRepository{}
	sellerID := "SLR-001"
	status := domain.InvoiceStatusPaid

	filter := domain.InvoiceFilter{
		SellerID: &sellerID,
		Status:   &status,
	}

	mongoFilter := repo.buildFilter(filter)
	assert.Equal(t, sellerID, mongoFilter["sellerId"])
	assert.Equal(t, status, mongoFilter["status"])
}
