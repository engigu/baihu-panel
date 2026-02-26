<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, nextTick, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import FileTreeNode from '@/components/FileTreeNode.vue'
import XTerminal from '@/components/XTerminal.vue'
import { Plus, Save, Play, RefreshCw, Upload, FolderUp, Pencil, Eye, X, Download, Trash2, Edit3 } from 'lucide-vue-next'
import { api, type FileNode } from '@/api'
import { toast } from 'vue-sonner'
import { PATHS, FILE_RUNNERS } from '@/constants'

const route = useRoute()
const router = useRouter()

const fileTree = ref<FileNode[]>([])
const expandedDirs = ref<Set<string>>(new Set())
const selectedFile = ref<string | null>(null)
const selectedPath = ref<string | null>(null)
const fileContent = ref('')
const originalContent = ref('')
const isLoading = ref(false)

const showCreateDialog = ref(false)
const newItemName = ref('')
const newItemType = ref<'file' | 'dir'>('file')
const createInDir = ref('')

const showDeleteDialog = ref(false)
const deleteTargetPath = ref('')

const archiveInputRef = ref<HTMLInputElement | null>(null)
const filesInputRef = ref<HTMLInputElement | null>(null)
const uploadTargetDir = ref('')

const showRenameDialog = ref(false)
const renamePath = ref('')
const newName = ref('')

const isEditMode = ref(false)
const hasChanges = computed(() => fileContent.value !== originalContent.value)

// 终端弹窗相关
const showTerminalDialog = ref(false)
const terminalRef = ref<InstanceType<typeof XTerminal> | null>(null)
const runCommand = ref('')
const scriptsDir = ref('')

async function fetchPaths() {
  try {
    const res = await api.settings.getPaths()
    scriptsDir.value = res.scripts_dir
  } catch {
    scriptsDir.value = PATHS.SCRIPTS_DIR
  }
}

// Monaco Editor 实例引用
const editorRef = shallowRef()

// 响应式字体大小
const isSmallScreen = ref(window.innerWidth < 1024)
const editorFontSize = computed(() => isSmallScreen.value ? 12 : 13)

// 编辑器挂载时的回调
function handleEditorMount(editor: any) {
  editorRef.value = editor
  // 强制设置换行符为 LF
  const model = editor.getModel()
  if (model) {
    model.setEOL(0) // 0 = LF, 1 = CRLF
  }
}

function handleResize() {
  isSmallScreen.value = window.innerWidth < 1024
}

async function loadTree() {
  try {
    fileTree.value = await api.files.tree()
  } catch {
    toast.error('加载文件树失败')
  }
}

async function handleSelect(node: FileNode) {
  selectedPath.value = node.path
  // 更新 URL 使用 query 参数
  router.replace({ name: 'editor', query: { file: node.path } })

  if (node.isDir) {
    if (expandedDirs.value.has(node.path)) {
      expandedDirs.value.delete(node.path)
    } else {
      expandedDirs.value.add(node.path)
    }
    expandedDirs.value = new Set(expandedDirs.value)
  } else {
    if (hasChanges.value && !confirm('当前文件有未保存的更改，是否放弃？')) return
    await loadFile(node.path)
  }
}

async function loadFile(path: string) {
  isLoading.value = true
  isEditMode.value = false
  try {
    const res = await api.files.getContent(path)
    selectedFile.value = path
    fileContent.value = res.content
    originalContent.value = res.content
  } catch {
    toast.error('加载文件失败')
  } finally {
    isLoading.value = false
  }
}

async function saveFile() {
  if (!selectedFile.value) return
  try {
    await api.files.saveContent(selectedFile.value, fileContent.value)
    originalContent.value = fileContent.value
    toast.success('保存成功')
  } catch {
    toast.error('保存失败')
  }
}

function openCreateDialog(parentDir = '') {
  newItemName.value = ''
  newItemType.value = 'file'
  createInDir.value = parentDir
  showCreateDialog.value = true
}

function handleCreate(parentDir: string) {
  openCreateDialog(parentDir)
}

