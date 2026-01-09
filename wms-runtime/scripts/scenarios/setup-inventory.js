import { createInventoryItem, receiveStock } from '../lib/inventory.js';

/**
 * Setup Inventory Script
 * Creates inventory items for all SKUs so orders can pick them
 */

// Load SKU data
const skusData = JSON.parse(open('../../data/skus.json'));

export const options = {
  iterations: 1,
  vus: 1,
};

export default function () {
  console.log('Creating inventory for all SKUs...');

  let successCount = 0;
  let failureCount = 0;

  // Create inventory for each SKU
  for (const product of skusData.products) {
    // Create a reasonable quantity based on reorder quantity
    const quantity = Math.max(1, Math.ceil(Math.min(product.reorderQuantity / 5, 20)));

    // Use specific locations based on category
    let location = 'LOC-A-01';
    if (product.category === 'monitors' || product.category === 'gaming') {
      location = 'LOC-HEAVY-01'; // Heavy items
    } else if (product.category === 'cables' || product.category === 'cases') {
      location = 'LOC-LIGHT-01'; // Light items
    } else if (product.category === 'phones' || product.category === 'watches') {
      location = 'LOC-HIGH-VALUE-01'; // High-value items
    } else {
      location = 'LOC-A-01'; // Default location
    }

    // First create the inventory item
    createInventoryItem(
      product.sku,
      product.productName,
      product.reorderPoint,
      product.reorderQuantity
    );

    // Then receive stock at the location
    const result = receiveStock(
      product.sku,
      location,
      location.split('-')[0], // Extract zone from location
      quantity,
      'SHP-SETUP-ALL-SKUS',
      'setup-script'
    );

    if (result) {
      successCount++;
    } else {
      failureCount++;
      console.warn(`✗ Failed to receive stock for ${product.sku}`);
    }
  }

  console.log(`\n✓ Setup complete! Created inventory for ${successCount} SKUs, ${failureCount} failures`);
}
