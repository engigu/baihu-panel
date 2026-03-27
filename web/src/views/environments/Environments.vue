<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import Pagination from '@/components/Pagination.vue'
import { Plus, Pencil, Trash2, Eye, EyeOff, Search, AlertTriangle, Terminal, Zap, ZapOff, Shield } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import { api, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ENV_TYPE } from '@/constants'

const { pageSize } = useSiteSettings()

const envVars = ref<EnvVar[]>([])
const showDialog = ref(false)
const editingEnv = ref<Partial<EnvVar>>({})
const isEdit = ref(false)
const showValues = ref<Record<string, boolean>>({})
const showDeleteDialog = ref(false)
const deleteEnvId = ref<string | null>(null)
const associatedTasks = ref<any[]>([])
const isDeleting = ref(false)
const valueTextareaRef = ref<HTMLTextAreaElement | null>(null)
const lineNumbersRef = ref<HTMLDivElement | null>(null)
const lineMeasureRef = ref<HTMLDivElement | null>(null)
const visualLineNumbers = ref<string[]>(['1'])
let textareaResizeObserver: ResizeObserver | null = null

const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
const activeTab = ref<string>(ENV_TYPE.NORMAL)
const isSecretSet = ref(true)
let searchTimer: ReturnType<typeof setTimeout> | null = null

async function checkSecretStatus() {
  try {
    isSecretSet.value = await api.env.secretStatus()
    if (!isSecretSet.value) {
      toast.warning('未检测到加密机密秘钥，请在启动时配置 BAIHU_SECRET_KEY 环境变量')
    }
  } catch (error) {
    console.error('检查秘钥状态失败', error)
  }
}

async function loadEnvVars() {
  try {
    const res = await api.env.list({ page: currentPage.value, page_size: pageSize.value, name: filterName.value || undefined, type: activeTab.value })
    envVars.value = res.data
    total.value = res.total
    // 初始化显示状态，根据数据库的 hidden 状态同步显示
    res.data.forEach(env => {
      showValues.value[env.id] = !env.hidden
    })
  } catch { toast.error('加载环境变量失败') }
}

watch(showDialog, (val) => {
  if (!val) loadEnvVars()
})

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadEnvVars()
  }, 300)
}

watch(activeTab, (val) => {
  currentPage.value = 1
  if (val === ENV_TYPE.SECRET) {
    checkSecretStatus()
  }
  loadEnvVars()
})

function handlePageChange(page: number) {
  currentPage.value = page
  loadEnvVars()
}

function openCreate() {
  editingEnv.value = { name: '', value: '', remark: '', type: activeTab.value, hidden: true, enabled: true }
  isEdit.value = false
  showDialog.value = true
  void updateVisualLineNumbers()
}

function openEdit(env: EnvVar) {
  editingEnv.value = { ...env, value: env.type === 'secret' ? '' : env.value }
  isEdit.value = true
  showDialog.value = true
  void updateVisualLineNumbers()
}

function syncValueLineNumbers() {
  if (!valueTextareaRef.value || !lineNumbersRef.value) return
  lineNumbersRef.value.scrollTop = valueTextareaRef.value.scrollTop
}

async function updateVisualLineNumbers() {
  await nextTick()

  const textarea = valueTextareaRef.value
  const measure = lineMeasureRef.value
  if (!textarea || !measure) return

  const style = window.getComputedStyle(textarea)
  const lineHeight = Number.parseFloat(style.lineHeight) || Number.parseFloat(style.fontSize) * 1.5 || 24
  const lines = String(editingEnv.value?.value ?? '').split('\n')

  measure.style.width = `${textarea.clientWidth}px`
  measure.innerHTML = ''

  const nextLineNumbers: string[] = []
  lines.forEach((line, index) => {
    const lineEl = document.createElement('div')
    lineEl.className = 'break-all whitespace-pre-wrap'
    lineEl.textContent = line || ' '
    measure.appendChild(lineEl)

    const visualRows = Math.max(1, Math.round(lineEl.getBoundingClientRect().height / lineHeight))
    nextLineNumbers.push(String(index + 1))
    for (let i = 1; i < visualRows; i += 1) {
      nextLineNumbers.push('\u00A0')
    }
  })

  visualLineNumbers.value = nextLineNumbers.length > 0 ? nextLineNumbers : ['1']
  syncValueLineNumbers()
}

