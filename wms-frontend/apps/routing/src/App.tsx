import React, { useState } from 'react';
import { Routes, Route, useParams } from 'react-router-dom';
import { RouteList } from './components/RouteList';
import { RouteDetails } from './components/RouteDetails';
import { RouteAnalysis } from './components/RouteAnalysis';
import { RouteStopList } from './components/RouteStopList';
import { RouteControls } from './components/RouteControls';
import { SkipStopModal } from './components/SkipStopModal';
import { StopCompletionForm } from './components/StopCompletionForm';
import { RouteVisualization } from './components/RouteVisualization';
import { StrategySelector } from './components/StrategySelector';
import { PickerAssignment } from './components/PickerAssignment';

function RouteAnalysisWrapper() {
  const { routeId } = useParams<{ routeId: string }>();
  return <RouteAnalysis routeId={routeId || ''} />;
}

function RoutingApp() {
  const [showSkipModal, setShowSkipModal] = useState<{ stopNumber: number; sequence: number } | null>(null);
  const [showCompleteModal, setShowCompleteModal] = useState<{ stopNumber: number; sequence: number; locationId: string; sku: string; quantity: number; status: string } | null>(null);
  const [selectedRouteId, setSelectedRouteId] = useState<string>('');

  const closeSkipModal = () => {
    setShowSkipModal(null);
  };

  const closeCompleteModal = () => {
    setShowCompleteModal(null);
  };

  return (
    <div className="routing">
      <Routes>
        <Route path="/" element={<RouteList />} />
        <Route path="/:routeId" element={<RouteDetails />} />
        <Route path="/:routeId/analysis" element={<RouteAnalysisWrapper />} />
      </Routes>

      <SkipStopModal
        routeId={selectedRouteId}
        stopNumber={showSkipModal?.stopNumber || 0}
        isOpen={showSkipModal !== null}
        onClose={closeSkipModal}
      />

      {showCompleteModal && (
        <StopCompletionForm
          routeId={selectedRouteId}
          stopNumber={showCompleteModal.stopNumber}
          isOpen={showCompleteModal !== null}
          onClose={closeCompleteModal}
          stopData={{
            sequence: showCompleteModal.sequence,
            locationId: showCompleteModal.locationId,
            sku: showCompleteModal.sku,
            quantity: showCompleteModal.quantity,
            status: showCompleteModal.status,
          }}
        />
      )}
    </div>
  );
}

export default RoutingApp;
