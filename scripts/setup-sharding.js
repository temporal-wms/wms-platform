// MongoDB Sharding Configuration Script
// Sets up sharding for horizontal scalability
//
// Prerequisites:
// 1. MongoDB cluster with sharding enabled (config servers + mongos routers)
// 2. Run this script through mongos (not directly on mongod)
// 3. Database should already exist with data
//
// Usage: mongosh mongodb://mongos:27017/admin setup-sharding.js

print("====================================");
print("MongoDB Sharding Configuration");
print("Temporal War WMS Platform");
print("====================================\n");

// Configuration
const DATABASE_NAME = "temporal_war";
const TENANT_FIELD = "tenantId";
const FACILITY_FIELD = "facilityId";

// Switch to admin database for sharding commands
const adminDb = db.getSiblingDB("admin");
const wmsDb = db.getSiblingDB(DATABASE_NAME);

print("Step 1: Verify sharding is enabled\n");

try {
  const shardingStatus = adminDb.runCommand({ listShards: 1 });
  if (shardingStatus.ok === 1) {
    print(`✅ Sharding is enabled`);
    print(`   Shards: ${shardingStatus.shards.length}`);
    shardingStatus.shards.forEach(shard => {
      print(`   - ${shard._id}: ${shard.host}`);
    });
  } else {
    print(`❌ Sharding is not enabled. Please configure sharding first.`);
    quit(1);
  }
} catch (error) {
  print(`❌ Error checking sharding status: ${error.message}`);
  print(`   Make sure you're connected to mongos, not mongod`);
  quit(1);
}

print("\nStep 2: Enable sharding for database\n");

try {
  const result = adminDb.runCommand({ enableSharding: DATABASE_NAME });
  if (result.ok === 1) {
    print(`✅ Database sharding enabled: ${DATABASE_NAME}`);
  }
} catch (error) {
  if (error.message.includes("already enabled")) {
    print(`⏭️  Database sharding already enabled: ${DATABASE_NAME}`);
  } else {
    print(`❌ Failed to enable database sharding: ${error.message}`);
    quit(1);
  }
}

print("\nStep 3: Configure shard keys for collections\n");

// Shard key configurations
// Key design principles:
// 1. Include tenantId first for tenant isolation
// 2. Include facilityId for facility-level partitioning
// 3. Include high-cardinality field (ID) to avoid hotspots
const shardConfigs = [
  {
    collection: "inventory",
    key: { tenantId: 1, facilityId: 1, sku: 1 },
    description: "Inventory items sharded by tenant + facility + SKU",
    unique: true
  },
  {
    collection: "inventory_transactions",
    key: { tenantId: 1, facilityId: 1, transactionId: 1 },
    description: "Transactions sharded by tenant + facility + transaction ID",
    unique: false
  },
  {
    collection: "inventory_reservations",
    key: { tenantId: 1, facilityId: 1, reservationId: 1 },
    description: "Reservations sharded by tenant + facility + reservation ID",
    unique: false
  },
  {
    collection: "inventory_allocations",
    key: { tenantId: 1, facilityId: 1, allocationId: 1 },
    description: "Allocations sharded by tenant + facility + allocation ID",
    unique: false
  },
  {
    collection: "orders",
    key: { tenantId: 1, facilityId: 1, orderId: 1 },
    description: "Orders sharded by tenant + facility + order ID",
    unique: false // Already has sparse unique index
  },
  {
    collection: "shipments",
    key: { tenantId: 1, facilityId: 1, shipmentId: 1 },
    description: "Shipments sharded by tenant + facility + shipment ID",
    unique: false
  },
  {
    collection: "waves",
    key: { tenantId: 1, facilityId: 1, waveId: 1 },
    description: "Waves sharded by tenant + facility + wave ID",
    unique: false
  },
  {
    collection: "workers",
    key: { tenantId: 1, facilityId: 1, workerId: 1 },
    description: "Workers sharded by tenant + facility + worker ID",
    unique: false
  },
  {
    collection: "outbox_events",
    key: { aggregateId: 1, createdAt: 1 },
    description: "Outbox events sharded by aggregate ID + created time",
    unique: false
  }
];

let successCount = 0;
let skipCount = 0;
let failCount = 0;

shardConfigs.forEach(config => {
  try {
    print(`Sharding collection: ${config.collection}`);
    print(`  Shard key: ${JSON.stringify(config.key)}`);
    print(`  Description: ${config.description}`);

    const result = adminDb.runCommand({
      shardCollection: `${DATABASE_NAME}.${config.collection}`,
      key: config.key,
      unique: config.unique
    });

    if (result.ok === 1) {
      print(`  Status: ✅ Sharded successfully\n`);
      successCount++;
    }
  } catch (error) {
    if (error.message.includes("already sharded")) {
      print(`  Status: ⏭️  Already sharded\n`);
      skipCount++;
    } else if (error.message.includes("ns not found")) {
      print(`  Status: ⚠️  Collection doesn't exist yet (will be sharded when created)\n`);
      skipCount++;
    } else {
      print(`  Status: ❌ Failed - ${error.message}\n`);
      failCount++;
    }
  }
});

