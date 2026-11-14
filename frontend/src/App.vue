<template>
  <el-config-provider :locale="locale === 'zh-CN' ? zhCN : enUS">
  <div class="common-layout">
    <el-container class="layout-container">
      <!-- 侧边栏 -->
      <el-aside width="200px" class="sidebar">
        <div class="logo">
          <div class="logo-box">Q</div>
          <span>qwq Ops</span>
        </div>
        <el-menu
          active-text-color="#409EFF"
          background-color="#10141d"
          text-color="#a1a7b7"
          :default-active="$route.path"
          class="el-menu-vertical"
          router
        >
          <el-menu-item index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <span>{{ t('menu.dashboard') }}</span>
          </el-menu-item>
          <el-menu-item index="/appstore">
            <el-icon><ShoppingCart /></el-icon>
            <span>{{ t('menu.appstore') }}</span>
          </el-menu-item>
          <el-menu-item index="/containers">
            <el-icon><Box /></el-icon>
            <span>{{ t('menu.containers') }}</span>
          </el-menu-item>
          <el-menu-item index="/websites">
            <el-icon><Monitor /></el-icon>
            <span>{{ t('menu.websites') }}</span>
          </el-menu-item>
          <el-menu-item index="/databases">
            <el-icon><Coin /></el-icon>
            <span>{{ t('menu.databases') }}</span>
          </el-menu-item>
          <el-menu-item index="/monitoring">
            <el-icon><DataLine /></el-icon>
            <span>{{ t('menu.monitoring') }}</span>
          </el-menu-item>
          <el-menu-item index="/users">
            <el-icon><User /></el-icon>
            <span>{{ t('menu.users') }}</span>
          </el-menu-item>
          <el-menu-item index="/terminal">
            <el-icon><ChatLineSquare /></el-icon>
            <span>{{ t('menu.terminal') }}</span>
          </el-menu-item>
          <el-menu-item index="/files">
            <el-icon><Folder /></el-icon>
            <span>{{ t('menu.files') }}</span>
          </el-menu-item>
          <el-menu-item index="/logs">
            <el-icon><Document /></el-icon>
            <span>{{ t('menu.logs') }}</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- 主内容区 -->
      <el-container>
        <el-header class="header">
          <div class="breadcrumb">
            <span class="current-page">{{ pageTitle }}</span>
          </div>
          <div class="header-actions">
            <el-dropdown @command="handleLocaleChange">
              <el-button text>
                <el-icon><Switch /></el-icon>
                {{ currentLocale }}
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="zh-CN">中文</el-dropdown-item>
                  <el-dropdown-item command="en-US">English</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button type="primary" plain size="small" @click="triggerPatrol" :loading="patrolLoading">
              <el-icon class="el-icon--left"><Lightning /></el-icon>
              {{ t('console.patrol') }}
            </el-button>
          </div>
        </el-header>
        <el-main class="main-content">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
  </el-config-provider>
</template>

<script setup>
// 导入 Vue 核心功能和第三方库
import { ref, computed, watchEffect } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { errorMessage } from './lib/errors'
import zhCN from 'element-plus/es/locale/lang/zh-cn'
import enUS from 'element-plus/es/locale/lang/en'

// 获取路由和国际化实例
const route = useRoute()
const { t, locale } = useI18n()

// 巡检按钮加载状态
const patrolLoading = ref(false)

// 当前语言显示文本
const currentLocale = computed(() => locale.value === 'zh-CN' ? '中文' : 'English')

// 根据当前路由动态生成页面标题
const pageTitle = computed(() => {
  if (route.name === 'NotFound') return t('console.notFound')
  const path = route.path.substring(1) || 'dashboard'
  return t(`menu.${path}`)
})

watchEffect(() => {
  document.documentElement.lang = locale.value
  document.title = `${pageTitle.value} · qwq Ops`
})

// 切换语言
const handleLocaleChange = (lang) => {
  locale.value = lang
  localStorage.setItem('locale', lang)
}

// 触发系统巡检
const triggerPatrol = async () => {
  patrolLoading.value = true
  try {
    await axios.post('/api/trigger')
    ElMessage.success(t('common.success'))
  } catch (e) {
    ElMessage.error(errorMessage(e, t('console.requestFailed')))
  } finally {
    patrolLoading.value = false
  }
}
</script>

<style>
body { margin: 0; background-color: #0b0e14; color: #e2e8f0; font-family: system-ui, 'Segoe UI', 'Microsoft YaHei', sans-serif; }

.layout-container { height: 100vh; }
.sidebar { background-color: #10141d; border-right: 1px solid #2c3038; }
.logo { height: 60px; display: flex; align-items: center; padding-left: 20px; font-size: 18px; font-weight: 600; color: #fff; border-bottom: 1px solid #2c3038; gap: 10px; }
.logo-box { width: 32px; height: 32px; background: #409EFF; border-radius: 6px; color: white; display: flex; align-items: center; justify-content: center; font-weight: bold; }
.el-menu { border-right: none !important; }
.el-menu-item.is-active { background-color: #1d2129 !important; border-right: 3px solid #409EFF; }

.header { background-color: #10141d; border-bottom: 1px solid #2c3038; display: flex; align-items: center; justify-content: space-between; color: #fff; padding: 0 20px; height: 60px; }
.current-page { font-size: 16px; font-weight: 500; }
.main-content { background-color: #0b0e14; padding: 20px; overflow-y: auto; }
.header-actions { display: flex; align-items: center; gap: 12px; }
.el-container, .main-content { min-width: 0; }
@media (max-width: 700px) {
  .sidebar { width: 64px !important; }
  .sidebar .el-menu-item span, .logo > span { display: none; }
  .logo { padding-left: 16px; }
  .header { padding: 0 12px; }
  .main-content { padding: 12px; }
}
</style>
