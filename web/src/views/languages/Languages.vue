<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Plus, Loader2, Terminal as TerminalIcon, X, AlertCircle, Boxes, Zap } from 'lucide-vue-next'
import { api, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'
import XTerminal from '@/components/XTerminal.vue'

// Import components
import RuntimeListTab from './components/RuntimeListTab.vue'
import EnvMirrorTab from './components/EnvMirrorTab.vue'
import InstallLanguageDialog from './components/InstallLanguageDialog.vue'
import LanguageDetailDialog from './components/LanguageDetailDialog.vue'

interface DisplayLanguage extends Omit<MiseLanguage, 'source'> {
    source: string
    isGlobal: boolean
}

const activeTab = ref('runtimes')

// Shared State
const languages = ref<DisplayLanguage[]>([])
const loading = ref(false)
const errorMsg = ref('')

const syncing = ref(false)
const showSyncConfirm = ref(false)

// Dialog Refs and States
const installDialogRef = ref<InstanceType<typeof InstallLanguageDialog> | null>(null)
const showDetailDialog = ref(false)
const selectedLang = ref<DisplayLanguage | null>(null)

// Terminal State
const showTerminalDialog = ref(false)
const terminalCommand = ref('')
const isInstallSuccess = ref(false)

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
            source: typeof item.source === 'object' ? (item.source.path || item.source.type || '-') : (item.source || '-'),
            isGlobal: !!item.is_global
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

function openInstallDialog() {
    installDialogRef.value?.openDialog()
}

function runInTerminal(command: string) {
    terminalCommand.value = command
    isInstallSuccess.value = false
    showTerminalDialog.value = true
}

function handleInstall(plugin: string, version: string) {
    const cmd = `mise install ${plugin}@${version}`
    runInTerminal(cmd)
}

function confirmDelete(lang: MiseLanguage) {
    const cmd = `mise uninstall ${lang.plugin}@${lang.version}`
    runInTerminal(cmd)
}

async function handleVerify(lang: MiseLanguage) {
    try {
        const { command } = await api.mise.verifyCommand(lang.plugin, lang.version)
        runInTerminal(command)
    } catch (e) {
        toast.error('获取验证命令失败')
    }
}

async function toggleDefault(lang: MiseLanguage) {
    try {
        if (lang.is_global) {
            await api.mise.unsetGlobal(lang.plugin, lang.version)
            toast.success(`已取消 ${lang.plugin} 的全局默认设置`)
        } else {
            await api.mise.useGlobal(lang.plugin, lang.version)
            toast.success(`已将 ${lang.plugin} ${lang.version} 设为全局默认版本`)
        }
        await loadLanguages()
    } catch (e) {
        toast.error('操作失败: ' + e)
    }
}

async function handleTerminalClose() {
    showTerminalDialog.value = false
    syncing.value = true
    try {
        await api.mise.sync()
        toast.success('环境同步完成')
    } catch (e) {
        toast.error('环境同步失败: ' + e)
    } finally {
        syncing.value = false
    }
    await loadLanguages()
}

function viewDetail(lang: DisplayLanguage) {
    selectedLang.value = lang
    showDetailDialog.value = true
}

onMounted(() => {
    loadLanguages()
})
</script>

<template>
    <div class="space-y-4">
        <!-- 顶栏：标题与操作 -->
        <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div>
                <h2 class="text-xl sm:text-2xl font-bold tracking-tight">语言依赖</h2>
                <p class="text-muted-foreground text-sm mt-0.5">管理编程语言及相关包依赖(Mise)</p>
            </div>

            <div class="flex flex-row items-center justify-start sm:justify-end gap-2 shrink-0 w-full md:w-auto overflow-x-auto pb-1 sm:pb-0">
                <Button @click="openInstallDialog" class="flex-1 sm:flex-none h-9 text-sm px-2 sm:px-4">
                    <Plus class="h-3.5 w-3.5 sm:mr-2" /> <span class="whitespace-nowrap">新增语言</span>
                </Button>
                <Tabs v-model="activeTab" class="flex-1 sm:flex-none">
                    <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 rounded-lg w-full sm:w-auto flex">
                        <TabsTrigger value="runtimes" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
                            <Boxes class="w-3.5 h-3.5 opacity-70" />
                            <span>运行时</span>
                        </TabsTrigger>
                        <TabsTrigger value="envs" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
                            <Zap class="w-3.5 h-3.5 opacity-70" />
                            <span>加速</span>
                        </TabsTrigger>
                    </TabsList>
                </Tabs>
            </div>
        </div>

        <div
            class="flex items-center gap-2.5 text-[13px] text-amber-600 dark:text-amber-500 bg-amber-500/10 px-4 py-2.5 rounded-lg border border-amber-500/20 leading-relaxed shadow-sm select-none">
            <AlertCircle class="h-4 w-4 shrink-0" />
            <span>
                <b class="font-bold">设为默认</b>：将选定版本设为系统全局默认 (mise use -g)，生效后所有未通过高级配置指定特定环境的任务将默认调用此环境。
            </span>
        </div>

        <Tabs v-model="activeTab" class="w-full">
            <div v-if="errorMsg"
                class="bg-destructive/10 border border-destructive/20 rounded-lg p-4 flex items-center gap-3 text-destructive mb-4">
                <AlertCircle class="h-5 w-5 shrink-0" />
                <p class="text-sm font-medium">{{ errorMsg }}</p>
            </div>

            <TabsContent value="runtimes" class="space-y-4 outline-none">
                <RuntimeListTab 
                    :languages="languages" 
                    :loading="loading" 
                    :syncing="syncing"
                    @sync="showSyncConfirm = true"
                    @refresh="loadLanguages"
                    @uninstall="confirmDelete"
                    @verify="handleVerify"
                    @toggle-default="toggleDefault"
                    @view-detail="viewDetail"
                />
            </TabsContent>

            <TabsContent value="envs" class="space-y-4 outline-none">
                <EnvMirrorTab />
            </TabsContent>
        </Tabs>

        <!-- 对话框组件 -->
        <InstallLanguageDialog ref="installDialogRef" @install="handleInstall" />
        <LanguageDetailDialog v-model:open="showDetailDialog" :lang="selectedLang" />

        <!-- 终端对话框 -->
        <Dialog v-model:open="showTerminalDialog">
            <DialogContent
                class="w-[calc(100%-2rem)] sm:max-w-[90vw] lg:max-w-4xl xl:max-w-5xl h-[60vh] sm:h-[70vh] flex flex-col p-0 overflow-hidden bg-[#1e1e1e] border-none shadow-2xl"
                :show-close-button="false" @interact-outside="(e) => e.preventDefault()"
                @escape-key-down="(e) => e.preventDefault()">
                <DialogHeader class="sr-only">
                    <DialogTitle>终端执行</DialogTitle>
                    <DialogDescription>正在执行 mise 相关指令</DialogDescription>
                </DialogHeader>
                <div class="flex flex-col h-full">
                    <div class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#3c3c3c]">
                        <div class="flex items-center gap-2">
                            <TerminalIcon class="h-4 w-4 text-white" />
                            <span class="text-xs font-medium text-gray-300">正在安装 / 执行: {{ terminalCommand }}</span>
                        </div>
                        <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white"
                            @click="handleTerminalClose">
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
