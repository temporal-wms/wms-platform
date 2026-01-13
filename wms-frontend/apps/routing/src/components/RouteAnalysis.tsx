import React, { useState } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { routingClient, type SkipReason } from '@wms/api-client';
import type { Route } from '@wms/types';
import { Modal, Button, Select, Input, Card, CardHeader, CardContent } from '@wms/ui';
import { AlertTriangle, Box, Lock } from 'lucide-react';

interface RouteAnalysisProps {
  routeId: string;
}

const skipReasonLabels: Record<SkipReason, string> = {
  out_of_stock: 'Out of Stock',
  location_blocked: 'Location Blocked',
  item_damaged: 'Item Damaged',
  other: 'Other',
};

const skipReasonDescriptions: Record<SkipReason, string> = {
  out_of_stock: 'No items available at location',
  location_blocked: 'Physical access blocked by pallet or other item',
  item_damaged: 'Items at location are damaged',
  other: 'Other reason - please specify',
};

export function RouteAnalysis({ routeId }: RouteAnalysisProps) {
  const [skipReason, setSkipReason] = useState<string>('');
  const [skipNotes, setSkipNotes] = useState('');
  const [showSkipModal, setShowSkipModal] = useState(false);
  const queryClient = useQueryClient();

  const { data: analysisData, isLoading: isLoadingAnalysis } = useQuery({
    queryKey: ['route-analysis', routeId],
    queryFn: () => routingClient.getRouteAnalysis(routeId),
    enabled: !!routeId,
  });

  const { data: routeData } = useQuery({
    queryKey: ['route', routeId],
    queryFn: () => routingClient.getRoute(routeId),
    enabled: !!routeId,
  });

  const skipMutation = useMutation({
    mutationFn: (stopNumber: number) =>
      routingClient.skipStop(routeId, stopNumber, skipReason as SkipReason, skipNotes),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      setShowSkipModal(false);
      setSkipReason('');
      setSkipNotes('');
    },
  });

  if (isLoadingAnalysis || routeData === undefined) {
    return <div className="p-6 text-center">Loading analysis...</div>;
  }

  const analysis = analysisData;
  const route = routeData;
  if (!analysis || !route) {
    return <div className="p-6 text-center">Analysis not available</div>;
  }

  const efficiencyColor = analysis.efficiency >= 80 ? 'text-green-600' : analysis.efficiency >= 60 ? 'text-yellow-600' : 'text-red-600';
  const efficiencyBg = analysis.efficiency >= 80 ? 'bg-green-100' : analysis.efficiency >= 60 ? 'bg-yellow-100' : 'bg-red-100';

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader title="Efficiency Score" />
          <CardContent>
            <div className="text-center">
              <div className={`inline-block px-6 py-3 rounded-full text-3xl font-bold ${efficiencyBg}`}>
                {analysis.efficiency.toFixed(1)}%
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Time Comparison" />
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-500">Estimated:</span>
                <span className="font-semibold">{analysis.estimatedVsActualTime.estimatedMinutes} min</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Actual:</span>
                <span className="font-semibold">{analysis.estimatedVsActualTime.actualMinutes} min</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Variance:</span>
                <span className={`font-semibold ${analysis.estimatedVsActualTime.variance > 0 ? 'text-red-600' : 'text-green-600'}`}>
                  {analysis.estimatedVsActualTime.variance > 0 ? '+' : ''}
                  {(analysis.estimatedVsActualTime.variance * 100).toFixed(1)}%
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Distance Metrics" />
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-500">Total Distance:</span>
                <span className="font-semibold">{analysis.distanceMetrics.totalDistance.toFixed(1)}m</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Average per Stop:</span>
                <span className="font-semibold">{analysis.distanceMetrics.averagePerStop.toFixed(1)}m</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader title="Completion Rate" />
          <CardContent>
            <div className="text-center">
              <div className="text-4xl font-bold text-blue-600">
                {(analysis.completionRate * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-gray-500 mt-2">
                {route.stopsCompleted} / {route.stopsTotal} stops
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Skipped Stops" />
          <CardContent>
            <div className="text-center">
              <div className="text-4xl font-bold text-red-600">
                {analysis.skippedStops}
              </div>
              <div className="text-sm text-gray-500 mt-2">
                stops skipped due to issues
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {analysis.recommendations && analysis.recommendations.length > 0 && (
        <Card>
          <CardHeader title="Recommendations" />
          <CardContent>
            <ul className="space-y-2">
              {analysis.recommendations.map((recommendation, index) => (
                <li key={index} className="flex items-start">
                  <AlertTriangle className="h-5 w-5 text-yellow-600 mt-0.5 mr-2 flex-shrink-0" />
                  <span className="text-gray-700">{recommendation}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
