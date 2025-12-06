<template>
  <div class="files-container">
    <!-- 顶部操作栏 -->
    <el-card class="box-card" shadow="never">
      <div class="toolbar">
        <div class="breadcrumb">
          <el-button link @click="loadFiles('/')"><el-icon><HomeFilled /></el-icon></el-button>
          <span v-for="(part, index) in pathParts" :key="index">
            <span class="separator">/</span>
            <el-button link @click="loadFiles(buildPath(index))">{{ part }}</el-button>
          </span>
        </div>
        <div class="actions">
          <el-button type="primary" size="small" @click="refresh"><el-icon><Refresh /></el-icon></el-button>
          <el-button type="success" size="small" @click="showMkdir = true"><el-icon><FolderAdd /></el-icon> 新建目录</el-button>
        </div>
      </div>
    </el-card>

    <!-- 文件列表 -->
    <el-card class="list-card" shadow="never" v-loading="loading">
      <el-table :data="files" style="width: 100%" @row-click="handleRowClick">
        <el-table-column width="50">
          <template #default="scope">
            <el-icon v-if="scope.row.is_dir" class="icon-dir"><Folder /></el-icon>
            <el-icon v-else class="icon-file"><Document /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="size" label="大小" width="120">
          <template #default="scope">
            {{ scope.row.is_dir ? '-' : formatSize(scope.row.size) }}
          </template>
        </el-table-column>
        <el-table-column prop="mode" label="权限" width="120" />
        <el-table-column prop="mod_time" label="修改时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="scope">
            <el-button v-if="!scope.row.is_dir" link type="primary" @click.stop="editFile(scope.row)">编辑</el-button>
            <el-button link type="danger" @click.stop="deleteFile(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 编辑器弹窗 -->
    <el-dialog v-model="showEditor" :title="'编辑: ' + currentFile" width="70%" top="5vh">
      <el-input
        v-model="fileContent"
        type="textarea"
        :rows="20"
        class="code-editor"
        spellcheck="false"
      />
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showEditor = false">取消</el-button>
          <el-button type="primary" @click="saveFile" :loading="saving">保存</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 新建文件夹弹窗 -->
    <el-dialog v-model="showMkdir" title="新建文件夹" width="30%">
      <el-input v-model="newDirName" placeholder="文件夹名称" />
      <template #footer>
        <el-button type="primary" @click="createDir">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const currentPath = ref('/')
const files = ref([])
const loading = ref(false)
const showEditor = ref(false)
const showMkdir = ref(false)
const currentFile = ref('')
const fileContent = ref('')
const saving = ref(false)
const newDirName = ref('')

const pathParts = computed(() => {
  return currentPath.value.split('/').filter(p => p)
})

const buildPath = (index) => {
  return '/' + pathParts.value.slice(0, index + 1).join('/')
}

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const loadFiles = async (path) => {
  loading.value = true
  try {
    const res = await axios.get(`/api/files/list?path=${encodeURIComponent(path)}`)
    files.value = res.data.files || []
    currentPath.value = res.data.path
  } catch (e) {
    ElMessage.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

const handleRowClick = (row) => {
  if (row.is_dir) {
    const nextPath = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
    loadFiles(nextPath)
  }
}

const refresh = () => loadFiles(currentPath.value)

const editFile = async (row) => {
  const filePath = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
  try {
    const res = await axios.get(`/api/files/content?path=${encodeURIComponent(filePath)}`)
    fileContent.value = res.data
    currentFile.value = filePath
    showEditor.value = true
  } catch (e) {
    ElMessage.error('无法读取文件')
  }
}

const saveFile = async () => {
  saving.value = true
  try {
    await axios.post('/api/files/save', {
      path: currentFile.value,
      content: fileContent.value
    })
    ElMessage.success('保存成功')
    showEditor.value = false
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const deleteFile = (row) => {
  ElMessageBox.confirm(`确定要删除 ${row.name} 吗？`, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(async () => {
    const filePath = currentPath.value === '/' ? '/' + row.name : currentPath.value + '/' + row.name
    await axios.get(`/api/files/action?type=delete&path=${encodeURIComponent(filePath)}`)
    ElMessage.success('已删除')
    refresh()
  })
}

const createDir = async () => {
  if (!newDirName.value) return
  const newPath = currentPath.value === '/' ? '/' + newDirName.value : currentPath.value + '/' + newDirName.value
  try {
    await axios.get(`/api/files/action?type=mkdir&path=${encodeURIComponent(newPath)}`)
    ElMessage.success('创建成功')
    showMkdir.value = false
    newDirName.value = ''
    refresh()
  } catch (e) {
    ElMessage.error('创建失败')
  }
}

onMounted(() => loadFiles('/'))
</script>

<style scoped>
.files-container { display: flex; flex-direction: column; gap: 20px; height: calc(100vh - 100px); }
.box-card { background: #1d2129; border: 1px solid #2c3038; color: #fff; }
.list-card { background: #1d2129; border: 1px solid #2c3038; color: #fff; flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.toolbar { display: flex; justify-content: space-between; align-items: center; }
.breadcrumb { display: flex; align-items: center; font-size: 14px; }
.separator { margin: 0 5px; color: #606266; }
.icon-dir { color: #E6A23C; font-size: 18px; }
.icon-file { color: #909399; font-size: 18px; }
.code-editor :deep(textarea) { background-color: #0f172a; color: #a5b4fc; font-family: 'Consolas', monospace; border-color: #334155; }
:deep(.el-table) { background-color: transparent; --el-table-tr-bg-color: transparent; --el-table-header-bg-color: #161920; --el-table-text-color: #c9cdd4; --el-table-border-color: #2c3038; --el-table-row-hover-bg-color: #272b36 !important; cursor: pointer; }
:deep(.el-dialog) { background-color: #1d2129; }
:deep(.el-dialog__title) { color: #fff; }
:deep(.el-dialog__body) { color: #fff; }
</style>