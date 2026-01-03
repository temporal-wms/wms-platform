// Data generators for WMS Platform load testing
import { ORDER_CONFIG, MULTI_ROUTE_CONFIG } from './config.js';

// Load static data
const skusData = JSON.parse(open('../../data/skus.json'));
const locationsData = JSON.parse(open('../../data/locations.json'));
const workersData = JSON.parse(open('../../data/workers.json'));

export const products = skusData.products;
export const requirementDefinitions = skusData.requirementDefinitions || {};
export const zones = locationsData.zones;
export const stockSetup = locationsData.stockSetup;
export const workers = workersData.workers;

// Priority weights: standard (70%), next_day (25%), same_day (5%)
const PRIORITY_WEIGHTS = [
  { value: 'standard', weight: 70 },
  { value: 'next_day', weight: 25 },
  { value: 'same_day', weight: 5 },
];

// US States for address generation
const US_STATES = [
  'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
  'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
  'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
  'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
  'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY',
];

const CITIES = [
  { city: 'New York', state: 'NY', zipCode: '10001' },
  { city: 'Los Angeles', state: 'CA', zipCode: '90001' },
  { city: 'Chicago', state: 'IL', zipCode: '60601' },
  { city: 'Houston', state: 'TX', zipCode: '77001' },
  { city: 'Phoenix', state: 'AZ', zipCode: '85001' },
  { city: 'Philadelphia', state: 'PA', zipCode: '19101' },
  { city: 'San Antonio', state: 'TX', zipCode: '78201' },
  { city: 'San Diego', state: 'CA', zipCode: '92101' },
  { city: 'Dallas', state: 'TX', zipCode: '75201' },
  { city: 'Seattle', state: 'WA', zipCode: '98101' },
  { city: 'Denver', state: 'CO', zipCode: '80201' },
  { city: 'Boston', state: 'MA', zipCode: '02101' },
  { city: 'Atlanta', state: 'GA', zipCode: '30301' },
  { city: 'Miami', state: 'FL', zipCode: '33101' },
  { city: 'Portland', state: 'OR', zipCode: '97201' },
];

const STREET_NAMES = [
  'Main St', 'Oak Ave', 'Maple Dr', 'Cedar Ln', 'Pine Rd',
  'Elm St', 'Washington Blvd', 'Park Ave', 'Lake Dr', 'River Rd',
  'Highland Ave', 'Forest Dr', 'Valley Rd', 'Spring St', 'Hill Rd',
];

const FIRST_NAMES = [
  'James', 'Mary', 'John', 'Patricia', 'Robert', 'Jennifer', 'Michael',
  'Linda', 'William', 'Barbara', 'David', 'Elizabeth', 'Richard', 'Susan',
  'Joseph', 'Jessica', 'Thomas', 'Sarah', 'Charles', 'Karen',
];

const LAST_NAMES = [
  'Smith', 'Johnson', 'Williams', 'Brown', 'Jones', 'Garcia', 'Miller',
  'Davis', 'Rodriguez', 'Martinez', 'Hernandez', 'Lopez', 'Gonzalez',
  'Wilson', 'Anderson', 'Thomas', 'Taylor', 'Moore', 'Jackson', 'Martin',
];

// Utility functions
function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randomElement(array) {
  return array[Math.floor(Math.random() * array.length)];
}

function weightedRandom(items) {
  const totalWeight = items.reduce((sum, item) => sum + item.weight, 0);
  let random = Math.random() * totalWeight;

  for (const item of items) {
    random -= item.weight;
    if (random <= 0) {
      return item.value;
    }
  }
  return items[items.length - 1].value;
}

function generateUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// Public generators
export function generateCustomerId() {
  return `CUST-${generateUUID().substring(0, 8).toUpperCase()}`;
}

export function generateOrderReference() {
  return `REF-${Date.now()}-${randomInt(1000, 9999)}`;
}

export function selectPriority() {
  return weightedRandom(PRIORITY_WEIGHTS);
}

export function selectRandomProducts(count = 1) {
  const maxCount = Math.min(count, products.length);
  const shuffled = [...products].sort(() => 0.5 - Math.random());
  return shuffled.slice(0, maxCount);
}

export function generateAddress() {
  const location = randomElement(CITIES);
  const streetNumber = randomInt(100, 9999);
  const streetName = randomElement(STREET_NAMES);
  const firstName = randomElement(FIRST_NAMES);
  const lastName = randomElement(LAST_NAMES);

  return {
    street: `${streetNumber} ${streetName}`,
    city: location.city,
    state: location.state,
    zipCode: location.zipCode,
    country: 'US',
    recipientName: `${firstName} ${lastName}`,
    phone: `+1${randomInt(200, 999)}${randomInt(100, 999)}${randomInt(1000, 9999)}`,
  };
}

