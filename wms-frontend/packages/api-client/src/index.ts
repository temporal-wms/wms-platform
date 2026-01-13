export { httpClient, createServiceClient } from './httpClient';
export { orderClient } from './clients/orderClient';
export { waveClient } from './clients/waveClient';
export { inventoryClient } from './clients/inventoryClient';
export { pickingClient } from './clients/pickingClient';
export { packingClient } from './clients/packingClient';
export { shippingClient } from './clients/shippingClient';
export { laborClient } from './clients/laborClient';
export { dashboardClient } from './clients/dashboardClient';
export { receivingClient } from './clients/receivingClient';
export { stowClient } from './clients/stowClient';
export { routingClient } from './clients/routingClient';
export { wallingClient } from './clients/wallingClient';
export { consolidationClient } from './clients/consolidationClient';
export { sortationClient } from './clients/sortationClient';
export { facilityClient } from './clients/facilityClient';

// Re-export types
export type { CreateOrderRequest, OrderFilters } from './clients/orderClient';
export type { CreateWaveRequest, WaveFilters } from './clients/waveClient';
export type { InventoryFilters, AdjustInventoryRequest } from './clients/inventoryClient';
export type { PickTaskFilters, ConfirmPickRequest } from './clients/pickingClient';
export type { PackTaskFilters, CreatePackageRequest } from './clients/packingClient';
export type { ShipmentFilters, CreateShipmentRequest } from './clients/shippingClient';
export type { WorkerFilters, CreateWorkerRequest, CreateShiftRequest } from './clients/laborClient';
export type { ShipmentFilters as ReceivingShipmentFilters, CreateShipmentRequest as ReceivingCreateShipmentRequest, ReceiveItemRequest } from './clients/receivingClient';
export type { StowTaskFilters, CreatePutawayRequest } from './clients/stowClient';
export type { RouteFilters, CreateRouteRequest, RouteItem, SkipReason } from './clients/routingClient';

export type { WallingTaskFilters, CreateWallingTaskRequest } from './clients/wallingClient';
export type { ConsolidationFilters, CreateConsolidationRequest, ConsolidateItemRequest } from './clients/consolidationClient';
export type { BatchFilters, CreateBatchRequest, SortPackageRequest, DispatchBatchRequest } from './clients/sortationClient';
export type { StationFilters, CreateStationRequest, UpdateStationRequest, FindCapableStationsRequest, SetCapabilitiesRequest, SetStatusRequest } from './clients/facilityClient';
