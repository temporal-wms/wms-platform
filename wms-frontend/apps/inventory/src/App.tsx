import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Package, MapPin, ArrowUpDown } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function InventoryDashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Inventory Management</h1>
        <p className="text-gray-500">Monitor stock levels, locations, and adjustments</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Total SKUs"
          value="2,847"
          icon={<Package className="h-5 w-5" />}
          trend={{ value: 12, direction: 'up' }}
        />
        <MetricCard
          title="Warehouse Locations"
          value="1,256"
          icon={<MapPin className="h-5 w-5" />}
        />
        <MetricCard
          title="Today's Adjustments"
          value="47"
          icon={<ArrowUpDown className="h-5 w-5" />}
          trend={{ value: 8, direction: 'down' }}
        />
      </div>

      <Card>
        <CardHeader title="Recent Inventory Activity" subtitle="Last 24 hours" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            Inventory activity will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<InventoryDashboard />} />
    </Routes>
  );
}
