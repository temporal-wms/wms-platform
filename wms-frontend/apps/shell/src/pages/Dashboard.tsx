import React, { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Package,
  Layers,
  MapPin,
  Truck,
  Users,
  AlertTriangle,
  CheckCircle,
  Clock,
  TrendingUp,
} from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard, MetricGrid, Badge } from '@wms/ui';
import { dashboardClient } from '@wms/api-client';
import { formatNumber, formatPercentage, formatRelativeTime } from '@wms/utils';
import type { DashboardMetrics } from '@wms/types';

// Mock data for initial development
const mockMetrics: DashboardMetrics = {
  orders: {
    total: 1234,
    pending: 45,
    inProgress: 128,
    completed: 1048,
    failed: 8,
    dlq: 5,
  },
  waves: {
    active: 12,
    completed: 89,
    ordersInWaves: 356,
  },
  picking: {
    activeTasks: 24,
    completedToday: 412,
    itemsPicked: 2847,
    averageTime: 180,
  },
  packing: {
    activeTasks: 18,
    completedToday: 389,
    packagesCreated: 512,
  },
  shipping: {
    pending: 67,
    shippedToday: 298,
    inTransit: 1456,
  },
  labor: {
    activeWorkers: 34,
    totalWorkers: 48,
    utilizationRate: 78.5,
  },
};

