import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { ListChecks, Route as RouteIcon, Clock } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function PickingDashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Picking Operations</h1>
        <p className="text-gray-500">Manage pick tasks and route optimization</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Active Pick Tasks"
          value="156"
          icon={<ListChecks className="h-5 w-5" />}
          trend={{ value: 23, direction: 'up' }}
        />
        <MetricCard
          title="Optimized Routes"
          value="42"
          icon={<RouteIcon className="h-5 w-5" />}
        />
        <MetricCard
          title="Avg Pick Time"
          value="2.4m"
          icon={<Clock className="h-5 w-5" />}
          trend={{ value: 5, direction: 'up' }}
        />
      </div>

      <Card>
        <CardHeader title="Active Pick Tasks" subtitle="Assigned to workers" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            Pick tasks will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<PickingDashboard />} />
    </Routes>
  );
}
