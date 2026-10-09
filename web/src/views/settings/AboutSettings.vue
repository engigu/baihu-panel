<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  ExternalLink,
  TriangleAlert,
  History,
  Activity,
  Cpu,
  Clock,
  Layers,
  Sparkles,
  BookOpen,
  GitBranch,
  ShieldCheck
} from 'lucide-vue-next'
import { api, type AboutInfo } from '@/api'
import { formatDateTime, formatUptime } from '@/utils/date'

const aboutInfo = ref<AboutInfo | null>(null)

const techStack = ['Golang', 'Vue 3', 'TypeScript', 'Vite', 'Tailwind CSS', 'Shadcn/ui']
const features = [
  '脚本管理',
  '定时任务',
  '多语言支持',
  '依赖管理',
  '在线终端',
  '执行日志',
  '环境变量',
  '消息推送',
  '容器部署',
  '备份恢复'
]

// 解析内存数字与单位
const parsedMemory = computed(() => {
  const val = aboutInfo.value?.mem_usage
  if (!val) return { num: '-', unit: 'MB' }
  const match = val.trim().match(/^([\d.]+)\s*([a-zA-Z]+)?$/)
  if (match) {
    return { num: match[1], unit: match[2] || 'MB' }
  }
  return { num: val, unit: '' }
})

// 由前端独立负责格式化运行时长（紧凑格式 compact 与完整悬停格式 full）
const uptimeInfo = computed(() => {
  return formatUptime(aboutInfo.value?.uptime)
})

async function loadAbout() {
  try {
    aboutInfo.value = await api.settings.getAbout()
  } catch {}
}

onMounted(loadAbout)
</script>

