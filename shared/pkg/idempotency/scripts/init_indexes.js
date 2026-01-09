// MongoDB Index Initialization Script for Idempotency Package
// This script creates all required indexes for idempotency functionality
//
// Usage:
//   mongosh mongodb://localhost:27017/wms --file init_indexes.js
//
// Or from MongoDB shell:
//   load('init_indexes.js')

print("========================================");
print("Idempotency Package - Index Initialization");
print("========================================\n");

// Get the current database
const db = db.getSiblingDB('wms'); // Change to your database name

print("Creating indexes for idempotency_keys collection...\n");

// 1. Unique compound index on serviceId + key
// This ensures one idempotency key per service
db.idempotency_keys.createIndex(
    { "serviceId": 1, "key": 1 },
    {
        unique: true,
        name: "idx_service_key",
        background: false
    }
);
print("✓ Created idx_service_key (unique compound index)");

// 2. TTL index on expiresAt
// Automatically removes documents after they expire (24 hours by default)
db.idempotency_keys.createIndex(
    { "expiresAt": 1 },
    {
        expireAfterSeconds: 0,
        name: "idx_ttl",
        background: false
    }
);
print("✓ Created idx_ttl (TTL index for automatic cleanup)");

// 3. Sparse index on lockedAt
// Helps query for locked/stale locks
db.idempotency_keys.createIndex(
    { "lockedAt": 1 },
    {
        sparse: true,
        name: "idx_locked",
        background: false
    }
);
print("✓ Created idx_locked (sparse index for lock management)");

// 4. Optional: Composite index for user-scoped keys
// Uncomment if you're using user-scoped idempotency
// db.idempotency_keys.createIndex(
//     { "userId": 1, "key": 1 },
//     {
//         sparse: true,
//         name: "idx_user_key",
//         background: false
//     }
// );
// print("✓ Created idx_user_key (user-scoped keys)");

print("\nCreating indexes for processed_messages collection...\n");

// 1. Unique compound index on messageId + topic + consumerGroup
// This ensures exactly-once message processing per consumer group
db.processed_messages.createIndex(
    { "messageId": 1, "topic": 1, "consumerGroup": 1 },
    {
        unique: true,
        name: "idx_msg_topic_group",
        background: false
    }
);
print("✓ Created idx_msg_topic_group (unique compound index)");

// 2. TTL index on expiresAt
// Automatically removes old processed message records
db.processed_messages.createIndex(
    { "expiresAt": 1 },
    {
        expireAfterSeconds: 0,
        name: "idx_ttl",
        background: false
    }
);
print("✓ Created idx_ttl (TTL index for automatic cleanup)");

// 3. Index on processedAt
// Helps with queries and monitoring
db.processed_messages.createIndex(
    { "processedAt": 1 },
    {
        name: "idx_processed_at",
        background: false
    }
);
print("✓ Created idx_processed_at (for queries and monitoring)");

// 4. Optional: Index on correlationId for debugging
// Uncomment if you frequently query by correlation ID
// db.processed_messages.createIndex(
//     { "correlationId": 1 },
//     {
//         sparse: true,
//         name: "idx_correlation",
//         background: false
//     }
// );
// print("✓ Created idx_correlation (for debugging)");

print("\n========================================");
print("Index creation completed!");
print("========================================\n");

// Print index information
print("Indexes for idempotency_keys:");
printjson(db.idempotency_keys.getIndexes());

print("\nIndexes for processed_messages:");
printjson(db.processed_messages.getIndexes());

print("\n✓ All idempotency indexes created successfully!");
print("✓ TTL indexes will automatically clean up expired documents");
print("✓ Unique indexes will prevent duplicate idempotency keys and messages\n");
