import React from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { receivingClient } from '@wms/api-client';
import { Card, CardHeader, CardContent, Badge, Button, PageLoading, EmptyState } from '@wms/ui';
import { ArrowLeft, CheckCircle, Clock, AlertTriangle, Package, Truck, Calendar } from 'lucide-react';

export function ShipmentDetails() {
  const { shipmentId } = useParams<{ shipmentId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['shipment', shipmentId],
    queryFn: () => receivingClient.getShipment(shipmentId!),
  });

  const startMutation = useMutation({
    mutationFn: (dockId: string) => receivingClient.startReceiving(shipmentId!, dockId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shipment', shipmentId] });
      queryClient.invalidateQueries({ queryKey: ['shipments'] });
    },
  });

  if (isLoading) return <PageLoading message="Loading shipment details..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading shipment: {error?.message}</div>;

  const shipment = data;
  if (!shipment) {
    return (
      <EmptyState
        icon={<Package className="h-12 w-12 text-gray-400" />}
        title="Shipment not found"
        description="The requested shipment could not be found."
        action={{ label: 'Back to Shipments', onClick: () => navigate('/receiving') }}
      />
    );
  }

  const receivedItems = shipment.expectedItems.filter(item => item.receivedQuantity > 0);
  const pendingItems = shipment.expectedItems.filter(item => item.receivedQuantity < item.expectedQuantity);
  const discrepanciesCount = shipment.discrepancies?.length || 0;

  const statusSteps = [
    { label: 'Expected', status: 'expected' },
    { label: 'Arrived', status: 'arrived' },
    { label: 'Receiving', status: 'receiving' },
    { label: 'Completed', status: 'completed' },
  ];
  const currentStepIndex = statusSteps.findIndex(step => step.status === shipment.status);

  return (
    <div className="space-y-6">
      <Link to="/receiving" className="flex items-center text-gray-600 hover:text-gray-900 mb-4">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Shipments
      </Link>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader title="Status" subtitle={`Shipment ${shipment.shipmentId}`} />
          <CardContent>
            <div className="flex items-center gap-2">
              {shipment.status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
              {shipment.status === 'arrived' && <Clock className="h-5 w-5 text-yellow-600" />}
              {shipment.status === 'receiving' && <Package className="h-5 w-5 text-blue-600" />}
              {shipment.status === 'expected' && <AlertTriangle className="h-5 w-5 text-orange-600" />}
              <span className={`font-semibold capitalize`}>{shipment.status}</span>
            </div>
          </CardContent>
        </Card>

        {shipment.asn && (
          <Card>
            <CardHeader title="ASN Details" />
            <CardContent className="space-y-2">
              <div className="flex justify-between">
                <span className="text-gray-500">Carrier:</span>
                <span className="font-medium">{shipment.asn.shippingCarrier}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Tracking:</span>
                <span className="font-mono">{shipment.asn.trackingNumber}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">ETA:</span>
                <span className="font-medium">
                  {shipment.asn.estimatedArrival ? new Date(shipment.asn.estimatedArrival).toLocaleString() : '-'}
                </span>
              </div>
            </CardContent>
          </Card>
        )}

        {shipment.supplier && (
          <Card>
            <CardHeader title="Supplier" />
            <CardContent className="space-y-2">
              <div className="flex justify-between">
                <span className="text-gray-500">Name:</span>
                <span className="font-medium">{shipment.supplier.name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Code:</span>
                <span className="font-mono">{shipment.supplier.code}</span>
              </div>
              {shipment.supplier.contactEmail && (
                <div className="flex justify-between">
                  <span className="text-gray-500">Email:</span>
                  <span className="font-medium">{shipment.supplier.contactEmail}</span>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>

      <Card>
        <CardHeader title="Receiving Progress" />
        <CardContent>
          <div className="flex items-center justify-between mb-6">
            {statusSteps.map((step, index) => (
              <div key={step.status} className="flex-1 flex flex-col items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                  index < currentStepIndex ? 'bg-primary-600 text-white' :
                  index === currentStepIndex ? 'bg-primary-100 border-2 border-primary-600 text-primary-600' :
                  'bg-gray-200 text-gray-400'
                }`}>
                  {index < currentStepIndex ? <CheckCircle className="h-5 w-5" /> :
                   index === currentStepIndex ? <div className="w-3 h-3 rounded-full bg-primary-600" /> :
                   <div className="w-3 h-3 rounded-full bg-gray-400" />}
                </div>
                <span className="text-xs text-center mt-2">{step.label}</span>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600">{receivedItems.length}</div>
              <div className="text-sm text-gray-500">Items Received</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-orange-600">{pendingItems.length}</div>
              <div className="text-sm text-gray-500">Pending Items</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-red-600">{discrepanciesCount}</div>
              <div className="text-sm text-gray-500">Discrepancies</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {shipment.receiptRecords && shipment.receiptRecords.length > 0 && (
        <Card>
          <CardHeader title="Received Items" />
          <CardContent>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-4">SKU</th>
                  <th className="text-left py-2 px-4">Expected</th>
                  <th className="text-left py-2 px-4">Received</th>
                  <th className="text-left py-2 px-4">Condition</th>
                  <th className="text-left py-2 px-4">Tote</th>
                  <th className="text-left py-2 px-4">Location</th>
                  <th className="text-left py-2 px-4">Received By</th>
                </tr>
              </thead>
              <tbody>
                {shipment.receiptRecords.map((record) => (
                  <tr key={record.receiptId} className="border-b hover:bg-gray-50">
                    <td className="py-2 px-4">{record.sku}</td>
                    <td className="py-2 px-4">-</td>
                    <td className="py-2 px-4">{record.receivedQty}</td>
                    <td className="py-2 px-4 capitalize">{record.condition}</td>
                    <td className="py-2 px-4">{record.toteId || '-'}</td>
                    <td className="py-2 px-4">{record.locationId || '-'}</td>
                    <td className="py-2 px-4">{record.receivedBy || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      {discrepanciesCount > 0 && (
        <Card className="border-error-200">
          <CardHeader title="Discrepancies" />
          <CardContent>
            {shipment.discrepancies.map((discrepancy, index) => (
              <div key={index} className="p-3 border-l-4 border-error-500 bg-error-50 mb-2">
                <div className="flex justify-between items-start">
                  <div>
                    <span className="font-medium">{discrepancy.sku}</span>
                    <span className="ml-2 px-2 py-1 rounded text-xs font-medium uppercase bg-error-100 text-error-800">
                      {discrepancy.type.replace('_', ' ')}
                    </span>
                  </div>
                  <div className="text-right text-sm">
                    <div className="text-gray-500">Expected: {discrepancy.expectedQty}</div>
                    <div className="text-error-600 font-medium">Actual: {discrepancy.actualQty}</div>
                  </div>
                </div>
                <div className="text-sm text-gray-600 mt-1">{discrepancy.description}</div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <div className="flex justify-between">
        {shipment.status === 'arrived' && !shipment.assignedWorkerId && (
          <Button onClick={() => startMutation.mutate('DOCK-A1')}>
            <Package className="h-4 w-4 mr-2" />
            Start Receiving
          </Button>
        )}
        {shipment.status === 'receiving' && (
          <Link to={`/receiving/${shipmentId}/receive`}>
            <Button>
              <CheckCircle className="h-4 w-4 mr-2" />
              Receive Items
            </Button>
          </Link>
        )}
      </div>
    </div>
  );
}
