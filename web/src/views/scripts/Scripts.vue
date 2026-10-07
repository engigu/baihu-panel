<script setup lang="ts">
import { ref, onMounted, computed, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { RefreshCw, FolderPlus, FilePlus, Save, Folder, File, Loader2, SlidersHorizontal } from 'lucide-vue-next'
import { api, type FileNode } from '@/api'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import FileTreeNode from '@/components/FileTreeNode.vue'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { DropdownMenu, DropdownMenuContent, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { ArrowDownAZ, ArrowUpZA, Clock, Search, X } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const route = useRoute()
const router = useRouter()

const fileTree = ref<FileNode[]>([])
const expandedDirs = ref<Set<string>>(new Set())
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


const selectedFile = ref<string | null>(null)
const selectedDir = ref<string | null>(null)  // 当前选中的文件夹
const fileContent = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)

const showCreateDialog = ref(false)
const createType = ref<'file' | 'folder'>('file')
const createName = ref('')
const showDeleteDialog = ref(false)
const deletePath = ref<string | null>(null)
const showUnsavedDialog = ref(false)
const pendingNode = ref<FileNode | null>(null)

// Monaco Editor 实例引用
const editorRef = shallowRef()

const hasChanges = computed(() => fileContent.value !== originalContent.value)

const editorLanguage = computed(() => {
  if (!selectedFile.value) return 'plaintext'
  const name = selectedFile.value.toLowerCase()
  if (name.endsWith('.ts')) return 'typescript'
  if (name.endsWith('.js')) return 'javascript'
  if (name.endsWith('.py')) return 'python'
  if (name.endsWith('.sh')) return 'shell'
  if (name.endsWith('.json')) return 'json'
  if (name.endsWith('.yaml') || name.endsWith('.yml')) return 'yaml'
  if (name.endsWith('.go')) return 'go'
  return 'plaintext'
})

// 编辑器挂载时的回调
function handleEditorMount(editor: any) {
  editorRef.value = editor
  // 强制设置换行符为 LF
  const model = editor.getModel()
  if (model) {
    model.setEOL(0) // 0 = LF, 1 = CRLF
  }
}

// 更新 URL - 每次清空重建
function updateUrl() {
  const query: Record<string, string> = {}
  if (selectedFile.value) query.file = selectedFile.value
  if (selectedDir.value) query.dir = selectedDir.value
  if (expandedDirs.value.size > 0) query.dirs = Array.from(expandedDirs.value).join(',')
  router.replace({ path: route.path, query })
}

// 懒加载子目录
async function loadNodeChildren(node: FileNode) {
  if (node.loading || node.loaded) return
  node.loading = true
  try {
    const children = await api.files.tree(node.path)
    sortTree(children)
    node.children = children
    node.loaded = true
  } catch (err: any) {
    toast.error(`加载目录 [${node.name}] 失败: ` + err.message)
  } finally {
    node.loading = false
  }
}

async function reloadSubdir(dirPath: string) {
  const parts = dirPath.split('/')
  let currentNodes = fileTree.value
  let targetNode: FileNode | null = null
  let currentPath = ''

  for (const part of parts) {
    currentPath = currentPath ? `${currentPath}/${part}` : part
    const found = currentNodes.find(n => n.path === currentPath)
    if (found && found.isDir) {
      if (currentPath === dirPath) {
        targetNode = found
        break
      }
      currentNodes = found.children || []
    }
  }

  if (targetNode) {
    try {
      const children = await api.files.tree(targetNode.path)
      sortTree(children)
      targetNode.children = children
      targetNode.loaded = true
    } catch {}
  }
}

// 展开文件所在的所有父目录并按需加载
async function expandParentDirs(filePath: string) {
  const parts = filePath.split('/')
  let current: string = ''
  let currentNodes = fileTree.value
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i] ?? ''
    current = current ? `${current}/${part}` : part
    expandedDirs.value.add(current)
    const dirNode = currentNodes.find(n => n.path === current)
    if (dirNode && dirNode.isDir) {
      if (!dirNode.loaded) {
        await loadNodeChildren(dirNode)
      }
      currentNodes = dirNode.children || []
    }
  }
}

// Sorting state
type SortMethod = 'name_asc' | 'name_desc' | 'time_desc' | 'time_asc'
const sortMethod = ref<SortMethod>('name_asc')

