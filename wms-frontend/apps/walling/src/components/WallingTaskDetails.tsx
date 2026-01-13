import React, { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { wallingClient } from '@wms/api-client';
import type { WallingTask } from '@wms/types';
import { Card, CardHeader, CardContent, Badge, Button, PageLoading, EmptyState } from '@wms/ui';
import { ArrowLeft, CheckCircle2, CheckCircle, Box, Layers, User, MapPin, XCircle } from 'lucide-react';

export function WallingTaskDetails() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [showAssignModal, setShowAssignModal] = useState(false);
  const [selectedWallinerId, setSelectedWallinerId] = useState<string>('');
  const [selectedStation, setSelectedStation] = useState('');

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['walling-task', taskId],
    queryFn: () => wallingClient.getTask(taskId!),
  });

  const assignMutation = useMutation({
    mutationFn: ({ wallinerId, station }: { wallinerId: string; station?: string }) =>
      wallingClient.assignWalliner(taskId!, wallinerId, station),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['walling-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['walling-tasks'] });
      setShowAssignModal(false);
      setSelectedWallinerId('');
      setSelectedStation('');
    },
  });

  const sortMutation = useMutation({
    mutationFn: (data: { sku: string; quantity: number; fromToteId: string }) =>
      wallingClient.sortItem(taskId!, data.sku, data.quantity, data.fromToteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['walling-task', taskId] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => wallingClient.completeTask(taskId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['walling-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['walling-tasks'] });
      navigate('/walling');
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (reason: string) => wallingClient.cancelTask(taskId!, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['walling-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['walling-tasks'] });
      navigate('/walling');
    },
  });

  if (isLoading) return <PageLoading message="Loading walling task details..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading task: {error?.message}</div>;

  const task = data;
  if (!task) {
    return (
      <EmptyState
        icon={<Layers className="h-12 w-12 text-gray-400" />}
        title="Task not found"
        description="The requested walling task could not be found."
        action={{ label: 'Back to Walling Tasks', onClick: () => navigate('/walling') }}
      />
    );
  }

  const totalItems = task.itemsToSort.reduce((sum, item) => sum + item.quantity, 0);
  const sortedItems = task.sortedItems || [];
  const sortedCount = sortedItems.reduce((sum, item) => sum + item.quantity, 0);
  const progress = totalItems > 0 ? (sortedCount / totalItems) * 100 : 0;
  const canSort = task.wallinerId && sortedCount < totalItems;
  const canComplete = sortedCount === totalItems;
  const canAssign = !task.wallinerId && task.status === 'pending';

  const sourceTotes = task.sourceTotes || [];

  return (
    <div className="space-y-6">
      <Link to="/walling" className="flex items-center text-gray-600 hover:text-gray-900 mb-4">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Walling Tasks
      </Link>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader title="Status" />
          <CardContent>
            <div className="flex items-center gap-2">
              {task.status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
              {task.status === 'in_progress' && <Layers className="h-5 w-5 text-blue-600" />}
              {task.status === 'assigned' && <User className="h-5 w-5 text-purple-600" />}
              {task.status === 'pending' && <Box className="h-5 w-5 text-gray-600" />}
              <span className={`font-semibold capitalize`}>{task.status.replace('_', ' ')}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Put-Wall Information" />
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between">
                <div>
                  <span className="text-gray-500">Put Wall ID:</span>
                  <Badge variant="info">{task.putWallId}</Badge>
                </div>
                <div>
                  <span className="text-gray-500">Destination Bin:</span>
                  <Badge variant="neutral">{task.destinationBin}</Badge>
                </div>
              </div>
              <div className="flex justify-between">
                <div>
                  <span className="text-gray-500">Order ID:</span>
                  <Badge>{task.orderId}</Badge>
                </div>
                <div>
                  <span className="text-gray-500">Wave ID:</span>
                  <Badge>{task.waveId || '-'}</Badge>
                </div>
              </div>
            </div>

            {task.wallinerId && (
              <div className="mt-3 pt-4 border-t border-gray-200">
                <div className="flex justify-between">
                  <div>
                    <span className="text-gray-500">Assigned Walliner:</span>
                    <span className="font-medium">{task.wallinerId}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">Station:</span>
                    <Badge variant="info">{task.station || '-'}</Badge>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Sorting Progress" />
          <CardContent>
            <div className="mb-4">
              <div className="flex justify-between items-center mb-2">
                <span className="text-2xl font-bold text-primary-600">{sortedCount}</span>
                <span className="text-sm text-gray-500">Items Sorted</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-3">
                <div 
                  className="bg-primary-600 h-3 rounded-full transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
            <div className="flex justify-between items-center mb-2">
              <div className="text-2xl font-bold text-yellow-600">{sortedCount > 0 ? `${totalItems - sortedCount} items` : '0 items'}</div>
              <div className="text-sm text-gray-500">of {totalItems} total</div>
            </div>
          </CardContent>
        </Card>

      <Card>
        <CardHeader title="Source Totes" />
        <CardContent>
          {sourceTotes.length === 0 ? (
            <div className="text-center py-8">
              <Layers className="h-12 w-12 text-gray-400 mx-auto mb-4" />
              <p className="text-lg text-gray-600 mb-4">No source totes</p>
            </div>
          ) : (
            <div className="space-y-2">
              {sourceTotes.map((tote, index) => (
                <div key={index} className="flex items-center justify-between p-3 border border-gray-200 rounded-lg">
                  <div className="flex-1">
                    <div className="text-sm font-semibold text-gray-900">{tote.pickTaskId}</div>
                    <div className="text-xs text-gray-500">Pick Task</div>
                  </div>
                  <div className="text-sm text-gray-600">
                    {tote.itemCount} items
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>

      <div className="flex gap-4 mt-6">
        {canAssign && (
          <Button
            onClick={() => setShowAssignModal(true)}
            disabled={assignMutation.isPending}
            >
            <User className="h-4 w-4 mr-2" />
            Assign Walliner
          </Button>
        )}
        {canSort && (
          <Button
            onClick={() => setShowAssignModal(true)}
            disabled={sortMutation.isPending || sortedCount === 0}
            >
            <Layers className="h-4 w-4 mr-2" />
            Sort Items
          </Button>
        )}
        {canComplete && (
          <Button
            onClick={() => completeMutation.mutate()}
            disabled={completeMutation.isPending}
            >
            <CheckCircle2 className="h-4 w-4 mr-2" />
            Complete Task
          </Button>
        )}
        {task.status !== 'pending' && (
          <Button
            variant="danger"
            onClick={() => {
              const reason = prompt('Enter cancellation reason:');
              if (reason && reason.trim()) {
                cancelMutation.mutate(reason.trim());
              }
            }}
            disabled={['completed', 'cancelled'].includes(task.status)}
            className="text-sm"
          >
            <XCircle className="h-4 w-4 mr-2" />
            Cancel Task
          </Button>
        )}
      </div>

      {showAssignModal && (
        <AssignWallinerModal
          taskId={taskId!}
          isOpen={showAssignModal}
          onClose={() => setShowAssignModal(false)}
          onAssign={(wallinerId: string, station?: string) => {
            assignMutation.mutate({ wallinerId, station });
            setSelectedWallinerId(wallinerId);
            setSelectedStation(station || '');
          }}
        />
      )}
    </div>
  );
}

interface AssignWallinerModalProps {
  taskId: string;
  isOpen: boolean;
  onClose: () => void;
  onAssign: (wallinerId: string, station?: string) => void;
}

function AssignWallinerModal({ taskId, isOpen, onClose, onAssign }: AssignWallinerModalProps) {
  const [selectedWorkerId, setSelectedWorkerId] = useState<string>('');
  const [selectedStationId, setSelectedStationId] = useState<string>('');

  const availableWorkers = [
    { id: 'WALLINER-001', name: 'John Doe', currentTask: null, tasksToday: 15, zone: 'zone-a' },
    { id: 'WALLINER-002', name: 'Jane Smith', currentTask: 'WALL-abc123', tasksToday: 12, zone: 'zone-a' },
    { id: 'WALLINER-003', name: 'Bob Johnson', currentTask: null, tasksToday: 8, zone: 'zone-a' },
    { id: 'WALLINER-004', name: 'Alice Williams', currentTask: 'WT-xyz456', tasksToday: 20, zone: 'zone-c' },
  ];

  const availableStations = ['PUT-STATION-001', 'PUT-STATION-002'];

  const handleAssign = () => {
    if (selectedWorkerId) {
      onAssign(selectedWorkerId, selectedStationId || undefined);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold mb-4">Assign Walliner</h2>

        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Available Workers:
            </label>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {availableWorkers.map((worker) => (
                <button
                  key={worker.id}
                  onClick={() => setSelectedWorkerId(worker.id)}
                  className={`p-3 border-2 rounded-lg transition-all ${
                    selectedWorkerId === worker.id
                      ? 'border-primary-600 bg-primary-50'
                      : 'border-gray-200 hover:border-primary-300 hover:bg-gray-50'
                  }`}
                >
                  <div className="text-center">
                    <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-700 mx-auto mb-2">
                      <span className="font-semibold">{worker.name.charAt(0)}</span>
                    </div>
                    <div className="text-sm font-medium text-gray-900">{worker.name}</div>
                    <div className="text-xs text-gray-500">{worker.id}</div>
                    <div className="text-xs text-gray-500 mt-1">{worker.tasksToday} tasks today</div>
                    {selectedWorkerId === worker.id && (
                      <CheckCircle className="h-5 w-5 text-green-600 mx-auto mt-2" />
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Station (Optional):
            </label>
            <select
              value={selectedStationId}
              onChange={(e) => setSelectedStationId(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2"
            >
              <option value="">Select Station</option>
              {availableStations.map((stationId) => (
                <option key={stationId} value={stationId}>{stationId}</option>
              ))}
            </select>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t">
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              onClick={handleAssign}
              disabled={!selectedWorkerId}
            >
              <User className="h-4 w-4 mr-2" />
              Assign Walliner
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
