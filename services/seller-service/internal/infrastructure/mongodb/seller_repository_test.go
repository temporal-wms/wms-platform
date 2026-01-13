package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wms-platform/services/seller-service/internal/domain"
	cloudevents "github.com/wms-platform/shared/pkg/cloudevents"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)

	db := client.Database("test_sellers")

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.Drop(ctx)
		client.Disconnect(ctx)
	}

	return db, cleanup
}

func TestNewSellerRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.collection)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.outboxRepo)
	assert.NotNil(t, repo.eventFactory)
}

func TestSellerRepository_Save(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, err := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	require.NoError(t, err)
	seller.Status = domain.SellerStatusActive

	err = repo.Save(ctx, seller)
	assert.NoError(t, err)

	var saved domain.Seller
	filter := bson.M{"sellerId": seller.SellerID}
	err = db.Collection("sellers").FindOne(ctx, filter).Decode(&saved)
	assert.NoError(t, err)
	assert.Equal(t, seller.SellerID, saved.SellerID)
	assert.Equal(t, seller.CompanyName, saved.CompanyName)
}

func TestSellerRepository_Save_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, err := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	require.NoError(t, err)

	err = repo.Save(ctx, seller)
	assert.NoError(t, err)

	seller.CompanyName = "Updated Corp"
	seller.ContactPhone = "555-9999"

	err = repo.Save(ctx, seller)
	assert.NoError(t, err)

	var saved domain.Seller
	filter := bson.M{"sellerId": seller.SellerID}
	err = db.Collection("sellers").FindOne(ctx, filter).Decode(&saved)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Corp", saved.CompanyName)
	assert.Equal(t, "555-9999", saved.ContactPhone)
}

func TestSellerRepository_FindByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, err := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	require.NoError(t, err)

	err = repo.Save(ctx, seller)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, seller.SellerID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, seller.SellerID, found.SellerID)
	assert.Equal(t, seller.CompanyName, found.CompanyName)

	notFound, err := repo.FindByID(ctx, "NONEXISTENT")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestSellerRepository_FindByTenantID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()

	seller1, _ := domain.NewSeller("TNT-001", "Corp A", "John", "john@a.com", domain.BillingCycleMonthly)
	seller2, _ := domain.NewSeller("TNT-001", "Corp B", "Jane", "jane@b.com", domain.BillingCycleMonthly)
	seller3, _ := domain.NewSeller("TNT-002", "Corp C", "Bob", "bob@c.com", domain.BillingCycleMonthly)

	repo.Save(ctx, seller1)
	repo.Save(ctx, seller2)
	repo.Save(ctx, seller3)

	results, err := repo.FindByTenantID(ctx, "TNT-001", domain.Pagination{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	results2, err := repo.FindByTenantID(ctx, "TNT-002", domain.Pagination{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.Len(t, results2, 1)
}

func TestSellerRepository_FindByStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()

	seller1, _ := domain.NewSeller("TNT-001", "Corp A", "John", "john@a.com", domain.BillingCycleMonthly)
	seller1.Status = domain.SellerStatusActive

	seller2, _ := domain.NewSeller("TNT-001", "Corp B", "Jane", "jane@b.com", domain.BillingCycleMonthly)
	seller2.Status = domain.SellerStatusSuspended

	seller3, _ := domain.NewSeller("TNT-001", "Corp C", "Bob", "bob@c.com", domain.BillingCycleMonthly)
	seller3.Status = domain.SellerStatusActive

	repo.Save(ctx, seller1)
	repo.Save(ctx, seller2)
	repo.Save(ctx, seller3)

	results, err := repo.FindByStatus(ctx, domain.SellerStatusActive, domain.Pagination{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	for _, s := range results {
		assert.Equal(t, domain.SellerStatusActive, s.Status)
	}
}

func TestSellerRepository_FindByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, _ := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)

	err := repo.Save(ctx, seller)
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, "john@test.com")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, seller.SellerID, found.SellerID)

	notFound, err := repo.FindByEmail(ctx, "nonexistent@test.com")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestSellerRepository_FindByAPIKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, _ := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)
	seller.Status = domain.SellerStatusActive

	_, rawKey, _ := seller.GenerateAPIKey("Test Key", []string{"orders:read"}, nil)
	assert.NotEmpty(t, rawKey)

	repo.Save(ctx, seller)

	found, err := repo.FindByAPIKey(ctx, seller.APIKeys[0].HashedKey)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, seller.SellerID, found.SellerID)

	notFound, err := repo.FindByAPIKey(ctx, "nonexistentkey")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestSellerRepository_UpdateStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, _ := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)

	repo.Save(ctx, seller)

	err := repo.UpdateStatus(ctx, seller.SellerID, domain.SellerStatusActive)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, seller.SellerID)
	assert.NoError(t, err)
	assert.Equal(t, domain.SellerStatusActive, found.Status)
}

func TestSellerRepository_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()

	seller1, _ := domain.NewSeller("TNT-001", "Corp A", "John", "john@a.com", domain.BillingCycleMonthly)
	seller2, _ := domain.NewSeller("TNT-001", "Corp B", "Jane", "jane@b.com", domain.BillingCycleMonthly)
	seller3, _ := domain.NewSeller("TNT-002", "Corp C", "Bob", "bob@c.com", domain.BillingCycleMonthly)

	repo.Save(ctx, seller1)
	repo.Save(ctx, seller2)
	repo.Save(ctx, seller3)

	count, err := repo.Count(ctx, domain.SellerFilter{})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)

	filter := domain.SellerFilter{}
	status := domain.SellerStatusActive
	filter.Status = &status

	count2, err := repo.Count(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count2)
}

func TestSellerRepository_Search(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()

	seller1, _ := domain.NewSeller("TNT-001", "Acme Corporation", "John", "john@acme.com", domain.BillingCycleMonthly)
	seller2, _ := domain.NewSeller("TNT-001", "Beta Industries", "Jane", "jane@beta.com", domain.BillingCycleMonthly)
	seller3, _ := domain.NewSeller("TNT-001", "Gamma Corp", "Bob", "bob@gamma.com", domain.BillingCycleMonthly)

	repo.Save(ctx, seller1)
	repo.Save(ctx, seller2)
	repo.Save(ctx, seller3)

	results, err := repo.Search(ctx, "Acme", domain.Pagination{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSellerRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()
	seller, _ := domain.NewSeller("TNT-001", "Test Corp", "John Doe", "john@test.com", domain.BillingCycleMonthly)

	repo.Save(ctx, seller)

	err := repo.Delete(ctx, seller.SellerID)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, seller.SellerID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, domain.SellerStatusClosed, found.Status)
}

func TestSellerRepository_BuildFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	eventFactory := cloudevents.NewEventFactory("/test")
	repo := NewSellerRepository(db, eventFactory)

	ctx := context.Background()

	seller1, _ := domain.NewSeller("TNT-001", "Corp A", "John", "john@a.com", domain.BillingCycleMonthly)
	seller2, _ := domain.NewSeller("TNT-002", "Corp B", "Jane", "jane@b.com", domain.BillingCycleMonthly)
	seller2.Status = domain.SellerStatusSuspended

	repo.Save(ctx, seller1)
	repo.Save(ctx, seller2)

	filter := domain.SellerFilter{}
	filter.TenantID = strPtr("TNT-001")

	count, err := repo.Count(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	filter2 := domain.SellerFilter{}
	status := domain.SellerStatusSuspended
	filter2.Status = &status

	count2, err := repo.Count(ctx, filter2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count2)
}

func strPtr(s string) *string {
	return &s
}
