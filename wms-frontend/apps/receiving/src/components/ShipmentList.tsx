import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { receivingClient } from '@wms/api-client';
import type { ReceivingShipmentFilters } from '@wms/api-client';
import type { ReceivingShipment } from '@wms/types';
import { Table, Column, Input, Select, Button, PageLoading, EmptyState } from '@wms/ui';
import { Package, Truck, Search, Filter, Calendar } from 'lucide-react';

export function ShipmentList() {
  const [filters, setFilters] = useState<ReceivingShipmentFilters>({});
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['shipments', filters, search],
    queryFn: () => receivingClient.getShipments(filters),
  });

  if (isLoading) return <PageLoading message="Loading shipments..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading shipments: {error?.message}</div>;

  const shipments = data?.data || [];
  const total = data?.total || 0;

  const columns: Column<ReceivingShipment>[] = [
    {
      key: 'shipmentId',
      header: 'Shipment ID',
      accessor: (row: ReceivingShipment) => (
        <Link
          to={`/receiving/${row.shipmentId}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {row.shipmentId}
        </Link>
      ),
    },
    {
      key: 'purchaseOrderId',
      header: 'PO Number',
      accessor: (row: ReceivingShipment) => row.purchaseOrderId,
    },
    {
      key: 'supplier',
      header: 'Supplier',
      accessor: (row: ReceivingShipment) => (
        <div>
          <div className="font-medium">{row.supplier.name}</div>
          <div className="text-sm text-gray-500">{row.supplier.code}</div>
        </div>
      ),
    },
    {
      key: 'carrier',
      header: 'Carrier',
      accessor: (row: ReceivingShipment) => row.asn?.shippingCarrier || '-',
    },
    {
      key: 'tracking',
      header: 'Tracking',
      accessor: (row: ReceivingShipment) => (
        <span className="font-mono text-sm">{row.asn?.trackingNumber || '-'}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (row: ReceivingShipment) => {
        const statusColors: Record<string, string> = {
          expected: 'bg-blue-100 text-blue-800',
          arrived: 'bg-yellow-100 text-yellow-800',
          receiving: 'bg-purple-100 text-purple-800',
          inspection: 'bg-orange-100 text-orange-800',
          completed: 'bg-green-100 text-green-800',
          cancelled: 'bg-gray-100 text-gray-800',
        };
        return (
          <span className={`px-3 py-1 rounded-full text-sm font-medium ${statusColors[row.status] || ''}`}>
            {row.status}
          </span>
        );
      },
    },
    {
      key: 'eta',
      header: 'ETA',
      accessor: (row: ReceivingShipment) => row.asn?.estimatedArrival ? new Date(row.asn.estimatedArrival).toLocaleDateString() : '-',
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (row: ReceivingShipment) => (
        <div className="flex gap-2">
          {row.status === 'expected' && (
            <Button size="sm" onClick={() => console.log('Mark arrived:', row.shipmentId)}>
              <Truck className="h-4 w-4" />
            </Button>
          )}
          <Link to={`/receiving/${row.shipmentId}`}>
            <Button variant="outline" size="sm">View</Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setFilters({ ...filters, search: value || undefined });
  };

  const handleFilterChange = (key: keyof ReceivingShipmentFilters, value: string | undefined) => {
    setFilters({ ...filters, [key]: value });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Inbound Shipments</h1>
          <p className="text-gray-500">{total} shipments total</p>
        </div>
        <Link to="/receiving/new">
          <Button>
            <Package className="h-4 w-4 mr-2" />
            Create Shipment
          </Button>
        </Link>
      </div>

      <div className="flex gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search shipments..."
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select
          value={filters.status}
          onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
          className="w-48"
          options={[
            { value: '', label: 'All Status' },
            { value: 'expected', label: 'Expected' },
            { value: 'arrived', label: 'Arrived' },
            { value: 'receiving', label: 'Receiving' },
            { value: 'completed', label: 'Completed' },
          ]}
        />
        <Input
          type="date"
          value={filters.fromDate}
          onChange={(e) => handleFilterChange('fromDate', e.target.value || undefined)}
          className="w-40"
        />
        <Link to="/receiving/expected">
          <Button variant="outline">
            <Calendar className="h-4 w-4 mr-2" />
            Expected Arrivals
          </Button>
        </Link>
      </div>

      {shipments.length === 0 ? (
        <EmptyState
          icon={<Package className="h-12 w-12 text-gray-400" />}
          title="No shipments found"
          description={search ? `No shipments match "${search}"` : 'There are no shipments yet. Create your first shipment to get started.'}
        />
      ) : (
        <Table
          columns={columns}
          data={shipments}
          keyExtractor={(item) => item.shipmentId}
        />
      )}
    </div>
  );
}
