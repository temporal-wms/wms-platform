import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { consolidationClient } from '@wms/api-client';
import type { ConsolidationFilters } from '@wms/api-client';
import type { Consolidation } from '@wms/types';
import { Table, Column, Input, Select, Button, PageLoading, EmptyState, Badge, Card, CardHeader, CardContent, MetricCard, MetricGrid } from '@wms/ui';
import { Package, Layers, CheckCircle, Clock, Truck, Search, Filter, ChevronRight, User, Activity, ChevronDown } from 'lucide-react';

export function ConsolidationList() {
  const [filters, setFilters] = useState<ConsolidationFilters>({});
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['consolidations', filters, search],
    queryFn: () => consolidationClient.getConsolidations(filters),
  });

  if (isLoading) return <PageLoading message="Loading consolidations..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading consolidations: {error?.message}</div>;

  const consolidations = data?.data || [];
  const total = data?.total || 0;

  const columns: Column<Consolidation>[] = [
    {
      key: 'consolidationId',
      header: 'Consolidation ID',
      accessor: (row: Consolidation) => (
        <Link
          to={`/consolidation/${row.consolidationId}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {row.consolidationId}
        </Link>
      ),
    },
    {
      key: 'orderId',
      header: 'Order ID',
      accessor: (row: Consolidation) => (
        <Link
          to={`/consolidation/order/${row.orderId}`}
          className="font-medium"
        >
          {row.orderId}
        </Link>
      ),
    },
    {
      key: 'waveId',
      header: 'Wave ID',
      accessor: (row: Consolidation) => row.waveId || '-',
    },
    {
      key: 'stationId',
      header: 'Station',
      accessor: (row: Consolidation) => row.stationId,
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (row: Consolidation) => {
        const statusColors: Record<string, string> = {
          in_progress: 'bg-purple-100 text-purple-800',
          completed: 'bg-green-100 text-green-800',
          cancelled: 'bg-gray-100 text-gray-800',
        };
        return (
          <Badge className={statusColors[row.status] || ''}>
            {row.status.replace('_', ' ')}
          </Badge>
        );
      },
    },
    {
      key: 'itemsTotal',
      header: 'Total Items',
      accessor: (row: Consolidation) => row.totalExpectedItems,
      align: 'center',
    },
    {
      key: 'itemsConsolidated',
      header: 'Consolidated',
      accessor: (row: Consolidation) => row.consolidatedItems,
      },
    {
      key: 'itemsPending',
      header: 'Pending',
      accessor: (row: Consolidation) => row.totalExpectedItems - row.consolidatedItems,
    },
    {
      key: 'duration',
      header: 'Duration',
      accessor: (row: Consolidation) => row.duration || '-',
    },
    {
      key: 'createdAt',
      header: 'Created At',
      accessor: (row: Consolidation) => row.createdAt ? new Date(row.createdAt).toLocaleString() : '-',
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (row: Consolidation) => (
        <div className="flex gap-2">
          {row.status === 'in_progress' && (
            <Button size="sm" onClick={() => console.log('View consolidation:', row.consolidationId)}>
              View
            </Button>
          )}
          <Link to={`/consolidation/${row.consolidationId}`}>
            <Button variant="outline" size="sm">View Details</Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleSearchChange = (value: string) => {
    setSearch(value);
    // Note: ConsolidationFilters doesn't have a search field, so we just update local state
  };

  const handleFilterChange = (key: keyof ConsolidationFilters, value: string | undefined) => {
    setFilters({ ...filters, [key]: value });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Consolidations</h1>
          <p className="text-gray-500">{total} consolidations total</p>
        </div>
        <Link to="/consolidation/new">
          <Button>
            <Package className="h-4 w-4 mr-2" />
            Create Consolidation
          </Button>
        </Link>
      </div>

      <div className="flex gap-4 flex-wrap">
        <div className="relative flex-1 min-w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search consolidations..."
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select
          value={filters.status}
          onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
          className="w-40"
          options={[
            { value: '', label: 'All Status' },
            { value: 'in_progress', label: 'In Progress' },
            { value: 'completed', label: 'Completed' },
            { value: 'cancelled', label: 'Cancelled' },
          ]}
        />
        <Select
          value={filters.stationId}
          onChange={(e) => handleFilterChange('stationId', e.target.value || undefined)}
          className="w-40"
          options={[
            { value: '', label: 'All Stations' },
            { value: 'STATION-01', label: 'STATION-01' },
            { value: 'STATION-02', label: 'STATION-02' },
          ]}
        />
      </div>

      {consolidations.length === 0 ? (
        <EmptyState
          icon={<Layers className="h-12 w-12 text-gray-400" />}
          title="No consolidations found"
          description="There are no consolidations yet. Create your first consolidation to get started."
        />
      ) : (
        <Table
          columns={columns}
          data={consolidations}
          keyExtractor={(item) => item.consolidationId}
        />
      )}
    </div>
  );
}
