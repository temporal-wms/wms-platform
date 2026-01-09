import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Check, Package } from 'lucide-react';
import { Card, CardHeader, CardContent, Button, Select, Badge, Table, Column, PageLoading } from '@wms/ui';
import { waveClient, CreateWaveRequest } from '@wms/api-client';

interface AvailableOrder {
  id: string;
  orderNumber: string;
  priority: string;
  itemCount: number;
}

// Mock available orders
const mockAvailableOrders: AvailableOrder[] = [
  { id: 'ord-10', orderNumber: 'ORD-2024-0010', priority: 'HIGH', itemCount: 5 },
  { id: 'ord-11', orderNumber: 'ORD-2024-0011', priority: 'RUSH', itemCount: 2 },
  { id: 'ord-12', orderNumber: 'ORD-2024-0012', priority: 'NORMAL', itemCount: 8 },
  { id: 'ord-13', orderNumber: 'ORD-2024-0013', priority: 'NORMAL', itemCount: 3 },
  { id: 'ord-14', orderNumber: 'ORD-2024-0014', priority: 'HIGH', itemCount: 4 },
  { id: 'ord-15', orderNumber: 'ORD-2024-0015', priority: 'LOW', itemCount: 1 },
];

export function CreateWave() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [selectedOrders, setSelectedOrders] = useState<Set<string>>(new Set());
  const [priority, setPriority] = useState<'LOW' | 'NORMAL' | 'HIGH' | 'RUSH'>('NORMAL');

  const { data: availableOrders = mockAvailableOrders, isLoading } = useQuery({
    queryKey: ['available-orders'],
    queryFn: () => waveClient.getAvailableOrders(1, 100).then((res) => res.data),
    retry: false,
  });

  const createMutation = useMutation({
    mutationFn: (request: CreateWaveRequest) => waveClient.createWave(request),
    onSuccess: (wave) => {
      queryClient.invalidateQueries({ queryKey: ['waves'] });
      navigate(`/waves/${wave.id}`);
    },
  });

  const toggleOrder = (orderId: string) => {
    const newSelected = new Set(selectedOrders);
    if (newSelected.has(orderId)) {
      newSelected.delete(orderId);
    } else {
      newSelected.add(orderId);
    }
    setSelectedOrders(newSelected);
  };

  const selectAll = () => {
    if (selectedOrders.size === availableOrders.length) {
      setSelectedOrders(new Set());
    } else {
      setSelectedOrders(new Set(availableOrders.map((o) => o.id)));
    }
  };

  const handleSubmit = () => {
    createMutation.mutate({
      orderIds: Array.from(selectedOrders),
      priority,
    });
  };

  const columns: Column<AvailableOrder>[] = [
    {
      key: 'select',
      header: (
        <input
          type="checkbox"
          checked={selectedOrders.size === availableOrders.length && availableOrders.length > 0}
          onChange={selectAll}
          className="rounded border-gray-300"
        />
      ),
      width: '50px',
      accessor: (order) => (
        <input
          type="checkbox"
          checked={selectedOrders.has(order.id)}
          onChange={() => toggleOrder(order.id)}
          className="rounded border-gray-300"
        />
      ),
    },
    {
      key: 'orderNumber',
      header: 'Order #',
      accessor: (order) => <span className="font-medium">{order.orderNumber}</span>,
    },
    {
      key: 'priority',
      header: 'Priority',
      accessor: (order) => {
        const variants: Record<string, 'error' | 'warning' | 'neutral' | 'success'> = {
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
        <div className="flex items-center justify-center gap-1">
          <Package className="h-4 w-4 text-gray-400" />
          {order.itemCount}
        </div>
      ),
    },
  ];

  if (isLoading) {
    return <PageLoading message="Loading available orders..." />;
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={() => navigate('/waves')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Create Wave</h1>
          <p className="text-gray-500">Select orders to group into a wave for picking</p>
        </div>
      </div>

      {/* Wave Settings */}
      <Card>
        <CardHeader title="Wave Settings" />
        <CardContent>
          <div className="max-w-xs">
            <Select
              label="Wave Priority"
              options={[
                { value: 'LOW', label: 'Low' },
                { value: 'NORMAL', label: 'Normal' },
                { value: 'HIGH', label: 'High' },
                { value: 'RUSH', label: 'Rush (Expedited)' },
              ]}
              value={priority}
              onChange={(e) => setPriority(e.target.value as typeof priority)}
            />
          </div>
        </CardContent>
      </Card>

      {/* Order Selection */}
      <Card>
        <CardHeader
          title="Select Orders"
          subtitle={`${availableOrders.length} orders available for waving`}
          action={
            selectedOrders.size > 0 && (
              <Badge variant="info">{selectedOrders.size} selected</Badge>
            )
          }
        />
        <Table
          columns={columns}
          data={availableOrders}
          keyExtractor={(order) => order.id}
          onRowClick={(order) => toggleOrder(order.id)}
          emptyMessage="No orders available for waving"
        />
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-between p-4 bg-white border border-gray-200 rounded-lg">
        <div className="text-sm text-gray-600">
          {selectedOrders.size} order{selectedOrders.size !== 1 ? 's' : ''} selected
        </div>
        <div className="flex items-center gap-3">
          <Button variant="secondary" onClick={() => navigate('/waves')}>
            Cancel
          </Button>
          <Button
            icon={<Check className="h-4 w-4" />}
            disabled={selectedOrders.size === 0}
            loading={createMutation.isPending}
            onClick={handleSubmit}
          >
            Create Wave
          </Button>
        </div>
      </div>
    </div>
  );
}
