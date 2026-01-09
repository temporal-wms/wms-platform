import React, { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Package,
  Layers,
  MapPin,
  Box,
  Truck,
  Clock,
  AlertTriangle,
  RotateCcw,
  XCircle,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardContent,
  CardFooter,
  Button,
  StatusBadge,
  Badge,
  Table,
  Column,
  ConfirmDialog,
  PageLoading,
} from '@wms/ui';
import { orderClient } from '@wms/api-client';
import { formatDateTime, formatNumber } from '@wms/utils';
import type { Order, OrderItem } from '@wms/types';

// Mock order for development
const mockOrder: Order = {
  id: '1',
  customerId: 'CUST-001',
  customerName: 'Acme Corporation',
  orderNumber: 'ORD-2024-0001',
  status: 'PICKING',
  priority: 'HIGH',
  items: [
    { id: '1', sku: 'SKU-001', productName: 'Industrial Widget A', quantity: 5, pickedQuantity: 3, packedQuantity: 0, locationId: 'LOC-A-01-01' },
    { id: '2', sku: 'SKU-002', productName: 'Precision Gadget B', quantity: 3, pickedQuantity: 0, packedQuantity: 0, locationId: 'LOC-B-02-03' },
    { id: '3', sku: 'SKU-003', productName: 'Heavy Duty Tool C', quantity: 2, pickedQuantity: 2, packedQuantity: 0, locationId: 'LOC-A-03-02' },
  ],
  waveId: 'WV-2024-001',
  createdAt: new Date(Date.now() - 7200000).toISOString(),
  updatedAt: new Date().toISOString(),
};

