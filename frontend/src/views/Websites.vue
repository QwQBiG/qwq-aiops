<template>
  <section>
    <el-alert :title="t('records.memoryNotice')" type="warning" :closable="false" show-icon />
    <p class="notice">{{ t('records.websiteNotice') }}</p>
    <el-card shadow="never">
      <template #header><div class="toolbar"><el-button type="primary" :disabled="busy || loading" @click="openCreate">{{ t('website.create') }}</el-button><el-button :loading="loading" :disabled="busy" @click="refresh">{{ t('common.refresh') }}</el-button></div></template>
      <el-table :data="websites" v-loading="loading" :empty-text="t('console.empty')">
        <el-table-column prop="domain" :label="t('website.domain')" min-width="180" />
        <el-table-column prop="backend_url" :label="t('websiteConfig.backend')" min-width="200" show-overflow-tooltip />
        <el-table-column :label="t('websiteConfig.enabled')" width="160"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ t(row.enabled ? 'console.enabled' : 'console.disabled') }}</el-tag></template></el-table-column>
        <el-table-column :label="t('console.createdAt')" min-width="180"><template #default="{ row }">{{ date(row.created_at) }}</template></el-table-column>
        <el-table-column :label="t('console.actions')" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link :disabled="busy || loading" @click="toggle(row)">{{ t(row.enabled ? 'website.disable' : 'website.enable') }}</el-button>
            <el-tooltip :content="t('websiteConfig.sslUnavailable')"><span><el-button link disabled>{{ t('website.ssl') }}</el-button></span></el-tooltip>
            <el-button link type="danger" :disabled="busy || loading" @click="remove(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-dialog v-model="visible" :title="t('website.create')" width="560px" :show-close="!saving" :close-on-click-modal="!saving" :close-on-press-escape="!saving">
      <el-form :model="form" :disabled="saving" label-position="top" @submit.prevent="create">
        <el-form-item :label="t('website.domain')" required><el-input v-model="form.domain" placeholder="example.com" /></el-form-item>
        <el-form-item :label="t('websiteConfig.backend')" required><el-input v-model="form.backend_url" placeholder="http://localhost:3000" /></el-form-item>
        <el-form-item :label="t('websiteConfig.balance')"><el-select v-model="form.load_balance"><el-option v-for="strategy in strategies" :key="strategy[0]" :value="strategy[0]" :label="t(`websiteConfig.${strategy[1]}`)" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button :disabled="saving" @click="visible = false">{{ t('common.cancel') }}</el-button><el-button type="primary" :loading="saving" @click="create">{{ t('common.create') }}</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { errorMessage, isCancellation } from '../lib/errors'
const { t, locale } = useI18n()
const websites = ref([]), loading = ref(false), visible = ref(false), saving = ref(false)
const pending = ref(false)
const busy = computed(() => saving.value || pending.value)
const form = ref({})
const strategies = [['round_robin', 'roundRobin'], ['least_conn', 'leastConnections'], ['ip_hash', 'ipHash']]
const controller = new AbortController()
const options = { signal: controller.signal }
const report = error => { if (!isCancellation(error)) ElMessage.error(errorMessage(error, t('console.requestFailed'))) }
const date = value => value ? new Date(value).toLocaleString(locale.value) : '—'
function openCreate() {
  if (busy.value || loading.value) return
  form.value = { domain: '', backend_url: '', ssl_enabled: false, load_balance: 'round_robin' }
  visible.value = true
}
async function load() {
  if (loading.value) return
  loading.value = true
  try { const { data } = await axios.get('/api/websites', options); websites.value = Array.isArray(data) ? data : [] }
  catch (error) { report(error) }
  finally { loading.value = false }
}
function refresh() {
  if (!busy.value) return load()
}
async function create() {
  if (busy.value || loading.value) return
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
  if (busy.value || loading.value) return
  pending.value = true
  try { await axios.put(`/api/websites/${row.id}`, { enabled: !row.enabled }, options); await load() }
  catch (error) { report(error) }
  finally { pending.value = false }
}
async function remove(row) {
  if (busy.value || loading.value) return
  pending.value = true
  try {
    await ElMessageBox.confirm(t('console.confirmDelete', { name: row.domain }), t('common.warning'), { confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel'), type: 'warning' })
    await axios.delete(`/api/websites/${row.id}`, options)
    ElMessage.success(t('console.deleted'))
    await load()
  } catch (error) { report(error) }
  finally { pending.value = false }
}
onBeforeRouteLeave(() => !busy.value)
onMounted(load)
onUnmounted(() => controller.abort())
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; }
.notice { color: var(--el-text-color-secondary); line-height: 1.7; }
</style>
