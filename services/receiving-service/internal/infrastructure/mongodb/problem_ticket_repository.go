package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wms-platform/services/receiving-service/internal/domain"
	"github.com/wms-platform/shared/pkg/cloudevents"
	"github.com/wms-platform/shared/pkg/outbox"
)

const problemTicketsCollection = "problem_tickets"

// ProblemTicketRepository implements domain.ProblemTicketRepository using MongoDB
type ProblemTicketRepository struct {
	db             *mongo.Database
	collection     *mongo.Collection
	outbox         *outbox.MongoOutboxRepository
	eventFactory   *cloudevents.EventFactory
}

// NewProblemTicketRepository creates a new MongoDB-based problem ticket repository
func NewProblemTicketRepository(db *mongo.Database, eventFactory *cloudevents.EventFactory) *ProblemTicketRepository {
	return &ProblemTicketRepository{
		db:           db,
		collection:   db.Collection(problemTicketsCollection),
		outbox:       outbox.NewMongoOutboxRepository(db),
		eventFactory: eventFactory,
	}
}

// Save saves a problem ticket and publishes domain events via outbox
func (r *ProblemTicketRepository) Save(ticket *domain.ProblemTicket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start a session for transaction
	session, err := r.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Execute in transaction
	_, err = session.WithTransaction(ctx, func(sessionCtx mongo.SessionContext) (interface{}, error) {
		// Upsert the ticket
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"ticketId": ticket.TicketID}
		update := bson.M{"$set": ticket}

		_, err := r.collection.UpdateOne(sessionCtx, filter, update, opts)
		if err != nil {
			return nil, err
		}

		// Publish domain events via outbox
		for _, event := range ticket.GetDomainEvents() {
			cloudEvent, err := r.eventFactory.CreateEvent(
				event.EventType(),
				"/receiving-service/problems/"+ticket.TicketID,
				event,
			)
			if err != nil {
				return nil, err
			}

			if err := r.outbox.Add(sessionCtx, cloudEvent); err != nil {
				return nil, err
			}
		}

		// Clear domain events
		ticket.ClearDomainEvents()

		return nil, nil
	})

	return err
}

// FindByID finds a problem ticket by its ID
func (r *ProblemTicketRepository) FindByID(ticketID string) (*domain.ProblemTicket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ticket domain.ProblemTicket
	err := r.collection.FindOne(ctx, bson.M{"ticketId": ticketID}).Decode(&ticket)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &ticket, nil
}

// FindByShipmentID finds all problem tickets for a shipment
func (r *ProblemTicketRepository) FindByShipmentID(shipmentID string) ([]*domain.ProblemTicket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"shipmentId": shipmentID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tickets []*domain.ProblemTicket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}

// FindPending finds pending problem tickets
func (r *ProblemTicketRepository) FindPending(limit int) ([]*domain.ProblemTicket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"resolution": bson.M{"$in": []string{
			string(domain.ResolutionPending),
			string(domain.ResolutionInvestigate),
		}},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tickets []*domain.ProblemTicket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}

// FindByResolution finds problem tickets by resolution status
func (r *ProblemTicketRepository) FindByResolution(resolution domain.ProblemResolution, limit int) ([]*domain.ProblemTicket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"resolution": string(resolution)}
	opts := options.Find().
		SetSort(bson.D{{Key: "updatedAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tickets []*domain.ProblemTicket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}

// GetOutboxRepository returns the outbox repository for publishing events
func (r *ProblemTicketRepository) GetOutboxRepository() *outbox.MongoOutboxRepository {
	return r.outbox
}
