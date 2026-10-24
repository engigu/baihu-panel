<script setup lang="ts">
import { computed } from 'vue'
import AppTaskRow from './rows/AppTaskRow.vue'
import NormalTaskRow from './rows/NormalTaskRow.vue'
import RepoTaskRow from './rows/RepoTaskRow.vue'
import { Checkbox } from '@/components/ui/checkbox'
import { ArrowUpDown, ArrowUp, ArrowDown, GitBranch, Terminal, Package } from 'lucide-vue-next'
import type { Task, Agent, AppUpdateInfo } from '@/api'
import { TASK_TYPE } from '@/constants'

const props = withDefaults(defineProps<{
  tasks: Task[]
  total: number
  currentPage: number
  pageSize: number
  filterType: string
  sortBy: string
  order: string
  executingTaskId?: string | null
  agentMap: Record<string, Agent>
  selectedIds?: string[]
  isAllSelected?: boolean
  isIndeterminate?: boolean
  updateMap?: Record<string, AppUpdateInfo>
}>(), {
  selectedIds: () => [],
  isAllSelected: false,
  isIndeterminate: false,
  updateMap: () => ({})
})

const hasSelection = computed(() => (props.selectedIds?.length || 0) > 0)

const emit = defineEmits<{
  'toggleSort': [field: string]
  'runTask': [id: string]
  'viewLogs': [id: string]
  'openEditTask': [task: Task]
  'openEditApp': [task: Task]
  'uninstallApp': [task: Task]
  'filterByApp': [appId: string]
  'toggleTask': [task: Task, enabled: boolean]
  'togglePin': [task: Task]
  'depInstall': [task: Task]
  'duplicateTask': [task: Task]
  'openExportDialog': [task: Task]
  'confirmDelete': [id: string]
  'shareApp': [task: Task]
  'toggleSelect': [id: string]
  'toggleAll': []
  'openAppDetail': [task: Task]
}>()
</script>

