package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
)

func TestRepositoryConstructors(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("billable activity", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewBillableActivityRepository(mt.DB)
		require.NotNil(t, repo)
	})

	mt.Run("storage calculation", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		repo := NewStorageCalculationRepository(mt.DB)
		require.NotNil(t, repo)
	})

	mt.Run("invoice", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(), // invoices indexes
			mtest.CreateSuccessResponse(), // outbox indexes
		)
		repo := NewInvoiceRepository(mt.DB, cloudevents.NewEventFactory("/billing-service"))
		require.NotNil(t, repo)
	})
}

func TestBillableActivityRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("save and list", func(mt *mtest.T) {
		coll := mt.DB.Collection("billable_activities")
		repo := &BillableActivityRepository{
			collection:   coll,
			tenantHelper: tenant.NewRepositoryHelper(false),
		}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err := repo.Save(ctx, &domain.BillableActivity{ActivityID: "ACT-001"})
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.SaveAll(ctx, []*domain.BillableActivity{
			{ActivityID: "ACT-001"},
			{ActivityID: "ACT-002"},
		})
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "activityId", Value: "ACT-001"},
			{Key: "type", Value: "pick"},
		}))
		activity, err := repo.FindByID(ctx, "ACT-001")
		require.NoError(t, err)
		require.NotNil(t, activity)
		assert.Equal(t, "ACT-001", activity.ActivityID)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "activityId", Value: "ACT-002"},
			{Key: "type", Value: "pack"},
		}))
		list, err := repo.FindBySellerID(ctx, "SLR-001", domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "activityId", Value: "ACT-003"},
			{Key: "type", Value: "shipping"},
		}))
		list, err = repo.FindUninvoiced(ctx, "SLR-001", time.Now().Add(-24*time.Hour), time.Now())
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "activityId", Value: "ACT-004"},
			{Key: "type", Value: "storage"},
		}))
		list, err = repo.FindByInvoiceID(ctx, "INV-001")
		require.NoError(t, err)
		require.Len(t, list, 1)

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err = repo.MarkAsInvoiced(ctx, []string{"ACT-001"}, "INV-001")
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "pick"},
			{Key: "total", Value: 12.5},
		}))
		sums, err := repo.SumBySellerAndType(ctx, "SLR-001", time.Now().Add(-24*time.Hour), time.Now())
		require.NoError(t, err)
		assert.Equal(t, 12.5, sums[domain.ActivityTypePick])

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "n", Value: int64(3)},
		}))
		count, err := repo.Count(ctx, domain.ActivityFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

func TestInvoiceRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("find and update", func(mt *mtest.T) {
		coll := mt.DB.Collection("invoices")
		repo := &InvoiceRepository{
			collection:   coll,
			db:           mt.DB,
			outboxRepo:   outboxMongo.NewOutboxRepository(mt.DB),
			eventFactory: cloudevents.NewEventFactory("/billing-service"),
			tenantHelper: tenant.NewRepositoryHelper(false),
		}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "invoiceId", Value: "INV-001"},
			{Key: "status", Value: "draft"},
		}))
		inv, err := repo.FindByID(ctx, "INV-001")
		require.NoError(t, err)
		require.NotNil(t, inv)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "invoiceId", Value: "INV-002"},
			{Key: "status", Value: "paid"},
		}))
		invoices, err := repo.FindBySellerID(ctx, "SLR-001", domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, invoices, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "invoiceId", Value: "INV-003"},
			{Key: "status", Value: "paid"},
		}))
		invoices, err = repo.FindByStatus(ctx, domain.InvoiceStatusPaid, domain.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, invoices, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "invoiceId", Value: "INV-004"},
			{Key: "status", Value: "finalized"},
		}))
		invoices, err = repo.FindOverdue(ctx)
		require.NoError(t, err)
		require.Len(t, invoices, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "invoiceId", Value: "INV-005"},
			{Key: "status", Value: "draft"},
		}))
		inv, err = repo.FindByPeriod(ctx, "SLR-001", time.Now().Add(-24*time.Hour), time.Now())
		require.NoError(t, err)
		require.NotNil(t, inv)

		mt.AddMockResponses(mtest.CreateSuccessResponse(
			bson.E{Key: "n", Value: 1},
			bson.E{Key: "nModified", Value: 1},
		))
		err = repo.UpdateStatus(ctx, "INV-001", domain.InvoiceStatusPaid)
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "n", Value: int64(4)},
		}))
		count, err := repo.Count(ctx, domain.InvoiceFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})
}

func TestInvoiceRepository_SaveTransaction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("save", func(mt *mtest.T) {
		coll := mt.DB.Collection("invoices")
		repo := &InvoiceRepository{
			collection:   coll,
			db:           mt.DB,
			outboxRepo:   outboxMongo.NewOutboxRepository(mt.DB),
			eventFactory: cloudevents.NewEventFactory("/billing-service"),
			tenantHelper: tenant.NewRepositoryHelper(false),
		}

		inv := domain.NewInvoice("TNT-001", "SLR-001", time.Now().Add(-24*time.Hour), time.Now(), "Acme", "billing@acme.com")

		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(), // outbox insertMany
			mtest.CreateSuccessResponse(), // commitTransaction
		)

		err := repo.Save(context.Background(), inv)
		require.NoError(t, err)
		assert.Empty(t, inv.DomainEvents())
	})
}

func TestStorageCalculationRepository_MockOps(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("storage operations", func(mt *mtest.T) {
		coll := mt.DB.Collection("storage_calculations")
		repo := &StorageCalculationRepository{
			collection:   coll,
			tenantHelper: tenant.NewRepositoryHelper(false),
		}
		ctx := context.Background()
		ns := coll.Database().Name() + "." + coll.Name()

		mt.AddMockResponses(mtest.CreateSuccessResponse())
		err := repo.Save(ctx, &domain.StorageCalculation{
			SellerID:        "SLR-001",
			CalculationDate: time.Now().Add(-24 * time.Hour),
		})
		require.NoError(t, err)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "calculationId", Value: "STO-001"},
			{Key: "sellerId", Value: "SLR-001"},
		}))
		calc, err := repo.FindBySellerAndDate(ctx, "SLR-001", time.Now().Add(-24*time.Hour))
		require.NoError(t, err)
		require.NotNil(t, calc)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "calculationId", Value: "STO-002"},
			{Key: "sellerId", Value: "SLR-001"},
		}))
		calcs, err := repo.FindBySellerAndPeriod(ctx, "SLR-001", time.Now().Add(-48*time.Hour), time.Now())
		require.NoError(t, err)
		require.Len(t, calcs, 1)

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "total", Value: 42.5},
		}))
		sum, err := repo.SumByPeriod(ctx, "SLR-001", time.Now().Add(-48*time.Hour), time.Now())
		require.NoError(t, err)
		assert.Equal(t, 42.5, sum)
	})
}
