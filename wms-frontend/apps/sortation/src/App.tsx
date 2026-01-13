import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { sortationClient } from '@wms/api-client';

function SortationApp() {
  return (
    <div className="sortation">
      <Routes>
        <Route path="/" element={<BatchList />} />
        <Route path="/new" element={<CreateBatch />} />
        <Route path="/:batchId" element={<BatchDetails />} />
      </Routes>
    </div>
  );
}

function BatchList() {
  const { data, isLoading } = useQuery({
    queryKey: ['batches'],
    queryFn: () => sortationClient.getBatches(),
  });

  if (isLoading) return <div>Loading batches...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Sortation Batches</h1>
      <div className="text-gray-500">
        Batch list will be rendered here
      </div>
    </div>
  );
}

function CreateBatch() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Create Batch</h1>
      <div className="text-gray-500">
        Batch creation form will be rendered here
      </div>
    </div>
  );
}

function BatchDetails() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Batch Details</h1>
      <div className="text-gray-500">
        Batch details will be rendered here
      </div>
    </div>
  );
}

export default SortationApp;
