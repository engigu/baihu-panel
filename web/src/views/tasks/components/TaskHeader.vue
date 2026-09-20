<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import TagInput from '@/components/TagInput.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import {
  Search, Tag, ChevronDown, RefreshCw, Wrench, Plus, GitBranch, Terminal, Package,
  Sparkles, Trash2, Pin, X, Server, Loader2, Layers
} from 'lucide-vue-next'
import { TASK_TYPE, TASK_TYPE_CONFIG } from '@/constants'
import type { TaskView } from '../Tasks.vue'

const props = defineProps<{
  filterName: string
  filterTags: string
  filterType: string
  filterAgentId: string | null
  filterAgentName: string
  taskViews: TaskView[]
  loading: boolean
}>()

const emit = defineEmits<{
  'update:filterName': [value: string]
  'update:filterTags': [value: string]
  'update:filterType': [value: string]
  'search': []
  'typeChange': []
  'refresh': []
  'clearAgentFilter': []
  'openBatchUpdate': []
  'confirmBatchDelete': []
  'openCreateTask': []
  'openCreateRepo': []
  'applyView': [view: TaskView]
  'toggleDefaultView': [index: number]
  'deleteView': [index: number]
  'saveView': [name: string]
}>()

const router = useRouter()
const newViewName = ref('')
const isSavingView = ref(false)

function handleSaveView() {
  if (!newViewName.value.trim()) return
  isSavingView.value = true
  emit('saveView', newViewName.value.trim())
  newViewName.value = ''
  isSavingView.value = false
}
</script>

