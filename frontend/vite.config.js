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
    minify: 'esbuild',           // 使用 esbuild 压缩（更快，Vite 内置）
    chunkSizeWarningLimit: 1500, // chunk 大小警告阈值 (KB)
    
    // Rollup 打包配置
    rollupOptions: {
      output: {
        // 手动分包，优化加载性能
        manualChunks(id) {
          // node_modules 中的包按目录分组
          if (id.includes('node_modules')) {
            // Element Plus 相关（包括图标和依赖）
            if (id.includes('element-plus') || id.includes('@element-plus')) {
              return 'element-plus'
            }
            // Vue 核心生态
            if (id.includes('vue') || id.includes('pinia') || id.includes('@vue')) {
              return 'vue-vendor'
            }
            // ECharts 图表库
            if (id.includes('echarts') || id.includes('zrender')) {
              return 'echarts'
            }
            // 其他第三方库
            return 'vendor'
          }
        }
      }
    }
  },
  
  // 开发服务器配置
  server: {
    port: 5173,  // 开发服务器端口
    
    // 代理配置 - 转发请求到后端服务
    proxy: {
      // API 请求代理到后端 8080 端口
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      // WebSocket 连接代理
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  }
})