<template>
  <div class="space-y-6">
    <!-- 左右双列平衡卡片 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
      <!-- 左列卡片：白虎面板与技术生态 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-3 pt-0 px-5 space-y-2">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2 flex-wrap min-w-0">
              <CardTitle class="text-base sm:text-lg font-bold">Baihu Panel</CardTitle>
            </div>

            <a
              href="https://engigu.github.io/baihu-panel/guide/changelog.html"
              target="_blank"
              class="shrink-0"
            >
              <Button
                variant="outline"
                size="sm"
                class="h-7 px-2.5 rounded-full border-primary/20 bg-primary/5 text-primary text-xs font-medium hover:bg-primary/10 gap-1.5 shadow-none"
              >
                <History class="w-3.5 h-3.5" />
                <span>更新日志</span>
              </Button>
            </a>
          </div>

          <CardDescription class="text-xs text-muted-foreground leading-relaxed pt-0.5">
            极致轻量、高性能自动化任务调度平台。深度集成 Mise 运行时，支持多语言环境动态切换与依赖全自动管理。
          </CardDescription>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0">
          <div class="space-y-2.5">
            <!-- 技术栈底座 -->
            <div class="space-y-1.5">
              <div class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Sparkles class="w-3.5 h-3.5 text-primary" />
                <span>技术底座</span>
              </div>
              <div class="flex flex-wrap gap-1.5">
                <Badge
                  v-for="tech in techStack"
                  :key="tech"
                  class="text-xs bg-primary/15 text-primary border-0 font-normal px-2.5 py-0.5"
                >
                  {{ tech }}
                </Badge>
              </div>
            </div>

            <!-- 核心业务特性 -->
            <div class="space-y-1.5">
              <div class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Layers class="w-3.5 h-3.5 text-muted-foreground" />
                <span>核心能力</span>
              </div>
              <div class="flex flex-wrap gap-1.5">
                <Badge
                  v-for="feature in features"
                  :key="feature"
                  class="text-xs bg-accent text-accent-foreground border-0 font-normal px-2.5 py-0.5"
                >
                  {{ feature }}
                </Badge>
              </div>
            </div>
          </div>

          <!-- 卡片底部免责声明 -->
          <div class="mt-4 p-3 rounded-xl bg-muted/30 border border-yellow-500/20 space-y-1.5">
            <div class="text-xs font-semibold text-yellow-600 dark:text-yellow-500 flex items-center gap-1.5">
              <TriangleAlert class="h-3.5 w-3.5 shrink-0" />
              <span>免责声明</span>
            </div>
            <div class="space-y-1 text-xs text-muted-foreground leading-relaxed">
              <p>本项目不提供、不内置任何具有实际业务逻辑的第三方脚本。</p>
              <p><strong>请勿轻易执行任何来源不明或不可信的外部脚本。</strong></p>
              <p>所有脚本及代码均需由用户自行添加或配置，用户须自行审核以确保其安全性。本项目仅作为基础调度工具，<strong class="text-foreground/70">无法且不保证任何被执行任务的安全性</strong>。</p>
              <p>本项目为业余开源开发，按“原样”提供，不保证不存在 Bug 或漏洞。开发者不对因使用本项目运行不安全脚本带来的数据泄露、系统损坏及法律责任等后果负责。</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- 右列卡片：系统运行状态与环境指标 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-2.5 pt-0 px-5">
          <div class="flex items-center justify-between">
            <div class="space-y-1">
              <CardTitle class="text-base flex items-center gap-2">
                <Activity class="w-4 h-4 text-primary" />
                <span>系统状态与指标</span>
              </CardTitle>
              <CardDescription class="text-xs">当前宿主环境资源开销与持续运行状态</CardDescription>
            </div>

            <div class="flex items-center gap-2">
              <a
                href="https://engigu.github.io/baihu-panel/"
                target="_blank"
                class="inline-flex"
              >
                <Button variant="ghost" size="sm" class="h-7.5 px-2 text-xs text-muted-foreground hover:text-foreground gap-1">
                  <BookOpen class="w-3.5 h-3.5" />
                  <span>文档</span>
                </Button>
              </a>
              <a
                href="https://github.com/engigu/baihu-panel/"
                target="_blank"
                class="inline-flex"
              >
                <Button variant="ghost" size="sm" class="h-7.5 px-2 text-xs text-muted-foreground hover:text-foreground gap-1">
                  <ExternalLink class="w-3.5 h-3.5" />
                  <span>GitHub</span>
                </Button>
              </a>
            </div>
          </div>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0">
          <div class="space-y-2.5">
            <!-- 2x2 规整指标网格（严格等高且排印统一） -->
            <div class="grid grid-cols-2 gap-2.5">
              <!-- 内存占用 -->
              <div class="p-3 rounded-xl border border-border/70 bg-muted/20 hover:bg-muted/30 transition-all flex flex-col justify-between gap-1 group min-h-[102px]">
                <div class="h-6 flex items-center justify-between text-xs text-muted-foreground font-medium">
                  <span>内存占用</span>
                  <div class="w-6 h-6 rounded-md bg-muted/50 flex items-center justify-center text-muted-foreground/70 group-hover:text-primary transition-colors">
                    <Cpu class="w-3.5 h-3.5" />
                  </div>
                </div>
                <div class="h-7 flex items-baseline gap-1 my-0.5 truncate">
                  <span class="text-lg sm:text-xl font-bold tracking-tight text-foreground font-inter">
                    {{ parsedMemory.num }}
                  </span>
                  <span class="text-xs font-semibold text-muted-foreground/75 font-inter">
                    {{ parsedMemory.unit }}
                  </span>
                </div>
                <div class="h-4 flex items-center text-[10px] text-muted-foreground truncate">
                  <span>常驻轻量内存</span>
                </div>
              </div>

              <!-- 活跃协程 -->
              <div class="p-3 rounded-xl border border-border/70 bg-muted/20 hover:bg-muted/30 transition-all flex flex-col justify-between gap-1 group min-h-[102px]">
                <div class="h-6 flex items-center justify-between text-xs text-muted-foreground font-medium">
                  <span>活跃协程</span>
                  <div class="w-6 h-6 rounded-md bg-muted/50 flex items-center justify-center text-muted-foreground/70 group-hover:text-primary transition-colors">
                    <Layers class="w-3.5 h-3.5" />
                  </div>
                </div>
                <div class="h-7 flex items-baseline gap-1 my-0.5 truncate">
                  <span class="text-lg sm:text-xl font-bold tracking-tight text-foreground font-inter">
                    {{ aboutInfo?.goroutines ?? '-' }}
                  </span>
                  <span class="text-xs font-medium text-muted-foreground/75">个</span>
                </div>
                <div class="h-4 flex items-center text-[10px] text-muted-foreground truncate">
                  <span>高效轻量并发</span>
                </div>
              </div>

              <!-- 连续运行时长 -->
              <div class="p-3 rounded-xl border border-border/70 bg-muted/20 hover:bg-muted/30 transition-all flex flex-col justify-between gap-1 group min-h-[102px]">
                <div class="h-6 flex items-center justify-between text-xs text-muted-foreground font-medium">
                  <span>运行时间</span>
                  <div class="w-6 h-6 rounded-md bg-muted/50 flex items-center justify-center text-muted-foreground/70 group-hover:text-primary transition-colors">
                    <Clock class="w-3.5 h-3.5" />
                  </div>
                </div>
                <div class="h-7 flex items-baseline gap-1 my-0.5 truncate" :title="uptimeInfo.full">
                  <template v-if="uptimeInfo.parts && uptimeInfo.parts.length">
                    <div v-for="(p, idx) in uptimeInfo.parts" :key="idx" class="flex items-baseline gap-0.5">
                      <span class="text-lg sm:text-xl font-bold tracking-tight text-foreground font-inter">
                        {{ p.value }}
                      </span>
                      <span class="text-xs font-semibold text-muted-foreground/75 font-inter">
                        {{ p.unit }}
                      </span>
                    </div>
                  </template>
                  <span v-else class="text-lg sm:text-xl font-bold tracking-tight text-foreground font-inter">-</span>
                </div>
                <div class="h-4 flex items-center text-[10px] text-muted-foreground truncate">
                  <span>持续稳定守护</span>
                </div>
              </div>

              <!-- 当前版本 / 远端对比 -->
              <div class="p-3 rounded-xl border border-border/70 bg-muted/20 hover:bg-muted/30 transition-all flex flex-col justify-between gap-1 group min-h-[102px]">
                <div class="h-6 flex items-center justify-between text-xs text-muted-foreground font-medium">
                  <span>当前版本</span>
                  <div class="w-6 h-6 rounded-md bg-muted/50 flex items-center justify-center transition-colors">
                    <span class="relative flex h-2 w-2">
                      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                      <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                    </span>
                  </div>
                </div>
                <div class="h-7 flex items-baseline gap-1 my-0.5 truncate" :title="aboutInfo?.version || ''">
                  <span class="text-lg sm:text-xl font-bold tracking-tight text-foreground font-inter truncate">
                    {{ aboutInfo?.version || 'dev' }}
                  </span>
                </div>
                <div class="h-4 flex items-center text-[10px] truncate">
                  <span
                    v-if="aboutInfo?.remote_version && aboutInfo.remote_version !== aboutInfo.version"
                    class="text-primary font-medium"
                  >
                    新版可用: {{ aboutInfo.remote_version }}
                  </span>
                  <span v-else class="text-muted-foreground">已是最新发布版</span>
                </div>
              </div>
            </div>

            <!-- 构建时间条 -->
            <div class="p-2.5 rounded-xl border border-border/60 bg-muted/20 flex items-center justify-between text-xs text-muted-foreground">
              <span class="flex items-center gap-1.5">
                <GitBranch class="w-3.5 h-3.5 text-muted-foreground/70" />
                <span>构建时间戳</span>
              </span>
              <span class="font-inter font-medium text-foreground/90">
                {{ aboutInfo?.build_time ? formatDateTime(aboutInfo.build_time) : 'unknown' }}
              </span>
            </div>
          </div>

          <!-- 右卡片底部架构说明条 -->
          <div class="mt-4 p-3 rounded-xl bg-muted/20 border border-border/60 text-xs text-muted-foreground leading-relaxed flex items-start gap-2.5">
            <ShieldCheck class="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
            <div>
              <span class="font-medium text-foreground">原生无侵入架构：</span>
              <span>Go 纯原生单文件编译，无需笨重外部数据库，冷启动毫秒级，平稳胜任高频定时调度。</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- 底部版权声明（单行内不换行） -->
    <div class="pt-2 text-center text-[10px] sm:text-xs text-muted-foreground flex items-center justify-center gap-1.5 sm:gap-2 whitespace-nowrap">
      <span>© 2025 - Present Baihu Panel. 保留所有权利。</span>
      <span class="opacity-20">|</span>
      <a
        href="https://github.com/engigu/baihu-panel/"
        target="_blank"
        class="inline-flex items-center gap-1 text-primary hover:underline shrink-0"
      >
        <ExternalLink class="w-3 h-3" />
        <span>GitHub 仓库</span>
      </a>
    </div>
  </div>
</template>

<style scoped>
.font-inter {
  font-family: 'Inter Variable', 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif !important;
  font-feature-settings: "cv05", "cv08", "cv11", "ss01", "ss03", "tnum" !important;
  font-optical-sizing: auto;
}
</style>
