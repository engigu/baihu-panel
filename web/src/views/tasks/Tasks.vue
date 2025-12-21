<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import Pagination from '@/components/Pagination.vue'
import { Plus, Play, Pencil, Trash2, Search } from 'lucide-vue-next'
import { api, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'

const { pageSize } = useSiteSettings()

const tasks = ref<Task[]>([])
const showDialog = ref(false)
const editingTask = ref<Partial<Task>>({})
const isEdit = ref(false)
const showDeleteDialog = ref(false)
const deleteTaskId = ref<number | null>(null)

const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const cronPresets = [
  { label: '每5秒', value: '*/5 * * * * *' },
  { label: '每30秒', value: '*/30 * * * * *' },
  { label: '每分钟', value: '0 * * * * *' },
  { label: '每5分钟', value: '0 */5 * * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每天0点', value: '0 0 0 * * *' },
  { label: '每天8点', value: '0 0 8 * * *' },
  { label: '每周一', value: '0 0 0 * * 1' },
  { label: '每月1号', value: '0 0 0 1 * *' },
]

async function loadTasks() {
  try {
    const res = await api.tasks.list({ page: currentPage.value, page_size: pageSize.value, name: filterName.value || undefined })
    tasks.value = res.data
    total.value = res.total
  } catch { toast.error('加载任务失败') }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadTasks()
  }, 300)
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadTasks()
}

function openCreate() {
  editingTask.value = { name: '', command: '', schedule: '0 * * * * *', timeout: 30, enabled: true }
  isEdit.value = false
  showDialog.value = true
}

function openEdit(task: Task) {
  editingTask.value = { ...task }
  isEdit.value = true
  showDialog.value = true
}

async function saveTask() {
  try {
    if (isEdit.value && editingTask.value.id) {
      await api.tasks.update(editingTask.value.id, editingTask.value)
      toast.success('任务已更新')
    } else {
      await api.tasks.create(editingTask.value)
      toast.success('任务已创建')
    }
    showDialog.value = false
    loadTasks()
  } catch { toast.error('保存失败') }
}

function confirmDelete(id: number) {
  deleteTaskId.value = id
  showDeleteDialog.value = true
}

async function deleteTask() {
  if (!deleteTaskId.value) return
  try {
    await api.tasks.delete(deleteTaskId.value)
    toast.success('任务已删除')
    loadTasks()
  } catch { toast.error('删除失败') }
  showDeleteDialog.value = false
  deleteTaskId.value = null
}

async function runTask(id: number) {
  try { await api.tasks.execute(id); toast.success('任务已执行') } catch { toast.error('执行失败') }
}

async function toggleTask(task: Task, enabled: boolean) {
  try {
    await api.tasks.update(task.id, { name: task.name, command: task.command, schedule: task.schedule, timeout: task.timeout, enabled })
    toast.success(enabled ? '任务已启用' : '任务已禁用')
    loadTasks()
  } catch { toast.error('操作失败') }
}

onMounted(loadTasks)
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold tracking-tight">定时任务</h2>
        <p class="text-muted-foreground">管理和调度自动化任务</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterName" placeholder="搜索任务..." class="h-9 pl-9 w-56 text-sm" @input="handleSearch" />
        </div>
        <Button @click="openCreate">
          <Plus class="h-4 w-4 mr-2" /> 新建任务
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card">
      <!-- 表头 -->
      <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium">
        <span class="w-12 shrink-0">ID</span>
        <span class="w-40 shrink-0">名称</span>
        <span class="flex-1">命令</span>
        <span class="w-32 shrink-0">定时规则</span>
        <span class="w-40 shrink-0">上次执行</span>
        <span class="w-40 shrink-0">下次执行</span>
        <span class="w-12 shrink-0 text-center">状态</span>
        <span class="w-28 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无任务
        </div>
        <div
          v-for="task in tasks"
          :key="task.id"
          class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors"
        >
          <span class="w-12 shrink-0 text-muted-foreground text-sm">#{{ task.id }}</span>
          <span class="w-40 font-medium truncate shrink-0 text-sm">{{ task.name }}</span>
          <code class="flex-1 text-muted-foreground truncate text-xs bg-muted px-2 py-1 rounded">{{ task.command }}</code>
          <code class="w-36 shrink-0 text-muted-foreground text-xs bg-muted px-2 py-1 rounded">{{ task.schedule }}</code>
          <span class="w-40 shrink-0 text-muted-foreground text-xs">{{ task.last_run || '-' }}</span>
          <span class="w-40 shrink-0 text-muted-foreground text-xs">{{ task.next_run || '-' }}</span>
          <span class="w-12 flex justify-center shrink-0 cursor-pointer" @click="toggleTask(task, !task.enabled)" :title="task.enabled ? '点击禁用' : '点击启用'">
            <span :class="['w-2 h-2 rounded-full', task.enabled ? 'bg-green-500' : 'bg-gray-400']" />
          </span>
          <span class="w-28 shrink-0 flex justify-center gap-1">
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="runTask(task.id)" title="执行">
              <Play class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEdit(task)" title="编辑">
              <Pencil class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(task.id)" title="删除">
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
          </span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <Dialog v-model:open="showDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{{ isEdit ? '编辑任务' : '新建任务' }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">任务名称</Label>
            <Input v-model="editingTask.name" placeholder="我的任务" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">执行命令</Label>
            <Input v-model="editingTask.command" placeholder="node script.js" class="col-span-3 font-mono" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">定时规则</Label>
            <Input v-model="editingTask.schedule" placeholder="0 * * * * *" class="col-span-3 font-mono" />
          </div>
          <div class="grid grid-cols-4 items-start gap-4">
            <span></span>
            <div class="col-span-3">
              <p class="text-xs text-muted-foreground mb-2">格式: 秒 分 时 日 月 周</p>
              <div class="flex flex-wrap gap-1.5">
              <span
                v-for="preset in cronPresets"
                :key="preset.value"
                class="px-2 py-0.5 text-xs rounded-md bg-muted hover:bg-accent cursor-pointer transition-colors"
                @click="editingTask.schedule = preset.value"
              >
                {{ preset.label }}
              </span>
              </div>
            </div>
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">超时(分钟)</Label>
            <Input v-model.number="editingTask.timeout" type="number" placeholder="30" class="col-span-3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">取消</Button>
          <Button @click="saveTask">保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>确定要删除此任务吗？此操作无法撤销。</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteTask">删除</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
