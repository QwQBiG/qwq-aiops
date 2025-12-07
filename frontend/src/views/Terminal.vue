<template>
  <div class="terminal-container">
    <!-- 聊天消息窗口 -->
    <div class="chat-window" ref="chatWindow">
      <div v-for="(msg, index) in messages" :key="index" :class="['message', msg.type]">
        <!-- 头像显示 -->
        <div class="avatar" v-if="msg.type !== 'user'">🤖</div>
        <div class="avatar" v-else>👤</div>
        <!-- 消息内容 -->
        <div class="content">
          <div v-if="msg.type === 'log'" class="log-content">{{ msg.content }}</div>
          <div v-else v-html="renderMarkdown(msg.content)"></div>
        </div>
      </div>
    </div>
    <!-- 输入区域 -->
    <div class="input-area">
      <el-input
        v-model="input"
        placeholder="输入运维指令，例如：看看内存、生成 Nginx 配置..."
        @keyup.enter="send"
        :disabled="loading"
      >
        <template #append>
          <el-button @click="send" :loading="loading">发送</el-button>
        </template>
      </el-input>
    </div>
  </div>
</template>

<script setup>
// AI 终端视图组件 - 提供智能运维对话交互功能
import { ref, onMounted, nextTick } from 'vue'
import { marked } from 'marked'

// 响应式数据
const messages = ref([
  { type: 'ai', content: '你好！我是 qwq 智能运维专家。请直接下达指令。' }
])
const input = ref('')              // 用户输入内容
const loading = ref(false)         // 加载状态
const chatWindow = ref(null)       // 聊天窗口引用
let ws = null                      // WebSocket 连接实例

// 渲染 Markdown 格式文本
const renderMarkdown = (text) => {
  return marked(text)
}

// 滚动到聊天窗口底部
const scrollToBottom = () => {
  nextTick(() => {
    if (chatWindow.value) {
      chatWindow.value.scrollTop = chatWindow.value.scrollHeight
    }
  })
}

// 建立 WebSocket 连接
const connectWS = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${protocol}//${window.location.host}/ws/chat`)
  
  // 处理接收到的消息
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    if (data.type === 'status') return

    // 避免重复消息
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg && lastMsg.content === data.content && lastMsg.type === (data.type === 'answer' ? 'ai' : 'log')) return

    // 根据消息类型添加到消息列表
    if (data.type === 'log') {
      messages.value.push({ type: 'log', content: data.content })
    } else if (data.type === 'answer') {
      messages.value.push({ type: 'ai', content: data.content })
      loading.value = false
    }
    scrollToBottom()
  }
}

// 发送用户消息
const send = () => {
  if (!input.value.trim()) return
  messages.value.push({ type: 'user', content: input.value })
  ws.send(input.value)
  input.value = ''
  loading.value = true
  scrollToBottom()
}

// 组件挂载时建立连接
onMounted(() => {
  connectWS()
})
</script>

<style scoped>
.terminal-container { display: flex; flex-direction: column; height: calc(100vh - 140px); background: #1e293b; border-radius: 8px; border: 1px solid #334155; }
.chat-window { flex: 1; overflow-y: auto; padding: 20px; }
.message { display: flex; gap: 15px; margin-bottom: 20px; }
.avatar { font-size: 24px; }
.content { background: #0f172a; padding: 10px 15px; border-radius: 8px; max-width: 80%; line-height: 1.6; font-size: 14px; }
.message.user { flex-direction: row-reverse; }
.message.user .content { background: #409EFF; color: white; }
.message.log .content { background: #000; color: #67C23A; font-family: monospace; border-left: 3px solid #67C23A; width: 100%; }
.input-area { padding: 20px; border-top: 1px solid #334155; background: #1e293b; }
:deep(pre) { background: #000; padding: 10px; border-radius: 4px; overflow-x: auto; }
:deep(code) { font-family: 'Consolas', monospace; }
</style>
