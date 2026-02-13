<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
    Plus, Globe, Search, RefreshCw, Loader2, Trash2,
    Terminal as TerminalIcon, X, AlertCircle,
    Check, ChevronsUpDown
} from 'lucide-vue-next'
import { api, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'
import XTerminal from '@/components/XTerminal.vue'
import { cn } from '@/lib/utils'

const SUPPORTED_DEPS_LANGS = [
    'python', 'node', 'ruby', 'go', 'rust', 'bun', 'php',
    'deno', 'dotnet', 'elixir', 'erlang', 'lua', 'nim',
    'dart', 'flutter', 'perl', 'crystal'
]

interface DisplayLanguage extends Omit<MiseLanguage, 'source'> {
    source: string
}

const languages = ref<DisplayLanguage[]>([])
const loading = ref(false)
const searchQuery = ref('')
const errorMsg = ref('')
const syncing = ref(false)
const showSyncConfirm = ref(false)

const showInstallDialog = ref(false)
const newLangPlugin = ref('')
const newLangVersion = ref('')

// 下拉列表相关
const availablePlugins = ref<string[]>([])
const loadingPlugins = ref(false)
const pluginSearch = ref('')
const openPluginPopover = ref(false)

const availableVersions = ref<string[]>([])
const loadingVersions = ref(false)
const versionSearch = ref('')
const openVersionPopover = ref(false)

const filteredPlugins = computed(() => {
    if (!pluginSearch.value) return availablePlugins.value
    const s = pluginSearch.value.toLowerCase()
    return availablePlugins.value.filter(p => p.toLowerCase().includes(s))
})

const filteredVersions = computed(() => {
    const list = availableVersions.value
    if (!versionSearch.value) return list
    const s = versionSearch.value.toLowerCase()
    return list.filter(v => v.toLowerCase().includes(s))
})

const showTerminalDialog = ref(false)
const terminalCommand = ref('')

const filteredLanguages = computed(() => {
    if (!searchQuery.value) return languages.value
    const q = searchQuery.value.toLowerCase()
    return languages.value.filter(l => l.plugin.toLowerCase().includes(q) || l.version.toLowerCase().includes(q))
})

async function loadLanguages() {
    loading.value = true
    errorMsg.value = ''
    try {
        const data = await api.mise.list()
        if (!data || !Array.isArray(data)) {
            languages.value = []
            return
        }
        languages.value = data.map(item => ({
            ...item,
            source: typeof item.source === 'object' ? (item.source.path || item.source.type || '-') : (item.source || '-')
        }))
    } catch (e) {
        toast.error('获取语言列表失败')
        errorMsg.value = String(e)
    } finally {
        loading.value = false
    }
}

async function handleSync() {
    syncing.value = true
    try {
        await api.mise.sync()
        toast.success('本地环境同步成功')
        await loadLanguages()
    } catch (e) {
        toast.error('同步失败: ' + e)
    } finally {
        syncing.value = false
        showSyncConfirm.value = false
    }
}

async function fetchPlugins() {
    if (availablePlugins.value.length > 0) return
    loadingPlugins.value = true
    try {
        availablePlugins.value = await api.mise.plugins()
    } catch (e) {
        console.error('Fetch plugins failed', e)
    } finally {
        loadingPlugins.value = false
    }
}

async function fetchVersions(plugin: string) {
    if (!plugin) return
    loadingVersions.value = true
    availableVersions.value = []
    try {
        availableVersions.value = await api.mise.versions(plugin)
    } catch (e) {
        console.error('Fetch versions failed', e)
    } finally {
        loadingVersions.value = false
        // 如果当前版本为空且已经有列表，默认选择第一个（通常是最新版本）
        if (!newLangVersion.value && availableVersions.value.length > 0) {
            newLangVersion.value = availableVersions.value[0] || ''
        }
    }
}

watch(newLangPlugin, (newVal) => {
    if (newVal) {
        fetchVersions(newVal)
    } else {
        availableVersions.value = []
    }
    newLangVersion.value = ''
})

function openInstallDialog() {
    newLangPlugin.value = ''
    newLangVersion.value = ''
    showInstallDialog.value = true
    fetchPlugins()
}

function startInstall() {
    if (!newLangPlugin.value.trim()) {
        toast.error('请输入或选择语言名称')
        return
    }
    if (!newLangVersion.value.trim()) {
        toast.error('请选择版本')
        return
    }
    const version = newLangVersion.value.trim()
    const cmd = `mise install ${newLangPlugin.value.trim()}@${version}`

    showInstallDialog.value = false
    runInTerminal(cmd)
}

function runInTerminal(command: string) {
    terminalCommand.value = command
    showTerminalDialog.value = true
}

function confirmDelete(lang: MiseLanguage) {
    const cmd = `mise uninstall ${lang.plugin}@${lang.version}`
    runInTerminal(cmd)
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

    if (mapping[name]) {
        return `https://cdn.jsdelivr.net/gh/devicons/devicon/icons/${mapping[name]}`
    }
    return ''
}

async function handleVerify(lang: MiseLanguage) {
    try {
        const { command } = await api.mise.verifyCommand(lang.plugin, lang.version)
        runInTerminal(command)
    } catch (e) {
        toast.error('获取验证命令失败')
    }
}

onMounted(loadLanguages)
</script>

<template>
    <div class="space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
                <h2 class="text-xl sm:text-2xl font-bold tracking-tight">语言依赖</h2>
                <p class="text-muted-foreground text-sm">管理系统环境中的编程语言运行时及相关包依赖 (Mise)</p>
            </div>
            <Button @click="openInstallDialog">
                <Plus class="h-4 w-4 mr-2" /> 新增语言
            </Button>
        </div>

        <div v-if="errorMsg"
            class="bg-destructive/10 border border-destructive/20 rounded-lg p-4 flex items-center gap-3 text-destructive">
            <AlertCircle class="h-5 w-5 shrink-0" />
            <p class="text-sm font-medium">{{ errorMsg }}</p>
        </div>

        <!-- 列表部分 -->
        <div class="rounded-lg border bg-card overflow-hidden">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between px-4 py-3 border-b bg-muted/30 gap-3">
                <div class="relative w-full sm:w-64">
                    <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input v-model="searchQuery" placeholder="搜索语言或版本..." class="h-9 pl-8 text-sm" />
                </div>
                <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
                    <Button variant="outline" class="h-9 px-3 text-sm flex-1 sm:flex-none"
                        @click="showSyncConfirm = true" :disabled="syncing || loading">
                        <RefreshCw class="h-4 w-4 sm:mr-2" :class="{ 'animate-spin': syncing }" />
                        <span class="hidden sm:inline">更新本地环境</span>
                        <span class="sm:hidden">同步</span>
                    </Button>
                    <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadLanguages"
                        :disabled="loading">
                        <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
                    </Button>
                </div>
            </div>

            <div class="divide-y max-h-[600px] overflow-y-auto min-h-[200px]">
                <div v-if="loading && languages.length === 0" class="text-center py-12 text-muted-foreground">
                    <Loader2 class="h-8 w-8 animate-spin mx-auto mb-2 opacity-20" />
                    正在扫描运行环境...
                </div>
                <div v-else-if="filteredLanguages.length === 0 && !loading"
                    class="text-center py-12 text-muted-foreground">
                    <Globe class="h-12 w-12 mx-auto mb-2 opacity-10" />
                    {{ searchQuery ? '未找到匹配的语言' : '未发现已安装的语言' }}
                </div>
                <div v-else v-for="lang in filteredLanguages" :key="lang.plugin + lang.version"
                    class="flex flex-col sm:flex-row sm:items-center justify-between px-4 py-4 hover:bg-muted/50 transition-colors gap-4">
                    <div class="flex items-center gap-4 min-w-0">
                        <div
                            class="h-9 w-9 sm:h-10 sm:w-10 rounded-full bg-primary/10 flex items-center justify-center font-bold text-primary uppercase overflow-hidden shrink-0">
                            <template v-if="getLangIcon(lang.plugin)">
                                <div class="w-full h-full bg-white/80 p-2 flex items-center justify-center">
                                    <img :src="getLangIcon(lang.plugin)" :alt="lang.plugin"
                                        class="w-full h-full object-contain" />
                                </div>
                            </template>
                            <template v-else>
                                {{ lang.plugin.length > 2 ? lang.plugin.substring(0, 2) : lang.plugin }}
                            </template>
                        </div>
                        <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                                <span class="font-bold capitalize truncate">{{ lang.plugin }}</span>
                                <Badge variant="outline" class="font-mono whitespace-nowrap">{{ lang.version }}</Badge>
                            </div>
                            <div class="text-xs text-muted-foreground mt-1 space-y-0.5">
                                <div class="font-mono opacity-60 truncate" :title="lang.source">来源: {{ lang.source }}
                                </div>
                                <div v-if="lang.installed_at" class="opacity-50">
                                    添加日期: {{ lang.installed_at }}
                                </div>
                            </div>
                        </div>
                    </div>
                    <div
                        class="flex items-center gap-2 sm:ml-auto w-full sm:w-auto overflow-x-auto pb-1 sm:pb-0 hide-scrollbar">
                        <Button v-if="SUPPORTED_DEPS_LANGS.includes(lang.plugin)" variant="outline" size="sm"
                            class="whitespace-nowrap flex-1 sm:flex-none"
                            @click="$router.push(`/dependencies?language=${lang.plugin}&version=${lang.version}`)">
                            依赖管理
                        </Button>
                        <Badge v-else variant="secondary"
                            class="h-8 opacity-60 flex-1 sm:flex-none justify-center whitespace-nowrap">
                            不支持管理
                        </Badge>
                        <Button variant="outline" size="sm" class="whitespace-nowrap flex-1 sm:flex-none"
                            @click="handleVerify(lang)">
                            环境验证
                        </Button>
                        <Button variant="ghost" size="icon" class="text-destructive h-8 w-8 shrink-0 ml-auto sm:ml-0"
                            @click="confirmDelete(lang)" title="卸载">
                            <Trash2 class="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            </div>
        </div>

        <!-- 安装对话框 (带搜索下拉) -->
        <Dialog v-model:open="showInstallDialog">
            <DialogContent class="sm:max-w-[400px]">
                <DialogHeader>
                    <DialogTitle>管理语言运行时</DialogTitle>
                    <DialogDescription>配置并安装新的编程语言环境</DialogDescription>
                </DialogHeader>
                <div class="grid gap-6 py-4">
                    <!-- 语言选择 -->
                    <div class="grid gap-2">
                        <Label>语言名称 (Mise Plugin)</Label>
                        <Popover v-model:open="openPluginPopover">
                            <PopoverTrigger asChild>
                                <Button variant="outline" role="combobox" :aria-expanded="openPluginPopover"
                                    class="justify-between w-full font-normal">
                                    {{ newLangPlugin || "选择或输入语言..." }}
                                    <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent class="p-0 w-[var(--reka-popover-trigger-width)]" align="start">
                                <div class="p-2 border-b">
                                    <div class="relative">
                                        <Search
                                            class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                                        <Input v-model="pluginSearch" placeholder="搜索插件..." class="h-8 pl-8 text-xs"
                                            @keydown.enter="() => { if (pluginSearch) { newLangPlugin = pluginSearch; openPluginPopover = false } }" />
                                    </div>
                                </div>
                                <ScrollArea class="h-64">
                                    <div class="p-1">
                                        <div v-if="loadingPlugins" class="flex items-center justify-center py-6">
                                            <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
                                        </div>
                                        <div v-else-if="filteredPlugins.length === 0"
                                            class="py-6 text-center text-xs text-muted-foreground">
                                            未找到匹配插件
                                        </div>
                                        <button v-else v-for="p in filteredPlugins" :key="p"
                                            @click="() => { newLangPlugin = p; openPluginPopover = false }"
                                            class="w-full flex items-center px-2 py-1.5 text-sm rounded-sm hover:bg-muted text-left transition-colors group">
                                            <div
                                                class="mr-2 h-4 w-4 shrink-0 flex items-center justify-center relative">
                                                <div v-if="getLangIcon(p)"
                                                    class="w-full h-full rounded-sm bg-white/80 overflow-hidden p-0.5">
                                                    <img :src="getLangIcon(p)" class="w-full h-full object-contain" />
                                                </div>
                                                <div v-else
                                                    class="w-full h-full flex items-center justify-center bg-primary/10 rounded-sm text-[8px] font-bold uppercase">
                                                    {{ p.substring(0, 2) }}
                                                </div>
                                                <Check v-if="newLangPlugin === p"
                                                    class="absolute -right-2 -top-1 h-3 w-3 text-primary bg-background rounded-full border shadow-sm" />
                                            </div>
                                            <span :class="{ 'font-bold text-primary': newLangPlugin === p }">{{ p
                                            }}</span>
                                        </button>
                                    </div>
                                </ScrollArea>
                            </PopoverContent>
                        </Popover>
                    </div>

                    <!-- 版本选择 -->
                    <div class="grid gap-2">
                        <Label>版本</Label>
                        <Popover v-model:open="openVersionPopover">
                            <PopoverTrigger asChild :disabled="!newLangPlugin">
                                <Button variant="outline" role="combobox" :aria-expanded="openVersionPopover"
                                    class="justify-between w-full font-normal" :disabled="!newLangPlugin">
                                    {{ newLangVersion || "选择或输入版本..." }}
                                    <div class="flex items-center">
                                        <Loader2 v-if="loadingVersions" class="mr-2 h-3 w-3 animate-spin opacity-50" />
                                        <ChevronsUpDown class="h-4 w-4 shrink-0 opacity-50" />
                                    </div>
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent class="p-0 w-[var(--reka-popover-trigger-width)]" align="start">
                                <div class="p-2 border-b">
                                    <div class="relative">
                                        <Search
                                            class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                                        <Input v-model="versionSearch" placeholder="搜索版本..." class="h-8 pl-8 text-xs"
                                            @keydown.enter="() => { if (versionSearch) { newLangVersion = versionSearch; openVersionPopover = false } }" />
                                    </div>
                                </div>
                                <ScrollArea class="h-64">
                                    <div class="p-1">
                                        <div v-if="loadingVersions" class="flex items-center justify-center py-6">
                                            <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
                                        </div>
                                        <div v-else-if="filteredVersions.length === 0"
                                            class="py-6 text-center text-xs text-muted-foreground">
                                            未找到匹配版本
                                        </div>
                                        <button v-else v-for="v in filteredVersions" :key="v"
                                            @click="() => { newLangVersion = v; openVersionPopover = false }"
                                            class="w-full flex items-center px-2 py-1.5 text-sm rounded-sm hover:bg-muted text-left transition-colors">
                                            <Check
                                                :class="cn('mr-2 h-3.5 w-3.5', newLangVersion === v ? 'opacity-100' : 'opacity-0')" />
                                            {{ v }}
                                        </button>
                                    </div>
                                </ScrollArea>
                            </PopoverContent>
                        </Popover>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" @click="showInstallDialog = false">取消</Button>
                    <Button @click="startInstall">开始安装</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>

        <!-- 终端对话框 -->
        <Dialog v-model:open="showTerminalDialog" @update:open="(val) => !val && loadLanguages()">
            <DialogContent class="max-w-4xl h-[600px] p-0 overflow-hidden bg-[#1e1e1e] border-none shadow-2xl">
                <DialogHeader class="sr-only">
                    <DialogTitle>终端执行</DialogTitle>
                    <DialogDescription>正在执行 mise 相关指令</DialogDescription>
                </DialogHeader>
                <div class="flex flex-col h-full">
                    <div class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#3c3c3c]">
                        <div class="flex items-center gap-2">
                            <TerminalIcon class="h-4 w-4 text-primary" />
                            <span class="text-xs font-medium text-gray-300">正在安装 / 执行: {{ terminalCommand }}</span>
                        </div>
                        <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white"
                            @click="showTerminalDialog = false">
                            <X class="h-4 w-4" />
                        </Button>
                    </div>
                    <div class="flex-1">
                        <XTerminal v-if="showTerminalDialog" :font-size="13" :initial-command="terminalCommand" />
                    </div>
                </div>
            </DialogContent>
        </Dialog>

        <!-- 同步确认对话框 -->
        <Dialog v-model:open="showSyncConfirm">
            <DialogContent class="sm:max-w-[400px]">
                <DialogHeader>
                    <DialogTitle>同步本地环境</DialogTitle>
                    <DialogDescription>
                        将实时扫描系统中已安装的所有 Mise 运行时并更新到数据库表中。
                        <p class="mt-2 text-destructive font-medium italic text-xs">注意：这可能会覆盖或更新表中的记录。</p>
                    </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                    <Button variant="outline" @click="showSyncConfirm = false" :disabled="syncing">取消</Button>
                    <Button @click="handleSync" :disabled="syncing">
                        <Loader2 v-if="syncing" class="mr-2 h-4 w-4 animate-spin" />
                        立即同步
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
</template>
