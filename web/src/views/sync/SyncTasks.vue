<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import Pagination from '@/components/Pagination.vue'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { Plus, Play, Pencil, Trash2, Search, ScrollText } from 'lucide-vue-next'
import { api, type SyncTask } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter } from 'vue-router'
import TextOverflow from '@/components/TextOverflow.vue'

const router = useRouter()
const { pageSize } = useSiteSettings()

const tasks = ref<SyncTask[]>([])
const showDialog = ref(false)
const editingTask = ref<Partial<SyncTask>>({})
const isEdit = ref(false)
const showDeleteDialog = ref(false)
const deleteTaskId = ref<number | null>(null)
const cleanType = ref('')
const cleanKeep = ref(30)
const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const cronPresets = [
  { label: '每5分钟', value: '0 */5 * * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每天0点', value: '0 0 0 * * *' },
  { label: '每天8点', value: '0 0 8 * * *' },
  { label: '每周一', value: '0 0 0 * * 1' },
  { label: '每月1号', value: '0 0 0 1 * *' },
]

const proxyOptions = [
  { label: '不使用代理', value: 'none' },
  { label: 'ghproxy.com', value: 'ghproxy' },
  { label: 'mirror.ghproxy.com', value: 'mirror' },
  { label: '自定义代理', value: 'custom' },
]

const cleanConfig = computed(() => {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) return ''
  return JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
})

async function loadTasks() {
  try {
    const res = await api.syncTasks.list({ page: currentPage.value, page_size: pageSize.value, name: filterName.value || undefined })
    tasks.value = res.data
    total.value = res.total
  } catch { toast.error('加载任务失败') }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { currentPage.value = 1; loadTasks() }, 300)
}

function handlePageChange(page: number) { currentPage.value = page; loadTasks() }

function openCreate() {
  editingTask.value = { name: '', source_type: 'url', source_url: '', branch: '', target_path: '', schedule: '0 0 * * * *', proxy: 'none', proxy_url: '', auth_token: '', enabled: true }
  cleanType.value = 'none'; cleanKeep.value = 30; isEdit.value = false; showDialog.value = true
}

function openEdit(task: SyncTask) {
  editingTask.value = { ...task }
  if (task.clean_config) {
    try { const c = JSON.parse(task.clean_config); cleanType.value = c.type || 'none'; cleanKeep.value = c.keep || 30 }
    catch { cleanType.value = 'none'; cleanKeep.value = 30 }
  } else { cleanType.value = 'none'; cleanKeep.value = 30 }
  isEdit.value = true; showDialog.value = true
}

async function saveTask() {
  try {
    editingTask.value.clean_config = cleanConfig.value
    if (isEdit.value && editingTask.value.id) { await api.syncTasks.update(editingTask.value.id, editingTask.value); toast.success('任务已更新') }
    else { await api.syncTasks.create(editingTask.value); toast.success('任务已创建') }
    showDialog.value = false; loadTasks()
  } catch { toast.error('保存失败') }
}

function confirmDelete(id: number) { deleteTaskId.value = id; showDeleteDialog.value = true }

async function deleteTask() {
  if (!deleteTaskId.value) return
  try { await api.syncTasks.delete(deleteTaskId.value); toast.success('任务已删除'); loadTasks() }
  catch { toast.error('删除失败') }
  showDeleteDialog.value = false; deleteTaskId.value = null
}

async function runTask(id: number) {
  try { await api.syncTasks.execute(id); toast.success('同步任务已执行'); loadTasks() }
  catch { toast.error('执行失败') }
}

async function toggleTask(task: SyncTask, enabled: boolean) {
  try { await api.syncTasks.update(task.id, { ...task, enabled }); toast.success(enabled ? '任务已启用' : '任务已禁用'); loadTasks() }
  catch { toast.error('操作失败') }
}

function viewLogs(taskId: number) { router.push({ path: '/sync/logs', query: { sync_task_id: String(taskId) } }) }