async function createItem() {
  if (!newItemName.value.trim()) {
    toast.error('请输入名称')
    return
  }
  try {
    const fullPath = createInDir.value ? `${createInDir.value}/${newItemName.value}` : newItemName.value
    await api.files.create(fullPath, newItemType.value === 'dir')
    toast.success('创建成功')
    showCreateDialog.value = false
    // 展开父目录
    if (createInDir.value) {
      expandedDirs.value.add(createInDir.value)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
    if (newItemType.value === 'file') {
      await loadFile(fullPath)
    }
  } catch {
    toast.error('创建失败')
  }
}

function confirmDelete(path: string) {
  deleteTargetPath.value = path
  showDeleteDialog.value = true
}

async function deleteItem() {
  try {
    await api.files.delete(deleteTargetPath.value)
    toast.success('删除成功')
    if (selectedFile.value === deleteTargetPath.value) {
      selectedFile.value = null
      fileContent.value = ''
      originalContent.value = ''
    }
    await loadTree()
  } catch {
    toast.error('删除失败')
  }
  showDeleteDialog.value = false
}

function openRenameDialog(path: string) {
  renamePath.value = path
  newName.value = path.split('/').pop() || ''
  showRenameDialog.value = true
}

async function renameItem() {
  if (!newName.value.trim()) {
    toast.error('请输入名称')
    return
  }
  if (newName.value.includes('/')) {
    toast.error('名称不能包含路径分隔符 /')
    return
  }
  const parts = renamePath.value.split('/')
  parts[parts.length - 1] = newName.value
  const newPath = parts.join('/')

  if (newPath === renamePath.value) {
    showRenameDialog.value = false
    return
  }

  try {
    await handleMove(renamePath.value, newPath, '重命名成功', true)
    showRenameDialog.value = false
  } catch {
    // Error handled in handleMove
  }
}

async function runScript() {
  if (!selectedFile.value) return

  // 获取文件所在目录和文件名
  const parts = selectedFile.value.split('/')
  const fileName = parts.pop() || selectedFile.value
  const dirPath = parts.length > 0 ? parts.join('/') : ''

  // 根据文件扩展名确定运行命令
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const runner = FILE_RUNNERS[ext]
  const cmd = runner ? `${runner} ${fileName}` : `./${fileName}`

  // 构建完整命令
  const baseDir = scriptsDir.value || PATHS.SCRIPTS_DIR
  if (dirPath) {
    runCommand.value = `cd ${baseDir}/${dirPath} && ${cmd}`
  } else {
    runCommand.value = `cd ${baseDir} && ${cmd}`
  }

  showTerminalDialog.value = true
  // 等待 DOM 更新后初始化终端，增加延迟确保 Dialog 完全渲染
  await nextTick()
  setTimeout(() => {
    terminalRef.value?.initTerminal(true)
  }, 100)
}

function closeTerminal() {
  showTerminalDialog.value = false
  // 延迟清理，确保 Dialog 关闭动画完成
  setTimeout(() => {
    terminalRef.value?.dispose()
  }, 300)
}

async function handleDownload(path: string) {
  try {
    const url = api.files.download(path)
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

async function handleMove(oldPath: string, newPath: string, successMsg = '移动成功', isRename = false) {
  try {
    if (isRename) {
      await api.files.rename(oldPath, newPath)
    } else {
      await api.files.move(oldPath, newPath)
    }
    toast.success(successMsg)
    if (selectedFile.value === oldPath) {
      selectedFile.value = newPath
      selectedPath.value = newPath
      router.replace({ name: 'editor', query: { file: newPath } })
    } else if (selectedPath.value === oldPath) {
      selectedPath.value = newPath
      router.replace({ name: 'editor', query: { file: newPath } })
    }
    await loadTree()
  } catch (err: any) {
    toast.error(err.message || '移动失败')
  }
}

async function handleCopyFile(path: string) {
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

function triggerArchiveUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  archiveInputRef.value?.click()
}

function triggerFilesUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  filesInputRef.value?.click()
}

async function handleArchiveUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!['zip', 'tar', 'gz', 'tgz'].includes(ext || '')) {
    toast.error('仅支持 zip、tar、gz、tgz 格式')
    input.value = ''
    return
  }

  try {
    await api.files.uploadArchive(file, uploadTargetDir.value)
    toast.success('导入成功')
    if (uploadTargetDir.value) {
      expandedDirs.value.add(uploadTargetDir.value)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
  } catch (err: any) {
    toast.error(err.message || '导入失败')
  }
  input.value = ''
}

async function handleFilesUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  try {
    // 获取相对路径（用于保持文件夹结构）
    const paths: string[] = []
    for (let i = 0; i < files.length; i++) {
      const file = files[i] as any
      // webkitRelativePath 用于文件夹上传时保持结构
      paths.push(file.webkitRelativePath || file.name)
    }

    await api.files.uploadFiles(files, paths, uploadTargetDir.value)
    toast.success('上传成功')
    if (uploadTargetDir.value) {
      expandedDirs.value.add(uploadTargetDir.value)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
  } catch (err: any) {
    toast.error(err.message || '上传失败')
  }
  input.value = ''
}

function getLanguage(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase()
  const langMap: Record<string, string> = {
    sh: 'shell', bash: 'shell', zsh: 'shell',
    js: 'javascript', ts: 'typescript',
    py: 'python', json: 'json', yaml: 'yaml', yml: 'yaml',
    md: 'markdown', sql: 'sql', xml: 'xml', html: 'html', css: 'css'
  }
  return langMap[ext || ''] || 'plaintext'
}

// 展开路径上的所有父目录
function expandParentDirs(path: string) {
  const parts = path.split('/')
  for (let i = 1; i < parts.length; i++) {
    expandedDirs.value.add(parts.slice(0, i).join('/'))
  }
  expandedDirs.value = new Set(expandedDirs.value)
}

// 从 URL 初始化选中状态
async function initFromUrl() {
  await loadTree()
  const urlPath = route.query.file as string
  if (urlPath) {
    selectedPath.value = urlPath
    expandParentDirs(urlPath)
    // 尝试加载文件内容（如果是文件）
    try {
      const res = await api.files.getContent(urlPath)
      selectedFile.value = urlPath
      fileContent.value = res.content
      originalContent.value = res.content
      isEditMode.value = false
    } catch {
      // 可能是文件夹，忽略错误
    }
  }
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    if (isEditMode.value && selectedFile.value) {
      e.preventDefault()
      saveFile()
    }
  }
}

