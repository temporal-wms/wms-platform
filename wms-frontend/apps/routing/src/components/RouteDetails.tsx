import React from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { Route } from '@wms/types';
import { Card, CardHeader, CardContent, Badge, Button, PageLoading, EmptyState } from '@wms/ui';
import { ArrowLeft, CheckCircle, Clock, Pause, Play, SquarePlay, XCircle, Route as RouteIcon, MapPin, User, ChevronRight } from 'lucide-react';

export function RouteDetails() {
  const { routeId } = useParams<{ routeId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['route', routeId],
    queryFn: () => routingClient.getRoute(routeId!),
  });

  const startMutation = useMutation({
    mutationFn: () => routingClient.startRoute(routeId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const pauseMutation = useMutation({
    mutationFn: (reason?: string) => routingClient.pauseRoute(routeId!, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const resumeMutation = useMutation({
    mutationFn: () => routingClient.resumeRoute(routeId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => routingClient.completeRoute(routeId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      navigate('/routing');
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (reason?: string) => routingClient.cancelRoute(routeId!, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      navigate('/routing');
    },
  });

  if (isLoading) return <PageLoading message="Loading route details..." />;
  if (isError) return <div className="p-6 text-error-600">Error loading route: {error?.message}</div>;

  const route = data;
  if (!route) {
    return (
      <EmptyState
        icon={<RouteIcon className="h-12 w-12 text-gray-400" />}
        title="Route not found"
        description="The requested route could not be found."
        action={{ label: 'Back to Routes', onClick: () => navigate('/routing') }}
      />
    );
  }

  const statusSteps = [
    { label: 'Calculated', status: 'calculated' },
    { label: 'In Progress', status: 'in_progress' },
    { label: 'Paused', status: 'paused' },
    { label: 'Completed', status: 'completed' },
  ];
  const currentStepIndex = statusSteps.findIndex(step => step.status === route.status);

  const canStart = route.status === 'calculated';
  const canPause = route.status === 'in_progress';
  const canResume = route.status === 'paused';
  const canComplete = ['in_progress', 'paused'].includes(route.status) && route.stopsCompleted + route.stopsSkipped === route.stopsTotal;
  const canCancel = !['completed', 'cancelled'].includes(route.status);

  return (
    <div className="space-y-6">
      <Link to="/routing" className="flex items-center text-gray-600 hover:text-gray-900 mb-4">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Routes
      </Link>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader title="Status" />
          <CardContent>
            <div className="flex items-center gap-2">
              {route.status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
              {route.status === 'in_progress' && <Play className="h-5 w-5 text-blue-600" />}
              {route.status === 'paused' && <Pause className="h-5 w-5 text-yellow-600" />}
              {route.status === 'calculated' && <Clock className="h-5 w-5 text-orange-600" />}
              <span className={`font-semibold capitalize`}>{route.status.replace('_', ' ')}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Picker" />
          <CardContent>
            <div className="flex items-center gap-2">
              <User className="h-5 w-5 text-gray-600" />
              <span className="font-medium">{route.pickerId}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Strategy" />
          <CardContent>
            <Badge className="bg-purple-100 text-purple-800">
              {route.strategy.replace('_', ' ')}
            </Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Progress" />
          <CardContent>
            <div className="text-center">
              <div className="text-3xl font-bold text-primary-600">
                {route.stopsCompleted}/{route.stopsTotal}
              </div>
              <div className="text-sm text-gray-500">
                {route.stopsSkipped > 0 && `${route.stopsSkipped} skipped`}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader title="Distance" />
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">
              {route.totalDistance.toFixed(1)}m
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Est. Time" />
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">
              {route.estimatedTimeMinutes}min
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Wave ID" />
          <CardContent>
            <div className="text-xl font-semibold">
              {route.waveId}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader title="Stops Progress" />
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

          <div className="space-y-4">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-4">Sequence</th>
                  <th className="text-left py-2 px-4">Location</th>
                  <th className="text-left py-2 px-4">SKU</th>
                  <th className="text-left py-2 px-4">Qty</th>
                  <th className="text-left py-2 px-4">Status</th>
                  <th className="text-left py-2 px-4">Actions</th>
                </tr>
              </thead>
              <tbody>
                {route.stops.map((stop, index) => (
                  <tr key={index} className="border-b hover:bg-gray-50">
                    <td className="py-2 px-4 font-semibold">{stop.sequence}</td>
                    <td className="py-2 px-4">{stop.locationId}</td>
                    <td className="py-2 px-4">{stop.sku}</td>
                    <td className="py-2 px-4">{stop.quantity}</td>
                    <td className="py-2 px-4">
                      <Badge className={
                        stop.status === 'completed' ? 'bg-green-100 text-green-800' :
                        stop.status === 'skipped' ? 'bg-red-100 text-red-800' :
                        'bg-gray-100 text-gray-800'
                      }>
                        {stop.status}
                      </Badge>
                    </td>
                    <td className="py-2 px-4">
                      <Link to={`/routing/${routeId}/stop/${stop.sequence}`}>
                        <Button size="sm" variant="outline">Complete</Button>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <div className="flex gap-4">
        {canStart && (
          <Button onClick={() => startMutation.mutate()}>
            <Play className="h-4 w-4 mr-2" />
            Start Route
          </Button>
        )}
        {canPause && (
          <Button variant="outline" onClick={() => pauseMutation.mutate('')}>
            <Pause className="h-4 w-4 mr-2" />
            Pause
          </Button>
        )}
        {canResume && (
          <Button onClick={() => resumeMutation.mutate()}>
            <Play className="h-4 w-4 mr-2" />
            Resume
          </Button>
        )}
        {canComplete && (
          <Button onClick={() => completeMutation.mutate()}>
            <CheckCircle className="h-4 w-4 mr-2" />
            Complete Route
          </Button>
        )}
        {canCancel && (
          <Button variant="danger" onClick={() => cancelMutation.mutate('')}>
            <XCircle className="h-4 w-4 mr-2" />
            Cancel Route
          </Button>
        )}
      </div>
    </div>
  );
}
