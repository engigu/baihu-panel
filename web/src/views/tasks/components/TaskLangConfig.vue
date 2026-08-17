<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Check, ChevronsUpDown, Plus, Search, X, Loader2 } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { api, type MiseLanguage } from '@/api'
import { getLangIcon } from '@/utils/icons'

export interface LangConfig {
  name: string
  version: string
  availableVersions: string[]
}

const props = defineProps<{
  modelValue: LangConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: LangConfig[]]
}>()

const selectedLangs = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const installedLangs = ref<MiseLanguage[]>([])
const loadingLangs = ref(false)
const availablePlugins = ref<string[]>([])
const pluginSearch = ref('')
const versionSearch = ref('')

const filteredPlugins = computed(() => {
  if (!pluginSearch.value) return availablePlugins.value
  const s = pluginSearch.value.toLowerCase()
  return availablePlugins.value.filter((p: string) => p.toLowerCase().includes(s))
})

function getFilteredVersions(versions: string[]) {
  if (!versionSearch.value) return versions
  const s = versionSearch.value.toLowerCase()
  return versions.filter((v: string) => v.toLowerCase().includes(s))
}

async function fetchInstalledLangs() {
  loadingLangs.value = true
  try {
    installedLangs.value = await api.mise.list()
    const plugins = new Set<string>()
    installedLangs.value.forEach((l: MiseLanguage) => plugins.add(l.plugin))
    availablePlugins.value = Array.from(plugins).sort()
    
    // 刷新已选语言的可用版本
    selectedLangs.value.forEach(lang => updateAvailableVersions(lang))
  } catch (e) {
    console.error('Fetch installed langs failed', e)
  } finally {
    loadingLangs.value = false
  }
}

function updateAvailableVersions(lang: LangConfig) {
  if (lang.name) {
    lang.availableVersions = installedLangs.value
      .filter((l: MiseLanguage) => l.plugin === lang.name)
      .map((l: MiseLanguage) => l.version)
      .sort((a: string, b: string) => b.localeCompare(a, undefined, { numeric: true }))
  } else {
    lang.availableVersions = []
  }
}

function addLang() {
  const newList = [...selectedLangs.value, { name: '', version: '', availableVersions: [] }]
  emit('update:modelValue', newList)
}

function removeLang(index: number) {
  const newList = [...selectedLangs.value]
  newList.splice(index, 1)
  emit('update:modelValue', newList)
}

function updateLangName(index: number, name: string) {
  const lang = selectedLangs.value[index]
  if (!lang) return
  lang.name = name
  lang.version = '' // reset version
  updateAvailableVersions(lang)
}

// 当组件挂载时加载语言环境列表
onMounted(() => {
  fetchInstalledLangs()
})
</script>

<template>
  <div class="space-y-2">
    <div v-for="(clang, idx) in selectedLangs" :key="idx" class="flex gap-2 p-2 rounded-lg bg-muted/20 border border-muted-foreground/10 group/lang relative overflow-hidden">
      <div class="absolute left-0 top-0 bottom-0 w-0.5 bg-primary/20 group-hover/lang:bg-primary transition-colors" />
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="ghost" class="justify-between flex-1 h-8 text-xs font-normal hover:bg-background/50">
            <div class="flex items-center gap-2 truncate">
              <div v-if="clang.name && getLangIcon(clang.name)" class="w-4 h-4 shrink-0 rounded-sm bg-white p-0.5 border shadow-sm">
                <img :src="getLangIcon(clang.name)" class="w-full h-full object-contain" />
              </div>
              <span class="font-medium">{{ clang.name || "选择环境..." }}</span>
            </div>
            <ChevronsUpDown class="ml-1 h-3 w-3 opacity-40" />
          </Button>
        </PopoverTrigger>
        <PopoverContent class="p-0 w-[240px]" align="start">
          <div class="p-2 border-b bg-muted/30">
            <div class="relative">
              <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <Input v-model="pluginSearch" placeholder="搜索已安装语言..." :class="cn('h-8 pl-8 bg-background border-muted-foreground/20', pluginSearch ? 'text-xs font-medium' : 'text-[10px]')" />
            </div>
          </div>
          <ScrollArea class="h-48 p-1">
            <div v-if="loadingLangs" class="flex items-center justify-center py-6">
              <Loader2 class="h-5 w-5 animate-spin text-primary/50" />
            </div>
            <button v-else v-for="p in filteredPlugins" :key="p" @click="updateLangName(idx, p)" class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent text-left transition-all mb-0.5">
              <span class="flex-1" :class="{ 'font-bold text-primary': clang.name === p }">{{ p }}</span>
              <Check v-if="clang.name === p" class="h-3 w-3 text-primary" />
            </button>
          </ScrollArea>
        </PopoverContent>
      </Popover>
      
      <Popover>
        <PopoverTrigger asChild :disabled="!clang.name">
          <Button variant="ghost" class="justify-between w-28 h-8 text-xs font-normal hover:bg-background/50" :disabled="!clang.name">
            <span class="truncate">{{ clang.version || "版本..." }}</span>
            <ChevronsUpDown class="h-3 w-3 opacity-40 ml-1" />
          </Button>
        </PopoverTrigger>
        <PopoverContent class="p-0 w-[160px]" align="start">
          <ScrollArea class="h-48 p-1">
            <div v-if="getFilteredVersions(clang.availableVersions).length === 0" class="py-6 text-center text-xs text-muted-foreground">
              无可用版本
            </div>
            <button v-else v-for="v in getFilteredVersions(clang.availableVersions)" :key="v" @click="clang.version = v" class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent font-mono mb-0.5 text-left">
              <span class="flex-1 truncate" :class="{ 'font-bold text-primary': clang.version === v }">{{ v }}</span>
              <Check v-if="clang.version === v" class="h-3 w-3 text-primary" />
            </button>
          </ScrollArea>
        </PopoverContent>
      </Popover>
      
      <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10 shrink-0" @click="removeLang(idx)">
        <X class="h-4 w-4" />
      </Button>
    </div>
    
    <Button variant="outline" size="sm" class="w-full h-9 text-xs border-dashed border-muted-foreground/30 text-muted-foreground hover:text-primary hover:border-primary/50 transition-all bg-muted/5 hover:bg-primary/5 rounded-xl" @click="addLang">
      <Plus class="h-4 w-4 mr-2" /> 添加运行时环境 (Mise)
    </Button>
  </div>
</template>
