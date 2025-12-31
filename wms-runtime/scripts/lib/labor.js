// Labor API helpers for K6 load testing
import http from 'k6/http';
import { check } from 'k6';
import { BASE_URLS, ENDPOINTS, HTTP_PARAMS } from './config.js';

const baseUrl = BASE_URLS.labor;

export function createWorker(workerId, employeeId, name) {
  const payload = JSON.stringify({
    workerId,
    employeeId,
    name,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.createWorker}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'worker created': (r) => r.status === 201 || r.status === 200 || r.status === 409,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function addWorkerSkill(workerId, taskType, level, certified) {
  const payload = JSON.stringify({
    taskType,
    level,
    certified,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.addSkill(workerId)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'skill added': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function startShift(workerId, shiftId, shiftType, zone) {
  const payload = JSON.stringify({
    shiftId,
    shiftType,
    zone,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.startShift(workerId)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'shift started': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function endShift(workerId) {
  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.endShift(workerId)}`,
    '{}',
    HTTP_PARAMS
  );

  const success = check(response, {
    'shift ended': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function startBreak(workerId, breakType) {
  const payload = JSON.stringify({
    breakType: breakType || 'break',
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.startBreak(workerId)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'break started': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function endBreak(workerId) {
  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.endBreak(workerId)}`,
    '{}',
    HTTP_PARAMS
  );

  const success = check(response, {
    'break ended': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function assignTask(workerId, taskId, taskType, priority) {
  const payload = JSON.stringify({
    taskId,
    taskType,
    priority: priority || 1,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.assignTask(workerId)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'task assigned': (r) => r.status === 200 || r.status === 201,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function completeTask(workerId, itemsProcessed) {
  const payload = JSON.stringify({
    itemsProcessed: itemsProcessed || 1,
  });

  const response = http.post(
    `${baseUrl}${ENDPOINTS.labor.completeTask(workerId)}`,
    payload,
    HTTP_PARAMS
  );

  const success = check(response, {
    'task completed': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function getWorker(workerId) {
  const response = http.get(
    `${baseUrl}${ENDPOINTS.labor.getWorker(workerId)}`,
    HTTP_PARAMS
  );

  const success = check(response, {
    'worker retrieved': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function getAvailableWorkers(zone) {
  let url = `${baseUrl}${ENDPOINTS.labor.availableWorkers}`;
  if (zone) {
    url += `?zone=${zone}`;
  }

  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'available workers retrieved': (r) => r.status === 200,
  });

  return {
    success,
    status: response.status,
    body: response.json(),
  };
}

export function listWorkers(limit, offset) {
  let url = `${baseUrl}${ENDPOINTS.labor.listWorkers}`;
  const params = [];
  if (limit) params.push(`limit=${limit}`);
  if (offset) params.push(`offset=${offset}`);
  if (params.length > 0) url += `?${params.join('&')}`;

  const response = http.get(url, HTTP_PARAMS);

  const success = check(response, {
    'workers listed': (r) => r.status === 200,
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
    'labor service healthy': (r) => r.status === 200,
  });
}
