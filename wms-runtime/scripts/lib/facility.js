// Facility Service API helpers for K6 load testing
import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS, FACILITY_CONFIG } from './config.js';

const baseUrl = BASE_URLS.facility;

// ============================================================================
// Station Discovery
// ============================================================================

/**
 * List all stations with optional pagination
 */
export function listStations(limit = 50, offset = 0) {
  const url = `${baseUrl}${ENDPOINTS.facility.stations}?limit=${limit}&offset=${offset}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'stations listed': (r) => r.status === 200,
  });

  let body = [];
  try {
    body = response.json();
  } catch {
    body = [];
  }

  return {
    success,
    status: response.status,
    stations: body,
  };
}

/**
 * Get a station by ID
 */
export function getStation(stationId) {
  const url = `${baseUrl}${ENDPOINTS.facility.get(stationId)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'station retrieved': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Get stations by zone
 */
export function getStationsByZone(zone) {
  const url = `${baseUrl}${ENDPOINTS.facility.byZone(zone)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'stations by zone retrieved': (r) => r.status === 200,
  });

  let body = [];
  try {
    body = response.json();
  } catch {
    body = [];
  }

  return {
    success,
    status: response.status,
    stations: body,
  };
}

/**
 * Get stations by type
 */
export function getStationsByType(stationType) {
  const url = `${baseUrl}${ENDPOINTS.facility.byType(stationType)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'stations by type retrieved': (r) => r.status === 200,
  });

  let body = [];
  try {
    body = response.json();
  } catch {
    body = [];
  }

  return {
    success,
    status: response.status,
    stations: body,
  };
}

/**
 * Get stations by status
 */
export function getStationsByStatus(status) {
  const url = `${baseUrl}${ENDPOINTS.facility.byStatus(status)}`;
  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'stations by status retrieved': (r) => r.status === 200,
  });

  let body = [];
  try {
    body = response.json();
  } catch {
    body = [];
  }

  return {
    success,
    status: response.status,
    stations: body,
  };
}

/**
 * Find stations with required capabilities
 */
export function findCapableStations(requirements, stationType = '', zone = '') {
  const url = `${baseUrl}${ENDPOINTS.facility.findCapable}`;
  const payload = JSON.stringify({
    requirements,
    stationType,
    zone,
  });

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'capable stations found': (r) => r.status === 200,
  });

  let body = [];
  try {
    body = response.json();
  } catch {
    body = [];
  }

  return {
    success,
    status: response.status,
    stations: body,
  };
}

// ============================================================================
// Station Management
// ============================================================================

/**
 * Create a new station
 */
