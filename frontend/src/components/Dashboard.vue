<template>
  <div class="dashboard">
    <!-- 状态卡片行 -->
    <el-row :gutter="20">
      <el-col :span="6" v-for="(item, index) in stats" :key="index">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-header">{{ item.title }}</div>
          <div class="stat-body">
            <el-progress type="dashboard" :percentage="item.percentage" :color="item.color" :width="120">
              <template #default="{ percentage }">
                <span class="percentage-value">{{ item.value }}</span>
                <span class="percentage-label">{{ item.unit }}</span>
              </template>
            </el-progress>
            <div class="stat-footer">{{ item.detail }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 应用监控列表 -->
    <el-card class="monitor-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <span>🌐 应用服务监控</span>
        </div>
      </template>
      <el-table :data="services" style="width: 100%" :row-class-name="tableRowClassName">
        <el-table-column prop="Name" label="服务名称" />
        <el-table-column prop="URL" label="监控地址" />
        <el-table-column prop="Latency" label="响应时间" />
        <el-table-column label="状态">
          <template #default="scope">
            <el-tag :type="scope.row.Success ? 'success' : 'danger'">
              {{ scope.row.Success ? '运行中' : '异常' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import axios from 'axios'

const stats = ref([
  { title: 'CPU 负载', percentage: 0, value: '0', unit: 'Load', detail: '1min / 5min / 15min', color: '#409EFF' },
  { title: '内存使用', percentage: 0, value: '0', unit: 'GB', detail: 'Total: 0 GB', color: '#67C23A' },
  { title: '系统磁盘', percentage: 0, value: '0', unit: '%', detail: 'Free: 0 GB', color: '#E6A23C' },
  { title: 'TCP 连接', percentage: 0, value: '0', unit: '个', detail: 'Established', color: '#F56C6C' },
])

const services = ref([])
let timer = null

const fetchData = async () => {
  try {
    const res = await axios.get('/api/stats')
    const data = Array.isArray(res.data) ? res.data[res.data.length - 1] : res.data
    if (!data) return

    // CPU
    const load = parseFloat(data.load.split(',')[0])
    stats.value[0].value = load
    stats.value[0].percentage = Math.min(load * 10, 100)
    stats.value[0].detail = data.load

    // 内存
    stats.value[1].value = (parseFloat(data.mem_used) / 1024).toFixed(1)
    stats.value[1].percentage = parseFloat(data.mem_pct)
    stats.value[1].detail = `Total: ${(parseFloat(data.mem_total) / 1024).toFixed(1)} GB`

    // 磁盘
    stats.value[2].value = data.disk_pct.replace('%', '')
    stats.value[2].percentage = parseFloat(data.disk_pct)
    stats.value[2].detail = `Free: ${data.disk_avail}`

    // TCP 连接数
    const tcpCount = parseInt(data.tcp_conn || 0)
    stats.value[3].value = tcpCount
    stats.value[3].percentage = Math.min((tcpCount / 10000) * 100, 100)

    // 服务列表
    if (data.services) {
      services.value = data.services
    }
  } catch (e) {
    console.error(e)
  }
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 2000)
})

onUnmounted(() => clearInterval(timer))

const tableRowClassName = ({ row }) => {
  return row.Success ? '' : 'warning-row'
}
</script>

<style scoped>
.stat-card { background: #1e293b; border: 1px solid #334155; color: #fff; height: 240px; }
.stat-header { font-size: 16px; font-weight: bold; margin-bottom: 20px; color: #909399; }
.stat-body { display: flex; flex-direction: column; align-items: center; }
.percentage-value { display: block; margin-top: 10px; font-size: 24px; font-weight: bold; }
.percentage-label { display: block; font-size: 12px; color: #909399; }
.stat-footer { margin-top: 15px; font-size: 12px; color: #909399; }
.monitor-card { margin-top: 20px; background: #1e293b; border: 1px solid #334155; color: #fff; }

:deep(.el-table) { 
  background-color: transparent; 
  --el-table-tr-bg-color: transparent; 
  --el-table-header-bg-color: #0f172a; 
  --el-table-text-color: #fff; 
  --el-table-border-color: #334155; 
  --el-table-row-hover-bg-color: #334155 !important;
}
:deep(.el-table__inner-wrapper::before) { background-color: #334155; }
:deep(.el-card__header) { border-bottom: 1px solid #334155; }
</style>