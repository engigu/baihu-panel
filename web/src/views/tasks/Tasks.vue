<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import TaskDialog from './TaskDialog.vue'
import RepoDialog from './RepoDialog.vue'
import { Plus, Play, Pencil, Trash2, Search, ScrollText, GitBranch, Terminal } from 'lucide-vue-next'
import { api, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter } from 'vue-router'
import TextOverflow from '@/components/TextOverflow.vue'

const router = useRouter()
const { pageSize } = useSiteSettings()

const tasks = ref<Task[]>([])
const showTaskDialog = ref(false)
const showRepoDialog = ref(false)
const editingTask = ref<Partial<Task>>({})
const isEdit = ref(false)
const showDeleteDialog = ref(false)
const deleteTaskId = ref<number | null>(null)

const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

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
  editingTask.value = { name: '', command: '', type: 'task', schedule: '0 * * * * *', timeout: 30, work_dir: '', enabled: true, clean_config: '', envs: '' }
  isEdit.value = false
  showTaskDialog.value = true
}

function openCreateRepo() {
  editingTask.value = { name: '', type: 'repo', schedule: '0 0 0 * * *', timeout: 30, enabled: true, clean_config: '', envs: '' }
  isEdit.value = false
  showRepoDialog.value = true
}

function openEdit(task: Task) {
  editingTask.value = { ...task }
  isEdit.value = true
  if (task.type === 'repo') {
    showRepoDialog.value = true
  } else {
    showTaskDialog.value = true
  }
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
    await api.tasks.update(task.id, { ...task, enabled })
    toast.success(enabled ? '任务已启用' : '任务已禁用')
    loadTasks()
  } catch { toast.error('操作失败') }
}

function viewLogs(taskId: number) {
  router.push({ path: '/history', query: { task_id: String(taskId) } })
}

function getTaskTypeTitle(type: string) {
  return type === 'repo' ? '仓库同步' : '普通任务'
}

onMounted(loadTasks)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">定时任务</h2>
        <p class="text-muted-foreground text-sm">管理和调度自动化任务</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterName" placeholder="搜索任务..." class="h-9 pl-9 w-full sm:w-56 text-sm" @input="handleSearch" />
        </div>
        <Button variant="outline" @click="openCreateRepo" class="shrink-0">
          <GitBranch class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">仓库同步</span>
        </Button>
        <Button @click="openCreate" class="shrink-0">
          <Plus class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">新建任务</span>
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <!-- 表头 -->
      <div class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 border-b bg-muted/50 text-xs sm:text-sm text-muted-foreground font-medium min-w-[360px] sm:min-w-[700px]">
        <span class="w-12 sm:w-14 shrink-0">ID</span>
        <span class="w-6 sm:w-12 shrink-0 text-center">类型</span>
        <span class="flex-1 min-w-0">名称</span>
        <span class="w-32 sm:flex-1 shrink-0 sm:shrink hidden sm:block">命令/地址</span>
        <span class="w-32 shrink-0 hidden md:block">定时规则</span>
        <span class="w-40 shrink-0 hidden lg:block">上次执行</span>
        <span class="w-40 shrink-0 hidden lg:block">下次执行</span>
        <span class="w-8 sm:w-12 shrink-0 text-center">状态</span>
        <span class="w-20 sm:w-36 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y min-w-[360px] sm:min-w-[700px]">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无任务
        </div>
        <div
          v-for="task in tasks"
          :key="task.id"
          class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 hover:bg-muted/50 transition-colors"
        >
          <span class="w-12 sm:w-14 shrink-0 text-muted-foreground text-xs sm:text-sm">#{{ task.id }}</span>
          <span class="w-6 sm:w-8 shrink-0 flex justify-center" :title="getTaskTypeTitle(task.type || 'task')">
            <GitBranch v-if="task.type === 'repo'" class="h-3.5 w-3.5 sm:h-4 sm:w-4 text-primary" />
            <Terminal v-else class="h-3.5 w-3.5 sm:h-4 sm:w-4 text-primary" />
          </span>
          <span class="flex-1 min-w-0 font-medium truncate text-xs sm:text-sm">{{ task.name }}</span>
          <code class="w-32 sm:flex-1 shrink-0 sm:shrink text-muted-foreground truncate text-xs bg-muted px-2 py-1 rounded hidden sm:block">
            <TextOverflow :text="task.command" :title="task.type === 'repo' ? '同步地址' : '执行命令'" />
          </code>
          <code class="w-36 shrink-0 text-muted-foreground text-xs bg-muted px-2 py-1 rounded hidden md:block">{{ task.schedule }}</code>
          <span class="w-40 shrink-0 text-muted-foreground text-xs hidden lg:block">{{ task.last_run || '-' }}</span>
          <span class="w-40 shrink-0 text-muted-foreground text-xs hidden lg:block">{{ task.next_run || '-' }}</span>
          <span class="w-8 sm:w-12 flex justify-center shrink-0 cursor-pointer" @click="toggleTask(task, !task.enabled)" :title="task.enabled ? '点击禁用' : '点击启用'">
            <span :class="['w-2 h-2 rounded-full', task.enabled ? 'bg-green-500' : 'bg-gray-400']" />
          </span>
          <span class="w-20 sm:w-36 shrink-0 flex justify-center gap-0.5 sm:gap-1">
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="runTask(task.id)" title="执行">
              <Play class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="viewLogs(task.id)" title="日志">
              <ScrollText class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="openEdit(task)" title="编辑">
              <Pencil class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7 text-destructive" @click="confirmDelete(task.id)" title="删除">
              <Trash2 class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
          </span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <!-- 普通任务弹窗 -->
    <TaskDialog v-model:open="showTaskDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 仓库同步弹窗 -->
    <RepoDialog v-model:open="showRepoDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 删除确认 -->
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
