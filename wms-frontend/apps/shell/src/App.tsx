import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { MainLayout } from '@wms/ui';
import { RemoteApp } from './components/RemoteApp';
import { ErrorBoundary } from './components/ErrorBoundary';
import { LocalDashboard } from './pages/Dashboard';

function App() {
  return (
    <MainLayout>
      <ErrorBoundary>
        <Routes>
          {/* Dashboard - local component or remote */}
          <Route path="/" element={<LocalDashboard />} />

          {/* Remote Microfrontends */}
          <Route path="/orders/*" element={<RemoteApp name="orders" />} />
          <Route path="/waves/*" element={<RemoteApp name="waves" />} />
          <Route path="/picking/*" element={<RemoteApp name="picking" />} />
          <Route path="/packing/*" element={<RemoteApp name="packing" />} />
          <Route path="/shipping/*" element={<RemoteApp name="shipping" />} />
          <Route path="/inventory/*" element={<RemoteApp name="inventory" />} />
          <Route path="/labor/*" element={<RemoteApp name="labor" />} />

          {/* 404 */}
          <Route
            path="*"
            element={
              <div className="flex flex-col items-center justify-center min-h-[400px]">
                <h1 className="text-4xl font-bold text-gray-900 mb-2">404</h1>
                <p className="text-gray-500">Page not found</p>
              </div>
            }
          />
        </Routes>
      </ErrorBoundary>
    </MainLayout>
  );
}

export default App;
