<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type Workflow, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { ArrowLeft, Save, GripVertical, GripHorizontal, Settings2, History, CheckCircle2, XCircle, Loader2, Map as MapIcon, Play, Clock } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

// Vue Flow Core
import { VueFlow, useVueFlow, MarkerType, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'

// Styles for vue-flow
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

const route = useRoute()
const router = useRouter()
const workflow = ref<Workflow | null>(null)
const tasks = ref<Task[]>([]) // All available tasks to drag

const { onConnect, addEdges, addNodes, onNodeClick, onEdgeClick, onPaneClick, removeNodes, removeEdges } = useVueFlow()

const elements = ref<any[]>([])
const selectedElement = ref<any>(null)
const selectedElementType = ref<'node' | 'edge' | null>(null)

// Setup vue flow connection behavior
onNodeClick((e) => {
  selectedElement.value = e.node
  selectedElementType.value = 'node'
})

onEdgeClick((e) => {
  selectedElement.value = e.edge
  selectedElementType.value = 'edge'
})

onPaneClick(() => {
  selectedElement.value = null
  selectedElementType.value = null
})

function updateEdgeCondition(val: any) {
  if (selectedElementType.value === 'edge' && selectedElement.value && typeof val === 'string') {
    if (!selectedElement.value.data) selectedElement.value.data = {}
    selectedElement.value.data.condition = val
    
    // Update label to display on canvas
    if (val === 'success') {
      selectedElement.value.label = '成功 (Success)'
      selectedElement.value.style = { stroke: '#10b981', strokeWidth: 2 } // green
    } else if (val === 'failed') {
      selectedElement.value.label = '失败 (Failed)'
      selectedElement.value.style = { stroke: '#ef4444', strokeWidth: 2 } // red
    } else {
      selectedElement.value.label = '总是 (Always)'
      selectedElement.value.style = { stroke: '#94a3b8', strokeWidth: 2 }
    }
  }
}

function deleteSelected() {
  if (selectedElementType.value === 'node') {
    removeNodes([selectedElement.value.id])
  } else if (selectedElementType.value === 'edge') {
    removeEdges([selectedElement.value.id])
  }
  selectedElement.value = null
  selectedElementType.value = null
}

// Generate a random ID for nodes
const getId = () => `node-${Date.now()}-${Math.floor(Math.random() * 1000)}`

// Setup vue flow connection behavior
onConnect((params: Connection) => {
  addEdges([{
    ...params,
    id: `edge-${params.source}-${params.target}-${Date.now()}`,
    label: '总是 (Always)',
    data: { 
      condition: 'always',
      nodeType: (findNode(params.target)?.data as any)?.nodeType || 'task'
    },
    type: 'smoothstep',
    markerEnd: MarkerType.ArrowClosed,
  }])
})

const { findNode } = useVueFlow()

async function loadWorkflow() {
  const wid = route.params.id as string
  try {
    workflow.value = await api.workflows.get(wid)
    if (workflow.value.flow_data) {
      try {
        const parsed = JSON.parse(workflow.value.flow_data)
        elements.value = [...(parsed.nodes || []), ...(parsed.edges || [])]
      } catch (e) {
        toast.error('解析流程数据失败')
      }
    }
  } catch {
    toast.error('系统出错加载工作流')
    router.push('/workflows')
  }
}

async function loadTasks() {
  try {
    const res = await api.tasks.list({ page: 1, page_size: 500 }) // Load max tasks possible
    tasks.value = res.data
  } catch {
    toast.error('加载可用任务失败')
  }
}

// Handling Drag & Drop from Sidebar
function onDragStart(event: DragEvent, task: Task) {
  if (event.dataTransfer) {
    event.dataTransfer.setData('application/vueflow', JSON.stringify(task))
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

function onDrop(event: DragEvent) {
  event.preventDefault()
  const data = event.dataTransfer?.getData('application/vueflow')
  if (!data) return

  const item = JSON.parse(data)
  const isControl = item.type === 'control'
  
  // Calculate relative position based on mouse drop
  const reactFlowBounds = (event.target as Element).getBoundingClientRect()
  const position = {
    x: event.clientX - reactFlowBounds.left - 50,
    y: event.clientY - reactFlowBounds.top - 20,
  }

  const newNode = {
    id: getId(),
    type: isControl ? 'output' : 'default', // Using 'output' node style for control nodes visually or custom ones
    position,
    label: item.name,
    data: { 
      taskId: item.id,
      nodeType: item.type || 'task',
      controlType: item.controlType || ''
    },
    // Customize style for control nodes
    style: isControl ? { background: '#fef3c7', border: '1px solid #f59e0b', color: '#92400e' } : {}
  }

  addNodes([newNode])
}

// Check if graph has cycles
function hasCycle(nodes: any[], edges: any[]): boolean {
  const adjList = new globalThis.Map<string, string[]>()
  nodes.forEach(n => adjList.set(n.id, []))
  edges.forEach(e => {
    if (adjList.has(e.source)) {
      adjList.get(e.source)!.push(e.target)
    }
  })

  const visited = new Set<string>()
  const recStack = new Set<string>()

  function dfs(nodeId: string): boolean {
    if (recStack.has(nodeId)) return true
    if (visited.has(nodeId)) return false

    visited.add(nodeId)
    recStack.add(nodeId)

    const neighbors = adjList.get(nodeId) || []
    for (const neighbor of neighbors) {
      if (dfs(neighbor)) return true
    }

    recStack.delete(nodeId)
    return false
  }

  for (const node of nodes) {
    if (dfs(node.id)) return true
  }

  return false
}

async function saveFlow() {
  if (!workflow.value) return
  
  const currentNodes = elements.value.filter(el => !el.source)
  const currentEdges = elements.value.filter(el => el.source)

  // 1. DAG 有效性检查（死循环检测）
  if (hasCycle(currentNodes, currentEdges)) {
    toast.error('保存失败：流程图存在死循环回路，请检查连线！')
    return
  }

  // Get all current nodes and edges from VueFlow instance
  const payload = {
    flow_data: JSON.stringify({
      nodes: currentNodes,
      edges: currentEdges
    })
  }
  
  try {
    await api.workflows.update(workflow.value.id, payload)
    toast.success('工作流编排保存成功')
  } catch {
    toast.error('保存失败')
  }
}

const isRunning = ref(false)
const globalEnvs = ref<string[]>([])
const newEnv = ref('')

function addEnv() {
  if (newEnv.value && newEnv.value.includes('=')) {
    globalEnvs.value.push(newEnv.value.trim())
    newEnv.value = ''
  } else {
    toast.error('格式错误，请使用 KEY=VALUE')
  }
}

function removeEnv(index: number) {
  globalEnvs.value.splice(index, 1)
}

async function triggerRun() {
  if (!workflow.value) return
  isRunning.value = true
  try {
    await api.workflows.run(workflow.value.id, globalEnvs.value)
    toast.success('工作流启动成功 (Manual Trigger)')
    setTimeout(loadRuns, 2000)
  } catch (e: any) {
    toast.error('启动失败: ' + (e.message || ''))
  } finally {
    isRunning.value = false
  }
}

const runHistory = ref<Array<{runId: string, startTime: string, logs: any[]}>>([])
const selectedRunId = ref<string | null>(null)

async function loadRuns() {
  const wid = route.params.id as string
  try {
    const res = await api.logs.list({ workflow_id: wid, page_size: 200 })
    const logs = res.data || []
    const grouped = new globalThis.Map<string, any[]>()
    for (const log of logs) {
      if (!log.workflow_run_id) continue
      if (!grouped.has(log.workflow_run_id)) {
        grouped.set(log.workflow_run_id, [])
      }
      grouped.get(log.workflow_run_id)!.push(log)
    }
    
    const runs = []
    for (const [runId, list] of grouped.entries()) {
      list.sort((a: any, b: any) => new Date(a.start_time || a.created_at).getTime() - new Date(b.start_time || b.created_at).getTime())
      runs.push({
        runId,
        startTime: list[0].start_time || list[0].created_at,
        logs: list
      })
    }
    runs.sort((a: any, b: any) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime())
    runHistory.value = runs
  } catch(e) { console.error(e) }
}

function selectRun(runId: string | null) {
  selectedRunId.value = runId
  if (!runId) {
    elements.value.forEach(el => {
      if (!el.source) el.class = ''
    })
    return
  }
  const run = runHistory.value.find(r => r.runId === runId)
  if (!run) return
  
  elements.value.forEach(el => {
    if (el.source) return
    const log = run.logs.find(l => String(l.task_id) === String(el.data?.taskId))
    if (!log) {
      el.class = 'opacity-40 grayscale transition-all duration-500'
    } else if (log.status === 'success') {
      el.class = 'border-2 border-[#10b981] bg-[#10b981]/10 shadow-[0_0_15px_rgba(16,185,129,0.3)] transition-all duration-500'
    } else if (log.status === 'failed' || log.status === 'error') {
      el.class = 'border-2 border-[#ef4444] bg-[#ef4444]/10 shadow-[0_0_15px_rgba(239,68,68,0.3)] transition-all duration-500'
    } else {
      el.class = 'border-2 border-[#3b82f6] bg-[#3b82f6]/10 animate-pulse transition-all duration-500'
    }
  })
}

function getRunStatus(run: any) {
  if (!run || !run.logs || run.logs.length === 0) return 'running'
  return run.logs[run.logs.length - 1].status
}

const showMiniMap = ref(false) // Default to hidden or visible? Let's go with visible but user can toggle.
// Actually, let's make it easy to toggle.

onMounted(() => {
  loadTasks()
  loadWorkflow()
  loadRuns()
})
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-80px)] overflow-hidden rounded-md border text-card-foreground shadow-sm">
    <!-- Header -->
    <header class="h-14 border-b bg-muted/40 px-4 flex items-center justify-between shrink-0">
      <div class="flex items-center gap-3">
        <Button variant="ghost" size="icon" @click="router.push('/workflows')" title="返回工作流列表">
          <ArrowLeft class="h-4 w-4" />
        </Button>
        <div>
          <h1 class="font-semibold">{{ workflow?.name || '加载中...' }}</h1>
          <p class="text-[10px] text-muted-foreground">{{ workflow?.description || '拖拽左侧任务节点布置流水线' }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="ghost" size="sm" @click="triggerRun" :disabled="isRunning" class="gap-1.5 h-8 text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50">
          <Loader2 v-if="isRunning" class="h-3.5 w-3.5 animate-spin" />
          <Play v-else class="h-3.5 w-3.5 fill-current" />
          立即触发执行
        </Button>
        <Button variant="outline" size="sm" @click="saveFlow" class="gap-1.5 h-8">
          <Save class="h-3.5 w-3.5" />
          保存流程图
        </Button>
      </div>
    </header>

    <!-- Main Workspace -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Left Sidebar: Task List -->
      <aside class="w-64 border-r bg-background flex flex-col pt-4 overflow-y-auto z-10">
        <div class="px-4 pb-2 mb-2 border-b">
          <h2 class="text-sm font-semibold flex items-center gap-2">
            <GripVertical class="h-4 w-4 text-muted-foreground" />
            系统可用任务
          </h2>
        </div>
        
        <div class="flex-1 overflow-y-auto px-2 space-y-1.5 pb-4">
          <div 
            v-for="task in tasks" 
            :key="task.id"
            :draggable="true"
            @dragstart="onDragStart($event, task)"
            class="px-3 py-2 text-sm border bg-card rounded cursor-grab active:cursor-grabbing hover:border-primary transition-colors flex items-center justify-between group"
          >
            <span class="truncate pr-2 font-medium">{{ task.name }}</span>
            <GripHorizontal class="h-3 w-3 text-muted-foreground opacity-50 group-hover:opacity-100" />
          </div>
        </div>
        <div class="px-4 py-2 mt-4 bg-muted/20 border-y">
          <h2 class="text-[10px] uppercase font-bold text-muted-foreground tracking-widest flex items-center gap-2">
            <Settings2 class="h-3 w-3" />
            流程控制组件
          </h2>
        </div>
        <div class="px-2 py-2 space-y-1.5">
          <div 
            :draggable="true"
            @dragstart="onDragStart($event, { id: -1, name: '延时等待', type: 'control', controlType: 'delay' } as any)"
            class="px-3 py-2 text-sm border bg-amber-50/50 border-amber-200 rounded cursor-grab active:cursor-grabbing hover:border-amber-400 transition-colors flex items-center justify-between group"
          >
            <div class="flex items-center gap-2">
              <Clock class="h-3.5 w-3.5 text-amber-600" />
              <span class="truncate font-medium text-amber-900">延时等待</span>
            </div>
            <GripHorizontal class="h-3 w-3 text-amber-400 opacity-50 group-hover:opacity-100" />
          </div>
        </div>
      </aside>

      <!-- Vue Flow Canvas -->
      <div class="flex-1 relative" @drop="onDrop" @dragover="onDragOver">
        <VueFlow v-model="elements" :default-zoom="1.5" :min-zoom="0.2" :max-zoom="4" fit-view-on-init>
          <Background pattern-color="#aaa" :gap="20" />
          
          <div class="absolute bottom-4 right-4 z-40 flex flex-col gap-2">
            <Button 
              variant="outline" 
              size="icon" 
              class="h-8 w-8 bg-background shadow-sm hover:bg-accent" 
              @click="showMiniMap = !showMiniMap"
              :title="showMiniMap ? '隐藏小地图' : '显示小地图'"
            >
              <MapIcon class="h-4 w-4" :class="showMiniMap ? 'text-primary' : 'text-muted-foreground'" />
            </Button>
          </div>

          <MiniMap v-if="showMiniMap" pannable zoomable />
          <Controls />
        </VueFlow>
      </div>

      <!-- Right Sidebar: Properties & History -->
      <aside class="w-64 border-l bg-background flex flex-col pt-4 z-10 transition-all">
        <div class="px-4 pb-2 mb-4 border-b">
          <h2 class="text-sm font-semibold flex items-center gap-2">
            <template v-if="selectedElement">
              <Settings2 class="h-4 w-4 text-muted-foreground" />
              {{ selectedElementType === 'node' ? '节点设置' : '连线分支设置' }}
            </template>
            <template v-else>
              <History class="h-4 w-4 text-muted-foreground" />
              全局与执行历史
            </template>
          </h2>
        </div>
        
        <div class="px-4 flex-1">
          <!-- Node Properties -->
          <div v-if="selectedElementType === 'node'" class="space-y-4">
            <div class="space-y-2">
              <Label>节点名称</Label>
              <Input v-model="selectedElement.label" />
            </div>

            <!-- Control Node Specific: Delay -->
            <div v-if="selectedElement.data?.controlType === 'delay'" class="space-y-4 pt-4 border-t border-dashed">
              <div class="space-y-2">
                <Label class="text-xs text-amber-700 font-bold flex items-center gap-1">
                  <Clock class="h-3 w-3" />
                  延时设置 (秒)
                </Label>
                <Input 
                  type="number" 
                  :model-value="selectedElement.data?.config || '5'" 
                  @input="(e: any) => selectedElement.data.config = e.target.value"
                  placeholder="例如: 5" 
                />
                <p class="text-[10px] text-muted-foreground">流程到达此节点后将暂停指定秒数后再继续。</p>
              </div>
            </div>

            <div class="pt-4">
              <Button variant="destructive" class="w-full" @click="deleteSelected">移除此节点</Button>
            </div>
          </div>
          
          <!-- Edge Properties -->
          <div v-if="selectedElementType === 'edge'" class="space-y-4">
            <div class="space-y-2">
              <Label>执行分支条件</Label>
              <Select :model-value="selectedElement.data?.condition || 'always'" @update:model-value="updateEdgeCondition">
                <SelectTrigger>
                  <SelectValue placeholder="选择触发条件" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="always">无条件关联 (Always)</SelectItem>
                  <SelectItem value="success">仅前端任务成功 (Success)</SelectItem>
                  <SelectItem value="failed">仅前端任务失败兜底 (Failed)</SelectItem>
                </SelectContent>
              </Select>
              <p class="text-[10px] text-muted-foreground mt-1">设置上游节点在何种完成状态下，会流通并触发下游任务。</p>
              <div class="pt-4">
                <Button variant="destructive" class="w-full" @click="deleteSelected">移除此连线</Button>
              </div>
            </div>
          </div>
          
          <!-- Global Workflow Properties (Default) -->
          <div v-if="!selectedElement" class="space-y-6">
            <div class="space-y-2">
              <Label class="text-xs uppercase font-bold text-muted-foreground tracking-wider">全局环境变量</Label>
              <div class="space-y-2">
                <div v-for="(env, idx) in globalEnvs" :key="idx" class="flex items-center gap-2 bg-muted/50 p-2 rounded text-xs font-mono group">
                  <span class="flex-1 truncate">{{ env }}</span>
                  <button @click="removeEnv(idx)" class="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity">
                    <XCircle class="h-3 w-3" />
                  </button>
                </div>
                <div class="flex gap-2">
                  <Input v-model="newEnv" placeholder="KEY=VALUE" class="h-8 text-xs font-mono" @keyup.enter="addEnv" />
                  <Button variant="outline" size="sm" class="h-8 px-2" @click="addEnv">添加</Button>
                </div>
                <p class="text-[10px] text-muted-foreground">这些变量将作为初始上下文，传递给所有根节点及下游任务。</p>
              </div>
            </div>

            <div class="border-t pt-4">
              <Label class="text-xs uppercase font-bold text-muted-foreground tracking-wider mb-2 block">执行历史记录</Label>
              <div class="flex items-center justify-between pb-2">
                <span class="text-[10px] text-muted-foreground">点击历史回放轨迹</span>
                <Button variant="ghost" size="icon" class="h-6 w-6" @click="loadRuns" title="刷新历史">
                  <History class="h-3 w-3" />
                </Button>
              </div>
              
              <div v-if="runHistory.length === 0" class="text-xs text-muted-foreground text-center py-4">暂无执行记录</div>
              
              <div class="space-y-2 overflow-y-auto max-h-[calc(100vh-450px)] pr-1">
                <div v-for="run in runHistory" :key="run.runId" 
                  @click="selectRun(run.runId === selectedRunId ? null : run.runId)"
                  :class="['p-3 rounded-md border text-sm cursor-pointer transition-colors hover:border-primary/50', run.runId === selectedRunId ? 'border-primary bg-primary/5' : 'bg-card']">
                  <div class="flex items-center justify-between mb-1">
                    <span class="font-bold truncate mr-2 font-mono text-[11px]" :title="run.runId">{{ run.runId.substring(7) }}</span>
                    <component :is="getRunStatus(run) === 'success' ? CheckCircle2 : (getRunStatus(run) === 'failed' ? XCircle : Loader2)" 
                      :class="['h-3.5 w-3.5 shrink-0', getRunStatus(run) === 'success' ? 'text-emerald-500' : (getRunStatus(run) === 'failed' ? 'text-rose-500' : 'text-blue-500 animate-spin')]" />
                  </div>
                  <div class="text-[10px] text-muted-foreground">{{ run.startTime ? run.startTime.substring(11, 19) : '--:--' }} · {{ run.logs?.length || 0 }} 节点</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style>
@reference "@/assets/index.css";

.vue-flow__node-default {
  @apply bg-background border border-primary/20 text-foreground font-medium rounded-md shadow-sm text-sm px-4 py-2 hover:border-primary w-[160px] text-center truncate;
}
.vue-flow__edge-path {
  stroke: #94a3b8 !important;
  stroke-width: 2px !important;
  transition: all 0.2s;
}
.vue-flow__edge:hover .vue-flow__edge-path {
  stroke: #64748b !important;
  stroke-width: 3px !important;
}
/* If an edge has explicit inline stroke style (e.g. green or red), override the global edge path */
.vue-flow__edge[style*="stroke: #10b981"] .vue-flow__edge-path,
.vue-flow__edge[style*="stroke: #ef4444"] .vue-flow__edge-path {
  stroke: inherit !important;
}
/* 强化连接点 (Handles) 的视觉大小和颜色，方便用户拖拽 */
.vue-flow__handle {
  @apply w-3 h-3 bg-primary border-2 border-background shadow-sm transition-transform;
}
.vue-flow__handle:hover {
  @apply scale-150 bg-primary/80;
}
</style>
