// MongoDB Schema Validation Rules
// Enforces data integrity and constraints at database level
//
// Usage: mongosh mongodb://localhost:27017/temporal_war add-schema-validation.js

print("====================================");
print("MongoDB Schema Validation Setup");
print("Adding validation rules for data integrity");
print("====================================\n");

const db = db.getSiblingDB('temporal_war');

let results = {
  success: [],
  failed: []
};

function addValidation(collectionName, validationRules, description) {
  try {
    print(`Adding validation: ${description}`);
    print(`  Collection: ${collectionName}\n`);

    db.runCommand({
      collMod: collectionName,
      validator: validationRules,
      validationLevel: "moderate", // moderate = only validate inserts and updates to valid docs
      validationAction: "error" // error = reject invalid documents
    });

    print(`  Status: ✅ Validation rules applied\n`);
    results.success.push({ collection: collectionName, description });
  } catch (error) {
    print(`  Status: ❌ Failed - ${error.message}\n`);
    results.failed.push({ collection: collectionName, description, error: error.message });
  }
}

print("=== 1. Inventory Collection ===\n");

addValidation(
  "inventory",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["sku", "tenantId", "facilityId", "warehouseId", "totalQuantity", "availableQuantity"],
      properties: {
        sku: {
          bsonType: "string",
          minLength: 1,
          maxLength: 100,
          description: "SKU must be a non-empty string (max 100 chars)"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1,
          description: "Tenant ID is required for multi-tenancy"
        },
        facilityId: {
          bsonType: "string",
          minLength: 1,
          description: "Facility ID is required"
        },
        warehouseId: {
          bsonType: "string",
          minLength: 1,
          description: "Warehouse ID is required"
        },
        totalQuantity: {
          bsonType: "int",
          minimum: 0,
          description: "Total quantity must be non-negative integer"
        },
        reservedQuantity: {
          bsonType: "int",
          minimum: 0,
          description: "Reserved quantity must be non-negative integer"
        },
        hardAllocatedQuantity: {
          bsonType: "int",
          minimum: 0,
          description: "Hard allocated quantity must be non-negative integer"
        },
        availableQuantity: {
          bsonType: "int",
          minimum: 0,
          description: "Available quantity must be non-negative integer"
        },
        velocityClass: {
          enum: ["A", "B", "C"],
          description: "Velocity class must be A, B, or C"
        },
        storageStrategy: {
          enum: ["chaotic", "directed", "velocity"],
          description: "Storage strategy must be chaotic, directed, or velocity"
        },
        pickFrequency: {
          bsonType: "int",
          minimum: 0,
          description: "Pick frequency must be non-negative"
        }
      }
    }
  },
  "Inventory validation: quantities, enums, required fields"
);

print("=== 2. Orders Collection ===\n");

addValidation(
  "orders",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["orderId", "tenantId", "facilityId", "status", "priority"],
      properties: {
        orderId: {
          bsonType: "string",
          minLength: 1,
          maxLength: 100,
          description: "Order ID is required"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1,
          description: "Tenant ID is required"
        },
        facilityId: {
          bsonType: "string",
          minLength: 1,
          description: "Facility ID is required"
        },
        status: {
          enum: [
            "received", "validated", "wave_assigned", "picking",
            "consolidated", "packed", "shipped", "delivered",
            "cancelled", "pending_retry", "dead_letter"
          ],
          description: "Status must be a valid order status"
        },
        priority: {
          enum: ["same_day", "next_day", "standard"],
          description: "Priority must be same_day, next_day, or standard"
        },
        items: {
          bsonType: "array",
          minItems: 1,
          description: "Order must have at least one item",
          items: {
            bsonType: "object",
            required: ["sku", "quantity"],
            properties: {
              sku: {
                bsonType: "string",
                minLength: 1
              },
              quantity: {
                bsonType: "int",
                minimum: 1,
                description: "Item quantity must be at least 1"
              },
              pickedQty: {
                bsonType: "int",
                minimum: 0
              }
            }
          }
        },
        promisedDeliveryAt: {
          bsonType: "date",
          description: "Promised delivery date must be a valid date"
        }
      }
    }
  },
  "Orders validation: status, priority, items array"
);

print("=== 3. Workers Collection ===\n");

