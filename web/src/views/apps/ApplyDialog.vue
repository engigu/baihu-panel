<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Eye, EyeOff, Loader2, Terminal, CheckCircle2, Check, AlertCircle, AlertTriangle, Sparkles, Package, ExternalLink, ChevronDown, ChevronRight, Maximize2, Minimize2, X } from 'lucide-vue-next'
import { api, type MarketplaceApp, type AppEnvItemSchema, type AppScenarioSchema, type ApplyAppPayload, type Task } from '@/api'
import { toast } from 'vue-sonner'
import TaskNotificationConfig from '@/views/tasks/components/TaskNotificationConfig.vue'
import TaskAdvancedConfig from '@/views/tasks/components/TaskAdvancedConfig.vue'
import TaskCronConfig from '@/views/tasks/components/TaskCronConfig.vue'

function formatDate(dateStr?: string) {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  } catch {
    return dateStr
  }
}

const demoMode = ref(false)
onMounted(async () => {
  try {
    const publicSite = await api.settings.getPublicSite()
    demoMode.value = publicSite.demo_mode || false
  } catch {}
})

const props = withDefaults(
  defineProps<{
    open: boolean
    targetApp?: MarketplaceApp | null
    task?: Partial<Task> | null
    mode?: 'deploy' | 'edit_task'
  }>(),
  {
    targetApp: null,
    task: null,
    mode: 'deploy'
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  'success': []
}>()

const activeTab = ref<'url' | 'yaml'>('url')
const manifestUrl = ref('')
const manifestYaml = ref('')

// 安装参数配置
const selectedScenario = ref('')
const envForm = ref<Record<string, any>>({})
const showSecrets = ref<Record<string, boolean>>({})

// 高级选项
const showAdvanced = ref(true)
const forceSetup = ref(false)
const skipSetup = ref(false)
const skipSync = ref(false)
const enableTelemetry = ref(true)

// 调度与通知策略配置
const form = ref<Partial<Task>>({
  schedule: '',
  random_range: 0,
  timeout: 30,
  retry_count: 0,
  retry_interval: 0,
  clean_config: '',
  unified_config: ''
})
const notificationConfigRef = ref<any>(null)

// 部署过程状态
const deploying = ref(false)
const deployLog = ref('')
const deploySuccess = ref(false)
const deployError = ref('')
const isFullscreenLog = ref(false)

function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && isFullscreenLog.value) {
    isFullscreenLog.value = false
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('keydown', handleKeyDown)
}

// 动态场景列表
const availableScenarios = computed<AppScenarioSchema[]>(() => {
  if (props.targetApp?.scenarios && props.targetApp.scenarios.length > 0) {
    return props.targetApp.scenarios
  }
  return []
})

// 判断应用是否已安装
const isInstalled = computed(() => {
  if (props.task) return true
  if (props.targetApp?.is_installed) return true
  if (props.targetApp?.current_scenario) return true
  return false
})

// 动态环境变量列表
const envSchemaList = computed<AppEnvItemSchema[]>(() => {
  if (props.targetApp?.env_schema && props.targetApp.env_schema.length > 0) {
    return props.targetApp.env_schema
  }
  return []
})

const imageLoadError = ref(false)

// 监听目标应用变更，初始化表单
watch(
  () => props.targetApp,
  (app) => {
    deployLog.value = ''
    deploySuccess.value = false
    deployError.value = ''
    envForm.value = {}
    showSecrets.value = {}
    imageLoadError.value = false
    isFullscreenLog.value = false
    forceSetup.value = false
    skipSetup.value = false
    skipSync.value = false

    if (app) {
      // 回显已保存的高级构建配置
      if (app.build_opts) {
        forceSetup.value = Boolean(app.build_opts.force_setup)
        skipSetup.value = Boolean(app.build_opts.skip_setup)
        skipSync.value = Boolean(app.build_opts.skip_sync)
      }

      // 初始化场景（优先使用已保存的当前场景）
      const savedSc = app.scenarios?.find(s => s.id === app.current_scenario)
      const defaultSc = app.scenarios?.find(s => s.default) || app.scenarios?.[0]
      selectedScenario.value = savedSc?.id || defaultSc?.id || ''

      // 初始化环境变量表单默认值（优先使用已保存的环境变量值）
      if (app.env_schema) {
        for (const item of app.env_schema) {
          if (app.env_values && app.env_values[item.key] !== undefined) {
            envForm.value[item.key] = app.env_values[item.key]
          } else if (item.default !== undefined && item.default !== null) {
            envForm.value[item.key] = item.type === 'boolean' ? Boolean(item.default) : String(item.default)
          } else if (item.type === 'boolean') {
            envForm.value[item.key] = false
          } else {
            envForm.value[item.key] = ''
          }
        }
      }

      // 如果有预置的 manifest_url
      if (app.manifest_url) {
        manifestUrl.value = app.manifest_url
      } else {
        manifestUrl.value = `https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps/${app.id}/app.yaml`
      }
    } else {
      selectedScenario.value = ''
      manifestUrl.value = ''
      manifestYaml.value = ''
    }
  },
  { immediate: true }
)

