package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Document size monitoring tool for inventory collection
// Alerts when documents are approaching MongoDB's 16MB limit

var (
	mongoURI  = flag.String("mongo-uri", "mongodb://localhost:27017", "MongoDB connection URI")
	dbName    = flag.String("db", "temporal_war", "Database name")
	threshold = flag.Int("threshold", 8388608, "Alert threshold in bytes (default: 8MB)")
	limit     = flag.Int("limit", 50, "Maximum number of results to display")
)

const (
	MB16 = 16777216 // 16 MB in bytes
	MB8  = 8388608  // 8 MB in bytes
	MB5  = 5242880  // 5 MB in bytes
	MB1  = 1048576  // 1 MB in bytes
)

type DocumentSizeInfo struct {
	SKU             string `bson:"sku"`
	Size            int    `bson:"size"`
	TransactionCount int    `bson:"txCount"`
	ReservationCount int    `bson:"resCount"`
	AllocationCount  int    `bson:"allocCount"`
}

func main() {
	flag.Parse()

	log.Printf("Starting document size monitoring...")
	log.Printf("MongoDB URI: %s", *mongoURI)
	log.Printf("Database: %s", *dbName)
	log.Printf("Alert Threshold: %d bytes (%.2f MB)", *threshold, float64(*threshold)/MB1)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(*mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB successfully")

	db := client.Database(*dbName)

	// Analyze inventory collection
	if err := analyzeCollection(context.Background(), db, "inventory"); err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}
}

func analyzeCollection(ctx context.Context, db *mongo.Database, collectionName string) error {
	collection := db.Collection(collectionName)

	// Get document count
	totalCount, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}

	fmt.Printf("\n=== Collection: %s ===\n", collectionName)
	fmt.Printf("Total Documents: %d\n\n", totalCount)

	// Aggregate pipeline to calculate document sizes and array lengths
	pipeline := []bson.M{
		{
			"$project": bson.M{
				"sku":      1,
				"size":     bson.M{"$bsonSize": "$$ROOT"},
				"txCount":  bson.M{"$size": bson.M{"$ifNull": []interface{}{"$transactions", []interface{}{}}}},
				"resCount": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$reservations", []interface{}{}}}},
				"allocCount": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$hardAllocations", []interface{}{}}}},
			},
		},
		{
			"$match": bson.M{
				"size": bson.M{"$gte": *threshold},
			},
		},
		{
			"$sort": bson.M{"size": -1},
		},
		{
			"$limit": int64(*limit),
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to run aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var largeDocuments []DocumentSizeInfo
	if err := cursor.All(ctx, &largeDocuments); err != nil {
		return fmt.Errorf("failed to decode results: %w", err)
	}

	// Display results
	if len(largeDocuments) == 0 {
		fmt.Println("✅ No documents exceed the threshold")
		return nil
	}

	fmt.Printf("⚠️  Found %d documents exceeding %d bytes:\n\n", len(largeDocuments), *threshold)
	fmt.Println("SKU                                  Size (MB)   Transactions  Reservations  Allocations  Status")
	fmt.Println("-----------------------------------  ----------  ------------  ------------  -----------  --------")

	for _, doc := range largeDocuments {
		sizeMB := float64(doc.Size) / MB1
		status := getStatus(doc.Size)
		fmt.Printf("%-35s  %10.2f  %12d  %12d  %11d  %s\n",
			doc.SKU,
			sizeMB,
			doc.TransactionCount,
			doc.ReservationCount,
			doc.AllocationCount,
			status,
		)
	}

	// Distribution analysis
	fmt.Println("\n=== Size Distribution ===")
	if err := analyzeSizeDistribution(ctx, collection); err != nil {
		log.Printf("WARNING: Failed to analyze distribution: %v", err)
	}

	// Recommendations
	fmt.Println("\n=== Recommendations ===")
	for _, doc := range largeDocuments {
		if doc.Size > MB8 {
			fmt.Printf("🚨 CRITICAL: SKU %s (%0.2f MB)\n", doc.SKU, float64(doc.Size)/MB1)
			if doc.TransactionCount > 1000 {
				fmt.Printf("   - %d transactions: Consider archiving to inventory_transactions collection\n", doc.TransactionCount)
			}
			if doc.ReservationCount > 100 {
				fmt.Printf("   - %d reservations: Consider moving to inventory_reservations collection\n", doc.ReservationCount)
			}
			if doc.AllocationCount > 100 {
				fmt.Printf("   - %d allocations: Consider moving to inventory_allocations collection\n", doc.AllocationCount)
			}
			fmt.Println()
		}
	}

	return nil
}

func analyzeSizeDistribution(ctx context.Context, collection *mongo.Collection) error {
	pipeline := []bson.M{
		{
			"$project": bson.M{
				"size": bson.M{"$bsonSize": "$$ROOT"},
			},
		},
		{
			"$bucket": bson.M{
				"groupBy": "$size",
				"boundaries": []int{
					0,
					MB1,    // 1MB
					MB5,    // 5MB
					MB8,    // 8MB
					MB16,   // 16MB
				},
				"default": "16MB+",
				"output": bson.M{
					"count": bson.M{"$sum": 1},
				},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	type BucketResult struct {
		ID    interface{} `bson:"_id"`
		Count int         `bson:"count"`
	}

	var results []BucketResult
	if err := cursor.All(ctx, &results); err != nil {
		return err
	}

	for _, result := range results {
		var label string
		switch result.ID {
		case 0:
			label = "0-1 MB"
		case MB1:
			label = "1-5 MB"
		case MB5:
			label = "5-8 MB"
		case MB8:
			label = "8-16 MB"
		default:
			label = fmt.Sprintf("%v", result.ID)
		}
		fmt.Printf("  %s: %d documents\n", label, result.Count)
	}

	return nil
}

func getStatus(size int) string {
	if size >= 12*MB1 {
		return "🔴 URGENT"
	} else if size >= MB8 {
		return "🟠 WARNING"
	} else if size >= MB5 {
		return "🟡 CAUTION"
	}
	return "🟢 OK"
}
