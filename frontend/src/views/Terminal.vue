<template>
  <section class="terminal-container">
    <div class="terminal-toolbar">
      <el-tag :type="state === 'connected' ? 'success' : 'info'">{{ t(`terminal.${state}`) }}</el-tag>
      <span v-if="loading" role="status">{{ t(`terminal.${activity}`) }}</span>
      <el-button v-if="loading" @click="cancel">{{ t('terminal.cancel') }}</el-button>
      <el-button v-if="state === 'disconnected'" @click="connection.connect">{{ t('terminal.reconnect') }}</el-button>
    </div>
    <div ref="chatWindow" class="chat-window" role="log" aria-live="polite">
      <p v-if="!messages.length" class="welcome">{{ t('terminal.welcome') }}</p>
      <div v-for="message in messages" :key="message.id" :class="['message', message.type]">
        <span class="speaker">{{ t(message.type === 'user' ? 'terminal.you' : message.type === 'log' ? 'terminal.output' : 'terminal.assistant') }}</span>
        <div v-if="message.type !== 'answer'" class="plain-content">{{ message.content }}</div>
        <div v-else class="content" v-html="renderMarkdown(message.content)"></div>
      </div>
    </div>
    <div class="input-area">
      <el-input v-model="input" :placeholder="t('terminal.placeholder')" :disabled="loading || state !== 'connected'" @keyup.enter="send">
        <template #append><el-button :disabled="loading || state !== 'connected'" @click="send">{{ t('terminal.send') }}</el-button></template>
      </el-input>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { renderMarkdown } from '../lib/content'
import { createTerminalConnection } from '../lib/terminal'

const { t } = useI18n()
const messages = ref([])
const input = ref('')
const loading = ref(false)
const state = ref('connecting')
const activity = ref('thinking')
const chatWindow = ref(null)
let nextId = 0

function append(type, content) {
  messages.value.push({ id: nextId++, type, content })
  if (messages.value.length > 200) messages.value.splice(0, messages.value.length - 200)
  nextTick(() => { if (chatWindow.value) chatWindow.value.scrollTop = chatWindow.value.scrollHeight })
}

const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
const connection = createTerminalConnection(`${protocol}//${window.location.host}/ws/chat`, {
  onState(value) {
    state.value = value
    if (value === 'disconnected') loading.value = false
  },
  onError(key) {
    loading.value = false
    ElMessage.error(t(`terminal.${key}`))
  },
  onMessage(message) {
    if (message.type === 'status') {
      if (message.content === 'done') loading.value = false
      else if (['thinking', 'executing'].includes(message.content)) activity.value = message.content
      return
    }
    append(message.type, message.content)
  }
})

function send() {
  const text = input.value.trim()
  if (!text || loading.value) return
  if (!connection.send(text)) {
    ElMessage.warning(t('terminal.connectionError'))
    return
  }
  append('user', text)
  input.value = ''
  activity.value = 'thinking'
  loading.value = true
}

function cancel() {
  connection.disconnect()
  append('log', t('terminal.cancelled'))
  connection.connect()
}

onMounted(connection.connect)
onUnmounted(connection.disconnect)
</script>

<style scoped>
.terminal-container { display: flex; flex-direction: column; height: calc(100vh - 140px); background: #1e293b; color: #e2e8f0; font-family: system-ui, 'Segoe UI', 'Microsoft YaHei', sans-serif; border-radius: 8px; border: 1px solid #334155; }
.terminal-toolbar { display: flex; align-items: center; gap: 12px; padding: 12px 20px; border-bottom: 1px solid #334155; }
.chat-window { flex: 1; overflow-y: auto; padding: 20px; }
.welcome { color: #cbd5e1; }
.message { margin-bottom: 20px; }
.speaker { display: block; font-size: 12px; margin-bottom: 6px; color: #94a3b8; }
.content, .plain-content { background: #0f172a; padding: 10px 15px; border-radius: 8px; line-height: 1.6; overflow-wrap: anywhere; }
.plain-content { white-space: pre-wrap; }
.user .plain-content { background: #1d4ed8; }
.log .plain-content { background: #000; color: #86efac; font-family: monospace; }
.input-area { padding: 20px; border-top: 1px solid #334155; }
:deep(pre) { background: #000; padding: 10px; border-radius: 4px; overflow-x: auto; }
:deep(code) { font-family: Consolas, monospace; }
:deep(strong), :deep(em), :deep(code) { color: inherit; }
:deep(a) { color: #93c5fd; }
</style>
