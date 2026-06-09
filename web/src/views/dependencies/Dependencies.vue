<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Trash2, Package, Search, RefreshCw, Loader2, Download, FileText, RotateCw, ChevronLeft, FileUp, Terminal as TerminalIcon } from 'lucide-vue-next'
import { api, type Dependency } from '@/api'
import TextOverflow from '@/components/TextOverflow.vue'
import { Checkbox } from '@/components/ui/checkbox'
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'vue-sonner'
import XTerminal from '@/components/XTerminal.vue'

const route = useRoute()
const language = computed(() => route.query.language as string || '')
const langVersion = computed(() => route.query.version as string || '')

const activeTab = ref('python')
const deps = ref<Dependency[]>([])
const loading = ref(false)
const installing = ref(false)
const reinstalling = ref<string | null>(null)
const reinstallingAll = ref(false)
const installedLangs = ref<string[]>([])

// 批量选择状态
const selectedDeps = ref<string[]>([])

// 安装对话框
const showInstallDialog = ref(false)
const newPkgName = ref('')
const newPkgVersion = ref('')
const newPkgRemark = ref('')

// 删除确认
const showDeleteDialog = ref(false)
const isForce = ref(false)
const depToDelete = ref<Dependency | null>(null)

// 详情/日志对话框
const showLogDialog = ref(false)
const logContent = ref('')
const logPkgName = ref('')

// 搜索
const searchQuery = ref('')

const filteredDeps = computed(() => {
  const list = deps.value.filter(d => d.language === activeTab.value)
  if (!searchQuery.value) return list
  const q = searchQuery.value.toLowerCase()
  return list.filter(d => d.name.toLowerCase().includes(q))
})
// 批量选择逻辑
const isAllSelected = computed(() => {
  const list = filteredDeps.value
  if (list.length === 0) return false
  return list.every(d => selectedDeps.value.includes(d.id))
})

function handleSelectAll(checked: boolean) {
  if (checked) {
    selectedDeps.value = filteredDeps.value.map(d => d.id)
  } else {
    selectedDeps.value = []
  }
}

function toggleSelect(id: string, checked: boolean) {
  if (checked) {
    if (!selectedDeps.value.includes(id)) {
      selectedDeps.value.push(id)
    }
  } else {
    selectedDeps.value = selectedDeps.value.filter(item => item !== id)
  }
}
async function loadDeps() {
  loading.value = true
  try {
    deps.value = await api.deps.list({
      language: language.value || activeTab.value,
      lang_version: langVersion.value
    })
  } catch {
    toast.error('加载依赖列表失败')
  } finally {
    loading.value = false
  }
}

async function loadInstalledLangs() {
  try {
    const langs = await api.mise.list()
    // 获取去重后的插件名，按字母排序
    installedLangs.value = [...new Set(langs.map(l => l.plugin))].sort()

    // 如果当前 activeTab 不在已安装列表中，且不是 system，则默认选中第一个
    if (activeTab.value !== 'system' && !installedLangs.value.includes(activeTab.value)) {
      if (installedLangs.value.length > 0) {
        activeTab.value = installedLangs.value[0]!
      }
    }
  } catch {
    toast.error('获取已安装环境失败')
  }
}

function openInstallDialog() {
  newPkgName.value = ''
  newPkgVersion.value = ''
  newPkgRemark.value = ''
  showInstallDialog.value = true
}

async function installPackage() {
  if (!newPkgName.value.trim()) {
    toast.error('请输入包名')
    return
  }

  const pkgData = {
    name: newPkgName.value.trim(),
    version: newPkgVersion.value.trim() || undefined,
    remark: newPkgRemark.value.trim() || undefined,
    language: language.value || activeTab.value,
    lang_version: langVersion.value || undefined
  }

  installing.value = true
  try {
    await api.deps.install(pkgData)
    toast.success('指令已发送，详情请查看日志')
    showInstallDialog.value = false
  } catch (e: any) {
    toast.error('安装过程出错: ' + e.message)
    showInstallDialog.value = false
  } finally {
    installing.value = false
    await loadDeps()
  }
}

function confirmDelete(dep: Dependency) {
  depToDelete.value = dep
  isForce.value = false
  showDeleteDialog.value = true
}

