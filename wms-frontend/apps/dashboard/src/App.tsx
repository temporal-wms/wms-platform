import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { BarChart3, TrendingUp, AlertTriangle, Activity } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function DashboardHome() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Operations Dashboard</h1>
        <p className="text-gray-500">Real-time KPIs and warehouse metrics</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          title="Orders Today"
          value="1,247"
          icon={<BarChart3 className="h-5 w-5" />}
          trend={{ value: 18, direction: 'up' }}
        />
        <MetricCard
          title="Fulfillment Rate"
          value="96.8%"
          icon={<TrendingUp className="h-5 w-5" />}
          trend={{ value: 2.3, direction: 'up' }}
        />
        <MetricCard
          title="Active Alerts"
          value="3"
          icon={<AlertTriangle className="h-5 w-5" />}
          trend={{ value: 1, direction: 'down' }}
        />
        <MetricCard
          title="System Health"
          value="99.9%"
          icon={<Activity className="h-5 w-5" />}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader title="Order Volume" subtitle="Last 7 days" />
          <CardContent>
            <div className="h-64 flex items-center justify-center text-gray-500">
              Order volume chart will appear here
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Throughput by Zone" subtitle="Today" />
          <CardContent>
            <div className="h-64 flex items-center justify-center text-gray-500">
              Zone throughput chart will appear here
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader title="Recent Alerts" subtitle="System notifications" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            System alerts will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<DashboardHome />} />
    </Routes>
  );
}
