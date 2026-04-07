<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { resetAuthCache } from '@/router'
import { LayoutDashboard, ListTodo, FileCode, Settings, LogOut, ScrollText, Terminal, Variable, KeyRound, Menu, X, Server, Globe, Bell } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import ThemeToggle from '@/components/ThemeToggle.vue'
import SystemNotice from '@/components/SystemNotice.vue'
import { api } from '@/api'
import { useSiteSettings } from '@/composables/useSiteSettings'

const SENTENCE_CACHE_KEY = 'sentence_cache'
const SENTENCE_CACHE_TIME_KEY = 'sentence_cache_time'
const CACHE_DURATION = 24 * 60 * 60 * 1000 // 24小时

// 从 localStorage 加载缓存的诗句
function loadSentenceFromCache(): string | null {
  try {
    const cached = localStorage.getItem(SENTENCE_CACHE_KEY)
    const cacheTime = localStorage.getItem(SENTENCE_CACHE_TIME_KEY)

    if (cached && cacheTime) {
      const age = Date.now() - parseInt(cacheTime)
      // 如果缓存未过期，使用缓存
      if (age < CACHE_DURATION) {
        return cached
      }
    }
  } catch {
    // 忽略错误
  }
  return null
}

// 保存诗句到 localStorage
function saveSentenceToCache(sentence: string) {
  try {
    localStorage.setItem(SENTENCE_CACHE_KEY, sentence)
    localStorage.setItem(SENTENCE_CACHE_TIME_KEY, Date.now().toString())
  } catch {
    // 忽略存储错误
  }
}

const route = useRoute()
const cachedSentence = loadSentenceFromCache()
const sentence = ref(cachedSentence || '欢迎使用白虎面板')
const { siteSettings, loadSettings } = useSiteSettings()
const mobileMenuOpen = ref(false)
const sentenceContent = computed(() => {
  const match = sentence.value.match(/^"(.+)"—— /)
  return match ? match[1] : sentence.value
})

const navItems = [
  { to: '/', icon: LayoutDashboard, label: '数据仪表', exact: true },
  { to: '/tasks', icon: ListTodo, label: '定时任务', exact: true },
  { to: '/agents', icon: Server, label: '远程执行', exact: true },
  { to: '/editor', icon: FileCode, label: '脚本编辑', exact: false },
  { to: '/history', icon: ScrollText, label: '执行历史', exact: true },
  { to: '/environments', icon: Variable, label: '变量机密', exact: true },
  { to: '/languages', icon: Globe, label: '语言依赖', exact: true },
  { to: '/terminal', icon: Terminal, label: '终端命令', exact: true },
  { to: '/notify', icon: Bell, label: '消息推送', exact: true },
  { to: '/logs', icon: KeyRound, label: '消息日志', exact: true },
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
    saveSentenceToCache(res.sentence) // 保存到缓存
  } catch {
    // 加载失败保持默认或缓存值
  }
}

onMounted(() => {
  loadSettings()
  loadSentence() // 后台更新诗句
})
</script>

<template>
  <!-- Root Container: Handles the background and centering on 2K+ screens -->
  <div class="h-screen w-full bg-slate-50/50 dark:bg-zinc-950 flex items-center justify-center 2xl:p-8 3xl:p-12 transition-all duration-500 overflow-hidden">
    
    <!-- Application Card: The main floating surface -->
    <div
      class="flex h-full w-full bg-background relative transition-all duration-500 overflow-hidden
             2xl:max-w-[1800px] 2xl:max-h-[92vh] 2xl:rounded-[1.25rem] 
             2xl:shadow-[0_50px_100px_-20px_rgba(0,0,0,0.25),0_0_0_1px_rgba(255,255,255,0.05)]
             2xl:ring-1 2xl:ring-slate-900/5 dark:2xl:ring-white/10
             2xl:border 2xl:border-slate-200/60 dark:2xl:border-white/5
             3xl:max-w-[2200px] 3xl:max-h-[90vh]">
      
      <!-- Mobile Menu Overlay -->
      <div v-if="mobileMenuOpen" class="fixed inset-0 bg-black/40 backdrop-blur-[2px] z-40 lg:hidden" @click="mobileMenuOpen = false" />
  
      <!-- Sidebar -->
      <aside :class="[
        'fixed lg:static inset-y-0 z-50 w-44 border-r bg-background flex flex-col transition-all duration-300 ease-in-out',
        mobileMenuOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
      ]">
        <div class="h-14 flex items-center justify-center px-4 font-semibold text-lg border-b border-slate-200/60 dark:border-white/10 relative">
          <span>{{ siteSettings.title }}</span>
          <Button variant="ghost" size="icon" class="h-8 w-8 lg:hidden absolute right-2" @click="mobileMenuOpen = false">
            <X class="h-4 w-4" />
          </Button>
        </div>
        <nav class="flex-1 px-3 py-6 space-y-1 flex flex-col items-center overflow-y-auto">
          <RouterLink v-for="item in navItems" :key="item.to" :to="item.to" custom v-slot="{ navigate }">
            <Button variant="ghost"
              :class="['justify-center gap-3 h-9 px-3 w-full max-w-[140px]', isItemActive(item) && 'bg-zinc-100 dark:bg-accent text-foreground dark:text-accent-foreground font-semibold shadow-sm']"
              @click="handleNavClick(navigate)">
              <component :is="item.icon" class="h-4 w-4" />
              {{ item.label }}
            </Button>
          </RouterLink>
        </nav>
        <div class="px-3 py-4 2xl:pb-12 border-t border-slate-200/60 dark:border-white/10 flex justify-center">
          <Button variant="ghost" 
            class="justify-start 2xl:justify-center gap-3 h-9 px-3 w-content 2xl:w-full 2xl:max-w-[140px] text-muted-foreground hover:text-foreground transition-all whitespace-nowrap"
            @click="logout">
            <LogOut class="h-4 w-4 shrink-0" />
            <span class="truncate">退出登录</span>
          </Button>
        </div>
      </aside>
  
      <!-- Main Content Area -->
      <main class="flex-1 flex flex-col min-w-0 relative">
        <!-- Top Navigation Bar -->
        <header class="h-14 border-b border-slate-200/60 dark:border-white/10 bg-background flex items-center justify-between px-4 lg:px-6 shrink-0 sticky top-0 z-30">
          <div class="flex items-center gap-2 sm:gap-3 flex-1 min-w-0 mr-4">
            <Button variant="ghost" size="icon" class="h-9 w-9 lg:hidden shrink-0" @click="mobileMenuOpen = true">
              <Menu class="h-5 w-5 text-muted-foreground" />
            </Button>
            <div class="flex flex-col sm:flex-row sm:items-baseline sm:gap-2 truncate">
              <span class="text-sm text-muted-foreground truncate font-medium" :title="sentence">
                <span class="hidden sm:inline">{{ sentence }}</span>
                <span class="sm:hidden">{{ sentenceContent }}</span>
              </span>
            </div>
          </div>
          <div class="flex items-center gap-1 sm:gap-2.5 shrink-0">
            <SystemNotice />
            <ThemeToggle />
          </div>
        </header>
  
        <!-- Page View Container -->
        <div class="flex-1 relative bg-background/50 flex flex-col overflow-hidden">
          <div class="flex-1 p-4 lg:p-6 mx-auto w-full overflow-y-auto custom-scrollbar flex flex-col">
            <RouterView />
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 10px;
}
.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.1);
}
</style>
