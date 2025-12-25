<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { resetAuthCache } from '@/router'
import { LayoutDashboard, ListTodo, FileCode, Settings, LogOut, ScrollText, Terminal, Variable, KeyRound, Package, Menu, X, RefreshCw } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { api } from '@/api'
import { useSiteSettings } from '@/composables/useSiteSettings'

const route = useRoute()
const sentence = ref('欢迎使用白虎面板')
const { siteSettings, loadSettings } = useSiteSettings()
const mobileMenuOpen = ref(false)

const navItems = [
  { to: '/', icon: LayoutDashboard, label: '数据仪表', exact: true },
  { to: '/tasks', icon: ListTodo, label: '定时任务', exact: true },
  { to: '/sync', icon: RefreshCw, label: '定时同步', exact: false },
  { to: '/editor', icon: FileCode, label: '脚本编辑', exact: false },
  { to: '/history', icon: ScrollText, label: '执行历史', exact: true },
  { to: '/environments', icon: Variable, label: '环境变量', exact: true },
  { to: '/dependencies', icon: Package, label: '依赖管理', exact: true },
  { to: '/terminal', icon: Terminal, label: '终端命令', exact: true },
  { to: '/loginlogs', icon: KeyRound, label: '登录日志', exact: true },
  { to: '/settings', icon: Settings, label: '系统设置', exact: true },
]

function isItemActive(item: (typeof navItems)[0]) {
  if (item.exact) {
    return route.path === item.to
  }
  return route.path.startsWith(item.to)
}

function handleNavClick(navigate: () => void) {
  navigate()
  mobileMenuOpen.value = false
}

async function logout() {
  try {
    await api.auth.logout()
  } catch {
    // 忽略错误
  }
  resetAuthCache()
  window.location.href = '/login'
}

async function loadSentence() {
  try {
    const res = await api.dashboard.sentence()
    sentence.value = res.sentence
  } catch {
    // 加载失败保持默认
  }
}

onMounted(() => {
  loadSettings()
  loadSentence()
})
</script>

<template>
  <div class="flex h-screen bg-muted/40">
    <!-- Mobile Menu Overlay -->
    <div
      v-if="mobileMenuOpen"
      class="fixed inset-0 bg-black/50 z-40 lg:hidden"
      @click="mobileMenuOpen = false"
    />

    <!-- Sidebar -->
    <aside
      :class="[
        'fixed lg:static inset-y-0 left-0 z-50 w-44 border-r bg-background flex flex-col transform transition-transform duration-200 ease-in-out lg:transform-none',
        mobileMenuOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
      ]"
    >
      <div class="h-14 flex items-center justify-center px-4 font-semibold text-lg border-b relative">
        <span>{{ siteSettings.title }}</span>
        <Button variant="ghost" size="icon" class="h-8 w-8 lg:hidden absolute right-2" @click="mobileMenuOpen = false">
          <X class="h-4 w-4" />
        </Button>
      </div>
      <nav class="flex-1 px-3 py-6 space-y-1 flex flex-col items-center">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          custom
          v-slot="{ navigate }"
        >
          <Button
            variant="ghost"
            :class="['justify-start gap-3 h-9 px-3', isItemActive(item) && 'bg-accent text-accent-foreground']"
            @click="handleNavClick(navigate)"
          >
            <component :is="item.icon" class="h-4 w-4" />
            {{ item.label }}
          </Button>
        </RouterLink>
      </nav>
      <div class="px-3 py-4 border-t flex justify-center">
        <Button variant="ghost" class="justify-start gap-3 h-9 px-3 text-muted-foreground hover:text-foreground" @click="logout">
          <LogOut class="h-4 w-4" />
          退出登录
        </Button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-auto w-full">
      <div class="h-14 border-b bg-background flex items-center justify-between px-4 lg:px-6">
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" class="h-8 w-8 lg:hidden" @click="mobileMenuOpen = true">
            <Menu class="h-5 w-5" />
          </Button>
          <span class="text-sm text-muted-foreground truncate max-w-[200px] sm:max-w-none">{{ sentence }}</span>
        </div>
        <ThemeToggle />
      </div>
      <div class="p-4 lg:p-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>
