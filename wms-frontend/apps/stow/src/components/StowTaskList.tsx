import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { stowClient } from '@wms/api-client';
import type { StowTaskFilters } from '@wms/api-client';
import type { PutawayTask, ItemConstraints, StorageStrategy, PutawayStatus } from '@wms/types';
import { Table, Column, Input, Button, PageLoading, EmptyState, Badge } from '@wms/ui';
import { Box, Search, Filter, User, ArrowRight } from 'lucide-react';

export function StowTaskList() {
  const [filters, setFilters] = useState<StowTaskFilters>({});
  const [search, setSearch] = useState('');

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['stow-tasks', filters],
    queryFn: () => stowClient.getTasks(filters),
  });

  if (isLoading) return <PageLoading message="Loading stow tasks..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading stow tasks: {error?.message}</div>;

  const tasks = data?.data || [];
  const total = data?.total || 0;

  const columns: Column<PutawayTask>[] = [
    {
      key: 'taskId',
      header: 'Task ID',
      accessor: (row) => (
        <Link 
          to={`/stow/${row.taskId}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {row.taskId}
        </Link>
      ),
    },
    {
      key: 'sku',
      header: 'SKU',
      accessor: (row) => (
        <span className="font-mono text-sm">{row.sku}</span>
      ),
    },
    {
      key: 'productName',
      header: 'Product',
      accessor: (row) => row.productName,
    },
    {
      key: 'quantity',
      header: 'Quantity',
      accessor: (row) => (
        <div>
          <span className="font-semibold">{row.stowedQuantity}</span>
          <span className="text-gray-500">/</span>
          <span>{row.quantity}</span>
        </div>
      ),
    },
    {
      key: 'strategy',
      header: 'Strategy',
      accessor: (row) => {
        const strategyColors: Record<string, string> = {
          chaotic: 'bg-orange-100 text-orange-800',
          directed: 'bg-blue-100 text-blue-800',
          velocity: 'bg-green-100 text-green-800',
          zone_based: 'bg-purple-100 text-purple-800',
        };
        return (
          <Badge className={strategyColors[row.strategy] || ''}>
            {row.strategy.replace('_', ' ')}
          </Badge>
        );
      },
    },
    {
      key: 'constraints',
      header: 'Constraints',
      accessor: (row) => {
        const constraints = row.constraints || {};
        const badges = [];

        if (constraints.hazmat) {
          badges.push(<Badge key="hazmat" variant="error">Hazmat</Badge>);
        }
        if (constraints.coldChain) {
          badges.push(<Badge key="cold" variant="info">Cold Chain</Badge>);
        }
        if (constraints.oversized) {
          badges.push(<Badge key="oversized" variant="warning">Oversized</Badge>);
        }
        if (constraints.fragile) {
          badges.push(<Badge key="fragile" variant="neutral">Fragile</Badge>);
        }
        if (constraints.highValue) {
          badges.push(<Badge key="highvalue" variant="success">High Value</Badge>);
        }

        return badges.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {badges}
          </div>
        ) : (
          <span className="text-gray-400">-</span>
        );
      },
    },
    {
      key: 'targetLocation',
      header: 'Target Location',
      accessor: (row) => row.targetLocationId || '-',
    },
    {
      key: 'assignedWorker',
      header: 'Assigned To',
      accessor: (row) => row.assignedWorkerId || '-',
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (row) => {
        const statusColors: Record<string, string> = {
          pending: 'bg-blue-100 text-blue-800',
          assigned: 'bg-purple-100 text-purple-800',
          in_progress: 'bg-yellow-100 text-yellow-800',
          completed: 'bg-green-100 text-green-800',
          cancelled: 'bg-gray-100 text-gray-800',
          failed: 'bg-red-100 text-red-800',
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
      accessor: (row) => {
        const progress = row.quantity > 0 ? (row.stowedQuantity / row.quantity) * 100 : 0;
        return (
          <div className="flex items-center gap-2">
            <div className="flex-1 bg-gray-200 rounded-full h-2">
              <div
                className="bg-primary-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="text-sm text-gray-600 font-medium">{progress.toFixed(0)}%</span>
          </div>
        );
      },
    },
    {
      key: 'priority',
      header: 'Priority',
      accessor: (row) => row.priority ? `P${row.priority}` : '-',
      align: 'center',
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      accessor: (row) => (
        <div className="flex gap-2">
          {row.status === 'pending' && (
            <Button size="sm" onClick={() => console.log('Assign task:', row.taskId)}>
              <User className="h-4 w-4 mr-2" />
              Assign
            </Button>
          )}
          {row.status === 'assigned' && (
            <Button size="sm" onClick={() => console.log('Start task:', row.taskId)}>
              <Box className="h-4 w-4 mr-2" />
              Start Stow
            </Button>
          )}
          {row.status === 'in_progress' && (
            <Link to={`/stow/${row.taskId}`}>
              <Button size="sm" variant="outline">
                Continue
              </Button>
            </Link>
          )}
          <Link to={`/stow/${row.taskId}`}>
            <Button size="sm" variant="outline">
              View Details
            </Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleSearchChange = (value: string) => {
    setSearch(value);
    // Search is handled client-side - filter data locally if needed
  };

  const handleFilterChange = (key: keyof StowTaskFilters, value: string | undefined) => {
    const newFilters = { ...filters };
    if (value) {
      newFilters[key] = value as any;
    } else {
      delete newFilters[key];
    }
    setFilters(newFilters);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Putaway Tasks</h1>
          <p className="text-gray-500">{total} tasks total</p>
        </div>
      </div>

      <div className="flex gap-4 flex-wrap">
        <div className="relative flex-1 min-w-64">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            placeholder="Search tasks..."
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>
        <select
          value={filters.status}
          onChange={(e) => handleFilterChange('status', e.target.value)}
          className="w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Status</option>
          <option value="pending">Pending</option>
          <option value="assigned">Assigned</option>
          <option value="in_progress">In Progress</option>
          <option value="completed">Completed</option>
          <option value="cancelled">Cancelled</option>
          <option value="failed">Failed</option>
        </select>
        <select
          value={filters.strategy}
          onChange={(e) => handleFilterChange('strategy', e.target.value)}
          className="w-48 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Strategies</option>
          <option value="chaotic">Chaotic</option>
          <option value="directed">Directed</option>
          <option value="velocity">Velocity</option>
          <option value="zone_based">Zone Based</option>
        </select>
        <select
          value={filters.workerId}
          onChange={(e) => handleFilterChange('workerId', e.target.value)}
          className="w-48 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="">All Workers</option>
        </select>
      </div>

      {tasks.length === 0 ? (
        <EmptyState
          icon={<Box className="h-12 w-12 text-gray-400" />}
          title="No stow tasks found"
          description={search ? `No tasks match "${search}"` : 'There are no stow tasks yet. Tasks will be created when receiving completes.'}
        />
      ) : (
        <Table columns={columns} data={tasks} keyExtractor={(task) => task.taskId} />
      )}
    </div>
  );
}
