import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { stowClient } from '@wms/api-client';
import { Card, CardHeader, CardContent, Badge, Button, EmptyState } from '@wms/ui';
import { User, RefreshCw, CheckCircle2, Search, Package } from 'lucide-react';

interface WorkerTaskAssignmentProps {
  taskId?: string;
}

export function WorkerTaskAssignment({ taskId }: WorkerTaskAssignmentProps) {
  const queryClient = useQueryClient();
  const [selectedWorker, setSelectedWorker] = useState<string | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  const { data: workersData } = useQuery({
    queryKey: ['workers'],
    queryFn: async () => {
      return [
        { id: 'PICKER-001', name: 'John Doe', currentTask: null, tasksToday: 15, zone: 'zone-a' },
        { id: 'PICKER-002', name: 'Jane Smith', currentTask: null, tasksToday: 12, zone: 'zone-b' },
        { id: 'PICKER-003', name: 'Bob Johnson', currentTask: 'ST-abc123', tasksToday: 8, zone: 'zone-a' },
        { id: 'PICKER-004', name: 'Alice Williams', currentTask: null, tasksToday: 20, zone: 'zone-c' },
      ];
    },
  });

  const { data: availableTasksData } = useQuery({
    queryKey: ['pending-tasks'],
    queryFn: () => stowClient.getPendingTasks(20),
  });

  const availableTasks = availableTasksData || [];
  const workers = workersData || [];
  const availableWorkers = workers.filter(w => !w.currentTask && w.zone !== 'zone-d');
  const busyWorkers = workers.filter(w => w.currentTask);

  const assignMutation = useMutation({
    mutationFn: ({ taskId, pickerId }: { taskId: string; pickerId: string }) =>
      stowClient.assignTask(taskId, pickerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stow-tasks'] });
      queryClient.invalidateQueries({ queryKey: ['pending-tasks'] });
      queryClient.invalidateQueries({ queryKey: ['task', selectedTaskId || taskId] });
    },
  });

  const handleAssign = (pickerId: string) => {
    if (!selectedTaskId || !selectedWorker) {
      alert('Please select a task and a worker');
      return;
    }
    assignMutation.mutate({ taskId: selectedTaskId, pickerId });
  };

  const handleTaskSelect = (taskId: string) => {
    setSelectedTaskId(taskId);
  };

  const handleWorkerSelect = (workerId: string) => {
    setSelectedWorker(workerId);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">Task Assignment</h3>
          <p className="text-sm text-gray-500">Assign stow tasks to available workers</p>
        </div>
        <Button variant="outline" onClick={() => window.location.reload()}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="lg:col-span-1">
          <CardHeader
            title="Pending Tasks"
            subtitle={`${availableTasks.length} unassigned`}
          />
          <CardContent>
            {availableTasks.length === 0 ? (
              <EmptyState
                icon={<Package className="h-12 w-12 text-gray-400" />}
                title="No pending tasks"
                description="There are no stow tasks waiting to be assigned. Tasks will be created when receiving completes."
              />
            ) : (
              <div className="space-y-2">
                {availableTasks.map((task) => (
                  <div
                    key={task.taskId}
                    className={`p-3 border-2 rounded-lg cursor-pointer transition-all ${
                      selectedTaskId === task.taskId
                        ? 'border-primary-500 bg-primary-50'
                        : 'border-gray-200 hover:border-primary-300 hover:bg-gray-50'
                    }`}
                    onClick={() => handleTaskSelect(task.taskId)}
                  >
                    <div className="flex justify-between items-start">
                      <div>
                        <div className="font-mono text-sm font-medium text-primary-600">
                          {task.taskId}
                        </div>
                        <div className="font-medium text-gray-900">
                          {task.productName}
                        </div>
                        <div className="text-sm text-gray-600">
                          SKU: {task.sku} | Qty: {task.quantity} | Zone: {task.targetLocationId?.split('-')[0] || '-'}
                        </div>
                      </div>
                      {selectedTaskId === task.taskId && (
                        <CheckCircle2 className="h-5 w-5 text-primary-600" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-1">
          <CardHeader
            title="Available Workers"
            subtitle={`${availableWorkers.length} available`}
          />
          <CardContent>
            <div className="space-y-4">
              {availableWorkers.length === 0 ? (
                <EmptyState
                  icon={<User className="h-12 w-12 text-gray-400" />}
                  title="No workers available"
                  description="All workers are currently busy or offline."
                />
              ) : (
                <div className="grid grid-cols-1 gap-4">
                  {availableWorkers.map((worker) => {
                    const isSelected = selectedWorker === worker.id;

                    return (
                      <div
                        key={worker.id}
                        className={`p-3 border-2 rounded-lg cursor-pointer transition-all ${
                          isSelected
                            ? 'border-primary-500 bg-primary-50'
                            : 'border-gray-200 hover:border-primary-300 hover:bg-gray-50'
                        }`}
                        onClick={() => handleWorkerSelect(worker.id)}
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-700">
                            <span className="font-semibold">{worker.name.charAt(0)}</span>
                          </div>
                          <div className="flex-1">
                            <div className="font-semibold text-gray-900">{worker.name}</div>
                            <div className="text-sm text-gray-600">
                              <div>Zone: <Badge variant="neutral">{worker.zone}</Badge></div>
                              <div>Tasks Today: {worker.tasksToday}</div>
                            </div>
                          </div>
                          {isSelected && (
                            <CheckCircle2 className="h-5 w-5 text-primary-600" />
                          )}
                        </div>
                        <Badge variant="success">
                          Available
                        </Badge>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader
          title="Busy Workers"
          subtitle={`${busyWorkers.length} currently assigned`}
        />
        <CardContent>
          {busyWorkers.length === 0 ? (
            <EmptyState
              icon={<User className="h-12 w-12 text-gray-400" />}
              title="No busy workers"
              description="All workers are currently available."
              />
          ) : (
            <div className="space-y-2">
              {busyWorkers.map((worker) => (
                <div key={worker.id} className="p-3 border border-gray-200 rounded-lg">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-yellow-100 flex items-center justify-center text-yellow-700">
                        <span className="font-semibold">{worker.name.charAt(0)}</span>
                      </div>
                      <div className="flex-1">
                        <div className="font-semibold text-gray-900">{worker.name}</div>
                        <div className="text-sm text-gray-600">
                          <div>Current Task: <span className="font-mono font-medium">{worker.currentTask || '-'}</span></div>
                          <div>Zone: <Badge variant="neutral">{worker.zone}</Badge></div>
                          <div>Tasks Today: {worker.tasksToday}</div>
                        </div>
                      </div>
                      <Badge variant="neutral">Busy</Badge>
                    </div>
                    <div className="text-sm text-gray-500">
                      Started: {worker.currentTask ? new Date().toLocaleTimeString() : '-'}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-3">
            <Button
              variant="outline"
              disabled={!selectedTaskId || !selectedWorker}
              onClick={() => window.location.reload()}
            >
              <Search className="h-4 w-4 mr-2" />
              Refresh Lists
            </Button>
            <Button
              disabled={!selectedTaskId || !selectedWorker}
              onClick={() => selectedWorker && handleAssign(selectedWorker)}
              className="flex-1"
            >
              <User className="h-4 w-4 mr-2" />
              Assign Task to Worker
            </Button>
          </div>
          <p className="text-sm text-gray-500 mt-2">
            Select a task from the pending list and a worker from the available list, then click Assign Task to Worker.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
