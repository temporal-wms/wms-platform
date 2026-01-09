export interface Order {
  orderId: string;
  customerId: string;
  status: string;
  priority: string;
  totalItems: number;
  totalWeight: number;
  promisedDeliveryAt: string;
  shipToCity: string;
  shipToState: string;
}

export interface PagedOrdersResult {
  data: Order[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface CreateWaveRequest {
  orderIds: string[];
  waveType: string;
  fulfillmentMode: string;
  zone: string;
}

export interface Wave {
  waveId: string;
  waveType: string;
  status: string;
  orderCount: number;
  createdAt: string;
}

export interface CreateWaveResponse {
  wave: Wave;
  failedOrders: string[];
}
