// K6 Stow Locations Helper Library
// Provides functions for managing storage locations in the stow service
//
// NOTE: Storage locations are primarily initialized via MongoDB scripts:
// - deployments/mongodb/init-locations.js: Runs automatically on container startup
// - Scripts below require location API endpoints to be exposed in stow service
//
// Primary method: MongoDB initialization (automatic)
// Secondary method: K6 setup.js script calls initializeDefaultLocations()
//   (requires location API endpoints in stow service)

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, HTTP_PARAMS } from './config.js';

/**
 * Creates a storage location for stow operations
 * @param {Object} locationData - Location details
 * @returns {Object|null} Created location or null if failed
 */
export function createStorageLocation(locationData) {
  const url = `${BASE_URLS.stow}/api/v1/locations`;

  const payload = JSON.stringify({
    locationId: locationData.locationId,
    zone: locationData.zone,
    aisle: locationData.aisle || 1,
    level: locationData.level || 1,
    position: locationData.position || 1,
    capacity: locationData.capacity || 100,
    currentQuantity: locationData.currentQuantity || 0,
    currentWeight: locationData.currentWeight || 0,
    maxWeight: locationData.maxWeight || 500,
    allowsHazmat: locationData.allowsHazmat || false,
    allowsColdChain: locationData.allowsColdChain || false,
    allowsOversized: locationData.allowsOversized || false,
    isActive: locationData.isActive !== false,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'create storage location status 201': (r) => r.status === 201 || r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to create storage location ${locationData.locationId}: ${response.status}`);
    return null;
  }

  try {
    const result = JSON.parse(response.body);
    console.log(`Created storage location: ${result.locationId || locationData.locationId}`);
    return result;
  } catch (e) {
    console.error(`Failed to parse location response: ${e.message}`);
    return null;
  }
}

/**
 * Gets a storage location by ID
 * @param {string} locationId - Location ID
 * @returns {Object|null} Location or null if not found
 */
export function getStorageLocation(locationId) {
  const url = `${BASE_URLS.stow}/api/v1/locations/${locationId}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'get storage location status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to get storage location ${locationId}: ${response.status}`);
    return null;
  }

  try {
    return JSON.parse(response.body);
  } catch (e) {
    console.error(`Failed to parse location response: ${e.message}`);
    return null;
  }
}

/**
 * Lists all storage locations
 * @param {Object} filters - Filter options
 * @returns {Array} Array of locations
 */
export function listStorageLocations(filters = {}) {
  let url = `${BASE_URLS.stow}/api/v1/locations`;
  const params = [];

  if (filters.zone) {
    params.push(`zone=${encodeURIComponent(filters.zone)}`);
  }
  if (filters.limit) {
    params.push(`limit=${filters.limit}`);
  }

  if (params.length > 0) {
    url += '?' + params.join('&');
  }

  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'list storage locations status 200': (r) => r.status === 200,
  });

  if (!success) {
    console.warn(`Failed to list storage locations: ${response.status}`);
    return [];
  }

  try {
    const data = JSON.parse(response.body);
    return Array.isArray(data) ? data : (data.locations || data.items || []);
  } catch (e) {
    console.error(`Failed to parse locations response: ${e.message}`);
    return [];
  }
}

/**
 * Checks if storage locations are available
 * @returns {boolean} True if locations exist
 */
export function checkStorageLocationsAvailable() {
  const locations = listStorageLocations({ limit: 1 });
  return locations.length > 0;
}

/**
 * Gets available capacity for a zone
 * @param {string} zone - Zone name
 * @returns {number} Total available capacity
 */
export function getZoneCapacity(zone) {
  const locations = listStorageLocations({ zone: zone });
  let totalCapacity = 0;
  let usedCapacity = 0;

  for (const loc of locations) {
    totalCapacity += loc.capacity || 0;
    usedCapacity += loc.currentQuantity || 0;
  }

  return totalCapacity - usedCapacity;
}

/**
 * Initializes default warehouse locations
 * Creates standard location set if none exist
 * @returns {Object} Summary of created locations
 */
export function initializeDefaultLocations() {
  console.log('Checking for existing storage locations...');

  if (checkStorageLocationsAvailable()) {
    console.log('Storage locations already exist, skipping initialization');
    return { skipped: true };
  }

  console.log('Initializing default storage locations...');

  const summary = {
    created: 0,
    failed: 0,
    zones: {
      RESERVE: 0,
      FORWARD_PICK: 0,
      OVERFLOW: 0,
    },
  };

  // Zone: RESERVE (primary long-term storage)
  for (let i = 1; i <= 50; i++) {
    const locationId = `LOC-RESERVE-${String(i).padStart(3, '0')}`;
    const aisle = Math.ceil(i / 10);
    const result = createStorageLocation({
      locationId: locationId,
      zone: 'RESERVE',
      aisle: aisle,
      level: ((i - 1) % 5) + 1,
      position: ((i - 1) % 10) + 1,
      capacity: 100,
      maxWeight: 500,
      allowsHazmat: true,
      allowsColdChain: false,
      allowsOversized: true,
    });

    if (result) {
      summary.created++;
      summary.zones.RESERVE++;
    } else {
      summary.failed++;
    }
  }

  // Zone: FORWARD_PICK (high-velocity picking area)
  for (let i = 1; i <= 30; i++) {
    const locationId = `LOC-PICK-${String(i).padStart(3, '0')}`;
    const aisle = Math.ceil(i / 10);
    const result = createStorageLocation({
      locationId: locationId,
      zone: 'FORWARD_PICK',
      aisle: aisle,
      level: ((i - 1) % 5) + 1,
      position: ((i - 1) % 10) + 1,
      capacity: 50,
      maxWeight: 300,
      allowsHazmat: false,
      allowsColdChain: true,
      allowsOversized: false,
    });

    if (result) {
      summary.created++;
      summary.zones.FORWARD_PICK++;
    } else {
      summary.failed++;
    }
  }

  // Zone: OVERFLOW (bulk storage)
  for (let i = 1; i <= 15; i++) {
    const locationId = `LOC-OVERFLOW-${String(i).padStart(3, '0')}`;
    const aisle = Math.ceil(i / 5);
    const result = createStorageLocation({
      locationId: locationId,
      zone: 'OVERFLOW',
      aisle: aisle,
      level: ((i - 1) % 3) + 1,
      position: ((i - 1) % 5) + 1,
      capacity: 200,
      maxWeight: 1000,
      allowsHazmat: true,
      allowsColdChain: true,
      allowsOversized: true,
    });

    if (result) {
      summary.created++;
      summary.zones.OVERFLOW++;
    } else {
      summary.failed++;
    }
  }

  console.log(`Initialization complete: ${summary.created} created, ${summary.failed} failed`);
  console.log(`  RESERVE: ${summary.zones.RESERVE}`);
  console.log(`  FORWARD_PICK: ${summary.zones.FORWARD_PICK}`);
  console.log(`  OVERFLOW: ${summary.zones.OVERFLOW}`);

  return summary;
}
