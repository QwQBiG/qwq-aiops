<template>
  <section>
    <el-alert :title="t('records.memoryNotice')" type="warning" :closable="false" show-icon />
    <p class="notice">{{ t('records.websiteNotice') }}</p>
    <el-card shadow="never">
      <template #header><div class="toolbar"><el-button type="primary" @click="openCreate">{{ t('website.create') }}</el-button><el-button :loading="loading" @click="load">{{ t('common.refresh') }}</el-button></div></template>
      <el-table :data="websites" v-loading="loading" :empty-text="t('console.empty')">
        <el-table-column prop="domain" :label="t('website.domain')" min-width="180" />
        <el-table-column prop="backend_url" :label="t('websiteConfig.backend')" min-width="200" show-overflow-tooltip />
        <el-table-column :label="t('websiteConfig.enabled')" width="160"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ t(row.enabled ? 'console.enabled' : 'console.disabled') }}</el-tag></template></el-table-column>
        <el-table-column :label="t('console.createdAt')" min-width="180"><template #default="{ row }">{{ date(row.created_at) }}</template></el-table-column>
        <el-table-column :label="t('console.actions')" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link @click="toggle(row)">{{ t(row.enabled ? 'website.disable' : 'website.enable') }}</el-button>
            <el-tooltip :content="t('websiteConfig.sslUnavailable')"><span><el-button link disabled>{{ t('website.ssl') }}</el-button></span></el-tooltip>
            <el-button link type="danger" @click="remove(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-dialog v-model="visible" :title="t('website.create')" width="560px">
      <el-form :model="form" label-position="top" @submit.prevent="create">
        <el-form-item :label="t('website.domain')" required><el-input v-model="form.domain" placeholder="example.com" /></el-form-item>
        <el-form-item :label="t('websiteConfig.backend')" required><el-input v-model="form.backend_url" placeholder="http://localhost:3000" /></el-form-item>
        <el-form-item :label="t('websiteConfig.balance')"><el-select v-model="form.load_balance"><el-option v-for="strategy in strategies" :key="strategy[0]" :value="strategy[0]" :label="t(`websiteConfig.${strategy[1]}`)" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="visible = false">{{ t('common.cancel') }}</el-button><el-button type="primary" :loading="saving" @click="create">{{ t('common.create') }}</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { errorMessage, isCancellation } from '../lib/errors'
const { t, locale } = useI18n()
const websites = ref([]), loading = ref(false), visible = ref(false), saving = ref(false)
const form = ref({})
const strategies = [['round_robin', 'roundRobin'], ['least_conn', 'leastConnections'], ['ip_hash', 'ipHash']]
const controller = new AbortController()
const options = { signal: controller.signal }
const report = error => { if (!isCancellation(error)) ElMessage.error(errorMessage(error, t('console.requestFailed'))) }
const date = value => value ? new Date(value).toLocaleString(locale.value) : '—'
function openCreate() {
  form.value = { domain: '', backend_url: '', ssl_enabled: false, load_balance: 'round_robin' }
  visible.value = true
}
async function load() {
  loading.value = true
  try { const { data } = await axios.get('/api/websites', options); websites.value = Array.isArray(data) ? data : [] }
  catch (error) { report(error) }
  finally { loading.value = false }
}
async function create() {
  if (!form.value.domain.trim() || !form.value.backend_url.trim()) return ElMessage.warning(t('websiteConfig.requiredFields'))
  saving.value = true
  try {
    await axios.post('/api/websites', form.value, options)
    visible.value = false
    ElMessage.success(t('console.created'))
    await load()
  } catch (error) { report(error) }
  finally { saving.value = false }
}
async function toggle(row) {
  try { await axios.put(`/api/websites/${row.id}`, { enabled: !row.enabled }, options); await load() }
  catch (error) { report(error) }
}
async function remove(row) {
  try {
    await ElMessageBox.confirm(t('console.confirmDelete', { name: row.domain }), t('common.warning'), { confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel'), type: 'warning' })
    await axios.delete(`/api/websites/${row.id}`, options)
    ElMessage.success(t('console.deleted'))
    await load()
  } catch (error) { report(error) }
}
onMounted(load)
onUnmounted(() => controller.abort())
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; }
.notice { color: var(--el-text-color-secondary); line-height: 1.7; }
</style>
