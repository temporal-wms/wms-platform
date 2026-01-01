import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Users, Calendar, Award } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function LaborDashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Labor Management</h1>
        <p className="text-gray-500">Workers, shifts, and task assignments</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Active Workers"
          value="45"
          icon={<Users className="h-5 w-5" />}
        />
        <MetricCard
          title="Today's Shifts"
          value="3"
          icon={<Calendar className="h-5 w-5" />}
        />
        <MetricCard
          title="Productivity"
          value="94%"
          icon={<Award className="h-5 w-5" />}
          trend={{ value: 3, direction: 'up' }}
        />
      </div>

      <Card>
        <CardHeader title="Worker Status" subtitle="Current shift" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            Worker assignments will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<LaborDashboard />} />
    </Routes>
  );
}
