import React, { Suspense, lazy, ComponentType } from 'react';
import { PageLoading } from '@wms/ui';
import { ErrorBoundary } from './ErrorBoundary';
import type { RemoteDefinition } from '../remotes/manifest';

type FederatedModule = {
  default: ComponentType;
};

const remoteComponentCache = new Map<string, React.LazyExoticComponent<ComponentType>>();

const getRemoteComponent = (remote: RemoteDefinition) => {
  if (!remoteComponentCache.has(remote.name)) {
    remoteComponentCache.set(remote.name, lazy(() => remote.loader()));
  }

  return remoteComponentCache.get(remote.name)!;
};

export interface RemoteAppProps {
  remote: RemoteDefinition;
  fallback?: React.ReactNode;
}

export function RemoteApp({ remote, fallback }: RemoteAppProps) {
  const RemoteComponent = getRemoteComponent(remote);

  return (
    <ErrorBoundary
      fallback={(error) => (
        <RemoteAppError
          name={remote.displayName}
          error={error}
          onRetry={() => window.location.reload()}
        />
      )}
    >
      <Suspense fallback={fallback || <PageLoading message={`Loading ${remote.displayName}...`} />}>
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
