<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, nextTick, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import XTerminal from '@/components/XTerminal.vue'
import { Save, Play, FilePen, TextCursorInput, Eye, X, Download, Trash2 } from 'lucide-vue-next'
import { api, type FileNode, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'
import { PATHS } from '@/constants'

import FileSidebar from './components/FileSidebar.vue'
import RunConfigDialog from './components/RunConfigDialog.vue'
import FileActionDialogs from './components/FileActionDialogs.vue'
import UnsavedConfirmDialog from './components/UnsavedConfirmDialog.vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { buildExecutionCommand } from './utils'
import { checkWindowsScriptWarnings } from '@/windows/scriptCheck'

const route = useRoute()
const router = useRouter()

// State for FileSidebar
const fileTree = ref<FileNode[]>([])
const expandedDirs = ref<Set<string>>(new Set())
const selectedPath = ref<string | null>(null)

// State for Editor
const selectedFile = ref<string | null>(null)
const fileContent = ref('')
const originalContent = ref('')
const isLoading = ref(false)
const isRefreshing = ref(false)
const isEditMode = ref(false)
const hasChanges = computed(() => fileContent.value !== originalContent.value)

const confirmLeave = ref({
  show: false,
  path: '',
  onConfirm: () => {}
})

// Component Refs
const dialogsRef = ref<InstanceType<typeof FileActionDialogs> | null>(null)
const terminalRef = ref<InstanceType<typeof XTerminal> | null>(null)

// State for RunConfig
const showRunDialog = ref(false)
const selectedEnvs = ref<{ plugin: string; version: string }[]>([])
const installedLangs = ref<MiseLanguage[]>([])

const langGroups = computed(() => {
  const groups: Record<string, string[]> = {}
  installedLangs.value.forEach(lang => {
    const plugin = lang?.plugin
    if (plugin) {
      if (!groups[plugin]) groups[plugin] = []
      groups[plugin]!.push(lang.version)
    }
  })
  return groups
})

// State for Terminal
const showTerminalDialog = ref(false)
const runCommand = ref('')
const scriptsDir = ref('')

// Windows script warning state
const showScriptWarningDialog = ref(false)
const scriptWarningMessage = ref('')
const onWarningConfirm = ref<(() => void) | null>(null)


async function fetchInstalledLangs() {
  try {
    installedLangs.value = await api.mise.list()
  } catch {
    installedLangs.value = []
  }
}

import { getLangIcon } from '@/utils/icons'

async function fetchPaths() {
  try {
    const res = await api.settings.getPaths()
    if (res?.scripts_dir) {
      scriptsDir.value = res.scripts_dir
    } else {
      scriptsDir.value = PATHS.SCRIPTS_DIR
    }
  } catch {
    scriptsDir.value = PATHS.SCRIPTS_DIR
  }
}

const editorRef = shallowRef()
const isSmallScreen = ref(window.innerWidth < 1024)
const editorFontSize = computed(() => isSmallScreen.value ? 12 : 13)

const editorOptions = computed(() => ({
  minimap: { enabled: false },
  fontSize: editorFontSize.value,
  lineNumbers: 'on' as const,
  scrollBeyondLastLine: false,
  readOnly: !isEditMode.value,
  domReadOnly: !isEditMode.value,
  automaticLayout: true,
  tabSize: 2,
  wordWrap: 'on' as const,
  folding: true,
  renderLineHighlight: 'all' as const,
}))

function handleEditorMount(editor: any) {
  editorRef.value = editor
  const model = editor.getModel()
  if (model) model.setEOL(0)
}

function handleResize() {
  isSmallScreen.value = window.innerWidth < 1024
}

// Sorting state
type SortMethod = 'name_asc' | 'name_desc' | 'time_desc' | 'time_asc'
const sortMethod = ref<SortMethod>('name_asc')

function sortTree(nodes: FileNode[]) {
  nodes.sort((a, b) => {
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
    if (node.children) sortTree(node.children)
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
  isRefreshing.value = true
  try {
    const nodes = await api.files.tree()
    sortTree(nodes)
    fileTree.value = nodes
  } catch {
    toast.error('加载文件树失败')
  } finally {
    isRefreshing.value = false
  }
}

async function handleSelect(node: FileNode) {
  selectedPath.value = node.path
  router.replace({ name: 'editor', query: { file: node.path } })

  if (node.isDir) {
    if (expandedDirs.value && expandedDirs.value.has(node.path)) {
      expandedDirs.value.delete(node.path)
    } else {
      expandedDirs.value.add(node.path)
    }
    expandedDirs.value = new Set(expandedDirs.value)
    selectedFile.value = null
  } else {
    if (hasChanges.value) {
      confirmLeave.value = {
        show: true,
        path: node.path,
        onConfirm: () => {
          originalContent.value = fileContent.value // 假装保存或标记为已确认放弃
          loadFile(node.path)
        }
      }
      return
    }
    await loadFile(node.path)
  }
}

async function loadFile(path: string) {
  isLoading.value = true
  isEditMode.value = false
  try {
    const res = await api.files.getContent(path)
    if (res) {
      selectedFile.value = path
      fileContent.value = res.content
      originalContent.value = res.content
    }
  } catch {
    toast.error('加载文件失败')
    selectedFile.value = null
  } finally {
    isLoading.value = false
  }
}

async function saveFile() {
  if (!selectedFile.value) return

  // Windows 脚本执行风险检测
  const warnings = checkWindowsScriptWarnings(selectedFile.value, fileContent.value)
  if (warnings.length > 0) {
    scriptWarningMessage.value = warnings.join('\n\n')
    showScriptWarningDialog.value = true
    onWarningConfirm.value = () => {
      showScriptWarningDialog.value = false
      executeSave()
    }
    return
  }

  await executeSave()
}

async function executeSave() {
  if (!selectedFile.value) return
  try {
    await api.files.saveContent(selectedFile.value, fileContent.value)
    originalContent.value = fileContent.value
    toast.success('保存成功')
  } catch {
    toast.error('保存失败')
  }
}

async function createItem(name: string, type: 'file' | 'dir', parent: string) {
  if (!name.trim()) { toast.error('请输入名称'); return }
  try {
    const fullPath = parent ? `${parent}/${name}` : name
    await api.files.create(fullPath, type === 'dir')
    toast.success('创建成功')
    if (dialogsRef.value) dialogsRef.value.closeCreate()
    if (parent) {
      expandedDirs.value.add(parent)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
    if (type === 'file') await loadFile(fullPath)
  } catch {
    toast.error('创建失败')
  }
}

async function deleteItem(path: string) {
  try {
    await api.files.delete(path)
    toast.success('删除成功')
    if (selectedFile.value === path) {
      selectedFile.value = null
      fileContent.value = ''
      originalContent.value = ''
    }
    if (selectedPath.value === path) selectedPath.value = null
    await loadTree()
  } catch {
    toast.error('删除失败')
  }
  if (dialogsRef.value) dialogsRef.value.closeDelete()
}

async function renameItem(oldPath: string, name: string) {
  if (!name.trim()) { toast.error('请输入名称'); return }
  if (name.includes('/')) { toast.error('不可包含 /'); return }
  const parts = oldPath.split('/')
  parts[parts.length - 1] = name
  const newPath = parts.join('/')
  if (newPath === oldPath) {
    if (dialogsRef.value) dialogsRef.value.closeRename()
    return
  }
  try {
    await handleMove(oldPath, newPath, '重命名成功', true)
    if (dialogsRef.value) dialogsRef.value.closeRename()
  } catch {}
}

async function handleMove(oldPath: string, newPath: string, msg = '移动成功', isRename = false) {
  try {
    if (isRename) await api.files.rename(oldPath, newPath)
    else await api.files.move(oldPath, newPath)
    toast.success(msg)
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
    toast.error(err.message || '操作失败')
  }
}

async function handleDownload(path: string) {
  const url = api.files.download(path)
  const a = document.createElement('a')
  a.href = url
  a.download = path.split('/').pop() || 'file'
  a.click()
  toast.success('下载中')
}

async function handleDownloadZip(path: string) {
  const url = api.files.downloadZip(path)
  const a = document.createElement('a')
  a.href = url
  a.download = (path.split('/').pop() || 'archive') + '.zip'
  a.click()
  toast.success('下载中')
}

async function handleCopyFile(path: string) {
  try {
    const parts = path.split('/')
    const filename = parts.pop() || ''
    const dir = parts.join('/')
    const dotIndex = filename.lastIndexOf('.')
    const newName = (dotIndex > 0 ? filename.substring(0, dotIndex) + '-副本' + filename.substring(dotIndex) : filename + '-副本')
    const target = dir ? `${dir}/${newName}` : newName
    await api.files.copy(path, target)
    toast.success('已复制')
    await loadTree()
  } catch (err: any) {
    toast.error('复制失败: ' + err.message)
  }
}

async function handleArchiveUpload(file: File, target: string) {
  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!['zip', 'tar', 'gz', 'tgz'].includes(ext || '')) {
    toast.error('仅支持 zip/tar/gz/tgz')
    return
  }
  try {
    await api.files.uploadArchive(file, target)
    toast.success('导入成功')
    if (target) {
      expandedDirs.value.add(target)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
  } catch (err: any) {
    toast.error(err.message || '导入失败')
  }
}

async function handleFilesUpload(files: FileList, paths: string[], target: string) {
  try {
    await api.files.uploadFiles(files, paths, target)
    toast.success('上传成功')
    if (target) {
      expandedDirs.value.add(target)
      expandedDirs.value = new Set(expandedDirs.value)
    }
    await loadTree()
  } catch (err: any) {
    toast.error(err.message || '上传失败')
  }
}

async function runScript() {
  if (!selectedFile.value) return
  const ext = selectedFile.value.split('.').pop()?.toLowerCase() || ''
  const extToLang: Record<string, string> = { 'py': 'python', 'js': 'node', 'ts': 'node', 'go': 'go' }
  const inferred = extToLang[ext]
  selectedEnvs.value = []
  if (inferred) {
    const firstV = installedLangs.value.find(l => l.plugin === inferred)?.version
    if (firstV) selectedEnvs.value.push({ plugin: inferred, version: firstV })
  }
  showRunDialog.value = true
}

async function startExecution() {
  if (!selectedFile.value) return
  
  const base = scriptsDir.value || PATHS.SCRIPTS_DIR
  runCommand.value = buildExecutionCommand(selectedFile.value, selectedEnvs.value, base)
  
  showRunDialog.value = false
  showTerminalDialog.value = true
  await nextTick()
  setTimeout(() => {
    if (terminalRef.value) {
      terminalRef.value.initTerminal(true)
    }
  }, 100)
}

function closeTerminal() {
  showTerminalDialog.value = false
  setTimeout(() => {
    if (terminalRef.value) {
      terminalRef.value.dispose()
    }
  }, 300)
}

function getLanguage(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase()
  const langMap: Record<string, string> = {
    sh: 'shell', 
    js: 'javascript', 
    ts: 'typescript', 
    py: 'python', 
    json: 'json', 
    yaml: 'yaml', 
    md: 'markdown',
    bat: 'bat',
    cmd: 'bat',
    ps1: 'powershell'
  }
  return langMap[ext || ''] || 'plaintext'
}

function expandParentDirs(path: string) {
  const parts = path.split('/')
  if (expandedDirs.value) {
    for (let i = 1; i < parts.length; i++) {
        expandedDirs.value.add(parts.slice(0, i).join('/'))
    }
    expandedDirs.value = new Set(expandedDirs.value)
  }
}

async function initFromUrl() {
  await loadTree()
  const q = route.query.file as string
  if (q) {
    selectedPath.value = q
    expandParentDirs(q)
    try {
      const res = await api.files.getContent(q)
      if (res) {
        selectedFile.value = q
        fileContent.value = res.content
        originalContent.value = res.content
      }
    } catch {}
  }
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's' && isEditMode.value && selectedFile.value) {
    e.preventDefault(); saveFile()
  }
}

onMounted(async () => {
  await initSortMethod()
  initFromUrl()
  fetchPaths()
  fetchInstalledLangs()
  window.addEventListener('resize', handleResize)
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <div class="flex-1 flex flex-col lg:flex-row h-0 lg:h-full gap-3 min-h-0 overflow-hidden">
    <FileSidebar
      :file-tree="fileTree"
      :expanded-dirs="expandedDirs"
      :selected-path="selectedPath"
      :is-refreshing="isRefreshing"
      v-model:sortMethod="sortMethod"
      @refresh="loadTree"
      @select="handleSelect"
      @delete="(path: string) => dialogsRef?.openDelete(path)"
      @create="(parent: string) => dialogsRef?.openCreate(parent)"
      @download="handleDownload"
      @download-zip="handleDownloadZip"
      @move="handleMove"
      @rename="(path: string) => dialogsRef?.openRename(path)"
      @duplicate="handleCopyFile"
      @upload-archive="handleArchiveUpload"
      @upload-files="handleFilesUpload"
    />

    <div class="flex-1 min-w-0 min-h-[300px] border rounded-md flex flex-col overflow-hidden">
      <div class="flex items-center justify-between p-2 border-b gap-2">
        <span class="text-xs font-medium truncate flex-1 min-w-0">
          {{ selectedPath || '选择文件或文件夹进行操作' }}
          <span v-if="hasChanges" class="text-orange-500 ml-1">●</span>
        </span>
        <div v-if="selectedPath" class="flex gap-1 shrink-0">
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="dialogsRef?.openRename(selectedPath)">
            <TextCursorInput class="h-3 w-3" /> <span class="hidden sm:inline">重命名</span>
          </Button>
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2 hover:bg-destructive/10 transition-colors" @click="dialogsRef?.openDelete(selectedPath)">
            <Trash2 class="h-3 w-3" /> <span class="hidden sm:inline">删除</span>
          </Button>

          <template v-if="selectedFile">
            <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="handleDownload(selectedFile)">
              <Download class="h-3 w-3" /> <span class="hidden sm:inline">下载</span>
            </Button>
            <Button v-if="!isEditMode" variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="isEditMode = true">
              <FilePen class="h-3 w-3" /> <span class="hidden sm:inline">编辑</span>
            </Button>
            <template v-else>
              <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="isEditMode = false; fileContent = originalContent">
                <Eye class="h-3 w-3" /> <span class="hidden sm:inline">查看</span>
              </Button>
              <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" :disabled="!hasChanges" @click="saveFile">
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
          theme="vs-dark" :options="editorOptions" @mount="handleEditorMount" />
        <div v-else class="h-full flex items-center justify-center text-muted-foreground text-sm">
          <span class="lg:hidden">从上方选择文件开始编辑</span>
          <span class="hidden lg:inline">从左侧选择文件开始编辑</span>
        </div>
      </div>
    </div>

    <FileActionDialogs
      ref="dialogsRef"
      :file-tree="fileTree"
      @create="createItem"
      @delete="deleteItem"
      @rename="renameItem"
    />

    <RunConfigDialog
      v-model:open="showRunDialog"
      v-model:selected-envs="selectedEnvs"
      :lang-groups="langGroups"
      :get-lang-icon="getLangIcon"
      @confirm="startExecution"
    />

    <Dialog v-model:open="showTerminalDialog">
      <DialogContent class="w-[calc(100%-1rem)] sm:max-w-[90vw] lg:max-w-4xl h-[60vh] sm:h-[80vh] flex flex-col p-0 overflow-hidden bg-[#1e1e1e] border-none shadow-2xl [&>button]:hidden">
        <div class="flex items-center justify-between px-3 py-2 border-b border-[#3c3c3c]">
          <span class="text-xs sm:text-sm font-medium text-gray-300">运行脚本</span>
          <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white" @click="closeTerminal">
            <X class="h-4 w-4" />
          </Button>
        </div>
        <div class="flex-1 overflow-hidden">
          <XTerminal v-if="showTerminalDialog" ref="terminalRef" :font-size="isSmallScreen ? 12 : 13" :initial-command="runCommand" :auto-connect="false" />
        </div>
      </DialogContent>
    </Dialog>

    <UnsavedConfirmDialog
      v-model:open="confirmLeave.show"
      :path="confirmLeave.path"
      @confirm="confirmLeave.onConfirm()"
    />

    <!-- Windows 脚本保存冲突警告 -->
    <BaihuDialog
      v-model:open="showScriptWarningDialog"
      title="Windows 脚本保存警告"
      icon="AlertCircle"
      size="sm"
    >
      <div class="flex flex-col sm:flex-row items-center sm:items-start gap-4 p-1">
        <div class="h-12 w-12 rounded-full bg-amber-500/10 flex items-center justify-center shrink-0">
          <AlertCircle class="h-6 w-6 text-amber-500" />
        </div>
        <div class="flex-1 text-center sm:text-left">
          <p class="text-sm text-foreground/90 leading-relaxed font-medium">检测到潜在的执行风险命令</p>
          <div class="text-[13px] text-muted-foreground mt-2 whitespace-pre-line leading-relaxed text-left">
            {{ scriptWarningMessage }}
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col-reverse sm:flex-row gap-2 w-full sm:w-auto mt-2 sm:mt-0">
          <Button variant="outline" @click="showScriptWarningDialog = false" class="w-full sm:w-24">取消</Button>
          <Button variant="destructive" @click="onWarningConfirm?.()" class="w-full sm:w-auto px-6">仍然保存</Button>
        </div>
      </template>
    </BaihuDialog>
  </div>
</template>
