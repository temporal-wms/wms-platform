package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/wms-platform/wes-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StageTemplateRepository implements domain.StageTemplateRepository using MongoDB
type StageTemplateRepository struct {
	collection *mongo.Collection
}

// NewStageTemplateRepository creates a new StageTemplateRepository
func NewStageTemplateRepository(db *mongo.Database) *StageTemplateRepository {
	collection := db.Collection("stage_templates")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "templateId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "pathType", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isDefault", Value: 1}},
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &StageTemplateRepository{
		collection: collection,
	}
}

// Save saves a stage template
func (r *StageTemplateRepository) Save(ctx context.Context, template *domain.StageTemplate) error {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, template)
	if err != nil {
		return fmt.Errorf("failed to insert stage template: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		template.ID = oid
	}

	return nil
}

// FindByID finds a template by its MongoDB ObjectID
func (r *StageTemplateRepository) FindByID(ctx context.Context, id string) (*domain.StageTemplate, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid object id: %w", err)
	}

	var template domain.StageTemplate
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&template)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find stage template: %w", err)
	}

	return &template, nil
}

// FindByTemplateID finds a template by its template ID
func (r *StageTemplateRepository) FindByTemplateID(ctx context.Context, templateID string) (*domain.StageTemplate, error) {
	var template domain.StageTemplate
	err := r.collection.FindOne(ctx, bson.M{"templateId": templateID}).Decode(&template)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find stage template: %w", err)
	}

	return &template, nil
}

// FindByPathType finds templates by path type
func (r *StageTemplateRepository) FindByPathType(ctx context.Context, pathType domain.ProcessPathType) ([]*domain.StageTemplate, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"pathType": pathType})
	if err != nil {
		return nil, fmt.Errorf("failed to find stage templates: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*domain.StageTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("failed to decode stage templates: %w", err)
	}

	return templates, nil
}

// FindActive finds all active templates
func (r *StageTemplateRepository) FindActive(ctx context.Context) ([]*domain.StageTemplate, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"active": true})
	if err != nil {
		return nil, fmt.Errorf("failed to find active stage templates: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*domain.StageTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("failed to decode stage templates: %w", err)
	}

	return templates, nil
}

// FindDefault finds the default template
func (r *StageTemplateRepository) FindDefault(ctx context.Context) (*domain.StageTemplate, error) {
	var template domain.StageTemplate
	err := r.collection.FindOne(ctx, bson.M{"isDefault": true, "active": true}).Decode(&template)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find default stage template: %w", err)
	}

	return &template, nil
}

// Update updates a stage template
func (r *StageTemplateRepository) Update(ctx context.Context, template *domain.StageTemplate) error {
	template.UpdatedAt = time.Now()

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"templateId": template.TemplateID},
		template,
	)
	if err != nil {
		return fmt.Errorf("failed to update stage template: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("stage template not found: %s", template.TemplateID)
	}

	return nil
}

// SeedDefaultTemplates seeds the default templates if they don't exist
func (r *StageTemplateRepository) SeedDefaultTemplates(ctx context.Context) error {
	templates := []*domain.StageTemplate{
		domain.DefaultPickPackTemplate(),
		domain.DefaultPickWallPackTemplate(),
		domain.DefaultPickConsolidatePackTemplate(),
	}

	// Set the first one as default
	templates[0].SetDefault()

	for _, template := range templates {
		existing, err := r.FindByTemplateID(ctx, template.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to check existing template: %w", err)
		}
		if existing == nil {
			if err := r.Save(ctx, template); err != nil {
				return fmt.Errorf("failed to save template %s: %w", template.TemplateID, err)
			}
		}
	}

	return nil
}
