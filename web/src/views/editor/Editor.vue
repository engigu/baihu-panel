<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, nextTick, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import XTerminal from '@/components/XTerminal.vue'
import { Save, Play, Pencil, Eye, X, Download } from 'lucide-vue-next'
import { api, type FileNode, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'
import { PATHS, FILE_RUNNERS } from '@/constants'

// New component imports
import FileSidebar from './components/FileSidebar.vue'
import RunConfigDialog from './components/RunConfigDialog.vue'
import FileActionDialogs from './components/FileActionDialogs.vue'

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
const isEditMode = ref(false)
const hasChanges = computed(() => fileContent.value !== originalContent.value)

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

async function fetchInstalledLangs() {
  try {
    installedLangs.value = await api.mise.list()
  } catch {
    installedLangs.value = []
  }
}

function getLangIcon(plugin: string) {
  const name = plugin.toLowerCase().trim()
  const mapping: Record<string, string> = {
    'python': 'python/python-original.svg',
    'node': 'nodejs/nodejs-original.svg',
    'nodejs': 'nodejs/nodejs-original.svg',
    'go': 'go/go-original.svg',
    'rust': 'rust/rust-original.svg',
    'ruby': 'ruby/ruby-plain.svg',
    'php': 'php/php-plain.svg',
    'java': 'java/java-plain.svg',
    'deno': 'deno/deno-plain.svg',
    'bun': 'bun/bun-plain.svg',
    'zig': 'zig/zig-original.svg',
    'dotnet': 'dot-net/dot-net-original.svg',
    '.net': 'dot-net/dot-net-original.svg',
    'elixir': 'elixir/elixir-original.svg',
    'erlang': 'erlang/erlang-original.svg',
    'crystal': 'crystal/crystal-original.svg',
    'lua': 'lua/lua-original.svg',
    'julia': 'julia/julia-original.svg',
    'nim': 'nim/nim-original.svg',
    'perl': 'perl/perl-original.svg',
    'scala': 'scala/scala-original.svg',
    'kotlin': 'kotlin/kotlin-original.svg',
    'clojure': 'clojure/clojure-line.svg',
    'dart': 'dart/dart-original.svg',
    'flutter': 'flutter/flutter-original.svg',
    'terraform': 'terraform/terraform-original.svg',
    'docker': 'docker/docker-original.svg',
    'kubernetes': 'kubernetes/kubernetes-plain.svg',
    'ansible': 'ansible/ansible-original.svg',
  }
  return mapping[name] ? `https://fastly.jsdelivr.net/gh/devicons/devicon/icons/${mapping[name]}` : ''
}

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

function handleEditorMount(editor: any) {
  editorRef.value = editor
  const model = editor.getModel()
  if (model) model.setEOL(0)
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
    if (hasChanges.value && !confirm('当前文件有未保存的更改，是否放弃？')) return
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
  const parts = selectedFile.value.split('/')
  const fileName = parts.pop() || selectedFile.value
  const dirPath = parts.join('/')
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  let runner = FILE_RUNNERS[ext] || ''

  const validEnvs = selectedEnvs.value.filter(e => e.plugin && e.version)
  let cmd = ''
    if (validEnvs.length > 0) {
    const specs = validEnvs.map(e => `${e.plugin}@${e.version}`).join(' ')
    if (!runner && !['sh', 'bash'].includes(ext)) {
       const first = validEnvs[0]!.plugin
       runner = (first === 'node' ? 'node' : first)
    }
    cmd = `mise exec ${specs} -- ${runner ? `${runner} ${fileName}` : `./${fileName}`}`
  } else {
    cmd = runner ? `${runner} ${fileName}` : `./${fileName}`
  }

  const base = scriptsDir.value || PATHS.SCRIPTS_DIR
  runCommand.value = `cd ${base}${dirPath ? '/' + dirPath : ''} && ${cmd}`
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
    sh: 'shell', js: 'javascript', ts: 'typescript', py: 'python', json: 'json', yaml: 'yaml', md: 'markdown'
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

onMounted(() => {
  initFromUrl(); fetchPaths(); fetchInstalledLangs()
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
    <FileSidebar
      :file-tree="fileTree"
      :expanded-dirs="expandedDirs"
      :selected-path="selectedPath"
      @refresh="loadTree"
      @select="handleSelect"
      @delete="confirm => dialogsRef?.openDelete(confirm)"
      @create="parent => dialogsRef?.openCreate(parent)"
      @download="handleDownload"
      @move="handleMove"
      @rename="path => dialogsRef?.openRename(path)"
      @duplicate="handleCopyFile"
      @upload-archive="handleArchiveUpload"
      @upload-files="handleFilesUpload"
    />

    <div class="flex-1 min-h-[300px] border rounded-md flex flex-col overflow-hidden">
      <div class="flex items-center justify-between p-2 border-b gap-2">
        <span class="text-xs font-medium truncate flex-1 min-w-0">
          {{ selectedPath || '选择文件或文件夹进行操作' }}
          <span v-if="hasChanges" class="text-orange-500 ml-1">●</span>
        </span>
        <div v-if="selectedPath" class="flex gap-1 shrink-0">
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="dialogsRef?.openRename(selectedPath)">
            <Pencil class="h-3 w-3" /> <span class="hidden sm:inline">重命名</span>
          </Button>
          <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="dialogsRef?.openDelete(selectedPath)">
            <X class="h-3 w-3 text-destructive" /> <span class="hidden sm:inline">删除</span>
          </Button>

          <template v-if="selectedFile">
            <Button variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="handleDownload(selectedFile)">
              <Download class="h-3 w-3" /> <span class="hidden sm:inline">下载</span>
            </Button>
            <Button v-if="!isEditMode" variant="ghost" size="sm" class="h-6 text-xs gap-1 px-2" @click="isEditMode = true">
              <Pencil class="h-3 w-3" /> <span class="hidden sm:inline">编辑</span>
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
          theme="vs-dark" :options="{
            minimap: { enabled: false },
            fontSize: editorFontSize,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            readOnly: !isEditMode,
            domReadOnly: !isEditMode
          }" @mount="handleEditorMount" />
        <div v-else class="h-full flex items-center justify-center text-muted-foreground text-sm">
          <span class="lg:hidden">从上方选择文件开始编辑</span>
          <span class="hidden lg:inline">从左侧选择文件开始编辑</span>
        </div>
      </div>
    </div>

    <FileActionDialogs
      ref="dialogsRef"
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
  </div>
</template>
