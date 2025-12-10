<template>
  <div class="websites-container">
    <!-- 工具栏：包含创建网站和刷新按钮 -->
    <el-card class="toolbar-card">
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ t('website.create') }}
      </el-button>
      <el-button @click="loadWebsites">
        <el-icon><Refresh /></el-icon>
        {{ t('common.refresh') }}
      </el-button>
    </el-card>

    <!-- 网站列表表格：显示所有已配置的网站信息 -->
    <el-card class="list-card">
      <el-table :data="websites" style="width: 100%" v-loading="loading">
        <!-- 域名列 -->
        <el-table-column prop="domain" label="域名" width="200" />
        <!-- 后端服务地址列 -->
        <el-table-column prop="backend_url" label="后端地址" width="200" />
        <!-- SSL证书状态列 -->
        <el-table-column label="SSL证书" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.ssl_enabled" type="success">已启用</el-tag>
            <el-tag v-else type="info">未启用</el-tag>
          </template>
        </el-table-column>
        <!-- 网站运行状态列 -->
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.enabled" type="success">运行中</el-tag>
            <el-tag v-else type="danger">已停止</el-tag>
          </template>
        </el-table-column>
        <!-- 创建时间列 -->
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <!-- 操作按钮列：启用/禁用、SSL管理、删除 -->
        <el-table-column label="操作" fixed="right" width="300">
          <template #default="{ row }">
            <el-button size="small" @click="toggleWebsite(row)">
              {{ row.enabled ? t('website.disable') : t('website.enable') }}
            </el-button>
            <el-button size="small" @click="manageSSL(row)">
              SSL管理
            </el-button>
            <el-button size="small" type="danger" @click="deleteWebsite(row)">
              {{ t('common.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建网站对话框：用于添加新的网站配置 -->
    <el-dialog v-model="createDialogVisible" title="创建网站" width="600px">
      <el-form :model="websiteForm" label-width="120px">
        <!-- 域名输入框 -->
        <el-form-item label="域名">
          <el-input v-model="websiteForm.domain" placeholder="example.com" />
        </el-form-item>
        <!-- 后端服务地址输入框 -->
        <el-form-item label="后端地址">
          <el-input v-model="websiteForm.backend_url" placeholder="http://localhost:3000" />
        </el-form-item>
        <!-- SSL启用开关 -->
        <el-form-item label="启用SSL">
          <el-switch v-model="websiteForm.ssl_enabled" />
        </el-form-item>
        <!-- 负载均衡策略选择器 -->
        <el-form-item label="负载均衡策略">
          <el-select v-model="websiteForm.load_balance">
            <el-option label="轮询" value="round_robin" />
            <el-option label="最少连接" value="least_conn" />
            <el-option label="IP哈希" value="ip_hash" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="createWebsite" :loading="creating">
          {{ t('common.create') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- SSL证书管理对话框：用于管理网站的SSL证书 -->
    <el-dialog v-model="sslDialogVisible" title="SSL证书管理" width="600px">
      <div v-if="currentWebsite">
        <!-- 显示当前网站的SSL证书信息 -->
        <el-descriptions :column="1" border>
          <el-descriptions-item label="域名">{{ currentWebsite.domain }}</el-descriptions-item>
          <el-descriptions-item label="SSL状态">
            <el-tag v-if="currentWebsite.ssl_enabled" type="success">已启用</el-tag>
            <el-tag v-else type="info">未启用</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="证书有效期" v-if="currentWebsite.ssl_cert_expiry">
            {{ formatDate(currentWebsite.ssl_cert_expiry) }}
          </el-descriptions-item>
        </el-descriptions>
        <!-- SSL证书操作按钮 -->
        <div class="ssl-actions">
          <el-button type="primary" @click="applySSL" :loading="applyingSSL">
            申请Let's Encrypt证书
          </el-button>
          <el-button @click="renewSSL" :loading="renewingSSL">
            续期证书
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
// Vue 3 组合式 API 和相关依赖导入
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

// 国际化函数
const { t } = useI18n()

// 响应式数据定义
const websites = ref([])              // 网站列表数据
const loading = ref(false)            // 加载状态
const createDialogVisible = ref(false) // 创建对话框显示状态
const sslDialogVisible = ref(false)   // SSL管理对话框显示状态
const creating = ref(false)           // 创建网站加载状态
const applyingSSL = ref(false)        // 申请SSL证书加载状态
const renewingSSL = ref(false)        // 续期SSL证书加载状态
const currentWebsite = ref(null)      // 当前选中的网站对象

// 网站表单数据结构
const websiteForm = ref({
  domain: '',                         // 域名
  backend_url: '',                    // 后端服务地址
  ssl_enabled: false,                 // 是否启用SSL
  load_balance: 'round_robin'         // 负载均衡策略
})

// 加载网站列表
const loadWebsites = async () => {
  loading.value = true
  try {
    const response = await axios.get('/api/websites')
    // 确保返回的数据是数组格式，避免 reduce 错误
    websites.value = Array.isArray(response.data) ? response.data : []
  } catch (error) {
    console.error('加载网站列表失败:', error)
    ElMessage.error('加载网站列表失败')
    // 出错时设置为空数组，避免渲染错误
    websites.value = []
  } finally {
    loading.value = false
  }
}

/**
 * 显示创建网站对话框
 * 重置表单数据并显示创建对话框
 */
const showCreateDialog = () => {
  // 重置表单为默认值
  websiteForm.value = {
    domain: '',
    backend_url: '',
    ssl_enabled: false,
    load_balance: 'round_robin'
  }
  createDialogVisible.value = true
}

/**
 * 创建新网站
 * 向后端API发送创建请求
 */
const createWebsite = async () => {
  creating.value = true
  try {
    await axios.post('/api/websites', websiteForm.value)
    ElMessage.success('网站创建成功')
    createDialogVisible.value = false
    loadWebsites() // 重新加载网站列表
  } catch (error) {
    console.error('创建网站失败:', error)
    ElMessage.error('网站创建失败')
  } finally {
    creating.value = false
  }
}

/**
 * 切换网站启用/禁用状态
 * @param {Object} website - 网站对象
 */
const toggleWebsite = async (website) => {
  try {
    await axios.put(`/api/websites/${website.id}`, {
      enabled: !website.enabled
    })
    // 更新本地状态
    website.enabled = !website.enabled
    ElMessage.success('状态更新成功')
  } catch (error) {
    console.error('状态更新失败:', error)
    ElMessage.error('状态更新失败')
  }
}

/**
 * 打开SSL管理对话框
 * @param {Object} website - 要管理SSL的网站对象
 */
const manageSSL = (website) => {
  currentWebsite.value = website
  sslDialogVisible.value = true
}

/**
 * 申请Let's Encrypt SSL证书
 * 为当前选中的网站申请免费SSL证书
 */
const applySSL = async () => {
  applyingSSL.value = true
  try {
    await axios.post(`/api/websites/${currentWebsite.value.id}/ssl/apply`)
    ElMessage.success('SSL证书申请成功')
    loadWebsites() // 刷新数据以显示最新的SSL状态
  } catch (error) {
    console.error('SSL证书申请失败:', error)
    ElMessage.error('SSL证书申请失败')
  } finally {
    applyingSSL.value = false
  }
}

/**
 * 续期SSL证书
 * 为即将过期的SSL证书进行续期
 */
const renewSSL = async () => {
  renewingSSL.value = true
  try {
    await axios.post(`/api/websites/${currentWebsite.value.id}/ssl/renew`)
    ElMessage.success('SSL证书续期成功')
    loadWebsites() // 刷新数据以显示最新的证书有效期
  } catch (error) {
    console.error('SSL证书续期失败:', error)
    ElMessage.error('SSL证书续期失败')
  } finally {
    renewingSSL.value = false
  }
}

/**
 * 删除网站配置
 * 显示确认对话框后删除指定网站
 * @param {Object} website - 要删除的网站对象
 */
const deleteWebsite = async (website) => {
  try {
    // 显示确认对话框
    await ElMessageBox.confirm('确定要删除此网站吗？', '警告', {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    
    // 发送删除请求
    await axios.delete(`/api/websites/${website.id}`)
    ElMessage.success('网站删除成功')
    loadWebsites() // 重新加载列表
  } catch (error) {
    // 用户取消删除操作时不显示错误信息
    if (error !== 'cancel') {
      console.error('网站删除失败:', error)
      ElMessage.error('网站删除失败')
    }
  }
}

/**
 * 格式化日期字符串为本地化显示格式
 * @param {string} dateStr - ISO日期字符串
 * @returns {string} 格式化后的日期字符串
 */
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 组件挂载时自动加载网站列表
onMounted(() => {
  loadWebsites()
})
</script>

<style scoped>
/* 网站管理页面容器样式 */
.websites-container {
  padding: 20px;
}

/* 工具栏卡片样式 */
.toolbar-card {
  margin-bottom: 20px;
}

/* 网站列表卡片样式 */
.list-card {
  margin-top: 20px;
}

/* SSL操作按钮组样式 */
.ssl-actions {
  margin-top: 20px;
  display: flex;
  gap: 10px;
}
</style>
