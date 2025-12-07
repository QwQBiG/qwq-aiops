// Vue 核心和插件导入
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// 应用组件和配置导入
import App from './App.vue'
import router from './router'
import i18n from './i18n'

// 创建 Vue 应用实例
const app = createApp(App)
const pinia = createPinia()

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