async function saveEnv() {
  try {
    if (isEdit.value && editingEnv.value.id) {
      await api.env.update(editingEnv.value.id, editingEnv.value)
      toast.success(editingEnv.value.type === ENV_TYPE.SECRET ? '机密已更新' : '变量已更新')
    } else {
      await api.env.create(editingEnv.value)
      toast.success(editingEnv.value.type === ENV_TYPE.SECRET ? '机密已创建' : '变量已创建')
    }
    showDialog.value = false
    loadEnvVars()
  } catch { toast.error('保存失败') }
}

async function confirmDelete(id: string) {
  deleteEnvId.value = id
  try {
    const res = await api.env.tasks(id)
    associatedTasks.value = res || []
    showDeleteDialog.value = true
  } catch {
    toast.error('检查机密引用失败')
  }
}

async function deleteEnv(force = false) {
  if (!deleteEnvId.value) return
  isDeleting.value = true
  try {
    const res = await api.env.delete(deleteEnvId.value, force)
    if (res.code === 409) {
      associatedTasks.value = res.data || []
      isDeleting.value = false
      return
    }
    if (res.code !== 200) {
      toast.error(res.msg || '删除失败')
      isDeleting.value = false
      return
    }
    toast.success(activeTab.value === ENV_TYPE.SECRET ? '机密已删除' : '变量已删除')
    loadEnvVars()
    showDeleteDialog.value = false
  } catch {
    toast.error('网络错误，删除失败')
  } finally {
    isDeleting.value = false
  }
}

watch(showDeleteDialog, (val) => {
  if (!val) {
    associatedTasks.value = []
    deleteEnvId.value = null
  }
})

watch(() => editingEnv.value.value, () => {
  void updateVisualLineNumbers()
})

watch(showDialog, async (val) => {
  if (val) {
    await updateVisualLineNumbers()
    if (valueTextareaRef.value) {
      textareaResizeObserver?.disconnect()
      textareaResizeObserver = new ResizeObserver(() => {
        void updateVisualLineNumbers()
      })
      textareaResizeObserver.observe(valueTextareaRef.value)
    }
  } else {
    textareaResizeObserver?.disconnect()
  }
})

function toggleShow(id: string) {
  showValues.value[id] = !showValues.value[id]
}

async function toggleEnabled(env: EnvVar) {
  try {
    await api.env.update(env.id, { ...env, enabled: !env.enabled })
    env.enabled = !env.enabled
    toast.success(env.enabled ? '变量已启用' : '变量已禁用')
  } catch {
    toast.error('操作失败')
  }
}

function maskValue(value: string) {
  return '•'.repeat(Math.min(value.length, 20))
}

onMounted(() => {
  if (activeTab.value === ENV_TYPE.SECRET) {
    checkSecretStatus()
  }
  loadEnvVars()
})