function sortTree(nodes: FileNode[]) {
  nodes.sort((a, b) => {
    // 文件夹始终排在前面
    if (a.isDir && !b.isDir) return -1
    if (!a.isDir && b.isDir) return 1
    
    switch (sortMethod.value) {
      case 'name_asc':
        return a.name.localeCompare(b.name)
      case 'name_desc':
        return b.name.localeCompare(a.name)
      case 'time_desc':
        return (b.modTime || 0) - (a.modTime || 0)
      case 'time_asc':
        return (a.modTime || 0) - (b.modTime || 0)
      default:
        return 0
    }
  })
  
  for (const node of nodes) {
    if (node.children) {
      sortTree(node.children)
    }
  }
}

watch(sortMethod, (newVal) => {
  sortTree(fileTree.value)
  api.settings.setSection('ui', { file_sort_method: newVal }).catch(() => {})
})

async function initSortMethod() {
  try {
    const val = await api.settings.get('ui', 'file_sort_method')
    if (val && ['name_asc', 'name_desc', 'time_desc', 'time_asc'].includes(val)) {
      sortMethod.value = val as SortMethod
    }
  } catch {}
}

async function loadTree() {
  loading.value = true
  try {
    const nodes = await api.files.tree()
    sortTree(nodes)
    fileTree.value = nodes

    // 若已有展开的目录，恢复它们的内容
    if (expandedDirs.value && expandedDirs.value.size > 0) {
      for (const dirPath of Array.from(expandedDirs.value)) {
        await reloadSubdir(dirPath)
      }
    }

    // 仅在首次加载时从 URL 恢复状态
    if (expandedDirs.value.size === 0 && selectedFile.value === null && selectedDir.value === null) {
      // 从 URL 恢复展开的目录
      const dirsParam = route.query.dirs
      if (dirsParam && typeof dirsParam === 'string') {
        const dirsList = dirsParam.split(',')
        for (const dir of dirsList) {
          expandedDirs.value.add(dir)
          await reloadSubdir(dir)
        }
      }

      // 从 URL 恢复选中的文件夹
      const dirParam = route.query.dir
      if (dirParam && typeof dirParam === 'string') {
        selectedDir.value = dirParam
        expandedDirs.value.add(dirParam)
        await reloadSubdir(dirParam)
      }

      // 从 URL 加载文件
      const fileParam = route.query.file
      if (fileParam && typeof fileParam === 'string') {
        await expandParentDirs(fileParam)
        await loadFileContent(fileParam)
      }
    }
  } catch {
    fileTree.value = []
  } finally {
    loading.value = false
  }
}

async function loadFileContent(path: string) {
  try {
    const res = await api.files.getContent(path)
    selectedFile.value = path
    fileContent.value = res.content
    originalContent.value = res.content
  } catch {
    selectedFile.value = null
    fileContent.value = ''
    originalContent.value = ''
  }
}

async function toggleDir(node: FileNode) {
  if (expandedDirs.value.has(node.path)) {
    expandedDirs.value.delete(node.path)
  } else {
    expandedDirs.value.add(node.path)
    if (!node.loaded && node.hasChildren !== false) {
      await loadNodeChildren(node)
    }
  }
  // 点击文件夹时不改变文件选择状态，只更新展开状态
  updateUrl()
}

async function handleSelect(node: FileNode) {
  if (node.isDir) {
    selectedDir.value = node.path
    selectedFile.value = null
    fileContent.value = ''
    originalContent.value = ''
    await toggleDir(node)
    return
  }

  if (hasChanges.value) {
    pendingNode.value = node
    showUnsavedDialog.value = true
    return
  }

  await selectFile(node)
}

async function selectFile(node: FileNode) {
  selectedDir.value = null
  loading.value = true
  try {
    await loadFileContent(node.path)
    expandParentDirs(node.path)
    updateUrl()
  } finally {
    loading.value = false
  }
}

async function confirmSwitchFile() {
  showUnsavedDialog.value = false
  if (pendingNode.value) {
    await selectFile(pendingNode.value)
    pendingNode.value = null
  }
}

async function saveFile() {
  if (!selectedFile.value) return
  saving.value = true
  try {
    await api.files.saveContent(selectedFile.value, fileContent.value)
    originalContent.value = fileContent.value
    toast.success('文件已保存')
  } catch {
    toast.error('保存失败')
  } finally {
    saving.value = false
  }
}

function openCreateDialog(type: 'file' | 'folder') {
  createType.value = type
  createName.value = ''
  showCreateDialog.value = true
}

