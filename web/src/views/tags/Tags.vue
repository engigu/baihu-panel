<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import { Plus, Pencil, Trash2, Search, Tag, X, RefreshCw, Layers, Terminal, Variable } from 'lucide-vue-next'
import { api, type TagItem } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { format } from 'date-fns'

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy-MM-dd HH:mm:ss')
  } catch {
    return dateStr
  }
}

const { pageSize } = useSiteSettings()

const tags = ref<TagItem[]>([])
const loading = ref(false)
const filterName = ref('')
const currentPage = ref(1)
const total = ref(0)
const activeTab = ref<string>('all')

let searchTimer: ReturnType<typeof setTimeout> | null = null

// Dialog state
const showEditDialog = ref(false)
const isEditMode = ref(false)
const editingTag = ref<Partial<TagItem>>({})
const tagForm = ref({
  name: '',
  type: 'task_tag' as 'task_tag' | 'env_tag'
})

async function loadTags() {
  loading.value = true
  try {
    const res = await api.tags.list({
      page: currentPage.value,
      page_size: pageSize.value,
      name: filterName.value || undefined,
      type: activeTab.value === 'all' ? undefined : activeTab.value
    })
    tags.value = res.data
    total.value = res.total
  } catch (err: any) {
    toast.error('加载标签失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadTags()
  }, 300)
}

watch(activeTab, () => {
  currentPage.value = 1
  loadTags()
})

function handlePageChange(page: number) {
  currentPage.value = page
  loadTags()
}

// Dialog actions
function openCreateDialog() {
  isEditMode.value = false
  tagForm.value = {
    name: '',
    type: 'task_tag'
  }
  showEditDialog.value = true
}

function openEditDialog(tag: TagItem) {
  isEditMode.value = true
  editingTag.value = tag
  tagForm.value = {
    name: tag.name,
    type: tag.type
  }
  showEditDialog.value = true
}

async function saveTag() {
  const name = tagForm.value.name.trim()
  if (!name) {
    toast.error('请输入标签名称')
    return
  }

  try {
    if (isEditMode.value && editingTag.value.id) {
      await api.tags.update(editingTag.value.id, { name })
      toast.success('更新成功')
    } else {
      await api.tags.create({ name, type: tagForm.value.type })
      toast.success('创建成功')
    }
    showEditDialog.value = false
    loadTags()
  } catch (err: any) {
    toast.error('保存失败: ' + err.message)
  }
}

// Delete action
const showDeleteConfirm = ref(false)
const tagToDelete = ref<TagItem | null>(null)

function confirmDelete(tag: TagItem) {
  tagToDelete.value = tag
  showDeleteConfirm.value = true
}

async function deleteTag() {
  if (!tagToDelete.value) return
  try {
    await api.tags.delete(tagToDelete.value.id)
    toast.success('删除成功')
    showDeleteConfirm.value = false
    tagToDelete.value = null
    loadTags()
  } catch (err: any) {
    toast.error('删除失败: ' + err.message)
  }
}

