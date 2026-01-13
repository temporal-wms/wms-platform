import type { ComponentType } from 'react';

export type FederatedModule = {
  default: ComponentType;
};

export type RemoteNavIcon =
  | 'dashboard'
  | 'package'
  | 'layers'
  | 'mapPin'
  | 'box'
  | 'truck'
  | 'warehouse'
  | 'users'
  | 'arrowDown'
  | 'moveDown'
  | 'route'
  | 'network'
  | 'settings'
  | 'merge';

export type RemoteDefinition = {
  name: string;
  displayName: string;
  basePath: string;
  routePath: string;
  devPort: number;
  k8sProxyPath: string;
  navIcon: RemoteNavIcon;
  loader: () => Promise<FederatedModule>;
};

const remoteManifest: RemoteDefinition[] = [
  {
    name: 'orders',
    displayName: 'Orders',
    basePath: '/orders',
    routePath: '/orders/*',
    devPort: 3001,
    k8sProxyPath: '/remotes/orders',
    navIcon: 'package',
    loader: () => import('orders/App') as Promise<FederatedModule>,
  },
  {
    name: 'waves',
    displayName: 'Waves',
    basePath: '/waves',
    routePath: '/waves/*',
    devPort: 3002,
    k8sProxyPath: '/remotes/waves',
    navIcon: 'layers',
    loader: () => import('waves/App') as Promise<FederatedModule>,
  },
  {
    name: 'inventory',
    displayName: 'Inventory',
    basePath: '/inventory',
    routePath: '/inventory/*',
    devPort: 3003,
    k8sProxyPath: '/remotes/inventory',
    navIcon: 'warehouse',
    loader: () => import('inventory/App') as Promise<FederatedModule>,
  },
  {
    name: 'picking',
    displayName: 'Picking',
    basePath: '/picking',
    routePath: '/picking/*',
    devPort: 3004,
    k8sProxyPath: '/remotes/picking',
    navIcon: 'mapPin',
    loader: () => import('picking/App') as Promise<FederatedModule>,
  },
  {
    name: 'packing',
    displayName: 'Packing',
    basePath: '/packing',
    routePath: '/packing/*',
    devPort: 3005,
    k8sProxyPath: '/remotes/packing',
    navIcon: 'box',
    loader: () => import('packing/App') as Promise<FederatedModule>,
  },
  {
    name: 'shipping',
    displayName: 'Shipping',
    basePath: '/shipping',
    routePath: '/shipping/*',
    devPort: 3006,
    k8sProxyPath: '/remotes/shipping',
    navIcon: 'truck',
    loader: () => import('shipping/App') as Promise<FederatedModule>,
  },
  {
    name: 'labor',
    displayName: 'Labor',
    basePath: '/labor',
    routePath: '/labor/*',
    devPort: 3007,
    k8sProxyPath: '/remotes/labor',
    navIcon: 'users',
    loader: () => import('labor/App') as Promise<FederatedModule>,
  },
  {
    name: 'dashboard',
    displayName: 'Dashboard',
    basePath: '/dashboard',
    routePath: '/dashboard/*',
    devPort: 3008,
    k8sProxyPath: '/remotes/dashboard',
    navIcon: 'dashboard',
    loader: () => import('dashboard/App') as Promise<FederatedModule>,
  },
  {
    name: 'receiving',
    displayName: 'Receiving',
    basePath: '/receiving',
    routePath: '/receiving/*',
    devPort: 3009,
    k8sProxyPath: '/remotes/receiving',
    navIcon: 'arrowDown',
    loader: () => import('receiving/App') as Promise<FederatedModule>,
  },
  {
    name: 'stow',
    displayName: 'Stow',
    basePath: '/stow',
    routePath: '/stow/*',
    devPort: 3010,
    k8sProxyPath: '/remotes/stow',
    navIcon: 'moveDown',
    loader: () => import('stow/App') as Promise<FederatedModule>,
  },
  {
    name: 'routing',
    displayName: 'Routing',
    basePath: '/routing',
    routePath: '/routing/*',
    devPort: 3011,
    k8sProxyPath: '/remotes/routing',
    navIcon: 'route',
    loader: () => import('routing/App') as Promise<FederatedModule>,
  },
  {
    name: 'walling',
    displayName: 'Walling',
    basePath: '/walling',
    routePath: '/walling/*',
    devPort: 3012,
    k8sProxyPath: '/remotes/walling',
    navIcon: 'layers',
    loader: () => import('walling/App') as Promise<FederatedModule>,
  },
  {
    name: 'consolidation',
    displayName: 'Consolidation',
    basePath: '/consolidation',
    routePath: '/consolidation/*',
    devPort: 3013,
    k8sProxyPath: '/remotes/consolidation',
    navIcon: 'merge',
    loader: () => import('consolidation/App') as Promise<FederatedModule>,
  },
  {
    name: 'sortation',
    displayName: 'Sortation',
    basePath: '/sortation',
    routePath: '/sortation/*',
    devPort: 3014,
    k8sProxyPath: '/remotes/sortation',
    navIcon: 'network',
    loader: () => import('sortation/App') as Promise<FederatedModule>,
  },
  {
    name: 'facility',
    displayName: 'Facility',
    basePath: '/facility',
    routePath: '/facility/*',
    devPort: 3015,
    k8sProxyPath: '/remotes/facility',
    navIcon: 'settings',
    loader: () => import('facility/App') as Promise<FederatedModule>,
  },
] satisfies RemoteDefinition[];

const parseEnabledList = (value?: string): string[] => {
  if (!value) return [];
  return value
    .split(',')
    .map((name) => name.trim())
    .filter(Boolean);
};

export const getEnabledRemoteNames = (value?: string): string[] => {
  const parsed = parseEnabledList(value);
  const knownNames = new Set(remoteManifest.map((remote) => remote.name));

  return parsed.length > 0
    ? parsed.filter((name) => knownNames.has(name))
    : remoteManifest.map((remote) => remote.name);
};

export const getEnabledRemotes = (value?: string): RemoteDefinition[] => {
  const enabledNames = getEnabledRemoteNames(value);
  if (enabledNames.length === 0 || enabledNames.length === remoteManifest.length) {
    return remoteManifest;
  }

  return enabledNames
    .map((name) => remoteManifest.find((remote) => remote.name === name))
    .filter((remote): remote is RemoteDefinition => Boolean(remote));
};

export const allRemotes = remoteManifest;
