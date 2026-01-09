import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, AlertTriangle, RotateCcw, XCircle, RefreshCw } from 'lucide-react';
import {
  Card,
  CardHeader,
  Table,
  Column,
  Pagination,
  Button,
  StatusBadge,
  Badge,
  ConfirmDialog,
  EmptyState,
  PageLoading,
} from '@wms/ui';
import { orderClient } from '@wms/api-client';
import { formatDateTime, formatRelativeTime } from '@wms/utils';
import type { Order } from '@wms/types';

// Mock DLQ orders
const mockDLQOrders: Order[] = [
  {
    id: 'dlq-1',
    customerId: 'CUST-ERR-001',
    customerName: 'Failed Corp',
    orderNumber: 'ORD-2024-ERR-001',
    status: 'DLQ',
    priority: 'HIGH',
    items: [
      { id: '1', sku: 'SKU-404', productName: 'Missing Item', quantity: 5, pickedQuantity: 0, packedQuantity: 0 },
    ],
    createdAt: new Date(Date.now() - 86400000).toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'dlq-2',
    customerId: 'CUST-ERR-002',
    customerName: 'Error Inc',
    orderNumber: 'ORD-2024-ERR-002',
    status: 'DLQ',
    priority: 'RUSH',
    items: [
      { id: '2', sku: 'SKU-500', productName: 'Server Error Item', quantity: 2, pickedQuantity: 0, packedQuantity: 0 },
    ],
    createdAt: new Date(Date.now() - 172800000).toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

export function DLQOrders() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [showRetryDialog, setShowRetryDialog] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  // Fetch DLQ orders
  const { data: dlqResponse, isLoading, refetch } = useQuery({
    queryKey: ['orders-dlq', page],
    queryFn: () => orderClient.getDLQOrders(page, 20),
    retry: false,
    placeholderData: {
      data: mockDLQOrders,
      total: mockDLQOrders.length,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    },
  });

  const orders = dlqResponse?.data || [];
  const total = dlqResponse?.total || 0;
  const totalPages = dlqResponse?.totalPages || 1;

  // Retry mutation
  const retryMutation = useMutation({
    mutationFn: (orderId: string) => orderClient.retryDLQOrder(orderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders-dlq'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      setShowRetryDialog(false);
      setSelectedOrder(null);
    },
  });

  // Cancel mutation
  const cancelMutation = useMutation({
    mutationFn: (orderId: string) => orderClient.cancelOrder(orderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders-dlq'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      setShowCancelDialog(false);
      setSelectedOrder(null);
    },
  });

  const columns: Column<Order>[] = [
    {
      key: 'orderNumber',
      header: 'Order #',
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
      key: 'failedAt',
      header: 'Failed At',
      accessor: (order) => (
        <span className="text-gray-500" title={formatDateTime(order.updatedAt)}>
          {formatRelativeTime(order.updatedAt)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      align: 'right',
      accessor: (order) => (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setSelectedOrder(order);
              setShowRetryDialog(true);
            }}
          >
            <RotateCcw className="h-4 w-4 mr-1" />
            Retry
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setSelectedOrder(order);
              setShowCancelDialog(true);
            }}
            className="text-error-600 hover:text-error-700"
          >
            <XCircle className="h-4 w-4" />
          </Button>
        </div>
      ),
    },
  ];

  if (isLoading) {
    return <PageLoading message="Loading DLQ orders..." />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate('/orders')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-gray-900">Dead Letter Queue</h1>
              {total > 0 && (
                <Badge variant="error">{total} orders</Badge>
              )}
            </div>
            <p className="text-gray-500">Orders that failed processing and need attention</p>
          </div>
        </div>
        <Button variant="ghost" onClick={() => refetch()} icon={<RefreshCw className="h-4 w-4" />}>
          Refresh
        </Button>
      </div>

      {/* Alert Banner */}
      {total > 0 && (
        <div className="flex items-start gap-3 p-4 bg-error-50 border border-error-200 rounded-lg">
          <AlertTriangle className="h-5 w-5 text-error-500 mt-0.5" />
          <div>
            <p className="font-medium text-error-700">
              {total} order{total !== 1 ? 's' : ''} require attention
            </p>
            <p className="text-sm text-error-600 mt-1">
              These orders failed during processing and were moved to the dead letter queue.
              Review each order and either retry or cancel.
            </p>
          </div>
        </div>
      )}

      {/* DLQ Table */}
      <Card padding="none">
        {orders.length === 0 ? (
          <EmptyState
            title="No orders in DLQ"
            description="All orders are processing normally"
            variant="search"
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
              currentPage={page}
              totalPages={totalPages}
              pageSize={20}
              totalItems={total}
              onPageChange={setPage}
            />
          </>
        )}
      </Card>

      {/* Retry Dialog */}
      <ConfirmDialog
        isOpen={showRetryDialog}
        onClose={() => {
          setShowRetryDialog(false);
          setSelectedOrder(null);
        }}
        onConfirm={() => selectedOrder && retryMutation.mutate(selectedOrder.id)}
        title="Retry Order"
        message={`Are you sure you want to retry order ${selectedOrder?.orderNumber}? The order will be reprocessed from the beginning.`}
        confirmLabel="Retry Order"
        loading={retryMutation.isPending}
      />

      {/* Cancel Dialog */}
      <ConfirmDialog
        isOpen={showCancelDialog}
        onClose={() => {
          setShowCancelDialog(false);
          setSelectedOrder(null);
        }}
        onConfirm={() => selectedOrder && cancelMutation.mutate(selectedOrder.id)}
        title="Cancel Order"
        message={`Are you sure you want to cancel order ${selectedOrder?.orderNumber}? This action cannot be undone.`}
        confirmLabel="Cancel Order"
        variant="danger"
        loading={cancelMutation.isPending}
      />
    </div>
  );
}
