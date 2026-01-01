import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import { Card, CardHeader, CardContent, CardFooter, Button, Input, Select } from '@wms/ui';
import { orderClient, CreateOrderRequest } from '@wms/api-client';

interface OrderItemForm {
  sku: string;
  productName: string;
  quantity: number;
}

export function CreateOrder() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [customerId, setCustomerId] = useState('');
  const [customerName, setCustomerName] = useState('');
  const [priority, setPriority] = useState<'LOW' | 'NORMAL' | 'HIGH' | 'RUSH'>('NORMAL');
  const [items, setItems] = useState<OrderItemForm[]>([
    { sku: '', productName: '', quantity: 1 },
  ]);

  const createMutation = useMutation({
    mutationFn: (request: CreateOrderRequest) => orderClient.createOrder(request),
    onSuccess: (order) => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      navigate(`/orders/${order.id}`);
    },
  });

  const addItem = () => {
    setItems([...items, { sku: '', productName: '', quantity: 1 }]);
  };

  const removeItem = (index: number) => {
    if (items.length > 1) {
      setItems(items.filter((_, i) => i !== index));
    }
  };

  const updateItem = (index: number, field: keyof OrderItemForm, value: string | number) => {
    const updated = [...items];
    updated[index] = { ...updated[index], [field]: value };
    setItems(updated);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const request: CreateOrderRequest = {
      customerId,
      customerName,
      priority,
      items: items.filter((item) => item.sku && item.quantity > 0),
    };

    createMutation.mutate(request);
  };

  const isValid =
    customerId.trim() &&
    customerName.trim() &&
    items.some((item) => item.sku.trim() && item.quantity > 0);

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={() => navigate('/orders')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Create Order</h1>
          <p className="text-gray-500">Add a new order to the system</p>
        </div>
      </div>

      <form onSubmit={handleSubmit}>
        {/* Customer Information */}
        <Card className="mb-6">
          <CardHeader title="Customer Information" />
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                label="Customer ID"
                placeholder="e.g., CUST-001"
                value={customerId}
                onChange={(e) => setCustomerId(e.target.value)}
                required
              />
              <Input
                label="Customer Name"
                placeholder="e.g., Acme Corp"
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
                required
              />
            </div>
          </CardContent>
        </Card>

        {/* Order Details */}
        <Card className="mb-6">
          <CardHeader title="Order Details" />
          <CardContent>
            <Select
              label="Priority"
              options={[
                { value: 'LOW', label: 'Low' },
                { value: 'NORMAL', label: 'Normal' },
                { value: 'HIGH', label: 'High' },
                { value: 'RUSH', label: 'Rush (Expedited)' },
              ]}
              value={priority}
              onChange={(e) => setPriority(e.target.value as typeof priority)}
            />
          </CardContent>
        </Card>

        {/* Order Items */}
        <Card className="mb-6">
          <CardHeader
            title="Order Items"
            action={
              <Button variant="secondary" size="sm" onClick={addItem} type="button">
                <Plus className="h-4 w-4 mr-1" /> Add Item
              </Button>
            }
          />
          <CardContent>
            <div className="space-y-4">
              {items.map((item, index) => (
                <div
                  key={index}
                  className="flex items-end gap-4 p-4 bg-gray-50 rounded-lg"
                >
                  <div className="flex-1">
                    <Input
                      label="SKU"
                      placeholder="e.g., SKU-001"
                      value={item.sku}
                      onChange={(e) => updateItem(index, 'sku', e.target.value)}
                      required
                    />
                  </div>
                  <div className="flex-1">
                    <Input
                      label="Product Name"
                      placeholder="e.g., Widget A"
                      value={item.productName}
                      onChange={(e) => updateItem(index, 'productName', e.target.value)}
                      required
                    />
                  </div>
                  <div className="w-24">
                    <Input
                      label="Qty"
                      type="number"
                      min={1}
                      value={item.quantity}
                      onChange={(e) => updateItem(index, 'quantity', parseInt(e.target.value) || 1)}
                      required
                    />
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    type="button"
                    onClick={() => removeItem(index)}
                    disabled={items.length === 1}
                    className="text-gray-400 hover:text-error-500"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3">
          <Button variant="secondary" type="button" onClick={() => navigate('/orders')}>
            Cancel
          </Button>
          <Button
            type="submit"
            loading={createMutation.isPending}
            disabled={!isValid}
          >
            Create Order
          </Button>
        </div>
      </form>
    </div>
  );
}
