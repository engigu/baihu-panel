<script setup lang="ts">
import { ref, watch } from 'vue'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  Package,
  Sparkles,
  ListTodo,
  Wrench,
  Play,
  Trash2,
  RefreshCw,
  Loader2,
  Clock,
  Terminal,
  FileCode,
  AlertTriangle
} from 'lucide-vue-next'
import { api, type AppDetailResponse, type Task } from '@/api'
import { getCronDescription } from '@/utils/cron'
import { toast } from 'vue-sonner'

const props = defineProps<{
  open: boolean
  appId: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'updated': []
}>()

const loading = ref(false)
const appData = ref<AppDetailResponse | null>(null)
const activeTab = ref('tasks')

// 场景切换状态
const switchingScenario = ref(false)
const selectedScenarioToSwitch = ref('')

// 重新构建状态
const rebuilding = ref(false)
const rebuildLog = ref('')

// 任务执行状态
const runningTaskId = ref<string | null>(null)

// 卸载状态
const showUninstallConfirm = ref(false)
const cleanDataOnUninstall = ref(true)
const uninstalling = ref(false)

// 环境变量表单状态
const envForm = ref<Record<string, string>>({})

watch(
  () => [props.open, props.appId],
  async ([open, id]) => {
    if (open && id) {
      await loadAppDetail(id as string)
    } else {
      appData.value = null
      rebuildLog.value = ''
    }
  },
  { immediate: true }
)

