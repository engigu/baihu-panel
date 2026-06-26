<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { toast } from 'vue-sonner'
import * as interconnectApi from '@/api/interconnect'
import { api, type EnvVar, type Task } from '@/api'
import { Network, Search, HardDrive, Check } from 'lucide-vue-next'

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

const filteredTasks = computed(() => {
  if (!taskSearch.value) return localTasks.value
  const kw = taskSearch.value.toLowerCase()
  return localTasks.value.filter((t: Task) => t.name.toLowerCase().includes(kw) || t.command.toLowerCase().includes(kw))
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

function toggleAllNodes() {
  if (selectedNodes.value.length === props.nodes.length && props.nodes.length > 0) {
    selectedNodes.value = []
  } else {
    selectedNodes.value = props.nodes.map(n => String(n.id))
  }
}

function toggleTask(id: any) {
  const idStr = String(id)
  const index = selectedTasks.value.indexOf(idStr)
  if (index > -1) {
    selectedTasks.value.splice(index, 1)
  } else {
    selectedTasks.value.push(idStr)
  }
}

function toggleAllTasks() {
  if (selectedTasks.value.length === filteredTasks.value.length && filteredTasks.value.length > 0) {
    selectedTasks.value = []
  } else {
    selectedTasks.value = filteredTasks.value.map((t: Task) => String(t.id))
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

async function handleSync() {
  if (selectedNodes.value.length === 0) {
    toast.warning('请先选择目标节点')
    return
  }

  const targetNodeIds = selectedNodes.value

  syncing.value = true
  try {
    if (activeSyncType.value === 'task') {
      if (selectedTasks.value.length === 0) {
        toast.warning('请先选择要同步的任务')
        return
      }
      const tasksToSync = localTasks.value.filter((t: Task) => selectedTasks.value.includes(t.id))
      const res = await interconnectApi.syncTask(targetNodeIds, tasksToSync)
      showSyncResult(res)
    } else {
      if (selectedEnvsIds.value.length === 0) {
        toast.warning('请先选择要同步的变量')
        return
      }
      
      const allSelectedEnvs = localEnvs.value.filter((e: EnvVar) => selectedEnvsIds.value.includes(e.id))
      const envsToSync = allSelectedEnvs.filter((e: EnvVar) => e.type !== 'secret')
      
      if (allSelectedEnvs.length > envsToSync.length) {
        toast.info('已自动过滤机密变量，因下发无意义')
      }
      
      if (envsToSync.length === 0) {
        syncing.value = false
        return
      }
      
      const res = await interconnectApi.syncEnv(targetNodeIds, envsToSync)
      showSyncResult(res)
    }
  } catch (error) {
    toast.error('下发请求失败')
  } finally {
    syncing.value = false
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
          <div v-for="node in nodes" :key="node.id" class="flex items-center gap-3 p-2 hover:bg-muted/50 rounded-md cursor-pointer select-none group/node" @click="toggleNode(node.id)">
            <div 
              class="size-4 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
              :class="selectedNodes.includes(String(node.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/node:border-muted-foreground/50 scale-95'"
            >
              <Check v-if="selectedNodes.includes(String(node.id))" class="size-3 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
            </div>
            <div class="flex flex-col flex-1 min-w-0">
              <span class="text-sm font-medium truncate">{{ node.name }}</span>
              <span class="text-xs text-muted-foreground truncate">{{ node.url }}</span>
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
              {{ selectedTasks.length === filteredTasks.length && filteredTasks.length > 0 ? '全不选' : '全选' }}
            </Button>
          </div>
          <div v-if="loading" class="text-center py-8 text-sm text-muted-foreground">加载中...</div>
          <div v-else-if="filteredTasks.length === 0" class="text-center py-8 text-sm text-muted-foreground">暂无任务</div>
          <div v-for="task in filteredTasks" :key="task.id" class="flex items-start gap-3 p-2 hover:bg-muted/50 rounded-md cursor-pointer select-none group/task" @click="toggleTask(task.id)">
            <div 
              class="mt-1 size-4 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
              :class="selectedTasks.includes(String(task.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/task:border-muted-foreground/50 scale-95'"
            >
              <Check v-if="selectedTasks.includes(String(task.id))" class="size-3 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
            </div>
            <div class="flex flex-col flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium truncate">{{ task.name }}</span>
                <span class="text-[10px] bg-muted px-1.5 py-0.5 rounded text-muted-foreground">{{ task.type === 'task' ? '任务' : (task.type === 'repo' ? '仓库' : task.type) }}</span>
              </div>
              <span class="text-xs text-muted-foreground truncate font-mono mt-0.5">{{ task.command }}</span>
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
               class="flex items-start gap-3 p-2 rounded-md select-none group/env" 
               :class="env.type === 'secret' ? 'opacity-50 cursor-not-allowed bg-muted/20' : 'hover:bg-muted/50 cursor-pointer'"
               @click="env.type === 'secret' ? null : toggleEnv(env.id)">
            <div 
              class="mt-1 size-4 rounded-[4px] border flex items-center justify-center shrink-0 transition-all duration-200 shadow-xs"
              :class="[
                selectedEnvsIds.includes(String(env.id)) ? 'bg-primary border-primary text-primary-foreground scale-100' : 'bg-transparent border-input group-hover/env:border-muted-foreground/50 scale-95',
                env.type === 'secret' ? 'opacity-50 bg-muted border-muted-foreground/30' : ''
              ]"
            >
              <Check v-if="selectedEnvsIds.includes(String(env.id))" class="size-3 stroke-[3.5] animate-in zoom-in-75 fade-in duration-150" />
            </div>
            <div class="flex flex-col flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium truncate">{{ env.name }}</span>
                <span v-if="env.type" class="text-[10px] bg-muted px-1.5 py-0.5 rounded" :class="env.type === 'secret' ? 'text-amber-500 bg-amber-500/10' : 'text-muted-foreground'">{{ env.type === 'secret' ? '机密' : (env.type === 'normal' ? '常量' : env.type) }}</span>
              </div>
              <span class="text-xs text-muted-foreground truncate font-mono mt-0.5">{{ env.type === 'secret' ? '********' : env.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