export function generatePromisedDeliveryDate(priority) {
  const now = new Date();
  let daysToAdd;

  switch (priority) {
    case 'same_day':
      daysToAdd = 0;
      break;
    case 'next_day':
      daysToAdd = 1;
      break;
    case 'standard':
    default:
      daysToAdd = randomInt(3, 7);
      break;
  }

  const deliveryDate = new Date(now);
  deliveryDate.setDate(deliveryDate.getDate() + daysToAdd);
  deliveryDate.setHours(23, 59, 59, 0);

  return deliveryDate.toISOString();
}

export function generateOrderItems(count = 1) {
  const selectedProducts = selectRandomProducts(count);

  return selectedProducts.map(product => ({
    sku: product.sku,
    name: product.productName,
    quantity: randomInt(1, 3),
    weight: product.weight,
  }));
}

export function generateOrder() {
  const customerId = generateCustomerId();
  const priority = selectPriority();
  const itemCount = randomInt(1, 3);
  const items = generateOrderItems(itemCount);
  const address = generateAddress();
  const promisedDeliveryAt = generatePromisedDeliveryDate(priority);

  return {
    customerId,
    items,
    shippingAddress: address,
    priority,
    promisedDeliveryAt,
  };
}

export function getRandomWorker() {
  return randomElement(workers);
}

export function getRandomZone() {
  return randomElement(zones);
}

export function getRandomLocation(zone) {
  if (zone) {
    const zoneData = zones.find(z => z.zone === zone);
    if (zoneData) {
      return randomElement(zoneData.locations);
    }
  }
  const allLocations = zones.flatMap(z => z.locations);
  return randomElement(allLocations);
}

// ============================================================================
// Order Type and Requirement-Based Generation
// ============================================================================

/**
 * Get item count based on order type configuration
 * @param {string|null} orderType - 'single', 'multi', or null for random distribution
 * @returns {number} Number of items for the order
 */
export function getOrderItemCount(orderType = null) {
  // Check for forced order type from config
  const effectiveType = orderType || ORDER_CONFIG.forceOrderType;

  if (effectiveType === 'single') {
    return 1;
  }
  if (effectiveType === 'multi') {
    return randomInt(2, ORDER_CONFIG.maxItemsPerOrder);
  }

  // Random distribution based on configured ratios
  if (Math.random() < ORDER_CONFIG.singleItemRatio) {
    return 1;
  }
  return randomInt(2, ORDER_CONFIG.maxItemsPerOrder);
}

/**
 * Shuffle array and take first N elements
 * @param {Array} array - Array to shuffle
 * @param {number} count - Number of elements to take
 * @returns {Array} Shuffled subset
 */
function shuffleAndTake(array, count) {
  const shuffled = [...array].sort(() => 0.5 - Math.random());
  return shuffled.slice(0, Math.min(count, shuffled.length));
}

/**
 * Select products that have a specific requirement
 * @param {string} requirement - Requirement to filter by (hazmat, fragile, etc.)
 * @param {number} count - Number of products to select
 * @returns {Array} Products with the specified requirement
 */
export function selectProductsWithRequirement(requirement, count = 1) {
  const matching = products.filter(p =>
    p.requirements && p.requirements.includes(requirement)
  );

  if (matching.length === 0) {
    console.warn(`No products found with requirement: ${requirement}`);
    return selectRandomProducts(count);
  }

  return shuffleAndTake(matching, count);
}

/**
 * Select products by category
 * @param {string} category - Category to filter by
 * @param {number} count - Number of products to select
 * @returns {Array} Products in the specified category
 */
export function selectProductsByCategory(category, count = 1) {
  const matching = products.filter(p => p.category === category);

  if (matching.length === 0) {
    console.warn(`No products found in category: ${category}`);
    return selectRandomProducts(count);
  }

  return shuffleAndTake(matching, count);
}

/**
 * Aggregate requirements from order items
 * @param {Array} items - Order items with sku property
 * @returns {Array} Unique requirements from all items
 */
export function aggregateOrderRequirements(items) {
  const requirements = new Set();

  for (const item of items) {
    const product = products.find(p => p.sku === item.sku);
    if (product?.requirements) {
      product.requirements.forEach(r => requirements.add(r));
    }
  }

  return Array.from(requirements);
}

