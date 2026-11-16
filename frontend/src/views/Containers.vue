<template>
  <el-card shadow="never">
    <template #header>
      <div class="toolbar"><span>{{ t('menu.containers') }}</span><el-button :loading="loading" :disabled="!!pending" @click="fetchContainers()">{{ t('common.refresh') }}</el-button></div>
    </template>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <el-table :data="containers" v-loading="loading" :empty-text="t('console.empty')">
      <el-table-column prop="name" :label="t('console.name')" min-width="160" />
      <el-table-column prop="image" :label="t('containers.image')" show-overflow-tooltip />
      <el-table-column prop="status" :label="t('console.status')" min-width="160" />
      <el-table-column :label="t('containers.state')" width="150">
        <template #default="{ row }"><el-tag :type="row.state === 'running' ? 'success' : 'info'">{{ row.state }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="t('console.actions')" width="210" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.state !== 'running'" link type="success" :disabled="loading || !!pending" @click="handleAction(row, 'start')">{{ t('containers.start') }}</el-button>
          <el-button v-else link type="danger" :disabled="loading || !!pending" @click="handleAction(row, 'stop')">{{ t('containers.stop') }}</el-button>
          <el-button link type="primary" :loading="pending === row.id" :disabled="loading || !!pending" @click="handleAction(row, 'restart')">{{ t('containers.restart') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { errorMessage, isCancellation } from '../lib/errors'
const { t } = useI18n()
const containers = ref([])
const loading = ref(false)
const pending = ref('')
const error = ref('')
const controller = new AbortController()

async function fetchContainers(afterAction = false) {
  if (loading.value || (pending.value && !afterAction)) return
  loading.value = true
  try {
    const { data } = await axios.get('/api/containers', { signal: controller.signal })
    containers.value = Array.isArray(data) ? data : []
    error.value = ''
  } catch (failure) {
    if (!isCancellation(failure)) error.value = errorMessage(failure, t('console.requestFailed'))
  } finally {
    loading.value = false
  }
}

async function handleAction(row, action) {
  if (pending.value || loading.value) return
  pending.value = row.id
  try {
    if (action !== 'start') await ElMessageBox.confirm(t('containers.confirmAction', { action: t(`containers.${action}`), name: row.name }), t('common.warning'), { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') })
    await axios.post('/api/container/action', null, { params: { id: row.id, action }, signal: controller.signal })
    ElMessage.success(t('console.saved'))
    await fetchContainers(true)
  } catch (failure) {
    if (!isCancellation(failure)) ElMessage.error(errorMessage(failure, t('console.requestFailed')))
  } finally {
    pending.value = ''
  }
}
onBeforeRouteLeave(() => !pending.value)
onMounted(fetchContainers)
onUnmounted(() => controller.abort())
</script>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; }
.el-alert { margin-bottom: 16px; }
</style>
