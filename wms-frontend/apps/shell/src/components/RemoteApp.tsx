import React, { Suspense, lazy, ComponentType } from 'react';
import { PageLoading } from '@wms/ui';
import { ErrorBoundary } from './ErrorBoundary';

// Type for federated module
type FederatedModule = {
  default: ComponentType;
};

// Lazy load remote apps
const remoteApps: Record<string, React.LazyExoticComponent<ComponentType>> = {
  orders: lazy(() => import('orders/App') as Promise<FederatedModule>),
  waves: lazy(() => import('waves/App') as Promise<FederatedModule>),
  inventory: lazy(() => import('inventory/App') as Promise<FederatedModule>),
  picking: lazy(() => import('picking/App') as Promise<FederatedModule>),
  packing: lazy(() => import('packing/App') as Promise<FederatedModule>),
  shipping: lazy(() => import('shipping/App') as Promise<FederatedModule>),
  labor: lazy(() => import('labor/App') as Promise<FederatedModule>),
  dashboard: lazy(() => import('dashboard/App') as Promise<FederatedModule>),
};

export interface RemoteAppProps {
  name: keyof typeof remoteApps;
  fallback?: React.ReactNode;
}

export function RemoteApp({ name, fallback }: RemoteAppProps) {
  const RemoteComponent = remoteApps[name];

  if (!RemoteComponent) {
    return (
      <div className="p-8 text-center">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Module Not Found
        </h2>
        <p className="text-gray-500">
          The "{name}" module is not available.
        </p>
      </div>
    );
  }

  return (
    <ErrorBoundary
      fallback={(error) => (
        <RemoteAppError
          name={name}
          error={error}
          onRetry={() => window.location.reload()}
        />
      )}
    >
      <Suspense fallback={fallback || <PageLoading message={`Loading ${name}...`} />}>
        <RemoteComponent />
      </Suspense>
    </ErrorBoundary>
  );
}

interface RemoteAppErrorProps {
  name: string;
  error?: Error | null;
  onRetry: () => void;
}

function RemoteAppError({ name, error, onRetry }: RemoteAppErrorProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[400px] p-8">
      <div className="text-center">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Failed to load {name} module
        </h2>
        <p className="text-gray-500 mb-4">
          The module might be unavailable or there was a network error.
        </p>
        {error && (
          <pre className="mt-2 p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 max-w-lg overflow-auto text-left whitespace-pre-wrap">
            {error.message}
            {error.stack && (
              <details className="mt-2">
                <summary className="cursor-pointer text-red-500">Stack trace</summary>
                <pre className="mt-2 text-xs">{error.stack}</pre>
              </details>
            )}
          </pre>
        )}
        <button
          onClick={onRetry}
          className="mt-4 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
        >
          Try Again
        </button>
      </div>
    </div>
  );
}
