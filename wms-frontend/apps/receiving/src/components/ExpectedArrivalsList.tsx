import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { receivingClient } from '@wms/api-client';
import type { ReceivingShipment } from '@wms/types';
import { Card, CardHeader, CardContent, Button, PageLoading, EmptyState, Badge } from '@wms/ui';
import { Package, Truck, Calendar, Clock, ChevronDown, ChevronUp } from 'lucide-react';

export function ExpectedArrivalsList() {
  const [selectedDate, setSelectedDate] = useState<string>(new Date().toISOString().split('T')[0]);
  const [expandedShipments, setExpandedShipments] = useState<Set<string>>(new Set());

  const { data, isLoading, isError } = useQuery({
    queryKey: ['expected-arrivals', selectedDate],
    queryFn: () => receivingClient.getExpectedArrivals(selectedDate),
  });

  const toggleExpand = (shipmentId: string) => {
    const newExpanded = new Set(expandedShipments);
    if (newExpanded.has(shipmentId)) {
      newExpanded.delete(shipmentId);
    } else {
      newExpanded.add(shipmentId);
    }
    setExpandedShipments(newExpanded);
  };

  if (isLoading) return <PageLoading message="Loading expected arrivals..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading expected arrivals</div>;

  const shipments = data || [];

  const groupedShipments = shipments.reduce((acc, shipment) => {
    const date = shipment.asn?.estimatedArrival 
      ? new Date(shipment.asn.estimatedArrival).toISOString().split('T')[0]
      : 'unknown';
    if (!acc[date]) acc[date] = [];
    acc[date].push(shipment);
    return acc;
  }, {} as Record<string, ReceivingShipment[]>);

  const sortedDates = Object.keys(groupedShipments).sort((a, b) => new Date(b).getTime() - new Date(a).getTime());
  const displayDates = [selectedDate, ...sortedDates.filter(d => d !== selectedDate)];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Expected Arrivals</h1>
          <p className="text-gray-500">Track upcoming inbound shipments by arrival date</p>
        </div>
        <Button
          variant="outline"
          onClick={() => setSelectedDate(new Date().toISOString().split('T')[0])}
        >
          <Calendar className="h-4 w-4 mr-2" />
          Today
        </Button>
      </div>

      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Select Date:
        </label>
        <input
          type="date"
          value={selectedDate}
          onChange={(e) => setSelectedDate(e.target.value)}
          className="w-64 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
        />
      </div>

      {displayDates.map((date) => {
        const dateShipments = groupedShipments[date] || [];
        if (dateShipments.length === 0) return null;

        const isToday = date === new Date().toISOString().split('T')[0];
        const formattedDate = isToday 
          ? 'Today' 
          : new Date(date).toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' });

        return (
          <div key={date} className="space-y-3">
            <h2 className="text-lg font-semibold text-gray-800 mb-3">
              {formattedDate}
              <span className="ml-2 px-2 py-1 bg-primary-100 text-primary-700 rounded text-sm">
                {dateShipments.length} shipment{dateShipments.length > 1 ? 's' : ''}
              </span>
            </h2>

            {dateShipments.map((shipment) => {
              const isExpanded = expandedShipments.has(shipment.shipmentId);
              const eta = shipment.asn?.estimatedArrival
                ? new Date(shipment.asn.estimatedArrival)
                : null;
              const etaDate = eta ? eta.toLocaleDateString() : '-';
              const etaTime = eta ? eta.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '-';
              const daysUntilArrival = eta ? Math.ceil((eta.getTime() - Date.now()) / (1000 * 60 * 60 * 24)) : null;
              const urgencyBadge = daysUntilArrival
                ? daysUntilArrival <= 1
                  ? { text: 'Today', variant: 'error' as const }
                  : daysUntilArrival <= 3
                  ? { text: 'This Week', variant: 'warning' as const }
                  : { text: 'Upcoming', variant: 'success' as const }
                : { text: 'TBD', variant: 'neutral' as const };

              return (
                <Card key={shipment.shipmentId} className="hover:shadow-md transition-shadow">
                  <CardHeader
                    title={shipment.purchaseOrderId}
                    subtitle={shipment.supplier?.name || 'Unknown Supplier'}
                  />
                  <CardContent>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-3">
                          <Package className="h-5 w-5 text-primary-600" />
                          <span className="font-mono text-sm font-medium">{shipment.shipmentId}</span>
                          <Badge variant={urgencyBadge.variant}>{urgencyBadge.text}</Badge>
                        </div>

                        <div className="space-y-2 text-sm text-gray-600">
                          <div className="flex items-center gap-2">
                            <Truck className="h-4 w-4" />
                            <span className="font-medium">{shipment.asn?.shippingCarrier || '-'}</span>
                            <span className="font-mono">{shipment.asn?.trackingNumber || '-'}</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Calendar className="h-4 w-4" />
                            <span>Arriving: <strong>{etaDate}</strong> at <strong>{etaTime}</strong></span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Clock className="h-4 w-4" />
                            <span>
                              Expected Items: <strong className="text-gray-900">
                                {shipment.expectedItems.reduce((sum, item) => sum + item.expectedQuantity, 0)}
                              </strong>
                            </span>
                          </div>
                        </div>

                        {isExpanded && (
                          <div className="mt-4 pt-4 border-t border-gray-200 space-y-2">
                            <h4 className="text-sm font-semibold text-gray-900 mb-2">Expected Items:</h4>
                            {shipment.expectedItems.map((item, index) => (
                              <div key={index} className="flex items-center justify-between py-2 hover:bg-gray-50 rounded px-2">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-sm font-medium">{item.sku}</span>
                                  <span className="text-gray-600">{item.productName}</span>
                                </div>
                                <span className="font-semibold text-gray-900">x{item.expectedQuantity}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>

                      <div className="flex flex-col items-center gap-2">
                        <button
                          onClick={() => toggleExpand(shipment.shipmentId)}
                          className="p-2 hover:bg-gray-100 rounded transition-colors"
                        >
                          {isExpanded ? (
                            <ChevronUp className="h-4 w-4 text-gray-600" />
                          ) : (
                            <ChevronDown className="h-4 w-4 text-gray-600" />
                          )}
                        </button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        );
      })}

      {shipments.length === 0 && (
        <EmptyState
          icon={<Package className="h-12 w-12 text-gray-400" />}
          title="No expected arrivals"
          description={selectedDate 
            ? `There are no shipments expected to arrive on ${new Date(selectedDate).toLocaleDateString()}`
            : 'Select a date to view expected arrivals'
          }
        />
      )}
    </div>
  );
}