export function LocalDashboard() {
  // Try to fetch real data, fallback to mock
  const { data: metrics = mockMetrics, isLoading } = useQuery({
    queryKey: ['dashboard-metrics'],
    queryFn: dashboardClient.getMetrics,
    refetchInterval: 30000, // Refresh every 30 seconds
    retry: false,
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
          <p className="text-gray-500">Real-time overview of warehouse operations</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Clock className="h-4 w-4" />
          Last updated: {formatRelativeTime(new Date())}
        </div>
      </div>

      {/* Key Metrics */}
      <MetricGrid columns={5}>
        <MetricCard
          title="Active Orders"
          value={formatNumber(metrics.orders.inProgress)}
          subtitle={`${metrics.orders.pending} pending`}
          icon={<Package className="h-6 w-6" />}
          trend={{ value: 12, direction: 'up', label: 'vs yesterday' }}
        />
        <MetricCard
          title="Active Waves"
          value={metrics.waves.active}
          subtitle={`${metrics.waves.ordersInWaves} orders in waves`}
          icon={<Layers className="h-6 w-6" />}
        />
        <MetricCard
          title="Picks Today"
          value={formatNumber(metrics.picking.completedToday)}
          subtitle={`${metrics.picking.activeTasks} in progress`}
          icon={<MapPin className="h-6 w-6" />}
          trend={{ value: 8, direction: 'up' }}
        />
        <MetricCard
          title="Shipped Today"
          value={formatNumber(metrics.shipping.shippedToday)}
          subtitle={`${metrics.shipping.pending} pending`}
          icon={<Truck className="h-6 w-6" />}
        />
        <MetricCard
          title="Worker Utilization"
          value={formatPercentage(metrics.labor.utilizationRate)}
          subtitle={`${metrics.labor.activeWorkers}/${metrics.labor.totalWorkers} active`}
          icon={<Users className="h-6 w-6" />}
          variant={metrics.labor.utilizationRate > 80 ? 'success' : 'warning'}
        />
      </MetricGrid>

      {/* Status Cards */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {/* Order Status */}
        <Card>
          <CardHeader title="Order Status" subtitle="Current order distribution" />
          <CardContent>
            <div className="space-y-3">
              <StatusRow
                label="Completed"
                value={metrics.orders.completed}
                total={metrics.orders.total}
                color="bg-success-500"
              />
              <StatusRow
                label="In Progress"
                value={metrics.orders.inProgress}
                total={metrics.orders.total}
                color="bg-primary-500"
              />
              <StatusRow
                label="Pending"
                value={metrics.orders.pending}
                total={metrics.orders.total}
                color="bg-warning-500"
              />
              <StatusRow
                label="Failed/DLQ"
                value={metrics.orders.failed + metrics.orders.dlq}
                total={metrics.orders.total}
                color="bg-error-500"
              />
            </div>
          </CardContent>
        </Card>

        {/* Picking Performance */}
        <Card>
          <CardHeader title="Picking Performance" subtitle="Today's picking metrics" />
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-900">
                  {formatNumber(metrics.picking.itemsPicked)}
                </p>
                <p className="text-sm text-gray-500">Items Picked</p>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-900">
                  {Math.floor(metrics.picking.averageTime / 60)}m {metrics.picking.averageTime % 60}s
                </p>
                <p className="text-sm text-gray-500">Avg Pick Time</p>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-gray-900">
                  {metrics.picking.activeTasks}
                </p>
                <p className="text-sm text-gray-500">Active Tasks</p>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <p className="text-2xl font-bold text-primary-600">
                  {metrics.picking.completedToday}
                </p>
                <p className="text-sm text-gray-500">Completed</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Alerts */}
        <Card>
          <CardHeader
            title="Alerts"
            subtitle="Issues requiring attention"
            action={
              <Badge variant="error" size="sm">
                {metrics.orders.failed + metrics.orders.dlq}
              </Badge>
            }
          />
          <CardContent>
            <div className="space-y-3">
              {metrics.orders.dlq > 0 && (
                <AlertItem
                  type="error"
                  title={`${metrics.orders.dlq} orders in DLQ`}
                  description="Orders requiring manual intervention"
                />
              )}
              {metrics.orders.failed > 0 && (
                <AlertItem
                  type="warning"
                  title={`${metrics.orders.failed} failed orders`}
                  description="Review and retry or cancel"
                />
              )}
              {metrics.labor.utilizationRate < 60 && (
                <AlertItem
                  type="warning"
                  title="Low worker utilization"
                  description={`Only ${formatPercentage(metrics.labor.utilizationRate)} utilization`}
                />
              )}
              {metrics.orders.failed === 0 && metrics.orders.dlq === 0 && metrics.labor.utilizationRate >= 60 && (
                <div className="flex items-center gap-3 p-3 bg-success-50 rounded-lg">
                  <CheckCircle className="h-5 w-5 text-success-500" />
                  <div>
                    <p className="font-medium text-success-700">All systems operational</p>
                    <p className="text-sm text-success-600">No issues detected</p>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Activity */}
      <Card>
        <CardHeader
          title="Recent Activity"
          subtitle="Latest operations across the warehouse"
          action={
            <button className="text-sm text-primary-600 hover:text-primary-700">
              View All
            </button>
          }
        />
        <CardContent>
          <div className="space-y-4">
            <ActivityItem
              icon={<Package className="h-5 w-5" />}
              title="Order #ORD-2024-1234 completed"
              description="All items shipped successfully"
              time="2 min ago"
            />
            <ActivityItem
              icon={<Layers className="h-5 w-5" />}
              title="Wave WV-2024-089 released"
              description="45 orders added to picking queue"
              time="5 min ago"
            />
            <ActivityItem
              icon={<MapPin className="h-5 w-5" />}
              title="Pick task completed"
              description="Worker #W-034 completed 12 items"
              time="8 min ago"
            />
            <ActivityItem
              icon={<Truck className="h-5 w-5" />}
              title="Shipment dispatched"
              description="Carrier: UPS, Tracking: 1Z999AA10123456784"
              time="12 min ago"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Helper components
interface StatusRowProps {
  label: string;
  value: number;
  total: number;
  color: string;
}

function StatusRow({ label, value, total, color }: StatusRowProps) {
  const percentage = (value / total) * 100;

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1">
        <div className="flex justify-between text-sm mb-1">
          <span className="text-gray-600">{label}</span>
          <span className="font-medium text-gray-900">{formatNumber(value)}</span>
        </div>
        <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full ${color}`}
            style={{ width: `${percentage}%` }}
          />
        </div>
      </div>
    </div>
  );
}

interface AlertItemProps {
  type: 'error' | 'warning' | 'info';
  title: string;
  description: string;
}

function AlertItem({ type, title, description }: AlertItemProps) {
  const styles = {
    error: 'bg-error-50 text-error-700',
    warning: 'bg-warning-50 text-warning-700',
    info: 'bg-info-50 text-info-700',
  };

  return (
    <div className={`flex items-start gap-3 p-3 rounded-lg ${styles[type]}`}>
      <AlertTriangle className="h-5 w-5 flex-shrink-0 mt-0.5" />
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-sm opacity-80">{description}</p>
      </div>
    </div>
  );
}

interface ActivityItemProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  time: string;
}

function ActivityItem({ icon, title, description, time }: ActivityItemProps) {
  return (
    <div className="flex items-start gap-3">
      <div className="p-2 bg-gray-100 rounded-lg text-gray-500">{icon}</div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-gray-900">{title}</p>
        <p className="text-sm text-gray-500 truncate">{description}</p>
      </div>
      <span className="text-sm text-gray-400 whitespace-nowrap">{time}</span>
    </div>
  );
}
