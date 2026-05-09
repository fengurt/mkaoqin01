import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    host: true,
    proxy: {
      '/v1': {
        target: 'http://127.0.0.1:8010',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://127.0.0.1:8010',
        changeOrigin: true,
      },
    },
  },
})