export function createStation(stationData) {
  const url = `${baseUrl}${ENDPOINTS.facility.stations}`;
  const payload = JSON.stringify(stationData);

  const response = http.post(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'station created': (r) => r.status === 201 || r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Update a station
 */
export function updateStation(stationId, updates) {
  const url = `${baseUrl}${ENDPOINTS.facility.update(stationId)}`;
  const payload = JSON.stringify(updates);

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'station updated': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Delete a station
 */
export function deleteStation(stationId) {
  const url = `${baseUrl}${ENDPOINTS.facility.delete(stationId)}`;
  const response = http.del(url, null, HTTP_PARAMS);

  const success = check(response, {
    'station deleted': (r) => r.status === 204 || r.status === 200,
  });

  return {
    success,
    status: response.status,
  };
}

// ============================================================================
// Capability Management
// ============================================================================

/**
 * Set all capabilities for a station
 */
export function setCapabilities(stationId, capabilities) {
  const url = `${baseUrl}${ENDPOINTS.facility.capabilities(stationId)}`;
  const payload = JSON.stringify({ capabilities });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'capabilities set': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Add a capability to a station
 */
export function addCapability(stationId, capability) {
  const url = `${baseUrl}${ENDPOINTS.facility.addCapability(stationId, capability)}`;
  const response = http.post(url, '{}', HTTP_PARAMS);

  const success = check(response, {
    'capability added': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Remove a capability from a station
 */
export function removeCapability(stationId, capability) {
  const url = `${baseUrl}${ENDPOINTS.facility.removeCapability(stationId, capability)}`;
  const response = http.del(url, null, HTTP_PARAMS);

  const success = check(response, {
    'capability removed': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

// ============================================================================
// Status Management
// ============================================================================

/**
 * Set station status
 */
export function setStationStatus(stationId, status) {
  const url = `${baseUrl}${ENDPOINTS.facility.status(stationId)}`;
  const payload = JSON.stringify({ status });

  const response = http.put(url, payload, HTTP_PARAMS);

  const success = check(response, {
    'station status set': (r) => r.status === 200,
  });

  let body = null;
  try {
    body = response.json();
  } catch {
    body = null;
  }

  return {
    success,
    status: response.status,
    station: body,
  };
}

/**
 * Activate a station
 */
export function activateStation(stationId) {
  return setStationStatus(stationId, 'active');
}

/**
 * Deactivate a station
 */
export function deactivateStation(stationId) {
  return setStationStatus(stationId, 'inactive');
}

/**
 * Set station to maintenance mode
 */
export function setMaintenance(stationId) {
  return setStationStatus(stationId, 'maintenance');
}

// ============================================================================
// Capacity Helpers
// ============================================================================

/**
 * Find an available station with required capabilities
 * Returns the station with the most available capacity
 */
export function findAvailableStation(requirements, stationType = '', zone = '') {
  const result = findCapableStations(requirements, stationType, zone);

  if (!result.success || !result.stations || result.stations.length === 0) {
    console.warn(`No stations found with capabilities: ${requirements.join(', ')}`);
    return null;
  }

  // Filter by active status and available capacity
  const available = result.stations.filter((station) => {
    const isActive = station.status === 'active';
    const currentTasks = station.currentTasks || 0;
    const maxTasks = station.maxConcurrentTasks || 1;
    const hasCapacity = currentTasks < maxTasks;
    return isActive && hasCapacity;
  });

  if (available.length === 0) {
    console.warn('No available stations with required capabilities and capacity');
    return null;
  }

  // Sort by available capacity (descending) for load balancing
  available.sort((a, b) => {
    const aAvailable = (a.maxConcurrentTasks || 1) - (a.currentTasks || 0);
    const bAvailable = (b.maxConcurrentTasks || 1) - (b.currentTasks || 0);
    return bAvailable - aAvailable;
  });

  return available[0];
}

/**
 * Get station capacity info
 */
export function getStationCapacity(stationId) {
  const result = getStation(stationId);

  if (!result.success || !result.station) {
    return null;
  }

  const station = result.station;
  const maxTasks = station.maxConcurrentTasks || 1;
  const currentTasks = station.currentTasks || 0;

  return {
    stationId: station.stationId,
    maxConcurrentTasks: maxTasks,
    currentTasks: currentTasks,
    availableCapacity: maxTasks - currentTasks,
    utilizationPercent: Math.round((currentTasks / maxTasks) * 100),
    isAvailable: currentTasks < maxTasks && station.status === 'active',
  };
}

// ============================================================================
// Health Check
// ============================================================================

/**
 * Check facility service health
 */
export function checkHealth() {
  const response = http.get(`${baseUrl}/health`);

  return check(response, {
    'facility service healthy': (r) => r.status === 200,
  });
}

/**
 * Check facility service readiness
 */
export function checkReady() {
  const response = http.get(`${baseUrl}/ready`);

  return check(response, {
    'facility service ready': (r) => r.status === 200,
  });
}

// ============================================================================
// Batch Operations
// ============================================================================

/**
 * Create multiple stations from an array
 */
export function createStations(stationsData) {
  const results = [];
  for (const stationData of stationsData) {
    const result = createStation(stationData);
    results.push({
      stationId: stationData.stationId,
      success: result.success,
      status: result.status,
    });
  }
  return results;
}

/**
 * Activate all stations in a list
 */
export function activateStations(stationIds) {
  const results = [];
  for (const stationId of stationIds) {
    const result = activateStation(stationId);
    results.push({
      stationId,
      success: result.success,
    });
  }
  return results;
}

/**
 * Get all active packing stations with gift_wrap capability
 */
export function getGiftWrapStations() {
  const result = findCapableStations(['gift_wrap'], 'packing', '');

  if (!result.success) {
    return [];
  }

  return result.stations.filter((s) => s.status === 'active');
}
