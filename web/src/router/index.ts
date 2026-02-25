import { createRouter, createWebHistory } from 'vue-router'
import { checkAuth } from '@/api'

// 获取 base URL（从后端注入的全局变量）
const BASE_URL = (window as any).__BASE_URL__ || ''

// 缓存认证状态，避免每次路由跳转都请求
let authChecked = false
let isAuth = false

async function getAuthStatus(force = false): Promise<boolean> {
  if (!force && authChecked) {
    return isAuth
  }
  isAuth = await checkAuth()
  authChecked = true
  return isAuth
}

// 重置认证状态（登录/登出时调用）
export function resetAuthCache() {
  authChecked = false
  isAuth = false
}

const router = createRouter({
  history: createWebHistory(BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/login/Login.vue'),
      meta: { guest: true }
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'dashboard', component: () => import('@/views/dashboard/Dashboard.vue') },
        { path: 'tasks', name: 'tasks', component: () => import('@/views/tasks/Tasks.vue') },
        { path: 'editor', name: 'editor', component: () => import('@/views/editor/Editor.vue') },
        { path: 'environments', name: 'environments', component: () => import('@/views/environments/Environments.vue') },
        { path: 'dependencies', name: 'dependencies', component: () => import('@/views/dependencies/Dependencies.vue') },
        { path: 'languages', name: 'languages', component: () => import('@/views/languages/Languages.vue') },
        { path: 'agents', name: 'agents', component: () => import('@/views/agents/Agents.vue') },
        { path: 'history', name: 'history', component: () => import('@/views/history/History.vue') },
        { path: 'loginlogs', name: 'loginlogs', component: () => import('@/views/loginlogs/LoginLogs.vue') },
        { path: 'exim', name: 'exim', component: () => import('@/views/exim/Exim.vue') },
        { path: 'terminal', name: 'terminal', component: () => import('@/views/terminal/Terminal.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/settings/Settings.vue') }
      ]
    }
  ]
})

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  // 首次访问或访问登录页时强制检查
  const forceCheck = !authChecked || to.path === '/login'
  const isAuthenticated = await getAuthStatus(forceCheck)

  // 检查是否需要认证
  if (to.matched.some(record => record.meta.requiresAuth)) {
    if (isAuthenticated) {
      next()
    } else {
      next('/login')
    }
  } else if (to.matched.some(record => record.meta.guest)) {
    // 已登录用户访问登录页，跳转到首页
    if (isAuthenticated) {
      next('/')
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
