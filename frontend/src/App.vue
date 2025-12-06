<template>
  <div class="common-layout">
    <el-container class="layout-container">
      <!-- 侧边栏 -->
      <el-aside width="220px" class="sidebar">
        <div class="logo">
          <el-icon class="logo-icon"><Monitor /></el-icon>
          <span>qwq AIOps</span>
        </div>
        <el-menu
          active-text-color="#409EFF"
          background-color="#1e293b"
          text-color="#fff"
          :default-active="activeMenu"
          class="el-menu-vertical"
          @select="handleSelect"
        >
          <el-menu-item index="dashboard">
            <el-icon><Odometer /></el-icon>
            <span>系统概览</span>
          </el-menu-item>
          <el-menu-item index="terminal">
            <el-icon><ChatLineSquare /></el-icon>
            <span>AI 智能终端</span>
          </el-menu-item>
          <el-menu-item index="logs">
            <el-icon><Document /></el-icon>
            <span>运行日志</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- 主内容区 -->
      <el-container>
        <el-header class="header">
          <div class="breadcrumb">{{ pageTitle }}</div>
          <el-button type="primary" size="small" @click="triggerPatrol" :loading="patrolLoading">
            ⚡ 立即巡检
          </el-button>
        </el-header>
        <el-main class="main-content">
          <component :is="currentComponent" />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Dashboard from './components/Dashboard.vue'
import Terminal from './components/Terminal.vue'
import Logs from './components/Logs.vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const activeMenu = ref('dashboard')
const patrolLoading = ref(false)

const components = {
  dashboard: Dashboard,
  terminal: Terminal,
  logs: Logs
}

const titles = {
  dashboard: '系统概览 / Dashboard',
  terminal: '智能运维 / AI Terminal',
  logs: '系统日志 / System Logs'
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
body { margin: 0; background-color: #0f172a; color: #e2e8f0; font-family: 'Inter', sans-serif; }
.layout-container { height: 100vh; }
.sidebar { background-color: #1e293b; border-right: 1px solid #334155; }
.logo { height: 60px; display: flex; align-items: center; justify-content: center; font-size: 20px; font-weight: bold; color: #fff; border-bottom: 1px solid #334155; gap: 10px; }
.logo-icon { color: #409EFF; }
.el-menu { border-right: none !important; }
.header { background-color: #1e293b; border-bottom: 1px solid #334155; display: flex; align-items: center; justify-content: space-between; color: #fff; }
.main-content { background-color: #0f172a; padding: 20px; }
</style>