// 计算完整路径（选中文件夹 + 文件名）
const createFullPath = computed(() => {
  if (!createName.value) return ''
  return selectedDir.value ? `${selectedDir.value}/${createName.value}` : createName.value
})

async function createItem() {
  if (!createName.value) return
  const fullPath = createFullPath.value
  const currentSelectedDir = selectedDir.value
  try {
    await api.files.create(fullPath, createType.value === 'folder')
    showCreateDialog.value = false
    toast.success(createType.value === 'file' ? '文件已创建' : '文件夹已创建')
    if (currentSelectedDir) {
      expandedDirs.value.add(currentSelectedDir)
    }
    await loadTree()
    selectedDir.value = currentSelectedDir
    if (createType.value === 'file') {
      await loadFileContent(fullPath)
      selectedDir.value = null
      updateUrl()
    }
  } catch { toast.error('创建失败') }
}

function confirmDeleteFile(path: string) {
  deletePath.value = path
  showDeleteDialog.value = true
}

async function handleDelete() {
  if (!deletePath.value) return
  const path = deletePath.value
  const currentSelectedDir = selectedDir.value
  try {
    await api.files.delete(path)
    toast.success('已删除')
    if (selectedFile.value === path) {
      selectedFile.value = null
      fileContent.value = ''
      originalContent.value = ''
      updateUrl()
    }
    if (selectedDir.value === path) {
      selectedDir.value = null
    }
    await loadTree()
    if (currentSelectedDir && currentSelectedDir !== path) {
      selectedDir.value = currentSelectedDir
    }
  } catch { toast.error('删除失败') }
  showDeleteDialog.value = false
  deletePath.value = null
}

async function handleDownload(path: string) {
  try {
    const url = api.files.download(path)
    // 使用后端直接下载接口
    const a = document.createElement('a')
    a.href = url
    a.download = path.split('/').pop() || 'file'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    toast.success('已发起下载')
  } catch (error: any) {
    toast.error('下载出错: ' + (error.message || '未知错误'))
  }
}

async function handleDownloadZip(path: string) {
  try {
    const url = api.files.downloadZip(path)
    const a = document.createElement('a')
    a.href = url
    a.download = (path.split('/').pop() || 'archive') + '.zip'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    toast.success('已发起下载')
  } catch (error: any) {
    toast.error('下载出错: ' + (error.message || '未知错误'))
  }
}

async function handleCopyFile(path: string) {
  console.log('Copy file requested:', path)
  try {
    const parts = path.split('/')
    const filename = parts.pop() || ''
    const dir = parts.join('/')
    
    const dotIndex = filename.lastIndexOf('.')
    let newFilename = ''
    if (dotIndex !== -1 && dotIndex > 0) {
      newFilename = filename.substring(0, dotIndex) + '-副本' + filename.substring(dotIndex)
    } else {
      newFilename = filename + '-副本'
    }
    
    const targetPath = dir ? `${dir}/${newFilename}` : newFilename
    
    await api.files.copy(path, targetPath)
    toast.success('已复制为 ' + newFilename)
    await loadTree()
  } catch (error: any) {
    toast.error('复制失败: ' + (error.message || '未知错误'))
  }
}

onMounted(async () => {
  await initSortMethod()
  loadTree()
})
</script>

