import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { Route, SkipReason } from '@wms/types';
import { Card, CardHeader, CardContent, Badge, Button, EmptyState } from '@wms/ui';
import { ArrowLeft, CheckCircle2, XCircle, MapPin, ChevronRight, Clock } from 'lucide-react';

interface RouteControlsProps {
  routeId: string;
  status: string;
  onRefresh?: () => void;
}

export function RouteControls({ routeId, status, onRefresh }: RouteControlsProps) {
  const queryClient = useQueryClient();

  const startMutation = useMutation({
    mutationFn: () => routingClient.startRoute(routeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const pauseMutation = useMutation({
    mutationFn: (reason: string) => routingClient.pauseRoute(routeId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const resumeMutation = useMutation({
    mutationFn: () => routingClient.resumeRoute(routeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => routingClient.completeRoute(routeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (reason: string) => routingClient.cancelRoute(routeId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const canStart = status === 'calculated';
  const canPause = status === 'in_progress';
  const canResume = status === 'paused';
  const canComplete = ['in_progress', 'paused'].includes(status);
  const canCancel = !['completed', 'cancelled'].includes(status);

  const handleAction = (action: 'start' | 'pause' | 'resume' | 'complete' | 'cancel') => {
    switch (action) {
      case 'start':
        if (canStart) {
          startMutation.mutate();
        } else {
          alert('Route must be in "calculated" status to start');
        }
        break;
      case 'pause':
        if (canPause) {
          const reason = prompt('Enter pause reason (optional):');
          if (reason !== null) {
            pauseMutation.mutate(reason);
          }
        } else {
          pauseMutation.mutate('');
        }
        break;
      case 'resume':
        if (canResume) {
          resumeMutation.mutate();
        } else {
          alert('Route must be in "paused" status to resume');
        }
        break;
      case 'complete':
        if (canComplete) {
          completeMutation.mutate();
        } else {
          alert('Route must be in progress to complete');
        }
        break;
      case 'cancel':
        if (canCancel) {
          const reason = prompt('Enter cancellation reason:');
          if (reason && reason.trim()) {
            cancelMutation.mutate(reason.trim());
          }
        }
        break;
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">Route Controls</h3>
          <p className="text-sm text-gray-500">Manage route lifecycle</p>
        </div>
        {onRefresh && (
          <Button variant="outline" size="sm" onClick={onRefresh}>
            Refresh
          </Button>
        )}
      </div>

      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        <div
          className={`${canStart ? 'bg-blue-50 border-blue-300 cursor-pointer' : 'opacity-50 cursor-not-allowed'} transition-all border rounded-lg`}
          onClick={() => canStart && handleAction('start')}
        >
          <div className="pt-4 pb-4 text-center">
            <div className="h-12 w-12 rounded-full bg-blue-100 flex items-center justify-center mx-auto mb-2">
              <CheckCircle2 className="h-6 w-6 text-blue-600" />
            </div>
            <div className="font-semibold mb-1">Start Route</div>
            <div className="text-xs text-gray-600">Begin picking</div>
          </div>
        </div>

        <div
          className={`${canPause ? 'bg-yellow-50 border-yellow-300 cursor-pointer' : 'opacity-50 cursor-not-allowed'} transition-all border rounded-lg`}
          onClick={() => canPause && handleAction('pause')}
        >
          <div className="pt-4 pb-4 text-center">
            <div className="h-12 w-12 rounded-full bg-yellow-100 flex items-center justify-center mx-auto mb-2">
              <Clock className="h-6 w-6 text-yellow-600" />
            </div>
            <div className="font-semibold mb-1">Pause Route</div>
            <div className="text-xs text-gray-600">Temporary pause</div>
          </div>
        </div>

        <div
          className={`${canResume ? 'bg-green-50 border-green-300 cursor-pointer' : 'opacity-50 cursor-not-allowed'} transition-all border rounded-lg`}
          onClick={() => canResume && handleAction('resume')}
        >
          <div className="pt-4 pb-4 text-center">
            <div className="h-12 w-12 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-2">
              <CheckCircle2 className="h-6 w-6 text-green-600" />
            </div>
            <div className="font-semibold mb-1">Resume Route</div>
            <div className="text-xs text-gray-600">Continue picking</div>
          </div>
        </div>

        <div
          className={`${canComplete ? 'bg-purple-50 border-purple-300 cursor-pointer' : 'opacity-50 cursor-not-allowed'} transition-all border rounded-lg`}
          onClick={() => canComplete && handleAction('complete')}
        >
          <div className="pt-4 pb-4 text-center">
            <div className="h-12 w-12 rounded-full bg-purple-100 flex items-center justify-center mx-auto mb-2">
              <CheckCircle2 className="h-6 w-6 text-purple-600" />
            </div>
            <div className="font-semibold mb-1">Complete Route</div>
            <div className="text-xs text-gray-600">Finish all stops</div>
          </div>
        </div>

        <div
          className={`${canCancel ? 'bg-red-50 border-red-300 cursor-pointer' : 'opacity-50 cursor-not-allowed'} transition-all border rounded-lg`}
          onClick={() => canCancel && handleAction('cancel')}
        >
          <div className="pt-4 pb-4 text-center">
            <div className="h-12 w-12 rounded-full bg-red-100 flex items-center justify-center mx-auto mb-2">
              <XCircle className="h-6 w-6 text-red-600" />
            </div>
            <div className="font-semibold mb-1">Cancel Route</div>
            <div className="text-xs text-gray-600">Abort this route</div>
          </div>
        </div>
      </div>

      <div className="flex gap-3 mt-4 pt-4 border-t border-gray-200">
        <Button
          variant="outline"
          onClick={() => onRefresh?.()}
        >
          <MapPin className="h-4 w-4 mr-2" />
          View Route Map
        </Button>
        <Button
          variant="outline"
          onClick={() => onRefresh?.()}
        >
          <ChevronRight className="h-4 w-4 mr-2" />
          Next Stop
        </Button>
      </div>
    </div>
  );
}
