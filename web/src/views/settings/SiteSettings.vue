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
  Sparkles,
  Cpu,
  KeyRound
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
import { useRegisterSettingSection, useFieldDiff } from './composables/useSettingsTracker'
import ModifiedBadge from './components/ModifiedBadge.vue'
import { withDemoGuard } from '@/utils/demo'

const { refreshSettings, siteSettings } = useSiteSettings()

const isDemoMode = computed(() => {
  return form.value.demo_mode === 'true' ||
    (form.value as any).demo_mode === true ||
    siteSettings.value?.demo_mode === 'true' ||
    (siteSettings.value as any)?.demo_mode === true
})

const initialForm = ref<SiteSettings | null>(null)

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
      openapi_enabled: res.openapi_enabled === true || (res as any).openapi_enabled === 'true',
      demo_mode: res.demo_mode
    }
    initialForm.value = JSON.parse(JSON.stringify(form.value))
  } catch { }
}

// 变动项比对与单字段追踪
const { isDiff, getOriginal, resetField, changedFields, diffClass } = useFieldDiff(form, initialForm)
const isDirty = computed(() => changedFields.value.length > 0)
const dirtyCount = computed(() => changedFields.value.length)

function resetForm() {
  if (!initialForm.value) return
  form.value = JSON.parse(JSON.stringify(initialForm.value))
  toast.info('已还原站点配置')
}

// 注册至全局设置变动追踪器
useRegisterSettingSection({
  id: 'site',
  tab: 'site',
  title: '站点与系统配置',
  isDirty,
  dirtyCount,
  saving: loading,
  save: saveSettings,
  reset: resetForm
})

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

const generateOpenapiToken = withDemoGuard(async () => {
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
}, '演示模式下禁止生成 OpenAPI Token')

const copyOpenapiToken = withDemoGuard(async () => {
  if (!form.value.openapi_token) return
  const success = await copyTextToClipboard(form.value.openapi_token)
  if (success) {
    toast.success('Token 已复制到剪贴板')
  } else {
    toast.error('复制失败，请手动复制')
  }
}, '演示模式下禁止操作 OpenAPI Token')

function openSwaggerDocs() {
  window.open('https://engigu.github.io/baihu-panel/guide/api.html', '_blank')
}

function openMcpDocs() {
  window.open('https://engigu.github.io/baihu-panel/guide/mcp.html', '_blank')
}

const showMcpSnippet = ref(false)
const mcpProtocol = ref<'streamable' | 'sse'>('streamable')

const mcpStreamableUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  const base = window.location.origin
  const token = form.value.openapi_token || '<Token>'
  return `${base}/open2api/v1/mcp?token=${token}`
})

const mcpSseUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  const base = window.location.origin
  const token = form.value.openapi_token || '<Token>'
  return `${base}/open2api/v1/mcp/sse?token=${token}`
})

const currentMcpUrl = computed(() => {
  return mcpProtocol.value === 'streamable' ? mcpStreamableUrl.value : mcpSseUrl.value
})

const mcpConfigJson = computed(() => {
  return JSON.stringify({
    mcpServers: {
      "baihu": {
        "url": currentMcpUrl.value
      }
    }
  }, null, 2)
})

const copyMcpUrl = withDemoGuard(async () => {
  if (!form.value.openapi_token) {
    toast.error('请先生成并保存 OpenAPI Token')
    return
  }
  const success = await copyTextToClipboard(currentMcpUrl.value)
  if (success) {
    const protoName = mcpProtocol.value === 'streamable' ? 'Streamable HTTP' : 'SSE'
    toast.success(`MCP 连接地址 (${protoName}) 已复制到剪贴板`)
  } else {
    toast.error('复制失败，请手动复制')
  }
}, '演示模式下禁止使用 MCP 能力')

const copyMcpConfigJson = withDemoGuard(async () => {
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
}, '演示模式下禁止使用 MCP 能力')

