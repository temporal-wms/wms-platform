import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Truck, FileText, MapPin } from 'lucide-react';
import { Card, CardHeader, CardContent, MetricCard } from '@wms/ui';

function ShippingDashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Shipping & SLAM</h1>
        <p className="text-gray-500">Shipment creation, manifests, and tracking</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Pending Shipments"
          value="67"
          icon={<Truck className="h-5 w-5" />}
        />
        <MetricCard
          title="Today's Manifests"
          value="12"
          icon={<FileText className="h-5 w-5" />}
          trend={{ value: 3, direction: 'up' }}
        />
        <MetricCard
          title="In Transit"
          value="234"
          icon={<MapPin className="h-5 w-5" />}
        />
      </div>

      <Card>
        <CardHeader title="Shipment Queue" subtitle="Ready for carrier pickup" />
        <CardContent>
          <div className="text-center py-8 text-gray-500">
            Shipments will appear here
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/*" element={<ShippingDashboard />} />
    </Routes>
  );
}
