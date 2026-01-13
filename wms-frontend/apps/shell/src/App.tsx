import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { MainLayout, NavItem } from '@wms/ui';
import {
  LayoutDashboard,
  Package,
  Layers,
  MapPin,
  Box,
  Truck,
  Warehouse,
  Users,
  ArrowDownCircle,
  MoveDown,
  Route as RouteIcon,
  Network,
  Settings,
  Merge,
} from 'lucide-react';
import { RemoteApp } from './components/RemoteApp';
import { ErrorBoundary } from './components/ErrorBoundary';
import { LocalDashboard } from './pages/Dashboard';
import { getEnabledRemotes, RemoteNavIcon } from './remotes/manifest';

const enabledRemotes = getEnabledRemotes(import.meta.env.VITE_ENABLED_REMOTES);

const navIconMap: Record<RemoteNavIcon, React.ReactNode> = {
  dashboard: <LayoutDashboard className="h-5 w-5" />,
  package: <Package className="h-5 w-5" />,
  layers: <Layers className="h-5 w-5" />,
  mapPin: <MapPin className="h-5 w-5" />,
  box: <Box className="h-5 w-5" />,
  truck: <Truck className="h-5 w-5" />,
  warehouse: <Warehouse className="h-5 w-5" />,
  users: <Users className="h-5 w-5" />,
  arrowDown: <ArrowDownCircle className="h-5 w-5" />,
  moveDown: <MoveDown className="h-5 w-5" />,
  route: <RouteIcon className="h-5 w-5" />,
  network: <Network className="h-5 w-5" />,
  settings: <Settings className="h-5 w-5" />,
  merge: <Merge className="h-5 w-5" />,
};

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: navIconMap.dashboard },
  ...enabledRemotes.map((remote) => ({
    label: remote.displayName,
    path: remote.basePath,
    icon: navIconMap[remote.navIcon] || navIconMap.package,
  })),
];

function App() {
  return (
    <MainLayout navItems={navItems}>
      <ErrorBoundary>
        <Routes>
          {/* Dashboard - local component or remote */}
          <Route path="/" element={<LocalDashboard />} />

          {/* Remote Microfrontends */}
          {enabledRemotes.map((remote) => (
            <Route key={remote.name} path={remote.routePath} element={<RemoteApp remote={remote} />} />
          ))}

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
