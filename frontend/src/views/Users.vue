<template>
  <section>
    <el-alert :title="t('records.memoryNotice')" type="warning" :closable="false" show-icon />
    <p class="notice">{{ t('records.authNotice') }}</p>
    <el-tabs v-model="activeTab">
      <el-tab-pane :label="t('records.users')" name="users">
        <el-button type="primary" @click="editUser()">{{ t('records.createUser') }}</el-button>
        <el-table :data="users" :empty-text="t('console.empty')" v-loading="loading">
          <el-table-column prop="username" :label="t('user.username')" min-width="150" />
          <el-table-column prop="email" :label="t('records.email')" min-width="180" />
          <el-table-column :label="t('user.role')"><template #default="{ row }">{{ row.roles?.join(', ') || '—' }}</template></el-table-column>
          <el-table-column :label="t('console.status')" width="130"><template #default="{ row }">{{ t(row.enabled ? 'console.enabled' : 'console.disabled') }}</template></el-table-column>
          <el-table-column :label="t('console.actions')" width="270">
            <template #default="{ row }">
              <el-button link @click="editUser(row)">{{ t('common.edit') }}</el-button>
              <el-tooltip :content="t('records.permissionInfo')"><span><el-button link disabled>{{ t('user.permissions') }}</el-button></span></el-tooltip>
              <el-button link type="danger" @click="remove('users', row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane :label="t('records.roles')" name="roles">
        <el-button type="primary" @click="editRole()">{{ t('records.createRole') }}</el-button>
        <el-table :data="roles" :empty-text="t('console.empty')" v-loading="loading">
          <el-table-column prop="name" :label="t('records.roleName')" />
          <el-table-column prop="description" :label="t('console.description')" />
          <el-table-column :label="t('console.actions')" width="180"><template #default="{ row }"><el-button link @click="editRole(row)">{{ t('common.edit') }}</el-button><el-button link type="danger" @click="remove('roles', row)">{{ t('common.delete') }}</el-button></template></el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane :label="t('records.permissions')" name="permissions">
        <el-alert :title="t('records.permissionInfo')" type="info" :closable="false" />
        <el-table :data="permissions"><el-table-column prop="resource" :label="t('console.name')" /><el-table-column prop="action" :label="t('console.actions')" /><el-table-column prop="description" :label="t('console.description')" /></el-table>
      </el-tab-pane>
    </el-tabs>
    <el-dialog v-model="userVisible" :title="t(userId ? 'records.editUser' : 'records.createUser')" width="520px">
      <el-form :model="userForm" label-position="top" @submit.prevent="saveUser">
        <el-form-item :label="t('user.username')" required><el-input v-model="userForm.username" :disabled="!!userId" /></el-form-item>
        <el-form-item :label="t('records.email')" required><el-input v-model="userForm.email" type="email" /></el-form-item>
        <el-form-item :label="t('user.role')"><el-select v-model="userForm.roles" multiple><el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.name" /></el-select></el-form-item>
        <el-form-item :label="t('console.enabled')"><el-switch v-model="userForm.enabled" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="userVisible = false">{{ t('common.cancel') }}</el-button><el-button type="primary" :loading="saving" @click="saveUser">{{ t('common.save') }}</el-button></template>
    </el-dialog>
    <el-dialog v-model="roleVisible" :title="t(roleId ? 'records.editRole' : 'records.createRole')" width="520px">
      <el-form :model="roleForm" label-position="top" @submit.prevent="saveRole">
        <el-form-item :label="t('records.roleName')" required><el-input v-model="roleForm.name" :disabled="!!roleId" /></el-form-item>
        <el-form-item :label="t('console.description')"><el-input v-model="roleForm.description" type="textarea" /></el-form-item>
      </el-form>
      <el-alert :title="t('records.permissionInfo')" type="info" :closable="false" />
      <template #footer><el-button @click="roleVisible = false">{{ t('common.cancel') }}</el-button><el-button type="primary" :loading="saving" @click="saveRole">{{ t('common.save') }}</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { errorMessage, isCancellation } from '../lib/errors'
const { t } = useI18n()
const activeTab = ref('users'), users = ref([]), roles = ref([]), permissions = ref([]), loading = ref(false), saving = ref(false)
const userVisible = ref(false), roleVisible = ref(false), userId = ref(null), roleId = ref(null), userForm = ref({}), roleForm = ref({})
const controller = new AbortController()
const options = { signal: controller.signal }
const report = error => { if (!isCancellation(error)) ElMessage.error(errorMessage(error, t('console.requestFailed'))) }
async function load() {
  loading.value = true
  try {
    const results = await Promise.all(['users', 'roles', 'permissions'].map(name => axios.get(`/api/${name}`, options)))
    for (const [index, target] of [users, roles, permissions].entries()) target.value = Array.isArray(results[index].data) ? results[index].data : []
  } catch (error) { report(error) }
  finally { loading.value = false }
}
function editUser(row) {
  userId.value = row?.id || null
  userForm.value = { username: row?.username || '', email: row?.email || '', roles: [...(row?.roles || [])], enabled: row?.enabled ?? true }
  userVisible.value = true
}
function editRole(row) {
  roleId.value = row?.id || null
  roleForm.value = { name: row?.name || '', description: row?.description || '', permissions: [...(row?.permissions || [])] }
  roleVisible.value = true
}
async function save(kind, id, form, visible) {
  saving.value = true
  try {
    if (id) await axios.put(`/api/${kind}/${id}`, form, options)
    else await axios.post(`/api/${kind}`, form, options)
    visible.value = false
    ElMessage.success(t('console.saved'))
    await load()
  } catch (error) { report(error) }
  finally { saving.value = false }
}
function saveUser() {
  if (!userForm.value.username.trim() || !userForm.value.email.trim()) return ElMessage.warning(t('records.requiredName'))
  return save('users', userId.value, userForm.value, userVisible)
}
function saveRole() {
  if (!roleForm.value.name.trim()) return ElMessage.warning(t('records.requiredName'))
  return save('roles', roleId.value, roleForm.value, roleVisible)
}
async function remove(kind, row) {
  try {
    await ElMessageBox.confirm(t('console.confirmDelete', { name: row.username || row.name }), t('common.warning'), { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') })
    await axios.delete(`/api/${kind}/${row.id}`, options)
    ElMessage.success(t('console.deleted'))
    await load()
  } catch (error) { report(error) }
}
onMounted(load)
onUnmounted(() => controller.abort())
</script>

<style scoped>
.notice { color: var(--el-text-color-secondary); line-height: 1.7; }
.el-table { margin-top: 16px; }
</style>