async function uninstallPackage() {
  if (!depToDelete.value) return
  const id = depToDelete.value.id
  const name = depToDelete.value.name
  const force = isForce.value
  try {
    await api.deps.uninstall(id, force)
    toast.success(force ? `"${name}" 记录已强制移除` : '卸载成功')
    await loadDeps()
  } catch (e: any) {
    const errorMsg = e.message || '卸载失败'
    toast.error(errorMsg, {
      duration: 10000,
      action: {
        label: '强制删除记录',
        onClick: async () => {
          try {
            await api.deps.uninstall(id, true)
            toast.success(`已强制删除 "${name}" 记录`)
            await loadDeps()
          } catch (err: any) {
            toast.error('强制删除失败: ' + err.message)
          }
        }
      },
      actionButtonStyle: {
        backgroundColor: '#ef4444',
        color: 'white'
      }
    })
  } finally {
    showDeleteDialog.value = false
    depToDelete.value = null
  }
}

import { ansiToHtml } from '@/utils/ansi'

const renderedLog = computed(() => {
  return ansiToHtml(logContent.value)
})

function showLog(dep: Dependency) {
  logPkgName.value = dep.name
  logContent.value = dep.log || '暂无日志'
  showLogDialog.value = true
}

async function reinstallPackage(dep: Dependency) {
  reinstalling.value = dep.id
  try {
    await api.deps.reinstall(dep.id)
    toast.success(`重装指令已发送`)
  } catch (e: any) {
    toast.error('重装错误: ' + e.message)
  } finally {
    reinstalling.value = null
    await loadDeps()
  }
}

async function reinstallAll() {
  reinstallingAll.value = true
  try {
    const lang = language.value || activeTab.value
    const ver = langVersion.value
    await api.deps.reinstallAll(lang, ver)
    toast.success('全部重装指令执行完毕')
  } catch (e: any) {
    toast.error('全部重装错误: ' + e.message)
  } finally {
    reinstallingAll.value = false
    await loadDeps()
  }
}

// 终端与批量安装控制
const showTerminalDialog = ref(false)
const terminalCommand = ref('')
const batchInstalling = ref(false)

function openTerminalWithCommand(command: string) {
  terminalCommand.value = command
  showTerminalDialog.value = true
}

function onTerminalSuccess() {
  toast.success('安装指令执行成功')
  loadDeps()
}

function onTerminalFailed() {
  toast.error('安装指令执行失败，请检查终端输出')
  loadDeps()
}

async function startBatchInstall() {
  if (selectedDeps.value.length === 0) return

  const selectedPackages = deps.value.filter(d => selectedDeps.value.includes(d.id))
  if (selectedPackages.length === 0) return

  batchInstalling.value = true
  try {
    const reqItems = selectedPackages.map(d => ({
      name: d.name,
      version: d.version || undefined,
      language: d.language,
      lang_version: d.lang_version || undefined
    }))

    const res = await api.deps.getBatchInstallCmd({ items: reqItems })
    if (res.command) {
      selectedDeps.value = [] // 重置选择
      openTerminalWithCommand(res.command)
    }
  } catch (e: any) {
    toast.error('生成批量安装命令失败: ' + e.message)
  } finally {
    batchInstalling.value = false
  }
}

// 清单导入
const showImportDialog = ref(false)
const manifestContent = ref('')
const importDb = ref(true)
const importingManifest = ref(false)

function openImportDialog() {
  manifestContent.value = ''
  importDb.value = true
  showImportDialog.value = true
}

async function handleImportManifest() {
  if (!manifestContent.value.trim()) {
    toast.error('请输入描述清单内容')
    return
  }

  importingManifest.value = true
  try {
    const res = await api.deps.import({
      language: language.value || activeTab.value,
      lang_version: langVersion.value || undefined,
      content: manifestContent.value.trim(),
      import_db: importDb.value
    })

    toast.success('依赖清单解析完成')
    showImportDialog.value = false

    if (res.command) {
      openTerminalWithCommand(res.command)
    }
  } catch (e: any) {
    toast.error('导入描述清单失败: ' + e.message)
  } finally {
    importingManifest.value = false
    await loadDeps()
  }
}

