import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { facilityClient } from '@wms/api-client';

function FacilityApp() {
  return (
    <div className="facility">
      <Routes>
        <Route path="/" element={<StationList />} />
        <Route path="/new" element={<CreateStation />} />
        <Route path="/:stationId" element={<StationDetails />} />
        <Route path="/find-capable" element={<FindCapableStations />} />
      </Routes>
    </div>
  );
}

function StationList() {
  const { data, isLoading } = useQuery({
    queryKey: ['stations'],
    queryFn: () => facilityClient.getStations(),
  });

  if (isLoading) return <div>Loading stations...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Facility Stations</h1>
      <div className="text-gray-500">
        Station list will be rendered here
      </div>
    </div>
  );
}

function CreateStation() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Create Station</h1>
      <div className="text-gray-500">
        Station creation form will be rendered here
      </div>
    </div>
  );
}

function StationDetails() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Station Details</h1>
      <div className="text-gray-500">
        Station details will be rendered here
      </div>
    </div>
  );
}

function FindCapableStations() {
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Find Capable Stations</h1>
      <div className="text-gray-500">
        Find capable stations will be rendered here
      </div>
    </div>
  );
}

export default FacilityApp;
