// MongoDB Migration Script: Add Multi-Tenant Fields
// Run with: mongosh mongodb://localhost:27017 001_add_tenant_fields.js
// Or: mongo mongodb://localhost:27017 001_add_tenant_fields.js

const DEFAULT_TENANT = "DEFAULT_TENANT";
const DEFAULT_FACILITY = "DEFAULT_FACILITY";
const DEFAULT_WAREHOUSE = "DEFAULT_WAREHOUSE";

print("=== WMS Platform Multi-Tenant Migration ===");
print("Starting migration at: " + new Date().toISOString());

// ============================================
// Order Service Database
// ============================================
print("\n--- Migrating order_service database ---");
const orderDb = db.getSiblingDB("order_service");

// Add tenant fields to orders collection
print("Updating orders collection...");
const orderResult = orderDb.orders.updateMany(
  { tenantId: { $exists: false } },
  {
    $set: {
      tenantId: DEFAULT_TENANT,
      facilityId: DEFAULT_FACILITY,
      warehouseId: DEFAULT_WAREHOUSE,
      sellerId: null,
      channelId: null,
      externalOrderId: null
    }
  }
);
print(`  Updated ${orderResult.modifiedCount} orders`);

// Create compound indexes for orders
print("Creating indexes on orders collection...");
orderDb.orders.createIndex(
  { tenantId: 1, facilityId: 1, orderId: 1 },
  { unique: true, name: "idx_tenant_facility_order", background: true }
);
orderDb.orders.createIndex(
  { tenantId: 1, sellerId: 1, status: 1, createdAt: -1 },
  { name: "idx_tenant_seller_status", background: true }
);
orderDb.orders.createIndex(
  { tenantId: 1, channelId: 1, externalOrderId: 1 },
  { name: "idx_tenant_channel_external", background: true }
);
orderDb.orders.createIndex(
  { sellerId: 1, status: 1 },
  { name: "idx_seller_status", background: true, sparse: true }
);
print("  Indexes created successfully");

// ============================================
// Inventory Service Database
// ============================================
print("\n--- Migrating inventory_service database ---");
const inventoryDb = db.getSiblingDB("inventory_service");

// Add tenant fields to inventory collection
print("Updating inventory collection...");
const inventoryResult = inventoryDb.inventory.updateMany(
  { tenantId: { $exists: false } },
  {
    $set: {
      tenantId: DEFAULT_TENANT,
      facilityId: DEFAULT_FACILITY,
      warehouseId: DEFAULT_WAREHOUSE,
      sellerId: null
    }
  }
);
print(`  Updated ${inventoryResult.modifiedCount} inventory records`);

// Create compound indexes for inventory
print("Creating indexes on inventory collection...");
inventoryDb.inventory.createIndex(
  { tenantId: 1, facilityId: 1, sku: 1 },
  { unique: true, name: "idx_tenant_facility_sku", background: true }
);
inventoryDb.inventory.createIndex(
  { tenantId: 1, sellerId: 1, sku: 1 },
  { name: "idx_tenant_seller_sku", background: true }
);
inventoryDb.inventory.createIndex(
  { sellerId: 1, warehouseId: 1 },
  { name: "idx_seller_warehouse", background: true, sparse: true }
);
print("  Indexes created successfully");

// ============================================
// Unit Service Database
// ============================================
print("\n--- Migrating unit_service database ---");
const unitDb = db.getSiblingDB("unit_service");

// Add tenant fields to units collection
print("Updating units collection...");
const unitResult = unitDb.units.updateMany(
  { tenantId: { $exists: false } },
  {
    $set: {
      tenantId: DEFAULT_TENANT,
      facilityId: DEFAULT_FACILITY,
      warehouseId: DEFAULT_WAREHOUSE,
      sellerId: null
    }
  }
);
print(`  Updated ${unitResult.modifiedCount} units`);

// Create compound indexes for units
print("Creating indexes on units collection...");
unitDb.units.createIndex(
  { tenantId: 1, facilityId: 1, unitId: 1 },
  { unique: true, name: "idx_tenant_facility_unit", background: true }
);
unitDb.units.createIndex(
  { tenantId: 1, sellerId: 1, status: 1 },
  { name: "idx_tenant_seller_status", background: true }
);
unitDb.units.createIndex(
  { sellerId: 1, sku: 1 },
  { name: "idx_seller_sku", background: true, sparse: true }
);
print("  Indexes created successfully");

// ============================================
// Facility Service Database
// ============================================
print("\n--- Migrating facility_service database ---");
const facilityDb = db.getSiblingDB("facility_service");

// Add tenant fields to stations collection
print("Updating stations collection...");
const stationResult = facilityDb.stations.updateMany(
  { tenantId: { $exists: false } },
  {
    $set: {
      tenantId: DEFAULT_TENANT,
      facilityId: DEFAULT_FACILITY
    }
  }
);
print(`  Updated ${stationResult.modifiedCount} stations`);

// Add tenant fields to facilities collection
print("Updating facilities collection...");
const facilityResult = facilityDb.facilities.updateMany(
  { tenantId: { $exists: false } },
  {
    $set: {
      tenantId: DEFAULT_TENANT
    }
  }
);
print(`  Updated ${facilityResult.modifiedCount} facilities`);

