import React, { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Play, XCircle, Package, Clock, CheckCircle } from 'lucide-react';
import {
  Card,
  CardHeader,
  CardContent,
  Button,
  StatusBadge,
  Badge,
  Table,
  Column,
  ConfirmDialog,
  PageLoading,
} from '@wms/ui';
import { waveClient, orderClient } from '@wms/api-client';
import { formatDateTime, formatRelativeTime } from '@wms/utils';
import type { Wave, Order } from '@wms/types';

// Mock data
const mockWave: Wave = {
  id: 'wv-1',
  waveNumber: 'WV-2024-001',
  status: 'IN_PROGRESS',
  orderIds: ['ord-1', 'ord-2', 'ord-3'],
  orderCount: 3,
  priority: 'HIGH',
  releasedAt: new Date(Date.now() - 1800000).toISOString(),
  createdAt: new Date(Date.now() - 3600000).toISOString(),
};

const mockOrders: Order[] = [
  {
    id: 'ord-1',
    customerId: 'CUST-001',
    customerName: 'Acme Corp',
    orderNumber: 'ORD-2024-0001',
    status: 'PICKING',
    priority: 'HIGH',
    items: [{ id: '1', sku: 'SKU-001', productName: 'Widget', quantity: 5, pickedQuantity: 2, packedQuantity: 0 }],
    waveId: 'wv-1',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'ord-2',
    customerId: 'CUST-002',
    customerName: 'TechStart',
    orderNumber: 'ORD-2024-0002',
    status: 'PICKING',
    priority: 'NORMAL',
    items: [{ id: '2', sku: 'SKU-002', productName: 'Gadget', quantity: 3, pickedQuantity: 0, packedQuantity: 0 }],
    waveId: 'wv-1',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'ord-3',
    customerId: 'CUST-003',
    customerName: 'Global Ltd',
    orderNumber: 'ORD-2024-0003',
    status: 'PICKED',
    priority: 'HIGH',
    items: [{ id: '3', sku: 'SKU-003', productName: 'Tool', quantity: 2, pickedQuantity: 2, packedQuantity: 0 }],
    waveId: 'wv-1',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

export function WaveDetails() {
  const { waveId } = useParams<{ waveId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showReleaseDialog, setShowReleaseDialog] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  const { data: wave = mockWave, isLoading } = useQuery({
    queryKey: ['wave', waveId],
    queryFn: () => waveClient.getWave(waveId!),
    enabled: !!waveId,
    retry: false,
  });

  // In real app, fetch orders by wave
  const orders = mockOrders;

  const releaseMutation = useMutation({
    mutationFn: () => waveClient.releaseWave(waveId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wave', waveId] });
      queryClient.invalidateQueries({ queryKey: ['waves'] });
      setShowReleaseDialog(false);
    },
  });

  const cancelMutation = useMutation({
    mutationFn: () => waveClient.cancelWave(waveId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wave', waveId] });
      queryClient.invalidateQueries({ queryKey: ['waves'] });
      setShowCancelDialog(false);
      navigate('/waves');
    },
  });

  if (isLoading) {
    return <PageLoading message="Loading wave details..." />;
  }

  const orderColumns: Column<Order>[] = [
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
      accessor: (order) => order.customerName,
    },
    {
      key: 'status',
      header: 'Status',
      accessor: (order) => <StatusBadge status={order.status} />,
    },
    {
      key: 'items',
      header: 'Items',
      align: 'center',
      accessor: (order) => {
        const total = order.items.reduce((sum, i) => sum + i.quantity, 0);
        const picked = order.items.reduce((sum, i) => sum + i.pickedQuantity, 0);
        return (
          <span className={picked >= total ? 'text-success-600' : ''}>
            {picked}/{total}
          </span>
        );
      },
    },
    {
      key: 'priority',
      header: 'Priority',
      accessor: (order) => {
        const variants: Record<string, 'error' | 'warning' | 'neutral'> = {
          RUSH: 'error',
          HIGH: 'warning',
          NORMAL: 'neutral',
        };
        return <Badge variant={variants[order.priority] || 'neutral'} size="sm">{order.priority}</Badge>;
      },
    },
  ];

  const canRelease = wave.status === 'READY';
  const canCancel = ['PLANNING', 'READY'].includes(wave.status);

  const completedOrders = orders.filter((o) => ['PICKED', 'PACKING', 'PACKED', 'COMPLETED'].includes(o.status)).length;
  const progressPercent = (completedOrders / orders.length) * 100;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate('/waves')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-gray-900">{wave.waveNumber}</h1>
              <StatusBadge status={wave.status} />
              <Badge variant={wave.priority === 'RUSH' ? 'error' : wave.priority === 'HIGH' ? 'warning' : 'neutral'}>
                {wave.priority}
              </Badge>
            </div>
            <p className="text-gray-500">{wave.orderCount} orders in this wave</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {canRelease && (
            <Button icon={<Play className="h-4 w-4" />} onClick={() => setShowReleaseDialog(true)}>
              Release Wave
            </Button>
          )}
          {canCancel && (
            <Button variant="danger" icon={<XCircle className="h-4 w-4" />} onClick={() => setShowCancelDialog(true)}>
              Cancel Wave
            </Button>
          )}
        </div>
      </div>

      {/* Progress */}
      {wave.status === 'IN_PROGRESS' && (
        <Card>
          <CardHeader title="Wave Progress" />
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Orders Completed</span>
                <span className="font-medium">{completedOrders} / {orders.length}</span>
              </div>
              <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary-500 rounded-full transition-all duration-500"
                  style={{ width: `${progressPercent}%` }}
                />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Wave Info */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader title="Wave Information" />
          <CardContent>
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-gray-500">Created</dt>
                <dd className="font-medium">{formatDateTime(wave.createdAt)}</dd>
              </div>
              {wave.releasedAt && (
                <div className="flex justify-between">
                  <dt className="text-gray-500">Released</dt>
                  <dd className="font-medium">{formatDateTime(wave.releasedAt)}</dd>
                </div>
              )}
              {wave.completedAt && (
                <div className="flex justify-between">
                  <dt className="text-gray-500">Completed</dt>
                  <dd className="font-medium">{formatDateTime(wave.completedAt)}</dd>
                </div>
              )}
              <div className="flex justify-between">
                <dt className="text-gray-500">Order Count</dt>
                <dd className="font-medium">{wave.orderCount}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader title="Status Summary" />
          <CardContent>
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="flex justify-center mb-2">
                  <Clock className="h-6 w-6 text-warning-500" />
                </div>
                <p className="text-2xl font-bold text-gray-900">
                  {orders.filter((o) => ['PENDING', 'VALIDATED', 'WAVED'].includes(o.status)).length}
                </p>
                <p className="text-sm text-gray-500">Pending</p>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="flex justify-center mb-2">
                  <Package className="h-6 w-6 text-primary-500" />
                </div>
                <p className="text-2xl font-bold text-gray-900">
                  {orders.filter((o) => o.status === 'PICKING').length}
                </p>
                <p className="text-sm text-gray-500">Picking</p>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="flex justify-center mb-2">
                  <CheckCircle className="h-6 w-6 text-success-500" />
                </div>
                <p className="text-2xl font-bold text-gray-900">{completedOrders}</p>
                <p className="text-sm text-gray-500">Completed</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Orders */}
      <Card>
        <CardHeader title="Orders in Wave" subtitle={`${orders.length} orders`} />
        <Table
          columns={orderColumns}
          data={orders}
          keyExtractor={(order) => order.id}
          onRowClick={(order) => navigate(`/orders/${order.id}`)}
        />
      </Card>

      {/* Dialogs */}
      <ConfirmDialog
        isOpen={showReleaseDialog}
        onClose={() => setShowReleaseDialog(false)}
        onConfirm={() => releaseMutation.mutate()}
        title="Release Wave"
        message={`Release wave ${wave.waveNumber}? This will create pick tasks for all ${wave.orderCount} orders.`}
        confirmLabel="Release Wave"
        loading={releaseMutation.isPending}
      />

      <ConfirmDialog
        isOpen={showCancelDialog}
        onClose={() => setShowCancelDialog(false)}
        onConfirm={() => cancelMutation.mutate()}
        title="Cancel Wave"
        message={`Cancel wave ${wave.waveNumber}? Orders will be removed from the wave and returned to available status.`}
        confirmLabel="Cancel Wave"
        variant="danger"
        loading={cancelMutation.isPending}
      />
    </div>
  );
}
