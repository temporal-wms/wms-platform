import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { Route } from '@wms/types';
import { Card, CardHeader, CardContent, Button, Badge, EmptyState } from '@wms/ui';
import { User, RefreshCw, CheckCircle2 } from 'lucide-react';

interface PickerAssignmentProps {
  routeId?: string;
  waveId?: string;
  onAssign?: (pickerId: string) => void;
}

export function PickerAssignment({ routeId, waveId, onAssign }: PickerAssignmentProps) {
  const queryClient = useQueryClient();
  const [availablePickers, setAvailablePickers] = useState<string[]>([]);
  const [assignedRoutes, setAssignedRoutes] = useState<Record<string, string>>({});

  const { data: availableRoutesData } = useQuery({
    queryKey: ['pending-routes'],
    queryFn: () => routingClient.getPendingRoutes(20),
  });

  const { data: workersData } = useQuery({
    queryKey: ['workers'],
    queryFn: async () => {
      return [
        { id: 'PICKER-001', name: 'John Doe', currentTask: null, tasksToday: 15, zone: 'zone-a' },
        { id: 'PICKER-002', name: 'Jane Smith', currentTask: 'RT-abc123', tasksToday: 12, zone: 'zone-b' },
        { id: 'PICKER-003', name: 'Bob Johnson', currentTask: null, tasksToday: 8, zone: 'zone-a' },
        { id: 'PICKER-004', name: 'Alice Williams', currentTask: 'RT-def456', tasksToday: 20, zone: 'zone-c' },
      ];
    },
  });

  const assignMutation = useMutation({
    mutationFn: ({ pickerId, targetRouteId }: { pickerId: string; targetRouteId: string }) =>
      routingClient.startRoute(targetRouteId),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      queryClient.invalidateQueries({ queryKey: ['route', variables.targetRouteId] });
      setAssignedRoutes(prev => ({ ...prev, [variables.pickerId]: variables.targetRouteId }));
      onAssign?.(variables.pickerId);
    },
  });

  const availableRoutes = availableRoutesData || [];
  const workers = workersData || [];

  const handleAssign = (pickerId: string, targetRouteId: string) => {
    assignMutation.mutate({ pickerId, targetRouteId });
  };

  const getWorkerStatus = (workerId: string) => {
    if (assignedRoutes[workerId]) {
      return 'assigned';
    }
    const worker = workers.find(w => w.id === workerId);
    if (worker?.currentTask) {
      return 'busy';
    }
    return 'available';
  };

  const availableWorkers = workers.filter(worker => getWorkerStatus(worker.id) === 'available');
  const busyWorkers = workers.filter(worker => getWorkerStatus(worker.id) === 'busy');

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">Assign Picker to Route</h3>
          <p className="text-sm text-gray-500">
            {availableRoutes.length} unassigned routes available
          </p>
        </div>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader
            title="Available Pickers"
            subtitle={`${availableWorkers.length} pickers available`}
          />
          <CardContent>
            {availableWorkers.length === 0 ? (
              <EmptyState
                icon={<User className="h-12 w-12 text-gray-400" />}
                title="No pickers available"
                description="All pickers are currently assigned or busy."
              />
            ) : (
              <div className="space-y-3">
                {availableWorkers.map((worker) => (
                  <div
                    key={worker.id}
                    className="flex items-center justify-between p-3 border border-gray-200 rounded-lg hover:border-primary-300 transition-colors cursor-pointer"
                    onClick={() => setAvailablePickers(prev => 
                      prev.includes(worker.id) 
                        ? prev.filter(id => id !== worker.id)
                        : [...prev, worker.id]
                    )}
                  >
                    <div className="flex items-center gap-3 flex-1">
                      <div className="flex items-center gap-2">
                        <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-700">
                          <span className="font-semibold">{worker.name.charAt(0)}</span>
                        </div>
                        <div>
                          <div className="font-medium text-gray-900">{worker.name}</div>
                          <div className="text-sm text-gray-500">{worker.id}</div>
                        </div>
                      </div>
                      <Badge variant="neutral">{worker.zone}</Badge>
                      <div className="text-sm text-gray-600">
                        {worker.tasksToday} tasks today
                      </div>
                    </div>
                    <div className="text-right">
                      {availablePickers.includes(worker.id) && (
                        <Badge className="bg-blue-100 text-blue-800">Selected</Badge>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader
            title="Unassigned Routes"
            subtitle={`${availableRoutes.length} routes`}
          />
          <CardContent>
            {availableRoutes.length === 0 ? (
              <EmptyState
                icon={<CheckCircle2 className="h-12 w-12 text-green-400" />}
                title="All routes assigned"
                description="All pending routes have been assigned to pickers."
              />
            ) : (
              <div className="space-y-2">
                {availableRoutes.map((route) => {
                  const routeWorkers = availablePickers.length > 0 
                    ? workers.filter(w => availablePickers.includes(w.id))
                    : workers;

                  return (
                    <div
                      key={route.routeId}
                      className="p-3 border border-gray-200 rounded-lg hover:border-primary-300 transition-colors"
                    >
                      <div className="flex items-start justify-between mb-2">
                        <div className="flex-1">
                          <div className="font-mono text-sm font-medium text-primary-600">
                            {route.routeId}
                          </div>
                          <div className="text-sm text-gray-600">
                            {route.stopsTotal} stops • {route.estimatedTimeMinutes} min
                          </div>
                        </div>
                        <div className="text-xs text-gray-500">
                          {new Date(route.createdAt).toLocaleString()}
                        </div>
                      </div>

                      {availablePickers.length > 0 && (
                        <div className="pt-2 border-t border-gray-200 mt-2">
                          <p className="text-sm text-gray-500 mb-2">Assign to picker:</p>
                          <div className="flex flex-wrap gap-2">
                            {routeWorkers.map((worker) => (
                              <Button
                                key={worker.id}
                                size="sm"
                                onClick={() => handleAssign(worker.id, route.routeId)}
                                disabled={assignMutation.isPending || getWorkerStatus(worker.id) !== 'available'}
                                className="flex items-center gap-2"
                              >
                                <User className="h-4 w-4" />
                                <div className="text-left">
                                  <div className="font-medium">{worker.name}</div>
                                  <div className="text-xs text-gray-500">{worker.id}</div>
                                </div>
                              </Button>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
