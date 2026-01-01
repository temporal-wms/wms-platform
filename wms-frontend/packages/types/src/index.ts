// Order Types
export interface Order {
  id: string;
  customerId: string;
  customerName: string;
  orderNumber: string;
  status: OrderStatus;
  priority: OrderPriority;
  items: OrderItem[];
  createdAt: string;
  updatedAt: string;
  waveId?: string;
  shipmentId?: string;
}

export type OrderStatus =
  | 'PENDING'
  | 'VALIDATED'
  | 'WAVED'
  | 'PICKING'
  | 'PICKED'
  | 'PACKING'
  | 'PACKED'
  | 'SHIPPING'
  | 'SHIPPED'
  | 'COMPLETED'
  | 'FAILED'
  | 'DLQ';

export type OrderPriority = 'LOW' | 'NORMAL' | 'HIGH' | 'RUSH';

export interface OrderItem {
  id: string;
  sku: string;
  productName: string;
  quantity: number;
  pickedQuantity: number;
  packedQuantity: number;
  locationId?: string;
}

// Wave Types
export interface Wave {
  id: string;
  waveNumber: string;
  status: WaveStatus;
  orderIds: string[];
  orderCount: number;
  priority: OrderPriority;
  scheduledAt?: string;
  releasedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export type WaveStatus =
  | 'PLANNING'
  | 'READY'
  | 'RELEASED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'CANCELLED';

// Inventory Types
export interface InventoryItem {
  id: string;
  sku: string;
  productName: string;
  quantity: number;
  reservedQuantity: number;
  availableQuantity: number;
  locationId: string;
  locationCode: string;
  zone: string;
  updatedAt: string;
}

export interface Location {
  id: string;
  code: string;
  zone: string;
  aisle: string;
  rack: string;
  level: string;
  position: string;
  type: LocationType;
  capacity: number;
  currentItems: number;
}

export type LocationType = 'PICK' | 'RESERVE' | 'STAGING' | 'SHIPPING' | 'RECEIVING';

// Picking Types
export interface PickTask {
  id: string;
  orderId: string;
  waveId: string;
  status: PickTaskStatus;
  workerId?: string;
  workerName?: string;
  items: PickTaskItem[];
  route: PickRoute;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export type PickTaskStatus =
  | 'PENDING'
  | 'ASSIGNED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'PARTIAL'
  | 'FAILED';

export interface PickTaskItem {
  id: string;
  sku: string;
  productName: string;
  quantity: number;
  pickedQuantity: number;
  locationId: string;
  locationCode: string;
  sequence: number;
}

export interface PickRoute {
  id: string;
  stops: RouteStop[];
  totalDistance: number;
  estimatedTime: number;
}

export interface RouteStop {
  sequence: number;
  locationId: string;
  locationCode: string;
  zone: string;
  items: PickTaskItem[];
}

// Packing Types
export interface PackTask {
  id: string;
  orderId: string;
  pickTaskId: string;
  status: PackTaskStatus;
  workerId?: string;
  workerName?: string;
  items: PackItem[];
  packages: Package[];
  startedAt?: string;
  completedAt?: string;
}

export type PackTaskStatus =
  | 'PENDING'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'FAILED';

export interface PackItem {
  id: string;
  sku: string;
  productName: string;
  quantity: number;
  packedQuantity: number;
}

export interface Package {
  id: string;
  trackingNumber?: string;
  weight: number;
  dimensions: {
    length: number;
    width: number;
    height: number;
  };
  items: string[];
}

// Shipping Types
export interface Shipment {
  id: string;
  orderId: string;
  trackingNumber: string;
  carrier: string;
  status: ShipmentStatus;
  packages: Package[];
  shippedAt?: string;
  deliveredAt?: string;
  estimatedDelivery?: string;
}

export type ShipmentStatus =
  | 'PENDING'
  | 'LABEL_CREATED'
  | 'PICKED_UP'
  | 'IN_TRANSIT'
  | 'OUT_FOR_DELIVERY'
  | 'DELIVERED'
  | 'FAILED';

// Labor Types
export interface Worker {
  id: string;
  employeeId: string;
  name: string;
  role: WorkerRole;
  status: WorkerStatus;
  currentZone?: string;
  currentTaskId?: string;
  shiftStart?: string;
  shiftEnd?: string;
  tasksCompleted: number;
  itemsProcessed: number;
}

export type WorkerRole = 'PICKER' | 'PACKER' | 'SHIPPER' | 'RECEIVER' | 'SUPERVISOR';

export type WorkerStatus = 'AVAILABLE' | 'BUSY' | 'BREAK' | 'OFFLINE';

export interface Shift {
  id: string;
  name: string;
  startTime: string;
  endTime: string;
  workers: string[];
  zone?: string;
}

// Dashboard / Metrics Types
export interface DashboardMetrics {
  orders: {
    total: number;
    pending: number;
    inProgress: number;
    completed: number;
    failed: number;
    dlq: number;
  };
  waves: {
    active: number;
    completed: number;
    ordersInWaves: number;
  };
  picking: {
    activeTasks: number;
    completedToday: number;
    itemsPicked: number;
    averageTime: number;
  };
  packing: {
    activeTasks: number;
    completedToday: number;
    packagesCreated: number;
  };
  shipping: {
    pending: number;
    shippedToday: number;
    inTransit: number;
  };
  labor: {
    activeWorkers: number;
    totalWorkers: number;
    utilizationRate: number;
  };
}

// Event Types for WebSocket / EventBus
export type WMSEvent =
  | { type: 'ORDER_CREATED'; payload: { orderId: string; orderNumber: string } }
  | { type: 'ORDER_STATUS_CHANGED'; payload: { orderId: string; status: OrderStatus; previousStatus: OrderStatus } }
  | { type: 'WAVE_CREATED'; payload: { waveId: string; orderCount: number } }
  | { type: 'WAVE_RELEASED'; payload: { waveId: string; orderIds: string[] } }
  | { type: 'PICK_TASK_ASSIGNED'; payload: { taskId: string; workerId: string; workerName: string } }
  | { type: 'PICK_COMPLETED'; payload: { taskId: string; orderId: string } }
  | { type: 'PACK_COMPLETED'; payload: { taskId: string; orderId: string; packages: number } }
  | { type: 'SHIPMENT_CREATED'; payload: { shipmentId: string; trackingNumber: string } }
  | { type: 'METRICS_UPDATED'; payload: Partial<DashboardMetrics> };

// API Response Types
export interface ApiResponse<T> {
  data: T;
  message?: string;
  timestamp: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
