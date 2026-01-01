import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { BoxIcon, Scan, CheckCircle } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function PackingDashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Packing Station</h1>
        <p className="text-gray-500">Pack verification, labeling, and quality control</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Pending Packing"
          value="89"
          icon={<BoxIcon className="h-5 w-5" />}
        />
        <MetricCard
          title="Scanned Today"
          value="342"
          icon={<Scan className="h-5 w-5" />}
          trend={{ value: 15, direction: 'up' }}
        />
        <MetricCard
          title="Verified"
          value="98.5%"
          icon={<CheckCircle className="h-5 w-5" />}
          trend={{ value: 2, direction: 'up' }}
        />
      </div>

      <Card>
        <CardHeader title="Pack Queue" subtitle="Orders ready for packing" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            Packing queue will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<PackingDashboard />} />
    </Routes>
  );
}
