// Facility Simulator - K6 Station Management Scenario
// Sets up and manages stations for the warehouse simulation
//
// Usage:
//   k6 run scripts/scenarios/facility-simulator.js
//   k6 run -e FACILITY_SERVICE_URL=http://localhost:8010 scripts/scenarios/facility-simulator.js
//
// Environment variables:
//   FACILITY_SERVICE_URL  - Facility service URL (default: http://localhost:8010)
//   FACILITY_DELAY_MS     - Delay between operations (default: 300)
//   MAX_STATIONS          - Max stations per iteration (default: 20)
//   CLEANUP_STATIONS      - Set to 'true' to cleanup test stations in teardown

import { sleep } from 'k6';
import { Counter, Rate, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { FACILITY_CONFIG } from '../lib/config.js';
import {
  listStations,
  getStation,
  createStation,
  updateStation,
  deleteStation,
  setCapabilities,
  addCapability,
  removeCapability,
  activateStation,
  deactivateStation,
  setMaintenance,
  getStationsByZone,
  getStationsByType,
  getStationsByStatus,
  findCapableStations,
  checkHealth,
} from '../lib/facility.js';

// Load station test data
const stationData = new SharedArray('stations', function () {
  const data = JSON.parse(open('../../data/stations.json'));
  return data.stations;
});

// Custom metrics
const stationsCreated = new Counter('facility_stations_created');
const stationsActive = new Gauge('facility_stations_active');
const capabilityChanges = new Counter('facility_capability_changes');
const statusChanges = new Counter('facility_status_changes');
const apiSuccessRate = new Rate('facility_api_success_rate');
const stationsDeleted = new Counter('facility_stations_deleted');
const stationsUpdated = new Counter('facility_stations_updated');

// Default options
export const options = {
  scenarios: {
    facility_setup: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '5m',
    },
  },
  thresholds: {
    facility_api_success_rate: ['rate>0.95'],
    http_req_duration: ['p(95)<500'],
  },
};

/**
 * Setup function - health check and initial state
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Facility Simulator Starting');
  console.log('='.repeat(60));
  console.log(`Stations to create: ${stationData.length}`);
  console.log(`Simulation delay: ${FACILITY_CONFIG.simulationDelayMs}ms`);
  console.log('='.repeat(60));

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    console.error('Facility service is not healthy!');
  }

  // Get existing stations
  const existingResult = listStations(100, 0);
  const existingStations = existingResult.stations || [];

  console.log(`Existing stations: ${existingStations.length}`);

  return {
    startTime: new Date().toISOString(),
    existingStationIds: existingStations.map((s) => s.stationId),
    stationsToCreate: stationData.length,
  };
}

/**
 * Main iteration - create and manage stations
 */