onMounted(() => {
  initFromUrl()
  fetchPaths()
  window.addEventListener('resize', handleResize)
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <div class="flex flex-col lg:flex-row h-[calc(100vh-100px)] gap-2">
    <!-- 文件树 -->
    <div class="w-full lg:w-56 h-48 lg:h-auto flex-shrink-0 border rounded-md flex flex-col">
      <div class="flex items-center justify-between p-2 border-b">
        <span class="text-xs font-medium">脚本文件</span>
        <div class="flex gap-1">
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="loadTree" title="刷新">
            <RefreshCw class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerFilesUpload('')" title="上传文件/文件夹(放在根目录)">
            <FolderUp class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerArchiveUpload('')" title="导入压缩包(放在根目录)">
            <Upload class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="openCreateDialog('')" title="新建">
            <Plus class="h-3 w-3" />
          </Button>
        </div>
        <input ref="archiveInputRef" type="file" accept=".zip,.tar,.gz,.tgz" class="hidden"
          @change="handleArchiveUpload" />
        <input ref="filesInputRef" type="file" multiple class="hidden" @change="handleFilesUpload" />
      </div>
      <div class="flex-1 overflow-auto p-1">
        <div v-if="fileTree.length === 0" class="text-xs text-muted-foreground text-center py-4">
          暂无文件
        </div>
        <FileTreeNode v-for="node in fileTree" :key="node.path" :node="node" :expanded-dirs="expandedDirs"
          :selected-path="selectedPath" @select="handleSelect" @delete="confirmDelete" @create="handleCreate"
          @download-file="handleDownload" @move="handleMove" @rename="openRenameDialog" @duplicate="handleCopyFile" />
      </div>
    </div>

    <!-- 编辑器 -->
    <div class="flex-1 min-h-[300px] border rounded-md flex flex-col overflow-hidden">
      <div class="flex items-center justify-between p-2 border-b gap-2">
        <span class="text-xs font-medium truncate flex-1 min-w-0">
          {{ selectedPath || '选择文件或文件夹进行操作' }}
          <span v-if="hasChanges" class="text-orange-500 ml-1">●</span>
        </span>
        <div v-if="selectedPath" class="flex gap-1 shrink-0">
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="openRenameDialog(selectedPath)">
            <Edit3 class="h-3 w-3" /> <span class="hidden sm:inline">重命名</span>
          </Button>
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="confirmDelete(selectedPath)">
            <Trash2 class="h-3 w-3 text-destructive" /> <span class="hidden sm:inline">删除</span>
          </Button>

          <template v-if="selectedFile">
            <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="handleDownload(selectedFile)">
              <Download class="h-3 w-3" /> <span class="hidden sm:inline">下载</span>
            </Button>
            <Button v-if="!isEditMode" variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2"
              @click="isEditMode = true">
              <Pencil class="h-3 w-3" /> <span class="hidden sm:inline">编辑</span>
            </Button>
            <template v-else>
              <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2"
                @click="isEditMode = false; fileContent = originalContent">
                <Eye class="h-3 w-3" /> <span class="hidden sm:inline">查看</span>
              </Button>
              <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" :disabled="!hasChanges"
                @click="saveFile">
                <Save class="h-3 w-3" /> <span class="hidden sm:inline">保存</span>
              </Button>
            </template>
            <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="runScript">
              <Play class="h-3 w-3" /> <span class="hidden sm:inline">运行</span>
            </Button>
          </template>
        </div>
      </div>
      <div class="flex-1">
        <vue-monaco-editor v-if="selectedFile" v-model:value="fileContent" :language="getLanguage(selectedFile)"
          theme="vs-dark" :options="{
            minimap: { enabled: false },
            fontSize: editorFontSize,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            quickSuggestions: isEditMode,
            suggestOnTriggerCharacters: isEditMode,
            wordBasedSuggestions: isEditMode ? 'currentDocument' : 'off',
            parameterHints: { enabled: isEditMode },
            autoClosingBrackets: 'always',
            autoClosingQuotes: 'always',
            formatOnPaste: true,
            tabSize: 4,
            insertSpaces: true,
            readOnly: !isEditMode,
            domReadOnly: !isEditMode
          }" @mount="handleEditorMount" />
        <div v-else class="h-full flex items-center justify-center text-muted-foreground text-sm">
          <span class="lg:hidden">从上方选择文件开始编辑</span>
          <span class="hidden lg:inline">从左侧选择文件开始编辑</span>
        </div>
      </div>
    </div>



    <!-- 新建对话框 -->
    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="max-w-xs" @openAutoFocus.prevent>
        <DialogHeader>
          <DialogTitle class="text-sm">新建</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-2">
          <div class="text-xs text-muted-foreground">
            位置: {{ createInDir || '根目录' }}
          </div>
          <RadioGroup v-model="newItemType" class="flex gap-4">
            <div class="flex items-center gap-2">
              <RadioGroupItem value="file" id="file" />
              <Label for="file" class="text-xs">文件</Label>
            </div>
            <div class="flex items-center gap-2">
              <RadioGroupItem value="dir" id="dir" />
              <Label for="dir" class="text-xs">文件夹</Label>
            </div>
          </RadioGroup>
          <div class="space-y-1">
            <Label class="text-xs">名称</Label>
            <Input v-model="newItemName" class="h-8 text-xs" placeholder="script.sh" @keyup.enter="createItem" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" class="h-7 text-xs" @click="showCreateDialog = false">取消</Button>
          <Button size="sm" class="h-7 text-xs" @click="createItem">创建</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 删除确认 -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="text-sm">确认删除</AlertDialogTitle>
          <AlertDialogDescription class="text-xs">确定要删除 {{ deleteTargetPath }} 吗？</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel class="h-7 text-xs">取消</AlertDialogCancel>
          <AlertDialogAction class="h-7 text-xs bg-destructive text-white hover:bg-destructive/90" @click="deleteItem">
            删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- 重命名对话框 -->
    <Dialog v-model:open="showRenameDialog">
      <DialogContent class="max-w-xs" @openAutoFocus.prevent>
        <DialogHeader>
          <DialogTitle class="text-sm">重命名</DialogTitle>
        </DialogHeader>
        <div class="space-y-3 py-2">
          <div class="space-y-1">
            <Label class="text-xs">新名称</Label>
            <Input v-model="newName" class="h-8 text-xs" placeholder="new_name.sh" @keyup.enter="renameItem" />
            <p class="text-[10px] text-muted-foreground">注：仅支持修改名称，不可包含路径分隔符 /</p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" class="h-7 text-xs" @click="showRenameDialog = false">取消</Button>
          <Button size="sm" class="h-7 text-xs" @click="renameItem">确定</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 终端弹窗 -->
    <Dialog v-model:open="showTerminalDialog">
      <DialogContent
        class="w-[calc(100%-2rem)] sm:max-w-3xl h-[60vh] sm:h-[70vh] flex flex-col p-0 overflow-hidden !bg-[#1e1e1e] border-[#3c3c3c]"
        :show-close-button="false" @openAutoFocus.prevent>
        <div class="flex items-center justify-between px-3 sm:px-4 py-2 border-b border-[#3c3c3c]">
          <span class="text-xs sm:text-sm font-medium text-gray-300">运行脚本</span>
          <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white hover:bg-white/10"
            @click="closeTerminal">
            <X class="h-4 w-4" />
          </Button>
        </div>
        <div class="flex-1 overflow-hidden">
          <XTerminal v-if="showTerminalDialog" ref="terminalRef" :font-size="isSmallScreen ? 12 : 13"
            :initial-command="runCommand" :auto-connect="false" />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>


<style scoped></style>
