import React from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { stowClient } from '@wms/api-client';
import type { PutawayTask, StorageStrategy, ItemConstraints } from '@wms/types';
import { Card, CardHeader, CardContent, Badge, Button, PageLoading, EmptyState } from '@wms/ui';
import { ArrowLeft, Box, CheckCircle, Clock, Package, MapPin, User, Wrench, Snowflake } from 'lucide-react';

export function StowTaskDetails() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['stow-task', taskId],
    queryFn: () => stowClient.getTask(taskId!),
  });

  const startMutation = useMutation({
    mutationFn: () => stowClient.startStow(taskId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stow-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['stow-tasks'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => stowClient.completeTask(taskId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stow-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['stow-tasks'] });
      navigate('/stow');
    },
  });

  if (isLoading) return <PageLoading message="Loading task details..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading task: {error?.message}</div>;

  const task = data;
  if (!task) {
    return (
      <EmptyState
        icon={<Package className="h-12 w-12 text-gray-400" />}
        title="Task not found"
        description="The requested task could not be found."
        action={{ label: 'Back to Tasks', onClick: () => navigate('/stow') }}
      />
    );
  }

  const constraints = task.constraints || {};
  const progress = task.quantity > 0 ? ((task.stowedQuantity || 0) / task.quantity) * 100 : 0;

  const constraintBadges = [];
  if (constraints.hazmat) {
    constraintBadges.push(<Badge key="hazmat" variant="error">Hazmat</Badge>);
  }
  if (constraints.coldChain) {
    constraintBadges.push(<Badge key="coldchain" variant="info">Cold Chain</Badge>);
  }
  if (constraints.oversized) {
    constraintBadges.push(<Badge key="oversized" variant="warning">Oversized</Badge>);
  }
  if (constraints.fragile) {
    constraintBadges.push(<Badge key="fragile" variant="neutral">Fragile</Badge>);
  }
  if (constraints.highValue) {
    constraintBadges.push(<Badge key="highvalue" variant="success">High Value</Badge>);
  }

  return (
    <div className="space-y-6">
      <Link to="/stow" className="flex items-center text-gray-600 hover:text-gray-900 mb-4">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Tasks
      </Link>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card className="lg:col-span-2">
          <CardHeader title="Status" />
          <CardContent>
            <div className="flex items-center gap-2">
              {task.status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
              {task.status === 'in_progress' && <Clock className="h-5 w-5 text-blue-600" />}
              {task.status === 'assigned' && <Package className="h-5 w-5 text-purple-600" />}
              {task.status === 'pending' && <Package className="h-5 w-5 text-gray-600" />}
              <span className={`font-semibold capitalize`}>{task.status.replace('_', ' ')}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Task Progress" />
          <CardContent>
            <div className="mb-4">
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

            <div className="grid grid-cols-3 gap-4">
              <div className="text-center p-4 bg-blue-50 rounded-lg">
                <div className="text-2xl font-bold text-blue-600">{task.stowedQuantity}</div>
                <div className="text-sm text-gray-600">Stowed</div>
              </div>
              <div className="text-center p-4 bg-yellow-50 rounded-lg">
                <div className="text-2xl font-bold text-yellow-600">{task.quantity - (task.stowedQuantity || 0)}</div>
                <div className="text-sm text-gray-600">Remaining</div>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="text-2xl font-bold text-gray-600">{task.quantity}</div>
                <div className="text-sm text-gray-600">Total</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="lg:col-span-3">
        <CardHeader title="Constraints" />
        <CardContent>
          <div className="space-y-2">
            <p className="text-sm text-gray-600 mb-3">
              {constraintBadges.length > 0 
                ? 'This item has special storage requirements:'
                : 'No special constraints'}
              </p>
            <div className="flex flex-wrap gap-2">
              {constraintBadges}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Item Information" />
        <CardContent>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-500">SKU:</span>
              <span className="font-mono font-semibold">{task.sku}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Product:</span>
              <span className="font-medium">{task.productName}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Quantity:</span>
              <span className="font-semibold">{task.quantity}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Strategy:</span>
              <Badge>{task.strategy.replace('_', ' ')}</Badge>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Source Tote:</span>
              <span className="font-medium">{task.sourceToteId || '-'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Target Location:</span>
              <span className="font-mono">{task.targetLocationId || '-'}</span>
            </div>
            {task.targetLocation && (
              <div className="mt-4 pt-4 border-t border-gray-200">
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4 text-gray-600" />
                  <span className="text-sm text-gray-600">
                    Zone: {task.targetLocation.zone} | Aisle: {task.targetLocation.aisle} | Bay: {task.targetLocation.bay} | Level: {task.targetLocation.level}
                  </span>
                </div>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-gray-500">Priority:</span>
              <Badge variant="neutral">P{task.priority}</Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Assignment" />
        <CardContent>
          <div className="space-y-3">
            {task.assignedWorkerId ? (
              <div className="flex items-center gap-3">
                <User className="h-5 w-5 text-gray-600" />
                <div>
                  <div className="font-medium text-gray-900">{task.assignedWorkerId}</div>
                  <div className="text-sm text-gray-600">Assigned worker</div>
                </div>
              </div>
            ) : (
              <div className="text-center py-8">
                <Package className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                <p className="text-gray-600 mb-4">Task not yet assigned to a worker</p>
                <p className="text-sm text-gray-500">
                  Assign this task to start the stow process
                </p>
              </div>
            )}
            <div className="flex items-center gap-3 mt-4">
              <Link to={`/stow/${taskId}/assign`}>
                <Button>
                  <User className="h-4 w-4 mr-2" />
                  Assign Worker
                </Button>
              </Link>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="flex gap-4">
        {task.status === 'pending' && (
          <Button onClick={() => startMutation.mutate()}>
            <Box className="h-4 w-4 mr-2" />
            Start Stow
          </Button>
        )}
        {task.status === 'assigned' && (
          <Button onClick={() => startMutation.mutate()}>
            <Box className="h-4 w-4 mr-2" />
            Start Stow
          </Button>
        )}
        {task.status === 'in_progress' && (
          <Button
            onClick={() => completeMutation.mutate()}
            disabled={progress < 100}
            variant="outline"
          >
            <Package className="h-4 w-4 mr-2" />
            Complete Task
          </Button>
        )}
        <Link to="/stow">
          <Button variant="outline">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Tasks
          </Button>
        </Link>
      </div>
    </div>
  );
}