export function OrderDetails() {
  const { orderId } = useParams<{ orderId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  // Fetch order
  const { data: order = mockOrder, isLoading } = useQuery({
    queryKey: ['order', orderId],
    queryFn: () => orderClient.getOrder(orderId!),
    enabled: !!orderId,
    retry: false,
  });

  // Cancel mutation
  const cancelMutation = useMutation({
    mutationFn: () => orderClient.cancelOrder(orderId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', orderId] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      setShowCancelDialog(false);
    },
  });

  // Retry DLQ mutation
  const retryMutation = useMutation({
    mutationFn: () => orderClient.retryDLQOrder(orderId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', orderId] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });

  if (isLoading) {
    return <PageLoading message="Loading order details..." />;
  }

  if (!order) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-gray-900">Order not found</h2>
        <Button variant="secondary" onClick={() => navigate('/orders')} className="mt-4">
          Back to Orders
        </Button>
      </div>
    );
  }

  const itemColumns: Column<OrderItem>[] = [
    {
      key: 'sku',
      header: 'SKU',
      accessor: (item) => <span className="font-mono text-sm">{item.sku}</span>,
    },
    {
      key: 'product',
      header: 'Product',
      accessor: (item) => item.productName,
    },
    {
      key: 'location',
      header: 'Location',
      accessor: (item) => (
        <span className="font-mono text-sm text-gray-500">{item.locationId || '-'}</span>
      ),
    },
    {
      key: 'quantity',
      header: 'Qty',
      align: 'center',
      accessor: (item) => formatNumber(item.quantity),
    },
    {
      key: 'picked',
      header: 'Picked',
      align: 'center',
      accessor: (item) => (
        <span className={item.pickedQuantity >= item.quantity ? 'text-success-600' : ''}>
          {item.pickedQuantity}/{item.quantity}
        </span>
      ),
    },
    {
      key: 'packed',
      header: 'Packed',
      align: 'center',
      accessor: (item) => (
        <span className={item.packedQuantity >= item.quantity ? 'text-success-600' : ''}>
          {item.packedQuantity}/{item.quantity}
        </span>
      ),
    },
  ];

  const canCancel = ['PENDING', 'VALIDATED'].includes(order.status);
  const canRetry = order.status === 'DLQ';

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
              <h1 className="text-2xl font-bold text-gray-900">{order.orderNumber}</h1>
              <StatusBadge status={order.status} />
              <Badge
                variant={
                  order.priority === 'RUSH'
                    ? 'error'
                    : order.priority === 'HIGH'
                    ? 'warning'
                    : 'neutral'
                }
              >
                {order.priority}
              </Badge>
            </div>
            <p className="text-gray-500">{order.customerName}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {canRetry && (
            <Button
              variant="primary"
              icon={<RotateCcw className="h-4 w-4" />}
              loading={retryMutation.isPending}
              onClick={() => retryMutation.mutate()}
            >
              Retry Order
            </Button>
          )}
          {canCancel && (
            <Button
              variant="danger"
              icon={<XCircle className="h-4 w-4" />}
              onClick={() => setShowCancelDialog(true)}
            >
              Cancel Order
            </Button>
          )}
        </div>
      </div>

      {/* Order Progress */}
      <OrderProgress status={order.status} />

      {/* Details Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Order Info */}
        <Card>
          <CardHeader title="Order Information" />
          <CardContent>
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-gray-500">Customer ID</dt>
                <dd className="font-medium">{order.customerId}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Created</dt>
                <dd className="font-medium">{formatDateTime(order.createdAt)}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Updated</dt>
                <dd className="font-medium">{formatDateTime(order.updatedAt)}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Total Items</dt>
                <dd className="font-medium">{order.items.length}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Total Qty</dt>
                <dd className="font-medium">
                  {order.items.reduce((sum, item) => sum + item.quantity, 0)}
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        {/* Related Resources */}
        <Card>
          <CardHeader title="Related Resources" />
          <CardContent>
            <div className="space-y-3">
              {order.waveId ? (
                <Link
                  to={`/waves/${order.waveId}`}
                  className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100"
                >
                  <div className="flex items-center gap-3">
                    <Layers className="h-5 w-5 text-primary-600" />
                    <span className="font-medium">Wave</span>
                  </div>
                  <span className="text-gray-500">{order.waveId}</span>
                </Link>
              ) : (
                <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg text-gray-400">
                  <div className="flex items-center gap-3">
                    <Layers className="h-5 w-5" />
                    <span>Wave</span>
                  </div>
                  <span>Not assigned</span>
                </div>
              )}
              {order.shipmentId ? (
                <Link
                  to={`/shipping/${order.shipmentId}`}
                  className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100"
                >
                  <div className="flex items-center gap-3">
                    <Truck className="h-5 w-5 text-primary-600" />
                    <span className="font-medium">Shipment</span>
                  </div>
                  <span className="text-gray-500">{order.shipmentId}</span>
                </Link>
              ) : (
                <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg text-gray-400">
                  <div className="flex items-center gap-3">
                    <Truck className="h-5 w-5" />
                    <span>Shipment</span>
                  </div>
                  <span>Not created</span>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Status Alert */}
        {order.status === 'DLQ' && (
          <Card>
            <CardHeader title="DLQ Alert" />
            <CardContent>
              <div className="flex items-start gap-3 p-3 bg-error-50 rounded-lg">
                <AlertTriangle className="h-5 w-5 text-error-500 mt-0.5" />
                <div>
                  <p className="font-medium text-error-700">Order in Dead Letter Queue</p>
                  <p className="text-sm text-error-600 mt-1">
                    This order failed processing and requires manual intervention. Review the error
                    and click Retry to reprocess.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Order Items */}
      <Card>
        <CardHeader
          title="Order Items"
          subtitle={`${order.items.length} items in this order`}
        />
        <Table
          columns={itemColumns}
          data={order.items}
          keyExtractor={(item) => item.id}
        />
      </Card>

      {/* Cancel Dialog */}
      <ConfirmDialog
        isOpen={showCancelDialog}
        onClose={() => setShowCancelDialog(false)}
        onConfirm={() => cancelMutation.mutate()}
        title="Cancel Order"
        message={`Are you sure you want to cancel order ${order.orderNumber}? This action cannot be undone.`}
        confirmLabel="Cancel Order"
        variant="danger"
        loading={cancelMutation.isPending}
      />
    </div>
  );
}

// Order Progress Component
function OrderProgress({ status }: { status: string }) {
  const steps = [
    { key: 'PENDING', label: 'Pending', icon: Clock },
    { key: 'VALIDATED', label: 'Validated', icon: Package },
    { key: 'WAVED', label: 'Waved', icon: Layers },
    { key: 'PICKING', label: 'Picking', icon: MapPin },
    { key: 'PACKING', label: 'Packing', icon: Box },
    { key: 'SHIPPING', label: 'Shipping', icon: Truck },
    { key: 'COMPLETED', label: 'Completed', icon: Package },
  ];

  const currentIndex = steps.findIndex((s) => s.key === status);
  const isFailed = ['FAILED', 'DLQ', 'CANCELLED'].includes(status);

  if (isFailed) {
    return (
      <div className="bg-error-50 border border-error-200 rounded-lg p-4">
        <div className="flex items-center gap-2 text-error-700">
          <AlertTriangle className="h-5 w-5" />
          <span className="font-medium">Order {status.toLowerCase()}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center justify-between">
        {steps.map((step, index) => {
          const Icon = step.icon;
          const isCompleted = index < currentIndex;
          const isCurrent = index === currentIndex;

          return (
            <React.Fragment key={step.key}>
              <div className="flex flex-col items-center">
                <div
                  className={`
                    w-10 h-10 rounded-full flex items-center justify-center
                    ${isCompleted ? 'bg-success-500 text-white' : ''}
                    ${isCurrent ? 'bg-primary-500 text-white' : ''}
                    ${!isCompleted && !isCurrent ? 'bg-gray-100 text-gray-400' : ''}
                  `}
                >
                  <Icon className="h-5 w-5" />
                </div>
                <span
                  className={`mt-2 text-xs font-medium ${
                    isCompleted || isCurrent ? 'text-gray-900' : 'text-gray-400'
                  }`}
                >
                  {step.label}
                </span>
              </div>
              {index < steps.length - 1 && (
                <div
                  className={`flex-1 h-0.5 mx-2 ${
                    index < currentIndex ? 'bg-success-500' : 'bg-gray-200'
                  }`}
                />
              )}
            </React.Fragment>
          );
        })}
      </div>
    </div>
  );
}
