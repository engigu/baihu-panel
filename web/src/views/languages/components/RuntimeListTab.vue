<script setup lang="ts">
import { ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Search, RefreshCw, Loader2, Globe, Trash2, Terminal as TerminalIcon } from 'lucide-vue-next'
import { getLangIcon } from '@/utils/icons'
import type { MiseLanguage } from '@/api'
import { cn } from '@/lib/utils'
import { useRouter } from 'vue-router'

interface DisplayLanguage extends Omit<MiseLanguage, 'source'> {
    source: string
    isGlobal: boolean
}

const SUPPORTED_DEPS_LANGS = [
    'python', 'node', 'ruby', 'go', 'rust', 'bun', 'php',
    'deno', 'dotnet', 'elixir', 'erlang', 'lua', 'nim',
    'dart', 'flutter', 'perl', 'crystal'
]

const props = defineProps<{
    languages: DisplayLanguage[]
    loading: boolean
    syncing: boolean
}>()

const emit = defineEmits<{
    (e: 'sync'): void
    (e: 'refresh'): void
    (e: 'uninstall', lang: MiseLanguage): void
    (e: 'verify', lang: MiseLanguage): void
    (e: 'toggle-default', lang: MiseLanguage): void
    (e: 'view-detail', lang: DisplayLanguage): void
}>()

const router = useRouter()
const searchQuery = ref('')

const filteredLanguages = computed(() => {
    if (!searchQuery.value) return props.languages
    const q = searchQuery.value.toLowerCase()
    return props.languages.filter(l => l.plugin.toLowerCase().includes(q) || l.version.toLowerCase().includes(q))
})
</script>

