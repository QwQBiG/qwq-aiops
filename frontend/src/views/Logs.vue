<template>
  <section class="logs-container">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <div class="log-toolbar"><el-switch v-model="follow" :active-text="t('logs.follow')" /></div>
    <div ref="logBox" class="log-window" role="log" aria-live="polite">
      <p v-if="!logs.length">{{ t('logs.empty') }}</p>
      <div v-for="(log, index) in logs" :key="index" class="log-line">{{ log }}</div>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { createPoller } from '../lib/poller'
import { errorMessage } from '../lib/errors'
const { t } = useI18n()
const logs = ref([])
const follow = ref(true)
const error = ref('')
const logBox = ref(null)
const poller = createPoller(async signal => {
  const { data } = await axios.get('/api/logs', { signal })
  logs.value = Array.isArray(data) ? data : []
  error.value = ''
  if (follow.value) await nextTick(() => { if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight })
}, { onError: failure => { error.value = errorMessage(failure, t('console.requestFailed')) } })
onMounted(poller.start)
onUnmounted(poller.stop)
</script>

<style scoped>
.logs-container { height: calc(100vh - 140px); display: flex; flex-direction: column; gap: 12px; }
.log-toolbar { display: flex; justify-content: flex-end; }
.log-window { flex: 1; min-height: 0; background: #000; color: #a5b4fc; padding: 15px; border-radius: 8px; overflow: auto; font-family: monospace; font-size: 13px; line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; }
.log-line { border-bottom: 1px solid #1e293b; }
</style>
