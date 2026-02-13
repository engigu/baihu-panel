<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Trash2, Package, Search, RefreshCw, Loader2, Download, FileText, RotateCw, ChevronLeft, Terminal as TerminalIcon, X } from 'lucide-vue-next'
import { api, type Dependency } from '@/api'
import TextOverflow from '@/components/TextOverflow.vue'
import XTerminal from '@/components/XTerminal.vue'
import { toast } from 'vue-sonner'

const route = useRoute()
const language = computed(() => route.query.language as string || '')
const langVersion = computed(() => route.query.version as string || '')

const activeTab = ref('python')
const deps = ref<Dependency[]>([])
const loading = ref(false)
const installing = ref(false)
const reinstalling = ref<number | null>(null)
const reinstallingAll = ref(false)
const installedLangs = ref<string[]>([])

// 安装对话框
const showInstallDialog = ref(false)
const newPkgName = ref('')
const newPkgVersion = ref('')
const newPkgRemark = ref('')

// 删除确认
const showDeleteDialog = ref(false)
const depToDelete = ref<Dependency | null>(null)

// 日志对话框
const showLogDialog = ref(false)
const logContent = ref('')
const logPkgName = ref('')

// 终端状态
const showTerminalDialog = ref(false)
const terminalCommand = ref('')
const terminalTitle = ref('依赖安装')
const isInstallSuccess = ref(false)
const pendingInstall = ref<{ name: string; version?: string; language: string; lang_version?: string; remark?: string } | null>(null)

// 搜索
const searchQuery = ref('')

const filteredDeps = computed(() => {
  const list = deps.value.filter(d => d.language === activeTab.value)
  if (!searchQuery.value) return list
  const q = searchQuery.value.toLowerCase()
  return list.filter(d => d.name.toLowerCase().includes(q))
})

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
  isInstallSuccess.value = false // 重置状态
  try {
    const { command } = await api.deps.getInstallCmd(pkgData)
    terminalCommand.value = command
    terminalTitle.value = `安装: ${pkgData.name}`
    pendingInstall.value = pkgData
    showInstallDialog.value = false
    showTerminalDialog.value = true
  } catch (e: unknown) {
    toast.error((e as Error).message || '获取安装命令失败')
  } finally {
    installing.value = false
  }
}

async function handleTerminalClose() {
  if (pendingInstall.value && isInstallSuccess.value) {
    try {
      // 终端关闭后，仅在成功时尝试在数据库中记录
      await api.deps.create(pendingInstall.value)
      toast.success('依赖记录已更新')
    } catch (e: any) {
      if (e.message !== '依赖已存在') {
        toast.error('记录依赖失败: ' + e.message)
      }
    }
  } else if (pendingInstall.value && !isInstallSuccess.value) {
    toast.error('安装未成功，记录未保存')
  }

  pendingInstall.value = null
  isInstallSuccess.value = false
  loadDeps()
}

function confirmDelete(dep: Dependency) {
  depToDelete.value = dep
  showDeleteDialog.value = true
}

async function uninstallPackage() {
  if (!depToDelete.value) return
  try {
    await api.deps.uninstall(depToDelete.value.id)
    toast.success('卸载成功')
    await loadDeps()
  } catch (e: unknown) {
    toast.error((e as Error).message || '卸载失败')
  } finally {
    showDeleteDialog.value = false
    depToDelete.value = null
  }
}

function showLog(dep: Dependency) {
  logPkgName.value = dep.name
  logContent.value = dep.log || '暂无日志'
  showLogDialog.value = true
}

async function reinstallPackage(dep: Dependency) {
  reinstalling.value = dep.id
  try {
    const { command } = await api.deps.getInstallCmd({
      name: dep.name,
      version: dep.version,
      language: dep.language,
      lang_version: dep.lang_version
    })
    terminalCommand.value = command
    terminalTitle.value = `重装: ${dep.name}`
    pendingInstall.value = null // 重新安装不需要再次记录
    showTerminalDialog.value = true
  } catch (e: unknown) {
    toast.error((e as Error).message || '获取命令失败')
  } finally {
    reinstalling.value = null
  }
}

