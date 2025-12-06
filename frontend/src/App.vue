<template>
  <div class="common-layout">
    <el-container class="layout-container">
      <!-- 侧边栏 -->
      <el-aside width="200px" class="sidebar">
        <div class="logo">
          <div class="logo-box">Q</div>
          <span>qwq AIOps</span>
        </div>
        <el-menu
          active-text-color="#409EFF"
          background-color="#10141d"
          text-color="#a1a7b7"
          :default-active="activeMenu"
          class="el-menu-vertical"
          @select="handleSelect"
        >
          <el-menu-item index="dashboard">
            <el-icon><Odometer /></el-icon>
            <span>概览</span>
          </el-menu-item>
          <el-menu-item index="containers">
            <el-icon><Box /></el-icon>
            <span>容器</span>
          </el-menu-item>
          <el-menu-item index="terminal">
            <el-icon><ChatLineSquare /></el-icon>
            <span>AI 终端</span>
          </el-menu-item>
          <el-menu-item index="files">
            <el-icon><Folder /></el-icon>
            <span>文件</span>
          </el-menu-item>
          <el-menu-item index="logs">
            <el-icon><Document /></el-icon>
            <span>日志</span>
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
            <el-button type="primary" plain size="small" @click="triggerPatrol" :loading="patrolLoading">
              <el-icon class="el-icon--left"><Lightning /></el-icon>立即巡检
            </el-button>
          </div>
        </el-header>
        <el-main class="main-content">
          <KeepAlive>
            <component :is="currentComponent" />
          </KeepAlive>
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import Dashboard from './components/Dashboard.vue'
import Terminal from './components/Terminal.vue'
import Logs from './components/Logs.vue'
import Containers from './components/Containers.vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import Files from './components/Files.vue'

const activeMenu = ref('dashboard')
const patrolLoading = ref(false)

const components = {
  dashboard: Dashboard,
  terminal: Terminal,
  logs: Logs,
  files: Files, 
  containers: Containers
}

const titles = {
  dashboard: '系统概览',
  terminal: '智能运维终端',
  logs: '系统运行日志',
  files: '文件管理',
  containers: '容器管理'
}

const currentComponent = computed(() => components[activeMenu.value])
const pageTitle = computed(() => titles[activeMenu.value])

const handleSelect = (key) => {
  activeMenu.value = key
}

const triggerPatrol = async () => {
  patrolLoading.value = true
  try {
    await axios.get('/api/trigger')
    ElMessage.success('巡检指令已发送')
  } catch (e) {
    ElMessage.error('触发失败')
  } finally {
    setTimeout(() => { patrolLoading.value = false }, 2000)
  }
}
</script>

<style>
body { margin: 0; background-color: #f5f7fa; color: #1f2329; font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif; }
@media (prefers-color-scheme: dark) {
  body { background-color: #0b0e14; color: #e2e8f0; }
}

.layout-container { height: 100vh; }
.sidebar { background-color: #10141d; border-right: 1px solid #2c3038; }
.logo { height: 60px; display: flex; align-items: center; padding-left: 20px; font-size: 18px; font-weight: 600; color: #fff; border-bottom: 1px solid #2c3038; gap: 10px; }
.logo-box { width: 32px; height: 32px; background: #409EFF; border-radius: 6px; color: white; display: flex; align-items: center; justify-content: center; font-weight: bold; }
.el-menu { border-right: none !important; }
.el-menu-item.is-active { background-color: #1d2129 !important; border-right: 3px solid #409EFF; }

.header { background-color: #10141d; border-bottom: 1px solid #2c3038; display: flex; align-items: center; justify-content: space-between; color: #fff; padding: 0 20px; height: 60px; }
.current-page { font-size: 16px; font-weight: 500; }
.main-content { background-color: #0b0e14; padding: 20px; overflow-y: auto; }
</style>