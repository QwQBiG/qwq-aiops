import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue')
  },
  {
    path: '/appstore',
    name: 'AppStore',
    component: () => import('../views/AppStore.vue')
  },
  {
    path: '/containers',
    name: 'Containers',
    component: () => import('../views/Containers.vue')
  },
  {
    path: '/websites',
    name: 'Websites',
    component: () => import('../views/Websites.vue')
  },
  {
    path: '/databases',
    name: 'Databases',
    component: () => import('../views/Databases.vue')
  },
  {
    path: '/monitoring',
    name: 'Monitoring',
    component: () => import('../views/Monitoring.vue')
  },
  {
    path: '/users',
    name: 'Users',
    component: () => import('../views/Users.vue')
  },
  {
    path: '/terminal',
    name: 'Terminal',
    component: () => import('../views/Terminal.vue')
  },
  {
    path: '/files',
    name: 'Files',
    component: () => import('../views/Files.vue')
  },
  {
    path: '/logs',
    name: 'Logs',
    component: () => import('../views/Logs.vue')
  }
]

const router = createRouter({
  history: createWebHistory('/'),
  routes
})

// 路由错误处理
router.onError((error) => {
  console.error('Router Error:', error)
})

export default router
