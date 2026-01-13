import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { wallingClient } from '@wms/api-client';

function WallingApp() {
  return (
    <div className="walling">
      <Routes>
        <Route path="/" element={<WallingTaskList />} />
        <Route path="/:taskId" element={<WallingTaskDetails />} />
      </Routes>
    </div>
  );
}

function WallingTaskList() {
  const { data, isLoading } = useQuery({
    queryKey: ['walling-tasks'],
    queryFn: () => wallingClient.getTasks(),
  });

  if (isLoading) return <div>Loading walling tasks...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Put-Wall Tasks</h1>
      <div className="text-gray-500">
        Walling task list will be rendered here
      </div>
    </div>
  );
}

function WallingTaskDetails() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Task Details</h1>
      <div className="text-gray-500">
        Walling task details will be rendered here
      </div>
    </div>
  );
}

export default WallingApp;
