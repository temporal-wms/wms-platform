// MongoDB Initialization Script - Storage Locations
// This script runs automatically when MongoDB container starts
// It creates the storage_locations collection with predefined warehouse locations

(function() {
  const dbName = 'stow_db';
  const db = db.getSiblingDB(dbName);

  // Check if collection already has data
  const count = db.storage_locations.countDocuments({});
  if (count > 0) {
    print(`[INFO] storage_locations collection already has ${count} documents, skipping initialization`);
    return;
  }

  print('[INFO] Initializing storage_locations collection...');

  const now = new Date();
  const locations = [];

  // Helper function to create locations
  function createLocations(zonePrefix, zoneType, count, capacity, maxWeight, features) {
    for (let i = 1; i <= count; i++) {
      const locationId = `${zonePrefix}-${String(i).padStart(3, '0')}`;
      const aisle = Math.ceil(i / 10); // Group 10 locations per aisle

      locations.push({
        locationId: locationId,
        zone: zoneType,
        aisle: aisle,
        level: ((i - 1) % 5) + 1, // 5 levels per aisle
        position: ((i - 1) % 10) + 1, // 10 positions per level
        capacity: capacity,
        currentQuantity: 0,
        currentWeight: 0,
        maxWeight: maxWeight,
        allowsHazmat: features.hazmat || false,
        allowsColdChain: features.coldChain || false,
        allowsOversized: features.oversized || false,
        isActive: true,
        createdAt: now,
        updatedAt: now,
      });
    }
  }

  // Zone: RESERVE (primary long-term storage)
  // 50 locations, capacity 100 units each, supports standard and oversized
  createLocations('LOC-RESERVE', 'RESERVE', 50, 100, 500, {
    hazmat: true,
    coldChain: false,
    oversized: true,
  });

  // Zone: FORWARD_PICK (high-velocity picking area)
  // 30 locations, capacity 50 units each, supports cold chain
  createLocations('LOC-PICK', 'FORWARD_PICK', 30, 50, 300, {
    hazmat: false,
    coldChain: true,
    oversized: false,
  });

  // Zone: OVERFLOW (bulk storage for high-demand items)
  // 15 locations, capacity 200 units each, supports all types
  createLocations('LOC-OVERFLOW', 'OVERFLOW', 15, 200, 1000, {
    hazmat: true,
    coldChain: true,
    oversized: true,
  });

  print(`[INFO] Total locations to create: ${locations.length}`);

  try {
    // Insert all locations
    const result = db.storage_locations.insertMany(locations, { ordered: false });
    print(`[SUCCESS] Created ${result.insertedIds.length} storage locations`);

    // Create indexes for performance
    db.storage_locations.createIndex({ locationId: 1 }, { unique: true });
    db.storage_locations.createIndex({ zone: 1 });
    db.storage_locations.createIndex({ zone: 1, aisle: 1 });
    db.storage_locations.createIndex({
      $expr: {
        $gte: [
          { $subtract: ['$capacity', '$currentQuantity'] },
          0
        ]
      }
    });
    print('[INFO] Created indexes on storage_locations');

  } catch (error) {
    print(`[ERROR] Failed to insert storage locations: ${error.message}`);
    throw error;
  }

  // Verify the data
  const inserted = db.storage_locations.countDocuments({});
  print(`[INFO] Verification: storage_locations now contains ${inserted} documents`);

  // Show distribution by zone
  const distribution = db.storage_locations.aggregate([
    { $group: { _id: '$zone', count: { $sum: 1 } } },
    { $sort: { _id: 1 } }
  ]).toArray();

  print('[INFO] Location distribution by zone:');
  distribution.forEach(function(doc) {
    print(`       ${doc._id}: ${doc.count} locations`);
  });
})();
