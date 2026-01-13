// MongoDB Index Optimization Script
// Adds missing indexes identified in the data modeling assessment
//
// Usage: mongosh mongodb://localhost:27017/temporal_war add-missing-indexes.js

print("====================================");
print("MongoDB Index Optimization");
print("Adding missing performance indexes");
print("====================================\n");

const db = db.getSiblingDB('temporal_war');

// Track results
let results = {
  success: [],
  failed: [],
  skipped: []
};

function addIndex(collection, indexSpec, indexOptions, description) {
  try {
    print(`Adding index: ${description}`);
    print(`  Collection: ${collection}`);
    print(`  Keys: ${JSON.stringify(indexSpec)}`);

    const result = db[collection].createIndex(indexSpec, indexOptions);

    if (result === "Index already exists") {
      print(`  Status: ⏭️  Already exists`);
      results.skipped.push({ collection, description });
    } else {
      print(`  Status: ✅ Created successfully`);
      results.success.push({ collection, description });
    }
    print("");
  } catch (error) {
    print(`  Status: ❌ Failed - ${error.message}`);
    results.failed.push({ collection, description, error: error.message });
    print("");
  }
}

print("=== 1. Inventory Collection Indexes ===\n");

// Velocity + Zone queries (for slotting optimization)
addIndex(
  "inventory",
  {
    tenantId: 1,
    facilityId: 1,
    velocityClass: 1,
    "locations.zone": 1
  },
  { name: "idx_tenant_facility_velocity_zone" },
  "Velocity-based slotting queries"
);

// Date range analytics
addIndex(
  "inventory",
  {
    tenantId: 1,
    facilityId: 1,
    createdAt: 1
  },
  { name: "idx_tenant_facility_created" },
  "Time-based analytics queries"
);

// Seller inventory lookup (3PL/FBA)
addIndex(
  "inventory",
  {
    tenantId: 1,
    facilityId: 1,
    sellerId: 1
  },
  { name: "idx_tenant_facility_seller", sparse: true },
  "Seller-specific inventory queries"
);

print("=== 2. Orders Collection Indexes ===\n");

// Date range queries for analytics/reporting
addIndex(
  "orders",
  {
    tenantId: 1,
    facilityId: 1,
    createdAt: 1,
    status: 1
  },
  { name: "idx_tenant_facility_created_status" },
  "Time-based order analytics"
);

// Promised delivery date queries (SLA monitoring)
addIndex(
  "orders",
  {
    tenantId: 1,
    facilityId: 1,
    promisedDeliveryAt: 1,
    status: 1
  },
  { name: "idx_tenant_facility_promised_status" },
  "SLA monitoring queries"
);

// Carrier + status queries (shipping analytics)
addIndex(
  "orders",
  {
    tenantId: 1,
    facilityId: 1,
    "shippingInfo.carrier": 1,
    status: 1
  },
  { name: "idx_tenant_facility_carrier_status", sparse: true },
  "Carrier-based analytics"
);

print("=== 3. Workers Collection Indexes ===\n");

// Skill-based task assignment
addIndex(
  "workers",
  {
    tenantId: 1,
    facilityId: 1,
    status: 1,
    "skills.type": 1,
    "skills.level": -1
  },
  { name: "idx_tenant_facility_status_skills" },
  "Skill-based worker assignment"
);

// Zone + status queries (zone-specific labor)
addIndex(
  "workers",
  {
    tenantId: 1,
    facilityId: 1,
    currentZone: 1,
    status: 1
  },
  { name: "idx_tenant_facility_zone_status" },
  "Zone-based worker queries"
);

// Performance analytics
addIndex(
  "workers",
  {
    tenantId: 1,
    facilityId: 1,
    "performanceMetrics.accuracyRate": -1
  },
  { name: "idx_tenant_facility_accuracy", sparse: true },
  "Worker performance analytics"
);

print("=== 4. Waves Collection Indexes ===\n");

// Wave release time queries
addIndex(
  "waves",
  {
    tenantId: 1,
    facilityId: 1,
    scheduledStart: 1,
    status: 1
  },
  { name: "idx_tenant_facility_scheduled_status" },
  "Wave scheduling queries"
);

// Wave type + priority queries
addIndex(
  "waves",
  {
    tenantId: 1,
    facilityId: 1,
    waveType: 1,
    priority: 1,
    status: 1
  },
  { name: "idx_tenant_facility_type_priority_status" },
  "Wave prioritization queries"
);

