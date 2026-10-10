<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api, type SiteSettings } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { copyToClipboard as copyTextToClipboard } from '@/utils/clipboard'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  RefreshCw,
  Copy,
  AlertTriangle,
  ExternalLink,
  Info,
  Clock,
  Globe,
  Database,
  Code2,
  Bell,
  Send,
  ShieldCheck,
  CalendarClock,
  Save,
  Sparkles,
  Cpu
} from 'lucide-vue-next'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

const { refreshSettings } = useSiteSettings()

const form = ref<SiteSettings>({
  title: '',
  subtitle: '',
  icon: '',
  page_size: '10',
  cookie_days: '7',
  openapi_enabled: false,
  openapi_token: '',
  openapi_token_expire: '',
  system_notice_days: '30',
  system_notice_max_count: '500',
  push_log_days: '15',
  push_log_max_count: '5000',
  login_log_days: '30',
  login_log_max_count: '1000',
  scheduler_log_days: '30',
  scheduler_log_max_count: '10000',
  cache_trim_enabled: true,
  mem_watermark_rate: '80',
  cache_max_mb: '60',
  cache_trim_spec: '@every 5m',
  task_finished_trim: true
})
const loading = ref(false)
const trimming = ref(false)
const showOpenapiConfirmDialog = ref(false)

const iconPreview = computed(() => {
  if (!form.value.icon) return ''
  if (form.value.icon.trim().startsWith('<svg')) {
    return form.value.icon
  }
  return ''
})

async function loadSettings() {
  try {
    const res = await api.settings.getSite()
    form.value = {
      ...res,
      page_size: res.page_size || '10',
      cookie_days: res.cookie_days || '7',
      system_notice_days: res.system_notice_days || '30',
      system_notice_max_count: res.system_notice_max_count || '500',
      push_log_days: res.push_log_days || '15',
      push_log_max_count: res.push_log_max_count || '5000',
      login_log_days: res.login_log_days || '30',
      login_log_max_count: res.login_log_max_count || '1000',
      scheduler_log_days: res.scheduler_log_days || '30',
      scheduler_log_max_count: res.scheduler_log_max_count || '10000',
      cache_trim_enabled: res.cache_trim_enabled === undefined ? true : (res.cache_trim_enabled === true || (res as any).cache_trim_enabled === 'true'),
      mem_watermark_rate: res.mem_watermark_rate || '80',
      cache_max_mb: res.cache_max_mb || '60',
      cache_trim_spec: res.cache_trim_spec || '@every 5m',
      task_finished_trim: res.task_finished_trim === undefined ? true : (res.task_finished_trim === true || (res as any).task_finished_trim === 'true'),
      openapi_enabled: res.openapi_enabled === true || (res as any).openapi_enabled === 'true'
    }
  } catch { }
}

async function saveSettings() {
  loading.value = true
  try {
    await api.settings.updateSite({
      ...form.value,
      page_size: String(form.value.page_size),
      cookie_days: String(form.value.cookie_days),
      system_notice_days: String(form.value.system_notice_days || '30'),
      system_notice_max_count: String(form.value.system_notice_max_count || '500'),
      push_log_days: String(form.value.push_log_days || '15'),
      push_log_max_count: String(form.value.push_log_max_count || '5000'),
      login_log_days: String(form.value.login_log_days || '30'),
      login_log_max_count: String(form.value.login_log_max_count || '1000'),
      scheduler_log_days: String(form.value.scheduler_log_days || '30'),
      scheduler_log_max_count: String(form.value.scheduler_log_max_count || '10000'),
      cache_trim_enabled: form.value.cache_trim_enabled,
      mem_watermark_rate: String(form.value.mem_watermark_rate || '80'),
      cache_max_mb: String(form.value.cache_max_mb || '60'),
      cache_trim_spec: String(form.value.cache_trim_spec || '@every 5m'),
      task_finished_trim: form.value.task_finished_trim
    })
    await refreshSettings()
    await loadSettings()
    toast.success('保存成功')
  } catch {
    toast.error('保存失败')
  } finally {
    loading.value = false
  }
}

async function generateOpenapiToken() {
  try {
    const res = await api.settings.generateOpenapiToken()
    form.value.openapi_token = res.token

    if (!form.value.openapi_token_expire) {
      const d = new Date()
      d.setFullYear(d.getFullYear() + 1)
      form.value.openapi_token_expire = d.toISOString().split('T')[0]
    }
  } catch {
    toast.error('生成 Token 失败')
  }
}