addValidation(
  "workers",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["workerId", "tenantId", "facilityId", "status"],
      properties: {
        workerId: {
          bsonType: "string",
          minLength: 1,
          description: "Worker ID is required"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1,
          description: "Tenant ID is required"
        },
        facilityId: {
          bsonType: "string",
          minLength: 1,
          description: "Facility ID is required"
        },
        status: {
          enum: ["available", "on_task", "on_break", "offline"],
          description: "Status must be available, on_task, on_break, or offline"
        },
        skills: {
          bsonType: "array",
          description: "Skills must be an array",
          items: {
            bsonType: "object",
            required: ["type", "level"],
            properties: {
              type: {
                enum: [
                  "picking", "packing", "receiving", "consolidation",
                  "replenishment", "walling", "sorting", "loading"
                ],
                description: "Skill type must be valid"
              },
              level: {
                bsonType: "int",
                minimum: 1,
                maximum: 5,
                description: "Skill level must be 1-5"
              },
              certified: {
                bsonType: "bool"
              }
            }
          }
        }
      }
    }
  },
  "Workers validation: status, skill types and levels"
);

print("=== 4. Waves Collection ===\n");

addValidation(
  "waves",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["waveId", "tenantId", "facilityId", "waveType", "status"],
      properties: {
        waveId: {
          bsonType: "string",
          minLength: 1,
          description: "Wave ID is required"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1
        },
        facilityId: {
          bsonType: "string",
          minLength: 1
        },
        waveType: {
          enum: ["digital", "wholesale", "priority", "mixed"],
          description: "Wave type must be digital, wholesale, priority, or mixed"
        },
        status: {
          enum: [
            "planning", "scheduled", "released", "in_progress",
            "completed", "cancelled"
          ],
          description: "Status must be valid wave status"
        },
        fulfillmentMode: {
          enum: ["wave", "waveless", "hybrid"],
          description: "Fulfillment mode must be wave, waveless, or hybrid"
        },
        priority: {
          bsonType: "int",
          minimum: 1,
          description: "Priority must be positive integer (1 = highest)"
        }
      }
    }
  },
  "Waves validation: type, status, fulfillment mode"
);

print("=== 5. Shipments Collection ===\n");

addValidation(
  "shipments",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["shipmentId", "tenantId", "facilityId", "status"],
      properties: {
        shipmentId: {
          bsonType: "string",
          minLength: 1,
          description: "Shipment ID is required"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1
        },
        facilityId: {
          bsonType: "string",
          minLength: 1
        },
        status: {
          enum: [
            "pending", "labeled", "manifested", "shipped",
            "delivered", "cancelled"
          ],
          description: "Status must be valid shipment status"
        },
        carrier: {
          bsonType: "object",
          required: ["code", "name"],
          properties: {
            code: {
              bsonType: "string",
              minLength: 1
            },
            name: {
              bsonType: "string",
              minLength: 1
            }
          }
        }
      }
    }
  },
  "Shipments validation: status, carrier structure"
);

print("=== 6. Inventory Transactions Collection ===\n");

addValidation(
  "inventory_transactions",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["transactionId", "sku", "tenantId", "type", "quantity", "createdAt"],
      properties: {
        transactionId: {
          bsonType: "string",
          minLength: 1,
          description: "Transaction ID is required"
        },
        sku: {
          bsonType: "string",
          minLength: 1,
          description: "SKU is required"
        },
        tenantId: {
          bsonType: "string",
          minLength: 1
        },
        type: {
          enum: [
            "receive", "pick", "adjust", "transfer", "ship",
            "shortage", "return_to_shelf"
          ],
          description: "Transaction type must be valid"
        },
        quantity: {
          bsonType: "int",
          description: "Quantity is required (can be negative for outbound)"
        },
        createdAt: {
          bsonType: "date",
          description: "Created date is required"
        }
      }
    }
  },
  "Transactions validation: type enum, required fields"
);

print("=== 7. Inventory Reservations Collection ===\n");

addValidation(
  "inventory_reservations",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["reservationId", "sku", "orderId", "quantity", "status", "expiresAt"],
      properties: {
        reservationId: {
          bsonType: "string",
          minLength: 1,
          description: "Reservation ID is required"
        },
        sku: {
          bsonType: "string",
          minLength: 1
        },
        orderId: {
          bsonType: "string",
          minLength: 1
        },
        quantity: {
          bsonType: "int",
          minimum: 1,
          description: "Quantity must be at least 1"
        },
        status: {
          enum: ["active", "staged", "fulfilled", "cancelled", "expired"],
          description: "Status must be valid reservation status"
        },
        expiresAt: {
          bsonType: "date",
          description: "Expiration date is required"
        }
      }
    }
  },
  "Reservations validation: status enum, expiration"
);

print("=== 8. Inventory Allocations Collection ===\n");

