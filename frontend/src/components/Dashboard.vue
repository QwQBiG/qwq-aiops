<template>
  <div class="dashboard">
    <!-- 状态卡片行 -->
    <el-row :gutter="20">
      <el-col :span="6" v-for="(item, index) in stats" :key="index">
        <el-card class="stat-card" shadow="never">
          <div class="stat-content">
            <div class="stat-info">
              <div class="stat-title">{{ item.title }}</div>
              <div class="stat-value">{{ item.value }} <span class="unit">{{ item.unit }}</span></div>
              <div class="stat-detail">{{ item.detail }}</div>
            </div>
            <el-progress type="circle" :percentage="item.percentage" :color="item.color" :width="70" :stroke-width="6" :show-text="false" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 应用监控列表 -->
    <el-card class="monitor-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>应用服务监控</span>
          <el-tag size="small" type="info">实时</el-tag>
        </div>
      </template>
      <el-table :data="services" style="width: 100%">
        <el-table-column prop="Name" label="服务名称">
          <template #default="scope">
            <div style="display: flex; align-items: center; gap: 8px">
              <div class="status-dot" :class="scope.row.Success ? 'up' : 'down'"></div>
              {{ scope.row.Name }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="URL" label="监控地址" />
        <el-table-column prop="Latency" label="响应时间" />
        <el-table-column label="状态">
          <template #default="scope">
            <el-tag :type="scope.row.Success ? 'success' : 'danger'" effect="dark" size="small">
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
  { title: 'CPU 负载', percentage: 0, value: '0', unit: '', detail: 'Load Avg', color: '#409EFF' },
  { title: '内存使用', percentage: 0, value: '0', unit: '%', detail: '0/0 GB', color: '#67C23A' },
  { title: '系统磁盘', percentage: 0, value: '0', unit: '%', detail: 'Root /', color: '#E6A23C' },
  { title: 'TCP 连接', percentage: 0, value: '0', unit: '', detail: 'Established', color: '#F56C6C' },
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
    stats.value[1].value = data.mem_pct
    stats.value[1].percentage = parseFloat(data.mem_pct)
    stats.value[1].detail = `${(parseFloat(data.mem_used)/1024).toFixed(1)} / ${(parseFloat(data.mem_total)/1024).toFixed(1)} GB`

    // 磁盘
    stats.value[2].value = data.disk_pct.replace('%', '')
    stats.value[2].percentage = parseFloat(data.disk_pct)
    stats.value[2].detail = `剩余 ${data.disk_avail}`

    // TCP
    const tcpCount = parseInt(data.tcp_conn || 0)
    stats.value[3].value = tcpCount
    stats.value[3].percentage = Math.min((tcpCount / 1000) * 100, 100)

    if (data.services) services.value = data.services
  } catch (e) { console.error(e) }
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 2000)
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.stat-card { background: #1d2129; border: 1px solid #2c3038; color: #fff; margin-bottom: 20px; }
.stat-content { display: flex; justify-content: space-between; align-items: center; }
.stat-title { font-size: 14px; color: #86909c; margin-bottom: 8px; }
.stat-value { font-size: 28px; font-weight: 600; color: #fff; }
.unit { font-size: 14px; color: #86909c; font-weight: normal; }
.stat-detail { font-size: 12px; color: #86909c; margin-top: 8px; }

.monitor-card { background: #1d2129; border: 1px solid #2c3038; color: #fff; }
.card-header { display: flex; justify-content: space-between; align-items: center; font-weight: 600; }

.status-dot { width: 8px; height: 8px; border-radius: 50%; }
.status-dot.up { background: #67C23A; box-shadow: 0 0 8px rgba(103, 194, 58, 0.5); }
.status-dot.down { background: #F56C6C; box-shadow: 0 0 8px rgba(245, 108, 108, 0.5); }

:deep(.el-table) { background-color: transparent; --el-table-tr-bg-color: transparent; --el-table-header-bg-color: #161920; --el-table-text-color: #c9cdd4; --el-table-border-color: #2c3038; --el-table-row-hover-bg-color: #272b36 !important; }
:deep(.el-table th.el-table__cell) { background-color: #161920; font-weight: 500; }
:deep(.el-card__header) { border-bottom: 1px solid #2c3038; padding: 15px 20px; }
</style>