// Create indexes for facilities
print("Creating indexes on facility collections...");
facilityDb.stations.createIndex(
  { tenantId: 1, facilityId: 1, stationId: 1 },
  { unique: true, name: "idx_tenant_facility_station", background: true }
);
facilityDb.facilities.createIndex(
  { tenantId: 1, facilityId: 1 },
  { unique: true, name: "idx_tenant_facility", background: true }
);
print("  Indexes created successfully");

// ============================================
// Seller Service Database (new)
// ============================================
print("\n--- Setting up seller_service database ---");
const sellerDb = db.getSiblingDB("seller_service");

// Create indexes for sellers
print("Creating indexes on sellers collection...");
sellerDb.sellers.createIndex(
  { sellerId: 1 },
  { unique: true, name: "idx_seller_id", background: true }
);
sellerDb.sellers.createIndex(
  { tenantId: 1, status: 1 },
  { name: "idx_tenant_status", background: true }
);
sellerDb.sellers.createIndex(
  { "companyInfo.email": 1 },
  { unique: true, name: "idx_email", background: true }
);
sellerDb.sellers.createIndex(
  { "apiKeys.keyHash": 1 },
  { name: "idx_api_key_hash", background: true, sparse: true }
);
print("  Indexes created successfully");

// ============================================
// Billing Service Database (new)
// ============================================
print("\n--- Setting up billing_service database ---");
const billingDb = db.getSiblingDB("billing_service");

// Create indexes for billable_activities
print("Creating indexes on billable_activities collection...");
billingDb.billable_activities.createIndex(
  { activityId: 1 },
  { unique: true, name: "idx_activity_id", background: true }
);
billingDb.billable_activities.createIndex(
  { sellerId: 1, occurredAt: -1 },
  { name: "idx_seller_date", background: true }
);
billingDb.billable_activities.createIndex(
  { sellerId: 1, invoiced: 1, occurredAt: 1 },
  { name: "idx_seller_uninvoiced", background: true }
);
billingDb.billable_activities.createIndex(
  { invoiceId: 1 },
  { name: "idx_invoice", background: true, sparse: true }
);

// Create indexes for invoices
print("Creating indexes on invoices collection...");
billingDb.invoices.createIndex(
  { invoiceId: 1 },
  { unique: true, name: "idx_invoice_id", background: true }
);
billingDb.invoices.createIndex(
  { invoiceNumber: 1 },
  { unique: true, name: "idx_invoice_number", background: true }
);
billingDb.invoices.createIndex(
  { sellerId: 1, status: 1, periodEnd: -1 },
  { name: "idx_seller_status", background: true }
);

// Create indexes for storage_calculations
print("Creating indexes on storage_calculations collection...");
billingDb.storage_calculations.createIndex(
  { calculationId: 1 },
  { unique: true, name: "idx_calculation_id", background: true }
);
billingDb.storage_calculations.createIndex(
  { sellerId: 1, calculationDate: -1 },
  { name: "idx_seller_date", background: true }
);
print("  Indexes created successfully");

// ============================================
// Channel Service Database (new)
// ============================================
print("\n--- Setting up channel_service database ---");
const channelDb = db.getSiblingDB("channel_service");

// Create indexes for channels
print("Creating indexes on channels collection...");
channelDb.channels.createIndex(
  { channelId: 1 },
  { unique: true, name: "idx_channel_id", background: true }
);
channelDb.channels.createIndex(
  { sellerId: 1 },
  { name: "idx_seller", background: true }
);
channelDb.channels.createIndex(
  { sellerId: 1, type: 1 },
  { name: "idx_seller_type", background: true }
);
channelDb.channels.createIndex(
  { status: 1, "syncSettings.orderSync.lastSyncAt": 1 },
  { name: "idx_status_sync", background: true }
);

// Create indexes for channel_orders
print("Creating indexes on channel_orders collection...");
channelDb.channel_orders.createIndex(
  { externalOrderId: 1 },
  { unique: true, name: "idx_external_order", background: true }
);
channelDb.channel_orders.createIndex(
  { channelId: 1, externalOrderId: 1 },
  { name: "idx_channel_external", background: true }
);
channelDb.channel_orders.createIndex(
  { channelId: 1, importedToWMS: 1 },
  { name: "idx_channel_imported", background: true }
);
channelDb.channel_orders.createIndex(
  { sellerId: 1, orderDate: -1 },
  { name: "idx_seller_date", background: true }
);

// Create indexes for sync_jobs
print("Creating indexes on sync_jobs collection...");
channelDb.sync_jobs.createIndex(
  { jobId: 1 },
  { unique: true, name: "idx_job_id", background: true }
);
channelDb.sync_jobs.createIndex(
  { channelId: 1, type: 1, status: 1 },
  { name: "idx_channel_type_status", background: true }
);
print("  Indexes created successfully");

// ============================================
// Summary
// ============================================
print("\n=== Migration Complete ===");
print("Completed at: " + new Date().toISOString());
print("\nSummary:");
print(`  - Orders updated: ${orderResult.modifiedCount}`);
print(`  - Inventory updated: ${inventoryResult.modifiedCount}`);
print(`  - Units updated: ${unitResult.modifiedCount}`);
print(`  - Stations updated: ${stationResult.modifiedCount}`);
print(`  - Facilities updated: ${facilityResult.modifiedCount}`);
print("\nAll indexes created successfully.");
print("\nNote: Run this script in a maintenance window for production databases.");