async function reinstallAll() {
  reinstallingAll.value = true
  try {
    const lang = language.value || activeTab.value
    const ver = langVersion.value
    const { command } = await api.deps.getReinstallAllCmd(lang, ver)

    terminalCommand.value = command
    terminalTitle.value = `全部重装: ${getTypeLabel(lang)}`
    pendingInstall.value = null // 全部重装不需要记录新条目
    showTerminalDialog.value = true
  } catch (e: unknown) {
    toast.error((e as Error).message || '获取命令失败')
  } finally {
    reinstallingAll.value = false
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

watch(activeTab, loadDeps)

// 如果 URL 中带了环境参数，自动切 Tab
onMounted(async () => {
  await loadInstalledLangs()
  if (language.value) activeTab.value = language.value
  loadDeps()
})
</script>

<template>
  <div class="space-y-4">
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
      <Badge variant="outline" class="font-mono text-xs border-primary/20 text-primary/80">Scoped Environment</Badge>
    </div>

    <div class="mt-4">
      <div class="rounded-lg border bg-card overflow-x-auto">
        <!-- 工具栏 -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-4 py-3 border-b bg-muted/10">
          <div class="flex items-center gap-2">
            <Badge variant="secondary">{{ filteredDeps.length }} 个包</Badge>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative flex-1 sm:flex-none">
              <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input v-model="searchQuery" placeholder="搜索包名..." class="h-9 pl-8 w-full sm:w-48 text-sm" />
            </div>
            <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadDeps" :disabled="loading">
              <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            </Button>
            <Button variant="outline" size="sm" class="h-9 shrink-0" @click="reinstallAll"
              :disabled="reinstallingAll || filteredDeps.length === 0">
              <RotateCw class="h-4 w-4 sm:mr-1.5" :class="{ 'animate-spin': reinstallingAll }" /> <span
                class="hidden sm:inline">全部重装</span>
            </Button>
            <Button size="sm" class="h-9 shrink-0" @click="openInstallDialog">
              <Download class="h-4 w-4 sm:mr-1.5" /> <span class="hidden sm:inline">安装</span>
            </Button>
          </div>
        </div>

        <!-- 表头 -->
        <div
          class="flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium min-w-[400px]">
          <span class="flex-1">包名</span>
          <span class="w-32">版本</span>
          <span class="w-48 hidden md:block">备注</span>
          <span class="w-24 text-center">操作</span>
        </div>

        <!-- 列表 -->
        <div class="divide-y max-h-[480px] overflow-y-auto min-w-[400px]">
          <div v-if="loading" class="text-center py-8 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin mx-auto mb-2" />
            加载中...
          </div>
          <div v-else-if="filteredDeps.length === 0" class="text-center py-8 text-muted-foreground">
            <Package class="h-8 w-8 mx-auto mb-2 opacity-50" />
            {{ searchQuery ? '无匹配结果' : '暂无依赖包' }}
          </div>
          <div v-else v-for="dep in filteredDeps" :key="dep.id"
            class="flex items-center gap-4 px-4 py-2 hover:bg-muted/30 transition-colors">
            <span class="flex-1 font-mono text-sm truncate">
              <TextOverflow :text="dep.name" title="包名" />
            </span>
            <span class="w-32 text-sm text-muted-foreground">{{ dep.version || '-' }}</span>
            <span class="w-48 text-sm text-muted-foreground truncate hidden md:block">
              <TextOverflow :text="dep.remark || '-'" title="备注" />
            </span>
            <span class="w-24 flex justify-center gap-1">
              <Button v-if="dep.log" variant="ghost" size="icon" class="h-7 w-7" @click="showLog(dep)">
                <FileText class="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="reinstallPackage(dep)"
                :disabled="reinstalling === dep.id">
                <RotateCw class="h-4 w-4" :class="{ 'animate-spin': reinstalling === dep.id }" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(dep)">
                <Trash2 class="h-4 w-4" />
              </Button>
            </span>
          </div>
        </div>
      </div>

      <!-- 安装对话框 -->
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

      <!-- 卸载确认 -->
      <AlertDialog v-model:open="showDeleteDialog">
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认卸载</AlertDialogTitle>
            <AlertDialogDescription>
              确定要卸载 "{{ depToDelete?.name }}" 吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="uninstallPackage">
              卸载
            </AlertDialogAction>
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
            <pre class="text-xs bg-muted p-3 rounded-lg whitespace-pre-wrap break-all font-mono">{{ logContent }}</pre>
          </div>
          <DialogFooter>
            <Button variant="outline" @click="showLogDialog = false">关闭</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- 终端对话框 -->
      <Dialog v-model:open="showTerminalDialog" @update:open="(val) => !val && handleTerminalClose()">
        <DialogContent class="max-w-4xl h-[600px] p-0 overflow-hidden bg-[#1e1e1e] border-none shadow-2xl">
          <DialogHeader class="sr-only">
            <DialogTitle>{{ terminalTitle }}</DialogTitle>
            <DialogDescription>正在执行依赖安装指令</DialogDescription>
          </DialogHeader>
          <div class="flex flex-col h-full">
            <div class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#3c3c3c]">
              <div class="flex items-center gap-2">
                <TerminalIcon class="h-4 w-4 text-primary" />
                <span class="text-xs font-medium text-gray-300">正在执行: {{ terminalCommand }}</span>
              </div>
              <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white"
                @click="showTerminalDialog = false">
                <X class="h-4 w-4" />
              </Button>
            </div>
            <div class="flex-1">
              <XTerminal v-if="showTerminalDialog" :font-size="13" :initial-command="terminalCommand"
                @success="isInstallSuccess = true" @failed="isInstallSuccess = false" />
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  </div>
</template>