<template>
  <div class="flex flex-col lg:flex-row h-[calc(100vh-2rem)] gap-3">
    <!-- File Tree -->
    <div class="w-full lg:w-56 flex-shrink-0 border rounded-lg bg-card flex flex-col max-h-[200px] lg:max-h-none">
      <div class="p-2 border-b flex items-center justify-between">
        <span class="text-sm font-medium pl-1">脚本文件</span>
        <div class="flex gap-0.5">
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
              <DropdownMenuRadioGroup v-model="sortMethod">
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
          <Button variant="ghost" size="icon" class="h-6 w-6" title="新建文件" @click="openCreateDialog('file')">
            <FilePlus class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" title="新建文件夹" @click="openCreateDialog('folder')">
            <FolderPlus class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" title="刷新" @click="loadTree">
            <RefreshCw class="h-3 w-3" :class="{ 'animate-spin': loading }" />
          </Button>
        </div>
      </div>
      <!-- 搜索过滤输入框与配置 -->
      <div class="px-2 py-1.5 border-b bg-muted/5">
        <div class="relative flex items-center gap-1">
          <div class="relative flex-1 flex items-center">
            <Search class="absolute left-2.5 h-3 w-3 text-muted-foreground/50" />
            <Input v-model="tempSearchQuery" placeholder="搜索全盘文件..." class="h-7 pl-7 pr-6 w-full text-[11px] placeholder:text-[11px] bg-background/50 border-muted-foreground/15 rounded-md focus-visible:ring-1 focus-visible:ring-ring/30" />
            <button v-if="tempSearchQuery" class="absolute right-2 text-muted-foreground/60 hover:text-foreground transition-colors focus:outline-none" @click="clearSearch">
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
      <div class="flex-1 overflow-auto p-1">
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
              @click="handleSelect(item)"
              class="flex items-center gap-2 py-1 px-2 rounded cursor-pointer text-xs hover:bg-muted group transition-colors"
              :class="(selectedFile === item.path || selectedDir === item.path) && 'bg-accent text-accent-foreground font-medium'"
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
            :selected-path="selectedFile || selectedDir"
            @select="handleSelect"
            @delete="confirmDeleteFile"
            @download-file="handleDownload"
            @download-zip="handleDownloadZip"
            @duplicate="handleCopyFile"
          />
        </template>
      </div>
    </div>

    <!-- Editor -->
    <div class="flex-1 border rounded-lg bg-card flex flex-col overflow-hidden min-h-[300px]">
      <div class="p-2 border-b flex items-center justify-between">
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-xs font-medium truncate">{{ selectedFile || '未选择文件' }}</span>
          <span v-if="hasChanges" class="text-xs text-orange-500 shrink-0">● 未保存</span>
        </div>
        <Button v-if="selectedFile" size="sm" class="h-6 text-xs gap-1 shrink-0" :disabled="!hasChanges || saving"
          @click="saveFile">
          <Save class="h-3 w-3" />
          <span class="hidden sm:inline">{{ saving ? '保存中...' : '保存' }}</span>
        </Button>
      </div>

      <div class="flex-1">
        <VueMonacoEditor v-if="selectedFile" v-model:value="fileContent" :language="editorLanguage" theme="vs-dark"
          :options="{
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
            wordWrap: 'on'
          }" style="height: 100%" @mount="handleEditorMount" />
        <div v-else class="h-full flex items-center justify-center text-muted-foreground text-sm">
          选择一个文件开始编辑
        </div>
      </div>
    </div>

    <!-- Create Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="max-w-xs">
        <DialogHeader>
          <DialogTitle class="text-sm">
            {{ createType === 'file' ? '新建文件' : '新建文件夹' }}
          </DialogTitle>
        </DialogHeader>
        <div class="py-2 space-y-2">
          <div class="space-y-1">
            <div class="text-xs text-muted-foreground mb-1">选择目录</div>
            <DirTreeSelect v-model="selectedDir" :file-tree="fileTree" :default-expand="selectedDir || ''" root-label="根目录" />
          </div>
          <Input v-model="createName" class="h-9 text-sm"
            :placeholder="createType === 'file' ? 'example.js' : 'folder-name'" @keyup.enter="createItem" />
          <div v-if="createName" class="text-xs text-muted-foreground">
            完整路径: {{ createFullPath }}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" class="h-8 text-xs px-4" @click="showCreateDialog = false">取消</Button>
          <Button size="sm" class="h-8 text-xs px-4" @click="createItem">创建</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 删除确认 -->
    <BaihuDialog v-model:open="showDeleteDialog" title="确认删除文件?">
      <div class="space-y-3">
        <p class="text-[15px] leading-relaxed text-muted-foreground">确定要删除以下脚本吗？此操作无法撤销。</p>
        <div class="bg-muted/30 p-3 rounded-lg border border-border/40 font-mono text-[11px] break-all text-destructive/80">
          {{ deletePath }}
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" @click="showDeleteDialog = false">取消</Button>
        <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="handleDelete">确认删除</Button>
      </template>
    </BaihuDialog>

    <!-- 未保存更改确认 -->
    <BaihuDialog v-model:open="showUnsavedDialog" title="未保存的更改">
      <div class="text-[15px] leading-relaxed text-muted-foreground">
        当前文件有未保存的更改，确定要切换文件吗？未保存的内容将会丢失。
      </div>
      <template #footer>
        <Button variant="ghost" @click="showUnsavedDialog = false">留在此页</Button>
        <Button @click="confirmSwitchFile">确定切换</Button>
      </template>
    </BaihuDialog>
  </div>
</template>
