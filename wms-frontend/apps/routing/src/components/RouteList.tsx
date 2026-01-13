import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { RouteFilters } from '@wms/api-client';
import type { Route } from '@wms/types';
import { Table, Column, Input, Select, Button, PageLoading, EmptyState, Badge } from '@wms/ui';
import { Route as RouteIcon, Search, Filter, User, MapPin } from 'lucide-react';

export function RouteList() {
  const [filters, setFilters] = useState<RouteFilters>({});
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['routes', filters],
    queryFn: () => routingClient.getRoutes(filters),
  });

  if (isLoading) return <PageLoading message="Loading routes..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading routes: {error?.message}</div>;

  const routes = data?.data || [];
  const total = data?.total || 0;

  const columns: Column<Route>[] = [
    {
      key: 'routeId',
      header: 'Route ID',
      accessor: (row: Route) => (
        <Link 
          to={`/routing/${row.routeId}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {row.routeId}
        </Link>
      ),
    },
    {
      key: 'pickerId',
      header: 'Picker',
      accessor: (row: Route) => row.pickerId,
    },
    {
      key: 'strategy',
      header: 'Strategy',
      accessor: (row: Route) => {
        const strategyColors: Record<string, string> = {
          shortest_path: 'bg-blue-100 text-blue-800',
          zone_based: 'bg-green-100 text-green-800',
          priority_first: 'bg-orange-100 text-orange-800',
          batch_pick: 'bg-purple-100 text-purple-800',
        };
        return (
          <Badge className={strategyColors[row.strategy] || ''}>
            {row.strategy.replace('_', ' ')}
          </Badge>
        );
      },
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (row: Route) => {
        const statusColors: Record<string, string> = {
          calculated: 'bg-blue-100 text-blue-800',
          in_progress: 'bg-purple-100 text-purple-800',
          paused: 'bg-yellow-100 text-yellow-800',
          completed: 'bg-green-100 text-green-800',
          cancelled: 'bg-gray-100 text-gray-800',
        };
        return (
          <span className={`px-3 py-1 rounded-full text-sm font-medium ${statusColors[row.status] || ''}`}>
            {row.status.replace('_', ' ')}
          </span>
        );
      },
    },
    {
      key: 'stops',
      header: 'Stops',
      accessor: (row: Route) => `${row.stopsCompleted}/${row.stopsTotal}`,
    },
    {
      key: 'distance',
      header: 'Distance (m)',
      accessor: (row: Route) => `${row.totalDistance.toFixed(1)}m`,
    },
    {
      key: 'estTime',
      header: 'Est Time',
      accessor: (row: Route) => `${row.estimatedTimeMinutes} min`,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (row: Route) => (
        <div className="flex gap-2">
          {row.status === 'calculated' && (
            <Button size="sm" onClick={() => console.log('Start route:', row.routeId)}>
              Start
            </Button>
          )}
          <Link to={`/routing/${row.routeId}`}>
            <Button variant="outline" size="sm">View</Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleSearchChange = (value: string) => {
    setSearch(value);
    // Search is handled client-side - filter data locally if needed
  };

  const handleFilterChange = (key: keyof RouteFilters, value: string | undefined) => {
    setFilters({ ...filters, [key]: value });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Routes</h1>
          <p className="text-gray-500">{total} routes total</p>
        </div>
        <Link to="/routing/analysis">
          <Button variant="outline">
            <MapPin className="h-4 w-4 mr-2" />
            Route Analysis
          </Button>
        </Link>
      </div>

      <div className="flex gap-4 flex-wrap">
        <div className="relative flex-1 min-w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search routes..."
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>
        <select
          value={filters.status || ''}
          onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
          className="w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Status</option>
          <option value="calculated">Calculated</option>
          <option value="in_progress">In Progress</option>
          <option value="paused">Paused</option>
          <option value="completed">Completed</option>
          <option value="cancelled">Cancelled</option>
        </select>
        <select
          value={filters.strategy || ''}
          onChange={(e) => handleFilterChange('strategy', e.target.value || undefined)}
          className="w-48 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Strategies</option>
          <option value="shortest_path">Shortest Path</option>
          <option value="zone_based">Zone Based</option>
          <option value="priority_first">Priority First</option>
          <option value="batch_pick">Batch Pick</option>
        </select>
        <select
          value={filters.pickerId || ''}
          onChange={(e) => handleFilterChange('pickerId', e.target.value || undefined)}
          className="w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Pickers</option>
        </select>
      </div>

      {routes.length === 0 ? (
        <EmptyState
          icon={<RouteIcon className="h-12 w-12 text-gray-400" />}
          title="No routes found"
          description={search ? `No routes match "${search}"` : 'There are no routes yet. Routes will be created when picking begins.'}
        />
      ) : (
        <Table columns={columns} data={routes} keyExtractor={(route) => route.routeId} />
      )}
    </div>
  );
}
