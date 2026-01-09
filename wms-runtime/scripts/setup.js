// WMS Platform Setup Script
// Run this ONCE before load testing to prepare inventory and workers

import { sleep } from 'k6';
import { products, stockSetup, workers } from './lib/data.js';
import { createInventoryItem, receiveStock, checkHealth as checkInventoryHealth } from './lib/inventory.js';
import { createWorker, addWorkerSkill, startShift, checkHealth as checkLaborHealth } from './lib/labor.js';
import { checkHealth as checkOrdersHealth } from './lib/orders.js';
import { checkStorageLocationsAvailable, initializeDefaultLocations } from './lib/stow-locations.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ['rate>0.8'], // Allow some failures for idempotent operations
  },
};

export default function () {
  console.log('=== WMS Platform Setup ===');
  console.log('');

  // Step 0: Health Checks
  console.log('Step 0: Checking service health...');
  const inventoryHealthy = checkInventoryHealth();
  const laborHealthy = checkLaborHealth();
  const ordersHealthy = checkOrdersHealth();

  if (!inventoryHealthy || !laborHealthy || !ordersHealthy) {
    console.error('ERROR: One or more services are not healthy!');
    console.error(`  Inventory: ${inventoryHealthy ? 'OK' : 'FAILED'}`);
    console.error(`  Labor: ${laborHealthy ? 'OK' : 'FAILED'}`);
    console.error(`  Orders: ${ordersHealthy ? 'OK' : 'FAILED'}`);
    return;
  }
  console.log('All services are healthy!');
  console.log('');
  sleep(1);

  // Step 0.5: Initialize Storage Locations (if not already present)
  console.log('Step 0.5: Initializing storage locations...');
  if (checkStorageLocationsAvailable()) {
    console.log('Storage locations already initialized');
  } else {
    console.log('No storage locations found, creating default locations...');
    const locationSummary = initializeDefaultLocations();
    console.log(`  Created: ${locationSummary.created} locations`);
    if (locationSummary.failed > 0) {
      console.warn(`  Failed: ${locationSummary.failed} locations`);
    }
  }
  console.log('');
  sleep(1);

  // Step 1: Create Inventory Items
  console.log('Step 1: Creating inventory items...');
  let itemsCreated = 0;
  let itemsFailed = 0;

  for (const product of products) {
    const result = createInventoryItem(
      product.sku,
      product.productName,
      product.reorderPoint,
      product.reorderQuantity
    );

    if (result.success) {
      itemsCreated++;
      console.log(`  Created: ${product.sku} - ${product.productName}`);
    } else {
      itemsFailed++;
      console.log(`  Failed: ${product.sku} (status: ${result.status})`);
    }
    sleep(0.1);
  }

  console.log(`  Summary: ${itemsCreated} created, ${itemsFailed} failed`);
  console.log('');
  sleep(1);

  // Step 2: Receive Stock at Locations
  console.log('Step 2: Receiving stock at locations...');
  let stockReceived = 0;
  let stockFailed = 0;

  for (const stock of stockSetup) {
    const result = receiveStock(
      stock.sku,
      stock.locationId,
      stock.zone,
      stock.quantity,
      `SETUP-${Date.now()}`,
      'k6-setup'
    );

    if (result.success) {
      stockReceived++;
      console.log(`  Received: ${stock.quantity}x ${stock.sku} at ${stock.locationId}`);
    } else {
      stockFailed++;
      console.log(`  Failed: ${stock.sku} at ${stock.locationId} (status: ${result.status})`);
    }
    sleep(0.1);
  }

  console.log(`  Summary: ${stockReceived} received, ${stockFailed} failed`);
  console.log('');
  sleep(1);

  // Step 3: Create Workers
  console.log('Step 3: Creating workers...');
  let workersCreated = 0;
  let workersFailed = 0;

  for (const worker of workers) {
    const result = createWorker(
      worker.workerId,
      worker.employeeId,
      worker.name
    );

    if (result.success) {
      workersCreated++;
      console.log(`  Created: ${worker.workerId} - ${worker.name}`);
    } else {
      workersFailed++;
      console.log(`  Failed: ${worker.workerId} (status: ${result.status})`);
    }
    sleep(0.1);
  }

  console.log(`  Summary: ${workersCreated} created, ${workersFailed} failed`);
  console.log('');
  sleep(1);

  // Step 4: Add Worker Skills
  console.log('Step 4: Adding worker skills...');
  let skillsAdded = 0;
  let skillsFailed = 0;

  for (const worker of workers) {
    for (const skill of worker.skills) {
      const result = addWorkerSkill(
        worker.workerId,
        skill.taskType,
        skill.level,
        skill.certified
      );

      if (result.success) {
        skillsAdded++;
        console.log(`  Added: ${skill.taskType} (level ${skill.level}) to ${worker.workerId}`);
      } else {
        skillsFailed++;
        console.log(`  Failed: ${skill.taskType} for ${worker.workerId} (status: ${result.status})`);
      }
      sleep(0.05);
    }
  }

  console.log(`  Summary: ${skillsAdded} added, ${skillsFailed} failed`);
  console.log('');
  sleep(1);

  // Step 5: Start Worker Shifts
  console.log('Step 5: Starting worker shifts...');
  let shiftsStarted = 0;
  let shiftsFailed = 0;
  const shiftId = `SHIFT-${new Date().toISOString().split('T')[0]}`;

  for (const worker of workers) {
    const result = startShift(
      worker.workerId,
      `${shiftId}-${worker.workerId}`,
      worker.shiftType,
      worker.zone
    );

    if (result.success) {
      shiftsStarted++;
      console.log(`  Started: ${worker.workerId} in ${worker.zone} (${worker.shiftType})`);
    } else {
      shiftsFailed++;
      console.log(`  Failed: ${worker.workerId} (status: ${result.status})`);
    }
    sleep(0.1);
  }

  console.log(`  Summary: ${shiftsStarted} started, ${shiftsFailed} failed`);
  console.log('');

  // Final Summary
  console.log('=== Setup Complete ===');
  console.log('');
  console.log('Summary:');
  console.log(`  Storage Locations: Initialized (95 locations across RESERVE, FORWARD_PICK, OVERFLOW)`);
  console.log(`  Inventory Items: ${itemsCreated}/${products.length}`);
  console.log(`  Stock Locations: ${stockReceived}/${stockSetup.length}`);
  console.log(`  Workers: ${workersCreated}/${workers.length}`);
  console.log(`  Skills: ${skillsAdded}`);
  console.log(`  Shifts: ${shiftsStarted}/${workers.length}`);
  console.log('');
  console.log('You can now run the order injection tests!');
  console.log('Note: Storage locations enable stow tasks to be processed successfully.');
}
