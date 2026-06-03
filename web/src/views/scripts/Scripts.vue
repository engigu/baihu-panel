<script setup lang="ts">
import { ref, onMounted, computed, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { RefreshCw, FolderPlus, FilePlus, Save } from 'lucide-vue-next'
import { api, type FileNode } from '@/api'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import FileTreeNode from '@/components/FileTreeNode.vue'
import { toast } from 'vue-sonner'

const route = useRoute()
const router = useRouter()

const fileTree = ref<FileNode[]>([])
const expandedDirs = ref<Set<string>>(new Set())
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

// 展开文件所在的所有父目录
function expandParentDirs(filePath: string) {
  const parts = filePath.split('/')
  let current: string = ''
  for (let i = 0; i < parts.length - 1; i++) {
    current = current ? `${current}/${parts[i]}` : parts[i] ?? ''
    expandedDirs.value.add(current)
  }
}

async function loadTree() {
  loading.value = true
  try {
    fileTree.value = await api.files.tree()

    // 仅在首次加载时从 URL 恢复状态
    if (expandedDirs.value.size === 0 && selectedFile.value === null && selectedDir.value === null) {
      // 从 URL 恢复展开的目录
      const dirsParam = route.query.dirs
      if (dirsParam && typeof dirsParam === 'string') {
        dirsParam.split(',').forEach(dir => expandedDirs.value.add(dir))
      }

      // 从 URL 恢复选中的文件夹
      const dirParam = route.query.dir
      if (dirParam && typeof dirParam === 'string') {
        selectedDir.value = dirParam
        expandedDirs.value.add(dirParam)
      }

      // 从 URL 加载文件
      const fileParam = route.query.file
      if (fileParam && typeof fileParam === 'string') {
        expandParentDirs(fileParam)
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

function toggleDir(path: string) {
  if (expandedDirs.value.has(path)) {
    expandedDirs.value.delete(path)
  } else {
    expandedDirs.value.add(path)
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
    toggleDir(node.path)
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

onMounted(loadTree)
</script>

<template>
  <div class="flex flex-col lg:flex-row h-[calc(100vh-2rem)] gap-3">
    <!-- File Tree -->
    <div class="w-full lg:w-56 flex-shrink-0 border rounded-lg bg-card flex flex-col max-h-[200px] lg:max-h-none">
      <div class="p-2 border-b flex items-center justify-between">
        <span class="text-sm font-medium pl-1">脚本文件</span>
        <div class="flex gap-0.5">
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

      <div class="flex-1 overflow-auto p-1">
        <div v-if="fileTree.length === 0" class="text-xs text-muted-foreground text-center py-4">
          暂无文件
        </div>
        <FileTreeNode v-for="node in fileTree" :key="node.path" :node="node" :expanded-dirs="expandedDirs"
          :selected-path="selectedFile || selectedDir" @select="handleSelect" @delete="confirmDeleteFile"
          @download-file="handleDownload" @download-zip="handleDownloadZip" @duplicate="handleCopyFile" />
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
          <div v-if="selectedDir" class="text-xs text-muted-foreground">
            位置: {{ selectedDir }}/
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
