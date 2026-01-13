package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/wms-platform/inventory-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration tool to extract transactions, reservations, and allocations
// from inventory documents into separate collections

var (
	mongoURI       = flag.String("mongo-uri", "mongodb://localhost:27017", "MongoDB connection URI")
	dbName         = flag.String("db", "temporal_war", "Database name")
	dryRun         = flag.Bool("dry-run", true, "Dry run mode (no actual writes)")
	batchSize      = flag.Int("batch-size", 100, "Batch size for processing")
	removeArrays   = flag.Bool("remove-arrays", false, "Remove arrays from inventory documents after migration")
)

type InventoryDocument struct {
	SKU             string                       `bson:"sku"`
	TenantID        string                       `bson:"tenantId"`
	FacilityID      string                       `bson:"facilityId"`
	WarehouseID     string                       `bson:"warehouseId"`
	SellerID        string                       `bson:"sellerId,omitempty"`
	Reservations    []domain.Reservation         `bson:"reservations,omitempty"`
	HardAllocations []domain.HardAllocation      `bson:"hardAllocations,omitempty"`
	Transactions    []domain.InventoryTransaction `bson:"transactions,omitempty"`
}

func main() {
	flag.Parse()

	log.Printf("Starting inventory migration...")
	log.Printf("MongoDB URI: %s", *mongoURI)
	log.Printf("Database: %s", *dbName)
	log.Printf("Dry Run: %v", *dryRun)
	log.Printf("Batch Size: %d", *batchSize)
	log.Printf("Remove Arrays: %v", *removeArrays)

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

	// Run migration
	if err := migrateInventory(context.Background(), db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
}

func migrateInventory(ctx context.Context, db *mongo.Database) error {
	inventoryColl := db.Collection("inventory")
	transactionsColl := db.Collection("inventory_transactions")
	reservationsColl := db.Collection("inventory_reservations")
	allocationsColl := db.Collection("inventory_allocations")

	var (
		totalDocs          int64
		totalTransactions  int64
		totalReservations  int64
		totalAllocations   int64
		docsWithTransactions int64
		docsWithReservations int64
		docsWithAllocations  int64
	)

	// Count total documents
	count, err := inventoryColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}
	log.Printf("Found %d inventory documents to process", count)

	// Process in batches
	opts := options.Find().SetBatchSize(int32(*batchSize))
	cursor, err := inventoryColl.Find(ctx, bson.M{}, opts)
	if err != nil {
		return fmt.Errorf("failed to query inventory: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc InventoryDocument
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("WARNING: Failed to decode document: %v", err)
			continue
		}

		totalDocs++

		// Migrate transactions
		if len(doc.Transactions) > 0 {
			docsWithTransactions++
			for _, txn := range doc.Transactions {
				totalTransactions++

				txnAgg := &domain.InventoryTransactionAggregate{
					TransactionID: txn.TransactionID,
					SKU:           doc.SKU,
					TenantID:      doc.TenantID,
					FacilityID:    doc.FacilityID,
					WarehouseID:   doc.WarehouseID,
					SellerID:      doc.SellerID,
					Type:          txn.Type,
					Quantity:      txn.Quantity,
					LocationID:    txn.LocationID,
					ReferenceID:   txn.ReferenceID,
					Reason:        txn.Reason,
					CreatedAt:     txn.CreatedAt,
					CreatedBy:     txn.CreatedBy,
				}

				if !*dryRun {
					if _, err := transactionsColl.InsertOne(ctx, txnAgg); err != nil {
						log.Printf("WARNING: Failed to insert transaction %s: %v", txn.TransactionID, err)
					}
				}
			}
		}

		// Migrate reservations
		if len(doc.Reservations) > 0 {
			docsWithReservations++
			for _, res := range doc.Reservations {
				totalReservations++

				resAgg := &domain.InventoryReservationAggregate{
					ReservationID: res.ReservationID,
					SKU:           doc.SKU,
					TenantID:      doc.TenantID,
					FacilityID:    doc.FacilityID,
					WarehouseID:   doc.WarehouseID,
					SellerID:      doc.SellerID,
					OrderID:       res.OrderID,
					Quantity:      res.Quantity,
					LocationID:    res.LocationID,
					Status:        domain.ReservationStatus(res.Status),
					UnitIDs:       res.UnitIDs,
					CreatedAt:     res.CreatedAt,
					ExpiresAt:     res.ExpiresAt,
					UpdatedAt:     res.CreatedAt, // Use created time if no update time
				}

				if !*dryRun {
					if _, err := reservationsColl.InsertOne(ctx, resAgg); err != nil {
						log.Printf("WARNING: Failed to insert reservation %s: %v", res.ReservationID, err)
					}
				}
			}
		}

		// Migrate hard allocations
		if len(doc.HardAllocations) > 0 {
			docsWithAllocations++
			for _, alloc := range doc.HardAllocations {
				totalAllocations++

				allocAgg := &domain.InventoryAllocationAggregate{
					AllocationID:      alloc.AllocationID,
					SKU:               doc.SKU,
					TenantID:          doc.TenantID,
					FacilityID:        doc.FacilityID,
					WarehouseID:       doc.WarehouseID,
					SellerID:          doc.SellerID,
					ReservationID:     alloc.ReservationID,
					OrderID:           alloc.OrderID,
					Quantity:          alloc.Quantity,
					SourceLocationID:  alloc.SourceLocationID,
					StagingLocationID: alloc.StagingLocationID,
					Status:            domain.AllocationStatus(alloc.Status),
					UnitIDs:           alloc.UnitIDs,
					StagedBy:          alloc.StagedBy,
					PackedBy:          alloc.PackedBy,
					ShippedBy:         "", // Not in original schema
					CreatedAt:         alloc.CreatedAt,
					PackedAt:          alloc.PackedAt,
					ShippedAt:         alloc.ShippedAt,
					UpdatedAt:         alloc.CreatedAt, // Use created time if no update time
				}

				if !*dryRun {
					if _, err := allocationsColl.InsertOne(ctx, allocAgg); err != nil {
						log.Printf("WARNING: Failed to insert allocation %s: %v", alloc.AllocationID, err)
					}
				}
			}
		}

		// Remove arrays from inventory document if requested
		if *removeArrays && !*dryRun {
			filter := bson.M{"sku": doc.SKU}
			update := bson.M{
				"$unset": bson.M{
					"transactions":    "",
					"reservations":    "",
					"hardAllocations": "",
				},
			}
			if _, err := inventoryColl.UpdateOne(ctx, filter, update); err != nil {
				log.Printf("WARNING: Failed to remove arrays from SKU %s: %v", doc.SKU, err)
			}
		}

		// Progress logging every 100 docs
		if totalDocs%100 == 0 {
			log.Printf("Processed %d/%d documents...", totalDocs, count)
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	// Print summary
	fmt.Println("\n=== Migration Summary ===")
	fmt.Printf("Total Documents Processed: %d\n", totalDocs)
	fmt.Printf("\nTransactions:\n")
	fmt.Printf("  Documents with transactions: %d\n", docsWithTransactions)
	fmt.Printf("  Total transactions migrated: %d\n", totalTransactions)
	fmt.Printf("\nReservations:\n")
	fmt.Printf("  Documents with reservations: %d\n", docsWithReservations)
	fmt.Printf("  Total reservations migrated: %d\n", totalReservations)
	fmt.Printf("\nAllocations:\n")
	fmt.Printf("  Documents with allocations: %d\n", docsWithAllocations)
	fmt.Printf("  Total allocations migrated: %d\n", totalAllocations)

	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No actual changes were made")
		fmt.Println("Run with -dry-run=false to perform actual migration")
	} else {
		fmt.Println("\n✅ Migration completed successfully!")
		if *removeArrays {
			fmt.Println("   Arrays removed from inventory documents")
		} else {
			fmt.Println("   Arrays retained in inventory documents (backward compatibility)")
		}
	}

	return nil
}