/**
 * Generate order items with at least one product having the specified requirement
 * @param {string} requirement - Required requirement for at least one item
 * @param {number} count - Total number of items
 * @returns {Array} Order items
 */
export function generateOrderItemsWithRequirement(requirement, count = 1) {
  // Ensure at least one item has the requirement
  const requiredProducts = selectProductsWithRequirement(requirement, 1);

  if (count === 1) {
    return requiredProducts.map(product => ({
      sku: product.sku,
      name: product.productName,
      quantity: randomInt(1, 3),
      weight: product.weight,
    }));
  }

  // For multi-item orders, add more products (can be any)
  const remainingCount = count - 1;
  const additionalProducts = selectRandomProducts(remainingCount);
  const allProducts = [...requiredProducts, ...additionalProducts];

  return allProducts.map(product => ({
    sku: product.sku,
    name: product.productName,
    quantity: randomInt(1, 3),
    weight: product.weight,
  }));
}

/**
 * Generate an order with specific type and optional requirement
 * @param {string|null} orderType - 'single', 'multi', or null for random
 * @param {string|null} forceRequirement - Force a specific requirement
 * @returns {Object} Generated order with requirements
 */
export function generateOrderWithType(orderType = null, forceRequirement = null) {
  const customerId = generateCustomerId();
  const priority = selectPriority();
  const itemCount = getOrderItemCount(orderType);

  // Determine effective requirement
  const effectiveRequirement = forceRequirement || ORDER_CONFIG.forceRequirement;

  // Generate items
  let items;
  if (effectiveRequirement) {
    items = generateOrderItemsWithRequirement(effectiveRequirement, itemCount);
  } else {
    items = generateOrderItems(itemCount);
  }

  // Aggregate requirements from all items
  const requirements = aggregateOrderRequirements(items);

  const address = generateAddress();
  const promisedDeliveryAt = generatePromisedDeliveryDate(priority);

  return {
    customerId,
    items,
    shippingAddress: address,
    priority,
    promisedDeliveryAt,
    requirements,                                          // Order-level requirements
    orderType: itemCount === 1 ? 'single_item' : 'multi_item',  // Order type indicator
  };
}

/**
 * Get available requirements from the loaded data
 * @returns {Array} List of valid requirements
 */
export function getAvailableRequirements() {
  return Object.keys(requirementDefinitions);
}

/**
 * Get products count by requirement
 * @returns {Object} Map of requirement to product count
 */
export function getProductCountByRequirement() {
  const counts = {};

  for (const req of ORDER_CONFIG.validRequirements) {
    counts[req] = products.filter(p =>
      p.requirements && p.requirements.includes(req)
    ).length;
  }

  return counts;
}

// ============================================================================
// Multi-Route Order Generation
// ============================================================================

/**
 * Zone mapping for location-based splitting
 * First letter of aisle determines zone
 */
const ZONE_MAPPING = {
  'A': 'ZONE-1', 'B': 'ZONE-1', 'C': 'ZONE-1', 'D': 'ZONE-1',
  'E': 'ZONE-2', 'F': 'ZONE-2', 'G': 'ZONE-2', 'H': 'ZONE-2',
  'I': 'ZONE-3', 'J': 'ZONE-3', 'K': 'ZONE-3', 'L': 'ZONE-3',
  'M': 'ZONE-4', 'N': 'ZONE-4', 'O': 'ZONE-4', 'P': 'ZONE-4',
  'Q': 'ZONE-5', 'R': 'ZONE-5', 'S': 'ZONE-5', 'T': 'ZONE-5',
  'U': 'ZONE-6', 'V': 'ZONE-6', 'W': 'ZONE-6', 'X': 'ZONE-6',
  'Y': 'ZONE-6', 'Z': 'ZONE-6',
};

/**
 * Get zone for a location
 * @param {Object} location - Location object with aisle property
 * @returns {string} Zone identifier
 */
export function getZoneForLocation(location) {
  if (!location || !location.aisle) {
    return 'ZONE-1';
  }
  const firstLetter = location.aisle.charAt(0).toUpperCase();
  return ZONE_MAPPING[firstLetter] || 'ZONE-1';
}

/**
 * Select products distributed across multiple zones for multi-route testing
 * @param {number} count - Total number of products to select
 * @param {number} minZones - Minimum number of zones to spread across
 * @returns {Array} Products from multiple zones
 */
