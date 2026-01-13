import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';
import path from 'path';

const isK8s = process.env.VITE_K8S_DEPLOY === 'true';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'facility',
      filename: 'remoteEntry.js',
      exposes: {
        './App': './src/App.tsx',
      },
      shared: {
        react: { singleton: true },
        'react-dom': { singleton: true },
        'react-router-dom': { singleton: true },
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
    port: 3015,
    cors: true,
  },
});