addValidation(
  "inventory_allocations",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["allocationId", "sku", "orderId", "quantity", "status"],
      properties: {
        allocationId: {
          bsonType: "string",
          minLength: 1,
          description: "Allocation ID is required"
        },
        sku: {
          bsonType: "string",
          minLength: 1
        },
        orderId: {
          bsonType: "string",
          minLength: 1
        },
        quantity: {
          bsonType: "int",
          minimum: 1,
          description: "Quantity must be at least 1"
        },
        status: {
          enum: ["staged", "packed", "shipped", "returned"],
          description: "Status must be valid allocation status"
        }
      }
    }
  },
  "Allocations validation: status enum, quantities"
);

print("=== 9. Outbox Events Collection ===\n");

addValidation(
  "outbox_events",
  {
    $jsonSchema: {
      bsonType: "object",
      required: ["_id", "aggregateId", "aggregateType", "eventType", "topic", "createdAt"],
      properties: {
        _id: {
          bsonType: "string",
          description: "Event ID is required (UUID)"
        },
        aggregateId: {
          bsonType: "string",
          minLength: 1,
          description: "Aggregate ID is required"
        },
        aggregateType: {
          bsonType: "string",
          minLength: 1,
          description: "Aggregate type is required"
        },
        eventType: {
          bsonType: "string",
          minLength: 1,
          description: "Event type is required"
        },
        topic: {
          bsonType: "string",
          minLength: 1,
          description: "Kafka topic is required"
        },
        retryCount: {
          bsonType: "int",
          minimum: 0,
          maximum: 10,
          description: "Retry count must be 0-10"
        },
        maxRetries: {
          bsonType: "int",
          minimum: 1,
          description: "Max retries must be positive"
        },
        createdAt: {
          bsonType: "date",
          description: "Created date is required"
        },
        publishedAt: {
          bsonType: "date",
          description: "Published date (nullable)"
        }
      }
    }
  },
  "Outbox events validation: required fields, retry limits"
);

print("\n====================================");
print("Schema Validation Summary");
print("====================================\n");

print(`✅ Successfully applied: ${results.success.length} validations`);
if (results.success.length > 0) {
  results.success.forEach(r => print(`   - ${r.collection}: ${r.description}`));
}

print(`\n❌ Failed: ${results.failed.length} validations`);
if (results.failed.length > 0) {
  results.failed.forEach(r => print(`   - ${r.collection}: ${r.description} (${r.error})`));
}

print("\n====================================");
print("Testing Validation Rules");
print("====================================\n");

print("Try inserting invalid documents to test:");
print("");
print("1. Invalid inventory (negative quantity):");
print('   db.inventory.insertOne({');
print('     sku: "TEST",');
print('     tenantId: "T1",');
print('     facilityId: "F1",');
print('     warehouseId: "W1",');
print('     totalQuantity: -5  // ❌ Should fail');
print('   });\n');

print("2. Invalid order status:");
print('   db.orders.insertOne({');
print('     orderId: "ORD-001",');
print('     tenantId: "T1",');
print('     facilityId: "F1",');
print('     status: "invalid_status",  // ❌ Should fail');
print('     priority: "standard",');
print('     items: [{ sku: "SKU-1", quantity: 1 }]');
print('   });\n');

print("3. Invalid worker skill level:");
print('   db.workers.insertOne({');
print('     workerId: "W-001",');
print('     tenantId: "T1",');
print('     facilityId: "F1",');
print('     status: "available",');
print('     skills: [{ type: "picking", level: 10 }]  // ❌ Should fail (max is 5)');
print('   });\n');

print("\n====================================");
print("Validation Levels");
print("====================================\n");

print("Current setting: moderate (default)");
print("  - Validates INSERT operations");
print("  - Validates UPDATE operations on documents that already pass validation");
print("  - Does NOT validate existing invalid documents\n");

print("To change validation level:");
print('  db.runCommand({ collMod: "inventory", validationLevel: "strict" });\n');

print("Validation levels:");
print("  - strict: Validates all inserts and updates");
print("  - moderate: Validates inserts and updates to valid docs (recommended)");
print("  - off: No validation\n");

print("\n====================================");
print("Viewing Validation Rules");
print("====================================\n");

print("To view validation rules for a collection:");
print('  db.getCollectionInfos({ name: "inventory" })[0].options.validator\n');

print("\n====================================");
print("Schema Validation Complete!");
print("====================================");
print("");
print("Benefits:");
print("✅ Enforces data integrity at database level");
print("✅ Catches bugs before they cause data corruption");
print("✅ Self-documenting schema constraints");
print("✅ Complements application-level validation");
print("");
print("Note: Validation rules do NOT retroactively validate existing data.");
print("Run data quality checks to find existing invalid documents.");
