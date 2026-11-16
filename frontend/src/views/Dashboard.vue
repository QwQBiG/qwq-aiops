<template>
  <section>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <el-alert v-if="metricErrors.length" :title="t('dashboard.metricsUnavailable')" type="warning" :closable="false" show-icon>
      <ul><li v-for="failure in metricErrors" :key="failure.key">{{ t(`dashboard.${failure.key}`) }}: {{ failure.message }}</li></ul>
    </el-alert>
    <el-empty v-if="!latest && !error" :description="t('console.waitingMetrics')" />
    <el-row v-if="latest" :gutter="16">
      <el-col v-for="card in cards" :key="card.key" :xs="24" :sm="12" :lg="6">
        <el-card class="stat-card" shadow="never">
          <div class="stat-title">{{ t(`dashboard.${card.key}`) }}</div>
          <div class="stat-value">{{ card.value }}</div>
          <div class="stat-detail">{{ card.detail }}</div>
          <el-progress v-if="card.percentage !== undefined" :percentage="card.percentage" :show-text="false" />
        </el-card>
      </el-col>
    </el-row>
    <el-card shadow="never" class="monitor-card">
      <template #header>{{ t('dashboard.services') }}</template>
      <el-table :data="services" :empty-text="t('dashboard.noServices')">
        <el-table-column prop="Name" :label="t('console.name')" />
        <el-table-column prop="URL" :label="t('dashboard.address')" show-overflow-tooltip />
        <el-table-column prop="Latency" :label="t('dashboard.latency')" />
        <el-table-column :label="t('console.status')">
          <template #default="{ row }">
            <el-tag :type="row.Success ? 'success' : 'danger'">{{ t(row.Success ? 'dashboard.healthy' : 'dashboard.unhealthy') }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { createPoller } from '../lib/poller'
import { errorMessage } from '../lib/errors'

const { t } = useI18n()
const latest = ref(null)
const error = ref('')
const services = computed(() => Array.isArray(latest.value?.services) ? latest.value.services : [])
const metricErrors = computed(() => Object.entries(latest.value?.errors || {}).map(([key, message]) => ({ key, message })))
const percentage = value => {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? Math.max(0, Math.min(100, parsed)) : undefined
}
const cards = computed(() => {
  const data = latest.value || {}
  const unavailable = t('dashboard.unavailable')
  const memory = [data.mem_used, data.mem_total].map(value => Number.parseFloat(value) / 1024)
  const memoryPercentage = percentage(data.mem_pct)
  const diskPercentage = percentage(data.disk_pct)
  return [
    { key: 'load', value: data.load?.split(',')[0] || unavailable, detail: t('dashboard.loadDetail') },
    { key: 'memory', value: memoryPercentage === undefined ? unavailable : `${data.mem_pct}%`, detail: memory.every(Number.isFinite) ? `${memory[0].toFixed(1)} / ${memory[1].toFixed(1)} GB` : unavailable, percentage: memoryPercentage },
    { key: 'disk', value: diskPercentage === undefined ? unavailable : data.disk_pct, detail: data.disk_avail ? t('dashboard.available', { size: data.disk_avail }) : unavailable, percentage: diskPercentage },
    { key: 'tcp', value: data.tcp_conn || unavailable, detail: t('dashboard.established') }
  ]
})
const poller = createPoller(async signal => {
  const { data } = await axios.get('/api/stats', { signal })
  latest.value = Array.isArray(data) ? data.at(-1) || null : data
  error.value = ''
}, { onError: failure => { error.value = errorMessage(failure, t('console.requestFailed')) } })
onMounted(poller.start)
onUnmounted(poller.stop)
</script>

<style scoped>
.stat-card { margin-bottom: 16px; }
.stat-title { color: #94a3b8; font-size: 14px; }
.stat-value { font-size: 28px; margin: 10px 0; }
.stat-detail { color: #94a3b8; font-size: 12px; margin-bottom: 12px; }
.monitor-card { margin-top: 8px; }
.el-alert { margin-bottom: 16px; }
</style>
