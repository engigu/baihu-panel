<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RefreshCw, FileUp, FileArchive, Plus, ArrowDownAZ, ArrowUpZA, Clock, AlertCircle, Search, X, SlidersHorizontal, Folder, File, Loader2 } from 'lucide-vue-next'
import { DropdownMenu, DropdownMenuContent, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import FileTreeNode from '@/components/FileTreeNode.vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { api, type FileNode } from '@/api'

const props = defineProps<{
  fileTree: FileNode[]
  expandedDirs: Set<string>
  selectedPath: string | null
  isRefreshing?: boolean
  sortMethod: 'name_asc' | 'name_desc' | 'time_desc' | 'time_asc'
}>()

const tempSearchQuery = ref('')
const searchQuery = ref('')
const isSearching = ref(false)
const searchResults = ref<FileNode[]>([])

// 搜索配置
interface SearchConfig {
  limit: number
  onlyFiles: boolean
}

const searchConfig = ref<SearchConfig>({
  limit: 100,
  onlyFiles: false
})

onMounted(() => {
  try {
    const saved = localStorage.getItem('file_search_config')
    if (saved) {
      searchConfig.value = { ...searchConfig.value, ...JSON.parse(saved) }
    }
  } catch {}
})

function updateSearchConfig(patch: Partial<SearchConfig>) {
  searchConfig.value = { ...searchConfig.value, ...patch }
  try {
    localStorage.setItem('file_search_config', JSON.stringify(searchConfig.value))
  } catch {}
  if (searchQuery.value) {
    performSearch(searchQuery.value)
  }
}

async function performSearch(query: string) {
  const q = query.trim()
  if (!q) {
    searchResults.value = []
    isSearching.value = false
    return
  }
  isSearching.value = true
  try {
    const res = await api.files.search(q, searchConfig.value.limit, searchConfig.value.onlyFiles)
    searchResults.value = res
  } catch {
    searchResults.value = []
  } finally {
    isSearching.value = false
  }
}

let debounceTimeout: number | undefined
watch(tempSearchQuery, (newVal) => {
  if (debounceTimeout) {
    clearTimeout(debounceTimeout)
  }
  if (!newVal.trim()) {
    searchQuery.value = ''
    searchResults.value = []
    isSearching.value = false
    return
  }
  debounceTimeout = window.setTimeout(() => {
    searchQuery.value = newVal
    performSearch(newVal)
  }, 250)
})

function clearSearch() {
  tempSearchQuery.value = ''
  searchQuery.value = ''
  searchResults.value = []
  isSearching.value = false
  if (debounceTimeout) {
    clearTimeout(debounceTimeout)
  }
}


const emit = defineEmits<{
  'update:sortMethod': [method: 'name_asc' | 'name_desc' | 'time_desc' | 'time_asc']
  refresh: []
  create: [path: string]
  select: [node: FileNode]
  delete: [path: string]
  download: [path: string]
  downloadZip: [path: string]
  move: [oldPath: string, newPath: string]
  rename: [path: string]
  duplicate: [path: string]
  uploadArchive: [file: File, target: string]
  uploadFiles: [files: FileList, paths: string[], target: string]
}>()

const archiveInputRef = ref<HTMLInputElement | null>(null)
const filesInputRef = ref<HTMLInputElement | null>(null)
const uploadTargetDir = ref('')

// 确认框状态
const confirmUpload = ref<{
  show: boolean
  title: string
  message: string
  onConfirm: () => void
}>({
  show: false,
  title: '',
  message: '',
  onConfirm: () => {}
})

const currentTargetDir = computed(() => {
  if (!props.selectedPath) return ''
  
  // 深度优先搜索以确定选中的是目录还是文件
  const findNode = (nodes: FileNode[], path: string): FileNode | null => {
    for (const n of nodes) {
      if (n.path === path) return n
      if (n.children) {
        const found = findNode(n.children, path)
        if (found) return found
      }
    }
    return null
  }
  
  const node = findNode(props.fileTree, props.selectedPath)
  if (node && node.isDir) {
    return node.path
  }
  
  // 如果选中的是文件，则返回它所在的目录
  const parts = props.selectedPath.split('/')
  parts.pop() // 移除文件名
  return parts.join('/')
})

function triggerArchiveUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (archiveInputRef.value) {
    archiveInputRef.value.value = ''
    archiveInputRef.value.click()
  }
}

function triggerFilesUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (filesInputRef.value) {
    filesInputRef.value.value = ''
    filesInputRef.value.click()
  }
}

function handleArchiveUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    const dirName = uploadTargetDir.value || '根目录'
    confirmUpload.value = {
      show: true,
      title: '确认导入压缩包',
      message: `即将把压缩包 [${file.name}] 导入到目录 [ ${dirName} ]。确认导入吗？`,
      onConfirm: () => {
        emit('uploadArchive', file, uploadTargetDir.value)
      }
    }
  }
}

function handleFilesUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const fileList = input.files
  if (fileList && fileList.length > 0) {
    const files: File[] = []
    const paths: string[] = []
    for (let i = 0; i < fileList.length; i++) {
      const f = fileList.item(i)
      if (f) {
        files.push(f)
        paths.push((f as any).webkitRelativePath || f.name)
      }
    }
    const dirName = uploadTargetDir.value || '根目录'
    confirmUpload.value = {
      show: true,
      title: '确认上传文件',
      message: `即将把 ${files.length} 个文件/文件夹上传到目录 [ ${dirName} ]。确认上传吗？`,
      onConfirm: () => {
        emit('uploadFiles', files as unknown as FileList, paths, uploadTargetDir.value)
      }
    }
  }
}
</script>

<template>
  <div class="w-full lg:w-56 h-48 lg:h-auto flex-shrink-0 border rounded-md flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <span class="text-xs font-medium">脚本文件</span>
      <div class="flex gap-1">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon" class="h-6 w-6" title="排序">
              <ArrowDownAZ class="h-3 w-3" v-if="sortMethod === 'name_asc'" />
              <ArrowUpZA class="h-3 w-3" v-else-if="sortMethod === 'name_desc'" />
              <Clock class="h-3 w-3" v-else-if="sortMethod === 'time_desc'" />
              <Clock class="h-3 w-3 rotate-180" v-else />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-auto min-w-[8rem]">
            <DropdownMenuRadioGroup :model-value="sortMethod" @update:model-value="v => emit('update:sortMethod', v as any)">
              <DropdownMenuRadioItem value="name_asc" class="text-xs">
                <ArrowDownAZ class="h-3.5 w-3.5 mr-2" />
                名称 (A-Z)
              </DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="name_desc" class="text-xs">
                <ArrowUpZA class="h-3.5 w-3.5 mr-2" />
                名称 (Z-A)
              </DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="time_desc" class="text-xs">
                <Clock class="h-3.5 w-3.5 mr-2" />
                修改时间 (最新)
              </DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="time_asc" class="text-xs">
                <Clock class="h-3.5 w-3.5 mr-2 rotate-180" />
                修改时间 (最旧)
              </DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('refresh')" :disabled="isRefreshing" title="刷新">
          <RefreshCw class="h-3 w-3" :class="{ 'animate-spin': isRefreshing }" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerFilesUpload(currentTargetDir)" :title="currentTargetDir ? `上传文件到: ${currentTargetDir}` : '上传文件到根目录'">
          <FileUp class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerArchiveUpload(currentTargetDir)" :title="currentTargetDir ? `导入压缩包到: ${currentTargetDir}` : '导入压缩包到根目录'">
          <FileArchive class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('create', currentTargetDir)" :title="currentTargetDir ? `在 ${currentTargetDir} 中新建` : '在根目录新建'">
          <Plus class="h-3 w-3" />
        </Button>
      </div>
      <input ref="archiveInputRef" type="file" accept=".zip,.tar,.gz,.tgz" class="hidden" @change="handleArchiveUpload" />
      <input ref="filesInputRef" type="file" multiple class="hidden" @change="handleFilesUpload" />
    </div>
    <!-- 搜索过滤输入框与配置 -->
    <div class="px-2 py-1.5 border-b bg-muted/5">
      <div class="relative flex items-center gap-1">
        <div class="relative flex-1 flex items-center">
          <Search class="absolute left-2.5 h-3 w-3 text-muted-foreground/50" />
          <Input
            v-model="tempSearchQuery"
            placeholder="搜索全盘文件..."
            class="h-7 pl-7 pr-6 w-full text-[11px] bg-background/50 border-muted-foreground/15 rounded-md focus-visible:ring-1 focus-visible:ring-ring/30"
          />
          <button
            v-if="tempSearchQuery"
            class="absolute right-2 text-muted-foreground/60 hover:text-foreground transition-colors focus:outline-none"
            @click="clearSearch"
          >
            <X class="w-3 h-3" />
          </button>
        </div>

        <!-- 搜索配置按钮与 Popover -->
        <Popover>
          <PopoverTrigger as-child>
            <Button
              variant="ghost"
              size="icon"
              class="h-7 w-7 text-muted-foreground hover:text-foreground shrink-0"
              title="搜索配置"
            >
              <SlidersHorizontal class="w-3.5 h-3.5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-60 p-3 shadow-lg rounded-xl text-xs" align="end">
            <div class="font-medium text-foreground mb-2.5 flex items-center justify-between">
              <span>搜索偏好配置</span>
              <span class="text-[10px] text-muted-foreground">实时生效</span>
            </div>

            <div class="space-y-3">
              <div>
                <label class="text-[11px] text-muted-foreground block mb-1.5">最大返回数量</label>
                <div class="grid grid-cols-4 gap-1">
                  <button
                    v-for="count in [50, 100, 200, 500]"
                    :key="count"
                    type="button"
                    @click="updateSearchConfig({ limit: count })"
                    class="py-1 px-1.5 text-[11px] rounded border text-center transition-all"
                    :class="searchConfig.limit === count
                      ? 'bg-primary text-primary-foreground border-primary font-medium'
                      : 'border-border/60 hover:bg-muted text-muted-foreground hover:text-foreground'"
                  >
                    {{ count }}
                  </button>
                </div>
              </div>

              <div class="flex items-center justify-between pt-2 border-t">
                <span class="text-[11px] text-muted-foreground">仅搜文件 (排除目录)</span>
                <Switch
                  :model-value="searchConfig.onlyFiles"
                  @update:model-value="updateSearchConfig({ onlyFiles: $event })"
                />
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>

    <!-- 列表展示区域：搜索结果 vs 目录树 -->
    <div class="flex-1 overflow-auto p-1 text-[13px]">
      <!-- 搜索模式 -->
      <template v-if="searchQuery">
        <div v-if="isSearching" class="py-8 text-center text-xs text-muted-foreground flex flex-col items-center gap-2">
          <Loader2 class="h-4 w-4 animate-spin text-primary" />
          <span>正在全盘检索...</span>
        </div>
        <div v-else-if="searchResults.length === 0" class="text-xs text-muted-foreground text-center py-8">
          未找到与 "{{ searchQuery }}" 匹配的文件
        </div>
        <div v-else class="space-y-0.5">
          <div class="px-2 py-1 text-[10px] text-muted-foreground/80 flex items-center justify-between border-b mb-1">
            <span>找到 {{ searchResults.length }} 个结果</span>
            <span v-if="searchResults.length >= searchConfig.limit" class="text-amber-500 font-medium">已达上限 {{ searchConfig.limit }}</span>
          </div>
          <div
            v-for="item in searchResults"
            :key="item.path"
            @click="emit('select', item)"
            class="flex items-center gap-2 py-1 px-2 rounded cursor-pointer text-xs hover:bg-muted group transition-colors"
            :class="selectedPath === item.path && 'bg-accent text-accent-foreground font-medium'"
          >
            <Folder v-if="item.isDir" class="h-3.5 w-3.5 text-yellow-500 shrink-0" />
            <File v-else class="h-3.5 w-3.5 text-blue-500 shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="truncate text-xs text-foreground font-medium leading-tight">{{ item.name }}</div>
              <div class="truncate text-[10px] text-muted-foreground/70 leading-tight mt-0.5">{{ item.path }}</div>
            </div>
          </div>
        </div>
      </template>

      <!-- 普通懒加载树视图 -->
      <template v-else>
        <div v-if="fileTree.length === 0" class="text-xs text-muted-foreground text-center py-4">
          暂无文件
        </div>
        <FileTreeNode
          v-for="node in fileTree"
          :key="node.path"
          :node="node"
          :expanded-dirs="expandedDirs"
          :selected-path="selectedPath"
          @select="n => emit('select', n)"
          @delete="p => emit('delete', p)"
          @create="p => emit('create', p)"
          @download-file="p => emit('download', p)"
          @download-zip="p => emit('downloadZip', p)"
          @move="(o, n) => emit('move', o, n)"
          @rename="p => emit('rename', p)"
          @duplicate="p => emit('duplicate', p)"
        />
      </template>
    </div>

    <!-- 上传确认对话框 -->
    <BaihuDialog v-model:open="confirmUpload.show" :title="confirmUpload.title" icon="Upload" size="sm">
      <div class="flex flex-col sm:flex-row items-center sm:items-start gap-4 p-1">
        <div class="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
          <AlertCircle class="h-6 w-6 text-primary" />
        </div>
        <div class="flex-1 text-center sm:text-left">
          <p class="text-sm text-foreground/90 leading-relaxed font-medium">确认操作？</p>
          <p class="text-[13px] text-muted-foreground mt-1 break-all">
            {{ confirmUpload.message }}
          </p>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col-reverse sm:flex-row gap-2 w-full sm:w-auto mt-2 sm:mt-0">
          <Button variant="outline" @click="confirmUpload.show = false" class="w-full sm:w-24">取消</Button>
          <Button @click="confirmUpload.show = false; confirmUpload.onConfirm()" class="w-full sm:w-auto px-6">立即开始</Button>
        </div>
      </template>
    </BaihuDialog>
  </div>
</template>
