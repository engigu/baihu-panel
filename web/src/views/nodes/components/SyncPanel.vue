<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { toast } from 'vue-sonner'
import * as interconnectApi from '@/api/interconnect'
import { api, type EnvVar, type Task } from '@/api'
import { Network, Search, HardDrive, Check, AlertCircle, RefreshCw, ChevronDown, ChevronRight } from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

const props = defineProps<{
  nodes: interconnectApi.InterconnectNode[]
}>()

const activeSyncType = ref<'task' | 'env'>('task')

const loading = ref(false)
const syncing = ref(false)

// Data
const localTasks = ref<Task[]>([])
const localEnvs = ref<EnvVar[]>([])

// Selection
const selectedNodes = ref<string[]>([])
const selectedTasks = ref<string[]>([])
const selectedEnvsIds = ref<string[]>([])

// Search
const taskSearch = ref('')
const envSearch = ref('')

// Load local data
async function loadLocalData() {
  loading.value = true
  try {
    const taskRes = await api.tasks.list({ page: 1, page_size: 9999 })
    localTasks.value = taskRes.data
    
    const envRes = await api.env.list({ page: 1, page_size: 9999 })
    localEnvs.value = envRes.data
  } catch (error) {
    toast.error('获取本地数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadLocalData()
})

const expandedRepos = ref<Record<string, boolean>>({})

function toggleRepoExpand(repoId: string) {
  const idStr = String(repoId)
  if (expandedRepos.value[idStr] === false) {
    expandedRepos.value[idStr] = true
  } else {
    expandedRepos.value[idStr] = false
  }
}

const taskTree = computed(() => {
  const kw = taskSearch.value.toLowerCase().trim()
  
  const childrenMap: Record<string, Task[]> = {}
  const rootTasks: Task[] = []
  
  localTasks.value.forEach(t => {
    if (t.type === 'repo') {
      rootTasks.push(t)
    } else if (t.repo_task_id) {
      const repoId = t.repo_task_id
      if (!childrenMap[repoId]) {
        childrenMap[repoId] = []
      }
      childrenMap[repoId]!.push(t)
    } else {
      rootTasks.push(t)
    }
  })
  
  const result: { parent: Task; children: Task[] }[] = []
  
  rootTasks.forEach(parent => {
    const children = childrenMap[parent.id] || []
    const parentMatches = !kw || parent.name.toLowerCase().includes(kw) || parent.command.toLowerCase().includes(kw)
    const matchingChildren = children.filter(c => !kw || c.name.toLowerCase().includes(kw) || c.command.toLowerCase().includes(kw))
    
    if (parentMatches || matchingChildren.length > 0) {
      const childrenToDisplay = parentMatches ? children : matchingChildren
      result.push({
        parent,
        children: childrenToDisplay
      })
      // If parent didn't match but child did, auto-expand
      if (kw && !parentMatches && matchingChildren.length > 0) {
        expandedRepos.value[String(parent.id)] = true
      }
    }
  })
  
  return result
})

const totalVisibleTasks = computed(() => {
  const list: Task[] = []
  taskTree.value.forEach(item => {
    list.push(item.parent)
    item.children.forEach(c => {
      list.push(c)
    })
  })
  return list
})

const filteredEnvs = computed(() => {
  if (!envSearch.value) return localEnvs.value
  const kw = envSearch.value.toLowerCase()
  return localEnvs.value.filter((e: EnvVar) => e.name.toLowerCase().includes(kw) || e.value.toLowerCase().includes(kw))
})

function toggleNode(nodeId: any) {
  const idStr = String(nodeId)
  const index = selectedNodes.value.indexOf(idStr)
  if (index > -1) {
    selectedNodes.value.splice(index, 1)
  } else {
    selectedNodes.value.push(idStr)
  }
}

// Check if all visible tasks are selected
const isAllTasksSelected = computed(() => {
  const visible = totalVisibleTasks.value
  if (visible.length === 0) return false
  return visible.every(t => selectedTasks.value.includes(String(t.id)))
})

function toggleAllNodes() {
  if (selectedNodes.value.length === props.nodes.length && props.nodes.length > 0) {
    selectedNodes.value = []
  } else {
    selectedNodes.value = props.nodes.map(n => String(n.id))
  }
}

function toggleTask(id: any) {
  const idStr = String(id)
  const task = localTasks.value.find(t => String(t.id) === idStr)
  if (!task) return

  const isCurrentlySelected = selectedTasks.value.includes(idStr)

  if (task.type === 'repo') {
    if (isCurrentlySelected) {
      selectedTasks.value = selectedTasks.value.filter(x => x !== idStr)
      const children = localTasks.value.filter(t => t.repo_task_id === task.id)
      const childIds = children.map(c => String(c.id))
      selectedTasks.value = selectedTasks.value.filter(x => !childIds.includes(x))
    } else {
      selectedTasks.value.push(idStr)
      const children = localTasks.value.filter(t => t.repo_task_id === task.id)
      children.forEach(c => {
        const cid = String(c.id)
        if (!selectedTasks.value.includes(cid)) {
          selectedTasks.value.push(cid)
        }
      })
    }
  } else {
    if (isCurrentlySelected) {
      selectedTasks.value = selectedTasks.value.filter(x => x !== idStr)
      if (task.repo_task_id) {
        selectedTasks.value = selectedTasks.value.filter(x => x !== String(task.repo_task_id))
      }
    } else {
      selectedTasks.value.push(idStr)
      if (task.repo_task_id) {
        const parentIdStr = String(task.repo_task_id)
        const siblings = localTasks.value.filter(t => t.repo_task_id === task.repo_task_id)
        const allSiblingsChecked = siblings.every(s => selectedTasks.value.includes(String(s.id)))
        if (allSiblingsChecked && !selectedTasks.value.includes(parentIdStr)) {
          selectedTasks.value.push(parentIdStr)
        }
      }
    }
  }
}

function toggleAllTasks() {
  const visible = totalVisibleTasks.value
  if (isAllTasksSelected.value) {
    const visibleIds = visible.map(t => String(t.id))
    selectedTasks.value = selectedTasks.value.filter(id => !visibleIds.includes(id))
  } else {
    visible.forEach(t => {
      const idStr = String(t.id)
      if (!selectedTasks.value.includes(idStr)) {
        selectedTasks.value.push(idStr)
      }
    })
  }
}

function toggleEnv(id: any) {
  const idStr = String(id)
  const index = selectedEnvsIds.value.indexOf(idStr)
  if (index > -1) {
    selectedEnvsIds.value.splice(index, 1)
  } else {
    selectedEnvsIds.value.push(idStr)
  }
}

function toggleAllEnvs() {
  if (selectedEnvsIds.value.length === filteredEnvs.value.filter((e: EnvVar) => e.type !== 'secret').length && filteredEnvs.value.length > 0) {
    selectedEnvsIds.value = []
  } else {
    selectedEnvsIds.value = filteredEnvs.value.filter((e: EnvVar) => e.type !== 'secret').map((e: EnvVar) => String(e.id))
  }
}

const showConfirmDialog = ref(false)
const previewLoading = ref(false)
const previewData = ref<any>(null)

async function handleSync() {
  if (selectedNodes.value.length === 0) {
    toast.warning('请先选择目标节点')
    return
  }

  if (activeSyncType.value === 'task' && selectedTasks.value.length === 0) {
    toast.warning('请先选择要同步的任务')
    return
  }
  
  if (activeSyncType.value === 'env' && selectedEnvsIds.value.length === 0) {
    toast.warning('请先选择要同步的变量')
    return
  }

  previewLoading.value = true
  showConfirmDialog.value = true
  previewData.value = null

  try {
    const reqData = activeSyncType.value === 'task' 
      ? { task_ids: selectedTasks.value } 
      : { env_ids: selectedEnvsIds.value }
    const res = await api.system.export(reqData)
    previewData.value = res
  } catch (error) {
    toast.error('获取预览数据失败')
    showConfirmDialog.value = false
  } finally {
    previewLoading.value = false
  }
}

async function confirmSync() {
  const targetNodeIds = selectedNodes.value
  syncing.value = true
  
  try {
    if (activeSyncType.value === 'task') {
      const tasksToSync = localTasks.value.filter((t: Task) => selectedTasks.value.includes(t.id))
      const res = await interconnectApi.syncTask(targetNodeIds, tasksToSync)
      showSyncResult(res)
    } else {
      const allSelectedEnvs = localEnvs.value.filter((e: EnvVar) => selectedEnvsIds.value.includes(e.id))
      const envsToSync = allSelectedEnvs.filter((e: EnvVar) => e.type !== 'secret')
      
      if (allSelectedEnvs.length > envsToSync.length) {
        toast.info('已自动过滤机密变量，因下发无意义')
      }
      
      if (envsToSync.length > 0) {
        const res = await interconnectApi.syncEnv(targetNodeIds, envsToSync)
        showSyncResult(res)
      }
    }
  } catch (error) {
    toast.error('下发请求失败')
  } finally {
    syncing.value = false
    showConfirmDialog.value = false
  }
}

function showSyncResult(res: any[]) {
  const targetNodeIds = selectedNodes.value
  const successCount = res.filter((r: any) => r.success).length
  if (successCount === targetNodeIds.length) {
    toast.success('全部分发成功')
  } else {
    toast.warning(`成功: ${successCount}, 失败: ${targetNodeIds.length - successCount}`)
  }
}
</script>

<template>
  <div class="flex flex-col lg:flex-row gap-6 h-[calc(100vh-16rem)] min-h-[500px]">
    <!-- 左侧：节点选择 -->
    <div class="w-full lg:w-1/3 flex flex-col border rounded-lg bg-card overflow-hidden h-[180px] lg:h-full shrink-0">
      <div class="p-3 border-b bg-muted/20 flex items-center justify-between">
        <h3 class="font-medium text-sm flex items-center gap-2">
          <HardDrive class="w-4 h-4 text-muted-foreground" />
          目标节点 ({{ selectedNodes.length }}/{{ nodes.length }})
        </h3>
        <Button variant="ghost" size="sm" class="h-6 text-xs px-2" @click="toggleAllNodes">
          {{ selectedNodes.length === nodes.length && nodes.length > 0 ? '全不选' : '全选' }}
        </Button>
      </div>
      <div class="flex-1 overflow-y-auto p-2">
        <div v-if="nodes.length === 0" class="text-center py-8 text-sm text-muted-foreground">
          暂无子节点
        </div>
        <div v-else class="space-y-1">
          <div v-for="node in nodes" :key="node.id" class="flex items-center gap-2 py-1 px-1.5 hover:bg-muted/50 rounded-md cursor-pointer select-none group/node" @click="toggleNode(node.id)">
            <div 
              class="size-3.5 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
              :class="selectedNodes.includes(String(node.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/node:border-muted-foreground/50 scale-95'"
            >
              <Check v-if="selectedNodes.includes(String(node.id))" class="size-2.5 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
            </div>
            <div class="flex items-center flex-1 min-w-0 justify-between gap-3">
              <span class="text-xs font-medium truncate">{{ node.name }}</span>
              <span class="text-[10px] text-muted-foreground truncate font-mono max-w-[50%]">{{ node.url }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
 
    <!-- 右侧：数据选择 -->
    <div class="w-full lg:w-2/3 flex flex-col border rounded-lg bg-card overflow-hidden flex-1 lg:h-full min-h-[250px] lg:min-h-0">
      <div class="p-3 border-b bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex bg-muted p-1 rounded-md">
          <button 
            class="px-3 py-1.5 text-xs font-medium rounded-sm transition-colors"
            :class="activeSyncType === 'task' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'"
            @click="activeSyncType = 'task'"
          >
            任务配置
          </button>
          <button 
            class="px-3 py-1.5 text-xs font-medium rounded-sm transition-colors"
            :class="activeSyncType === 'env' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'"
            @click="activeSyncType = 'env'"
          >
            变量配置
          </button>
        </div>

        <div class="flex items-center gap-2 w-full justify-end sm:w-auto">
          <div class="relative flex-grow sm:flex-grow-0 sm:w-48">
            <Search class="w-3.5 h-3.5 absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input v-model="taskSearch" v-if="activeSyncType === 'task'" placeholder="搜索任务..." class="h-8 pl-7 text-xs" />
            <Input v-model="envSearch" v-if="activeSyncType === 'env'" placeholder="搜索变量..." class="h-8 pl-7 text-xs" />
          </div>
          <Button class="h-8 gap-1.5" size="sm" @click="handleSync" :disabled="syncing">
            <Network class="w-3.5 h-3.5" :class="{'animate-pulse': syncing}" />
            {{ syncing ? '正在下发...' : '下发' }}
          </Button>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto p-2">
        <!-- 任务列表 -->
        <div v-if="activeSyncType === 'task'" key="task-list" class="space-y-1">
          <div class="flex items-center justify-between px-2 py-1 mb-2">
            <span class="text-xs text-muted-foreground">已选 {{ selectedTasks.length }} 项</span>
            <Button variant="ghost" size="sm" class="h-6 text-xs px-2" @click="toggleAllTasks">
              {{ isAllTasksSelected && totalVisibleTasks.length > 0 ? '全不选' : '全选' }}
            </Button>
          </div>
          <div v-if="loading" class="text-center py-8 text-sm text-muted-foreground">加载中...</div>
          <div v-else-if="taskTree.length === 0" class="text-center py-8 text-sm text-muted-foreground">暂无任务</div>
          <div v-for="item in taskTree" :key="item.parent.id" class="space-y-1">
            <!-- 父节点 (仓库或独立任务) -->
            <div class="flex items-center gap-2 py-1 px-1.5 hover:bg-muted/50 rounded-md cursor-pointer select-none group/task" @click="toggleTask(item.parent.id)">
              <!-- 展开/折叠按钮 -->
              <div v-if="item.parent.type === 'repo'" class="w-3.5 h-3.5 flex items-center justify-center shrink-0" @click.stop="toggleRepoExpand(item.parent.id)">
                <ChevronDown v-if="expandedRepos[item.parent.id] !== false" class="w-3 h-3 text-muted-foreground hover:text-foreground" />
                <ChevronRight v-else class="w-3 h-3 text-muted-foreground hover:text-foreground" />
              </div>
              <div v-else class="w-3.5 h-3.5 shrink-0"></div>

              <div 
                class="size-3.5 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
                :class="selectedTasks.includes(String(item.parent.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/task:border-muted-foreground/50 scale-95'"
              >
                <Check v-if="selectedTasks.includes(String(item.parent.id))" class="size-2.5 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
              </div>
              <div class="flex items-center flex-1 min-w-0 justify-between gap-4">
                <div class="flex items-center gap-1.5 min-w-0">
                  <span class="text-xs font-medium truncate">{{ item.parent.name }}</span>
                  <span class="text-[9px] bg-muted px-1.5 py-0.2 rounded text-muted-foreground shrink-0">{{ item.parent.type === 'task' ? '任务' : '仓库' }}</span>
                </div>
                <span class="text-[10px] text-muted-foreground truncate font-mono max-w-[60%]">{{ item.parent.command }}</span>
              </div>
            </div>

            <!-- 子节点 (归属于该仓库的任务) -->
            <div v-if="item.parent.type === 'repo' && expandedRepos[item.parent.id] !== false && item.children.length > 0" class="pl-5 space-y-1 border-l ml-3.5 pb-0.5">
              <div v-for="child in item.children" :key="child.id" class="flex items-center gap-2 py-0.5 px-1.5 hover:bg-muted/50 rounded-md cursor-pointer select-none group/subtask" @click="toggleTask(child.id)">
                <div 
                  class="size-3.5 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
                  :class="selectedTasks.includes(String(child.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/subtask:border-muted-foreground/50 scale-95'"
                >
                  <Check v-if="selectedTasks.includes(String(child.id))" class="size-2.5 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
                </div>
                <div class="flex items-center flex-1 min-w-0 justify-between gap-4">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <span class="text-xs font-medium truncate">{{ child.name }}</span>
                    <span class="text-[9px] bg-muted px-1.5 py-0.2 rounded text-muted-foreground shrink-0">任务</span>
                  </div>
                  <span class="text-[10px] text-muted-foreground truncate font-mono max-w-[60%]">{{ child.command }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 变量列表 -->
        <div v-if="activeSyncType === 'env'" key="env-list" class="space-y-1">
          <div class="flex items-center justify-between px-2 py-1 mb-2">
            <span class="text-xs text-muted-foreground">已选 {{ selectedEnvsIds.length }} 项</span>
            <Button variant="ghost" size="sm" class="h-6 text-xs px-2" @click="toggleAllEnvs">
              {{ selectedEnvsIds.length === filteredEnvs.length && filteredEnvs.length > 0 ? '全不选' : '全选' }}
            </Button>
          </div>
          <div v-if="loading" class="text-center py-8 text-sm text-muted-foreground">加载中...</div>
          <div v-else-if="filteredEnvs.length === 0" class="text-center py-8 text-sm text-muted-foreground">暂无变量</div>
          <div v-for="env in filteredEnvs" :key="env.id" 
               class="flex items-center gap-2 py-1 px-1.5 rounded-md select-none group/env" 
               :class="env.type === 'secret' ? 'opacity-50 cursor-not-allowed bg-muted/20' : 'hover:bg-muted/50 cursor-pointer'"
               @click="env.type === 'secret' ? null : toggleEnv(env.id)">
            <div 
              class="size-3.5 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
              :class="[
                selectedEnvsIds.includes(String(env.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/env:border-muted-foreground/50 scale-95',
                env.type === 'secret' ? 'opacity-50 bg-muted border-muted-foreground/30' : ''
              ]"
            >
              <Check v-if="selectedEnvsIds.includes(String(env.id))" class="size-2.5 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
            </div>
            <div class="flex items-center flex-1 min-w-0 justify-between gap-4">
              <div class="flex items-center gap-1.5 min-w-0">
                <span class="text-xs font-medium truncate">{{ env.name }}</span>
                <span v-if="env.type" class="text-[9px] bg-muted px-1.5 py-0.2 rounded shrink-0" :class="env.type === 'secret' ? 'text-amber-500 bg-amber-500/10' : 'text-muted-foreground'">{{ env.type === 'secret' ? '机密' : '常量' }}</span>
              </div>
              <span class="text-[10px] text-muted-foreground truncate font-mono max-w-[60%]">{{ env.type === 'secret' ? '********' : env.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 确认同步弹窗 -->
    <Dialog v-model:open="showConfirmDialog">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>确认同步数据</DialogTitle>
          <DialogDescription>
            以下是系统自动解析后即将全量同步到目标节点的所有关联数据。
          </DialogDescription>
        </DialogHeader>

        <div class="py-4">
          <div v-if="previewLoading" class="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <RefreshCw class="w-8 h-8 animate-spin mb-4" />
            <p>正在智能解析关联数据...</p>
          </div>
          <div v-else-if="previewData" class="space-y-4">
            <div class="rounded-lg border bg-card text-card-foreground p-3">
              <div class="flex items-center gap-2 font-medium tracking-tight mb-2 text-xs">
                <AlertCircle class="h-4 w-4 text-primary" />
                即将同步以下数据：
              </div>
              
              <div class="max-h-[220px] overflow-y-auto space-y-3 pr-1 text-xs">
                <!-- 任务列表 -->
                <div v-if="previewData.tasks && previewData.tasks.length > 0">
                  <div class="font-semibold text-muted-foreground mb-1">任务 ({{ previewData.tasks.length }})</div>
                  <div class="space-y-1 pl-2 border-l-2 border-primary/30">
                    <div v-for="t in previewData.tasks" :key="t.id" class="flex justify-between items-center py-1 border-b border-muted/30 last:border-0 pb-1 mb-1 last:pb-0 last:mb-0">
                      <div class="flex flex-col min-w-0 max-w-[60%]">
                        <span class="font-medium truncate">{{ t.name }}</span>
                        <div v-if="t.tags" class="flex flex-wrap gap-1 mt-0.5">
                          <span v-for="tag in t.tags.split(',')" :key="tag" class="text-[9px] bg-primary/10 text-primary px-1 py-0.2 rounded">{{ tag }}</span>
                        </div>
                      </div>
                      <span class="text-[10px] text-muted-foreground font-mono truncate max-w-[38%]">{{ t.command }}</span>
                    </div>
                  </div>
                </div>

                <!-- 变量列表 -->
                <div v-if="previewData.envs && previewData.envs.length > 0">
                  <div class="font-semibold text-muted-foreground mb-1">环境变量 ({{ previewData.envs.length }})</div>
                  <div class="space-y-1 pl-2 border-l-2 border-amber-500/30">
                    <div v-for="e in previewData.envs" :key="e.id" class="flex justify-between items-center py-0.5">
                      <span class="font-medium truncate max-w-[50%]">{{ e.name }}</span>
                      <span class="text-[10px] text-muted-foreground font-mono truncate max-w-[45%]">{{ e.type === 'secret' ? '********' : e.value }}</span>
                    </div>
                  </div>
                </div>

                <!-- 标签列表 -->
                <div v-if="previewData.tags && previewData.tags.length > 0">
                  <div class="font-semibold text-muted-foreground mb-1">标签 ({{ previewData.tags.length }})</div>
                  <div class="space-y-1 pl-2 border-l-2 border-teal-500/30 flex flex-wrap gap-1.5 py-0.5">
                    <span v-for="tag in previewData.tags" :key="tag.id" class="text-[10px] bg-teal-500/10 text-teal-600 dark:text-teal-400 px-1.5 py-0.5 rounded font-medium">
                      {{ tag.name }}
                    </span>
                  </div>
                </div>

                <!-- 通知绑定 -->
                <div v-if="previewData.bindings && previewData.bindings.length > 0">
                  <div class="font-semibold text-muted-foreground mb-1">通知规则 ({{ previewData.bindings.length }})</div>
                  <div class="space-y-1 pl-2 border-l-2 border-emerald-500/30">
                    <div v-for="b in previewData.bindings" :key="b.id" class="flex justify-between items-center py-0.5">
                      <span class="font-medium truncate max-w-[50%]">事件: {{ b.event_type }}</span>
                      <span class="text-[10px] text-muted-foreground truncate max-w-[45%]">通道: {{ b.channel_id }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <p class="text-[11px] text-muted-foreground mt-4 leading-normal">
              * 同步将会覆盖目标节点上的同名数据，并且机密变量的实际值不会被下发。
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showConfirmDialog = false" :disabled="syncing">取消</Button>
          <Button @click="confirmSync" :disabled="previewLoading || syncing">
            <RefreshCw v-if="syncing" class="w-4 h-4 mr-2 animate-spin" />
            确认下发
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
