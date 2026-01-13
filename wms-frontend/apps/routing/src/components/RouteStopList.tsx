import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { RouteStop } from '@wms/types';
import { Card, CardHeader, CardContent, Button, Badge, EmptyState } from '@wms/ui';
import { MapPin, CheckCircle2, XCircle, SkipForward, Package } from 'lucide-react';

interface RouteStopListProps {
  routeId: string;
  stops: RouteStop[];
  currentStopSequence?: number;
}

export function RouteStopList({ routeId, stops, currentStopSequence }: RouteStopListProps) {
  const { data: routeData } = useQuery({
    queryKey: ['route', routeId],
    queryFn: () => routingClient.getRoute(routeId),
    enabled: !!routeId,
  });

  const route = routeData;
  const completedStops = stops.filter(s => s.status === 'completed').length;
  const skippedStops = stops.filter(s => s.status === 'skipped').length;
  const pendingStops = stops.filter(s => s.status === 'pending').length;
  const totalStops = stops.length;
  const progress = totalStops > 0 ? (completedStops / totalStops) * 100 : 0;

  const [showSkipModal, setShowSkipModal] = useState<{ stopNumber: number; sequence: number } | null>(null);
  const [showCompleteModal, setShowCompleteModal] = useState<{ stopNumber: number; sequence: number } | null>(null);

  const handleSkipClick = (sequence: number) => {
    setShowSkipModal({ stopNumber: sequence, sequence });
  };

  const handleCompleteClick = (sequence: number) => {
    setShowCompleteModal({ stopNumber: sequence, sequence });
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader
          title={`Stop Progress: ${completedStops}/${totalStops}`}
          subtitle={`${skippedStops} skipped`}
        />
        <CardContent>
          <div className="mb-6">
            <div className="w-full bg-gray-200 rounded-full h-3">
              <div 
                className="bg-primary-600 h-3 rounded-full transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-6 mb-6">
            <div className="text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl font-bold text-blue-600">{completedStops}</div>
              <div className="text-sm text-gray-600">Completed</div>
            </div>
            <div className="text-center p-4 bg-yellow-50 rounded-lg">
              <div className="text-2xl font-bold text-yellow-600">{skippedStops}</div>
              <div className="text-sm text-gray-600">Skipped</div>
            </div>
            <div className="text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-600">{pendingStops}</div>
              <div className="text-sm text-gray-600">Pending</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {stops.length === 0 ? (
        <EmptyState
          icon={<MapPin className="h-12 w-12 text-gray-400" />}
          title="No stops found"
          description="This route has no stops defined."
        />
      ) : (
        <div className="space-y-3">
          {stops.map((stop, index) => {
            const isCurrentStop = currentStopSequence === stop.sequence;
            const isCompleted = stop.status === 'completed';
            const isSkipped = stop.status === 'skipped';
            const isPending = stop.status === 'pending';

            return (
              <Card 
                key={stop.sequence} 
                className={`border-2 transition-all ${
                  isCompleted ? 'border-green-500 bg-green-50' :
                  isSkipped ? 'border-red-500 bg-red-50' :
                  isPending ? 'border-gray-300' :
                  'border-blue-300 bg-blue-50'
                }`}
              >
                <CardContent className="pt-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3 flex-1">
                      <div className={`w-10 h-10 rounded-full flex items-center justify-center text-lg font-semibold ${
                        isCompleted ? 'bg-green-100 text-green-700' :
                        isSkipped ? 'bg-red-100 text-red-700' :
                        isPending ? 'bg-blue-100 text-blue-700' :
                        'bg-gray-100 text-gray-700'
                      }`}>
                        {stop.sequence}
                      </div>

                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <MapPin className="h-5 w-5 text-gray-600" />
                          <span className="font-mono text-sm font-medium text-gray-900">{stop.locationId}</span>
                          <Badge variant="neutral" className="ml-2">{stop.zone}</Badge>
                        </div>
                        <div className="font-medium text-gray-900">{stop.sku}</div>
                        <div className="text-sm text-gray-600">
                          Qty: {stop.quantity} | {isCompleted ? `Actual: ${stop.actualQuantity || 0}` : ''}
                        </div>
                        {stop.skipReason && (
                          <div className="text-sm text-red-600 mt-1">
                            {stop.skipReason.replace('_', ' ')}
                          </div>
                        )}
                      </div>
                    </div>

                    {isCurrentStop && isPending && (
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleSkipClick(stop.sequence)}
                          disabled={isCompleted || isSkipped}
                        >
                          <SkipForward className="h-4 w-4 mr-2" />
                          Skip
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => handleCompleteClick(stop.sequence)}
                          disabled={isCompleted || isSkipped}
                        >
                          <CheckCircle2 className="h-4 w-4 mr-2" />
                          Complete
                        </Button>
                      </div>
                    )}

                    {isCompleted && (
                      <div className="text-right">
                        <div className="flex items-center gap-2 mb-1">
                          <Package className="h-5 w-5 text-green-600" />
                          <span className="font-semibold text-green-600">Picked</span>
                        </div>
                        <div className="text-sm text-gray-600">
                          {stop.completedAt && new Date(stop.completedAt).toLocaleTimeString()}
                        </div>
                      </div>
                    )}

                    {isSkipped && (
                      <div className="text-right">
                        <div className="flex items-center gap-2 mb-1">
                          <XCircle className="h-5 w-5 text-red-600" />
                          <span className="font-semibold text-red-600">Skipped</span>
                        </div>
                        <div className="text-xs text-gray-500">
                          Reason: {stop.notes || stop.skipReason?.replace('_', ' ')}
                        </div>
                      </div>
                    )}

                    {isPending && !isCurrentStop && (
                      <div className="text-center text-sm text-gray-500">
                        Pending
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
