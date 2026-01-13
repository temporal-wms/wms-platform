import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { consolidationClient } from '@wms/api-client';

function ConsolidationApp() {
  return (
    <div className="consolidation">
      <Routes>
        <Route path="/" element={<ConsolidationList />} />
        <Route path="/:consolidationId" element={<ConsolidationDetails />} />
        <Route path="/order/:orderId" element={<OrderProgress />} />
      </Routes>
    </div>
  );
}

function ConsolidationList() {
  const { data, isLoading } = useQuery({
    queryKey: ['consolidations'],
    queryFn: () => consolidationClient.getConsolidations(),
  });

  if (isLoading) return <div>Loading consolidations...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Consolidations</h1>
      <div className="text-gray-500">
        Consolidation list will be rendered here
      </div>
    </div>
  );
}

function ConsolidationDetails() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Consolidation Details</h1>
      <div className="text-gray-500">
        Consolidation details will be rendered here
      </div>
    </div>
  );
}

function OrderProgress() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Order Progress</h1>
      <div className="text-gray-500">
        Order progress will be rendered here
      </div>
    </div>
  );
}

export default ConsolidationApp;