watch(
  [() => props.open, () => props.task, () => props.targetApp, () => props.mode],
  ([isOpen, taskVal, appVal, modeVal]) => {
    if (!isOpen) return

    enableTelemetry.value = modeVal !== 'edit_task' && !taskVal

    if (taskVal) {
      form.value = {
        schedule: taskVal.schedule || '',
        random_range: taskVal.random_range || 0,
        timeout: taskVal.timeout || 30,
        retry_count: taskVal.retry_count || 0,
        retry_interval: taskVal.retry_interval || 0,
        clean_config: taskVal.clean_config || '',
        unified_config: taskVal.unified_config || '',
        id: taskVal.id,
        updated_at: taskVal.updated_at
      }
      if (taskVal.unified_config) {
        try {
          const cfg = JSON.parse(taskVal.unified_config)
          if (cfg.app?.build_opts) {
            forceSetup.value = Boolean(cfg.app.build_opts.force_setup)
            skipSetup.value = Boolean(cfg.app.build_opts.skip_setup)
            skipSync.value = Boolean(cfg.app.build_opts.skip_sync)
          }
          if (cfg.app?.current_scenario) {
            selectedScenario.value = cfg.app.current_scenario
          }
        } catch { /* ignore */ }
      }
    } else {
      form.value = {
        schedule: '',
        random_range: 0,
        timeout: 30,
        retry_count: 0,
        retry_interval: 0,
        clean_config: '',
        unified_config: ''
      }
    }

    const taskIdToLoad = taskVal?.id || appVal?.id
    if (taskIdToLoad) {
      setTimeout(() => {
        notificationConfigRef.value?.loadConfig(taskIdToLoad)
      }, 50)
    }
  },
  { immediate: true }
)

function toggleSecretVisibility(key: string) {
  showSecrets.value[key] = !showSecrets.value[key]
}

const logSectionRef = ref<HTMLDivElement | null>(null)
const logContainerRef = ref<HTMLPreElement | null>(null)
const fullscreenLogContainerRef = ref<HTMLPreElement | null>(null)

function scrollToBottom() {
  import('vue').then(({ nextTick }) => {
    nextTick(() => {
      if (isFullscreenLog.value && fullscreenLogContainerRef.value) {
        fullscreenLogContainerRef.value.scrollTop = fullscreenLogContainerRef.value.scrollHeight
      } else if (logContainerRef.value) {
        logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
      }
    })
  })
}

function focusLogSection() {
  setTimeout(() => {
    logSectionRef.value?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }, 100)
}

const showOverwriteConfirm = ref(false)
const overwriteAppName = ref('')

