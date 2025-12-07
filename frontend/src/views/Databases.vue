<template>
  <div class="databases-container">
    <el-row :gutter="20">
      <!-- 左侧：连接列表 -->
      <el-col :span="6">
        <el-card class="connections-card">
          <template #header>
            <div class="card-header">
              <span>数据库连接</span>
              <el-button size="small" type="primary" @click="showCreateConnection">
                <el-icon><Plus /></el-icon>
              </el-button>
            </div>
          </template>
          <el-menu :default-active="activeConnection?.id" @select="selectConnection">
            <el-menu-item
              v-for="conn in connections"
              :key="conn.id"
              :index="conn.id"
            >
              <el-icon>
                <Coin v-if="conn.type === 'mysql'" />
                <DataBoard v-else-if="conn.type === 'postgresql'" />
                <Histogram v-else-if="conn.type === 'redis'" />
                <Document v-else />
              </el-icon>
              <span>{{ conn.name }}</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- 右侧：SQL编辑器和结果 -->
      <el-col :span="18">
        <el-card class="editor-card" v-if="activeConnection">
          <template #header>
            <div class="card-header">
              <span>{{ activeConnection.name }} - SQL编辑器</span>
              <div>
                <el-button size="small" type="primary" @click="executeQuery" :loading="executing">
                  <el-icon><CaretRight /></el-icon>
                  执行 (Ctrl+Enter)
                </el-button>
                <el-button size="small" @click="clearEditor">
                  <el-icon><Delete /></el-icon>
                  清空
                </el-button>
              </div>
            </div>
          </template>

          <!-- Monaco Editor -->
          <div ref="editorContainer" class="editor-container"></div>

          <!-- AI优化建议 -->
          <div v-if="aiSuggestion" class="ai-suggestion">
            <el-alert type="info" :closable="false">
              <template #title>
                <el-icon><MagicStick /></el-icon>
                AI优化建议
              </template>
              <div v-html="aiSuggestion"></div>
            </el-alert>
          </div>

          <!-- 查询结果 -->
          <div v-if="queryResult" class="query-result">
            <el-tabs v-model="activeTab">
              <el-tab-pane label="结果" name="result">
                <el-table
                  :data="queryResult.rows"
                  style="width: 100%"
                  max-height="400"
                  border
                >
                  <el-table-column
                    v-for="col in queryResult.columns"
                    :key="col"
                    :prop="col"
                    :label="col"
                    min-width="120"
                  />
                </el-table>
                <div class="result-info">
                  <span>共 {{ queryResult.rows?.length || 0 }} 行</span>
                  <span>执行时间: {{ queryResult.execution_time }}ms</span>
                </div>
              </el-tab-pane>
              <el-tab-pane label="消息" name="message">
                <pre>{{ queryResult.message || '查询执行成功' }}</pre>
              </el-tab-pane>
            </el-tabs>
          </div>
        </el-card>

        <el-empty v-else description="请选择一个数据库连接" />
      </el-col>
    </el-row>

    <!-- 创建连接对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建数据库连接" width="500px">
      <el-form :model="connectionForm" label-width="100px">
        <el-form-item label="连接名称">
          <el-input v-model="connectionForm.name" />
        </el-form-item>
        <el-form-item label="数据库类型">
          <el-select v-model="connectionForm.type" style="width: 100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="Redis" value="redis" />
            <el-option label="MongoDB" value="mongodb" />
          </el-select>
        </el-form-item>
        <el-form-item label="主机">
          <el-input v-model="connectionForm.host" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="connectionForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="connectionForm.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="connectionForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="数据库名">
          <el-input v-model="connectionForm.database" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createConnection" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
// 数据库管理视图 - 提供数据库连接管理、SQL 查询执行和 AI 优化建议功能
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as monaco from 'monaco-editor'

// 响应式数据
const connections = ref([])           // 数据库连接列表
const activeConnection = ref(null)    // 当前选中的连接
const createDialogVisible = ref(false) // 创建连接对话框显示状态
const creating = ref(false)           // 创建连接加载状态
const executing = ref(false)          // 查询执行加载状态
const queryResult = ref(null)         // 查询结果数据
const aiSuggestion = ref('')          // AI 优化建议
const activeTab = ref('result')       // 当前激活的标签页
const editorContainer = ref(null)     // Monaco 编辑器容器引用
let editor = null                     // Monaco 编辑器实例

// 连接表单数据
const connectionForm = ref({
  name: '',
  type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: 'root',
  password: '',
  database: ''
})

// 加载连接列表
const loadConnections = async () => {
  try {
    const response = await axios.get('/api/databases/connections')
    connections.value = response.data
  } catch (error) {
    ElMessage.error('加载连接列表失败')
  }
}

// 选择数据库连接并初始化 SQL 编辑器
const selectConnection = async (id) => {
  activeConnection.value = connections.value.find(c => c.id === id)
  queryResult.value = null
  aiSuggestion.value = ''
  
  // 等待 DOM 更新后初始化 Monaco 编辑器
  await nextTick()
  if (!editor && editorContainer.value) {
    editor = monaco.editor.create(editorContainer.value, {
      value: '-- 输入SQL查询\nSELECT * FROM users LIMIT 10;',
      language: 'sql',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14
    })

    // 注册 Ctrl+Enter 快捷键执行查询
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
      executeQuery()
    })
  }
}

// 显示创建连接对话框并重置表单
const showCreateConnection = () => {
  connectionForm.value = {
    name: '',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: '',
    database: ''
  }
  createDialogVisible.value = true
}

// 创建新的数据库连接
const createConnection = async () => {
  creating.value = true
  try {
    await axios.post('/api/databases/connections', connectionForm.value)
    ElMessage.success('连接创建成功')
    createDialogVisible.value = false
    loadConnections()
  } catch (error) {
    ElMessage.error('连接创建失败')
  } finally {
    creating.value = false
  }
}

// 执行 SQL 查询并获取 AI 优化建议
const executeQuery = async () => {
  if (!editor || !activeConnection.value) return

  const query = editor.getValue()
  if (!query.trim()) {
    ElMessage.warning('请输入SQL查询')
    return
  }

  executing.value = true
  try {
    const response = await axios.post(`/api/databases/connections/${activeConnection.value.id}/execute`, {
      query: query
    })
    queryResult.value = response.data
    
    // 如果后端返回 AI 优化建议则显示
    if (response.data.ai_suggestion) {
      aiSuggestion.value = response.data.ai_suggestion
    }
    
    ElMessage.success('查询执行成功')
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '查询执行失败')
  } finally {
    executing.value = false
  }
}

// 清空 SQL 编辑器内容
const clearEditor = () => {
  if (editor) {
    editor.setValue('')
  }
}

// 组件挂载时加载连接列表
onMounted(() => {
  loadConnections()
})

// 组件卸载前销毁编辑器实例
onBeforeUnmount(() => {
  if (editor) {
    editor.dispose()
  }
})
</script>

<style scoped>
.databases-container {
  padding: 20px;
  height: calc(100vh - 100px);
}

.connections-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.editor-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.editor-container {
  height: 300px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  margin-bottom: 20px;
}

.ai-suggestion {
  margin-bottom: 20px;
}

.query-result {
  margin-top: 20px;
}

.result-info {
  margin-top: 10px;
  display: flex;
  justify-content: space-between;
  color: #909399;
  font-size: 12px;
}
</style>
