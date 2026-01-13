import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    css: false,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: [
        'node_modules/',
        '**/node_modules/**',
        '**/dist/**',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData/**',
        '**/__tests__/**',
        '**/test/**',
        '**/*.test.{ts,tsx}',
        '**/*.spec.{ts,tsx}',
        '**/remoteEntry.js',
        '**/__federation__/**',
      ],
      all: true,
      lines: 80,
      functions: 80,
      branches: 80,
      statements: 80,
    },
    include: ['**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules', 'dist'],
  },
  resolve: {
    alias: {
      '@wms/ui': path.resolve(__dirname, './packages/ui/src'),
      '@wms/api-client': path.resolve(__dirname, './packages/api-client/src'),
      '@wms/utils': path.resolve(__dirname, './packages/utils/src'),
      '@wms/types': path.resolve(__dirname, './packages/types/src'),
      '@wms/config': path.resolve(__dirname, './packages/config/src'),
    },
  },
});
