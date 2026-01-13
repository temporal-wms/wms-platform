import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';
import path from 'path';
import { getEnabledRemotes } from './src/remotes/manifest';

// For K8s deployment, use relative paths (nginx proxy)
// For local dev, use localhost URLs
const isK8s = process.env.VITE_K8S_DEPLOY === 'true';
const enabledRemotes = getEnabledRemotes(process.env.VITE_ENABLED_REMOTES);

const remotesConfig = enabledRemotes.reduce<Record<string, string>>((acc, remote) => {
  const envKey = `VITE_${remote.name.toUpperCase()}_URL`;
  const overrideUrl = process.env[envKey];
  const baseUrl = isK8s ? remote.k8sProxyPath : overrideUrl || `http://localhost:${remote.devPort}`;
  acc[remote.name] = `${baseUrl}/assets/remoteEntry.js`;
  return acc;
}, {});

export default defineConfig({
  plugins: [
    react(),
      federation({
        name: 'shell',
        remotes: remotesConfig,
      shared: {
        react: { singleton: true, requiredVersion: '^18.0.0' },
        'react-dom': { singleton: true, requiredVersion: '^18.0.0' },
        'react-router-dom': { singleton: true, requiredVersion: '^6.0.0' },
        zustand: { singleton: true },
        '@tanstack/react-query': { singleton: true },
      },
    }),
  ],
  resolve: {
    alias: {
      '@wms/ui': path.resolve(__dirname, '../../packages/ui/src/index.ts'),
      '@wms/api-client': path.resolve(__dirname, '../../packages/api-client/src/index.ts'),
      '@wms/types': path.resolve(__dirname, '../../packages/types/src/index.ts'),
      '@wms/utils': path.resolve(__dirname, '../../packages/utils/src/index.ts'),
      '@wms/config': path.resolve(__dirname, '../../packages/config/src/index.ts'),
    },
  },
  build: {
    target: 'esnext',
    minify: false,
    cssCodeSplit: false,
  },
  server: {
    port: 3000,
    cors: true,
  },
});
