<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { LayoutDashboard, ListTodo, FileCode, Settings, LogOut, ScrollText, Terminal, Variable, KeyRound } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { api } from '@/api'
import { useSiteSettings } from '@/composables/useSiteSettings'

const route = useRoute()
const sentence = ref('欢迎使用白虎面板')
const { siteSettings, loadSettings } = useSiteSettings()

const navItems = [
  { to: '/', icon: LayoutDashboard, label: '数据仪表', exact: true },
  { to: '/tasks', icon: ListTodo, label: '定时任务', exact: true },
  { to: '/editor', icon: FileCode, label: '脚本编辑', exact: false },
  { to: '/history', icon: ScrollText, label: '执行历史', exact: true },
  { to: '/environments', icon: Variable, label: '环境变量', exact: true },
  { to: '/terminal', icon: Terminal, label: '终端命令', exact: true },
  { to: '/login-logs', icon: KeyRound, label: '登录日志', exact: true },
  { to: '/settings', icon: Settings, label: '系统设置', exact: true },
]

function isItemActive(item: (typeof navItems)[0]) {
  if (item.exact) {
    return route.path === item.to
  }
  return route.path.startsWith(item.to)
}

async function logout() {
  try {
    await api.auth.logout()
  } catch {
    // 忽略错误
  }
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
    <!-- Sidebar -->
    <aside class="w-56 border-r bg-background flex flex-col">
      <div class="h-14 flex items-center px-10 font-semibold text-lg border-b">
        {{ siteSettings.title }}
      </div>
      <nav class="flex-1 px-6 py-9 space-y-1">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          custom
          v-slot="{ navigate }"
        >
          <Button
            variant="ghost"
            :class="['w-[80%] justify-start gap-3 h-9 px-3', isItemActive(item) && 'bg-accent text-accent-foreground']"
            @click="navigate"
          >
            <component :is="item.icon" class="h-4 w-4" />
            {{ item.label }}
          </Button>
        </RouterLink>
      </nav>
      <div class="px-3 py-4 border-t">
        <Button variant="ghost" class="w-full justify-start gap-3 h-9 px-3 text-muted-foreground hover:text-foreground" @click="logout">
          <LogOut class="h-4 w-4" />
          退出登录
        </Button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-auto">
      <div class="h-14 border-b bg-background flex items-center justify-between px-6">
        <span class="text-sm text-muted-foreground">{{ sentence }}</span>
        <ThemeToggle />
      </div>
      <div class="p-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>