onBeforeUnmount(() => {
  textareaResizeObserver?.disconnect()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">环境变量</h2>
        <p class="text-muted-foreground text-sm">管理脚本执行时的环境变量</p>
      </div>
      <div class="flex flex-col sm:flex-row items-center gap-2">
        <div class="flex w-full sm:w-auto items-center gap-2">
          <div class="relative flex-1 sm:flex-none">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input v-model="filterName" placeholder="搜索名称..." class="h-9 pl-9 w-full sm:w-48 text-sm"
              @input="handleSearch" />
          </div>
          <Button @click="openCreate" class="shrink-0 h-9" :disabled="activeTab === ENV_TYPE.SECRET && !isSecretSet">
            <Plus class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">新建{{ activeTab === ENV_TYPE.SECRET ? '机密' : '变量' }}</span>
          </Button>
        </div>
        <Tabs v-model="activeTab" class="w-full sm:w-auto shrink-0">
          <TabsList class="grid w-full grid-cols-2 sm:w-[200px] h-9">
            <TabsTrigger value="normal" class="text-sm">
              <span>环境变量</span>
            </TabsTrigger>
            <TabsTrigger value="secret" class="text-sm">
              <Shield class="w-3.5 h-3.5 mr-1" />
              <span>机密</span>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
    </div>

    <div v-if="activeTab === 'secret' && !isSecretSet" class="flex flex-col items-center justify-center p-12 text-center rounded-lg border bg-card border-dashed">
      <div class="h-12 w-12 rounded-full bg-destructive/10 flex items-center justify-center mb-4">
        <AlertTriangle class="h-6 w-6 text-destructive" />
      </div>
      <h3 class="text-lg font-bold mb-2">服务未配置加密秘钥</h3>
      <p class="text-sm text-muted-foreground max-w-md">
        必须在程序启动时通过环境变量 <code class="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">BAIHU_SECRET_KEY</code> 配置秘钥，才能启用机密管理功能。秘钥将仅存在于内存中，为您提供强安全的数据落盘加密保护。
      </p>
    </div>

    <div v-else class="rounded-lg border bg-card overflow-x-auto mt-4">
      <!-- 表头 -->
      <div
        class="flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium min-w-[500px]">
        <span class="w-32 sm:w-48 shrink-0">名称</span>
        <span class="w-24 sm:flex-1 shrink-0 sm:shrink">值</span>
        <span class="w-32 sm:w-48 shrink-0 hidden md:block">备注说明</span>
        <span class="w-28 sm:w-36 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y min-w-[500px]">
        <div v-if="envVars.length === 0" class="text-sm text-muted-foreground text-center py-8">
          {{ activeTab === ENV_TYPE.SECRET ? '暂无机密' : '暂无环境变量' }}
        </div>
        <div v-for="env in envVars" :key="env.id"
          class="flex items-center gap-4 px-4 py-2 hover:bg-muted/30 transition-colors">
          <code
            class="w-32 sm:w-48 font-medium truncate shrink-0 text-xs bg-muted/40 px-2 py-1 rounded">{{ env.name }}</code>
          <span class="w-24 sm:flex-1 shrink-0 sm:shrink font-mono text-muted-foreground truncate text-xs">
            <TextOverflow :text="showValues[env.id] ? env.value : maskValue(env.value)" title="查看值" />
          </span>
          <span class="w-32 sm:w-48 shrink-0 text-muted-foreground truncate text-sm hidden md:block">
            <TextOverflow :text="env.remark || '-'" title="备注描述" />
          </span>
          <span class="w-28 sm:w-36 shrink-0 flex justify-center gap-1">
            <Button variant="ghost" size="icon" class="h-7 w-7 rounded-md transition-all"
              :class="env.enabled ? 'text-green-500 bg-green-500/10 hover:bg-green-500/20' : 'text-muted-foreground bg-muted/50 hover:bg-muted'"
              @click="toggleEnabled(env)" :title="env.enabled ? '已启用（点击禁用）' : '已禁用（点击启用）'">
              <Zap v-if="env.enabled" class="h-3.5 w-3.5 fill-current" />
              <ZapOff v-else class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="toggleShow(env.id)"
              :title="showValues[env.id] ? '隐藏' : '显示'">
              <Eye v-if="!showValues[env.id]" class="h-3.5 w-3.5" />
              <EyeOff v-else class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEdit(env)" title="编辑">
              <Pencil class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(env.id)"
              title="删除">
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
          </span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <Dialog v-model:open="showDialog">
      <DialogContent class="w-[calc(100vw-2rem)] max-w-md min-w-0">
        <DialogHeader>
          <DialogTitle>{{ isEdit ? (editingEnv.type === ENV_TYPE.SECRET ? '更新机密' : '编辑变量') : (editingEnv.type === ENV_TYPE.SECRET ? '新建机密' : '新建变量') }}</DialogTitle>
          <div v-if="editingEnv.type === ENV_TYPE.SECRET" class="flex items-center gap-2.5 p-3 mt-3 rounded-xl bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400 text-xs leading-relaxed">
            <Shield class="h-4 w-4 shrink-0 text-amber-500" />
            <p>机密会在保存后得到<b class="text-amber-700 dark:text-amber-300">强加密保护</b>，且<b class="text-amber-700 dark:text-amber-300">仅在计划任务定时执行时才会被注入环境</b>。终端命令、调试运行、测试运行均无法获取机密内容。在执行日志内，机密文本会被自动打码。</p>
          </div>
          <DialogDescription v-else class="sr-only">编辑变量的名称、值、备注以及启用和隐藏状态。</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-2 min-w-0">
          <div class="space-y-2 min-w-0">
            <Label>{{ editingEnv.type === ENV_TYPE.SECRET ? '机密名称' : '变量名' }}</Label>
            <Input v-model="editingEnv.name" class="w-full min-w-0" :placeholder="editingEnv.type === ENV_TYPE.SECRET ? '例如：GITHUB_TOKEN' : 'MY_VAR'" />
          </div>
          <div class="space-y-2 min-w-0">
            <Label>
              {{ editingEnv.type === ENV_TYPE.SECRET ? '机密内容' : '变量值' }}
              <span v-if="editingEnv.type === ENV_TYPE.SECRET && isEdit" class="text-muted-foreground ml-1 font-normal text-xs">(输入新值即可覆盖)</span>
            </Label>
            <div class="relative flex min-w-0 overflow-hidden rounded-md border border-input bg-transparent shadow-xs focus-within:border-ring focus-within:ring-ring/50 focus-within:ring-[3px]">
              <div ref="lineNumbersRef" class="flex max-h-40 w-6 shrink-0 flex-col overflow-hidden border-r border-border bg-muted/30 py-2 text-right font-mono text-[10px] leading-6 text-muted-foreground">
                <span v-for="(line, index) in visualLineNumbers" :key="`${index}-${line}`" class="block h-6 px-1">{{ line }}</span>
              </div>
              <textarea
                ref="valueTextareaRef"
                v-model="editingEnv.value"
                rows="5"
                :placeholder="editingEnv.type === ENV_TYPE.SECRET && isEdit ? '' : (editingEnv.type === ENV_TYPE.SECRET ? '输入机密内容...' : '输入变量内容...')"
                class="max-h-40 min-h-16 w-full min-w-0 resize-none overflow-x-hidden bg-transparent pl-2 pr-3 py-2 font-mono placeholder:font-sans text-sm leading-6 break-all whitespace-pre-wrap outline-none"
                @scroll="syncValueLineNumbers"
              />
              <div aria-hidden="true" class="pointer-events-none absolute bottom-1.5 right-1.5 h-3.5 w-3.5 opacity-45">
                <span class="absolute bottom-0 right-0 h-px w-3 rotate-[-45deg] bg-border" />
                <span class="absolute bottom-1 right-0.5 h-px w-2 rotate-[-45deg] bg-border/80" />
                <span class="absolute bottom-2 right-1 h-px w-1 rotate-[-45deg] bg-border/60" />
              </div>
            </div>
            <div
              ref="lineMeasureRef"
              aria-hidden="true"
              class="pointer-events-none invisible fixed left-0 top-0 -z-10 min-h-16 break-all whitespace-pre-wrap px-2 py-2 font-mono text-sm leading-6"
            />
          </div>
          <div class="space-y-2 min-w-0">
            <Label>备注</Label>
            <Textarea v-model="editingEnv.remark" class="w-full min-w-0 resize-none break-all text-sm placeholder:font-sans" rows="3" :placeholder="editingEnv.type === ENV_TYPE.SECRET ? '机密用途说明...' : '变量用途说明...'" />
          </div>
          <div class="flex items-center justify-between space-x-2 pt-2" v-if="editingEnv.type !== ENV_TYPE.SECRET">
            <Label class="text-sm font-medium">隐藏变量值</Label>
            <Switch v-model="editingEnv.hidden" />
          </div>
          <div class="flex items-center justify-between space-x-2 pt-2">
            <Label class="text-sm font-medium">{{ editingEnv.type === ENV_TYPE.SECRET ? '启用机密' : '启用变量' }}</Label>
            <Switch v-model="editingEnv.enabled" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">取消</Button>
          <Button @click="saveEnv">保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>
            <div v-if="associatedTasks.length > 0" class="space-y-4 pt-1">
              <div class="flex items-start gap-3 p-3 rounded-lg bg-destructive/10 border border-destructive/20">
                <AlertTriangle class="h-5 w-5 text-destructive shrink-0 mt-0.5" />
                <div class="space-y-1">
                  <p class="text-sm font-bold text-destructive">{{ activeTab === ENV_TYPE.SECRET ? '机密' : '环境变量' }}正在使用中</p>
                  <p class="text-xs text-muted-foreground leading-relaxed">
                    该{{ activeTab === ENV_TYPE.SECRET ? '机密' : '变量' }}已被以下任务引用，直接删除可能导致任务运行失败。建议先移除引用或选择“强制删除”。
                  </p>
                </div>
              </div>

              <div class="space-y-2">
                <div class="flex items-center justify-between px-1">
                  <p class="text-[11px] font-bold text-muted-foreground uppercase tracking-widest">关联任务 ({{
                    associatedTasks.length }})</p>
                </div>
                <div class="bg-muted/30 rounded-lg p-1.5 max-h-40 overflow-y-auto space-y-1 border border-border/40">
                  <div v-for="task in associatedTasks" :key="task.id"
                    class="text-xs flex items-center justify-between bg-background/50 p-2 rounded-md border border-border/50 hover:bg-background transition-colors">
                    <div class="flex items-center gap-2 min-w-0">
                      <Terminal class="h-3 w-3 text-primary/70" />
                      <span class="font-medium truncate">{{ task.name }}</span>
                    </div>
                    <code
                      class="text-[10px] text-muted-foreground/70 font-mono bg-muted/50 px-1.5 py-0.5 rounded">{{ task.id }}</code>
                  </div>
                </div>
              </div>

              <div class="p-3 rounded-lg bg-secondary/30 border border-border/20">
                <p class="text-xs text-muted-foreground leading-relaxed">
                  <span class="font-bold text-foreground/80">提示：</span>选择强制删除将自动解除以上任务对该{{ activeTab === 'secret' ? '机密' : '变量' }}的绑定并执行物理删除。
                </p>
              </div>
            </div>
            <p v-else class="py-2">确定要删除此{{ activeTab === ENV_TYPE.SECRET ? '机密' : '环境变量' }}吗？此操作无法撤销，请谨慎操作。</p>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="isDeleting">取消</AlertDialogCancel>
          <Button v-if="associatedTasks.length > 0" variant="destructive" @click="deleteEnv(true)"
            :disabled="isDeleting">
            <template v-if="isDeleting">删除中...</template>
            <template v-else>强制删除</template>
          </Button>
          <Button v-else variant="destructive" @click="deleteEnv(false)" :disabled="isDeleting">
            <template v-if="isDeleting">删除中...</template>
            <template v-else>确认删除</template>
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
