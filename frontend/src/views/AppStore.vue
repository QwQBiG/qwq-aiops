<template>
  <div class="appstore-container">
    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <el-row :gutter="20">
        <el-col :span="12">
          <el-input
            v-model="searchQuery"
            :placeholder="t('appstore.search')"
            clearable
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </el-col>
        <el-col :span="12">
          <el-select v-model="selectedCategory" :placeholder="t('appstore.category')" @change="handleSearch">
            <el-option label="全部" value="" />
            <el-option label="数据库" value="database" />
            <el-option label="Web服务器" value="webserver" />
            <el-option label="开发工具" value="devtools" />
            <el-option label="监控工具" value="monitoring" />
            <el-option label="消息队列" value="messagequeue" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <!-- 应用列表 -->
    <el-row :gutter="20" class="app-grid">
      <el-col :span="6" v-for="app in filteredApps" :key="app.id">
        <el-card class="app-card" shadow="hover">
          <div class="app-icon">
            <el-icon :size="48"><Box /></el-icon>
          </div>
          <h3>{{ app.name }}</h3>
          <p class="app-description">{{ app.description }}</p>
          <div class="app-meta">
            <el-tag size="small">{{ app.version }}</el-tag>
            <el-tag size="small" type="info">{{ app.category }}</el-tag>
          </div>
          <div class="app-actions">
            <el-button
              v-if="!app.installed"
              type="primary"
              size="small"
              @click="installApp(app)"
              :loading="app.installing"
            >
              {{ t('appstore.install') }}
            </el-button>
            <el-button
              v-else
              type="danger"
              size="small"
              @click="uninstallApp(app)"
              :loading="app.uninstalling"
            >
              {{ t('appstore.uninstall') }}
            </el-button>
            <el-button size="small" @click="showDetails(app)">
              {{ t('appstore.details') }}
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 应用详情对话框 -->
    <el-dialog v-model="detailsVisible" :title="currentApp?.name" width="600px">
      <div v-if="currentApp">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="版本">{{ currentApp.version }}</el-descriptions-item>
          <el-descriptions-item label="分类">{{ currentApp.category }}</el-descriptions-item>
          <el-descriptions-item label="描述">{{ currentApp.description }}</el-descriptions-item>
          <el-descriptions-item label="端口">{{ currentApp.ports?.join(', ') }}</el-descriptions-item>
        </el-descriptions>
        <div class="app-readme" v-if="currentApp.readme">
          <h4>说明文档</h4>
          <div v-html="currentApp.readme"></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
// 应用商店视图组件 - 提供应用浏览、搜索、安装和卸载功能
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

// 响应式数据
const searchQuery = ref('')        // 搜索关键词
const selectedCategory = ref('')   // 选中的分类
const apps = ref([])                // 应用列表
const detailsVisible = ref(false)  // 详情对话框显示状态
const currentApp = ref(null)       // 当前查看的应用

// 过滤应用列表 - 根据搜索词和分类筛选
const filteredApps = computed(() => {
  return apps.value.filter(app => {
    const matchSearch = !searchQuery.value || 
      app.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      app.description.toLowerCase().includes(searchQuery.value.toLowerCase())
    const matchCategory = !selectedCategory.value || app.category === selectedCategory.value
    return matchSearch && matchCategory
  })
})

// 加载应用列表和安装状态
const loadApps = async () => {
  try {
    // 获取所有可用应用模板
    const response = await axios.get('/api/appstore/templates')
    apps.value = response.data.map(app => ({
      ...app,
      installed: false,
      installing: false,
      uninstalling: false
    }))
    
    // 获取已安装应用并更新状态
    const installedResponse = await axios.get('/api/appstore/instances')
    const installedIds = installedResponse.data.map(inst => inst.template_id)
    apps.value.forEach(app => {
      app.installed = installedIds.includes(app.id)
    })
  } catch (error) {
    ElMessage.error('加载应用列表失败')
  }
}

// 安装应用 - 提示输入实例名称后创建应用实例
const installApp = async (app) => {
  try {
    await ElMessageBox.prompt('请输入应用实例名称', '安装应用', {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      inputPattern: /^[a-zA-Z0-9_-]+$/,
      inputErrorMessage: '名称只能包含字母、数字、下划线和连字符'
    })
    
    app.installing = true
    await axios.post('/api/appstore/instances', {
      template_id: app.id,
      name: app.name.toLowerCase(),
      config: {}
    })
    app.installed = true
    ElMessage.success('应用安装成功')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('应用安装失败')
    }
  } finally {
    app.installing = false
  }
}

// 卸载应用 - 确认后删除应用实例
const uninstallApp = async (app) => {
  try {
    await ElMessageBox.confirm('确定要卸载此应用吗？', '警告', {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    
    app.uninstalling = true
    // 查找对应的实例ID并删除
    const response = await axios.get('/api/appstore/instances')
    const instance = response.data.find(inst => inst.template_id === app.id)
    if (instance) {
      await axios.delete(`/api/appstore/instances/${instance.id}`)
      app.installed = false
      ElMessage.success('应用卸载成功')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('应用卸载失败')
    }
  } finally {
    app.uninstalling = false
  }
}

// 显示应用详情对话框
const showDetails = (app) => {
  currentApp.value = app
  detailsVisible.value = true
}

// 搜索处理（实际逻辑在 computed 中）
const handleSearch = () => {
  // 搜索逻辑已在 filteredApps 计算属性中处理
}

// 组件挂载时加载应用列表
onMounted(() => {
  loadApps()
})
</script>

<style scoped>
.appstore-container {
  padding: 20px;
}

.search-card {
  margin-bottom: 20px;
}

.app-grid {
  margin-top: 20px;
}

.app-card {
  text-align: center;
  margin-bottom: 20px;
  transition: transform 0.2s;
}

.app-card:hover {
  transform: translateY(-5px);
}

.app-icon {
  margin: 20px 0;
  color: #409EFF;
}

.app-card h3 {
  margin: 10px 0;
  font-size: 18px;
}

.app-description {
  color: #909399;
  font-size: 14px;
  min-height: 40px;
  margin: 10px 0;
}

.app-meta {
  margin: 15px 0;
  display: flex;
  justify-content: center;
  gap: 10px;
}

.app-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 15px;
}

.app-readme {
  margin-top: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
}

.app-readme h4 {
  margin-top: 0;
}
</style>
