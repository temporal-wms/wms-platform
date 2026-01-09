package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GenerateID generates a new MongoDB ObjectID
func GenerateID() primitive.ObjectID {
	return primitive.NewObjectID()
}

// GenerateIDString generates a new MongoDB ObjectID as a string
func GenerateIDString() string {
	return primitive.NewObjectID().Hex()
}

// ParseID parses a string into a MongoDB ObjectID
func ParseID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

// Now returns the current time in UTC
func Now() time.Time {
	return time.Now().UTC()
}

// BuildFilter builds a BSON filter from key-value pairs
func BuildFilter(pairs ...interface{}) bson.M {
	filter := bson.M{}
	for i := 0; i < len(pairs)-1; i += 2 {
		key, ok := pairs[i].(string)
		if ok {
			filter[key] = pairs[i+1]
		}
	}
	return filter
}

// BuildUpdate builds a BSON update document
func BuildUpdate(set bson.M) bson.M {
	return bson.M{"$set": set}
}

// BuildUpdateWithTimestamp builds a BSON update document with automatic updatedAt
func BuildUpdateWithTimestamp(set bson.M) bson.M {
	set["updatedAt"] = Now()
	return bson.M{"$set": set}
}

// BuildIncrementUpdate builds a BSON increment update
func BuildIncrementUpdate(field string, value interface{}) bson.M {
	return bson.M{
		"$inc": bson.M{field: value},
		"$set": bson.M{"updatedAt": Now()},
	}
}

// BuildPushUpdate builds a BSON push update for arrays
func BuildPushUpdate(field string, value interface{}) bson.M {
	return bson.M{
		"$push": bson.M{field: value},
		"$set":  bson.M{"updatedAt": Now()},
	}
}

// BuildPullUpdate builds a BSON pull update for arrays
func BuildPullUpdate(field string, condition bson.M) bson.M {
	return bson.M{
		"$pull": bson.M{field: condition},
		"$set":  bson.M{"updatedAt": Now()},
	}
}

// SortAscending creates an ascending sort option
func SortAscending(field string) bson.D {
	return bson.D{{Key: field, Value: 1}}
}

// SortDescending creates a descending sort option
func SortDescending(field string) bson.D {
	return bson.D{{Key: field, Value: -1}}
}

// SortMultiple creates a multi-field sort option
func SortMultiple(fields ...SortField) bson.D {
	sort := bson.D{}
	for _, f := range fields {
		if f.Descending {
			sort = append(sort, bson.E{Key: f.Field, Value: -1})
		} else {
			sort = append(sort, bson.E{Key: f.Field, Value: 1})
		}
	}
	return sort
}

// SortField represents a field to sort by
type SortField struct {
	Field      string
	Descending bool
}

// Pagination represents pagination options
type Pagination struct {
	Page     int64
	PageSize int64
}

// DefaultPagination returns default pagination options
func DefaultPagination() *Pagination {
	return &Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Skip returns the number of documents to skip
func (p *Pagination) Skip() int64 {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the maximum number of documents to return
func (p *Pagination) Limit() int64 {
	return p.PageSize
}

// BaseDocument contains common fields for all MongoDB documents
type BaseDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// NewBaseDocument creates a new BaseDocument with initialized fields
func NewBaseDocument() BaseDocument {
	now := Now()
	return BaseDocument{
		ID:        GenerateID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetUpdated sets the UpdatedAt field to the current time
func (b *BaseDocument) SetUpdated() {
	b.UpdatedAt = Now()
}
