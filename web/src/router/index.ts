import { createRouter, createWebHistory } from 'vue-router'
import { checkAuth } from '@/api'

const router = createRouter({
  history: createWebHistory(),
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
        { path: 'editor/:path(.*)?', name: 'editor', component: () => import('@/views/editor/Editor.vue') },
        { path: 'environments', name: 'environments', component: () => import('@/views/environments/Environments.vue') },
        { path: 'history', name: 'history', component: () => import('@/views/history/History.vue') },
        { path: 'login-logs', name: 'login-logs', component: () => import('@/views/login-logs/LoginLogs.vue') },
        { path: 'terminal', name: 'terminal', component: () => import('@/views/terminal/Terminal.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/settings/Settings.vue') }
      ]
    }
  ]
})

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  const isAuthenticated = await checkAuth()
  
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
