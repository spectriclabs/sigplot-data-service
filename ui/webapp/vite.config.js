import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: '/sigplot/ui/',
  build: {
    outDir: 'dist',
  },
  server: {
    proxy: {
      '/sds': 'http://localhost:5055',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
});
