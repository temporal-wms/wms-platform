package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/billing-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/kafka"
	"github.com/wms-platform/shared/pkg/outbox"
	outboxMongo "github.com/wms-platform/shared/pkg/outbox/mongodb"
	"github.com/wms-platform/shared/pkg/tenant"
)

// BillableActivityRepository implements domain.BillableActivityRepository
type BillableActivityRepository struct {
	collection   *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewBillableActivityRepository creates a new BillableActivityRepository
func NewBillableActivityRepository(db *mongo.Database) *BillableActivityRepository {
	collection := db.Collection("billable_activities")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "sellerId", Value: 1},
				{Key: "activityDate", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "sellerId", Value: 1},
				{Key: "invoiced", Value: 1},
				{Key: "activityDate", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "invoiceId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "activityId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "referenceType", Value: 1},
				{Key: "referenceId", Value: 1},
			},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &BillableActivityRepository{
		collection:   collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save persists a billable activity
func (r *BillableActivityRepository) Save(ctx context.Context, activity *domain.BillableActivity) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"activityId": activity.ActivityID}
	update := bson.M{"$set": activity}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// SaveAll persists multiple activities
func (r *BillableActivityRepository) SaveAll(ctx context.Context, activities []*domain.BillableActivity) error {
	if len(activities) == 0 {
		return nil
	}

	docs := make([]interface{}, len(activities))
	for i, a := range activities {
		docs[i] = a
	}

	_, err := r.collection.InsertMany(ctx, docs)
	return err
}

