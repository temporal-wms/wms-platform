import { httpClient, createServiceClient } from '../httpClient';
import type {
  Station,
  CreateStationRequest,
  UpdateStationRequest,
  FindCapableStationsRequest,
  SetCapabilitiesRequest,
  SetStatusRequest,
  StationCapability,
} from '@wms/types';

const client = createServiceClient('facility');

export interface StationFilters {
  zone?: string;
  type?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export const facilityClient = {
  getStations: async (filters?: StationFilters): Promise<Station[]> => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    return client.get<Station[]>(`api/v1/stations?${params.toString()}`);
  },

  getStation: async (stationId: string): Promise<Station> => {
    return client.get<Station>(`api/v1/stations/${stationId}`);
  },

  createStation: async (request: CreateStationRequest): Promise<Station> => {
    return client.post<Station>('api/v1/stations', request);
  },

  updateStation: async (
    stationId: string,
    updates: UpdateStationRequest
  ): Promise<Station> => {
    return client.put<Station>(`api/v1/stations/${stationId}`, updates);
  },

  deleteStation: async (stationId: string): Promise<void> => {
    return client.delete<void>(`api/v1/stations/${stationId}`);
  },

  setCapabilities: async (
    stationId: string,
    request: SetCapabilitiesRequest
  ): Promise<Station> => {
    return client.put<Station>(`api/v1/stations/${stationId}/capabilities`, request);
  },

  addCapability: async (
    stationId: string,
    capability: StationCapability
  ): Promise<Station> => {
    return client.post<Station>(`api/v1/stations/${stationId}/capabilities/${capability}`);
  },

  removeCapability: async (
    stationId: string,
    capability: StationCapability
  ): Promise<Station> => {
    return client.delete<Station>(`api/v1/stations/${stationId}/capabilities/${capability}`);
  },

  setStatus: async (
    stationId: string,
    request: SetStatusRequest
  ): Promise<Station> => {
    return client.put<Station>(`api/v1/stations/${stationId}/status`, request);
  },

  findCapableStations: async (
    request: FindCapableStationsRequest
  ): Promise<Station[]> => {
    return client.post<Station[]>('api/v1/stations/find-capable', request);
  },

  getStationsByZone: async (zone: string): Promise<Station[]> => {
    return client.get<Station[]>(`api/v1/stations/zone/${zone}`);
  },

  getStationsByType: async (type: string): Promise<Station[]> => {
    return client.get<Station[]>(`api/v1/stations/type/${type}`);
  },

  getStationsByStatus: async (status: string): Promise<Station[]> => {
    return client.get<Station[]>(`api/v1/stations/status/${status}`);
  },
};

export type {
  CreateStationRequest,
  UpdateStationRequest,
  FindCapableStationsRequest,
  SetCapabilitiesRequest,
  SetStatusRequest,
};
