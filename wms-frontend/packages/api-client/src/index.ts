export { httpClient, createServiceClient } from './httpClient';
export { orderClient } from './clients/orderClient';
export { waveClient } from './clients/waveClient';
export { inventoryClient } from './clients/inventoryClient';
export { pickingClient } from './clients/pickingClient';
export { packingClient } from './clients/packingClient';
export { shippingClient } from './clients/shippingClient';
export { laborClient } from './clients/laborClient';
export { dashboardClient } from './clients/dashboardClient';

// Re-export types
export type { CreateOrderRequest, OrderFilters } from './clients/orderClient';
export type { CreateWaveRequest, WaveFilters } from './clients/waveClient';
export type { InventoryFilters, AdjustInventoryRequest } from './clients/inventoryClient';
export type { PickTaskFilters, ConfirmPickRequest } from './clients/pickingClient';
export type { PackTaskFilters, CreatePackageRequest } from './clients/packingClient';
export type { ShipmentFilters, CreateShipmentRequest } from './clients/shippingClient';
export type { WorkerFilters, CreateWorkerRequest, CreateShiftRequest } from './clients/laborClient';
