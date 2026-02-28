<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import { Plus, Pencil, Trash2, Search, Network, CheckCircle2, Clock, Play } from 'lucide-vue-next'
import { api, type Workflow } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter } from 'vue-router'
import WorkflowDialog from './WorkflowDialog.vue'

const router = useRouter()
const { pageSize } = useSiteSettings()

const workflows = ref<Workflow[]>([])
const showDialog = ref(false)
const editingWorkflow = ref<Partial<Workflow>>({})
const isEdit = ref(false)
const showDeleteDialog = ref(false)
const deleteId = ref<string | null>(null)

const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

async function loadWorkflows() {
  try {
    const res = await api.workflows.list({
      page: currentPage.value,
      page_size: pageSize.value,
      name: filterName.value || undefined
    })
    workflows.value = res.data
    total.value = res.total
  } catch { toast.error('加载工作流失败') }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadWorkflows()
  }, 300)
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadWorkflows()
}

function openCreate() {
  editingWorkflow.value = { name: '', description: '', schedule: '', enabled: true, flow_data: '' }
  isEdit.value = false
  showDialog.value = true
}

function openEdit(workflow: Workflow) {
  editingWorkflow.value = { ...workflow }
  isEdit.value = true
  showDialog.value = true
}

function confirmDelete(id: string) {
  deleteId.value = id
  showDeleteDialog.value = true
}

async function deleteWorkflow() {
  if (!deleteId.value) return
  try {
    await api.workflows.delete(deleteId.value)
    toast.success('工作流已删除')
    loadWorkflows()
  } catch { toast.error('删除失败') }
  showDeleteDialog.value = false
  deleteId.value = null
}

async function toggleWorkflow(workflow: Workflow, enabled: boolean) {
  try {
    await api.workflows.update(workflow.id, { ...workflow, enabled })
    toast.success(enabled ? '工作流已启用' : '工作流已禁用')
    loadWorkflows()
  } catch { toast.error('操作失败') }
}

function editBoard(id: string) {
  router.push(`/workflows/${id}`)
}

async function runWorkflow(id: string) {
  try {
    await api.workflows.run(id)
    toast.success('工作流已触发后台运行')
    loadWorkflows()
  } catch(err: any) { 
    toast.error('触发工作流失败: ' + (err.message || ''))
  }
}

onMounted(() => {
  loadWorkflows()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">工作流编排</h2>
        <p class="text-muted-foreground text-sm">将独立任务串接为复杂的自动化流水线</p>
      </div>
      <div class="flex flex-col sm:flex-row gap-2.5 w-full md:w-auto">
        <div class="flex items-center gap-2 w-full sm:w-auto">
          <div class="relative flex-1 sm:flex-none">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input v-model="filterName" placeholder="搜索工作流..." class="h-9 pl-9 w-full sm:w-48 text-sm"
              @input="handleSearch" />
          </div>
        </div>
        <Button @click="openCreate" class="gap-2 shrink-0">
          <Plus class="h-4 w-4" />
          创建工作流
        </Button>
      </div>
    </div>

    <!-- 卡片网格 -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div v-for="workflow in workflows" :key="workflow.id"
        class="group relative flex flex-col bg-card text-card-foreground rounded-lg border shadow-sm hover:shadow-md transition-all">
        <!-- 卡片头部: 标题和状态 -->
        <div class="p-4 sm:p-5 flex flex-col gap-3">
          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-2.5 overflow-hidden">
              <div class="p-1.5 rounded-md text-purple-600 bg-purple-50 shrink-0 border border-purple-100">
                <Network class="h-4 w-4" />
              </div>
              <h3 class="font-medium truncate" :title="workflow.name">
                {{ workflow.name }}
              </h3>
            </div>
          </div>
          <div class="text-xs text-muted-foreground line-clamp-2 h-8">
            {{ workflow.description || '暂无描述' }}
          </div>
        </div>

        <!-- 详细信息 -->
        <div class="px-4 sm:px-5 pb-4 flex flex-col gap-2.5">
          <div class="flex items-center gap-4 text-xs text-muted-foreground overflow-hidden">
             <div class="flex items-center gap-1.5 shrink-0" title="创建时间">
              <Clock class="h-3.5 w-3.5" />
              <span>创建于 {{ workflow.created_at?.split(' ')[0] || '--' }}</span>
            </div>
             <div class="flex items-center gap-1.5 shrink-0" title="上次运行">
              <CheckCircle2 class="h-3.5 w-3.5" />
              <span>{{ workflow.last_run ? workflow.last_run.split(' ')[0] : '从未运行' }}</span>
            </div>
          </div>
        </div>

        <!-- 卡片底部: 操作按钮 -->
        <div class="mt-auto px-4 sm:px-5 py-3 border-t bg-muted/20 flex items-center justify-between rounded-b-lg">
          <div class="flex items-center gap-2 relative z-10">
            <label class="relative inline-flex items-center cursor-pointer scale-90 origin-left" title="启动/停用">
              <input type="checkbox" :checked="workflow.enabled" @change="(e) => toggleWorkflow(workflow, (e.target as HTMLInputElement).checked)"
                class="sr-only peer">
              <div
                class="w-11 h-6 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary">
              </div>
            </label>
          </div>
          <div class="flex gap-0.5">
             <Button variant="ghost" size="icon" class="h-8 w-8 text-green-600 hover:text-green-700 hover:bg-green-50" @click="runWorkflow(workflow.id)" title="立即运行 (触发起始节点)">
              <Play class="h-4 w-4" />
            </Button>
             <Button variant="ghost" size="icon" class="h-8 w-8 text-primary" @click="editBoard(workflow.id)" title="绘制面板">
              <Network class="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" class="h-8 w-8" @click="openEdit(workflow)" title="编辑基础信息">
              <Pencil class="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" class="h-8 w-8 text-destructive hover:bg-destructive/10"
              @click="confirmDelete(workflow.id)" title="删除">
              <Trash2 class="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="workflows.length === 0" class="flex flex-col items-center justify-center p-8 sm:p-12 border border-dashed rounded-lg bg-muted/10">
      <Network class="h-10 w-10 text-muted-foreground/30 mb-4" />
      <p class="text-base font-medium text-foreground mb-1">暂无工作流</p>
      <p class="text-sm text-muted-foreground mb-4 text-center max-w-sm">开始创建一个任务流，将琐碎的任务编排为自动化流水线</p>
      <Button @click="openCreate">
        <Plus class="h-4 w-4 mr-2" />
        创建工作流
      </Button>
    </div>

    <Pagination v-if="total > pageSize" :page="currentPage" @update:page="handlePageChange" :total="total" />

    <!-- 创建和编辑信息的弹窗 -->
    <WorkflowDialog 
      v-model="showDialog"
      :workflow="editingWorkflow" 
      :is-edit="isEdit" 
      @saved="loadWorkflows" 
    />

    <!-- 删除确认对​​话框 -->
    <AlertDialog :open="showDeleteDialog" @update:open="showDeleteDialog = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除工作流？</AlertDialogTitle>
          <AlertDialogDescription>
            此操作无法撤销。工作流配置将被永久删除。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="deleteWorkflow">
            确认删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