<template>
    <div class="rounded-lg border bg-card overflow-hidden">
        <div class="flex items-center justify-between px-3 sm:px-4 py-2 sm:py-3 border-b bg-muted/30 gap-2 sm:gap-4">
            <div class="relative w-full sm:w-64 group">
                <Search
                    class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/60 transition-colors group-focus-within:text-primary" />
                <Input v-model="searchQuery" placeholder="搜索语言或版本..."
                    class="h-9 pl-9 text-sm bg-muted/20 border-border/40 focus:bg-background transition-all" />
            </div>
            <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
                <Button variant="outline" class="h-8 sm:h-9 px-2 sm:px-3 text-xs sm:text-sm shadow-sm"
                    @click="emit('sync')" :disabled="syncing || loading">
                    <RefreshCw class="h-3.5 w-3.5 sm:mr-2" :class="{ 'animate-spin': syncing }" />
                    <span class="hidden sm:inline">更新环境</span>
                    <span class="sm:hidden">同步</span>
                </Button>
                <Button variant="outline" size="icon" class="h-8 w-8 sm:h-9 sm:w-9 shrink-0 shadow-sm"
                    @click="emit('refresh')" :disabled="loading">
                    <RefreshCw class="h-3.5 w-3.5 sm:h-4 sm:w-4" :class="{ 'animate-spin': loading }" />
                </Button>
            </div>
        </div>

        <div class="max-h-[700px] overflow-y-auto min-h-[200px] p-3 sm:p-4">
            <div v-if="loading && languages.length === 0" class="text-center py-12 text-muted-foreground">
                <Loader2 class="h-8 w-8 animate-spin mx-auto mb-2 opacity-20" />
                正在扫描运行环境...
            </div>
            <div v-else-if="filteredLanguages.length === 0 && !loading"
                class="text-center py-12 text-muted-foreground">
                <Globe class="h-12 w-12 mx-auto mb-2 opacity-10" />
                {{ searchQuery ? '未找到匹配的语言' : '未发现已安装的语言' }}
            </div>
            <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
                <div v-for="(lang, index) in filteredLanguages" :key="lang.plugin + lang.version"
                    class="relative group/item flex flex-col p-3.5 sm:p-4 rounded-xl border bg-card/40 hover:bg-accent/40 hover:border-primary/40 transition-all duration-200 shadow-sm">
                    
                    <!-- 顶部：图标、名称、版本与删除 -->
                    <div class="flex items-start justify-between gap-3">
                        <div class="flex items-center gap-3 overflow-hidden">
                            <!-- 序号 -->
                            <div class="flex items-center justify-center w-6 sm:w-7 shrink-0">
                                <span class="text-[9px] font-mono text-muted-foreground/50 tabular-nums">#{{
                                    filteredLanguages.length - index }}</span>
                            </div>
                            <!-- 图标 -->
                            <div class="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center font-bold text-primary uppercase shrink-0 border border-primary/10 shadow-sm overflow-hidden bg-background">
                                <template v-if="getLangIcon(lang.plugin)">
                                    <div class="w-full h-full p-2 flex items-center justify-center">
                                        <img :src="getLangIcon(lang.plugin)" :alt="lang.plugin" class="w-full h-full object-contain" />
                                    </div>
                                </template>
                                <template v-else>
                                    <span class="text-xs">{{ lang.plugin.substring(0, 2) }}</span>
                                </template>
                            </div>
                            <!-- 基本信息 -->
                            <div class="min-w-0">
                                <div class="flex items-center gap-2">
                                    <span class="font-bold capitalize truncate text-sm sm:text-base text-foreground">{{ lang.plugin }}</span>
                                    <Badge v-if="lang.isGlobal" variant="secondary"
                                        class="bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20 text-[10px] h-4.5 px-1.5 font-medium shrink-0">
                                        默认
                                    </Badge>
                                </div>
                                <div class="mt-0.5">
                                    <Badge variant="outline" class="font-mono text-[10px] h-4.5 px-1.5 bg-muted/50 border-muted-foreground/20 leading-none">{{ lang.version }}</Badge>
                                </div>
                            </div>
                        </div>

                        <!-- 右上角删除按钮 -->
                        <Button variant="ghost" size="icon" 
                            class="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all sm:opacity-0 group-hover/item:opacity-100" 
                            @click="emit('uninstall', lang)" title="卸载">
                            <Trash2 class="h-4 w-4" />
                        </Button>
                    </div>

                    <!-- 中间：完整路径 (可点击查看详情) -->
                    <div class="mt-3 px-1">
                        <div class="text-[11px] text-muted-foreground cursor-help hover:text-primary transition-colors flex items-center bg-muted/30 rounded-md py-1.5 px-2 group/path" @click="emit('view-detail', lang)">
                            <TerminalIcon class="h-3 w-3 mr-1.5 shrink-0 opacity-50" />
                            <span class="font-mono truncate select-all">{{ lang.source }}</span>
                        </div>
                    </div>

                    <!-- 底部：操作按钮网格 -->
                    <div class="mt-4 flex items-center gap-2">
                        <Button variant="outline" size="sm"
                            :disabled="!SUPPORTED_DEPS_LANGS.includes(lang.plugin)"
                            class="h-8 px-0 text-[11px] font-semibold flex-1 bg-background/50 disabled:opacity-40 disabled:cursor-not-allowed"
                            @click="router.push(`/dependencies?language=${lang.plugin}&version=${lang.version}`)">
                            依赖管理
                        </Button>
                        <Button variant="outline" size="sm"
                            class="h-8 px-0 text-[11px] font-semibold flex-1 bg-background/50"
                            @click="emit('verify', lang)">
                            运行验证
                        </Button>
                        <Button variant="outline" size="sm"
                            :class="cn('h-8 px-0 text-[11px] font-semibold flex-1 bg-background/50', 
                                lang.isGlobal ? 'text-amber-600 border-amber-500/30 bg-amber-500/5 dark:bg-amber-500/10' : '')"
                            @click="emit('toggle-default', lang)">
                            {{ lang.isGlobal ? '取消默认' : '设为默认' }}
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