async function loadAppDetail(id: string) {
  loading.value = true
  try {
    const res = await api.apps.get(id)
    appData.value = res
    selectedScenarioToSwitch.value = res.app.current_scenario || ''

    // 初始化已有的环境变量
    if (res.manifest?.env_schema) {
      for (const item of res.manifest.env_schema) {
        envForm.value[item.key] = ''
      }
    }
  } catch (err: any) {
    toast.error('加载应用详情失败: ' + (err.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 切换场景
async function handleSwitchScenario(scId: any) {
  if (!appData.value || switchingScenario.value || !scId) return
  const scenarioStr = String(scId)
  if (scenarioStr === appData.value.app.current_scenario) return

  switchingScenario.value = true
  try {
    await api.apps.switchScenario(appData.value.app.id, scenarioStr)
    toast.success(`场景成功切换为: ${scenarioStr}`)
    await loadAppDetail(props.appId)
    emit('updated')
  } catch (err: any) {
    toast.error('切换场景失败: ' + (err.message || '未知错误'))
  } finally {
    switchingScenario.value = false
  }
}

// 触发单个受控任务执行
async function handleRunTask(task: Task) {
  runningTaskId.value = task.id
  try {
    await api.tasks.execute(task.id)
    toast.success(`任务 [${task.name}] 已触发执行！`)
  } catch (err: any) {
    toast.error('触发失败: ' + (err.message || '未知错误'))
  } finally {
    runningTaskId.value = null
  }
}

// 启停单个受控任务
async function handleToggleTask(task: Task, enabled: boolean) {
  try {
    await api.tasks.update(task.id, { enabled })
    task.enabled = enabled
    toast.success(`任务 [${task.name}] 已${enabled ? '启用' : '禁用'}`)
  } catch (err: any) {
    toast.error('更新失败: ' + (err.message || '未知错误'))
  }
}

// 触发重新预编译自愈
async function handleRebuild() {
  if (!appData.value || rebuilding.value) return
  rebuilding.value = true
  rebuildLog.value = ''
  try {
    const res = await api.apps.rebuild(appData.value.app.id)
    rebuildLog.value = res.log || '重建完成'
    toast.success('依赖预编译与环境修复成功！')
    await loadAppDetail(props.appId)
    emit('updated')
  } catch (err: any) {
    rebuildLog.value = err.data?.log || err.message || '重建失败'
    toast.error('重建失败: ' + (err.message || '未知错误'))
  } finally {
    rebuilding.value = false
  }
}

// 卸载应用
async function handleUninstall() {
  if (!appData.value || uninstalling.value) return
  uninstalling.value = true
  try {
    await api.apps.remove(appData.value.app.id, true)
    toast.success('应用已成功卸载！')
    showUninstallConfirm.value = false
    emit('update:open', false)
    emit('updated')
  } catch (err: any) {
    toast.error('卸载失败: ' + (err.message || '未知错误'))
  } finally {
    uninstalling.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="!w-[640px] !max-w-[640px] max-h-[85vh] p-0 gap-0 overflow-hidden bg-background border border-border/80 shadow-2xl rounded-2xl flex flex-col focus:outline-none" @openAutoFocus.prevent @pointerDownOutside.prevent @interactOutside.prevent>
      <!-- 头部 -->
      <div class="px-5 py-4 border-b border-border/60 bg-muted/20 flex items-center justify-between shrink-0">
        <div v-if="appData" class="flex items-center gap-3 min-w-0 pr-6">
          <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0 overflow-hidden shadow-xs">
            <img v-if="appData.app.icon" :src="appData.app.icon" class="w-full h-full object-cover" :alt="appData.app.name" />
            <Package v-else class="w-5 h-5 text-primary" />
          </div>

          <div class="space-y-0.5 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <DialogTitle class="text-base font-bold text-foreground truncate">
                {{ appData.app.name }}
              </DialogTitle>
              <Badge variant="secondary" class="font-mono text-[10px] px-1.5 py-0 shrink-0">
                v{{ appData.app.version }}
              </Badge>
              <Badge variant="outline" class="text-[10px] text-emerald-500 border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0 font-medium shrink-0">
                ● 已就绪
              </Badge>
            </div>

            <div class="flex items-center gap-2 text-xs text-muted-foreground font-mono truncate">
              <span v-if="appData.app.author">作者: {{ appData.app.author }}</span>
              <span v-if="appData.app.category">• {{ appData.app.category }}</span>
              <span>• {{ appData.tasks?.length || 0 }} 个任务</span>
            </div>
          </div>
        </div>

        <div v-else class="h-10 flex items-center justify-center">
          <Loader2 class="w-4 h-4 animate-spin text-muted-foreground" />
        </div>
      </div>

      <!-- 中间内容区 -->
      <div v-if="appData" class="flex-1 overflow-y-auto px-5 py-4 space-y-4">
        <!-- 顶部场景总控卡片 -->
        <div class="p-3.5 rounded-xl border border-primary/20 bg-primary/5 flex flex-col gap-2.5">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary shrink-0">
                <Sparkles class="w-3.5 h-3.5" />
              </div>
              <div class="text-xs font-bold text-foreground">
                当前运行场景
              </div>
            </div>

            <!-- 场景切换器 (若有多个场景) -->
            <div v-if="appData.manifest?.scenarios && appData.manifest.scenarios.length > 1" class="w-48 shrink-0">
              <Select
                :model-value="appData.app.current_scenario"
                :disabled="switchingScenario"
                @update:model-value="handleSwitchScenario($event)"
              >
                <SelectTrigger class="h-8 text-xs font-semibold bg-background border-primary/30 text-primary shadow-xs">
                  <SelectValue placeholder="切换场景..." />
                </SelectTrigger>
                <SelectContent align="end" class="w-64 text-xs">
                  <SelectItem
                    v-for="sc in appData.manifest.scenarios"
                    :key="sc.id"
                    :value="sc.id"
                    class="text-xs py-2 cursor-pointer"
                  >
                    <div class="flex flex-col min-w-0 gap-0.5">
                      <span class="font-bold text-foreground truncate">{{ sc.name }}</span>
                      <span v-if="sc.description" class="text-[11px] text-muted-foreground truncate leading-tight">{{ sc.description }}</span>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <!-- 场景说明 -->
          <div class="text-xs bg-background/60 p-2.5 rounded-lg border border-primary/10 flex flex-col gap-0.5">
            <span class="font-bold text-primary text-xs truncate">
              {{ appData.manifest?.scenarios?.find(s => s.id === appData?.app.current_scenario)?.name || appData.app.current_scenario || '默认场景' }}
            </span>
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              {{ appData.manifest?.scenarios?.find(s => s.id === appData?.app.current_scenario)?.description || '激活中的自动化组合策略' }}
            </p>
          </div>
        </div>

        <!-- 标签页导航 -->
        <Tabs v-model="activeTab" class="w-full">
          <TabsList class="grid grid-cols-3 w-full h-8 p-0.5 bg-muted/20 border border-border/40 rounded-lg">
            <TabsTrigger value="tasks" class="text-xs h-7 flex items-center gap-1">
              <ListTodo class="w-3.5 h-3.5 opacity-70" />
              <span>受控任务 ({{ appData.tasks?.length || 0 }})</span>
            </TabsTrigger>
            <TabsTrigger value="manifest" class="text-xs h-7 flex items-center gap-1">
              <FileCode class="w-3.5 h-3.5 opacity-70" />
              <span>应用规范</span>
            </TabsTrigger>
            <TabsTrigger value="maintenance" class="text-xs h-7 flex items-center gap-1">
              <Wrench class="w-3.5 h-3.5 opacity-70" />
              <span>自愈与卸载</span>
            </TabsTrigger>
          </TabsList>

          <!-- Tab 1: 受控任务清单 -->
          <TabsContent value="tasks" class="space-y-2.5 mt-3">
            <div class="text-[11px] text-muted-foreground px-0.5">
              任务启停与 Cron 表达式由应用场景统一控制与编排
            </div>

            <div class="space-y-2">
              <div
                v-for="task in appData.tasks"
                :key="task.id"
                class="flex items-center justify-between p-2.5 rounded-lg border border-border/50 bg-card hover:bg-muted/20 transition-colors gap-3"
              >
                <div class="space-y-1 min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-bold text-foreground truncate">{{ task.name }}</span>
                    <Badge v-if="task.schedule" variant="secondary" class="text-[10px] font-mono px-1.5 py-0 flex items-center gap-1 shrink-0">
                      <Clock class="w-2.5 h-2.5" />
                      {{ getCronDescription(task.schedule) }}
                    </Badge>
                  </div>
                  <code class="text-[11px] font-mono text-muted-foreground block truncate max-w-[280px] bg-muted/40 px-1.5 py-0.5 rounded">
                    {{ task.command }}
                  </code>
                </div>

                <div class="flex items-center gap-2 shrink-0">
                  <Switch
                    :checked="Boolean(task.enabled)"
                    @update:checked="handleToggleTask(task, $event)"
                  />
                  <Button
                    size="sm"
                    variant="ghost"
                    class="h-7 px-2 text-xs text-primary hover:bg-primary/10"
                    title="立即执行任务"
                    :disabled="runningTaskId === task.id"
                    @click="handleRunTask(task)"
                  >
                    <Loader2 v-if="runningTaskId === task.id" class="w-3.5 h-3.5 animate-spin" />
                    <Play v-else class="w-3.5 h-3.5 mr-1" />
                    <span>执行</span>
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- Tab 2: 应用规范原始清单 -->
          <TabsContent value="manifest" class="space-y-3 mt-3">
            <div class="flex items-center justify-between text-xs text-muted-foreground">
              <span>声明式应用原始配置 (YAML Manifest)</span>
              <span class="font-mono text-[11px] truncate max-w-[260px]">{{ appData.app.manifest_path || '内存驻留' }}</span>
            </div>
            <pre class="bg-neutral-950 text-neutral-200 font-mono text-xs p-3.5 rounded-xl overflow-x-auto max-h-96 select-text border border-neutral-800 leading-relaxed">{{ appData.app.manifest_raw || '无原始 YAML' }}</pre>
          </TabsContent>

          <!-- Tab 3: 维护与自愈工具箱 -->
          <TabsContent value="maintenance" class="space-y-3.5 mt-3">
            <!-- 重新预编译自愈 -->
            <div class="p-3.5 rounded-xl border border-border/80 bg-muted/20 space-y-2">
              <div class="flex items-center justify-between">
                <h4 class="text-xs font-bold text-foreground flex items-center gap-1.5">
                  <RefreshCw class="w-3.5 h-3.5 text-primary" />
                  重新编译运行环境 (Rebuild / TryFix)
                </h4>
                <Button
                  size="sm"
                  variant="outline"
                  class="shrink-0 text-xs h-7 px-2.5"
                  :disabled="rebuilding"
                  @click="handleRebuild"
                >
                  <Loader2 v-if="rebuilding" class="w-3 h-3 mr-1 animate-spin" />
                  <RefreshCw v-else class="w-3 h-3 mr-1" />
                  {{ rebuilding ? '编译中...' : '开始编译' }}
                </Button>
              </div>
              <p class="text-[11px] text-muted-foreground leading-relaxed">
                全量清除编译缓存并重新建立执行依赖产物。
              </p>
            </div>

            <!-- 重建执行输出日志 -->
            <div v-if="rebuilding || rebuildLog" class="space-y-1.5">
              <div class="flex items-center gap-1.5 text-xs font-mono font-bold">
                <Terminal class="w-3.5 h-3.5" />
                自愈日志 (Rebuild Output)
              </div>
              <pre class="bg-neutral-950 text-neutral-200 font-mono text-[11px] p-3 rounded-lg overflow-x-auto max-h-40 border border-neutral-800">{{ rebuildLog || '正在执行重建...' }}</pre>
            </div>

            <!-- 危险区域: 卸载应用 -->
            <div class="p-3.5 rounded-xl border border-destructive/30 bg-destructive/5 space-y-2.5">
              <div>
                <h4 class="text-xs font-bold text-destructive flex items-center gap-1.5">
                  <AlertTriangle class="w-3.5 h-3.5" />
                  卸载此应用
                </h4>
                <p class="text-[11px] text-muted-foreground mt-0.5">
                  移除应用及其受控任务，可选是否清理本地目录数据。
                </p>
              </div>

              <div v-if="!showUninstallConfirm">
                <Button
                  size="sm"
                  variant="destructive"
                  class="text-xs h-7 px-3 font-semibold"
                  @click="showUninstallConfirm = true"
                >
                  <Trash2 class="w-3.5 h-3.5 mr-1" />
                  卸载应用...
                </Button>
              </div>

              <div v-else class="space-y-3 p-3 rounded-lg border border-destructive/20 bg-destructive/5">
                <div class="flex items-start gap-2 text-xs text-destructive">
                  <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
                  <div class="leading-relaxed font-medium">
                    卸载应用将彻底删除<b>本地代码文件夹</b>以及<b>所有关联的受控任务与配置</b>，该操作无法撤销。
                  </div>
                </div>
                <div class="flex items-center gap-2 justify-end pt-1">
                  <Button size="sm" variant="ghost" class="text-xs h-7" @click="showUninstallConfirm = false">
                    取消
                  </Button>
                  <Button
                    size="sm"
                    variant="destructive"
                    class="text-xs h-7 font-medium"
                    :disabled="uninstalling"
                    @click="handleUninstall"
                  >
                    <Loader2 v-if="uninstalling" class="w-3 h-3 mr-1 animate-spin" />
                    确认彻底卸载
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>

      <!-- 底部关闭按钮 -->
      <div class="px-6 py-3 border-t border-border/60 bg-muted/20 flex items-center justify-end">
        <Button size="sm" variant="outline" @click="$emit('update:open', false)">
          关闭详情
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