// FindByID retrieves an activity by ID
func (r *BillableActivityRepository) FindByID(ctx context.Context, activityID string) (*domain.BillableActivity, error) {
	var activity domain.BillableActivity
	filter := bson.M{"activityId": activityID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&activity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &activity, nil
}

// FindBySellerID retrieves activities for a seller
func (r *BillableActivityRepository) FindBySellerID(ctx context.Context, sellerID string, pagination domain.Pagination) ([]*domain.BillableActivity, error) {
	filter := bson.M{"sellerId": sellerID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "activityDate", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindUninvoiced retrieves activities not yet invoiced for a seller
func (r *BillableActivityRepository) FindUninvoiced(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) ([]*domain.BillableActivity, error) {
	filter := bson.M{
		"sellerId": sellerID,
		"invoiced": false,
		"activityDate": bson.M{
			"$gte": periodStart,
			"$lte": periodEnd,
		},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().SetSort(bson.D{{Key: "activityDate", Value: 1}})
	return r.findMany(ctx, filter, opts)
}

// FindByInvoiceID retrieves activities for an invoice
func (r *BillableActivityRepository) FindByInvoiceID(ctx context.Context, invoiceID string) ([]*domain.BillableActivity, error) {
	filter := bson.M{"invoiceId": invoiceID}
	opts := options.Find().SetSort(bson.D{{Key: "type", Value: 1}})
	return r.findMany(ctx, filter, opts)
}

// MarkAsInvoiced marks activities as invoiced
func (r *BillableActivityRepository) MarkAsInvoiced(ctx context.Context, activityIDs []string, invoiceID string) error {
	filter := bson.M{"activityId": bson.M{"$in": activityIDs}}
	update := bson.M{
		"$set": bson.M{
			"invoiced":  true,
			"invoiceId": invoiceID,
		},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

// SumBySellerAndType returns sum of amounts by activity type for a seller
func (r *BillableActivityRepository) SumBySellerAndType(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (map[domain.ActivityType]float64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"sellerId": sellerID,
				"activityDate": bson.M{
					"$gte": periodStart,
					"$lte": periodEnd,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$type",
				"total": bson.M{"$sum": "$amount"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[domain.ActivityType]float64)
	for cursor.Next(ctx) {
		var row struct {
			ID    domain.ActivityType `bson:"_id"`
			Total float64             `bson:"total"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		result[row.ID] = row.Total
	}

	return result, nil
}

// Count returns total count matching filter
func (r *BillableActivityRepository) Count(ctx context.Context, filter domain.ActivityFilter) (int64, error) {
	mongoFilter := r.buildFilter(filter)
	mongoFilter = r.tenantHelper.WithTenantFilterOptional(ctx, mongoFilter)
	return r.collection.CountDocuments(ctx, mongoFilter)
}

func (r *BillableActivityRepository) findMany(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*domain.BillableActivity, error) {
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var activities []*domain.BillableActivity
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *BillableActivityRepository) buildFilter(filter domain.ActivityFilter) bson.M {
	mongoFilter := bson.M{}
	if filter.TenantID != nil {
		mongoFilter["tenantId"] = *filter.TenantID
	}
	if filter.SellerID != nil {
		mongoFilter["sellerId"] = *filter.SellerID
	}
	if filter.FacilityID != nil {
		mongoFilter["facilityId"] = *filter.FacilityID
	}
	if filter.Type != nil {
		mongoFilter["type"] = *filter.Type
	}
	if filter.Invoiced != nil {
		mongoFilter["invoiced"] = *filter.Invoiced
	}
	return mongoFilter
}

// InvoiceRepository implements domain.InvoiceRepository
type InvoiceRepository struct {
	collection   *mongo.Collection
	db           *mongo.Database
	outboxRepo   *outboxMongo.OutboxRepository
	eventFactory *cloudevents.EventFactory
	tenantHelper *tenant.RepositoryHelper
}

// NewInvoiceRepository creates a new InvoiceRepository
func NewInvoiceRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *InvoiceRepository {
	collection := db.Collection("invoices")
	outboxRepo := outboxMongo.NewOutboxRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "invoiceId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "sellerId", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "sellerId", Value: 1},
				{Key: "periodStart", Value: 1},
				{Key: "periodEnd", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "dueDate", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "invoiceNumber", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)
	_ = outboxRepo.EnsureIndexes(ctx)

	return &InvoiceRepository{
		collection:   collection,
		db:           db,
		outboxRepo:   outboxRepo,
		eventFactory: eventFactory,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save persists an invoice with domain events
func (r *InvoiceRepository) Save(ctx context.Context, invoice *domain.Invoice) error {
	invoice.UpdatedAt = time.Now().UTC()

	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"invoiceId": invoice.InvoiceID}
		update := bson.M{"$set": invoice}

		if _, err := r.collection.UpdateOne(sessCtx, filter, update, opts); err != nil {
			return nil, fmt.Errorf("failed to save invoice: %w", err)
		}

		domainEvents := invoice.DomainEvents()
		if len(domainEvents) > 0 {
			outboxEvents := make([]*outbox.OutboxEvent, 0, len(domainEvents))

			for _, event := range domainEvents {
				var cloudEvent *cloudevents.WMSCloudEvent
				switch e := event.(type) {
				case *domain.InvoiceCreatedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "invoice/"+e.InvoiceID, e)
				case *domain.InvoiceFinalizedEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "invoice/"+e.InvoiceID, e)
				case *domain.InvoicePaidEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "invoice/"+e.InvoiceID, e)
				case *domain.InvoiceOverdueEvent:
					cloudEvent = r.eventFactory.CreateEvent(sessCtx, e.EventType(), "invoice/"+e.InvoiceID, e)
				default:
					continue
				}

				outboxEvent, err := outbox.NewOutboxEventFromCloudEvent(
					invoice.InvoiceID,
					"Invoice",
					kafka.Topics.BillingEvents,
					cloudEvent,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create outbox event: %w", err)
				}
				outboxEvents = append(outboxEvents, outboxEvent)
			}

			if len(outboxEvents) > 0 {
				if err := r.outboxRepo.SaveAll(sessCtx, outboxEvents); err != nil {
					return nil, fmt.Errorf("failed to save outbox events: %w", err)
				}
			}
		}

		invoice.ClearDomainEvents()
		return nil, nil
	})

	return err
}

// FindByID retrieves an invoice by ID
func (r *InvoiceRepository) FindByID(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	var invoice domain.Invoice
	filter := bson.M{"invoiceId": invoiceID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &invoice, nil
}

// FindBySellerID retrieves invoices for a seller
func (r *InvoiceRepository) FindBySellerID(ctx context.Context, sellerID string, pagination domain.Pagination) ([]*domain.Invoice, error) {
	filter := bson.M{"sellerId": sellerID}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindByStatus retrieves invoices by status
func (r *InvoiceRepository) FindByStatus(ctx context.Context, status domain.InvoiceStatus, pagination domain.Pagination) ([]*domain.Invoice, error) {
	filter := bson.M{"status": status}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(pagination.Skip()).
		SetLimit(pagination.Limit())

	return r.findMany(ctx, filter, opts)
}

// FindOverdue retrieves overdue invoices
func (r *InvoiceRepository) FindOverdue(ctx context.Context) ([]*domain.Invoice, error) {
	filter := bson.M{
		"status":  domain.InvoiceStatusFinalized,
		"dueDate": bson.M{"$lt": time.Now()},
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	return r.findMany(ctx, filter, nil)
}

// FindByPeriod retrieves invoices for a billing period
func (r *InvoiceRepository) FindByPeriod(ctx context.Context, sellerID string, periodStart, periodEnd time.Time) (*domain.Invoice, error) {
	var invoice domain.Invoice
	filter := bson.M{
		"sellerId":    sellerID,
		"periodStart": periodStart,
		"periodEnd":   periodEnd,
	}
	filter = r.tenantHelper.WithTenantFilterOptional(ctx, filter)

	err := r.collection.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &invoice, nil
}

// UpdateStatus updates invoice status
func (r *InvoiceRepository) UpdateStatus(ctx context.Context, invoiceID string, status domain.InvoiceStatus) error {
	filter := bson.M{"invoiceId": invoiceID}
	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("invoice not found")
	}
	return nil
}

// Count returns total count matching filter
func (r *InvoiceRepository) Count(ctx context.Context, filter domain.InvoiceFilter) (int64, error) {
	mongoFilter := r.buildFilter(filter)
	return r.collection.CountDocuments(ctx, mongoFilter)
}

func (r *InvoiceRepository) findMany(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*domain.Invoice, error) {
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invoices []*domain.Invoice
	if err := cursor.All(ctx, &invoices); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *InvoiceRepository) buildFilter(filter domain.InvoiceFilter) bson.M {
	mongoFilter := bson.M{}
	if filter.TenantID != nil {
		mongoFilter["tenantId"] = *filter.TenantID
	}
	if filter.SellerID != nil {
		mongoFilter["sellerId"] = *filter.SellerID
	}
	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}
	return mongoFilter
}

// GetOutboxRepository returns the outbox repository
func (r *InvoiceRepository) GetOutboxRepository() outbox.Repository {
	return r.outboxRepo
}

// StorageCalculationRepository implements domain.StorageCalculationRepository
type StorageCalculationRepository struct {
	collection   *mongo.Collection
	tenantHelper *tenant.RepositoryHelper
}

// NewStorageCalculationRepository creates a new StorageCalculationRepository
func NewStorageCalculationRepository(db *mongo.Database) *StorageCalculationRepository {
	collection := db.Collection("storage_calculations")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "sellerId", Value: 1},
				{Key: "calculationDate", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "facilityId", Value: 1},
				{Key: "calculationDate", Value: 1},
			},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &StorageCalculationRepository{
		collection:   collection,
		tenantHelper: tenant.NewRepositoryHelper(false),
	}
}

// Save persists a storage calculation
func (r *StorageCalculationRepository) Save(ctx context.Context, calc *domain.StorageCalculation) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{
		"sellerId":        calc.SellerID,
		"calculationDate": calc.CalculationDate,
	}
	update := bson.M{"$set": calc}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindBySellerAndDate retrieves calculation for a seller on a date
func (r *StorageCalculationRepository) FindBySellerAndDate(ctx context.Context, sellerID string, date time.Time) (*domain.StorageCalculation, error) {
	var calc domain.StorageCalculation
	// Normalize date to start of day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	filter := bson.M{
		"sellerId": sellerID,
		"calculationDate": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&calc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &calc, nil
}

// FindBySellerAndPeriod retrieves calculations for a period
func (r *StorageCalculationRepository) FindBySellerAndPeriod(ctx context.Context, sellerID string, start, end time.Time) ([]*domain.StorageCalculation, error) {
	filter := bson.M{
		"sellerId": sellerID,
		"calculationDate": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "calculationDate", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var calcs []*domain.StorageCalculation
	if err := cursor.All(ctx, &calcs); err != nil {
		return nil, err
	}
	return calcs, nil
}

// SumByPeriod returns total storage fees for a period
func (r *StorageCalculationRepository) SumByPeriod(ctx context.Context, sellerID string, start, end time.Time) (float64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"sellerId": sellerID,
				"calculationDate": bson.M{
					"$gte": start,
					"$lte": end,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$totalAmount"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total float64 `bson:"total"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
	}
	return result.Total, nil
}
