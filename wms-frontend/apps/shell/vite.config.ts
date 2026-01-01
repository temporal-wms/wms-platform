import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';
import path from 'path';

// For K8s deployment, use relative paths (nginx proxy)
// For local dev, use localhost URLs
const isK8s = process.env.VITE_K8S_DEPLOY === 'true';

const remoteUrls = {
  orders: isK8s ? '/remotes/orders' : (process.env.VITE_ORDERS_URL || 'http://localhost:3001'),
  waves: isK8s ? '/remotes/waves' : (process.env.VITE_WAVES_URL || 'http://localhost:3002'),
  inventory: isK8s ? '/remotes/inventory' : (process.env.VITE_INVENTORY_URL || 'http://localhost:3003'),
  picking: isK8s ? '/remotes/picking' : (process.env.VITE_PICKING_URL || 'http://localhost:3004'),
  packing: isK8s ? '/remotes/packing' : (process.env.VITE_PACKING_URL || 'http://localhost:3005'),
  shipping: isK8s ? '/remotes/shipping' : (process.env.VITE_SHIPPING_URL || 'http://localhost:3006'),
  labor: isK8s ? '/remotes/labor' : (process.env.VITE_LABOR_URL || 'http://localhost:3007'),
  dashboard: isK8s ? '/remotes/dashboard' : (process.env.VITE_DASHBOARD_URL || 'http://localhost:3008'),
};

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'shell',
      remotes: {
        orders: `${remoteUrls.orders}/assets/remoteEntry.js`,
        waves: `${remoteUrls.waves}/assets/remoteEntry.js`,
        inventory: `${remoteUrls.inventory}/assets/remoteEntry.js`,
        picking: `${remoteUrls.picking}/assets/remoteEntry.js`,
        packing: `${remoteUrls.packing}/assets/remoteEntry.js`,
        shipping: `${remoteUrls.shipping}/assets/remoteEntry.js`,
        labor: `${remoteUrls.labor}/assets/remoteEntry.js`,
        dashboard: `${remoteUrls.dashboard}/assets/remoteEntry.js`,
      },
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