async function startDeployProcess() {
  if (demoMode.value) {
    toast.error('演示模式下禁止安装或部署应用')
    return
  }
  if (!form.value.schedule || !form.value.schedule.trim()) {
    toast.error('请填写定时规则 (Cron表达式)')
    return
  }
  if (props.mode === 'edit_task') {
    const taskId = props.task?.id || props.targetApp?.id
    if (!taskId) return
    deploying.value = true
    try {
      // 解析现有 unified_config 并注入最新的 build_opts 及场景与环境变量
      let parsedUnified: any = {}
      if (form.value.unified_config) {
        try {
          parsedUnified = JSON.parse(form.value.unified_config)
        } catch { /* ignore */ }
      }
      if (!parsedUnified.app) {
        parsedUnified.app = {}
      }
      parsedUnified.app.build_opts = {
        force_setup: forceSetup.value,
        skip_setup: skipSetup.value,
        skip_sync: skipSync.value
      }
      if (selectedScenario.value) {
        parsedUnified.app.current_scenario = selectedScenario.value
      }
      const updatedUnifiedStr = JSON.stringify(parsedUnified)

      const updatePayload: any = {
        name: props.task?.name || props.targetApp?.name,
        remark: props.task?.remark || props.targetApp?.description,
        schedule: form.value.schedule || '',
        random_range: form.value.random_range || 0,
        timeout: form.value.timeout || 30,
        retry_count: form.value.retry_count || 0,
        retry_interval: form.value.retry_interval || 0,
        clean_config: form.value.clean_config || '',
        unified_config: updatedUnifiedStr
      }
      await api.tasks.update(taskId, updatePayload)
      if (notificationConfigRef.value) {
        await notificationConfigRef.value.saveConfig(taskId)
      }
      toast.success('更新应用调度配置成功！')
      emit('success')
      emit('update:open', false)
    } catch (err: any) {
      toast.error('更新应用配置失败: ' + (err.message || err))
    } finally {
      deploying.value = false
    }
    return
  }

  // 如果尚未进行过同名确认
  const appIdToCheck = props.targetApp?.id
  const appNameToCheck = props.targetApp?.name || props.task?.name
  if (appIdToCheck || appNameToCheck) {
    try {
      const installedApps = await api.apps.list()
      const existing = installedApps.find((a: any) => 
        (appIdToCheck && a.id === appIdToCheck) || 
        (appNameToCheck && a.name === appNameToCheck) ||
        (appIdToCheck && a.id === `app:${appIdToCheck}`)
      )
      if (existing) {
        overwriteAppName.value = existing.name || appIdToCheck || appNameToCheck || ''
        showOverwriteConfirm.value = true
        return
      }
    } catch { /* ignore */ }
  }
  executeDeploy()
}

function confirmOverwriteDeploy() {
  showOverwriteConfirm.value = false
  executeDeploy()
}

