<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import Pagination from '@/components/Pagination.vue'
import { Plus, Pencil, Trash2, Eye, EyeOff, Search } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import { api, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { Switch } from '@/components/ui/switch'

const { pageSize } = useSiteSettings()

const envVars = ref<EnvVar[]>([])
const showDialog = ref(false)
const editingEnv = ref<Partial<EnvVar>>({})
const isEdit = ref(false)
const showValues = ref<Record<number, boolean>>({})
const showDeleteDialog = ref(false)
const deleteEnvId = ref<number | null>(null)

const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

async function loadEnvVars() {
  try {
    const res = await api.env.list({ page: currentPage.value, page_size: pageSize.value, name: filterName.value || undefined })
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

function handlePageChange(page: number) {
  currentPage.value = page
  loadEnvVars()
}

function openCreate() {
  editingEnv.value = { name: '', value: '', remark: '', hidden: true }
  isEdit.value = false
  showDialog.value = true
}

function openEdit(env: EnvVar) {
  editingEnv.value = { ...env }
  isEdit.value = true
  showDialog.value = true
}

async function saveEnv() {
  try {
    if (isEdit.value && editingEnv.value.id) {
      await api.env.update(editingEnv.value.id, editingEnv.value)
      toast.success('变量已更新')
    } else {
      await api.env.create(editingEnv.value)
      toast.success('变量已创建')
    }
    showDialog.value = false
    loadEnvVars()
  } catch { toast.error('保存失败') }
}

function confirmDelete(id: number) {
  deleteEnvId.value = id
  showDeleteDialog.value = true
}

async function deleteEnv() {
  if (!deleteEnvId.value) return
  try {
    await api.env.delete(deleteEnvId.value)
    toast.success('变量已删除')
    loadEnvVars()
  } catch { toast.error('删除失败') }
  showDeleteDialog.value = false
  deleteEnvId.value = null
}

function toggleShow(id: number) {
  showValues.value[id] = !showValues.value[id]
}

function maskValue(value: string) {
  return '•'.repeat(Math.min(value.length, 20))
}

onMounted(loadEnvVars)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">环境变量</h2>
        <p class="text-muted-foreground text-sm">管理脚本执行时的环境变量</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterName" placeholder="搜索变量..." class="h-9 pl-9 w-full sm:w-56 text-sm"
            @input="handleSearch" />
        </div>
        <Button @click="openCreate" class="shrink-0">
          <Plus class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">新建变量</span>
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <!-- 表头 -->
      <div
        class="flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium min-w-[500px]">
        <span class="w-32 sm:w-48 shrink-0">变量名</span>
        <span class="w-24 sm:flex-1 shrink-0 sm:shrink">值</span>
        <span class="w-32 sm:w-48 shrink-0 hidden md:block">备注</span>
        <span class="w-20 sm:w-24 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y min-w-[500px]">
        <div v-if="envVars.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无环境变量
        </div>
        <div v-for="env in envVars" :key="env.id"
          class="flex items-center gap-4 px-4 py-2 hover:bg-muted/30 transition-colors">
          <code
            class="w-32 sm:w-48 font-medium truncate shrink-0 text-xs bg-muted/40 px-2 py-1 rounded">{{ env.name }}</code>
          <span class="w-24 sm:flex-1 shrink-0 sm:shrink font-mono text-muted-foreground truncate text-xs">
            <TextOverflow :text="showValues[env.id] ? env.value : maskValue(env.value)" title="变量值" />
          </span>
          <span class="w-32 sm:w-48 shrink-0 text-muted-foreground truncate text-sm hidden md:block">
            <TextOverflow :text="env.remark || '-'" title="备注" />
          </span>
          <span class="w-20 sm:w-24 shrink-0 flex justify-center gap-1">
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
      <DialogContent class="max-w-md" @openAutoFocus.prevent>
        <DialogHeader>
          <DialogTitle>{{ isEdit ? '编辑变量' : '新建变量' }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-2">
            <Label>变量名</Label>
            <Input v-model="editingEnv.name" class="font-mono" placeholder="MY_VAR" />
          </div>
          <div class="space-y-2">
            <Label>变量值</Label>
            <Input v-model="editingEnv.value" class="font-mono" placeholder="value" />
          </div>
          <div class="space-y-2">
            <Label>备注</Label>
            <Textarea v-model="editingEnv.remark" class="resize-none" rows="3" placeholder="变量说明..." />
          </div>
          <div class="flex items-center justify-between space-x-2 pt-2">
            <Label class="text-sm font-medium">隐藏变量值</Label>
            <Switch v-model="editingEnv.hidden" />
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
          <AlertDialogDescription>确定要删除此环境变量吗？此操作无法撤销。</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteEnv">删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