export function selectProductsAcrossZones(count, minZones = 2) {
  // Group locations by zone
  const locationsByZone = {};
  for (const zone of zones) {
    for (const loc of zone.locations || []) {
      const zoneName = getZoneForLocation(loc);
      if (!locationsByZone[zoneName]) {
        locationsByZone[zoneName] = [];
      }
      locationsByZone[zoneName].push(loc);
    }
  }

  const zoneNames = Object.keys(locationsByZone);
  if (zoneNames.length < minZones) {
    // Not enough zones, fall back to random selection
    return selectRandomProducts(count);
  }

  // Select zones to use
  const shuffledZones = [...zoneNames].sort(() => 0.5 - Math.random());
  const selectedZones = shuffledZones.slice(0, Math.min(minZones, shuffledZones.length));

  // Distribute items across selected zones
  const itemsPerZone = Math.ceil(count / selectedZones.length);
  const selectedProducts = [];

  for (const zoneName of selectedZones) {
    // Get products from this zone (based on available locations)
    const zoneLocations = locationsByZone[zoneName];
    const zoneProductsCount = Math.min(itemsPerZone, count - selectedProducts.length);

    // Select random products and assign zone locations
    const zoneProducts = selectRandomProducts(zoneProductsCount).map(product => ({
      ...product,
      locationHint: zoneLocations[Math.floor(Math.random() * zoneLocations.length)],
    }));

    selectedProducts.push(...zoneProducts);
  }

  return selectedProducts.slice(0, count);
}

/**
 * Generate a large order that will trigger multi-route splitting
 * @param {number} itemCount - Number of items (should be > maxItemsPerRoute)
 * @param {boolean} spreadAcrossZones - Whether to distribute items across zones
 * @returns {Object} Generated large order
 */
export function generateLargeOrder(itemCount = null, spreadAcrossZones = true) {
  const customerId = generateCustomerId();
  const priority = selectPriority();

  // Default to triggering multi-route by exceeding capacity
  const effectiveItemCount = itemCount || MULTI_ROUTE_CONFIG.largeOrderItemCount;

  // Select products
  let selectedProducts;
  if (spreadAcrossZones) {
    selectedProducts = selectProductsAcrossZones(effectiveItemCount, 3);
  } else {
    selectedProducts = selectRandomProducts(effectiveItemCount);
  }

  const items = selectedProducts.map(product => ({
    sku: product.sku,
    name: product.productName,
    quantity: 1, // Keep quantity low to have more distinct items
    weight: product.weight,
    locationHint: product.locationHint,
  }));

  const requirements = aggregateOrderRequirements(items);
  const address = generateAddress();
  const promisedDeliveryAt = generatePromisedDeliveryDate(priority);

  return {
    customerId,
    items,
    shippingAddress: address,
    priority,
    promisedDeliveryAt,
    requirements,
    orderType: 'large_multi_route',
    expectedRoutes: Math.ceil(effectiveItemCount / MULTI_ROUTE_CONFIG.maxItemsPerRoute),
  };
}

/**
 * Generate order items with zone hints for multi-route testing
 * @param {number} count - Number of items
 * @param {boolean} spreadAcrossZones - Whether to spread across zones
 * @returns {Array} Order items with zone hints
 */
export function generateOrderItemsWithZones(count, spreadAcrossZones = true) {
  const selectedProducts = spreadAcrossZones
    ? selectProductsAcrossZones(count, 2)
    : selectRandomProducts(count);

  return selectedProducts.map(product => ({
    sku: product.sku,
    name: product.productName,
    quantity: randomInt(1, 2),
    weight: product.weight,
    locationHint: product.locationHint,
  }));
}

/**
 * Check if an order would trigger multi-route based on item count
 * @param {Object} order - Order object with items array
 * @returns {boolean} True if order would trigger multi-route
 */
export function wouldTriggerMultiRoute(order) {
  if (!MULTI_ROUTE_CONFIG.enableMultiRoute) {
    return false;
  }

  const totalItems = order.items?.reduce((sum, item) => sum + (item.quantity || 1), 0) || 0;
  return totalItems > MULTI_ROUTE_CONFIG.maxItemsPerRoute;
}

/**
 * Estimate the number of routes for an order
 * @param {Object} order - Order object with items array
 * @returns {number} Estimated number of routes
 */
export function estimateRouteCount(order) {
  if (!MULTI_ROUTE_CONFIG.enableMultiRoute) {
    return 1;
  }

  const totalItems = order.items?.reduce((sum, item) => sum + (item.quantity || 1), 0) || 0;
  return Math.max(1, Math.ceil(totalItems / MULTI_ROUTE_CONFIG.maxItemsPerRoute));
}
