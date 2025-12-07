<template>
  <div class="users-container">
    <el-tabs v-model="activeTab">
      <!-- 用户管理 -->
      <el-tab-pane label="用户管理" name="users">
        <el-button type="primary" @click="showCreateUser" style="margin-bottom: 20px">
          <el-icon><Plus /></el-icon>
          创建用户
        </el-button>

        <el-table :data="users" style="width: 100%">
          <el-table-column prop="username" label="用户名" width="150" />
          <el-table-column prop="email" label="邮箱" width="200" />
          <el-table-column label="角色" width="200">
            <template #default="{ row }">
              <el-tag v-for="role in row.roles" :key="role" style="margin-right: 5px">
                {{ role }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.enabled" type="success">启用</el-tag>
              <el-tag v-else type="danger">禁用</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="250">
            <template #default="{ row }">
              <el-button size="small" @click="editUser(row)">编辑</el-button>
              <el-button size="small" @click="managePermissions(row)">权限</el-button>
              <el-button size="small" type="danger" @click="deleteUser(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 角色管理 -->
      <el-tab-pane label="角色管理" name="roles">
        <el-button type="primary" @click="showCreateRole" style="margin-bottom: 20px">
          <el-icon><Plus /></el-icon>
          创建角色
        </el-button>

        <el-table :data="roles" style="width: 100%">
          <el-table-column prop="name" label="角色名称" width="150" />
          <el-table-column prop="description" label="描述" />
          <el-table-column label="权限数量" width="120">
            <template #default="{ row }">
              {{ row.permissions?.length || 0 }}
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="editRole(row)">编辑</el-button>
              <el-button size="small" type="danger" @click="deleteRole(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 权限列表 -->
      <el-tab-pane label="权限列表" name="permissions">
        <el-table :data="permissions" style="width: 100%">
          <el-table-column prop="resource" label="资源" width="200" />
          <el-table-column prop="action" label="操作" width="150" />
          <el-table-column prop="description" label="描述" />
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 创建/编辑用户对话框 -->
    <el-dialog v-model="userDialogVisible" :title="editingUser ? '编辑用户' : '创建用户'" width="500px">
      <el-form :model="userForm" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="userForm.username" :disabled="!!editingUser" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="userForm.email" type="email" />
        </el-form-item>
        <el-form-item label="密码" v-if="!editingUser">
          <el-input v-model="userForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userForm.roles" multiple style="width: 100%">
            <el-option
              v-for="role in roles"
              :key="role.id"
              :label="role.name"
              :value="role.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="userForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUser" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 创建/编辑角色对话框 -->
    <el-dialog v-model="roleDialogVisible" :title="editingRole ? '编辑角色' : '创建角色'" width="600px">
      <el-form :model="roleForm" label-width="100px">
        <el-form-item label="角色名称">
          <el-input v-model="roleForm.name" :disabled="!!editingRole" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="roleForm.description" type="textarea" />
        </el-form-item>
        <el-form-item label="权限">
          <el-transfer
            v-model="roleForm.permissions"
            :data="permissionOptions"
            :titles="['可用权限', '已选权限']"
            filterable
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRole" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 用户权限管理对话框 -->
    <el-dialog v-model="permissionDialogVisible" title="管理用户权限" width="600px">
      <div v-if="currentUser">
        <h4>{{ currentUser.username }} 的权限</h4>
        <el-tree
          :data="permissionTree"
          show-checkbox
          node-key="id"
          :default-checked-keys="currentUserPermissions"
          @check="handlePermissionCheck"
        />
      </div>
      <template #footer>
        <el-button @click="permissionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUserPermissions" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('users')
const users = ref([])
const roles = ref([])
const permissions = ref([])
const userDialogVisible = ref(false)
const roleDialogVisible = ref(false)
const permissionDialogVisible = ref(false)
const saving = ref(false)
const editingUser = ref(null)
const editingRole = ref(null)
const currentUser = ref(null)
const currentUserPermissions = ref([])

const userForm = ref({
  username: '',
  email: '',
  password: '',
  roles: [],
  enabled: true
})

const roleForm = ref({
  name: '',
  description: '',
  permissions: []
})

// 权限选项（用于穿梭框）
const permissionOptions = computed(() => {
  return permissions.value.map(p => ({
    key: `${p.resource}:${p.action}`,
    label: `${p.resource} - ${p.action}`,
    disabled: false
  }))
})

// 权限树（用于树形选择）
const permissionTree = computed(() => {
  const tree = []
  const resourceMap = {}
  
  permissions.value.forEach(p => {
    if (!resourceMap[p.resource]) {
      resourceMap[p.resource] = {
        id: p.resource,
        label: p.resource,
        children: []
      }
      tree.push(resourceMap[p.resource])
    }
    resourceMap[p.resource].children.push({
      id: `${p.resource}:${p.action}`,
      label: p.action
    })
  })
  
  return tree
})

// 加载用户列表
const loadUsers = async () => {
  try {
    const response = await axios.get('/api/users')
    users.value = response.data
  } catch (error) {
    ElMessage.error('加载用户列表失败')
  }
}

// 加载角色列表
const loadRoles = async () => {
  try {
    const response = await axios.get('/api/roles')
    roles.value = response.data
  } catch (error) {
    ElMessage.error('加载角色列表失败')
  }
}

// 加载权限列表
const loadPermissions = async () => {
  try {
    const response = await axios.get('/api/permissions')
    permissions.value = response.data
  } catch (error) {
    ElMessage.error('加载权限列表失败')
  }
}

// 显示创建用户对话框
const showCreateUser = () => {
  editingUser.value = null
  userForm.value = {
    username: '',
    email: '',
    password: '',
    roles: [],
    enabled: true
  }
  userDialogVisible.value = true
}

// 编辑用户
const editUser = (user) => {
  editingUser.value = user
  userForm.value = {
    username: user.username,
    email: user.email,
    roles: user.roles || [],
    enabled: user.enabled
  }
  userDialogVisible.value = true
}

// 保存用户
const saveUser = async () => {
  saving.value = true
  try {
    if (editingUser.value) {
      await axios.put(`/api/users/${editingUser.value.id}`, userForm.value)
      ElMessage.success('用户更新成功')
    } else {
      await axios.post('/api/users', userForm.value)
      ElMessage.success('用户创建成功')
    }
    userDialogVisible.value = false
    loadUsers()
  } catch (error) {
    ElMessage.error(editingUser.value ? '用户更新失败' : '用户创建失败')
  } finally {
    saving.value = false
  }
}

// 删除用户
const deleteUser = async (user) => {
  try {
    await ElMessageBox.confirm('确定要删除此用户吗？', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await axios.delete(`/api/users/${user.id}`)
    ElMessage.success('用户删除成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('用户删除失败')
    }
  }
}

// 管理用户权限
const managePermissions = async (user) => {
  currentUser.value = user
  try {
    const response = await axios.get(`/api/users/${user.id}/permissions`)
    currentUserPermissions.value = response.data.map(p => `${p.resource}:${p.action}`)
    permissionDialogVisible.value = true
  } catch (error) {
    ElMessage.error('加载用户权限失败')
  }
}

// 处理权限选择
const handlePermissionCheck = (data, checked) => {
  // 权限选择逻辑
}

// 保存用户权限
const saveUserPermissions = async () => {
  saving.value = true
  try {
    await axios.put(`/api/users/${currentUser.value.id}/permissions`, {
      permissions: currentUserPermissions.value
    })
    ElMessage.success('权限更新成功')
    permissionDialogVisible.value = false
  } catch (error) {
    ElMessage.error('权限更新失败')
  } finally {
    saving.value = false
  }
}

// 显示创建角色对话框
const showCreateRole = () => {
  editingRole.value = null
  roleForm.value = {
    name: '',
    description: '',
    permissions: []
  }
  roleDialogVisible.value = true
}

// 编辑角色
const editRole = (role) => {
  editingRole.value = role
  roleForm.value = {
    name: role.name,
    description: role.description,
    permissions: role.permissions || []
  }
  roleDialogVisible.value = true
}

// 保存角色
const saveRole = async () => {
  saving.value = true
  try {
    if (editingRole.value) {
      await axios.put(`/api/roles/${editingRole.value.id}`, roleForm.value)
      ElMessage.success('角色更新成功')
    } else {
      await axios.post('/api/roles', roleForm.value)
      ElMessage.success('角色创建成功')
    }
    roleDialogVisible.value = false
    loadRoles()
  } catch (error) {
    ElMessage.error(editingRole.value ? '角色更新失败' : '角色创建失败')
  } finally {
    saving.value = false
  }
}

// 删除角色
const deleteRole = async (role) => {
  try {
    await ElMessageBox.confirm('确定要删除此角色吗？', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await axios.delete(`/api/roles/${role.id}`)
    ElMessage.success('角色删除成功')
    loadRoles()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('角色删除失败')
    }
  }
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadUsers()
  loadRoles()
  loadPermissions()
})
</script>

<style scoped>
.users-container {
  padding: 20px;
}
</style>
