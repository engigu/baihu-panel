<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { resetAuthCache } from '@/router'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)

const siteTitle = ref('白虎面板')
const siteSubtitle = ref('极致轻量、高性能的自动化任务调度平台')
const siteIcon = ref('')
const demoMode = ref(false)

// 从 localStorage 加载缓存的站点设置
function loadCachedSettings() {
  try {
    const cached = localStorage.getItem('site_settings')
    if (cached) {
      const settings = JSON.parse(cached)
      siteTitle.value = settings.title || '白虎面板'
      siteSubtitle.value = settings.subtitle || '极致轻量、高性能的自动化任务调度平台'
      siteIcon.value = settings.icon || ''
      demoMode.value = settings.demo_mode || false
      document.title = siteTitle.value

      // 设置 favicon
      if (siteIcon.value && siteIcon.value.trim().startsWith('<svg')) {
        const blob = new Blob([siteIcon.value], { type: 'image/svg+xml' })
        const url = URL.createObjectURL(blob)
        let link = document.querySelector("link[rel*='icon']") as HTMLLinkElement
        if (!link) {
          link = document.createElement('link')
          link.rel = 'icon'
          document.head.appendChild(link)
        }
        link.type = 'image/svg+xml'
        link.href = url
      }

      return true
    }
  } catch (e) {
    console.error('加载缓存的站点设置失败:', e)
  }
  return false
}

async function loadSiteSettings() {
  // 先加载缓存
  const hasCached = loadCachedSettings()

  // 后台异步更新
  try {
    const res = await api.settings.getPublicSite()
    siteTitle.value = res.title || '白虎面板'
    siteSubtitle.value = res.subtitle || '极致轻量、高性能的自动化任务调度平台'
    siteIcon.value = res.icon || ''
    demoMode.value = res.demo_mode || false
    document.title = siteTitle.value

    // 保存到 localStorage
    localStorage.setItem('site_settings', JSON.stringify({
      title: siteTitle.value,
      subtitle: siteSubtitle.value,
      icon: siteIcon.value,
      demo_mode: demoMode.value
    }))

    // 演示模式下自动填充账号密码
    if (demoMode.value) {
      username.value = 'admin'
      password.value = '123456'
    }

    // 设置 favicon
    if (siteIcon.value && siteIcon.value.trim().startsWith('<svg')) {
      const blob = new Blob([siteIcon.value], { type: 'image/svg+xml' })
      const url = URL.createObjectURL(blob)
      let link = document.querySelector("link[rel*='icon']") as HTMLLinkElement
      if (!link) {
        link = document.createElement('link')
        link.rel = 'icon'
        document.head.appendChild(link)
      }
      link.type = 'image/svg+xml'
      link.href = url
    }
  } catch {
    // 如果没有缓存，使用默认值
    if (!hasCached) {
      siteTitle.value = '白虎面板'
      siteSubtitle.value = '极致轻量、高性能的自动化任务调度平台'
    }
  }
}

async function handleLogin() {
  loading.value = true
  try {
    await api.auth.login({ username: username.value, password: password.value })
    resetAuthCache() // 重置认证缓存
    toast.success('登录成功')
    router.push('/')
  } catch {
    toast.error('登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}

onMounted(loadSiteSettings)
</script>

<template>
  <div
    class="min-h-screen flex items-center justify-center bg-[radial-gradient(ellipse_at_top_left,_var(--tw-gradient-stops))] from-slate-50 via-slate-100 to-slate-200 dark:from-slate-950 dark:via-slate-900 dark:to-black p-4 relative overflow-hidden">
    <!-- 装饰性背景 -->
    <div
      class="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-primary/5 rounded-full blur-[120px] pointer-events-none">
    </div>
    <div
      class="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-primary/10 rounded-full blur-[120px] pointer-events-none">
    </div>

    <!-- 右上角主题切换 -->
    <div class="absolute top-6 right-6 z-20">
      <ThemeToggle />
    </div>

    <div class="z-10 w-full max-w-[440px]">
      <div
        class="bg-background/80 backdrop-blur-xl border border-white/40 dark:border-white/10 shadow-[0_20px_50px_rgba(0,0,0,0.12)] rounded-[2.5rem] overflow-hidden">
        <div class="p-8 sm:p-12">
          <!-- 头部区域 -->
          <div class="flex flex-col items-center text-center mb-10">
            <div class="mb-6 relative group">
              <div
                class="absolute inset-0 bg-primary/20 blur-2xl rounded-full scale-150 opacity-50 group-hover:opacity-80 transition-opacity">
              </div>
              <div v-if="siteIcon" class="relative w-16 h-16 [&>svg]:w-full [&>svg]:h-full drop-shadow-md"
                v-html="siteIcon" />
              <div v-else
                class="relative w-16 h-16 bg-gradient-to-br from-primary/10 to-primary/5 rounded-2xl flex items-center justify-center border border-white/50 shadow-inner">
                <img src="/logo.svg" alt="Logo" class="w-10 h-10 object-contain opacity-90"
                  @error="(e: any) => e.target.style.display = 'none'" />
              </div>
            </div>

            <h1
              class="text-3xl font-extrabold tracking-tight bg-clip-text text-transparent bg-gradient-to-b from-foreground to-foreground/70 mb-2">
              {{ siteTitle }}</h1>
            <p class="text-muted-foreground/80 text-sm font-medium max-w-[260px] leading-relaxed">{{ siteSubtitle }}</p>
          </div>

          <!-- 登录表单 -->
          <form @submit.prevent="handleLogin" class="space-y-6">
            <div class="space-y-4">
              <div class="space-y-2">
                <Label class="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1">用户名</Label>
                <Input v-model="username" placeholder="请输入用户名"
                  class="h-12 text-base rounded-2xl bg-background/50 border-white/50 dark:border-white/5 focus:ring-4 focus:ring-primary/10 transition-all" />
              </div>
              <div class="space-y-2">
                <Label class="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1">密码</Label>
                <Input v-model="password" type="password" placeholder="请输入密码"
                  class="h-12 text-base rounded-2xl bg-background/50 border-white/50 dark:border-white/5 focus:ring-4 focus:ring-primary/10 transition-all" />
              </div>
            </div>
            <Button type="submit"
              class="w-full h-12 text-base font-bold rounded-2xl bg-gradient-to-r from-primary/90 to-primary shadow-lg shadow-primary/20 hover:shadow-primary/30 hover:scale-[1.02] active:scale-[0.98] transition-all"
              :disabled="loading">
              {{ loading ? '验证中...' : '立即登录' }}
            </Button>
          </form>

          <p v-if="demoMode" class="mt-8 text-center text-xs text-muted-foreground/60">
            演示模式：已为您自动填充测试账号
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
