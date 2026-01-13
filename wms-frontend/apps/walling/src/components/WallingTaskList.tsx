import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { wallingClient } from '@wms/api-client';
import type { WallingTaskFilters } from '@wms/api-client';
import type { WallingTask } from '@wms/types';
import { Table, Column, Input, Select, Button, PageLoading, EmptyState, Badge } from '@wms/ui';
import { Layers, Search, Filter, User, MapPin, ChevronDown, ChevronUp } from 'lucide-react';

export function WallingTaskList() {
  const [filters, setFilters] = useState<WallingTaskFilters>({});
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['walling-tasks', filters],
    queryFn: () => wallingClient.getTasks(filters),
  });

  if (isLoading) return <PageLoading message="Loading walling tasks..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading walling tasks: {error?.message}</div>;

  const tasks = data?.data || [];
  const total = data?.total || 0;

  const columns: Column<WallingTask>[] = [
    {
      key: 'taskId',
      header: 'Task ID',
      accessor: (row: WallingTask) => (
        <Link 
          to={`/walling/${row.taskId}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {row.taskId}
        </Link>
      ),
    },
    {
      key: 'orderId',
      header: 'Order ID',
      accessor: (row: WallingTask) => row.orderId,
    },
    {
      key: 'waveId',
      header: 'Wave ID',
      accessor: (row: WallingTask) => row.waveId,
    },
    {
      key: 'putWall',
      header: 'Put Wall',
      accessor: (row: WallingTask) => (
        <Badge variant="neutral">{row.putWallId}</Badge>
      ),
    },
    {
      key: 'bin',
      header: 'Destination Bin',
      accessor: (row: WallingTask) => row.destinationBin,
    },
    {
      key: 'walliner',
      header: 'Walliner',
      accessor: (row: WallingTask) => row.wallinerId || '-',
    },
    {
      key: 'itemsTotal',
      header: 'Items',
      accessor: (row: WallingTask) => `${row.itemsToSort.reduce((sum: number, item: any) => sum + item.quantity, 0)}`,
      },
    {
      key: 'itemsSorted',
      header: 'Sorted',
      accessor: (row: WallingTask) => `${row.sortedItems.reduce((sum: number, item: any) => sum + item.quantity, 0)}`,
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (row: WallingTask) => {
        const statusColors: Record<string, string> = {
          pending: 'bg-blue-100 text-blue-800',
          assigned: 'bg-purple-100 text-purple-800',
          in_progress: 'bg-yellow-100 text-yellow-800',
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
      key: 'progress',
      header: 'Progress',
      accessor: (row: WallingTask) => {
        const total = row.itemsToSort.reduce((sum: number, item: any) => sum + item.quantity, 0);
        const sorted = row.sortedItems.reduce((sum: number, item: any) => sum + item.quantity, 0);
        const progress = total > 0 ? (sorted / total) * 100 : 0;
        return (
          <div className="w-24 bg-gray-200 rounded-full h-2">
            <div 
              className="bg-primary-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        );
      },
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (row: WallingTask) => (
        <div className="flex gap-2">
          {row.status === 'pending' && (
            <Button size="sm" onClick={() => console.log('Assign:', row.taskId)}>
              <User className="h-4 w-4 mr-2" />
              Assign
            </Button>
          )}
          <Link to={`/walling/${row.taskId}`}>
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

  const handleFilterChange = (key: keyof WallingTaskFilters, value: string | undefined) => {
    setFilters({ ...filters, [key]: value });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Put-Wall Tasks</h1>
          <p className="text-gray-500">{total} tasks total</p>
        </div>
        <Link to="/walling/new">
          <Button>
            <Layers className="h-4 w-4 mr-2" />
            Create Walling Task
          </Button>
        </Link>
      </div>

      <div className="flex gap-4 flex-wrap">
        <div className="relative flex-1 min-w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search walling tasks..."
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
          <option value="pending">Pending</option>
          <option value="assigned">Assigned</option>
          <option value="in_progress">In Progress</option>
          <option value="completed">Completed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      {tasks.length === 0 ? (
        <EmptyState
          icon={<Layers className="h-12 w-12 text-gray-400" />}
          title="No walling tasks found"
          description={search ? `No tasks match "${search}"` : 'There are no walling tasks yet. Create your first task to get started.'}
        />
      ) : (
        <Table columns={columns} data={tasks} keyExtractor={(task) => task.taskId} />
      )}
    </div>
  );
}
