import { createUnits } from '../lib/unit.js';

/**
 * Setup Units Script
 * Creates units for all test SKUs so that workflows can reserve them
 */

// Load SKU data
const skusData = JSON.parse(open('../../data/skus.json'));

export const options = {
  iterations: 1,
  vus: 1,
};

export default function () {
  console.log('Creating units for all SKUs...');

  let successCount = 0;
  let failureCount = 0;

  // Create units for each SKU
  for (const product of skusData.products) {
    // Create a reasonable quantity of units based on reorder quantity
    // Use a smaller number to avoid creating too many units
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

    const result = createUnits(
      product.sku,
      'SHP-SETUP-ALL-SKUS',
      location,
      quantity,
      'setup-script'
    );

    if (result.success) {
      successCount++;
    } else {
      failureCount++;
      console.warn(`✗ Failed to create units for ${product.sku}`);
    }
  }

  console.log(`\n✓ Setup complete! Created units for ${successCount} SKUs, ${failureCount} failures`);
}
