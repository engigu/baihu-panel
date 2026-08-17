<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Badge } from '@/components/ui/badge'
import { ExternalLink, TriangleAlert, History } from 'lucide-vue-next'
import { api, type AboutInfo } from '@/api'

const aboutInfo = ref<AboutInfo | null>(null)

const techStack = ['Golang', 'Vue 3', 'TypeScript', 'Vite', 'Tailwind CSS', 'Shadcn/ui']
const features = ['脚本管理', '定时任务', '多语言支持', '依赖管理', '在线终端', '执行日志', '环境变量', '消息推送', '容器部署', '备份恢复']

async function loadAbout() {
  try {
    aboutInfo.value = await api.settings.getAbout()
  } catch { }
}

onMounted(loadAbout)
</script>

<template>
  <div>
    <!-- 站点关于 -->
    <div class="mb-8 flex flex-col lg:flex-row justify-between items-start gap-4">
      <div class="flex-1 min-w-0 w-full">
        <div class="flex items-start justify-between mb-1.5 gap-4">
          <h3 class="text-xl font-bold">白虎面板 (Baihu Panel)</h3>
          <a href="https://engigu.github.io/baihu-panel/guide/changelog.html" target="_blank"
            class="lg:hidden inline-flex items-center gap-1.5 h-8 px-3 rounded-full border border-primary/20 bg-primary/5 text-primary text-[10px] sm:text-xs font-semibold hover:bg-primary/10 transition-all whitespace-nowrap shadow-sm">
            <History class="h-3 w-3" />
            更新日志
          </a>
        </div>
        <p class="text-sm text-muted-foreground leading-relaxed">极致轻量、高性能的自动化任务调度平台。深度集成 Mise 运行时管理，支持多语言环境动态切换与全自动依赖管理。</p>
      </div>
      <a href="https://engigu.github.io/baihu-panel/guide/changelog.html" target="_blank"
        class="hidden lg:inline-flex items-center gap-1.5 h-9 px-4 rounded-full border border-primary/20 bg-primary/5 text-primary text-xs font-semibold hover:bg-primary/10 transition-all whitespace-nowrap shadow-sm shadow-primary/5">
        <History class="h-3.5 w-3.5" />
        查看更新日志
      </a>
    </div>

    <div class="grid sm:grid-cols-2 gap-x-8 gap-y-5">
      <!-- 左侧：技术栈和功能特性 -->
      <div class="space-y-5">
        <div>
          <h4 class="text-sm font-medium mb-2">技术栈</h4>
          <div class="flex flex-wrap gap-1.5">
            <Badge v-for="tech in techStack" :key="tech" class="text-xs bg-primary/15 text-primary border-0">{{ tech }}
            </Badge>
          </div>
        </div>

        <div>
          <h4 class="text-sm font-medium mb-2">功能特性</h4>
          <div class="flex flex-wrap gap-1.5">
            <Badge v-for="feature in features" :key="feature" class="text-xs bg-accent text-accent-foreground">{{
              feature }}</Badge>
          </div>
        </div>
      </div>

      <!-- 右侧：系统信息 -->
      <div>
        <h4 class="text-sm font-medium mb-2">系统信息</h4>
        <div class="space-y-2">
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">当前版本:</span>
            <div class="flex items-center gap-1.5">
              <span class="text-muted-foreground text-sm">{{ aboutInfo?.version || 'dev' }}</span>
              <Badge v-if="aboutInfo?.remote_version && aboutInfo.remote_version === aboutInfo.version" variant="secondary"
                class="text-[10px] h-4 px-1 bg-green-500/10 text-green-600 border-green-500/20 shadow-none">
                最新版本
              </Badge>
            </div>
          </div>
          <div v-if="aboutInfo?.remote_version && aboutInfo.remote_version !== aboutInfo.version"
            class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">最新版本:</span>
            <span class="text-muted-foreground text-sm">{{ aboutInfo.remote_version }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">构建时间:</span>
            <span class="text-muted-foreground text-sm">{{ aboutInfo?.build_time || '-' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">内存使用:</span>
            <span class="text-muted-foreground text-sm">{{ aboutInfo?.mem_usage || '-' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">协程数量:</span>
            <span class="text-muted-foreground text-sm">{{ aboutInfo?.goroutines || '-' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground text-sm">运行时间:</span>
            <span class="text-muted-foreground text-sm">{{ aboutInfo?.uptime || '-' }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-8 p-4 bg-muted/40 rounded-lg border border-yellow-500/20">
      <h4 class="text-sm font-semibold text-yellow-600 dark:text-yellow-500 mb-2 flex items-center gap-1.5">
        <TriangleAlert class="h-4 w-4" />
        免责声明
      </h4>
      <div class="space-y-1.5 text-xs text-muted-foreground">
        <p>本项目不提供、不内置任何具有实际业务逻辑的第三方脚本。</p>
        <p><strong>请勿轻易执行任何来源不明或不可信的外部脚本。</strong></p>
        <p>所有脚本及代码均需由用户自行添加或配置，用户须自行审核以确保其安全性。本项目仅作为基础调度工具，<strong class="text-foreground/70">无法且不保证任何被执行任务的安全性</strong>。</p>
        <p>本项目为业余开源开发，按“原样”提供，不保证不存在 Bug 或漏洞。开发者不对因使用本项目运行不安全脚本带来的数据泄露、系统损坏及法律责任等后果负责。</p>
      </div>
    </div>

    <!-- 底部：版权和链接 -->
    <div class="mt-6 pt-4 border-t flex items-center justify-center gap-2 text-[10px] sm:text-sm text-muted-foreground whitespace-nowrap overflow-hidden">
      <span>© 2025 - Present 保留所有权利。</span>
      <span class="opacity-20">|</span>
      <a href="https://github.com/engigu/baihu-panel/" target="_blank"
        class="inline-flex items-center gap-1 text-primary hover:underline">
        <ExternalLink class="h-3 w-3" />
        GitHub
      </a>
    </div>
  </div>
</template>
