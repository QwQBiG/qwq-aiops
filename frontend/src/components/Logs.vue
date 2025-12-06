<template>
  <div class="logs-container">
    <div class="log-window" ref="logBox">
      <div v-for="(log, index) in logs" :key="index" class="log-line">{{ log }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import axios from 'axios'

const logs = ref([])
const logBox = ref(null)
let timer = null

const fetchLogs = async () => {
  const res = await axios.get('/api/logs')
  if (JSON.stringify(res.data) !== JSON.stringify(logs.value)) {
    logs.value = res.data
    nextTick(() => {
      if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
    })
  }
}

onMounted(() => {
  fetchLogs()
  timer = setInterval(fetchLogs, 2000)
})

onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.logs-container { height: calc(100vh - 140px); }
.log-window { height: 100%; background: #000; color: #a5b4fc; padding: 15px; border-radius: 8px; overflow-y: auto; font-family: monospace; font-size: 13px; line-height: 1.5; }
.log-line { border-bottom: 1px solid #1e293b; }
</style>