import React, { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { receivingClient } from '@wms/api-client';
import type { ReceiveItemRequest, ItemCondition } from '@wms/types';
import { Card, CardHeader, CardContent, Button, PageLoading, Select, Badge } from '@wms/ui';
import { ArrowLeft, CheckCircle2, XCircle, Package, Scan } from 'lucide-react';

export function ReceiveItemsForm() {
  const { shipmentId } = useParams<{ shipmentId: string }>();
  const queryClient = useQueryClient();

  const [receivedItems, setReceivedItems] = useState<Record<string, number>>({});
  const [selectedCondition, setSelectedCondition] = useState<ItemCondition>('good');

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['shipment', shipmentId],
    queryFn: () => receivingClient.getShipment(shipmentId!),
  });

  const receiveMutation = useMutation({
    mutationFn: ({ sku, quantity, condition, toteId, locationId, notes }: ReceiveItemRequest & { notes?: string }) => 
      receivingClient.receiveItem(shipmentId!, { sku, quantity, condition, toteId, locationId, workerId: 'CURRENT_USER', notes }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shipment', shipmentId] });
      queryClient.invalidateQueries({ queryKey: ['shipments'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => receivingClient.completeReceiving(shipmentId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shipment', shipmentId] });
      queryClient.invalidateQueries({ queryKey: ['shipments'] });
    },
  });

  if (isLoading) return <PageLoading message="Loading shipment..." />;
  if (isError) return <div className="p-6 text-error-600">Error: {error?.message}</div>;

  const shipment = data;
  if (!shipment) {
    return (
      <div className="p-6 text-center">
        <h2 className="text-xl font-semibold mb-2">Shipment not found</h2>
        <Link to="/receiving">Back to Shipments</Link>
      </div>
    );
  }

  const pendingItems = shipment.expectedItems.filter(item => {
    const received = receivedItems[item.sku] || 0;
    return received < item.expectedQuantity;
  });

  const allItemsReceived = pendingItems.length === 0;

  const handleReceiveItem = (sku: string, expectedQty: number) => {
    const received = receivedItems[sku] || 0;
    const remaining = expectedQty - received;

    if (remaining <= 0) {
      alert(`All expected quantity for ${sku} has been received`);
      return;
    }

    receiveMutation.mutate({
      sku,
      quantity: 1,
      condition: selectedCondition,
      toteId: `TOTE-${sku}`,
      locationId: `RCV-STAGE-01`,
      workerId: 'CURRENT_USER',
      notes: '',
    });
  };

  const handleQuickReceiveAll = () => {
    pendingItems.forEach(item => {
      const received = receivedItems[item.sku] || 0;
      if (received < item.expectedQuantity) {
        receiveMutation.mutate({
          sku: item.sku,
          quantity: item.expectedQuantity - received,
          condition: 'good',
          toteId: `TOTE-${item.sku}`,
          locationId: `RCV-STAGE-01`,
          workerId: 'CURRENT_USER',
          notes: '',
        });
      }
    });
  };

  const totalExpected = shipment.expectedItems.reduce((sum, item) => sum + item.expectedQuantity, 0);
  const totalReceived = Object.values(receivedItems).reduce((sum, qty) => sum + qty, 0);
  const progress = totalExpected > 0 ? (totalReceived / totalExpected) * 100 : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Receive Items</h1>
          <p className="text-gray-500">Shipment: {shipment.shipmentId}</p>
        </div>
        <Link to={`/receiving/${shipmentId}`}>
          <Button variant="outline">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Details
          </Button>
        </Link>
      </div>

      <Card>
        <CardHeader title="Receiving Progress" />
        <CardContent>
          <div className="mb-6">
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm text-gray-500">Progress</span>
              <span className="text-lg font-semibold">{Math.round(progress)}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-3">
              <div 
                className="bg-primary-600 h-3 rounded-full transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>

          <div className="flex gap-4 mb-6">
            <div className="flex-1 text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl font-bold text-blue-600">{totalReceived}</div>
              <div className="text-sm text-gray-600">Items Received</div>
            </div>
            <div className="flex-1 text-center p-4 bg-yellow-50 rounded-lg">
              <div className="text-2xl font-bold text-yellow-600">{pendingItems.length}</div>
              <div className="text-sm text-gray-600">Items Pending</div>
            </div>
            <div className="flex-1 text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-600">{totalExpected}</div>
              <div className="text-sm text-gray-600">Total Expected</div>
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Default Condition for Receiving:
            </label>
            <Select
              value={selectedCondition}
              onChange={(e) => setSelectedCondition(e.target.value as ItemCondition)}
              className="w-64"
              options={[
                { value: 'good', label: 'Good' },
                { value: 'damaged', label: 'Damaged' },
                { value: 'rejected', label: 'Rejected' },
              ]}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Items to Receive" subtitle={`${pendingItems.length} items pending`} />
        <CardContent>
          {pendingItems.length === 0 ? (
            <div className="text-center py-8">
              <CheckCircle2 className="h-12 w-12 text-green-500 mx-auto mb-4" />
              <p className="text-lg text-gray-600 mb-4">All items received!</p>
              <Link to={`/receiving/${shipmentId}`}>
                <Button>View Shipment Details</Button>
              </Link>
            </div>
          ) : (
            <>
              <div className="flex justify-end mb-4">
                <Button
                  onClick={handleQuickReceiveAll}
                  disabled={receiveMutation.isPending}
                  className="text-sm"
                >
                  <Scan className="h-4 w-4 mr-2" />
                  Receive All as Good
                </Button>
              </div>

              <div className="space-y-3">
                {pendingItems.map((item, index) => {
                  const received = receivedItems[item.sku] || 0;
                  const remaining = item.expectedQuantity - received;
                  const progress = (received / item.expectedQuantity) * 100;

                  return (
                    <div key={item.sku} className="border border-gray-200 rounded-lg p-4 hover:border-primary-300 transition-colors">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium">{item.productName}</span>
                            <Badge variant="neutral">{item.sku}</Badge>
                          </div>
                          <div className="text-sm text-gray-600">
                            Expected: {item.expectedQuantity} | Received: {received} | Remaining: {remaining}
                          </div>
                        </div>
                        <Button
                          size="sm"
                          onClick={() => handleReceiveItem(item.sku, item.expectedQuantity)}
                          disabled={receiveMutation.isPending}
                        >
                          <Package className="h-4 w-4 mr-2" />
                          Receive 1
                        </Button>
                      </div>

                      <div className="flex items-center gap-4">
                        <div className="flex-1">
                          <div className="w-full bg-gray-200 rounded-full h-2 mb-2">
                            <div 
                              className="bg-primary-600 h-2 rounded-full transition-all duration-300"
                              style={{ width: `${progress}%` }}
                            />
                          </div>
                          <div className="text-xs text-gray-500">
                            Tote: TOTE-{item.sku} | Location: RCV-STAGE-01
                          </div>
                        </div>
                      </div>

                      {remaining <= 0 && (
                        <div className="flex items-center text-green-600 text-sm mt-2">
                          <CheckCircle2 className="h-4 w-4 mr-2" />
                          Complete
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {allItemsReceived && (
        <Card className="border-green-200">
          <CardContent className="pt-6">
            <div className="text-center">
              <CheckCircle2 className="h-16 w-16 text-green-500 mx-auto mb-4" />
              <h2 className="text-xl font-semibold mb-2">All Items Received</h2>
              <p className="text-gray-600 mb-6">You've received all items for this shipment.</p>
              <div className="flex gap-4 justify-center">
                <Link to={`/receiving/${shipmentId}`}>
                  <Button variant="outline">
                    <ArrowLeft className="h-4 w-4 mr-2" />
                    Back to Details
                  </Button>
                </Link>
                <Button
                  onClick={() => completeMutation.mutate()}
                  disabled={completeMutation.isPending}
                  className="min-w-48"
                >
                  <CheckCircle2 className="h-4 w-4 mr-2" />
                  Complete Receiving
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
