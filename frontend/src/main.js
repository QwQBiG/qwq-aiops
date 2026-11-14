// Vue 核心和插件导入
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import { Odometer, ShoppingCart, Box, Monitor, Coin, DataLine, User, ChatLineSquare, Folder, Document, Switch, Lightning, Refresh, Plus, HomeFilled, FolderAdd } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 应用组件和配置导入
import App from './App.vue'
import router from './router'
import i18n from './i18n'

// 创建 Vue 应用实例
const app = createApp(App)
document.documentElement.classList.add('dark')

// 全局错误处理器
// 捕获 Vue 组件中未处理的错误，防止应用崩溃
app.config.errorHandler = (err, instance, info) => {
  console.error('Vue Error:', err, info)
  ElMessage.error(i18n.global.t('console.requestFailed'))
}

// Register only the icons used by the current pages.
for (const [key, component] of Object.entries({ Odometer, ShoppingCart, Box, Monitor, Coin, DataLine, User, ChatLineSquare, Folder, Document, Switch, Lightning, Refresh, Plus, HomeFilled, FolderAdd })) {
  app.component(key, component)
}

// 安装插件
app.use(router)     // 路由
app.use(ElementPlus) // UI 组件库
app.use(i18n)       // 国际化

// 挂载应用到 DOM
app.mount('#app')