function getTypeLabel(type: string) {
  const labels: Record<string, string> = {
    python: 'Python',
    node: 'Node.js',
    ruby: 'Ruby',
    go: 'Go',
    rust: 'Rust',
    bun: 'Bun',
    php: 'PHP',
    deno: 'Deno',
    dotnet: '.NET',
    elixir: 'Elixir',
    erlang: 'Erlang',
    lua: 'Lua',
    nim: 'Nim',
    dart: 'Dart',
    flutter: 'Flutter',
    perl: 'Perl',
    crystal: 'Crystal'
  }
  return labels[type] || type.charAt(0).toUpperCase() + type.slice(1)
}

watch(activeTab, () => {
  selectedDeps.value = []
  loadDeps()
})

// 如果 URL 中带了环境参数，自动切 Tab
onMounted(async () => {
  await loadInstalledLangs()
  if (language.value) activeTab.value = language.value
  loadDeps()
})
</script>

<template>
  <div class="space-y-4 relative pb-16">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <Button v-if="language" variant="ghost" size="icon" @click="$router.back()" class="h-8 w-8">
          <ChevronLeft class="h-5 w-5" />
        </Button>
        <div>
          <h2 class="text-xl sm:text-2xl font-bold tracking-tight">依赖管理</h2>
          <p class="text-muted-foreground text-sm">管理工具运行环境的依赖包</p>
        </div>
      </div>
    </div>

    <!-- 当前环境信息 -->
    <div v-if="language && langVersion"
      class="bg-primary/5 border border-primary/10 rounded-lg p-3 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Package class="h-4 w-4 text-primary/80" />
        <span class="text-sm">正在管理环境: <span class="font-bold font-mono">{{ language }}@{{ langVersion }}</span></span>
      </div>
      <Badge variant="outline" class="font-mono text-xs border-primary/20 text-primary/80">隔离环境</Badge>
    </div>

    <div class="mt-4">
      <div class="rounded-lg border bg-card overflow-x-auto">
        <!-- 工具栏 -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-3 sm:px-4 py-2.5 sm:py-3 border-b bg-muted/10">
          <!-- 第一行：包数和搜索框 -->
          <div class="flex items-center justify-between gap-3 w-full sm:w-auto">
            <Badge variant="secondary" class="h-7 px-2 text-[11px] sm:text-xs bg-primary/10 text-primary border-primary/20 shrink-0">{{ filteredDeps.length }} 个包</Badge>
            <div class="relative flex-1 sm:flex-none max-w-[180px] sm:w-[200px]">
              <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input v-model="searchQuery" placeholder="搜索包名..." class="h-9 pl-9 w-full text-sm bg-background focus:bg-background transition-all" />
            </div>
          </div>
          <!-- 第二行：操作按钮 -->
          <div class="flex items-center gap-1.5 sm:gap-2 justify-end w-full sm:w-auto overflow-x-auto pb-0.5 sm:pb-0 scrollbar-none">
            <Button variant="outline" size="icon" class="h-9 w-9 shrink-0 shadow-sm" @click="loadDeps" :disabled="loading">
              <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            </Button>
            <Button variant="outline" size="sm" class="h-9 px-2.5 sm:px-3 text-xs sm:text-sm shrink-0 shadow-sm" @click="reinstallAll"
              :disabled="reinstallingAll || filteredDeps.length === 0">
              <RotateCw class="h-3.5 w-3.5 mr-1" :class="{ 'animate-spin': reinstallingAll }" /> 
              <span>全部重装</span>
            </Button>
            <Button variant="outline" size="sm" class="h-9 px-2.5 sm:px-3 text-xs sm:text-sm shrink-0 shadow-sm" @click="openImportDialog"
              :disabled="activeTab !== 'python' && activeTab !== 'node'">
              <FileUp class="h-3.5 w-3.5 mr-1" /> 
              <span>导入清单</span>
            </Button>
            <Button size="sm" class="h-9 px-2.5 sm:px-3 text-xs sm:text-sm shrink-0 shadow-sm font-semibold" @click="openInstallDialog">
              <Download class="h-3.5 w-3.5 mr-1" /> 
              <span>安装包</span>
            </Button>
          </div>
        </div>

        <!-- 表头 -->
        <div
          class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 h-10 border-b bg-muted/20 text-xs text-muted-foreground font-bold uppercase tracking-wider">
          <div class="w-6 sm:w-8 shrink-0 flex items-center justify-center">
            <input
              type="checkbox"
              :checked="isAllSelected"
              @change="(e) => handleSelectAll((e.target as HTMLInputElement).checked)"
              class="appearance-none size-4 shrink-0 rounded-[5px] border border-zinc-300 dark:border-zinc-700/60 bg-zinc-50/50 dark:bg-zinc-900/30 hover:border-primary/60 checked:border-primary checked:bg-gradient-to-br checked:from-primary checked:to-indigo-600 checked:bg-[url('data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%23fff%22%20stroke-width%3D%223.5%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Cpolyline%20points%3D%2220%206%209%2017%204%2012%22%2F%3E%3C%2Fsvg%3E')] checked:bg-[size:10px_10px] checked:bg-center checked:bg-no-repeat focus:outline-none focus:ring-2 focus:ring-primary/20 checked:shadow-[0_2px_8px_rgba(99,102,241,0.35)] checked:scale-[1.05] active:scale-95 transition-all duration-200 cursor-pointer ease-out"
            />
          </div>
          <span class="w-12 hidden sm:block shrink-0 pl-1">序号</span>
          <span class="flex-1">包名</span>
          <span class="w-16 sm:w-24 shrink-0 px-1 sm:px-2">版本</span>
          <span class="w-48 hidden md:block shrink-0 px-2 font-medium">备注说明</span>
          <span class="w-20 sm:w-32 text-center shrink-0">操作</span>
        </div>

        <!-- 列表 -->
        <div class="divide-y max-h-[480px] overflow-y-auto">
          <div v-if="loading" class="text-center py-8 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin mx-auto mb-2" />
            加载中...
          </div>
          <div v-else-if="filteredDeps.length === 0" class="text-center py-8 text-muted-foreground">
            <Package class="h-8 w-8 mx-auto mb-2 opacity-50" />
            {{ searchQuery ? '无匹配结果' : '暂无依赖包' }}
          </div>
          <div v-else v-for="(dep, index) in filteredDeps" :key="dep.id"
            class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 h-10 hover:bg-muted/30 transition-colors group"
            :class="{ 'bg-primary/5': selectedDeps.includes(dep.id) }">
            <div class="w-6 sm:w-8 shrink-0 flex items-center justify-center">
              <input
                type="checkbox"
                :checked="selectedDeps.includes(dep.id)"
                @change="(e) => toggleSelect(dep.id, (e.target as HTMLInputElement).checked)"
                class="appearance-none size-4 shrink-0 rounded-[5px] border border-zinc-300 dark:border-zinc-700/60 bg-zinc-50/50 dark:bg-zinc-900/30 hover:border-primary/60 checked:border-primary checked:bg-gradient-to-br checked:from-primary checked:to-indigo-600 checked:bg-[url('data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%23fff%22%20stroke-width%3D%223.5%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Cpolyline%20points%3D%2220%206%209%2017%204%2012%22%2F%3E%3C%2Fsvg%3E')] checked:bg-[size:10px_10px] checked:bg-center checked:bg-no-repeat focus:outline-none focus:ring-2 focus:ring-primary/20 checked:shadow-[0_2px_8px_rgba(99,102,241,0.35)] checked:scale-[1.05] active:scale-95 transition-all duration-200 cursor-pointer ease-out"
              />
            </div>
            <div class="w-12 hidden sm:block shrink-0 text-[9px] text-muted-foreground tabular-nums pl-1">#{{ filteredDeps.length - index }}</div>
            <span class="flex-1 font-mono text-[12px] sm:text-[13px] truncate font-medium min-w-0">
              <TextOverflow :text="dep.name" title="包名" />
            </span>
            <span class="w-16 sm:w-24 shrink-0 px-1 sm:px-2 text-[11px] sm:text-[12px] text-muted-foreground font-mono truncate">{{ dep.version || '-' }}</span>
            <span class="w-48 text-[12px] text-muted-foreground truncate hidden md:block shrink-0 px-2">
              <TextOverflow :text="dep.remark || '-'" title="备注" />
            </span>
            <span class="w-20 sm:w-32 flex justify-center gap-0.5 sm:gap-1 shrink-0 opacity-80 group-hover:opacity-100">
              <Button v-if="dep.log || dep.id" variant="ghost" size="icon"
                class="h-6 w-6 sm:h-7 sm:w-7 text-blue-500 hover:text-blue-600 hover:bg-blue-50/10" @click="showLog(dep)"
                title="查看安装日志">
                <FileText class="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-6 w-6 sm:h-7 sm:w-7 text-amber-500 hover:text-amber-600 hover:bg-amber-50/10" @click="reinstallPackage(dep)"
                :disabled="reinstalling === dep.id" title="重新安装">
                <RotateCw class="h-3.5 w-3.5" :class="{ 'animate-spin': reinstalling === dep.id }" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7 text-destructive hover:bg-destructive/10"
                @click="confirmDelete(dep)" title="卸载并删除记录">
                <Trash2 class="h-3.5 w-3.5" />
              </Button>
            </span>
          </div>
        </div>
      </div>

      <!-- 批量操作工具栏 -->
      <transition name="fade-slide">
        <div v-if="selectedDeps.length > 1"
          class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-background/90 backdrop-blur-md border border-primary/20 shadow-xl rounded-full px-4 sm:px-6 py-2 sm:py-2.5 flex items-center justify-between gap-2.5 sm:gap-4 w-[calc(100%-2rem)] sm:w-auto max-w-md transition-all duration-300">
          <span class="text-xs sm:text-sm font-medium text-foreground whitespace-nowrap">
            已选 <span class="text-primary font-bold">{{ selectedDeps.length }}</span> 个包
          </span>
          <div class="flex items-center gap-1 sm:gap-2">
            <Button size="sm" class="rounded-full shadow-sm h-8 sm:h-9 text-xs sm:text-sm px-3" @click="startBatchInstall" :disabled="batchInstalling">
              <Loader2 v-if="batchInstalling" class="h-3.5 w-3.5 mr-1 animate-spin" />
              <Download v-else class="h-3.5 w-3.5 mr-1" />
              批量安装
            </Button>
            <Button variant="ghost" size="sm" class="rounded-full h-8 px-2 sm:px-3 text-xs text-muted-foreground hover:text-foreground" @click="selectedDeps = []">
              取消
            </Button>
          </div>
        </div>
      </transition>

      <!-- 安装单个包对话框 -->
      <Dialog v-model:open="showInstallDialog">
        <DialogContent class="sm:max-w-[400px]" @openAutoFocus.prevent>
          <DialogHeader>
            <DialogTitle>安装 {{ getTypeLabel(activeTab) }} 包</DialogTitle>
            <DialogDescription class="sr-only">输入包名和版本号进行安装</DialogDescription>
          </DialogHeader>
          <div class="grid gap-4 py-4">
            <div class="grid grid-cols-4 items-center gap-4">
              <Label class="text-right">包名</Label>
              <Input v-model="newPkgName"
                :placeholder="activeTab === 'python' ? 'requests' : (activeTab === 'node' ? 'lodash' : 'package-name')"
                class="col-span-3" />
            </div>
            <div class="grid grid-cols-4 items-center gap-4">
              <Label class="text-right">版本</Label>
              <Input v-model="newPkgVersion" placeholder="可选，如 1.0.0" class="col-span-3" />
            </div>
            <div class="grid grid-cols-4 items-center gap-4">
              <Label class="text-right">备注</Label>
              <Input v-model="newPkgRemark" placeholder="可选" class="col-span-3" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="showInstallDialog = false">取消</Button>
            <Button @click="installPackage" :disabled="installing">
              <Loader2 v-if="installing" class="h-4 w-4 mr-2 animate-spin" />
              安装
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- 导入描述清单文件对话框 -->
      <Dialog v-model:open="showImportDialog">
        <DialogContent class="sm:max-w-[500px]" @openAutoFocus.prevent>
          <DialogHeader>
            <DialogTitle>导入依赖描述清单</DialogTitle>
            <DialogDescription>
              支持粘贴 Python requirements.txt 或 Node.js package.json 的文本内容进行解析。
            </DialogDescription>
          </DialogHeader>
          <div class="grid gap-4 py-2">
            <div class="flex flex-col gap-2">
              <Label class="text-sm font-semibold">清单内容</Label>
              <Textarea v-model="manifestContent" rows="10"
                class="text-xs font-mono bg-muted/40 placeholder:font-sans placeholder:text-muted-foreground resize-none"
                placeholder="在此粘贴 requirements.txt 或 package.json 文本..." />
            </div>
            <div class="flex items-center gap-2">
              <Checkbox id="importDb" v-model:checked="importDb" />
              <Label for="importDb" class="text-xs font-medium cursor-pointer select-none">
                将解析出的包导入至面板列表中进行可视化生命周期维护
              </Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="showImportDialog = false">取消</Button>
            <Button @click="handleImportManifest" :disabled="importingManifest">
              <Loader2 v-if="importingManifest" class="h-4 w-4 mr-2 animate-spin" />
              <FileUp v-else class="h-4 w-4 mr-2" />
              解析并执行安装
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- 实时终端日志会话对话框 -->
      <Dialog v-model:open="showTerminalDialog" :closeOnOutsideClick="false">
        <DialogContent class="sm:max-w-[700px] h-[480px] flex flex-col p-4 bg-[#1e1e1e] border-zinc-800" @openAutoFocus.prevent>
          <DialogHeader class="shrink-0 border-b border-zinc-800 pb-2 mb-2">
            <DialogTitle class="flex items-center gap-2 text-zinc-100">
              <TerminalIcon class="h-5 w-5 text-primary" />
              实时依赖安装控制台
            </DialogTitle>
            <DialogDescription class="text-zinc-400">
              后台正在使用隔离环境运行安装指令，可实时查看流式输出日志。
            </DialogDescription>
          </DialogHeader>

          <!-- Xterm.js 容器 -->
          <div class="flex-1 bg-[#1e1e1e] rounded-lg overflow-hidden border border-zinc-800 relative">
            <XTerminal v-if="showTerminalDialog" :initialCommand="terminalCommand"
              @success="onTerminalSuccess" @failed="onTerminalFailed" />
          </div>

          <DialogFooter class="shrink-0 pt-2 border-t border-zinc-800 mt-2">
            <Button variant="outline" size="sm" @click="showTerminalDialog = false" class="border-zinc-800 text-zinc-300 hover:bg-zinc-800">
              关闭控制台
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- 卸载确认 -->
      <AlertDialog v-model:open="showDeleteDialog">
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认卸载</AlertDialogTitle>
            <AlertDialogDescription>
              确定要卸载 "{{ depToDelete?.name }}" 吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter class="flex-col sm:flex-row gap-3">
            <div class="flex items-center gap-2 mr-auto mb-2 sm:mb-0">
              <Checkbox id="force" v-model:checked="isForce" />
              <Label for="force" class="text-sm font-medium text-red-500 cursor-pointer select-none">
                强制删除 (卸载失败时仍移除记录)
              </Label>
            </div>
            <div class="flex justify-end gap-2">
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="uninstallPackage">
                {{ isForce ? '强制删除' : '卸载' }}
              </AlertDialogAction>
            </div>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <!-- 日志对话框 -->
      <Dialog v-model:open="showLogDialog">
        <DialogContent class="sm:max-w-[600px]" @openAutoFocus.prevent>
          <DialogHeader>
            <DialogTitle>安装日志 - {{ logPkgName }}</DialogTitle>
            <DialogDescription class="sr-only">查看依赖包的详细安装输出日志</DialogDescription>
          </DialogHeader>
          <div class="max-h-[400px] overflow-y-auto">
            <pre class="text-xs bg-muted p-3 rounded-lg whitespace-pre-wrap break-all font-mono" v-html="renderedLog"></pre>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="showLogDialog = false">关闭</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  </div>
</template>

<style scoped>
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
.fade-slide-enter-from,
.fade-slide-leave-to {
  transform: translate(-50%, 1.5rem) scale(0.95);
  opacity: 0;
}
</style>