async function copyOpenapiToken() {
  if (!form.value.openapi_token) return
  const success = await copyTextToClipboard(form.value.openapi_token)
  if (success) {
    toast.success('Token 已复制到剪贴板')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

function openSwaggerDocs() {
  window.open('https://engigu.github.io/baihu-panel/guide/api.html', '_blank')
}

function openMcpDocs() {
  window.open('https://engigu.github.io/baihu-panel/guide/mcp.html', '_blank')
}

const showMcpSnippet = ref(false)

const mcpSseUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  const base = window.location.origin
  const token = form.value.openapi_token || '<Token>'
  return `${base}/open2api/v1/mcp/sse?token=${token}`
})

const mcpConfigJson = computed(() => {
  return JSON.stringify({
    mcpServers: {
      "baihu": {
        "url": mcpSseUrl.value
      }
    }
  }, null, 2)
})

async function copyMcpSseUrl() {
  if (!form.value.openapi_token) {
    toast.error('请先生成并保存 OpenAPI Token')
    return
  }
  const success = await copyTextToClipboard(mcpSseUrl.value)
  if (success) {
    toast.success('MCP SSE 连接地址已复制到剪贴板')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

async function copyMcpConfigJson() {
  if (!form.value.openapi_token) {
    toast.error('请先生成并保存 OpenAPI Token')
    return
  }
  const success = await copyTextToClipboard(mcpConfigJson.value)
  if (success) {
    toast.success('MCP 客户端配置 JSON 已复制')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

async function handleTrimCache() {
  trimming.value = true
  try {
    const res = await api.settings.trimCache()
    toast.success(res.message || '内存与文件缓存回收成功')
  } catch (err: any) {
    toast.error(err?.message || '触发缓存回收失败')
  } finally {
    trimming.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
      <!-- 左列：站点外观 + OpenAPI 接口集成 -->
      <div class="space-y-6">
        <!-- 模块 1: 站点外观与基础配置 -->
        <Card class="shadow-sm">
          <CardHeader class="pb-4">
            <CardTitle class="flex items-center gap-2 text-base">
              <Globe class="w-4 h-4 text-primary" />
              <span>站点外观与基础配置</span>
            </CardTitle>
            <CardDescription>自定义站点品牌标识、标题标语及系统常规运行参数</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">站点标题</Label>
              <Input v-model="form.title" placeholder="白虎面板" class="h-9" />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">站点标语</Label>
              <Input v-model="form.subtitle" placeholder="轻量级定时任务管理系统" class="h-9" />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">站点图标 (SVG 代码)</Label>
              <div class="flex items-center gap-2">
                <Input v-model="form.icon" placeholder="<svg>...</svg>" class="flex-1 font-mono text-xs h-9" />
                <div
                  v-if="iconPreview"
                  class="p-1.5 border rounded-lg bg-card w-9 h-9 flex items-center justify-center shrink-0 [&>svg]:w-5 [&>svg]:h-5 shadow-sm"
                  v-html="iconPreview"
                />
              </div>
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">系统常规配置</Label>
              <div class="grid grid-cols-2 gap-3">
                <div class="relative">
                  <Input
                    v-model="form.page_size"
                    type="number"
                    class="h-9 pr-12 text-sm [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">条/页</span>
                </div>
                <div class="relative">
                  <Input
                    v-model="form.cookie_days"
                    type="number"
                    class="h-9 pr-14 text-sm [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">天过期</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- 模块 2: OpenAPI 开放接口能力 -->
        <Card class="shadow-sm">
          <CardHeader class="pb-4 space-y-2">
            <div class="flex items-center justify-between gap-3">
              <CardTitle class="flex items-center gap-2 text-sm sm:text-base font-semibold min-w-0">
                <Code2 class="w-4 h-4 text-sky-500 shrink-0" />
                <span class="truncate">OpenAPI 接口与集成</span>
                <Badge variant="secondary" class="font-normal text-[10px] bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20 shrink-0">
                  推荐接入
                </Badge>
              </CardTitle>
              <div class="flex items-center gap-2 shrink-0">
                <Switch v-model="form.openapi_enabled" id="openapi-enabled" />
                <Label for="openapi-enabled" class="text-xs cursor-pointer">开启</Label>
              </div>
            </div>
            <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
              <span class="truncate">通过 Bearer Token 实现外部系统调用</span>
              <div class="flex items-center gap-2.5 shrink-0">
                <a
                  href="#"
                  @click.prevent="openMcpDocs"
                  class="flex items-center gap-1 text-xs text-purple-600 dark:text-purple-400 hover:underline"
                >
                  <Sparkles class="w-3 h-3 text-purple-500" />
                  MCP 指南
                  <ExternalLink class="w-2.5 h-2.5" />
                </a>
                <a
                  href="#"
                  @click.prevent="openSwaggerDocs"
                  class="flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline"
                >
                  API 文档
                  <ExternalLink class="w-2.5 h-2.5" />
                </a>
              </div>
            </div>
          </CardHeader>
          <CardContent class="space-y-4">
            <p class="text-xs text-muted-foreground leading-relaxed">
              外部请求时需携带请求头
              <code class="bg-muted px-1.5 py-0.5 rounded text-[11px] font-mono select-all inline-block break-all">Authorization: Bearer &lt;Token&gt;</code>。
            </p>

            <div class="space-y-3">
              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground">Token 密钥</Label>
                <div class="flex items-center space-x-2">
                  <Input
                    v-model="form.openapi_token"
                    placeholder="点击生成 32 位 Token"
                    class="text-xs sm:text-sm h-9 font-mono flex-1 min-w-0"
                    :disabled="!form.openapi_enabled"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    class="h-9 w-9 shrink-0"
                    @click="showOpenapiConfirmDialog = true"
                    title="随机生成"
                    :disabled="!form.openapi_enabled"
                  >
                    <RefreshCw class="w-4 h-4" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    class="h-9 w-9 shrink-0"
                    @click="copyOpenapiToken"
                    title="复制 Token"
                    :disabled="!form.openapi_token || !form.openapi_enabled"
                  >
                    <Copy class="w-4 h-4" />
                  </Button>
                </div>
              </div>

              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground">截止有效期</Label>
                <Input
                  v-model="form.openapi_token_expire"
                  type="date"
                  class="w-full dark:[color-scheme:dark] h-9 text-xs sm:text-sm"
                  :disabled="!form.openapi_enabled"
                />
                <p class="text-[10px] text-muted-foreground">超过此日期后 Token 将失效，置空代表永不过期</p>
              </div>
            </div>

            <!-- 模块 2.1: 模型上下文协议 (MCP) 快捷连接指引 -->
            <div class="pt-3 border-t border-border/60">
              <div class="p-3 rounded-xl border bg-gradient-to-br from-purple-500/5 via-background to-sky-500/5 border-purple-500/25 space-y-2.5">
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <Sparkles class="w-3.5 h-3.5 text-purple-500 shrink-0" />
                    <span class="text-xs font-semibold text-foreground">AI 客户端连接 (MCP)</span>
                    <Badge variant="outline" class="font-normal text-[9px] px-1.5 py-0 h-4 border-purple-500/30 text-purple-600 dark:text-purple-400 shrink-0">
                      Cursor / Claude
                    </Badge>
                  </div>
                  <button
                    type="button"
                    @click="showMcpSnippet = !showMcpSnippet"
                    class="text-[11px] text-purple-600 dark:text-purple-400 hover:underline flex items-center gap-0.5 shrink-0"
                  >
                    {{ showMcpSnippet ? '收起配置' : '配置示例' }}
                  </button>
                </div>

                <div class="space-y-1">
                  <Label class="text-[11px] text-muted-foreground flex items-center justify-between">
                    <span>SSE 远程连接 URL</span>
                    <span v-if="!form.openapi_token" class="text-amber-500 text-[10px]">需先生成 Token</span>
                  </Label>
                  <div class="flex items-center gap-1.5">
                    <Input
                      :model-value="mcpSseUrl"
                      readonly
                      class="text-xs h-8 font-mono bg-muted/40 select-all flex-1 min-w-0"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      class="h-8 px-2.5 shrink-0 text-xs gap-1"
                      @click="copyMcpSseUrl"
                      :disabled="!form.openapi_token || !form.openapi_enabled"
                    >
                      <Copy class="w-3 h-3" />
                      复制
                    </Button>
                  </div>
                </div>

                <!-- 展开的 JSON 客户端配置 -->
                <div v-if="showMcpSnippet" class="space-y-1.5 pt-1">
                  <div class="flex items-center justify-between">
                    <span class="text-[10px] text-muted-foreground font-mono">claude_desktop_config.json / Cursor</span>
                    <button
                      type="button"
                      @click="copyMcpConfigJson"
                      class="text-[10px] text-purple-600 dark:text-purple-400 hover:underline flex items-center gap-1"
                    >
                      <Copy class="w-2.5 h-2.5" />
                      复制配置 JSON
                    </button>
                  </div>
                  <pre class="p-2 rounded-lg bg-muted/80 text-[10px] font-mono leading-relaxed overflow-x-auto select-all text-foreground/90 max-h-36 border border-border/50">{{ mcpConfigJson }}</pre>
                  <p class="text-[10px] text-muted-foreground leading-normal">
                    提示：本地环境还可直接在终端使用 <code class="bg-muted px-1 py-0.5 rounded font-mono text-[10px]">baihu mcp</code> 启动 Stdio 管道连接。
                  </p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- 右列：日志生命周期与自动清理 -->
      <div class="space-y-6">
        <Card class="shadow-sm">
          <CardHeader class="pb-4">
            <CardTitle class="flex items-center gap-2 text-base">
              <Database class="w-4 h-4 text-indigo-500" />
              <span>日志生命周期与自动清理</span>
            </CardTitle>
            <CardDescription>按保留天数与条数双维度监控，定期自动清理历史过期数据</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <!-- 4 大日志清理条目（响应式单列/双行适配，小屏下自动折行不挤压） -->
            <div class="space-y-2.5">
              <!-- 系统通知 -->
              <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400 shrink-0">
                    <Bell class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="text-xs font-semibold text-foreground truncate">系统通知清理</div>
                    <div class="text-[10px] text-muted-foreground truncate">站内消息通知及系统公告</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.system_notice_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.system_notice_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 推送日志 -->
              <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-violet-500/10 text-violet-600 dark:text-violet-400 shrink-0">
                    <Send class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="text-xs font-semibold text-foreground truncate">推送日志清理</div>
                    <div class="text-[10px] text-muted-foreground truncate">通道消息分发与推送结果</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.push_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.push_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 登录日志 -->
              <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shrink-0">
                    <ShieldCheck class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="text-xs font-semibold text-foreground truncate">登录日志清理</div>
                    <div class="text-[10px] text-muted-foreground truncate">用户登录认证与安全审计</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.login_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.login_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 调度日志 -->
              <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-amber-500/10 text-amber-600 dark:text-amber-400 shrink-0">
                    <CalendarClock class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="text-xs font-semibold text-foreground truncate">调度日志清理</div>
                    <div class="text-[10px] text-muted-foreground truncate">定时任务触发与运行记录</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.scheduler_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.scheduler_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 说明底条 -->
            <div class="p-3 bg-muted/30 rounded-xl border border-border/70 space-y-1.5 text-[11px] text-muted-foreground">
              <div class="flex items-start gap-2">
                <Info class="w-3.5 h-3.5 text-blue-500 shrink-0 mt-0.5" />
                <div class="leading-relaxed">
                  <span class="font-medium text-foreground">双重维度限制：</span>
                  同时监控天数与数量。超出天数则物理删除，超出数量上限自动淘汰最早记录。
                </div>
              </div>
              <div class="flex items-start gap-2">
                <Clock class="w-3.5 h-3.5 text-amber-500 shrink-0 mt-0.5" />
                <div class="leading-relaxed">
                  <span class="font-medium text-foreground">周期执行说明：</span>
                  服务启动时全量检测一次，运行期间后台每隔 1 小时自动触发巡检清理。
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- 模块 3.5: 容器内存与缓存优化 -->
        <Card class="shadow-sm">
          <CardHeader class="pb-4 space-y-2">
            <div class="flex items-center justify-between gap-3">
              <CardTitle class="flex items-center gap-2 text-sm sm:text-base font-semibold min-w-0">
                <Cpu class="w-4 h-4 text-emerald-500 shrink-0" />
                <span class="truncate">容器内存与缓存优化</span>
              </CardTitle>
              <div class="flex items-center gap-2 shrink-0">
                <Switch v-model="form.cache_trim_enabled" id="cache-trim-enabled" />
                <Label for="cache-trim-enabled" class="text-xs cursor-pointer">开启自适应</Label>
              </div>
            </div>
            <CardDescription class="text-xs">
              监控容器内存水位与文件缓存，在内存超标或任务执行完毕后自动安全释放
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <!-- 警戒水位线 -->
              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground">内存警戒水位</Label>
                <div class="relative">
                  <Input
                    v-model="form.mem_watermark_rate"
                    type="number"
                    class="h-9 pr-8 text-sm"
                    min="10"
                    max="99"
                    :disabled="!form.cache_trim_enabled"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">%</span>
                </div>
                <p class="text-[10px] text-muted-foreground">容器内存达此比例时介入 (推荐 70-90%)</p>
              </div>

              <!-- 缓存释放阈值 -->
              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground">缓存触发阈值</Label>
                <div class="relative">
                  <Input
                    v-model="form.cache_max_mb"
                    type="number"
                    class="h-9 pr-10 text-sm"
                    min="10"
                    :disabled="!form.cache_trim_enabled"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">MB</span>
                </div>
                <p class="text-[10px] text-muted-foreground">文件缓存超此大小时介入 (默认 60MB)</p>
              </div>
            </div>

            <!-- 任务结束回收与手动触发行 -->
            <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shrink-0">
                  <Sparkles class="w-4 h-4" />
                </div>
                <div class="min-w-0">
                  <div class="text-xs font-semibold text-foreground truncate">任务结束自动回收</div>
                  <div class="text-[10px] text-muted-foreground truncate">任务全部执行平息后，自动释放脚本产物与环境依赖缓存</div>
                </div>
              </div>
              <div class="flex items-center gap-3 shrink-0 self-end sm:self-auto">
                <div class="flex items-center gap-1.5">
                  <Switch v-model="form.task_finished_trim" id="task-finished-trim" :disabled="!form.cache_trim_enabled" />
                  <Label for="task-finished-trim" class="text-xs cursor-pointer">自动</Label>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="h-8 px-2.5 text-xs gap-1"
                  @click="handleTrimCache"
                  :disabled="trimming"
                >
                  <RefreshCw :class="['w-3 h-3', trimming ? 'animate-spin' : '']" />
                  {{ trimming ? '清理中...' : '立即清理' }}
                </Button>
              </div>
            </div>

            <!-- 说明底条 -->
            <div class="p-3 bg-muted/30 rounded-xl border border-border/70 space-y-1.5 text-[11px] text-muted-foreground">
              <div class="flex items-start gap-2">
                <Info class="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5" />
                <div class="leading-relaxed">
                  <span class="font-medium text-foreground">回收说明：</span>
                  平时保留缓存以加速磁盘读写，仅在内存超标或任务执行完成后自动安全释放，避免容器内存堆积。
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- 模块 4: 配置保存与生效面板 -->
        <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
          <CardHeader class="pb-3 pt-0 px-5">
            <div class="space-y-1">
              <CardTitle class="text-base flex items-center gap-2">
                <Save class="w-4 h-4 text-emerald-500" />
                <span>保存与应用配置</span>
              </CardTitle>
              <CardDescription class="text-xs leading-relaxed pt-0.5">
                提交并持久化当前设置，站点品牌、会话参数、日志清理策略及全局 OpenAPI 访问凭据将立即同步生效。
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent class="px-5 pb-0 pt-0">
            <!-- 底部操作按钮栏 -->
            <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <span class="text-[11px] text-muted-foreground flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-emerald-500 shrink-0 animate-pulse"></span>
                修改提交后全站配置即时同步生效
              </span>
              <Button @click="saveSettings" :disabled="loading" class="h-8.5 px-5 text-xs font-medium shadow-sm w-full sm:w-auto justify-center">
                {{ loading ? '保存中...' : '保存站点配置' }}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <!-- OpenAPI Token 重新生成确认弹窗 -->
    <AlertDialog :open="showOpenapiConfirmDialog" @update:open="showOpenapiConfirmDialog = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="flex items-center gap-2">
            <AlertTriangle class="w-5 h-5 text-amber-500" />
            确认重新生成 Token？
          </AlertDialogTitle>
          <AlertDialogDescription>
            此操作将立刻覆盖当前配置框内的 OpenAPI Token，原有的 Token 在点击【保存设置】后将会永久失效，导致所有使用旧 Token 的外部系统无法访问。确认要继续吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="generateOpenapiToken">重新生成</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<style scoped>
:deep(input[type='number']::-webkit-inner-spin-button),
:deep(input[type='number']::-webkit-outer-spin-button) {
  -webkit-appearance: none !important;
  margin: 0 !important;
  display: none !important;
}
:deep(input[type='number']) {
  -moz-appearance: textfield !important;
  appearance: textfield !important;
}
</style>
