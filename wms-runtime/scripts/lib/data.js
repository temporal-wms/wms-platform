// Data generators for WMS Platform load testing

// Load static data
const skusData = JSON.parse(open('../../data/skus.json'));
const locationsData = JSON.parse(open('../../data/locations.json'));
const workersData = JSON.parse(open('../../data/workers.json'));

export const products = skusData.products;
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