print("\n====================================");
print("Sharding Configuration Summary");
print("====================================\n");

print(`✅ Successfully sharded: ${successCount} collections`);
print(`⏭️  Already sharded/pending: ${skipCount} collections`);
print(`❌ Failed: ${failCount} collections\n`);

print("\n====================================");
print("Shard Distribution Analysis");
print("====================================\n");

print("Checking data distribution across shards:\n");

shardConfigs.forEach(config => {
  try {
    const stats = wmsDb[config.collection].getShardDistribution();
    print(`Collection: ${config.collection}`);
    // Note: getShardDistribution() prints its own output
  } catch (error) {
    print(`Collection: ${config.collection}`);
    print(`  (Distribution stats not available - collection may not exist yet)\n`);
  }
});

print("\n====================================");
print("Important: Query Pattern Guidelines");
print("====================================\n");

print("To avoid scatter-gather queries, ALWAYS include shard key fields:");
print("");
print("✅ GOOD - Uses shard key (single shard query):");
print('   db.inventory.find({');
print('     tenantId: "TENANT-001",');
print('     facilityId: "FAC-001",');
print('     sku: "SKU-123"');
print('   })');
print("");
print("❌ BAD - Missing shard key (scatter-gather across all shards):");
print('   db.inventory.find({ sku: "SKU-123" })');
print("");
print("⚠️  ACCEPTABLE - Partial shard key (queries multiple shards within tenant):");
print('   db.inventory.find({');
print('     tenantId: "TENANT-001",');
print('     velocityClass: "A"');
print('   })');
print("");

print("\n====================================");
print("Monitoring Commands");
print("====================================\n");

print("1. Check sharding status:");
print("   sh.status()\n");

print("2. View balancer status:");
print("   sh.getBalancerState()\n");

print("3. Check chunk distribution:");
print("   db.printShardingStatus()\n");

print("4. Monitor query routing (explain plan):");
print('   db.inventory.find({ tenantId: "T1", sku: "S1" }).explain("executionStats")\n');

print("5. View current operations:");
print('   db.currentOp({ "command.shardVersion": { $exists: true } })\n');

print("\n====================================");
print("Balancing Configuration");
print("====================================\n");

print("Configure balancing windows for production:");
print("");
print("// Enable balancer only during off-peak hours (2 AM - 6 AM)");
print("db.settings.updateOne(");
print('  { _id: "balancer" },');
print("  {");
print("    $set: {");
print('      activeWindow: {');
print('        start: "02:00",');
print('        stop: "06:00"');
print("      }");
print("    }");
print("  },");
print("  { upsert: true }");
print(");\n");

print("// Check balancer configuration");
print('db.settings.find({ _id: "balancer" }).pretty();\n');

print("\n====================================");
print("Shard Key Selection Rationale");
print("====================================\n");

print("Chosen Pattern: { tenantId: 1, facilityId: 1, <highCardinalityId>: 1 }");
print("");
print("✅ Benefits:");
print("   - Tenant isolation (each tenant's data grouped together)");
print("   - Facility-level partitioning (facilities can be on different shards)");
print("   - High cardinality prevents hotspots (many chunks per tenant)");
print("   - Supports multi-tenant queries efficiently");
print("");
print("⚠️  Trade-offs:");
print("   - Queries MUST include tenantId to be efficient");
print("   - Cross-tenant queries will scatter-gather (acceptable - rare)");
print("   - Requires application-level enforcement of tenant context");
print("");

print("\n====================================");
print("Migration to Sharded Cluster");
print("====================================\n");

print("If migrating existing data:");
print("");
print("1. Take a backup before enabling sharding:");
print("   mongodump --uri='mongodb://localhost:27017' --db=temporal_war\n");

print("2. Enable sharding during low-traffic window\n");

print("3. Monitor initial chunk migration (can take hours):");
print("   sh.status()\n");

print("4. Validate data distribution:");
print("   db.inventory.getShardDistribution()\n");

print("5. Test query performance:");
print("   - Run explain() on critical queries");
print("   - Verify single-shard routing for tenant-scoped queries");
print("   - Monitor slow query log\n");

print("\n====================================");
print("Sharding Configuration Complete!");
print("====================================");
print("");
print("Next steps:");
print("1. Monitor chunk distribution over next 24-48 hours");
print("2. Adjust balancing windows if needed");
print("3. Update application code to always include shard key in queries");
print("4. Set up monitoring alerts for unbalanced shards");
print("5. Document shard key requirements for developers");
