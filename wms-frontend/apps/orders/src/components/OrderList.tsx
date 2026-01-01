import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, Link } from 'react-router-dom';
import { Plus, AlertTriangle, RefreshCw } from 'lucide-react';
import {
  Card,
  CardHeader,
  CardContent,
  Table,
  Column,
  Pagination,
  Button,
  StatusBadge,
  Badge,
  SearchInput,
  Select,
  EmptyState,
  PageLoading,
} from '@wms/ui';
import { orderClient, OrderFilters } from '@wms/api-client';
import { formatDateTime, formatRelativeTime } from '@wms/utils';
import type { Order } from '@wms/types';

const statusOptions = [
  { value: '', label: 'All Statuses' },
  { value: 'PENDING', label: 'Pending' },
  { value: 'VALIDATED', label: 'Validated' },
  { value: 'WAVED', label: 'Waved' },
  { value: 'PICKING', label: 'Picking' },
  { value: 'PACKING', label: 'Packing' },
  { value: 'SHIPPING', label: 'Shipping' },
  { value: 'COMPLETED', label: 'Completed' },
  { value: 'FAILED', label: 'Failed' },
];

const priorityOptions = [
  { value: '', label: 'All Priorities' },
  { value: 'LOW', label: 'Low' },
  { value: 'NORMAL', label: 'Normal' },
  { value: 'HIGH', label: 'High' },
  { value: 'RUSH', label: 'Rush' },
];

export function OrderList() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<OrderFilters>({
    page: 1,
    pageSize: 20,
  });
  const [search, setSearch] = useState('');

  // Fetch orders from backend
  const {
    data: ordersResponse,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['orders', filters],
    queryFn: () => orderClient.getOrders(filters),
    retry: 1,
  });

  const orders = ordersResponse?.data || [];
  const total = ordersResponse?.total || 0;
  const totalPages = ordersResponse?.totalPages || 1;

  const columns: Column<Order>[] = [
    {
      key: 'orderNumber',
      header: 'Order #',
      sortable: true,
      accessor: (order) => (
        <Link
          to={`/orders/${order.id}`}
          className="font-medium text-primary-600 hover:text-primary-700"
        >
          {order.orderNumber}
        </Link>
      ),
    },
    {
      key: 'customer',
      header: 'Customer',
      accessor: (order) => (
        <div>
          <p className="font-medium text-gray-900">{order.customerName}</p>
          <p className="text-sm text-gray-500">{order.customerId}</p>
        </div>
      ),
    },
    {
      key: 'items',
      header: 'Items',
      align: 'center',
      accessor: (order) => (
        <Badge variant="neutral" size="sm">
          {order.items.length} items
        </Badge>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      accessor: (order) => <StatusBadge status={order.status} />,
    },
    {
      key: 'priority',
      header: 'Priority',
      accessor: (order) => {
        const variants: Record<string, 'error' | 'warning' | 'success' | 'neutral'> = {
          RUSH: 'error',
          HIGH: 'warning',
          NORMAL: 'neutral',
          LOW: 'success',
        };
        return <Badge variant={variants[order.priority] || 'neutral'}>{order.priority}</Badge>;
      },
    },
    {
      key: 'createdAt',
      header: 'Created',
      sortable: true,
      accessor: (order) => (
        <span className="text-gray-500" title={formatDateTime(order.createdAt)}>
          {formatRelativeTime(order.createdAt)}
        </span>
      ),
    },
  ];

  const handleSearch = (value: string) => {
    setSearch(value);
    setFilters((prev) => ({ ...prev, search: value, page: 1 }));
  };

  const handleFilterChange = (key: keyof OrderFilters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, page: 1 }));
  };

  if (isLoading) {
    return <PageLoading message="Loading orders..." />;
  }

  if (isError) {
    return (
      <Card padding="lg">
        <div className="text-center py-8">
          <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Failed to load orders</h2>
          <p className="text-gray-500 mb-4">
            {error instanceof Error ? error.message : 'Unable to connect to the order service'}
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
          <h1 className="text-2xl font-bold text-gray-900">Orders</h1>
          <p className="text-gray-500">Manage and track customer orders</p>
        </div>
        <div className="flex items-center gap-3">
          <Link to="/orders/dlq">
            <Button variant="outline" icon={<AlertTriangle className="h-4 w-4" />}>
              DLQ Orders
            </Button>
          </Link>
          <Link to="/orders/new">
            <Button icon={<Plus className="h-4 w-4" />}>New Order</Button>
          </Link>
        </div>
      </div>

      {/* Filters */}
      <Card padding="md">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex-1 min-w-[200px] max-w-md">
            <SearchInput
              placeholder="Search orders..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={handleSearch}
            />
          </div>
          <Select
            options={statusOptions}
            value={filters.status || ''}
            onChange={(e) => handleFilterChange('status', e.target.value)}
          />
          <Select
            options={priorityOptions}
            value={filters.priority || ''}
            onChange={(e) => handleFilterChange('priority', e.target.value)}
          />
          <Button variant="ghost" onClick={() => refetch()} icon={<RefreshCw className="h-4 w-4" />}>
            Refresh
          </Button>
        </div>
      </Card>

      {/* Orders Table */}
      <Card padding="none">
        {orders.length === 0 ? (
          <EmptyState
            title="No orders found"
            description="Create your first order or adjust your filters"
            action={{
              label: 'Create Order',
              onClick: () => navigate('/orders/new'),
            }}
          />
        ) : (
          <>
            <Table
              columns={columns}
              data={orders}
              keyExtractor={(order) => order.id}
              onRowClick={(order) => navigate(`/orders/${order.id}`)}
            />
            <Pagination
              currentPage={filters.page || 1}
              totalPages={totalPages}
              pageSize={filters.pageSize || 20}
              totalItems={total}
              onPageChange={(page) => setFilters((prev) => ({ ...prev, page }))}
              onPageSizeChange={(pageSize) => setFilters((prev) => ({ ...prev, pageSize, page: 1 }))}
            />
          </>
        )}
      </Card>
    </div>
  );
}
