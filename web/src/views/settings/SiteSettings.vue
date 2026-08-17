<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { api, type SiteSettings } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { copyToClipboard as copyTextToClipboard } from '@/utils/clipboard'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { RefreshCw, Copy, AlertTriangle, ExternalLink, Info, Clock } from 'lucide-vue-next'
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
  scheduler_log_max_count: '10000'
})
const loading = ref(false)
const showOpenapiConfirmDialog = ref(false)

const iconPreview = computed(() => {
  if (!form.value.icon) return ''
  // 简单验证是否是 SVG
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
      scheduler_log_max_count: String(form.value.scheduler_log_max_count || '10000')
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

    // 如果没有设置过期时间，默认给一年后
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

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-6">
    <!-- 站点基础参数 -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">站点标题</Label>
        <Input v-model="form.title" placeholder="白虎面板" class="h-9" />
      </div>
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">站点标语</Label>
        <Input v-model="form.subtitle" placeholder="轻量级定时任务管理系统" class="h-9" />
      </div>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">站点图标 (SVG 代码)</Label>
        <div class="flex items-center gap-2">
          <Input v-model="form.icon" placeholder="<svg>...</svg>" class="flex-1 font-mono text-xs h-9" />
          <div v-if="iconPreview"
            class="p-1.5 border rounded bg-white dark:bg-white w-9 h-9 flex items-center justify-center shrink-0 [&>svg]:w-5 [&>svg]:h-5"
            v-html="iconPreview" />
        </div>
      </div>
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">系统常规配置</Label>
        <div class="grid grid-cols-2 gap-3">
          <div class="relative">
            <Input v-model="form.page_size" type="number" class="h-9 pr-12 text-sm" />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">条/页</span>
          </div>
          <div class="relative">
            <Input v-model="form.cookie_days" type="number" class="h-9 pr-14 text-sm" />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">天过期</span>
          </div>
        </div>
      </div>
    </div>

    <div class="pt-6 border-t mt-6">
      <h3 class="text-lg font-medium text-foreground mb-1">日志清理策略</h3>
      <p class="text-xs text-muted-foreground mb-4">自动清理超过指定天数或数量的日志记录，保持系统性能。</p>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">系统通知清理</Label>
          <div class="grid grid-cols-2 gap-3">
            <div class="relative">
              <Input v-model="form.system_notice_days" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">天清理</span>
            </div>
            <div class="relative">
              <Input v-model="form.system_notice_max_count" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">条保留</span>
            </div>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">推送日志清理</Label>
          <div class="grid grid-cols-2 gap-3">
            <div class="relative">
              <Input v-model="form.push_log_days" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">天清理</span>
            </div>
            <div class="relative">
              <Input v-model="form.push_log_max_count" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">条保留</span>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">登录日志清理</Label>
          <div class="grid grid-cols-2 gap-3">
            <div class="relative">
              <Input v-model="form.login_log_days" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">天清理</span>
            </div>
            <div class="relative">
              <Input v-model="form.login_log_max_count" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">条保留</span>
            </div>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">调度日志清理</Label>
          <div class="grid grid-cols-2 gap-3">
            <div class="relative">
              <Input v-model="form.scheduler_log_days" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">天清理</span>
            </div>
            <div class="relative">
              <Input v-model="form.scheduler_log_max_count" type="number" class="h-9 pr-14 text-sm" min="0" />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">条保留</span>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-6 p-4 bg-muted/30 rounded-lg border border-dashed border-border flex flex-col gap-3">
        <div class="flex items-start gap-3">
          <div class="p-1.5 bg-blue-500/10 rounded-full">
            <Info class="w-4 h-4 text-blue-600 dark:text-blue-400" />
          </div>
          <div class="space-y-1">
            <p class="text-sm font-medium">双重维度限制</p>
            <p class="text-xs text-muted-foreground leading-relaxed">
              系统将根据天数和数量同时进行监测。满足任一条件即执行清理：超过天数的旧数据将被物理删除；若日志总数超过限制条数，则自动剔除最早产生的记录。
            </p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <div class="p-1.5 bg-amber-500/10 rounded-full">
            <Clock class="w-4 h-4 text-amber-600 dark:text-amber-400" />
          </div>
          <div class="space-y-1">
            <p class="text-sm font-medium">执行周期说明</p>
            <p class="text-xs text-muted-foreground leading-relaxed">
              清理任务在白虎面板后端服务启动时立即执行一次。在运行期间，系统将自动开启后台巡检计数器，每隔 1 小时进行周期性自动清理。
            </p>
          </div>
        </div>
      </div>
    </div>

    <div class="pt-6 border-t mt-6">
      <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
        <div class="flex items-center gap-2">
          <h3 class="text-lg font-medium text-foreground whitespace-nowrap">OpenAPI Token</h3>
          <Badge variant="secondary"
            class="font-normal text-xs bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20 whitespace-nowrap">
            推荐方式</Badge>
        </div>
        <div class="flex items-center justify-between sm:justify-end w-full sm:w-auto gap-4">
          <a href="#" @click.prevent="openSwaggerDocs"
            class="flex items-center gap-1 text-xs text-blue-600 hover:underline shrink-0">
            查看接口文档
            <ExternalLink class="w-3 h-3" />
          </a>
          <div class="flex items-center gap-2 shrink-0">
            <Switch v-model="form.openapi_enabled" id="openapi-enabled" />
            <Label for="openapi-enabled" class="text-xs cursor-pointer">开启 OpenAPI</Label>
          </div>
        </div>
      </div>
      <p class="text-xs text-muted-foreground mb-4 leading-relaxed">开启全局 OpenAPI 直接访问能力，配置后可通过请求头 <code
          class="bg-muted px-1.5 py-0.5 rounded text-[11px] select-all font-sans">Authorization: Bearer &lt;在此生成的Token&gt;</code>
        以第三方身份调用系统的所有接口，请妥善保管 Token 并设置合理的有效期。<span
          class="text-amber-600 dark:text-amber-500 font-medium">注意：必须先开启本功能才能查看接口文档页面和对接调用。</span></p>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">Token 密钥</Label>
          <div class="flex items-center space-x-2">
            <Input v-model="form.openapi_token" placeholder="点击右侧按钮生成 32 位随机 Token" class="text-sm h-9" />
            <Button type="button" variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="showOpenapiConfirmDialog = true" title="随机生成">
              <RefreshCw class="w-4 h-4" />
            </Button>
            <Button type="button" variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="copyOpenapiToken" title="复制"
              :disabled="!form.openapi_token">
              <Copy class="w-4 h-4" />
            </Button>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">截止有效期</Label>
          <Input v-model="form.openapi_token_expire" type="date" class="w-full dark:[color-scheme:dark] h-9" />
          <p class="text-[10px] text-muted-foreground">超过此日期后该 Token 将失效，置空代表该特性完全关闭</p>
        </div>
      </div>
    </div>

    <div class="flex justify-end pt-2">
      <Button @click="saveSettings" :disabled="loading">
        {{ loading ? '保存中...' : '保存设置' }}
      </Button>
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
