<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Plus, Trash2, Pencil, ShieldAlert, AlertCircle, Copy, Check } from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { api, type NotifyFilter, type EventType } from '@/api'
import { toast } from 'vue-sonner'
import { copyToClipboard } from '@/utils/clipboard'

const props = defineProps<{
  eventTypes: EventType[]
}>()

const filters = ref<NotifyFilter[]>([])
const loading = ref(false)
const copiedBlock = ref<string | null>(null)

// 编辑规则表单弹窗状态
const showDialog = ref(false)
const isEditing = ref(false)
const editingFilter = ref<Partial<NotifyFilter>>({
  name: '',
  event: 'all',
  keyword: '',
  match_field: 'all',
  is_regex: false,
  action: 'silent',
  custom_title: '',
  custom_text: '',
  enabled: true
})

// 删除规则状态
const showDeleteConfirm = ref(false)
const deletingFilterId = ref('')

// 加载所有规则
async function loadFilters() {
  loading.value = true
  try {
    filters.value = await api.notify.getFilters()
  } catch (e: any) {
    toast.error('加载过滤规则失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

// 切换规则启用状态
async function toggleFilter(filter: NotifyFilter, enabled: boolean) {
  try {
    await api.notify.saveFilter({
      ...filter,
      enabled
    })
    filter.enabled = enabled
    toast.success(`${enabled ? '已启用' : '已禁用'}规则: ${filter.name}`)
  } catch (e: any) {
    toast.error('操作失败: ' + e.message)
  }
}

// 打开新建弹窗
function openNewDialog() {
  editingFilter.value = {
    name: '',
    event: 'all',
    keyword: '',
    match_field: 'all',
    is_regex: false,
    action: 'silent',
    custom_title: '',
    custom_text: '',
    enabled: true
  }
  isEditing.value = false
  showDialog.value = true
}

// 打开编辑弹窗
function openEditDialog(filter: NotifyFilter) {
  editingFilter.value = { ...filter }
  isEditing.value = true
  showDialog.value = true
}

// 保存规则
async function saveFilter() {
  const f = editingFilter.value
  if (!f.name?.trim()) {
    toast.error('请输入规则名称')
    return
  }
  if (!f.keyword?.trim()) {
    toast.error('请输入匹配关键字')
    return
  }
  if (f.action === 'custom' && !f.custom_title?.trim() && !f.custom_text?.trim()) {
    toast.error('自定义动作必须填写自定义标题或内容模版')
    return
  }

  try {
    await api.notify.saveFilter(f)
    toast.success(isEditing.value ? '规则更新成功' : '规则创建成功')
    showDialog.value = false
    await loadFilters()
  } catch (e: any) {
    toast.error('保存失败: ' + e.message)
  }
}

// 确认删除
function confirmDelete(id: string) {
  deletingFilterId.value = id
  showDeleteConfirm.value = true
}

// 执行删除
async function deleteFilter() {
  try {
    await api.notify.deleteFilter(deletingFilterId.value)
    toast.success('删除成功')
    showDeleteConfirm.value = false
    await loadFilters()
  } catch (e: any) {
    toast.error('删除失败: ' + e.message)
  }
}

function handleCopy(text: string, blockId: string) {
  copyToClipboard(text).then((success) => {
    if (success) {
      copiedBlock.value = blockId
      toast.success('已复制规则 ID')
      setTimeout(() => {
        copiedBlock.value = null
      }, 2000)
    }
  })
}

function getEventLabel(eventValue: string): string {
  if (eventValue === 'all') return '所有事件'
  const found = props.eventTypes.find(t => t.type === eventValue)
  return found ? found.label : eventValue
}

onMounted(() => {
  loadFilters()
})
</script>

<template>
  <Card>
    <CardHeader class="pb-5">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div class="flex items-start gap-2.5">
          <ShieldAlert class="w-5 h-5 text-primary shrink-0 mt-0.5" />
          <div>
            <CardTitle>通知匹配过滤规则</CardTitle>
            <CardDescription class="text-xs sm:text-sm">根据事件内容、日志中的关键字，对全局通知进行静默拦截或转换自定义通知格式</CardDescription>
          </div>
        </div>
        <Button size="sm" @click="openNewDialog" class="w-full sm:w-auto shrink-0">
          <Plus class="w-4 h-4 mr-1" />
          添加规则
        </Button>
      </div>
    </CardHeader>
    <CardContent>
      <div v-if="loading" class="text-center py-12 text-muted-foreground">
        正在加载过滤规则...
      </div>
      <div v-else-if="filters.length === 0" class="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
        <ShieldAlert class="w-10 h-10 mx-auto mb-2 opacity-20" />
        <p class="text-xs font-medium">暂无过滤匹配规则</p>
        <p class="text-[10px] opacity-70 mt-1">点击右上角"添加规则"配置拦截或伪装通知</p>
      </div>
      <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div v-for="filter in filters" :key="filter.id"
          class="flex flex-col p-4 rounded-lg border bg-card hover:bg-accent/10 transition-colors">
          
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2 flex-wrap min-w-0">
              <span class="font-bold text-sm truncate max-w-[150px]">{{ filter.name }}</span>
              <Badge variant="outline" class="text-[9px] px-1 h-4 font-normal">{{ getEventLabel(filter.event) }}</Badge>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-[10px] text-muted-foreground">{{ filter.enabled ? '已启用' : '已禁用' }}</span>
              <Switch :model-value="filter.enabled" @update:model-value="(val: boolean) => toggleFilter(filter, val)" class="scale-75 origin-right" />
            </div>
          </div>

          <div class="flex flex-col gap-1 text-[11px] text-muted-foreground mb-4">
            <div class="flex items-center justify-between">
              <span>匹配范围:</span>
              <span class="text-foreground/90 font-medium">
                {{ filter.match_field === 'all' ? '标题 + 正文 + 日志' : filter.match_field === 'content' ? '正文' : '日志' }}
              </span>
            </div>
            <div class="flex items-center justify-between gap-2">
              <span>关键字:</span>
              <code class="px-1.5 py-0.5 rounded bg-secondary/50 text-foreground font-mono text-[10px] truncate max-w-[180px]">
                {{ filter.keyword }}
              </code>
            </div>
            <div class="flex items-center justify-between">
              <span>匹配动作:</span>
              <Badge :variant="filter.action === 'silent' ? 'destructive' : 'default'" class="text-[9px] px-1 h-3.5 font-normal">
                {{ filter.action === 'silent' ? '静默拦截' : '自定义格式' }}
              </Badge>
            </div>
            <div v-if="filter.action === 'custom'" class="border-t border-border/30 mt-1.5 pt-1.5 space-y-0.5 text-[10px]">
              <div v-if="filter.custom_title" class="truncate"><b>新标题:</b> {{ filter.custom_title }}</div>
              <div v-if="filter.custom_text" class="truncate"><b>新正文:</b> {{ filter.custom_text }}</div>
            </div>
          </div>

          <div class="mt-auto pt-3 border-t flex items-center justify-between">
            <span class="text-[9px] text-muted-foreground/60 font-mono truncate max-w-[120px]">ID: {{ filter.id }}</span>
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="icon"
                class="h-7 w-7 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                @click="handleCopy(filter.id, 'filter-' + filter.id)" title="复制 ID">
                <Check v-if="copiedBlock === 'filter-' + filter.id" class="w-3.5 h-3.5 text-emerald-500" />
                <Copy v-else class="w-3.5 h-3.5 text-muted-foreground/80" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-7 w-7 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                @click="openEditDialog(filter)" title="编辑">
                <Pencil class="w-3.5 h-3.5 text-muted-foreground/80" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-7 w-7 rounded-full text-destructive hover:bg-destructive/10 transition-colors"
                @click="confirmDelete(filter.id)" title="删除">
                <Trash2 class="w-3.5 h-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>

  <!-- 规则配置弹窗 -->
  <Dialog :open="showDialog" @update:open="showDialog = $event">
    <DialogContent class="sm:max-w-lg max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ isEditing ? '编辑过滤规则' : '添加过滤规则' }}</DialogTitle>
        <DialogDescription>配置通知的事件拦截与自定义伪装逻辑</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <!-- 规则名称 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">规则名称<span class="text-destructive ml-0.5">*</span></Label>
          <Input v-model="editingFilter.name" placeholder="例如：过滤掉测试任务失败告警" />
        </div>

        <!-- 目标事件 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">目标事件</Label>
          <Select v-model="editingFilter.event">
            <SelectTrigger>
              <SelectValue placeholder="选择目标事件" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有事件</SelectItem>
              <SelectItem v-for="t in eventTypes" :key="t.type" :value="t.type">
                {{ t.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- 匹配字段 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">匹配字段</Label>
          <Select v-model="editingFilter.match_field">
            <SelectTrigger>
              <SelectValue placeholder="选择匹配字段范围" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">标题 + 正文 + 日志</SelectItem>
              <SelectItem value="content">正文内容 (Content)</SelectItem>
              <SelectItem value="log">任务执行日志 (Log)</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- 匹配关键字 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">匹配关键字<span class="text-destructive ml-0.5">*</span></Label>
          <Input v-model="editingFilter.keyword" placeholder="匹配的文本短语或正则表达式" class="font-mono text-sm" />
        </div>

        <!-- 正则表达式模式 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">正则匹配</Label>
          <div class="flex items-center gap-2">
            <Switch :model-value="editingFilter.is_regex || false" @update:model-value="(val: boolean) => editingFilter.is_regex = val" />
            <span class="text-xs text-muted-foreground">开启后匹配关键字将按正则表达式解析</span>
          </div>
        </div>

        <!-- 执行动作 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">匹配后动作</Label>
          <Select v-model="editingFilter.action">
            <SelectTrigger>
              <SelectValue placeholder="选择匹配后的执行动作" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="silent">静默拦截 (丢弃通知)</SelectItem>
              <SelectItem value="custom">自定义转换 (转换内容)</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- 启用状态 -->
        <div class="grid grid-cols-[110px_1fr] items-center gap-3">
          <Label class="text-right text-sm">设置为开启</Label>
          <Switch :model-value="editingFilter.enabled ?? true" @update:model-value="(val: boolean) => editingFilter.enabled = val" />
        </div>

        <!-- 自定义通知字段，仅在动作是 custom 时展示 -->
        <div v-if="editingFilter.action === 'custom'" class="border-t pt-4 mt-4 space-y-4">
          <div class="flex flex-col gap-1.5 px-3 py-2 rounded bg-muted/40 text-[11px] text-muted-foreground leading-normal border border-border/40">
            <div class="flex items-center gap-1.5 font-bold text-foreground">
              <AlertCircle class="h-3.5 w-3.5 text-primary shrink-0" />
              <span>支持以下全局/事件专属占位符：</span>
            </div>
            <div class="grid grid-cols-2 gap-x-4 gap-y-1 mt-1 pl-5 list-disc">
              <div>• <code v-text="'{{username}}'"></code> : 关联的用户名 (登录/修改密码)</div>
              <div>• <code v-text="'{{ip}}'"></code> : 来源 IP 地址 (登录/破译警告)</div>
              <div>• <code v-text="'{{status_label}}'"></code> : 登录状态 (成功/失败)</div>
              <div>• <code v-text="'{{task_id}}'"></code> : 计划任务 ID</div>
              <div>• <code v-text="'{{task_name}}'"></code> : 计划任务名称</div>
              <div>• <code v-text="'{{start_time}}'"></code> : 任务启动日期时间</div>
              <div>• <code v-text="'{{duration}}'"></code> : 任务运行耗时 (ms)</div>
              <div>• <code v-text="'{{error}}'"></code> : 任务失败的错误详情</div>
            </div>
          </div>

          <div class="grid grid-cols-[110px_1fr] items-center gap-3">
            <Label class="text-right text-sm">自定义标题</Label>
            <Input v-model="editingFilter.custom_title" placeholder="留空则保持原标题不变" />
          </div>

          <div class="grid grid-cols-[110px_1fr] items-start gap-3">
            <Label class="text-right text-sm pt-1.5">自定义正文</Label>
            <textarea
              v-model="editingFilter.custom_text"
              placeholder="填写修改后的正文模板，留空则保持原正文不变"
              class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring min-h-[90px] resize-none"
            />
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="showDialog = false">取消</Button>
        <Button @click="saveFilter">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- 删除确认 -->
  <BaihuDialog v-model:open="showDeleteConfirm" title="确认删除规则?">
    <div class="text-[15px] leading-relaxed text-muted-foreground">
      规则删除后将不再拦截相匹配的通知，且无法撤销。确定要删除吗？
    </div>
    <template #footer>
      <Button variant="ghost" @click="showDeleteConfirm = false">取消</Button>
      <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="deleteFilter">确认删除</Button>
    </template>
  </BaihuDialog>
</template>
