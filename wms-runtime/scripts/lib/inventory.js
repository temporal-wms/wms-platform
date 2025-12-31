// Inventory API helpers for K6 load testing
import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS } from './config.js';

const baseUrl = BASE_URLS.inventory;

export function createInventoryItem(sku, productName, reorderPoint, reorderQuantity) {
  const payload = JSON.stringify({
    sku,
    productName,
    reorderPoint,
    reorderQuantity,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.inventory.create}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'inventory item created': (r) => r.status === 201 || r.status === 200 || r.status === 409,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function receiveStock(sku, locationId, zone, quantity, referenceId, createdBy) {
  const payload = JSON.stringify({
    locationId,
    zone,
    quantity,
    referenceId: referenceId || `RECEIVE-${Date.now()}`,
    createdBy: createdBy || 'k6-load-test',
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.inventory.receive(sku)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'stock received': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function reserveStock(sku, orderId, locationId, quantity) {
  const payload = JSON.stringify({
    orderId,
    locationId,
    quantity,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.inventory.reserve(sku)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'stock reserved': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function pickStock(sku, orderId, locationId, quantity, createdBy) {
  const payload = JSON.stringify({
    orderId,
    locationId,
    quantity,
    createdBy: createdBy || 'k6-load-test',
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.inventory.pick(sku)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'stock picked': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function releaseReservation(sku, orderId) {
  const payload = JSON.stringify({
    orderId,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.inventory.release(sku)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'reservation released': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function getInventoryItem(sku) {
  const response = http.get(
    `${baseUrl}${ENDPOINTS.inventory.get(sku)}`,
    HTTP_PARAMS
  );

  const success = check(response, {
    'inventory item retrieved': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function getLowStockItems() {
  const response = http.get(
    `${baseUrl}${ENDPOINTS.inventory.lowStock}`,
    HTTP_PARAMS
  );

  const success = check(response, {
    'low stock items retrieved': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function checkHealth() {
  const response = http.get(`${baseUrl}/health`);

  return check(response, {
    'inventory service healthy': (r) => r.status === 200,
  });
}