const handleTrimCache = withDemoGuard(async () => {
  trimming.value = true
  try {
    const res = await api.settings.trimCache()
    toast.success(res.message || '内存与文件缓存回收成功')
  } catch (err: any) {
    toast.error(err?.message || '触发缓存回收失败')
  } finally {
    trimming.value = false
  }
}, '演示模式下禁止清理缓存')

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
      <!-- 左列：站点外观 + OpenAPI 接口集成 -->
      <div class="space-y-6">
        <!-- 模块 1: 站点外观与基础配置 -->
        <Card class="shadow-sm">
          <CardHeader class="pb-4 px-4 sm:px-6">
            <CardTitle class="flex items-center gap-2 text-base">
              <Globe class="w-4 h-4 text-primary" />
              <span>站点外观与基础配置</span>
            </CardTitle>
            <CardDescription>自定义站点品牌标识、标题标语及系统常规运行参数</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4 px-4 sm:px-6">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                <span>站点标题</span>
                <ModifiedBadge
                  v-if="isDiff('title')"
                  :original-value="getOriginal('title')"
                  @reset="resetField('title')"
                />
              </Label>
              <Input v-model="form.title" placeholder="白虎面板" class="h-9" :class="diffClass('title')" />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                <span>站点标语</span>
                <ModifiedBadge
                  v-if="isDiff('subtitle')"
                  :original-value="getOriginal('subtitle')"
                  @reset="resetField('subtitle')"
                />
              </Label>
              <Input v-model="form.subtitle" placeholder="轻量级定时任务管理系统" class="h-9" :class="diffClass('subtitle')" />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                <span>站点图标 (SVG 代码)</span>
                <ModifiedBadge
                  v-if="isDiff('icon')"
                  :original-value="getOriginal('icon')"
                  @reset="resetField('icon')"
                />
              </Label>
              <div class="flex items-center gap-2">
                <Input v-model="form.icon" placeholder="<svg>...</svg>" class="flex-1 font-mono text-xs h-9" :class="diffClass('icon')" />
                <div
                  v-if="iconPreview"
                  class="p-1.5 border rounded-lg bg-card w-9 h-9 flex items-center justify-center shrink-0 [&>svg]:w-5 [&>svg]:h-5 shadow-sm"
                  v-html="iconPreview"
                />
              </div>
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                <span>会话保持天数 (Cookie 过期时间)</span>
                <ModifiedBadge
                  v-if="isDiff('cookie_days')"
                  :original-value="getOriginal('cookie_days')"
                  @reset="resetField('cookie_days')"
                />
              </Label>
              <div class="relative">
                <Input
                  v-model="form.cookie_days"
                  type="number"
                  class="h-9 pr-14 text-sm [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  :class="diffClass('cookie_days')"
                />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">天过期</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- 模块 2: OpenAPI 开放接口能力 -->
        <Card class="shadow-sm">
          <CardHeader class="pb-3.5 space-y-1.5 px-4 sm:px-6">
            <div class="flex items-start justify-between gap-3">
              <div class="space-y-1 min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <CardTitle class="flex items-center gap-2 text-sm sm:text-base font-semibold">
                    <Code2 class="w-4 h-4 text-sky-500 shrink-0" />
                    <span class="whitespace-nowrap font-bold">OpenAPI 接口与集成</span>
                  </CardTitle>
                  <Badge
                    v-if="isDemoMode"
                    variant="destructive"
                    class="text-[10px] px-1.5 py-0 font-normal shrink-0"
                  >
                    演示模式禁用
                  </Badge>
                  <Badge v-else variant="secondary" class="font-normal text-[10px] bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20 shrink-0">
                    推荐接入
                  </Badge>
                </div>
                <CardDescription class="text-xs text-muted-foreground pt-0.5">
                  提供标准 RESTful API 与智能体 MCP 协议接入
                </CardDescription>
              </div>
              <div class="flex items-center gap-2 shrink-0 pt-0.5">
                <ModifiedBadge
                  v-if="isDiff('openapi_enabled')"
                  :original-value="getOriginal('openapi_enabled')"
                  @reset="resetField('openapi_enabled')"
                />
                <Switch v-model="form.openapi_enabled" id="openapi-enabled" :disabled="isDemoMode" />
                <Label for="openapi-enabled" class="text-xs cursor-pointer select-none">
                  {{ form.openapi_enabled ? '已开启' : '开启' }}
                </Label>
              </div>
            </div>
          </CardHeader>
          <CardContent class="space-y-3.5 px-4 sm:px-6">
            <!-- 演示模式警示条 -->
            <div v-if="isDemoMode" class="flex items-center gap-2 p-2.5 rounded-xl bg-destructive/10 text-destructive text-xs">
              <AlertTriangle class="h-4 w-4 shrink-0" />
              <span>演示模式下已锁定 OpenAPI 接口服务与访问凭据。</span>
            </div>

            <!-- 调用规范看板：将散乱请求头说明与 API 文档链接一体化融合 -->
            <div class="p-2.5 rounded-xl border bg-muted/30 border-border/70 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs">
              <div class="flex items-center gap-2 min-w-0">
                <ShieldCheck class="w-4 h-4 text-sky-500 shrink-0" />
                <span class="text-muted-foreground text-[11px] shrink-0">鉴权请求头:</span>
                <code class="text-[11px] font-mono text-foreground bg-background px-1.5 py-0.5 rounded border border-border/40 font-medium truncate select-all">
                  Authorization: Bearer &lt;Token&gt;
                </code>
              </div>
              <a
                href="#"
                @click.prevent="openSwaggerDocs"
                class="flex items-center gap-1 text-[11px] text-sky-600 dark:text-sky-400 hover:underline shrink-0 self-end sm:self-auto font-medium"
              >
                <span>接口 API 文档</span>
                <ExternalLink class="w-2.5 h-2.5" />
              </a>
            </div>

            <!-- 凭据表单项 -->
            <div class="space-y-3">
              <div class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
                    <KeyRound class="w-3.5 h-3.5 text-muted-foreground" />
                    <span>Token 访问密钥</span>
                  </Label>
                  <div class="flex items-center gap-2">
                    <ModifiedBadge
                      v-if="isDiff('openapi_token')"
                      :original-value="getOriginal('openapi_token')"
                      @reset="resetField('openapi_token')"
                    />
                    <span v-if="form.openapi_token" class="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium flex items-center gap-1">
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                      <span>已生成凭据</span>
                    </span>
                    <span v-else class="text-[10px] text-amber-500">
                      未生成凭据
                    </span>
                  </div>
                </div>
                <div class="flex items-center space-x-2">
                  <Input
                    v-model="form.openapi_token"
                    :placeholder="isDemoMode ? '演示模式下已锁定访问凭据' : '点击右侧按钮生成 32 位 Token'"
                    class="text-xs sm:text-sm h-9 flex-1 min-w-0"
                    :class="[
                      diffClass('openapi_token'),
                      form.openapi_token ? 'font-mono' : 'demo-placeholder-text placeholder:text-muted-foreground/70'
                    ]"
                    :disabled="isDemoMode || !form.openapi_enabled"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    class="h-9 w-9 shrink-0 shadow-xs"
                    @click="isDemoMode ? toast.error('演示模式下禁止生成 OpenAPI Token') : (showOpenapiConfirmDialog = true)"
                    :title="isDemoMode ? '演示模式禁用' : '随机生成全新 Token'"
                    :disabled="isDemoMode || !form.openapi_enabled"
                  >
                    <RefreshCw class="w-4 h-4" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    class="h-9 w-9 shrink-0 shadow-xs"
                    @click="copyOpenapiToken"
                    :title="isDemoMode ? '演示模式禁用' : '复制 Token'"
                    :disabled="isDemoMode || !form.openapi_token || !form.openapi_enabled"
                  >
                    <Copy class="w-4 h-4" />
                  </Button>
                </div>
              </div>

              <div class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
                    <Clock class="w-3.5 h-3.5 text-muted-foreground" />
                    <span>截止有效期</span>
                  </Label>
                  <div class="flex items-center gap-2">
                    <ModifiedBadge
                      v-if="isDiff('openapi_token_expire')"
                      :original-value="getOriginal('openapi_token_expire')"
                      @reset="resetField('openapi_token_expire')"
                    />
                    <span class="text-[10px] text-muted-foreground">置空代表永不过期</span>
                  </div>
                </div>
                <Input
                  v-model="form.openapi_token_expire"
                  type="date"
                  class="w-full dark:[color-scheme:dark] h-9 text-xs sm:text-sm"
                  :class="diffClass('openapi_token_expire')"
                  :disabled="isDemoMode || !form.openapi_enabled"
                />
              </div>
            </div>

            <!-- 模块 2.1: 模型上下文协议 (MCP) 快捷连接指引 -->
            <div class="pt-2 border-t border-border/60">
              <div class="p-3.5 rounded-xl border bg-gradient-to-br from-purple-500/[0.03] via-background to-sky-500/[0.03] border-border/70 space-y-3">
                <div class="flex items-center justify-between gap-2 flex-wrap">
                  <div class="flex items-center gap-1.5 shrink-0">
                    <Sparkles class="w-3.5 h-3.5 text-purple-400/80 shrink-0" />
                    <span class="text-xs font-bold text-foreground whitespace-nowrap">AI 客户端连接 (MCP)</span>
                    <Badge
                      v-if="isDemoMode"
                      variant="destructive"
                      class="text-[9px] px-1.5 py-0 h-4 font-normal shrink-0"
                    >
                      演示模式禁用
                    </Badge>
                    <Badge v-else variant="outline" class="font-normal text-[9px] px-1.5 py-0 h-4 border-border/70 bg-muted/40 text-muted-foreground shrink-0">
                      Cursor / Claude
                    </Badge>
                  </div>
                  <div class="flex items-center gap-2.5 shrink-0 text-xs">
                    <a
                      href="#"
                      @click.prevent="openMcpDocs"
                      class="text-[11px] text-muted-foreground hover:text-foreground hover:underline flex items-center gap-0.5 transition-colors"
                    >
                      <span>MCP 指南</span>
                      <ExternalLink class="w-2.5 h-2.5" />
                    </a>
                    <span class="text-border/80">|</span>
                    <button
                      type="button"
                      @click="showMcpSnippet = !showMcpSnippet"
                      class="text-[11px] text-muted-foreground hover:text-foreground hover:underline flex items-center gap-0.5 transition-colors"
                    >
                      <span>{{ showMcpSnippet ? '收起配置' : '配置示例' }}</span>
                    </button>
                  </div>
                </div>

                <!-- 演示模式警示条 -->
                <div v-if="isDemoMode" class="flex items-center gap-2 p-2 rounded-lg bg-destructive/10 text-destructive text-[11px]">
                  <AlertTriangle class="h-3.5 w-3.5 shrink-0" />
                  <span>演示模式下已禁用 MCP 远程协议连接与数据访问能力。</span>
                </div>

                <!-- 传输协议选择切换器 -->
                <div class="space-y-1.5 pt-0.5">
                  <div class="grid grid-cols-2 sm:inline-flex p-0.5 rounded-lg bg-muted/60 border border-border/40 text-[10px] w-full sm:w-auto">
                    <button
                      type="button"
                      @click="mcpProtocol = 'streamable'"
                      :class="[
                        'px-2 py-1.5 sm:py-0.5 rounded-md font-medium transition-colors text-center flex items-center justify-center',
                        mcpProtocol === 'streamable'
                          ? 'bg-background text-foreground shadow-sm'
                          : 'text-muted-foreground hover:text-foreground'
                      ]"
                    >
                      <span class="whitespace-nowrap">Streamable HTTP (推荐)</span>
                    </button>
                    <button
                      type="button"
                      @click="mcpProtocol = 'sse'"
                      :class="[
                        'px-2 py-1.5 sm:py-0.5 rounded-md font-medium transition-colors text-center flex items-center justify-center',
                        mcpProtocol === 'sse'
                          ? 'bg-background text-foreground shadow-sm'
                          : 'text-muted-foreground hover:text-foreground'
                      ]"
                    >
                      <span class="whitespace-nowrap">HTTP + SSE (兼容旧版)</span>
                    </button>
                  </div>
                </div>

                <div class="space-y-1.5">
                  <div class="text-[11px] text-muted-foreground flex items-center justify-between gap-2 flex-wrap">
                    <div class="flex items-center gap-1 min-w-0">
                      <span>连接 URL</span>
                      <span class="font-mono text-[10px] text-muted-foreground/80">
                        ({{ mcpProtocol === 'streamable' ? '/open2api/v1/mcp' : '/open2api/v1/mcp/sse' }})
                      </span>
                    </div>
                    <span v-if="isDemoMode" class="text-destructive text-[10px] shrink-0">演示模式下不可用</span>
                    <span v-else-if="!form.openapi_token" class="text-amber-500 text-[10px] shrink-0">需先生成并保存 Token</span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <Input
                      :model-value="isDemoMode ? '演示模式下已禁用 MCP 协议与远程连接' : currentMcpUrl"
                      :placeholder="isDemoMode ? '演示模式下已禁用 MCP 协议与远程连接' : currentMcpUrl"
                      readonly
                      :tabindex="isDemoMode ? -1 : 0"
                      :class="[
                        'text-xs h-8.5 flex-1 min-w-0 transition-colors',
                        isDemoMode
                          ? 'demo-placeholder-text text-muted-foreground/75 dark:text-muted-foreground/70 bg-muted/20 border-border/60 cursor-not-allowed select-none caret-transparent'
                          : 'font-mono text-foreground bg-muted/40 select-all'
                      ]"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      class="h-8.5 px-2.5 shrink-0 text-xs gap-1"
                      @click="copyMcpUrl"
                      :disabled="isDemoMode || !form.openapi_token || !form.openapi_enabled"
                      :title="isDemoMode ? '演示模式下禁止使用 MCP 能力' : '复制'"
                    >
                      <Copy class="w-3 h-3" />
                      <span>复制</span>
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
                      :disabled="isDemoMode"
                      class="text-[10px] text-muted-foreground hover:text-foreground hover:underline flex items-center gap-1 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      <Copy class="w-2.5 h-2.5" />
                      <span>复制配置 JSON</span>
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
          <CardHeader class="pb-4 px-4 sm:px-6">
            <CardTitle class="flex items-center gap-2 text-base">
              <Database class="w-4 h-4 text-indigo-500" />
              <span>日志生命周期与自动清理</span>
            </CardTitle>
            <CardDescription>按保留天数与条数双维度监控，定期自动清理历史过期数据</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4 px-4 sm:px-6">
            <!-- 4 大日志清理条目（响应式单列/双行适配，小屏下自动折行不挤压） -->
            <div class="space-y-2.5">
              <!-- 系统通知 -->
              <div
                class="p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                :class="isDiff('system_notice_days') || isDiff('system_notice_max_count') ? 'border-amber-500/40 bg-amber-500/[0.04]' : 'bg-muted/20 border-border/70'"
              >
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400 shrink-0">
                    <Bell class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-semibold text-foreground truncate">系统通知清理</span>
                      <ModifiedBadge
                        v-if="isDiff('system_notice_days') || isDiff('system_notice_max_count')"
                        :show-reset="false"
                      />
                    </div>
                    <div class="text-[10px] text-muted-foreground truncate">站内消息通知及系统公告</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.system_notice_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('system_notice_days')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.system_notice_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('system_notice_max_count')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 推送日志 -->
              <div
                class="p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                :class="isDiff('push_log_days') || isDiff('push_log_max_count') ? 'border-amber-500/40 bg-amber-500/[0.04]' : 'bg-muted/20 border-border/70'"
              >
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-violet-500/10 text-violet-600 dark:text-violet-400 shrink-0">
                    <Send class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-semibold text-foreground truncate">推送日志清理</span>
                      <ModifiedBadge
                        v-if="isDiff('push_log_days') || isDiff('push_log_max_count')"
                        :show-reset="false"
                      />
                    </div>
                    <div class="text-[10px] text-muted-foreground truncate">通道消息分发与推送结果</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.push_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('push_log_days')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.push_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('push_log_max_count')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 登录日志 -->
              <div
                class="p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                :class="isDiff('login_log_days') || isDiff('login_log_max_count') ? 'border-amber-500/40 bg-amber-500/[0.04]' : 'bg-muted/20 border-border/70'"
              >
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shrink-0">
                    <ShieldCheck class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-semibold text-foreground truncate">登录日志清理</span>
                      <ModifiedBadge
                        v-if="isDiff('login_log_days') || isDiff('login_log_max_count')"
                        :show-reset="false"
                      />
                    </div>
                    <div class="text-[10px] text-muted-foreground truncate">用户登录认证与安全审计</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.login_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('login_log_days')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.login_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('login_log_max_count')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">条</span>
                  </div>
                </div>
              </div>

              <!-- 调度日志 -->
              <div
                class="p-3 rounded-xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                :class="isDiff('scheduler_log_days') || isDiff('scheduler_log_max_count') ? 'border-amber-500/40 bg-amber-500/[0.04]' : 'bg-muted/20 border-border/70'"
              >
                <div class="flex items-center gap-2.5 min-w-0">
                  <div class="p-1.5 rounded-lg bg-amber-500/10 text-amber-600 dark:text-amber-400 shrink-0">
                    <CalendarClock class="w-4 h-4" />
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-semibold text-foreground truncate">调度日志清理</span>
                      <ModifiedBadge
                        v-if="isDiff('scheduler_log_days') || isDiff('scheduler_log_max_count')"
                        :show-reset="false"
                      />
                    </div>
                    <div class="text-[10px] text-muted-foreground truncate">定时任务触发与运行记录</div>
                  </div>
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
                  <div class="relative flex-1 sm:w-24">
                    <Input
                      v-model="form.scheduler_log_days"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('scheduler_log_days')"
                      min="0"
                    />
                    <span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground pointer-events-none">天</span>
                  </div>
                  <div class="relative flex-1 sm:w-32">
                    <Input
                      v-model="form.scheduler_log_max_count"
                      type="number"
                      class="h-8 pr-7 text-xs"
                      :class="diffClass('scheduler_log_max_count')"
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
          <CardHeader class="pb-4 space-y-2 px-4 sm:px-6">
            <div class="flex items-center justify-between gap-3">
              <CardTitle class="flex items-center gap-2 text-sm sm:text-base font-semibold min-w-0">
                <Cpu class="w-4 h-4 text-emerald-500 shrink-0" />
                <span class="whitespace-nowrap">容器内存与缓存优化</span>
                <Badge
                  v-if="isDemoMode"
                  variant="destructive"
                  class="text-[10px] px-1.5 py-0 font-normal shrink-0"
                >
                  演示模式禁用
                </Badge>
              </CardTitle>
              <div class="flex items-center gap-2 shrink-0">
                <ModifiedBadge
                  v-if="isDiff('cache_trim_enabled')"
                  :original-value="getOriginal('cache_trim_enabled')"
                  @reset="resetField('cache_trim_enabled')"
                />
                <Switch v-model="form.cache_trim_enabled" id="cache-trim-enabled" :disabled="isDemoMode" />
                <Label for="cache-trim-enabled" class="text-xs cursor-pointer select-none">开启自适应</Label>
              </div>
            </div>
            <CardDescription class="text-xs">
              监控容器内存水位与文件缓存，在内存超标或任务执行完毕后自动安全释放
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4 px-4 sm:px-6">
            <!-- 演示模式警示条 -->
            <div v-if="isDemoMode" class="flex items-center gap-2 p-2.5 rounded-xl bg-destructive/10 text-destructive text-xs">
              <AlertTriangle class="h-4 w-4 shrink-0" />
              <span>演示模式下已锁定容器内存水位与文件缓存配置修改。</span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <!-- 警戒水位线 -->
              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                  <span>内存警戒水位</span>
                  <ModifiedBadge
                    v-if="isDiff('mem_watermark_rate')"
                    :original-value="getOriginal('mem_watermark_rate')"
                    @reset="resetField('mem_watermark_rate')"
                  />
                </Label>
                <div class="relative">
                  <Input
                    v-model="form.mem_watermark_rate"
                    type="number"
                    class="h-9 pr-8 text-sm"
                    :class="diffClass('mem_watermark_rate')"
                    min="10"
                    max="99"
                    :disabled="isDemoMode || !form.cache_trim_enabled"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">%</span>
                </div>
                <p class="text-[10px] text-muted-foreground">容器内存达此比例时介入 (推荐 70-90%)</p>
              </div>

              <!-- 缓存释放阈值 -->
              <div class="space-y-1.5">
                <Label class="text-xs font-medium text-foreground flex items-center justify-between">
                  <span>缓存触发阈值</span>
                  <ModifiedBadge
                    v-if="isDiff('cache_max_mb')"
                    :original-value="getOriginal('cache_max_mb')"
                    @reset="resetField('cache_max_mb')"
                  />
                </Label>
                <div class="relative">
                  <Input
                    v-model="form.cache_max_mb"
                    type="number"
                    class="h-9 pr-10 text-sm"
                    :class="diffClass('cache_max_mb')"
                    min="10"
                    :disabled="isDemoMode || !form.cache_trim_enabled"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">MB</span>
                </div>
                <p class="text-[10px] text-muted-foreground">限额容器超此阈值介入 (默认 60MB，无限额时 512MB)</p>
              </div>
            </div>

            <!-- 任务结束回收与手动触发行 -->
            <div class="p-3 rounded-xl border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shrink-0">
                  <Sparkles class="w-4 h-4" />
                </div>
                <div class="min-w-0">
                  <div class="text-xs font-semibold text-foreground truncate">任务结束工作区清理</div>
                  <div class="text-[10px] text-muted-foreground truncate">任务平息后局部清理工作区产物（基础语言底座保持内存常驻防 IO 震荡）</div>
                </div>
              </div>
              <div class="flex items-center gap-3 shrink-0 self-end sm:self-auto">
                <div class="flex items-center gap-1.5">
                  <ModifiedBadge
                    v-if="isDiff('task_finished_trim')"
                    :original-value="getOriginal('task_finished_trim')"
                    @reset="resetField('task_finished_trim')"
                  />
                  <Switch v-model="form.task_finished_trim" id="task-finished-trim" :disabled="isDemoMode || !form.cache_trim_enabled" />
                  <Label for="task-finished-trim" class="text-xs cursor-pointer select-none">自动</Label>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="h-8 px-2.5 text-xs gap-1"
                  @click="handleTrimCache"
                  :disabled="isDemoMode || trimming"
                  :title="isDemoMode ? '演示模式下禁止清理缓存' : '立即清理'"
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
                  基础语言环境（Python/Node/Mise等）始终保持内存常驻以保障任务秒级启动与 0 读盘；仅在内存超标或任务执行完成后对其局部工作区安全防抖清理（30秒冷却），避免容器内存与临时产物堆积。
                </div>
              </div>
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

:deep(.demo-placeholder-text),
:deep(.demo-placeholder-text::placeholder) {
  font-family: var(--font-main) !important;
  font-size: 11.5px !important;
  font-weight: 400 !important;
  letter-spacing: 0.015em !important;
  -webkit-font-smoothing: antialiased !important;
  -moz-osx-font-smoothing: grayscale !important;
  text-rendering: optimizeLegibility !important;
}
</style>
