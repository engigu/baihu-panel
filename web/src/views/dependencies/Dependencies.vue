<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Trash2, Package, Search, RefreshCw, Loader2, Download, FileText, RotateCw, ChevronLeft } from 'lucide-vue-next'
import { api, type Dependency } from '@/api'
import TextOverflow from '@/components/TextOverflow.vue'
import { toast } from 'vue-sonner'

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

// 安装对话框
const showInstallDialog = ref(false)
const newPkgName = ref('')
const newPkgVersion = ref('')
const newPkgRemark = ref('')

// 删除确认
const showDeleteDialog = ref(false)
const depToDelete = ref<Dependency | null>(null)

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
          <span class="w-32 text-center">操作</span>
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
            <span class="w-32 flex justify-center gap-1">
              <Button v-if="dep.log || dep.id" variant="ghost" size="icon"
                class="h-7 w-7 text-blue-500 hover:text-blue-600 hover:bg-blue-50/10" @click="showLog(dep)"
                title="查看安装日志">
                <FileText class="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-7 w-7 text-amber-500 hover:text-amber-600 hover:bg-amber-50/10" @click="reinstallPackage(dep)"
                :disabled="reinstalling === dep.id" title="重新安装">
                <RotateCw class="h-4 w-4" :class="{ 'animate-spin': reinstalling === dep.id }" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:bg-destructive/10"
                @click="confirmDelete(dep)" title="卸载并删除记录">
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
    </div>
  </div>
</template>
