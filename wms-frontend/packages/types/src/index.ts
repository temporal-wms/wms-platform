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
  type: InventoryLocationType;
  capacity: number;
  currentItems: number;
}

export type InventoryLocationType = 'PICK' | 'RESERVE' | 'STAGING' | 'SHIPPING' | 'RECEIVING';

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
  status: ShippingShipmentStatus;
  packages: Package[];
  shippedAt?: string;
  deliveredAt?: string;
  estimatedDelivery?: string;
}

export type ShippingShipmentStatus =
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

// Receiving Types
export interface ReceivingShipment {
  shipmentId: string;
  purchaseOrderId: string;
  asn: ASN;
  supplier: Supplier;
  expectedItems: ExpectedItemResponse[];
  receiptRecords: ReceiptRecord[];
  discrepancies: Discrepancy[];
  status: ReceivingShipmentStatus;
  receivingDockId?: string;
  assignedWorkerId?: string;
  arrivedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ASN {
  asnId: string;
  shippingCarrier: string;
  trackingNumber: string;
  estimatedArrival: string;
}

export interface Supplier {
  supplierId: string;
  name: string;
  code: string;
  contactEmail?: string;
}

export interface ExpectedItemResponse {
  sku: string;
  productName: string;
  expectedQuantity: number;
  receivedQuantity: number;
  damagedQuantity: number;
  unitCost?: number;
}

export interface ReceiptRecord {
  receiptId: string;
  sku: string;
  receivedQty: number;
  condition: ItemCondition;
  toteId?: string;
  locationId?: string;
  receivedBy?: string;
  receivedAt: string;
}

export interface Discrepancy {
  sku: string;
  type: DiscrepancyType;
  expectedQty: number;
  actualQty: number;
  description: string;
  detectedAt: string;
}

export type ReceivingShipmentStatus = 'expected' | 'arrived' | 'receiving' | 'inspection' | 'completed' | 'cancelled';
export type ItemCondition = 'good' | 'damaged' | 'rejected';
export type DiscrepancyType = 'shortage' | 'overage' | 'damage' | 'wrong_item';

export interface CreateShipmentRequest {
  purchaseOrderId: string;
  asn: ASN;
  supplier: Supplier;
  expectedItems: ExpectedItem[];
}

export interface ExpectedItem {
  sku: string;
  productName: string;
  expectedQuantity: number;
  unitCost?: number;
  weight?: number;
  isHazmat?: boolean;
  requiresColdChain?: boolean;
}

export interface ReceiveItemRequest {
  sku: string;
  quantity: number;
  condition: ItemCondition;
  toteId?: string;
  locationId?: string;
  workerId: string;
  notes?: string;
}

// Stow Types
export interface PutawayTask {
  taskId: string;
  shipmentId: string;
  sku: string;
  productName: string;
  quantity: number;
  sourceToteId?: string;
  sourceLocationId?: string;
  targetLocationId?: string;
  targetLocation?: StorageLocation;
  strategy: StorageStrategy;
  constraints: ItemConstraints;
  status: PutawayStatus;
  assignedWorkerId?: string;
  priority: number;
  stowedQuantity: number;
  failureReason?: string;
  createdAt: string;
  updatedAt: string;
}

export type StorageStrategy = 'chaotic' | 'directed' | 'velocity' | 'zone_based';
export type PutawayStatus = 'pending' | 'assigned' | 'in_progress' | 'completed' | 'cancelled' | 'failed';
export type StorageLocationType = 'pick_face' | 'reserve' | 'floor_stack' | 'pallet_rack' | 'cold_storage' | 'hazmat_zone';

export interface StorageLocation {
  locationId: string;
  locationCode: string;
  zone: string;
  aisle: string;
  bay: string;
  level: string;
  type: StorageLocationType;
  capacity: number;
  currentItems: number;
  available: boolean;
}

export interface ItemConstraints {
  hazmat?: boolean;
  coldChain?: boolean;
  oversized?: boolean;
  fragile?: boolean;
  highValue?: boolean;
}

export interface CreatePutawayRequest {
  shipmentId: string;
  sku: string;
  productName: string;
  quantity: number;
  sourceToteId?: string;
  sourceLocationId?: string;
  strategy?: StorageStrategy;
  constraints?: ItemConstraints;
  priority?: number;
}

// Routing Types
export interface Route {
  routeId: string;
  waveId: string;
  orderId?: string;
  pickerId: string;
  status: RouteStatus;
  strategy: string;
  totalDistance: number;
  estimatedTimeMinutes: number;
  actualTimeMinutes?: number;
  stopsTotal: number;
  stopsCompleted: number;
  stopsSkipped: number;
  stops: RouteStop[];
  startedAt?: string;
  completedAt?: string;
  pausedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export type RouteStatus = 'calculated' | 'in_progress' | 'paused' | 'completed' | 'cancelled';

export interface RouteStop {
  sequence: number;
  locationId: string;
  zone: string;
  aisle: string;
  bay: string;
  level: string;
  sku: string;
  quantity: number;
  actualQuantity?: number;
  status: 'pending' | 'completed' | 'skipped';
  skipReason?: string;
  notes?: string;
  completedAt?: string;
}

export interface RouteAnalysis {
  routeId: string;
  efficiency: number;
  estimatedVsActualTime: {
    estimatedMinutes: number;
    actualMinutes: number;
    variance: number;
  };
  distanceMetrics: {
    totalDistance: number;
    averagePerStop: number;
  };
  completionRate: number;
  skippedStops: number;
  recommendations: string[];
}

export interface CreateRouteRequest {
  waveId: string;
  orderId?: string;
  pickerId: string;
  strategy: string;
  items: RouteItem[];
}

export interface RouteItem {
  sku: string;
  locationId: string;
  quantity: number;
  priority?: number;
}

export type SkipReason = 'out_of_stock' | 'location_blocked' | 'item_damaged' | 'other';

// Walling Types
export interface WallingTask {
  taskId: string;
  orderId: string;
  waveId: string;
  routeId?: string;
  wallinerId?: string;
  status: WallingTaskStatus;
  sourceTotes: SourceTote[];
  destinationBin: string;
  putWallId: string;
  itemsToSort: ItemToSort[];
  sortedItems: SortedItem[];
  station?: string;
  priority: number;
  createdAt: string;
  completedAt?: string;
}

export type WallingTaskStatus = 'pending' | 'assigned' | 'in_progress' | 'completed' | 'cancelled';

export interface SourceTote {
  toteId: string;
  pickTaskId: string;
  itemCount: number;
}

export interface ItemToSort {
  sku: string;
  productName: string;
  quantity: number;
  fromToteId: string;
  sortedQty: number;
}

export interface SortedItem {
  sku: string;
  quantity: number;
  fromToteId: string;
  toBinId: string;
  sortedAt: string;
  verified: boolean;
}

export interface CreateWallingTaskRequest {
  orderId: string;
  waveId: string;
  routeId?: string;
  putWallId: string;
  destinationBin: string;
  sourceTotes: SourceTote[];
  itemsToSort: ItemToSort[];
}

// Consolidation Types
export interface Consolidation {
  consolidationId: string;
  orderId: string;
  waveId?: string;
  stationId: string;
  status: ConsolidationStatus;
  expectedItems: ConsolidationItem[];
  totalExpectedItems: number;
  totalExpectedQuantity: number;
  consolidatedItems: number;
  consolidatedQuantity: number;
  consolidationRecords: ConsolidationRecord[];
  duration?: string;
  createdAt: string;
  completedAt?: string;
  updatedAt: string;
}

export type ConsolidationStatus = 'in_progress' | 'completed' | 'cancelled';

export interface ConsolidationItem {
  sku: string;
  productName: string;
  quantity: number;
  consolidated: boolean;
  consolidatedQty: number;
}

export interface ConsolidationRecord {
  sku: string;
  quantity: number;
  toteId?: string;
  routeId?: string;
  workerId?: string;
  consolidatedAt: string;
}

export interface CreateConsolidationRequest {
  orderId: string;
  waveId?: string;
  stationId: string;
  expectedItems: ConsolidationItem[];
}

export interface ConsolidateItemRequest {
  sku: string;
  quantity: number;
  toteId?: string;
  routeId?: string;
  workerId?: string;
}

// Sortation Types
export interface SortationBatch {
  batchId: string;
  sortationCenter: string;
  destinationGroup: string;
  carrierId: string;
  packages: SortedPackage[];
  assignedChute?: string;
  status: SortationStatus;
  totalPackages: number;
  sortedCount: number;
  totalWeight: number;
  trailerId?: string;
  dispatchDock?: string;
  scheduledDispatch?: string;
  createdAt: string;
  updatedAt: string;
  dispatchedAt?: string;
  sortingProgress: number;
}

export type SortationStatus = 'receiving' | 'sorting' | 'ready' | 'dispatching' | 'dispatched' | 'cancelled';

export interface SortedPackage {
  packageId: string;
  orderId: string;
  trackingNumber?: string;
  destination: string;
  carrierId: string;
  weight?: number;
  assignedChute?: string;
  sortedAt?: string;
  sortedBy?: string;
  isSorted: boolean;
}

export interface CreateBatchRequest {
  sortationCenter: string;
  destinationGroup: string;
  carrierId: string;
}

export interface SortPackageRequest {
  packageId: string;
  orderId?: string;
  trackingNumber?: string;
  destination: string;
  carrierId?: string;
  weight?: number;
}

export interface DispatchBatchRequest {
  trailerId: string;
  dispatchDock: string;
}

// Facility Types
export interface Station {
  stationId: string;
  name: string;
  zone: string;
  stationType: StationType;
  status: StationStatus;
  capabilities: string[];
  maxConcurrentTasks: number;
  currentTasks: number;
  availableCapacity: number;
  assignedWorkerId?: string;
  equipment: StationEquipment[];
  createdAt: string;
  updatedAt: string;
}

export type StationType = 'packing' | 'consolidation' | 'shipping' | 'receiving';
export type StationStatus = 'active' | 'inactive' | 'maintenance';

export type StationCapability =
  | 'single_item' | 'multi_item' | 'gift_wrap' | 'hazmat' | 'oversized'
  | 'fragile' | 'cold_chain' | 'high_value';

export interface StationEquipment {
  equipmentId: string;
  equipmentType: EquipmentType;
  status: string;
}

export type EquipmentType = 'scale' | 'printer' | 'cold_storage' | 'hazmat_cabinet';

export interface CreateStationRequest {
  stationId: string;
  name: string;
  zone: string;
  stationType: StationType;
  capabilities?: string[];
  maxConcurrentTasks?: number;
}

export interface UpdateStationRequest {
  name?: string;
  zone?: string;
  maxConcurrentTasks?: number;
}

export interface FindCapableStationsRequest {
  requirements: StationCapability[];
  stationType?: StationType;
  zone?: string;
}

export interface SetCapabilitiesRequest {
  capabilities: string[];
}

export interface SetStatusRequest {
  status: StationStatus;
}

export interface Chute {
  chuteId: string;
  chuteNumber: number;
  destination: string;
  carrierId: string;
  capacity: number;
  currentCount: number;
  status: 'active' | 'full' | 'maintenance';
}
