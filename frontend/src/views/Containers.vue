<template>
  <div class="containers">
    <el-card class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>容器列表</span>
          <el-button type="primary" size="small" @click="fetchContainers">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
      </template>
      
      <el-table :data="containers" style="width: 100%" v-loading="loading">
        <el-table-column prop="name" label="名称" width="200" />
        <el-table-column prop="image" label="镜像" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="220" />
        <el-table-column label="运行状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.state === 'running' ? 'success' : 'info'" size="small" effect="dark">
              {{ scope.row.state === 'running' ? '运行中' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="scope">
            <el-button-group>
              <el-button 
                v-if="scope.row.state !== 'running'" 
                type="success" size="small" link 
                @click="handleAction(scope.row.id, 'start')">
                启动
              </el-button>
              <el-button 
                v-if="scope.row.state === 'running'" 
                type="danger" size="small" link 
                @click="handleAction(scope.row.id, 'stop')">
                停止
              </el-button>
              <el-button type="primary" size="small" link @click="handleAction(scope.row.id, 'restart')">
                重启
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const containers = ref([])
const loading = ref(false)

const fetchContainers = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/containers')
    containers.value = res.data || []
  } catch (e) {
    ElMessage.error('获取容器列表失败')
  } finally {
    loading.value = false
  }
}

const handleAction = async (id, action) => {
  try {
    await axios.get(`/api/container/action?id=${id}&action=${action}`)
    ElMessage.success('操作指令已发送')
    setTimeout(fetchContainers, 1000)
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

onMounted(() => {
  fetchContainers()
})
</script>

<style scoped>
.box-card { background: #1d2129; border: 1px solid #2c3038; color: #fff; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
:deep(.el-table) { background-color: transparent; --el-table-tr-bg-color: transparent; --el-table-header-bg-color: #161920; --el-table-text-color: #c9cdd4; --el-table-border-color: #2c3038; --el-table-row-hover-bg-color: #272b36 !important; }
:deep(.el-card__header) { border-bottom: 1px solid #2c3038; }
</style>
