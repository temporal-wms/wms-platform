import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, Link } from 'react-router-dom';
import { Plus, Play, RefreshCw, Layers } from 'lucide-react';
import {
  Card,
  CardHeader,
  Table,
  Column,
  Pagination,
  Button,
  StatusBadge,
  Badge,
  Select,
  EmptyState,
  PageLoading,
  MetricCard,
  MetricGrid,
} from '@wms/ui';
import { waveClient, WaveFilters } from '@wms/api-client';
import { formatDateTime, formatRelativeTime } from '@wms/utils';
import type { Wave } from '@wms/types';
import { AlertTriangle } from 'lucide-react';

const statusOptions = [
  { value: '', label: 'All Statuses' },
  { value: 'PLANNING', label: 'Planning' },
  { value: 'READY', label: 'Ready' },
  { value: 'RELEASED', label: 'Released' },
  { value: 'IN_PROGRESS', label: 'In Progress' },
  { value: 'COMPLETED', label: 'Completed' },
];

export function WaveList() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<WaveFilters>({
    page: 1,
    pageSize: 20,
  });

  const { data: wavesResponse, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['waves', filters],
    queryFn: () => waveClient.getWaves(filters),
    retry: 1,
  });

  const { data: stats } = useQuery({
    queryKey: ['wave-stats'],
    queryFn: waveClient.getWaveStats,
    retry: 1,
  });

  const waves = wavesResponse?.data || [];
  const total = wavesResponse?.total || 0;
  const totalPages = wavesResponse?.totalPages || 1;

  const columns: Column<Wave>[] = [
    {
      key: 'waveNumber',
      header: 'Wave #',
      sortable: true,
      accessor: (wave) => (
        <Link
          to={`/waves/${wave.id}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {wave.waveNumber}
        </Link>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      accessor: (wave) => <StatusBadge status={wave.status} />,
    },
    {
      key: 'orders',
      header: 'Orders',
      align: 'center',
      accessor: (wave) => (
        <Badge variant="neutral">{wave.orderCount} orders</Badge>
      ),
    },
    {
      key: 'priority',
      header: 'Priority',
      accessor: (wave) => {
        const variants: Record<string, 'error' | 'warning' | 'success' | 'neutral'> = {
          RUSH: 'error',
          HIGH: 'warning',
          NORMAL: 'neutral',
          LOW: 'success',
        };
        return <Badge variant={variants[wave.priority] || 'neutral'}>{wave.priority}</Badge>;
      },
    },
    {
      key: 'released',
      header: 'Released',
      accessor: (wave) =>
        wave.releasedAt ? (
          <span className="text-gray-500">{formatRelativeTime(wave.releasedAt)}</span>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: 'created',
      header: 'Created',
      sortable: true,
      accessor: (wave) => (
        <span className="text-gray-500" title={formatDateTime(wave.createdAt)}>
          {formatRelativeTime(wave.createdAt)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (wave) =>
        wave.status === 'READY' ? (
          <Button
            variant="secondary"
            size="sm"
            icon={<Play className="h-4 w-4" />}
            onClick={(e) => {
              e.stopPropagation();
              // Release wave
            }}
          >
            Release
          </Button>
        ) : null,
    },
  ];

  if (isLoading) {
    return <PageLoading message="Loading waves..." />;
  }

  if (isError) {
    return (
      <Card padding="lg">
        <div className="text-center py-8">
          <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Failed to load waves</h2>
          <p className="text-gray-500 mb-4">
            {error instanceof Error ? error.message : 'Unable to connect to the waving service'}
          </p>
          <Button onClick={() => refetch()} icon={<RefreshCw className="h-4 w-4" />}>
            Retry
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Waves</h1>
          <p className="text-gray-500">Plan and manage order waves for picking</p>
        </div>
        <Link to="/waves/new">
          <Button icon={<Plus className="h-4 w-4" />}>Create Wave</Button>
        </Link>
      </div>

      {/* Stats */}
      <MetricGrid columns={4}>
        <MetricCard
          title="Active Waves"
          value={stats?.active || 0}
          icon={<Layers className="h-6 w-6" />}
          variant="default"
        />
        <MetricCard
          title="Orders in Waves"
          value={stats?.ordersInWaves || 0}
          subtitle="Across all active waves"
        />
        <MetricCard
          title="Completed Today"
          value={stats?.completed || 0}
          variant="success"
        />
        <MetricCard
          title="Total Waves"
          value={stats?.total || 0}
          subtitle="All time"
        />
      </MetricGrid>

      {/* Filters */}
      <Card padding="md">
        <div className="flex flex-wrap items-center gap-4">
          <Select
            options={statusOptions}
            value={filters.status || ''}
            onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value || undefined, page: 1 }))}
          />
          <Button variant="ghost" onClick={() => refetch()} icon={<RefreshCw className="h-4 w-4" />}>
            Refresh
          </Button>
        </div>
      </Card>

      {/* Waves Table */}
      <Card padding="none">
        {waves.length === 0 ? (
          <EmptyState
            title="No waves found"
            description="Create a wave to group orders for picking"
            action={{
              label: 'Create Wave',
              onClick: () => navigate('/waves/new'),
            }}
          />
        ) : (
          <>
            <Table
              columns={columns}
              data={waves}
              keyExtractor={(wave) => wave.id}
              onRowClick={(wave) => navigate(`/waves/${wave.id}`)}
            />
            <Pagination
              currentPage={filters.page || 1}
              totalPages={totalPages}
              pageSize={filters.pageSize || 20}
              totalItems={total}
              onPageChange={(page) => setFilters((prev) => ({ ...prev, page }))}
            />
          </>
        )}
      </Card>
    </div>
  );
}