<template>
  <div class="rounded-lg border bg-card overflow-hidden">
    <!-- 小屏 (sm:hidden) 专属多选控制条：当有选中项时展示在列表顶部 -->
    <div
      v-if="hasSelection"
      class="sm:hidden flex items-center justify-between px-3.5 py-2 bg-primary/5 border-b border-primary/20 text-xs"
    >
      <div class="flex items-center gap-2">
        <Checkbox
          :checked="isAllSelected ? true : (isIndeterminate ? 'indeterminate' : false)"
          id="mobile-select-all"
          @update:checked="$emit('toggleAll')"
        />
        <label for="mobile-select-all" class="text-xs font-semibold cursor-pointer select-none">
          {{ isAllSelected ? '已全选本页' : '全选本页' }}
        </label>
      </div>
      <span class="text-[11px] text-muted-foreground font-mono">
        已选 {{ selectedIds?.length || 0 }} / {{ tasks.length }} 项
      </span>
    </div>

    <!-- 大屏/中屏统一表格头部 (sm 隐藏，md/lg 显示) -->
    <div class="hidden sm:flex items-center gap-2 px-4 py-2 border-b bg-muted/20 text-xs text-muted-foreground font-medium select-none group/header">
      <!-- 对应 StatusDot 宽度的微距占位，保持与行内首项像素级对齐 -->
      <span class="w-2 shrink-0" />

      <!-- 序号列 / 全选 Checkbox 同位无缝替换 -->
      <div class="w-12 shrink-0 flex items-center">
        <Checkbox
          v-if="hasSelection || isAllSelected || isIndeterminate"
          :checked="isAllSelected ? true : (isIndeterminate ? 'indeterminate' : false)"
          title="全选/取消当前页"
          @update:checked="$emit('toggleAll')"
        />
        <template v-else>
          <span class="group-hover/header:hidden">序号</span>
          <Checkbox
            class="hidden group-hover/header:flex"
            :checked="false"
            title="全选当前页"
            @update:checked="$emit('toggleAll')"
          />
        </template>
      </div>
      <span class="w-8 shrink-0 text-center whitespace-nowrap">类型</span>
      <span
        class="w-44 lg:w-56 shrink-0 flex items-center gap-1 cursor-pointer select-none hover:text-foreground transition-colors"
        @click="$emit('toggleSort', 'name')"
      >
        <span>名称</span>
        <ArrowUpDown v-if="sortBy !== 'name'" class="h-3 w-3 opacity-40 shrink-0" />
        <ArrowUp v-else-if="order === 'asc'" class="h-3 w-3 text-primary shrink-0" />
        <ArrowDown v-else class="h-3 w-3 text-primary shrink-0" />
      </span>

      <span v-if="filterType !== TASK_TYPE.APP && !filterType?.startsWith('app:')" class="hidden md:inline-block w-24 lg:w-32 shrink-0">执行位置</span>

      <span class="flex-1 min-w-0 flex items-center gap-1.5 shrink-0 overflow-hidden whitespace-nowrap">
        <template v-if="filterType === TASK_TYPE.APP || filterType?.startsWith('app:')">
          <Package class="h-3.5 w-3.5 opacity-50 text-emerald-500 shrink-0" />
          <span class="truncate">应用场景</span>
        </template>
        <template v-else-if="filterType === TASK_TYPE.REPO">
          <GitBranch class="h-3.5 w-3.5 opacity-50 shrink-0" />
          <span class="truncate">仓库地址</span>
        </template>
        <template v-else>
          <Terminal class="h-3.5 w-3.5 opacity-50 shrink-0" />
          <span class="truncate">命令内容</span>
        </template>
      </span>

      <span class="hidden lg:inline-block w-24 lg:w-28 shrink-0 whitespace-nowrap">
        {{ filterType === TASK_TYPE.REPO ? '同步周期' : '定时规则' }}
      </span>

      <span class="hidden md:flex w-28 lg:w-36 shrink-0 items-center gap-1 cursor-pointer select-none hover:text-foreground transition-colors whitespace-nowrap" @click="$emit('toggleSort', 'next_run')">
        <span>执行时间</span>
        <ArrowUpDown v-if="sortBy !== 'next_run'" class="h-3 w-3 opacity-40 shrink-0" />
        <ArrowUp v-else-if="order === 'asc'" class="h-3 w-3 text-primary shrink-0" />
        <ArrowDown v-else class="h-3 w-3 text-primary shrink-0" />
      </span>

      <span class="w-12 lg:w-14 shrink-0 text-center flex items-center justify-center gap-1 cursor-pointer select-none hover:text-foreground transition-colors whitespace-nowrap" @click="$emit('toggleSort', 'enabled')">
        <span>状态</span>
        <ArrowUpDown v-if="sortBy !== 'enabled'" class="h-3 w-3 opacity-40 shrink-0" />
        <ArrowUp v-else-if="order === 'asc'" class="h-3 w-3 text-primary shrink-0" />
        <ArrowDown v-else class="h-3 w-3 text-primary shrink-0" />
      </span>
      <span class="w-20 lg:w-24 shrink-0 text-center whitespace-nowrap">操作</span>
    </div>

    <!-- 列表数据 -->
    <div class="divide-y text-sm">
      <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-12">
        暂无符合条件的任务或实体
      </div>
      <div v-for="(task, index) in tasks" :key="task.id" class="w-full">
        <!-- 业务场景 A: 白虎应用主实体 -->
        <AppTaskRow
          v-if="task.type === TASK_TYPE.APP"
          :task="task"
          :index="index"
          :total="total"
          :current-page="currentPage"
          :page-size="pageSize"
          :executing-task-id="executingTaskId"
          :agent-map="agentMap"
          :selected="selectedIds.includes(task.id)"
          :has-selection="hasSelection"
          :update-info="updateMap[task.id] || null"
          @run="$emit('runTask', $event)"
          @view-logs="$emit('viewLogs', $event)"
          @filter-by-app="$emit('filterByApp', $event)"
          @edit-app="$emit('openEditApp', $event)"
          @uninstall-app="$emit('uninstallApp', $event)"
          @toggle-task="(t, val) => $emit('toggleTask', t, val)"
          @share-app="$emit('shareApp', $event)"
          @toggle-select="$emit('toggleSelect', $event)"
          @open-detail="$emit('openAppDetail', $event)"
        />

        <!-- 业务场景 B: Git 仓库同步任务 -->
        <RepoTaskRow
          v-else-if="task.type === TASK_TYPE.REPO"
          :task="task"
          :index="index"
          :total="total"
          :current-page="currentPage"
          :page-size="pageSize"
          :executing-task-id="executingTaskId"
          :agent-map="agentMap"
          :selected="selectedIds.includes(task.id)"
          :has-selection="hasSelection"
          @run="$emit('runTask', $event)"
          @view-logs="$emit('viewLogs', $event)"
          @edit="$emit('openEditTask', $event)"
          @toggle-task="(t, en) => $emit('toggleTask', t, en)"
          @toggle-pin="$emit('togglePin', $event)"
          @export-cmd="$emit('openExportDialog', $event)"
          @duplicate="$emit('duplicateTask', $event)"
          @delete="$emit('confirmDelete', $event)"
          @toggle-select="$emit('toggleSelect', $event)"
        />

        <!-- 业务场景 C: 普通脚本任务 -->
        <NormalTaskRow
          v-else
          :task="task"
          :index="index"
          :total="total"
          :current-page="currentPage"
          :page-size="pageSize"
          :executing-task-id="executingTaskId"
          :agent-map="agentMap"
          :selected="selectedIds.includes(task.id)"
          :has-selection="hasSelection"
          @run="$emit('runTask', $event)"
          @view-logs="$emit('viewLogs', $event)"
          @edit="$emit('openEditTask', $event)"
          @toggle-task="(t, en) => $emit('toggleTask', t, en)"
          @toggle-pin="$emit('togglePin', $event)"
          @dep-install="$emit('depInstall', $event)"
          @duplicate="$emit('duplicateTask', $event)"
          @delete="$emit('confirmDelete', $event)"
          @toggle-select="$emit('toggleSelect', $event)"
        />
      </div>
    </div>

    <!-- 底部嵌入式分页栏插槽 -->
    <slot name="pagination" />
  </div>
</template>
