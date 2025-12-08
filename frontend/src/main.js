// Vue 核心和插件导入
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

// 应用组件和配置导入
import App from './App.vue'
import router from './router'
import i18n from './i18n'

// 配置 axios 全局响应拦截器
// 用于统一处理 API 请求错误，自动显示错误提示
axios.interceptors.response.use(
  response => response,
  error => {
    console.error('API Error:', error)
    // 优先使用服务器返回的错误信息，否则使用默认提示
    const message = error.response?.data?.message || error.message || '请求失败'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

// 创建 Vue 应用实例
const app = createApp(App)
const pinia = createPinia()

// 全局错误处理器
// 捕获 Vue 组件中未处理的错误，防止应用崩溃
app.config.errorHandler = (err, instance, info) => {
  console.error('Vue Error:', err, info)
  ElMessage.error('应用错误: ' + err.message)
}

// 注册所有 Element Plus 图标组件
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 安装插件
app.use(pinia)      // 状态管理
app.use(router)     // 路由
app.use(ElementPlus) // UI 组件库
app.use(i18n)       // 国际化

// 挂载应用到 DOM
app.mount('#app')