print("=== 5. Shipments Collection Indexes ===\n");

// Carrier + manifest queries
addIndex(
  "shipments",
  {
    tenantId: 1,
    facilityId: 1,
    "carrier.code": 1,
    status: 1
  },
  { name: "idx_tenant_facility_carrier_status" },
  "Carrier-based shipment queries"
);

// Manifest generation queries
addIndex(
  "shipments",
  {
    tenantId: 1,
    facilityId: 1,
    "manifest.manifestId": 1
  },
  { name: "idx_tenant_facility_manifest", sparse: true },
  "Manifest lookup queries"
);

// Delivery date tracking
addIndex(
  "shipments",
  {
    tenantId: 1,
    facilityId: 1,
    estimatedDelivery: 1,
    status: 1
  },
  { name: "idx_tenant_facility_delivery_status" },
  "Delivery tracking queries"
);

print("=== 6. Stations Collection Indexes ===\n");

// Station type + capability queries
addIndex(
  "stations",
  {
    tenantId: 1,
    facilityId: 1,
    stationType: 1,
    status: 1,
    capabilities: 1
  },
  { name: "idx_tenant_facility_type_status_caps" },
  "Capability-based station routing"
);

// Zone + sequence queries (directed routing)
addIndex(
  "stations",
  {
    tenantId: 1,
    facilityId: 1,
    zone: 1,
    sequenceInZone: 1
  },
  { name: "idx_tenant_facility_zone_sequence" },
  "Sequential station routing"
);

print("=== 7. New Collections (from refactoring) ===\n");

// These indexes are created by repositories, but verify they exist
const newCollections = [
  {
    name: "inventory_transactions",
    indexes: [
      { keys: { sku: 1, createdAt: -1 }, name: "idx_sku_created" },
      { keys: { referenceId: 1 }, name: "idx_reference" }
    ]
  },
  {
    name: "inventory_reservations",
    indexes: [
      { keys: { reservationId: 1 }, name: "idx_reservation_id" },
      { keys: { orderId: 1 }, name: "idx_order_id" },
      { keys: { sku: 1, status: 1 }, name: "idx_sku_status" }
    ]
  },
  {
    name: "inventory_allocations",
    indexes: [
      { keys: { allocationId: 1 }, name: "idx_allocation_id" },
      { keys: { reservationId: 1 }, name: "idx_reservation_id" },
      { keys: { orderId: 1 }, name: "idx_order_id" }
    ]
  }
];

newCollections.forEach(coll => {
  print(`Verifying indexes for ${coll.name}:`);
  const existingIndexes = db[coll.name].getIndexes();

  if (existingIndexes.length === 0) {
    print(`  ⚠️  Collection does not exist yet (will be created by repositories)`);
  } else {
    print(`  ✅ Collection exists with ${existingIndexes.length} indexes`);
  }
  print("");
});

print("\n====================================");
print("Index Optimization Summary");
print("====================================\n");

print(`✅ Successfully created: ${results.success.length} indexes`);
if (results.success.length > 0) {
  results.success.forEach(r => print(`   - ${r.collection}: ${r.description}`));
}

print(`\n⏭️  Already existed: ${results.skipped.length} indexes`);
if (results.skipped.length > 0) {
  results.skipped.forEach(r => print(`   - ${r.collection}: ${r.description}`));
}

print(`\n❌ Failed: ${results.failed.length} indexes`);
if (results.failed.length > 0) {
  results.failed.forEach(r => print(`   - ${r.collection}: ${r.description} (${r.error})`));
}

print("\n====================================");
print("Verification Queries");
print("====================================\n");

print("Run these queries to verify indexes are being used:\n");

print("1. Inventory velocity queries:");
print('   db.inventory.find({ tenantId: "T1", velocityClass: "A" }).explain("executionStats");\n');

print("2. Order date range queries:");
print('   db.orders.find({ tenantId: "T1", createdAt: { $gte: new Date("2025-01-01") } }).explain("executionStats");\n');

print("3. Worker skill queries:");
print('   db.workers.find({ tenantId: "T1", status: "available", "skills.type": "picking" }).explain("executionStats");\n');

print("Look for winningPlan.stage === 'IXSCAN' (good)");
print("Avoid winningPlan.stage === 'COLLSCAN' (bad - table scan)\n");

print("====================================");
print("Index Optimization Complete!");
print("====================================");
