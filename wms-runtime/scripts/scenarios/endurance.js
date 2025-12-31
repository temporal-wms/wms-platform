// WMS Platform - 1 Hour Endurance Test
// Comprehensive test with varying load patterns to understand system behavior
// Includes: normal load, peaks, spikes, recovery periods, and sustained load

import { sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { generateOrder } from '../lib/data.js';
import { createOrder, checkHealth } from '../lib/orders.js';
import { THRESHOLDS } from '../lib/config.js';

// Custom metrics
const ordersCreated = new Counter('orders_created');
const ordersFailed = new Counter('orders_failed');
const orderSuccessRate = new Rate('order_success_rate');
const orderDuration = new Trend('order_duration');

// Phase tracking
const currentPhase = new Gauge('current_phase');

// Priority distribution tracking
const prioritySameDay = new Counter('priority_same_day');
const priorityNextDay = new Counter('priority_next_day');
const priorityStandard = new Counter('priority_standard');

// Error tracking by type
const errorsByStatus = new Counter('errors_by_status');
const timeoutErrors = new Counter('timeout_errors');

export const options = {
  scenarios: {
    // Main endurance scenario with varying load patterns over 1 hour
    endurance_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        // === PHASE 1: Warm-up (5 min) ===
        { duration: '2m', target: 5 },    // Gentle start
        { duration: '3m', target: 10 },   // Warm-up to baseline

        // === PHASE 2: Normal Load (10 min) ===
        { duration: '10m', target: 15 },  // Sustained normal load

        // === PHASE 3: First Peak (5 min) ===
        { duration: '1m', target: 40 },   // Rapid spike
        { duration: '2m', target: 40 },   // Sustain peak
        { duration: '2m', target: 15 },   // Recovery to normal

        // === PHASE 4: Gradual Ramp (8 min) ===
        { duration: '2m', target: 20 },   // Slight increase
        { duration: '2m', target: 30 },   // Building up
        { duration: '2m', target: 25 },   // Slight decrease
        { duration: '2m', target: 20 },   // Stabilize

        // === PHASE 5: Major Peak (6 min) ===
        { duration: '30s', target: 60 },  // Sharp spike!
        { duration: '2m', target: 60 },   // Sustain high load
        { duration: '30s', target: 80 },  // Push higher
        { duration: '2m', target: 50 },   // Partial recovery
        { duration: '1m', target: 20 },   // Full recovery

        // === PHASE 6: Chaotic Load (10 min) ===
        { duration: '1m', target: 35 },   // Unpredictable
        { duration: '30s', target: 15 },  // Drop
        { duration: '1m', target: 45 },   // Spike
        { duration: '30s', target: 25 },  // Drop
        { duration: '1m', target: 55 },   // Higher spike
        { duration: '1m', target: 20 },   // Recovery
        { duration: '1m', target: 40 },   // Back up
        { duration: '30s', target: 10 },  // Low
        { duration: '1m', target: 50 },   // High again
        { duration: '2m', target: 25 },   // Stabilize

        // === PHASE 7: Sustained High Load (8 min) ===
        { duration: '2m', target: 45 },   // Ramp to high
        { duration: '4m', target: 45 },   // Sustained high load
        { duration: '2m', target: 30 },   // Gradual decrease

        // === PHASE 8: Final Stress & Cooldown (8 min) ===
        { duration: '1m', target: 70 },   // Final stress push
        { duration: '2m', target: 70 },   // Hold stress
        { duration: '2m', target: 40 },   // Start cooldown
        { duration: '2m', target: 15 },   // Continue cooldown
        { duration: '1m', target: 0 },    // Complete shutdown
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    // Balanced thresholds for endurance testing
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    http_req_failed: ['rate<0.05'],
    'order_success_rate': ['rate>0.90'],
    'order_duration': ['p(95)<2500', 'p(99)<5000'],
    'http_reqs': ['rate>1'],
  },
};

// Phase descriptions for logging
const PHASES = [
  { name: 'Warm-up', start: 0, end: 5, description: 'Gentle system warm-up' },
  { name: 'Normal Load', start: 5, end: 15, description: 'Baseline sustained load' },
  { name: 'First Peak', start: 15, end: 20, description: 'Initial spike test' },
  { name: 'Gradual Ramp', start: 20, end: 28, description: 'Progressive load increase' },
  { name: 'Major Peak', start: 28, end: 34, description: 'High stress period' },
  { name: 'Chaotic Load', start: 34, end: 44, description: 'Unpredictable load patterns' },
  { name: 'Sustained High', start: 44, end: 52, description: 'Extended high load' },
  { name: 'Final Stress', start: 52, end: 60, description: 'Final push and cooldown' },
];

function getCurrentPhase(elapsedMinutes) {
  for (let i = 0; i < PHASES.length; i++) {
    if (elapsedMinutes >= PHASES[i].start && elapsedMinutes < PHASES[i].end) {
      return { index: i + 1, ...PHASES[i] };
    }
  }
  return { index: PHASES.length, ...PHASES[PHASES.length - 1] };
}

export function setup() {
  console.log('╔══════════════════════════════════════════════════════════════╗');
  console.log('║         WMS PLATFORM - 1 HOUR ENDURANCE TEST                 ║');
  console.log('╠══════════════════════════════════════════════════════════════╣');
  console.log('║ This test simulates realistic traffic patterns over 1 hour   ║');
  console.log('║ to understand system behavior under various conditions.      ║');
  console.log('╚══════════════════════════════════════════════════════════════╝');
  console.log('');
  console.log('Test Phases:');
  console.log('');
  console.log('  Phase 1 (0-5 min):   Warm-up           - 5-10 VUs');
  console.log('  Phase 2 (5-15 min):  Normal Load       - 15 VUs sustained');
  console.log('  Phase 3 (15-20 min): First Peak        - Spike to 40 VUs');
  console.log('  Phase 4 (20-28 min): Gradual Ramp      - 20-30 VUs varied');
  console.log('  Phase 5 (28-34 min): Major Peak        - Up to 80 VUs!');
  console.log('  Phase 6 (34-44 min): Chaotic Load      - Unpredictable 10-55 VUs');
  console.log('  Phase 7 (44-52 min): Sustained High    - 45 VUs sustained');
  console.log('  Phase 8 (52-60 min): Final Stress      - 70 VUs then cooldown');
  console.log('');
  console.log('Total duration: 60 minutes');
  console.log('');

  // Health check
  const healthy = checkHealth();
  if (!healthy) {
    throw new Error('Order service is not healthy! Cannot start endurance test.');
  }
  console.log('✓ Order service is healthy');
  console.log('');
  console.log('Starting endurance test...');
  console.log('═'.repeat(64));
  console.log('');

  return {
    startTime: Date.now(),
    lastPhaseLogged: 0,
  };
}

export default function (data) {
  // Calculate elapsed time
  const elapsedMs = Date.now() - data.startTime;
  const elapsedMinutes = elapsedMs / 60000;

  // Get current phase
  const phase = getCurrentPhase(elapsedMinutes);
  currentPhase.add(phase.index);

  // Generate a random order
  const order = generateOrder();

  // Track priority distribution
  switch (order.priority) {
    case 'same_day':
      prioritySameDay.add(1);
      break;
    case 'next_day':
      priorityNextDay.add(1);
      break;
    case 'standard':
      priorityStandard.add(1);
      break;
  }

  // Create the order
  const startTime = Date.now();
  const result = createOrder(order);
  const duration = Date.now() - startTime;

  // Record metrics
  orderDuration.add(duration);

  if (result.success && result.orderId) {
    ordersCreated.add(1);
    orderSuccessRate.add(1);
  } else {
    ordersFailed.add(1);
    orderSuccessRate.add(0);

    // Track error types
    if (result.status) {
      errorsByStatus.add(1, { status: result.status.toString() });
    }
    if (duration > 29000) {
      timeoutErrors.add(1);
    }

    // Log errors occasionally (1 in 20)
    if (Math.random() < 0.05) {
      console.log(`[${phase.name}] Error: status=${result.status}, VU=${__VU}, duration=${duration}ms`);
    }
  }

  // Dynamic sleep based on phase
  // Normal phases: longer sleep, Peak phases: shorter sleep
  let sleepTime;
  switch (phase.name) {
    case 'Major Peak':
    case 'Final Stress':
      sleepTime = 0.1 + Math.random() * 0.2;  // 100-300ms (aggressive)
      break;
    case 'Chaotic Load':
      sleepTime = 0.2 + Math.random() * 0.3;  // 200-500ms (varied)
      break;
    case 'First Peak':
    case 'Sustained High':
      sleepTime = 0.3 + Math.random() * 0.3;  // 300-600ms (moderate)
      break;
    default:
      sleepTime = 0.4 + Math.random() * 0.4;  // 400-800ms (normal)
  }

  sleep(sleepTime);
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  const durationMinutes = duration / 60;

  console.log('');
  console.log('═'.repeat(64));
  console.log('');
  console.log('╔══════════════════════════════════════════════════════════════╗');
  console.log('║              ENDURANCE TEST COMPLETE                         ║');
  console.log('╚══════════════════════════════════════════════════════════════╝');
  console.log('');
  console.log(`Total duration: ${durationMinutes.toFixed(2)} minutes`);
  console.log('');
  console.log('Key Metrics to Analyze:');
  console.log('');
  console.log('  1. Response Time Patterns');
  console.log('     - How did p95/p99 change during peaks?');
  console.log('     - Was there response time degradation over time?');
  console.log('');
  console.log('  2. Error Rate Analysis');
  console.log('     - Which phases had the highest error rates?');
  console.log('     - What types of errors occurred (timeouts, 5xx, etc)?');
  console.log('');
  console.log('  3. Recovery Behavior');
  console.log('     - How quickly did the system recover after peaks?');
  console.log('     - Did performance return to baseline?');
  console.log('');
  console.log('  4. Throughput');
  console.log('     - What was the maximum sustained throughput?');
  console.log('     - Did throughput drop during stress periods?');
  console.log('');
  console.log('  5. Resource Correlation');
  console.log('     - Compare with CPU/memory metrics from monitoring');
  console.log('     - Identify bottlenecks and saturation points');
  console.log('');
}
