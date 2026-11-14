import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Vite 配置文件 - qwq AIOps 前端构建配置
// https://vitejs.dev/config/
export default defineConfig({
  // Vue 3 插件
  plugins: [vue()],
  
  // 基础路径
  base: '/',
  
  // 构建配置
  build: {
    outDir: 'dist',              // 输出目录
    assetsDir: 'assets',         // 静态资源目录
    sourcemap: false,            // 生产环境不生成 sourcemap
    minify: true,
    chunkSizeWarningLimit: 1000
  },
  
  // 开发服务器配置
  server: {
    port: 5173,  // 开发服务器端口
    
    // 代理配置 - 转发请求到后端服务
    proxy: {
      // API 请求代理到后端 8080 端口
      '/api': {
        target: 'http://localhost:8080',
        // Keep the browser's Host for same-origin request checks.
        changeOrigin: false
      },
      // WebSocket 连接代理
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  }
})
