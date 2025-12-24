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
const siteSubtitle = ref('轻量级定时任务管理系统')
const siteIcon = ref('')

async function loadSiteSettings() {
  try {
    const res = await api.settings.getPublicSite()
    siteTitle.value = res.title || '白虎面板'
    siteSubtitle.value = res.subtitle || '轻量级定时任务管理系统'
    siteIcon.value = res.icon || ''
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
  } catch {
    // 使用默认值
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
  <div class="min-h-screen flex items-center justify-center bg-muted/30 p-4 relative">
    <!-- 右上角主题切换 -->
    <div class="absolute top-4 right-4">
      <ThemeToggle />
    </div>
    
    <div class="border rounded-lg bg-background shadow-sm overflow-hidden w-full max-w-[400px] lg:max-w-none lg:w-auto">
      <div class="flex">
        <!-- 左侧登录表单 -->
        <div class="w-full lg:w-96 p-6 sm:p-10">
          <div class="space-y-6 sm:space-y-8">
            <div class="space-y-2">
              <h1 class="text-xl sm:text-2xl font-bold tracking-tight">{{ siteTitle }}</h1>
              <p class="text-muted-foreground text-sm sm:text-base">{{ siteSubtitle }}</p>
            </div>
            <form @submit.prevent="handleLogin" class="space-y-4 sm:space-y-5">
              <div class="space-y-2">
                <Label>用户名</Label>
                <Input v-model="username" placeholder="请输入用户名" class="h-10 text-base" />
              </div>
              <div class="space-y-2">
                <Label>密码</Label>
                <Input v-model="password" type="password" placeholder="请输入密码" class="h-10 text-base" />
              </div>
              <Button type="submit" class="w-full h-10" :disabled="loading">
                {{ loading ? '登录中...' : '登录' }}
              </Button>
            </form>
          </div>
        </div>
        <!-- 右侧 Logo 展示（大屏显示） -->
        <div class="hidden lg:flex w-64 bg-muted/50 dark:bg-muted/30 items-center justify-center">
          <div v-if="siteIcon" class="w-44 h-44 [&>svg]:w-full [&>svg]:h-full" v-html="siteIcon" />
          <img v-else src="/logo.svg" alt="Logo" class="w-44 h-44" />
        </div>
      </div>
    </div>
  </div>
</template>