onMounted(() => {
  loadTags()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex flex-col xl:flex-row xl:items-center gap-3 xl:gap-6 pb-2 border-b border-border/40">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <h1 class="text-xl font-bold tracking-tight text-foreground select-none">
            标签管理
          </h1>
        </div>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">统一查看、重命名或清理系统中的任务与环境变量标签。</p>
      </div>

      <!-- Controls Toolbar -->
      <div class="flex flex-row items-center flex-wrap gap-2 w-full xl:w-auto xl:ml-auto xl:justify-end">
        <!-- Search Input -->
        <div class="flex flex-row items-center gap-2 w-full sm:flex-1 xl:flex-none xl:w-auto text-sm">
          <div class="relative flex-1 xl:flex-none xl:w-[240px] group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input
              v-model="filterName"
              placeholder="搜索标签..."
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch"
            />
            <button
              v-if="filterName"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              @click="filterName = ''; handleSearch()"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>

        <div class="flex items-center gap-2 w-full sm:w-auto sm:justify-end">
          <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadTags" :disabled="loading" title="刷新">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
          </Button>

          <Button @click="openCreateDialog" class="flex-1 sm:flex-none px-2 xl:px-3 h-9 shadow-sm font-medium gap-1.5" title="新建标签">
            <Plus class="h-4 w-4" />
            <span>新建标签</span>
          </Button>

          <!-- Type Select Dropdown -->
          <Select v-model="activeTab">
            <SelectTrigger class="h-9 flex-1 sm:flex-none sm:w-[120px] text-xs bg-muted/20 border-muted-foreground/10 focus:bg-background">
              <SelectValue placeholder="全部类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部类型</SelectItem>
              <SelectItem value="task_tag">任务标签</SelectItem>
              <SelectItem value="env_tag">环境变量</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>

    <!-- Table Container -->
    <div class="rounded-lg border bg-card overflow-hidden">
      
      <!-- ========== 1. 大/中屏布局 (Large/Medium Screen >= 640px) ========== -->
      <div class="hidden sm:block">
        <!-- Table Header -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-64 shrink-0 pl-1">标签名称</span>
          <span class="w-32 shrink-0">类型</span>
          <span class="w-24 shrink-0 text-center">关联资源数</span>
          <span class="flex-1 min-w-0">创建时间</span>
          <span class="w-24 shrink-0 text-right pr-2">操作</span>
        </div>

        <!-- Rows -->
        <div class="divide-y text-sm">
          <div v-if="loading" class="p-8 text-center text-muted-foreground flex items-center justify-center gap-2">
            <RefreshCw class="h-4 w-4 animate-spin text-primary" />
            <span>加载中...</span>
          </div>
          <div v-else-if="tags.length === 0" class="p-8 text-center text-muted-foreground">
            没有找到匹配的标签
          </div>
          <div
            v-else
            v-for="(tag, idx) in tags"
            :key="tag.id"
            class="flex items-center gap-4 px-4 py-2 hover:bg-muted/30 transition-colors"
          >
            <!-- Serial Number -->
            <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-[11px]">
              #{{ total - (currentPage - 1) * pageSize - idx }}
            </div>
            
            <!-- Tag Name -->
            <div class="w-64 shrink-0 font-medium flex items-center gap-2 overflow-hidden">
              <Tag class="h-3.5 w-3.5 text-primary/75 shrink-0" />
              <span class="truncate cursor-help" :title="tag.name">{{ tag.name }}</span>
            </div>

            <!-- Type -->
            <div class="w-32 shrink-0 flex items-center pl-4">
              <component
                :is="tag.type === 'task_tag' ? Terminal : Variable"
                class="h-4 w-4 shrink-0"
                :class="tag.type === 'task_tag' ? 'text-primary' : 'text-amber-500 dark:text-amber-400'"
                :title="tag.type === 'task_tag' ? '任务标签' : '环境变量'"
              />
            </div>

            <!-- Association Count -->
            <div class="w-24 shrink-0 text-center">
              <span
                class="font-mono text-xs font-semibold px-2 py-0.5 rounded-full"
                :class="tag.association_count > 0 ? 'bg-primary/10 text-primary border border-primary/20' : 'bg-muted text-muted-foreground'"
              >
                {{ tag.association_count }}
              </span>
            </div>

            <!-- Created At -->
            <div class="flex-1 min-w-0 text-xs text-muted-foreground truncate">
              {{ formatDate(tag.created_at) }}
            </div>

            <!-- Actions -->
            <div class="w-24 shrink-0 flex justify-end gap-1">
              <Button variant="ghost" size="icon" class="h-7 w-7 text-muted-foreground hover:text-foreground" @click="openEditDialog(tag)">
                <Pencil class="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:bg-destructive/10" @click="confirmDelete(tag)">
                <Trash2 class="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 2. 小屏布局 (Small Screen < 640px) ========== -->
      <div class="divide-y sm:hidden">
        <div v-if="loading" class="p-8 text-center text-muted-foreground flex items-center justify-center gap-2">
          <RefreshCw class="h-4 w-4 animate-spin text-primary" />
          <span>加载中...</span>
        </div>
        <div v-else-if="tags.length === 0" class="p-8 text-center text-muted-foreground">
          暂无标签
        </div>
        <div v-else v-for="(tag, idx) in tags" :key="`small-${tag.id}`" class="p-3.5 hover:bg-muted/50 transition-colors">
          <div class="flex items-start justify-between mb-2 pb-2 border-b border-border/40">
            <div class="flex items-center gap-2 flex-1 min-w-0 pr-2">
              <span class="text-[10px] text-muted-foreground tabular-nums shrink-0">
                #{{ total - (currentPage - 1) * pageSize - idx }}
              </span>
              <Tag class="h-3.5 w-3.5 text-primary/75 shrink-0" />
              <span class="font-bold text-sm truncate">{{ tag.name }}</span>
            </div>
            <div class="flex items-center">
              <component
                :is="tag.type === 'task_tag' ? Terminal : Variable"
                class="h-3.5 w-3.5 shrink-0"
                :class="tag.type === 'task_tag' ? 'text-primary' : 'text-amber-500 dark:text-amber-400'"
                :title="tag.type === 'task_tag' ? '任务标签' : '环境变量'"
              />
            </div>
          </div>

          <div class="flex items-center justify-between text-xs text-muted-foreground mt-1">
            <div class="flex items-center gap-1">
              <span>关联资源:</span>
              <span
                class="font-mono font-bold px-1.5 py-0.2 rounded bg-primary/10 text-primary"
                :class="{ 'bg-muted text-muted-foreground': tag.association_count === 0 }"
              >
                {{ tag.association_count }}
              </span>
            </div>
            <div>{{ formatDate(tag.created_at) }}</div>
          </div>

          <div class="grid grid-cols-2 items-center pt-2 mt-3.5 border-t border-border/40 -mx-3.5 -mb-3.5">
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none" @click="openEditDialog(tag)">
              <Pencil class="h-3.5 w-3.5" />
              <span>编辑</span>
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-destructive/5 text-destructive rounded-none border-l border-border/10" @click="confirmDelete(tag)">
              <Trash2 class="h-3.5 w-3.5" />
              <span>删除</span>
            </Button>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <Pagination
        :page="currentPage"
        :total="total"
        @update:page="handlePageChange"
      />
    </div>

    <!-- Create/Edit Dialog -->
    <Dialog v-model:open="showEditDialog">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Layers class="h-5 w-5 text-primary" />
            <span>{{ isEditMode ? '编辑标签' : '新建标签' }}</span>
          </DialogTitle>
          <DialogDescription class="text-xs">
            {{ isEditMode ? '修改选定标签的显示名称，修改后将同步应用到所有关联资源。' : '手动新建一个全新标签，之后可在定时任务或环境变量中直接选用。' }}
          </DialogDescription>
        </DialogHeader>

        <div class="grid gap-4 py-3">
          <div class="flex flex-col gap-2">
            <label for="tag-name" class="text-xs font-semibold text-foreground/80">标签名称</label>
            <Input id="tag-name" v-model="tagForm.name" placeholder="请输入标签名称" class="h-9" @keydown.enter="saveTag" />
          </div>
          <div v-if="!isEditMode" class="flex flex-col gap-2">
            <label class="text-xs font-semibold text-foreground/80">标签类型</label>
            <div class="flex gap-6 mt-1">
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="radio" v-model="tagForm.type" value="task_tag" class="accent-primary h-3.5 w-3.5" />
                <span class="text-sm font-medium">任务标签</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="radio" v-model="tagForm.type" value="env_tag" class="accent-primary h-3.5 w-3.5" />
                <span class="text-sm font-medium">环境变量标签</span>
              </label>
            </div>
          </div>
        </div>

        <DialogFooter class="gap-2 sm:gap-0">
          <Button variant="outline" size="sm" @click="showEditDialog = false" class="h-9">取消</Button>
          <Button @click="saveTag" size="sm" class="h-9">确定</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete Confirm Dialog -->
    <Dialog v-model:open="showDeleteConfirm">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle class="text-destructive flex items-center gap-2">
            <Trash2 class="h-5 w-5" />
            <span>确认删除标签</span>
          </DialogTitle>
          <DialogDescription class="text-xs pt-1">
            确认要删除标签 <strong class="text-foreground">"{{ tagToDelete?.name }}"</strong> 吗？
            <br />
            <span class="text-destructive/90 font-semibold block mt-2 p-2 bg-destructive/5 rounded-md border border-destructive/10">
              提示：此操作会物理删除该标签，并自动解除所有 {{ tagToDelete?.association_count }} 个相关资源的绑定关联。
            </span>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-2 sm:gap-0">
          <Button variant="outline" size="sm" @click="showDeleteConfirm = false" class="h-9">取消</Button>
          <Button variant="destructive" size="sm" @click="deleteTag" class="h-9">确认删除</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