onMounted(() => { loadTasks() })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">定时同步</h2>
        <p class="text-muted-foreground text-sm">从远程 URL 或 Git 仓库同步文件</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterName" placeholder="搜索任务..." class="h-9 pl-9 w-full sm:w-56 text-sm" @input="handleSearch" />
        </div>
        <Button @click="openCreate" class="shrink-0"><Plus class="h-4 w-4 sm:mr-2" /><span class="hidden sm:inline">新建同步</span></Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[700px]">
        <span class="w-12 shrink-0">ID</span>
        <span class="w-24 shrink-0">名称</span>
        <span class="w-16 shrink-0">类型</span>
        <span class="flex-1 shrink-0">源地址</span>
        <span class="w-32 shrink-0 hidden md:block">定时规则</span>
        <span class="w-36 shrink-0 hidden lg:block">上次同步</span>
        <span class="w-12 shrink-0 text-center">状态</span>
        <span class="w-32 shrink-0 text-center">操作</span>
      </div>
      <div class="divide-y min-w-[700px]">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">暂无同步任务</div>
        <div v-for="task in tasks" :key="task.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
          <span class="w-12 shrink-0 text-muted-foreground text-sm">#{{ task.id }}</span>
          <span class="w-24 font-medium truncate shrink-0 text-sm"><TextOverflow :text="task.name" title="任务名称" /></span>
          <span class="w-16 shrink-0"><span :class="['px-2 py-0.5 text-xs rounded', task.source_type === 'git' ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300' : 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300']">{{ task.source_type.toUpperCase() }}</span></span>
          <code class="flex-1 shrink-0 text-muted-foreground truncate text-xs bg-muted px-2 py-1 rounded"><TextOverflow :text="task.source_url" title="源地址" /></code>
          <code class="w-32 shrink-0 text-muted-foreground text-xs bg-muted px-2 py-1 rounded hidden md:block">{{ task.schedule }}</code>
          <span class="w-36 shrink-0 text-muted-foreground text-xs hidden lg:flex items-center gap-1">{{ task.last_sync || '-' }}<span v-if="task.last_status" :class="['w-2 h-2 rounded-full', task.last_status === 'success' ? 'bg-green-500' : 'bg-red-500']" /></span>
          <span class="w-12 flex justify-center shrink-0 cursor-pointer" @click="toggleTask(task, !task.enabled)" :title="task.enabled ? '点击禁用' : '点击启用'"><span :class="['w-2 h-2 rounded-full', task.enabled ? 'bg-green-500' : 'bg-gray-400']" /></span>
          <span class="w-32 shrink-0 flex justify-center gap-1">
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="runTask(task.id)" title="执行"><Play class="h-3.5 w-3.5" /></Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="viewLogs(task.id)" title="日志"><ScrollText class="h-3.5 w-3.5" /></Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEdit(task)" title="编辑"><Pencil class="h-3.5 w-3.5" /></Button>
            <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(task.id)" title="删除"><Trash2 class="h-3.5 w-3.5" /></Button>
          </span>
        </div>
      </div>
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <Dialog v-model:open="showDialog">
      <DialogContent class="sm:max-w-[550px] max-h-[90vh] overflow-y-auto">
        <DialogHeader><DialogTitle>{{ isEdit ? '编辑同步任务' : '新建同步任务' }}</DialogTitle></DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">任务名称</Label>
            <Input v-model="editingTask.name" placeholder="我的同步任务" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">源类型</Label>
            <Select :model-value="editingTask.source_type" @update:model-value="v => editingTask.source_type = String(v)">
              <SelectTrigger class="col-span-3"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="url">URL (单文件)</SelectItem><SelectItem value="git">Git (仓库)</SelectItem></SelectContent>
            </Select>
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">源地址</Label>
            <Input v-model="editingTask.source_url" :placeholder="editingTask.source_type === 'git' ? 'https://github.com/user/repo.git' : 'https://example.com/file.py'" class="col-span-3 font-mono text-sm" />
          </div>
          <div v-if="editingTask.source_type === 'git'" class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">分支</Label>
            <Input v-model="editingTask.branch" placeholder="main (留空使用默认分支)" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">目标路径</Label>
            <div class="col-span-3">
              <DirTreeSelect :model-value="editingTask.target_path || ''" @update:model-value="v => editingTask.target_path = v" :show-files="editingTask.source_type === 'url'" />
              <p class="text-xs text-muted-foreground mt-1">{{ editingTask.source_type === 'git' ? '仓库将克隆到此目录' : '文件将保存到此路径' }}</p>
            </div>
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">定时规则</Label>
            <Input v-model="editingTask.schedule" placeholder="0 0 * * * *" class="col-span-3 font-mono" />
          </div>
          <div class="grid grid-cols-4 items-start gap-4">
            <span></span>
            <div class="col-span-3">
              <p class="text-xs text-muted-foreground mb-2">格式: 秒 分 时 日 月 周</p>
              <div class="flex flex-wrap gap-1.5">
                <span v-for="preset in cronPresets" :key="preset.value" class="px-2 py-0.5 text-xs rounded-md bg-muted hover:bg-accent cursor-pointer transition-colors" @click="editingTask.schedule = preset.value">{{ preset.label }}</span>
              </div>
            </div>
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">代理加速</Label>
            <Select :model-value="editingTask.proxy || 'none'" @update:model-value="v => editingTask.proxy = String(v)">
              <SelectTrigger class="col-span-3"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem v-for="opt in proxyOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem></SelectContent>
            </Select>
          </div>
          <div v-if="editingTask.proxy === 'custom'" class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">代理地址</Label>
            <Input v-model="editingTask.proxy_url" placeholder="https://your-proxy.com/" class="col-span-3 font-mono text-sm" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">认证Token</Label>
            <Input v-model="editingTask.auth_token" type="password" placeholder="可选，用于私有仓库" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">日志清理</Label>
            <div class="col-span-3 flex gap-2">
              <Select :model-value="cleanType" @update:model-value="v => cleanType = String(v || 'none')">
                <SelectTrigger class="w-28"><SelectValue placeholder="不清理" /></SelectTrigger>
                <SelectContent><SelectItem value="none">不清理</SelectItem><SelectItem value="day">按天数</SelectItem><SelectItem value="count">按条数</SelectItem></SelectContent>
              </Select>
              <Input v-if="cleanType && cleanType !== 'none'" v-model.number="cleanKeep" type="number" :placeholder="cleanType === 'day' ? '保留天数' : '保留条数'" class="flex-1" />
            </div>
          </div>
        </div>
        <DialogFooter><Button variant="outline" @click="showDialog = false">取消</Button><Button @click="saveTask">保存</Button></DialogFooter>
      </DialogContent>
    </Dialog>

    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader><AlertDialogTitle>确认删除</AlertDialogTitle><AlertDialogDescription>确定要删除此同步任务吗？此操作无法撤销。</AlertDialogDescription></AlertDialogHeader>
        <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteTask">删除</AlertDialogAction></AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
