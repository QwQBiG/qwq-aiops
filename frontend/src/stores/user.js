import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  const userInfo = ref(null)
  const token = ref(localStorage.getItem('token') || '')
  const permissions = ref([])

  const setToken = (newToken) => {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  const setUserInfo = (info) => {
    userInfo.value = info
    permissions.value = info.permissions || []
  }

  const logout = () => {
    token.value = ''
    userInfo.value = null
    permissions.value = []
    localStorage.removeItem('token')
  }

  const hasPermission = (permission) => {
    return permissions.value.includes(permission) || permissions.value.includes('*')
  }

  return {
    userInfo,
    token,
    permissions,
    setToken,
    setUserInfo,
    logout,
    hasPermission
  }
})
