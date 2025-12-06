import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8899',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:8899',
        ws: true
      }
    }
  }
})