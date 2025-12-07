<template>
  <div class="websites-container">
    <!-- 工具栏 -->
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

    <!-- 网站列表 -->
    <el-card class="list-card">
      <el-table :data="websites" style="width: 100%" v-loading="loading">
        <el-table-column prop="domain" label="域名" width="200" />
        <el-table-column prop="backend_url" label="后端地址" width="200" />
        <el-table-column label="SSL证书" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.ssl_enabled" type="success">已启用</el-tag>
            <el-tag v-else type="info">未启用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.enabled" type="success">运行中</el-tag>
            <el-tag v-else type="danger">已停止</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
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

    <!-- 创建网站对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建网站" width="600px">
      <el-form :model="websiteForm" label-width="120px">
        <el-form-item label="域名">
          <el-input v-model="websiteForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="后端地址">
          <el-input v-model="websiteForm.backend_url" placeholder="http://localhost:3000" />
        </el-form-item>
        <el-form-item label="启用SSL">
          <el-switch v-model="websiteForm.ssl_enabled" />
        </el-form-item>
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

    <!-- SSL管理对话框 -->
    <el-dialog v-model="sslDialogVisible" title="SSL证书管理" width="600px">
      <div v-if="currentWebsite">
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
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const websites = ref([])
const loading = ref(false)
const createDialogVisible = ref(false)
const sslDialogVisible = ref(false)
const creating = ref(false)
const applyingSSL = ref(false)
const renewingSSL = ref(false)
const currentWebsite = ref(null)

const websiteForm = ref({
  domain: '',
  backend_url: '',
  ssl_enabled: false,
  load_balance: 'round_robin'
})

// 加载网站列表
const loadWebsites = async () => {
  loading.value = true
  try {
    const response = await axios.get('/api/websites')
    websites.value = response.data
  } catch (error) {
    ElMessage.error('加载网站列表失败')
  } finally {
    loading.value = false
  }
}

// 显示创建对话框
const showCreateDialog = () => {
  websiteForm.value = {
    domain: '',
    backend_url: '',
    ssl_enabled: false,
    load_balance: 'round_robin'
  }
  createDialogVisible.value = true
}

// 创建网站
const createWebsite = async () => {
  creating.value = true
  try {
    await axios.post('/api/websites', websiteForm.value)
    ElMessage.success('网站创建成功')
    createDialogVisible.value = false
    loadWebsites()
  } catch (error) {
    ElMessage.error('网站创建失败')
  } finally {
    creating.value = false
  }
}

// 切换网站状态
const toggleWebsite = async (website) => {
  try {
    await axios.put(`/api/websites/${website.id}`, {
      enabled: !website.enabled
    })
    website.enabled = !website.enabled
    ElMessage.success('状态更新成功')
  } catch (error) {
    ElMessage.error('状态更新失败')
  }
}

// 管理SSL
const manageSSL = (website) => {
  currentWebsite.value = website
  sslDialogVisible.value = true
}

// 申请SSL证书
const applySSL = async () => {
  applyingSSL.value = true
  try {
    await axios.post(`/api/websites/${currentWebsite.value.id}/ssl/apply`)
    ElMessage.success('SSL证书申请成功')
    loadWebsites()
  } catch (error) {
    ElMessage.error('SSL证书申请失败')
  } finally {
    applyingSSL.value = false
  }
}

// 续期SSL证书
const renewSSL = async () => {
  renewingSSL.value = true
  try {
    await axios.post(`/api/websites/${currentWebsite.value.id}/ssl/renew`)
    ElMessage.success('SSL证书续期成功')
    loadWebsites()
  } catch (error) {
    ElMessage.error('SSL证书续期失败')
  } finally {
    renewingSSL.value = false
  }
}

// 删除网站
const deleteWebsite = async (website) => {
  try {
    await ElMessageBox.confirm('确定要删除此网站吗？', '警告', {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    
    await axios.delete(`/api/websites/${website.id}`)
    ElMessage.success('网站删除成功')
    loadWebsites()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('网站删除失败')
    }
  }
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadWebsites()
})
</script>

<style scoped>
.websites-container {
  padding: 20px;
}

.toolbar-card {
  margin-bottom: 20px;
}

.list-card {
  margin-top: 20px;
}

.ssl-actions {
  margin-top: 20px;
  display: flex;
  gap: 10px;
}
</style>