export default function (data) {
  const delaySeconds = FACILITY_CONFIG.simulationDelayMs / 1000;
  const createdStations = [];

  console.log('\n' + '='.repeat(50));
  console.log('Phase 1: Creating Stations');
  console.log('='.repeat(50));

  // Create stations from test data
  for (const station of stationData) {
    // Skip if already exists
    if (data.existingStationIds.includes(station.stationId)) {
      console.log(`Station ${station.stationId} already exists, skipping`);
      continue;
    }

    const result = createStation(station);
    apiSuccessRate.add(result.success);

    if (result.success) {
      stationsCreated.add(1);
      createdStations.push(station.stationId);
      console.log(`Created station: ${station.stationId} (${station.stationType})`);
    } else {
      console.error(`Failed to create station ${station.stationId}: ${result.status}`);
    }

    sleep(delaySeconds);
  }

  console.log(`\nCreated ${createdStations.length} stations`);

  // Phase 2: Activate all stations
  console.log('\n' + '='.repeat(50));
  console.log('Phase 2: Activating Stations');
  console.log('='.repeat(50));

  let activeCount = 0;
  for (const stationId of createdStations) {
    const result = activateStation(stationId);
    apiSuccessRate.add(result.success);

    if (result.success) {
      activeCount++;
      statusChanges.add(1);
      console.log(`Activated: ${stationId}`);
    }

    sleep(delaySeconds / 2);
  }

  stationsActive.add(activeCount);
  console.log(`\nActivated ${activeCount} stations`);

  // Phase 3: Verify station capabilities
  console.log('\n' + '='.repeat(50));
  console.log('Phase 3: Verifying Capabilities');
  console.log('='.repeat(50));

  // Find gift wrap stations
  const giftWrapResult = findCapableStations(['gift_wrap'], 'packing', '');
  apiSuccessRate.add(giftWrapResult.success);
  console.log(`Gift wrap stations found: ${giftWrapResult.stations.length}`);

  // Find packing stations
  const packingResult = getStationsByType('packing');
  apiSuccessRate.add(packingResult.success);
  console.log(`Packing stations found: ${packingResult.stations.length}`);

  // Find consolidation stations
  const consolResult = getStationsByType('consolidation');
  apiSuccessRate.add(consolResult.success);
  console.log(`Consolidation stations found: ${consolResult.stations.length}`);

  // Find shipping stations
  const shipResult = getStationsByType('shipping');
  apiSuccessRate.add(shipResult.success);
  console.log(`Shipping stations found: ${shipResult.stations.length}`);

  sleep(delaySeconds);

  // Phase 4: Test capability management
  console.log('\n' + '='.repeat(50));
  console.log('Phase 4: Testing Capability Management');
  console.log('='.repeat(50));

  // Pick a station to test capability changes
  if (createdStations.length > 0) {
    const testStationId = createdStations[0];

    // Add a capability
    const addResult = addCapability(testStationId, 'test_capability');
    apiSuccessRate.add(addResult.success);
    if (addResult.success) {
      capabilityChanges.add(1);
      console.log(`Added test_capability to ${testStationId}`);
    }

    sleep(delaySeconds);

    // Remove the capability
    const removeResult = removeCapability(testStationId, 'test_capability');
    apiSuccessRate.add(removeResult.success);
    if (removeResult.success) {
      capabilityChanges.add(1);
      console.log(`Removed test_capability from ${testStationId}`);
    }
  }

  // Phase 5: Test status transitions
  console.log('\n' + '='.repeat(50));
  console.log('Phase 5: Testing Status Transitions');
  console.log('='.repeat(50));

  if (createdStations.length > 1) {
    const testStationId = createdStations[1];

    // Set to maintenance
    const maintResult = setMaintenance(testStationId);
    apiSuccessRate.add(maintResult.success);
    if (maintResult.success) {
      statusChanges.add(1);
      console.log(`Set ${testStationId} to maintenance`);
    }

    sleep(delaySeconds);

    // Set back to active
    const activeResult = activateStation(testStationId);
    apiSuccessRate.add(activeResult.success);
    if (activeResult.success) {
      statusChanges.add(1);
      console.log(`Reactivated ${testStationId}`);
    }
  }

  // Phase 6: Query by zone
  console.log('\n' + '='.repeat(50));
  console.log('Phase 6: Querying by Zone');
  console.log('='.repeat(50));

  const zones = ['zone-a', 'zone-b', 'zone-c', 'zone-d', 'zone-e'];
  for (const zone of zones) {
    const zoneResult = getStationsByZone(zone);
    apiSuccessRate.add(zoneResult.success);
    console.log(`Zone ${zone}: ${zoneResult.stations.length} stations`);
    sleep(delaySeconds / 3);
  }

  // Phase 7: Query active stations
  console.log('\n' + '='.repeat(50));
  console.log('Phase 7: Querying Active Stations');
  console.log('='.repeat(50));

  const activeStationsResult = getStationsByStatus('active');
  apiSuccessRate.add(activeStationsResult.success);
  console.log(`Active stations: ${activeStationsResult.stations.length}`);

  // Summary
  console.log('\n' + '='.repeat(50));
  console.log('Facility Setup Summary');
  console.log('='.repeat(50));
  console.log(`Stations created: ${createdStations.length}`);
  console.log(`Active stations: ${activeStationsResult.stations.length}`);
  console.log(`Gift wrap capable: ${giftWrapResult.stations.length}`);
  console.log('='.repeat(50));

  return {
    createdStations,
    giftWrapStations: giftWrapResult.stations.map((s) => s.stationId),
    packingStations: packingResult.stations.map((s) => s.stationId),
    consolidationStations: consolResult.stations.map((s) => s.stationId),
    shippingStations: shipResult.stations.map((s) => s.stationId),
  };
}

/**
 * Teardown function - cleanup and report
 */
export function teardown(data) {
  console.log('\n' + '='.repeat(60));
  console.log('Facility Simulator Complete');
  console.log('='.repeat(60));
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);

  // Optionally cleanup test stations
  const cleanup = __ENV.CLEANUP_STATIONS === 'true';
  if (cleanup) {
    console.log('\nCleaning up test stations...');
    for (const station of stationData) {
      const result = deleteStation(station.stationId);
      if (result.success) {
        console.log(`Deleted: ${station.stationId}`);
      }
    }
  } else {
    console.log('\nStations preserved for further testing');
    console.log('Set CLEANUP_STATIONS=true to cleanup on next run');
  }

  console.log('='.repeat(60));
}
