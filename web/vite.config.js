import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: true,
    allowedHosts: true,
    port: 5273,
    proxy: {
      '/api': 'http://127.0.0.1:8180',
    },
  },
  build: {
    outDir: 'dist',
  },
})
