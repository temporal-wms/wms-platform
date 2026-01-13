import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { ShipmentList } from './components/ShipmentList';
import { ShipmentDetails } from './components/ShipmentDetails';
import { CreateShipment } from './components/CreateShipment';
import { ReceiveItemsForm } from './components/ReceiveItemsForm';
import { DiscrepancyPanel } from './components/DiscrepancyPanel';
import { ExpectedArrivalsList } from './components/ExpectedArrivalsList';
import { ProgressStepper } from './components/ProgressStepper';

function ReceivingApp() {
  return (
    <div className="receiving">
      <Routes>
        <Route path="/" element={<ShipmentList />} />
        <Route path="/new" element={<CreateShipment />} />
        <Route path="/:shipmentId" element={<ShipmentDetails />} />
        <Route path="/:shipmentId/receive" element={<ReceiveItemsForm />} />
        <Route path="/expected" element={<ExpectedArrivalsList />} />
      </Routes>
    </div>
  );
}

export default ReceivingApp;