async function executeDeploy() {
  if (demoMode.value) {
    toast.error('演示模式下禁止安装或部署应用')
    return
  }
  deploying.value = true
  deployLog.value = '>> 正在建立实时日志连接...\n'
  deployError.value = ''
  deploySuccess.value = false

  focusLogSection()

  // 格式化环境变量为字符串键值对
  const formattedEnvs: Record<string, string> = {}
  for (const [k, v] of Object.entries(envForm.value)) {
    if (v !== undefined && v !== null && v !== '') {
      formattedEnvs[k] = String(v)
    }
  }

  const payload: ApplyAppPayload = {
    scenario_id: selectedScenario.value || undefined,
    env_values: Object.keys(formattedEnvs).length > 0 ? formattedEnvs : undefined,
    force_setup: forceSetup.value,
    skip_setup: skipSetup.value,
    skip_sync: skipSync.value,
    schedule: form.value.schedule || undefined,
    random_range: form.value.random_range || 0,
    timeout: form.value.timeout || 30,
    retry_count: form.value.retry_count || 0,
    retry_interval: form.value.retry_interval || 0,
    clean_config: form.value.clean_config || undefined,
    unified_config: form.value.unified_config || undefined,
    enable_telemetry: enableTelemetry.value
  }

  if (props.targetApp) {
    if (props.targetApp.manifest_raw) {
      payload.raw_yaml = props.targetApp.manifest_raw
      payload.path_or_url = manifestUrl.value
    } else {
      payload.path_or_url = manifestUrl.value
    }
  } else {
    if (activeTab.value === 'url') {
      if (!manifestUrl.value.trim()) {
        toast.error('请输入 YAML 文件路径或远程 URL')
        deploying.value = false
        deployLog.value = ''
        return
      }
      payload.path_or_url = manifestUrl.value.trim()
    } else {
      if (!manifestYaml.value.trim()) {
        toast.error('请粘贴 YAML 清单内容')
        deploying.value = false
        deployLog.value = ''
        return
      }
      payload.raw_yaml = manifestYaml.value.trim()
    }
  }

  try {
    const response = await fetch('/api/v1/apps/apply?stream=true', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream'
      },
      body: JSON.stringify(payload)
    })

    if (!response.ok) {
      const errText = await response.text()
      throw new Error(errText || `HTTP ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('浏览器不支持 ReadableStream')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      // 保留最后一个可能不完整的行
      buffer = lines.pop() || ''

      let currentEvent = 'message'
      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed) continue

        if (trimmed.startsWith('event:')) {
          currentEvent = trimmed.substring(6).trim()
        } else if (trimmed.startsWith('data:')) {
          const dataStr = line.substring(line.indexOf('data:') + 5).trim()
          if (currentEvent === 'done') {
            deploySuccess.value = true
            const taskIdToSave = props.task?.id || props.targetApp?.id
            if (taskIdToSave && notificationConfigRef.value) {
              await notificationConfigRef.value.saveConfig(taskIdToSave)
            }
            toast.success('应用部署成功！')
            emit('success')
          } else if (currentEvent === 'error') {
            try {
              const errObj = JSON.parse(dataStr)
              deployError.value = errObj.error || '部署错误'
            } catch {
              deployError.value = dataStr || '部署错误'
            }
            toast.error(deployError.value)
          } else {
            // 普通 log 输出
            deployLog.value += dataStr + '\n'
            scrollToBottom()
          }
        }
      }
    }

    if (!deploySuccess.value && !deployError.value) {
      deploySuccess.value = true
      const taskIdToSave = props.task?.id || props.targetApp?.id
      if (taskIdToSave && notificationConfigRef.value) {
        await notificationConfigRef.value.saveConfig(taskIdToSave)
      }
      toast.success('应用部署成功！')
      emit('success')
    }
  } catch (err: any) {
    deployError.value = err.message || '部署失败'
    deployLog.value += `\n[ERROR] ${deployError.value}\n`
    toast.error(deployError.value)
  } finally {
    deploying.value = false
    scrollToBottom()
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-[580px] max-h-[82vh] flex flex-col p-0 overflow-hidden bg-background border-border shadow-2xl rounded-2xl" @openAutoFocus.prevent @pointerDownOutside.prevent @interactOutside.prevent>
      <!-- 头部 -->
      <DialogHeader class="px-5 pt-5 pb-3.5 border-b border-border/60 bg-muted/20">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0 overflow-hidden shadow-xs">
            <img
              v-if="targetApp?.icon && !imageLoadError"
              :src="targetApp.icon"
              class="w-full h-full object-cover"
              :alt="targetApp.name"
              @error="imageLoadError = true"
            />
            <Package v-else class="w-5 h-5 text-primary" />
          </div>
          <div class="space-y-0.5 min-w-0">
            <DialogTitle class="text-base font-bold flex items-center gap-2 truncate">
              <span class="truncate">{{ targetApp ? (mode === 'edit_task' ? `编辑应用调度配置：${targetApp.name}` : (isInstalled ? `配置 / 重新部署：${targetApp.name}` : `部署应用：${targetApp.name}`)) : '导入应用 (Apply Manifest)' }}</span>
              <Badge v-if="targetApp?.version" variant="secondary" class="text-[10px] font-mono shrink-0">
                v{{ targetApp.version }}
              </Badge>
              <Badge v-if="targetApp?.last_commit" variant="outline" class="text-[9px] font-mono shrink-0 text-muted-foreground/80" :title="`仓库最近更新时间: ${targetApp.last_commit}`">
                更新于 {{ formatDate(targetApp.last_commit) }}
              </Badge>
            </DialogTitle>
            <DialogDescription class="text-xs text-muted-foreground truncate">
              {{ targetApp?.description || '通过白虎声明式规范一键部署代码源与任务配置' }}
            </DialogDescription>
          </div>
        </div>
      </DialogHeader>

      <!-- 演示模式提示 Banner -->
      <div v-if="demoMode" class="mx-5 mt-3 flex items-center gap-2 p-3 rounded-md bg-destructive/10 text-destructive text-xs font-medium">
        <AlertTriangle class="w-4 h-4 shrink-0" />
        <span>演示模式限制：当前系统开启演示模式，禁止安装、部署或修改应用配置。</span>
      </div>

      <!-- 中间可滚动表单区 -->
      <div class="flex-1 overflow-y-auto px-6 py-4 space-y-6">

        <!-- 如果是自定义导入模式（无 targetApp） -->
        <div v-if="!targetApp" class="space-y-4">
          <Tabs v-model="activeTab" class="w-full">
            <TabsList class="grid grid-cols-2 w-full">
              <TabsTrigger value="url">远程或本地路径 (URL / Path)</TabsTrigger>
              <TabsTrigger value="yaml">粘贴 YAML 清单 (Raw YAML)</TabsTrigger>
            </TabsList>
          </Tabs>

          <div v-if="activeTab === 'url'" class="space-y-2">
            <Label class="text-xs font-medium">Manifest 路径或 URL</Label>
            <Input
              v-model="manifestUrl"
              placeholder="https://raw.githubusercontent.com/.../app.yaml 或 ./apps/my-app/app.yaml"
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              spellcheck="false"
              data-lpignore="true"
              class="font-mono text-xs"
            />
            <p class="text-[11px] text-muted-foreground">支持输入 GitHub Raw 直链、CDN 地址或服务端本地绝对/相对路径。</p>
          </div>

          <div v-else class="space-y-2">
            <Label class="text-xs font-medium">YAML 内容</Label>
            <textarea
              v-model="manifestYaml"
              rows="8"
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              spellcheck="false"
              data-lpignore="true"
              class="w-full font-mono text-xs p-3 rounded-lg border border-border bg-muted/10 focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder="spec_version: v1&#10;id: my-app&#10;name: 我的应用..."
            ></textarea>
          </div>
        </div>

        <!-- 场景预设选择（如有） -->
        <div v-if="availableScenarios.length > 0" class="space-y-3">
          <div class="flex items-center justify-between">
            <Label class="text-xs font-semibold flex items-center gap-1.5">
              <Sparkles class="w-3.5 h-3.5 text-primary" />
              运行场景预设 (Scenario)
            </Label>
            <span class="text-[11px] text-muted-foreground">切换场景将自动分配对应的任务启停与 Cron 策略</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div
              v-for="sc in availableScenarios"
              :key="sc.id"
              class="relative flex flex-col p-3 rounded-lg border transition-all cursor-pointer text-left"
              :class="[
                selectedScenario === sc.id
                  ? 'border-primary bg-primary/5 ring-1 ring-primary'
                  : 'border-border hover:border-border/80 hover:bg-muted/30'
              ]"
              @click="selectedScenario = sc.id"
            >
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs font-bold text-foreground">{{ sc.name }}</span>
                <Badge v-if="sc.default" variant="outline" class="text-[10px] px-1.5 py-0 text-primary border-primary/30">
                  推荐
                </Badge>
              </div>
              <p class="text-[11px] text-muted-foreground line-clamp-2">{{ sc.description || '无场景描述' }}</p>
            </div>
          </div>
        </div>

        <!-- 动态环境变量表单 (Env Schema) -->
        <div v-if="envSchemaList.length > 0" class="space-y-4">
          <div class="border-t border-border/50 pt-4 flex items-center justify-between">
            <Label class="text-xs font-semibold">环境变量与凭证契约 (Env Schema)</Label>
            <span class="text-[11px] text-muted-foreground">系统将自动为本应用变量绑定专属 Tag</span>
          </div>

          <div class="space-y-3.5">
            <div
              v-for="schema in envSchemaList"
              :key="schema.key"
              class="space-y-1.5 bg-muted/20 p-3 rounded-lg border border-border/40"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="flex flex-wrap items-center gap-1.5 min-w-0">
                  <Label class="text-xs font-medium shrink-0">
                    {{ schema.label }}
                    <span v-if="schema.required" class="text-destructive text-xs">*</span>
                  </Label>
                  <span class="text-[10px] font-mono text-muted-foreground break-all">({{ schema.key }})</span>
                </div>
                <Badge v-if="schema.tag" variant="secondary" class="text-[10px] font-mono shrink-0">
                  Tag: {{ schema.tag }}
                </Badge>
              </div>

              <!-- 密码/密钥类型 (Secret) -->
              <div v-if="schema.type === 'secret'" class="relative">
                <Input
                  v-model="envForm[schema.key]"
                  type="text"
                  :placeholder="schema.placeholder || '请输入机密凭证'"
                  autocomplete="off"
                  autocorrect="off"
                  autocapitalize="off"
                  spellcheck="false"
                  data-lpignore="true"
                  data-1p-ignore="true"
                  :style="{ webkitTextSecurity: showSecrets[schema.key] ? 'none' : 'disc' }"
                  class="text-xs pr-10 font-mono"
                />
                <button
                  type="button"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  @click="toggleSecretVisibility(schema.key)"
                >
                  <EyeOff v-if="showSecrets[schema.key]" class="w-4 h-4" />
                  <Eye v-else class="w-4 h-4" />
                </button>
              </div>

              <!-- 下拉选择类型 (Select) -->
              <div v-else-if="schema.type === 'select'">
                <Select v-model="envForm[schema.key]">
                  <SelectTrigger class="text-xs h-9">
                    <SelectValue :placeholder="schema.placeholder || '请选择'" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="opt in schema.options || []"
                      :key="String(opt.value)"
                      :value="String(opt.value)"
                    >
                      {{ opt.label }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <!-- 开关类型 (Boolean) -->
              <div v-else-if="schema.type === 'boolean'" class="flex items-center gap-2 pt-1">
                <Switch
                  :checked="Boolean(envForm[schema.key])"
                  @update:checked="envForm[schema.key] = $event"
                />
                <span class="text-xs text-muted-foreground">
                  {{ envForm[schema.key] ? '已开启' : '已关闭' }}
                </span>
              </div>

              <!-- 普通字符串/数字输入 -->
              <div v-else>
                <Input
                  v-model="envForm[schema.key]"
                  :type="schema.type === 'number' ? 'number' : 'text'"
                  :placeholder="schema.placeholder || '请输入配置值'"
                  autocomplete="off"
                  autocorrect="off"
                  autocapitalize="off"
                  spellcheck="false"
                  data-lpignore="true"
                  class="text-xs font-mono"
                />
              </div>

              <p v-if="schema.description" class="text-[11px] text-muted-foreground">
                {{ schema.description }}
              </p>
            </div>
          </div>
        </div>

        <!-- 高级部署控制选项 -->
        <div class="border-t border-border/50 pt-3 space-y-2.5">
          <div class="flex items-center justify-between">
            <button
              type="button"
              class="text-xs font-medium text-muted-foreground hover:text-foreground flex items-center gap-1.5 transition-colors"
              @click="showAdvanced = !showAdvanced"
            >
              <ChevronDown v-if="showAdvanced" class="w-3.5 h-3.5" />
              <ChevronRight v-else class="w-3.5 h-3.5" />
              <span>高级构建与部署控制</span>
            </button>
            <span v-if="showAdvanced" class="text-[11px] text-muted-foreground">可自由定制构建流程与加速策略</span>
          </div>

          <div v-if="showAdvanced" class="space-y-2">
            <!-- 1. 强制重新编译 -->
            <div
              class="flex items-center justify-between p-2.5 rounded-lg border transition-all cursor-pointer select-none"
              :class="[
                forceSetup
                  ? 'border-primary/60 bg-primary/5 ring-1 ring-primary/40'
                  : 'border-border/60 hover:border-border hover:bg-muted/20'
              ]"
              @click="forceSetup = !forceSetup; if (forceSetup) skipSetup = false"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <Checkbox
                  v-model="forceSetup"
                  class="shrink-0"
                  @click.stop
                  @update:model-value="(val) => { if (val) skipSetup = false }"
                />
                <div class="flex flex-col min-w-0">
                  <span class="text-xs font-bold text-foreground">强制重新编译 (Rebuild)</span>
                  <span class="text-[11px] text-muted-foreground truncate">跳过探活断言，强行重新执行 setup.install 依赖安装与预编译</span>
                </div>
              </div>
              <Badge variant="secondary" class="text-[10px] shrink-0 font-normal ml-2">
                适用：依赖更新 / 修复编译
              </Badge>
            </div>

            <!-- 2. 跳过环境与依赖安装 -->
            <div
              class="flex items-center justify-between p-2.5 rounded-lg border transition-all cursor-pointer select-none"
              :class="[
                skipSetup
                  ? 'border-primary/60 bg-primary/5 ring-1 ring-primary/40'
                  : 'border-border/60 hover:border-border hover:bg-muted/20'
              ]"
              @click="skipSetup = !skipSetup; if (skipSetup) forceSetup = false"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <Checkbox
                  v-model="skipSetup"
                  class="shrink-0"
                  @click.stop
                  @update:model-value="(val) => { if (val) forceSetup = false }"
                />
                <div class="flex flex-col min-w-0">
                  <span class="text-xs font-bold text-foreground">跳过环境与依赖安装</span>
                  <span class="text-[11px] text-muted-foreground truncate">完全跳过 setup 阶段，仅同步环境变量与任务映射</span>
                </div>
              </div>
              <Badge variant="secondary" class="text-[10px] shrink-0 font-normal ml-2">
                适用：环境就绪 / 极速改配置
              </Badge>
            </div>

            <!-- 3. 跳过代码源同步 -->
            <div
              class="flex items-center justify-between p-2.5 rounded-lg border transition-all cursor-pointer select-none"
              :class="[
                skipSync
                  ? 'border-primary/60 bg-primary/5 ring-1 ring-primary/40'
                  : 'border-border/60 hover:border-border hover:bg-muted/20'
              ]"
              @click="skipSync = !skipSync"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <Checkbox
                  v-model="skipSync"
                  class="shrink-0"
                  @click.stop
                />
                <div class="flex flex-col min-w-0">
                  <span class="text-xs font-bold text-foreground">跳过代码源同步</span>
                  <span class="text-[11px] text-muted-foreground truncate">使用本地已有代码，不重新拉取 Git / URL 源码</span>
                </div>
              </div>
              <Badge variant="secondary" class="text-[10px] shrink-0 font-normal ml-2">
                适用：二次开发 / 避免覆盖
              </Badge>
            </div>
          </div>
        </div>
        <!-- 调度策略 Section -->
        <section class="border-t border-border/50 pt-4 space-y-4">
          <div class="flex items-center gap-2 mb-1">
            <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
            <h3 class="text-xs font-bold text-foreground/90">调度策略</h3>
          </div>

          <div class="grid gap-5 pl-3 border-l border-muted">
            <TaskCronConfig v-model="form.schedule" />
            <TaskAdvancedConfig v-model="form" />
          </div>
        </section>

        <!-- 通知配置 -->
        <TaskNotificationConfig ref="notificationConfigRef" :task-id="props.task?.id || props.targetApp?.id" />

        <!-- 部署日志与输出区 (Terminal Viewer) -->
        <div ref="logSectionRef" v-if="mode !== 'edit_task' && (deploying || deployLog)" class="space-y-2 border-t border-border/50 pt-4">
          <div class="flex items-center justify-between text-xs font-mono">
            <span class="flex items-center gap-1.5 font-bold text-foreground">
              <Terminal class="w-3.5 h-3.5" />
              执行输出日志 (Deployment Log)
            </span>

            <div class="flex items-center gap-3">
              <span v-if="deploying" class="text-primary flex items-center gap-1">
                <Loader2 class="w-3 h-3 animate-spin" />
                正在执行依赖安装与任务编排...
              </span>
              <span v-else-if="deploySuccess" class="text-emerald-500 font-semibold flex items-center gap-1">
                <CheckCircle2 class="w-3.5 h-3.5" />
                部署就绪
              </span>
              <span v-else-if="deployError" class="text-destructive font-semibold flex items-center gap-1">
                <AlertCircle class="w-3.5 h-3.5" />
                执行失败
              </span>

              <button
                type="button"
                class="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1 text-[11px]"
                :title="isFullscreenLog ? '退出全屏日志' : '全屏展开日志'"
                @click="isFullscreenLog = !isFullscreenLog"
              >
                <Minimize2 v-if="isFullscreenLog" class="w-3.5 h-3.5" />
                <Maximize2 v-else class="w-3.5 h-3.5" />
                <span>{{ isFullscreenLog ? '还原' : '全屏' }}</span>
              </button>
            </div>
          </div>

          <div
            v-if="isFullscreenLog"
            class="fixed inset-4 z-50 flex flex-col bg-neutral-950/98 border border-neutral-700 rounded-xl shadow-2xl backdrop-blur-md overflow-hidden"
          >
            <!-- 全屏日志顶栏 -->
            <div class="flex items-center justify-between px-4 py-2.5 bg-neutral-900/90 border-b border-neutral-800 text-xs font-mono text-neutral-300 select-none">
              <span class="flex items-center gap-2 font-bold">
                <Terminal class="w-4 h-4 text-primary" />
                执行输出日志 (Deployment Log - 全屏)
              </span>

              <div class="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 px-2 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800"
                  @click="isFullscreenLog = false"
                >
                  <Minimize2 class="w-3.5 h-3.5 mr-1" />
                  退出全屏 (Esc)
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded-lg"
                  @click="isFullscreenLog = false"
                >
                  <X class="w-4 h-4" />
                </Button>
              </div>
            </div>

            <pre
              ref="fullscreenLogContainerRef"
              class="flex-1 text-neutral-200 font-mono text-[12px] leading-relaxed p-4 overflow-y-auto overflow-x-auto select-text bg-transparent whitespace-pre-wrap break-all"
            >{{ deployLog || '正在初始化进程...' }}</pre>
          </div>

          <pre
            v-else
            ref="logContainerRef"
            class="bg-neutral-950 text-neutral-200 font-mono text-[11px] leading-relaxed p-3.5 rounded-lg overflow-y-auto overflow-x-auto max-h-56 select-text border border-neutral-800 transition-all duration-200 whitespace-pre-wrap break-all"
          >{{ deployLog || '正在初始化进程...' }}</pre>
        </div>
      </div>

      <!-- 底部操作按钮 -->
      <div class="px-4 sm:px-6 py-3.5 border-t border-border/60 bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="text-[11px] text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1 min-w-0">
          <span class="text-emerald-500 font-medium shrink-0">100% 免费开源</span>
          <template v-if="mode !== 'edit_task' && !task">
            <span class="text-muted-foreground/40 shrink-0">•</span>
            <label class="inline-flex items-center gap-1.5 cursor-pointer shrink-0 select-none text-muted-foreground hover:text-foreground transition-colors group" title="部署时仅向统计服务发送匿名计数(+1)，绝不收集任何个人/服务器敏感数据">
              <div
                class="w-3.5 h-3.5 rounded border flex items-center justify-center transition-all duration-150 shrink-0"
                :class="enableTelemetry ? 'bg-primary border-primary text-primary-foreground shadow-xs' : 'border-input bg-background group-hover:border-muted-foreground/60'"
                @click="enableTelemetry = !enableTelemetry"
              >
                <Check v-if="enableTelemetry" class="w-2.5 h-2.5 stroke-[3]" />
              </div>
              <span>参与匿名热度统计</span>
            </label>
          </template>
          <template v-if="targetApp?.homepage">
            <span class="text-muted-foreground/40 shrink-0">•</span>
            <a
              :href="targetApp.homepage"
              target="_blank"
              class="hover:text-primary inline-flex items-center gap-0.5 underline underline-offset-2 shrink-0 ml-0.5"
            >
              项目主页 <ExternalLink class="w-3 h-3" />
            </a>
          </template>
        </div>

        <div class="flex items-center justify-end gap-2.5 shrink-0 ml-auto sm:ml-0">
          <Button variant="outline" size="sm" :disabled="deploying" @click="$emit('update:open', false)">
            {{ deploySuccess ? '完成' : '取消' }}
          </Button>

          <Button size="sm" :disabled="deploying || demoMode" @click="startDeployProcess">
            <Loader2 v-if="deploying" class="w-3.5 h-3.5 mr-1.5 animate-spin" />
            <Sparkles v-else class="w-3.5 h-3.5 mr-1.5" />
            {{ deploying ? (mode === 'edit_task' ? '正在保存...' : '正在部署中...') : (mode === 'edit_task' ? '保存配置' : (deploySuccess ? '重新部署' : (isInstalled ? '重新部署应用' : '开始部署应用'))) }}
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>

  <!-- 覆盖已有应用确认弹窗 -->
  <BaihuDialog v-model:open="showOverwriteConfirm" title="应用已存在确认" size="sm" icon="AlertTriangle">
    <div class="space-y-3 pt-1">
      <p class="text-xs text-muted-foreground leading-relaxed">
        检测到系统中已安装同名应用 <b class="text-foreground font-semibold">{{ overwriteAppName }}</b>。
      </p>
      <div class="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-start gap-2 text-xs text-amber-600 dark:text-amber-400">
        <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
        <div class="leading-relaxed">
          继续部署将更新并覆盖现有应用配置与依赖，确定要覆盖吗？
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="outline" size="sm" @click="showOverwriteConfirm = false">取消</Button>
      <Button size="sm" class="bg-amber-600 hover:bg-amber-700 text-white font-medium" @click="confirmOverwriteDeploy">
        确认覆盖并部署
      </Button>
    </template>
  </BaihuDialog>
</template>