<template>
  <div class="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
    <!-- 左侧标题与视图切换 -->
    <div class="flex flex-col shrink-0">
      <Popover>
        <PopoverTrigger as-child>
          <div class="flex items-center gap-2 cursor-pointer group w-fit">
            <h2 class="text-xl sm:text-2xl font-bold tracking-tight">
              {{ filterType === TASK_TYPE.REPO ? '仓库同步' : (filterType === TASK_TYPE.APP ? '已装应用' : '调度实体') }}
            </h2>
            <div class="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-muted/50 group-hover:bg-primary/10 transition-colors border border-transparent group-hover:border-primary/20">
              <span class="text-[10px] font-bold text-muted-foreground group-hover:text-primary uppercase tracking-wider">视图</span>
              <ChevronDown class="h-3.5 w-3.5 text-muted-foreground group-hover:text-primary transition-colors" />
            </div>
          </div>
        </PopoverTrigger>

        <PopoverContent class="w-64 p-3 shadow-xl border-muted-foreground/10" align="start" :side-offset="8" @open-autofocus="(e: Event) => e.preventDefault()">
          <div class="space-y-4">
            <div>
              <div class="flex items-center justify-between mb-2 px-1">
                <h4 class="text-sm font-semibold">我的视图</h4>
              </div>
              <div v-if="taskViews.length === 0" class="text-xs text-muted-foreground px-1 py-4 text-center border-2 border-dashed rounded-md bg-muted/20">
                暂无保存的视图
              </div>
              <div class="space-y-1.5 max-h-[220px] overflow-y-auto custom-scrollbar">
                <div
                  v-for="(view, index) in taskViews"
                  :key="index"
                  class="flex items-center justify-between pl-2 pr-1 py-1.5 rounded-lg border transition-all cursor-pointer"
                  :class="view.isDefault ? 'bg-primary/5 text-primary border-primary/30' : 'bg-muted/10 border-muted-foreground/10 hover:bg-muted/30'"
                  @click="$emit('applyView', view)"
                >
                  <div class="flex items-center gap-1.5 min-w-0 flex-1">
                    <button
                      type="button"
                      class="p-0.5 rounded-md transition-colors shrink-0 focus:outline-none"
                      :class="view.isDefault ? 'text-primary' : 'text-muted-foreground/40 hover:text-primary hover:bg-primary/10'"
                      :title="view.isDefault ? '取消默认视图' : '设为默认视图'"
                      @click.stop="$emit('toggleDefaultView', index)"
                    >
                      <Pin class="h-3.5 w-3.5" :class="{ 'fill-primary rotate-45': view.isDefault }" />
                    </button>
                    <span class="text-xs font-medium truncate max-w-[130px]" :class="{ 'font-semibold': view.isDefault }">{{ view.name }}</span>
                  </div>
                  <button
                    type="button"
                    class="p-1 rounded-md text-muted-foreground/60 hover:bg-destructive/10 hover:text-destructive transition-colors focus:outline-none"
                    @click.stop="$emit('deleteView', index)"
                  >
                    <X class="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </div>

            <div class="pt-3 border-t space-y-2.5">
              <h4 class="text-xs font-semibold px-1 text-foreground/70 uppercase tracking-wider">保存当前过滤为新视图</h4>
              <div class="flex gap-2">
                <Input v-model="newViewName" placeholder="视图名称..." class="h-9 text-xs bg-muted/30 focus:bg-background" @keydown.enter="handleSaveView" />
                <Button size="sm" class="h-9 px-3" :disabled="isSavingView" @click="handleSaveView">
                  <Plus v-if="!isSavingView" class="h-4 w-4" />
                  <Loader2 v-else class="h-4 w-4 animate-spin" />
                </Button>
              </div>
            </div>
          </div>
        </PopoverContent>
      </Popover>
      <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">管理和调度自动化执行任务</p>
    </div>

    <!-- 右侧操作与筛选 -->
    <div class="flex flex-row items-center flex-wrap gap-2 w-full xl:w-auto xl:ml-auto xl:justify-end">
      <!-- 搜索与标签 -->
      <div class="flex flex-row items-center gap-2 w-full sm:flex-1 xl:flex-none xl:w-auto text-sm">
        <div class="relative flex-1 xl:flex-none xl:w-[240px] group">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input
            :model-value="filterName"
            placeholder="搜索任务..."
            type="search"
            autocomplete="off"
            name="filter_name_input"
            class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
            @update:model-value="$emit('update:filterName', String($event)); $emit('search')"
          />
        </div>
        <TagInput
          :model-value="filterTags"
          placeholder="搜索标签..."
          :icon="Tag"
          multiple
          class="h-9 flex-1 xl:flex-none xl:w-[180px] bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
          @enter="$emit('search')"
          @update:model-value="$emit('update:filterTags', $event); $emit('search')"
        />
      </div>

      <div class="flex items-center gap-1.5 w-full sm:gap-2 sm:w-auto sm:justify-end">
        <!-- 节点 Tag Badge -->
        <div v-if="filterAgentId" class="hidden xl:flex items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm shrink-0">
          <Server class="h-3.5 w-3.5" />
          <span>{{ filterAgentName }}</span>
          <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="$emit('clearAgentFilter')" />
        </div>

        <!-- 刷新按钮 -->
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" :disabled="loading" title="刷新" @click="$emit('refresh')">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        </Button>

        <!-- 批量操作下拉菜单 -->
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="outline" class="shrink-0 px-2 xl:px-3 h-9 shadow-sm gap-1">
              <Wrench class="h-4 w-4" />
              <span class="hidden xl:inline">批量操作</span>
              <ChevronDown class="h-3.5 w-3.5 opacity-60" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-40">
            <DropdownMenuItem class="cursor-pointer gap-2" @click="$emit('openBatchUpdate')">
              <Sparkles class="h-4 w-4 text-primary" />
              <span>批量修改配置</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="cursor-pointer gap-2 text-destructive focus:text-destructive" @click="$emit('confirmBatchDelete')">
              <Trash2 class="h-4 w-4" />
              <span>批量删除任务</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <!-- 统一新建下拉菜单按钮 -->
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button class="shrink-0 px-2.5 xl:px-3.5 h-9 shadow-sm font-medium gap-1 text-sm bg-primary text-primary-foreground hover:bg-primary/90">
              <Plus class="h-4 w-4" />
              <span class="hidden sm:inline">新建</span>
              <ChevronDown class="h-3.5 w-3.5 opacity-75" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-52 p-1">
            <DropdownMenuItem class="cursor-pointer gap-2.5 py-2" @click="$emit('openCreateTask')">
              <Terminal class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]?.color" />
              <div class="flex flex-col min-w-0">
                <span class="font-medium text-sm leading-none mb-1">新建脚本任务</span>
                <span class="text-[11px] text-muted-foreground leading-none">自定义单次 / 定时 CLI 任务</span>
              </div>
            </DropdownMenuItem>
            <DropdownMenuItem class="cursor-pointer gap-2.5 py-2" @click="$emit('openCreateRepo')">
              <GitBranch class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.REPO]?.color" />
              <div class="flex flex-col min-w-0">
                <span class="font-medium text-sm leading-none mb-1">添加仓库同步</span>
                <span class="text-[11px] text-muted-foreground leading-none">自动同步 Git / 远程代码仓库</span>
              </div>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="cursor-pointer gap-2.5 py-2" @click="router.push('/apps')">
              <Package class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.APP]?.color" />
              <div class="flex flex-col min-w-0">
                <span class="font-medium text-sm leading-none mb-1">从应用市场安装</span>
                <span class="text-[11px] text-muted-foreground leading-none">探索并一键部署预装应用</span>
              </div>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <!-- 调度实体类型切换下拉框 -->
        <Select :model-value="filterType" @update:model-value="$emit('update:filterType', String($event)); $emit('typeChange')">
          <SelectTrigger class="h-9 flex-1 sm:flex-none sm:w-[128px] px-2.5 text-sm font-medium bg-muted/20 border-muted-foreground/20 shadow-xs gap-1.5 justify-between">
            <SelectValue placeholder="全部类型">
              <div class="flex items-center gap-2 min-w-0 text-sm">
                <Layers v-if="filterType === 'all'" class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG['all']?.color" />
                <Terminal v-else-if="filterType === TASK_TYPE.NORMAL || filterType?.startsWith('app:')" class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]?.color" />
                <GitBranch v-else-if="filterType === TASK_TYPE.REPO" class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.REPO]?.color" />
                <Package v-else-if="filterType === TASK_TYPE.APP" class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.APP]?.color" />
                <span class="truncate">{{ filterType === 'all' ? '全部类型' : (filterType === TASK_TYPE.REPO ? '仓库同步' : (filterType === TASK_TYPE.APP ? '已装应用' : '脚本任务')) }}</span>
              </div>
            </SelectValue>
          </SelectTrigger>
          <SelectContent align="end" class="w-[132px] min-w-[132px] p-1 text-sm font-medium">
            <SelectItem value="all" class="py-1.5 pl-2 pr-6 text-sm">
              <div class="flex items-center gap-2">
                <Layers class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG['all']?.color" />
                <span>全部类型</span>
              </div>
            </SelectItem>
            <SelectItem :value="TASK_TYPE.NORMAL" class="py-1.5 pl-2 pr-6 text-sm">
              <div class="flex items-center gap-2">
                <Terminal class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]?.color" />
                <span>脚本任务</span>
              </div>
            </SelectItem>
            <SelectItem :value="TASK_TYPE.REPO" class="py-1.5 pl-2 pr-6 text-sm">
              <div class="flex items-center gap-2">
                <GitBranch class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.REPO]?.color" />
                <span>仓库同步</span>
              </div>
            </SelectItem>
            <SelectItem :value="TASK_TYPE.APP" class="py-1.5 pl-2 pr-6 text-sm">
              <div class="flex items-center gap-2">
                <Package class="w-4 h-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.APP]?.color" />
                <span>已装应用</span>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  </div>
</template>
