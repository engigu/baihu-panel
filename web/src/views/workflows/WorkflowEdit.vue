<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, WORKFLOW, LOG_STATUS, type Workflow, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { ArrowLeft, Save, GripVertical, GripHorizontal, Settings2, History, CheckCircle2, XCircle, Loader2, Map as MapIcon, Play, Clock, Info, Search, Network } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { PanelLeft, PanelRight, X } from 'lucide-vue-next'

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

const { onConnect, addEdges, addNodes, onNodeClick, onEdgeClick, onPaneClick, removeNodes, removeEdges, findNode } = useVueFlow()

const elements = ref<any[]>([])
const selectedElement = ref<any>(null)
const selectedElementType = ref<'node' | 'edge' | null>(null)

// Sidebar Visibility - Responsive by default
const isLeftVisible = ref(window.innerWidth >= 1024)
const isRightVisible = ref(window.innerWidth >= 1280)

// Sidebar Pagination & Search
const scriptSearch = ref('')
const tasksPage = ref(1)
const hasMoreTasks = ref(true)
const isLoadingTasks = ref(false)

// Setup vue flow connection behavior
onNodeClick((e: any) => {
  selectedElement.value = e.node
  selectedElementType.value = 'node'
})

onEdgeClick((e: any) => {
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
    if (val === WORKFLOW.CONDITION.SUCCESS) {
      selectedElement.value.label = '成功 (Exit: 0)'
      selectedElement.value.style = { stroke: '#10b981', strokeWidth: 2 } // green
    } else if (val === WORKFLOW.CONDITION.FAILED) {
      selectedElement.value.label = '失败 (Exit: !=0)'
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

onConnect((params: Connection) => {
  addEdges([{
    ...params,
    id: `edge-${params.source}-${params.target}-${Date.now()}`,
    label: '总是 (Always)',
    data: { 
      condition: WORKFLOW.CONDITION.ALWAYS,
      nodeType: (findNode(params.target)?.data as any)?.nodeType || WORKFLOW.NODE_TYPE.TASK
    },
    type: 'smoothstep',
    markerEnd: MarkerType.ArrowClosed,
  }])
})

async function loadWorkflow() {
  const wid = route.params.id as string
  try {
    const res = await api.workflows.get(wid)
    console.log('Workflow loaded:', res)
    workflow.value = res
    
    if (workflow.value.nodes && workflow.value.nodes.length > 0) {
      // 转换后端扁平的 posX/posY 到 VueFlow 的 position 结构
      elements.value = [
        ...(workflow.value.nodes.map((n: any) => ({
          ...n,
          position: { x: n.posX || 0, y: n.posY || 0 }
        }))), 
        ...(workflow.value.edges || [])
      ]
    } else if ((workflow.value as any).flow_data) {
      try {
        const parsed = JSON.parse((workflow.value as any).flow_data)
        elements.value = [...(parsed.nodes || []), ...(parsed.edges || [])]
      } catch (e) {
        toast.error('解析流程数据失败')
      }
    }
  } catch (err: any) {
    console.error('Failed to load workflow:', err)
    toast.error('系统出错加载工作流: ' + (err.message || ''))
    router.push('/workflows')
  }
}

async function loadTasks(reset = false) {
  if (reset) {
    tasksPage.value = 1
    hasMoreTasks.value = true
    tasks.value = []
  }

  if (!hasMoreTasks.value || isLoadingTasks.value) return

  isLoadingTasks.value = true
  try {
    const res = await api.tasks.list({ 
      page: tasksPage.value, 
      page_size: 50,
      name: scriptSearch.value || undefined
    })
    
    if (res.data.length < 50) {
      hasMoreTasks.value = false
    }
    
    tasks.value = [...tasks.value, ...(res.data || [])]
    tasksPage.value++
  } catch {
    toast.error('加载可用脚本失败')
  } finally {
    isLoadingTasks.value = false
  }
}

// Search debounce
let searchTimeout: any = null
watch(scriptSearch, () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    loadTasks(true)
  }, 300)
})

function handleTaskScroll(event: any) {
  const { scrollTop, clientHeight, scrollHeight } = event.target
  if (scrollTop + clientHeight >= scrollHeight - 30) {
    if (!isLoadingTasks.value && hasMoreTasks.value) {
      loadTasks()
    }
  }
}

// Handling Drag & Drop from Sidebar
function onDragStart(event: DragEvent, task: any) {
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
  const data = event.dataTransfer?.getData('application/vueflow')
  if (!data) return

  const item = JSON.parse(data)
  const isControl = item.type === WORKFLOW.NODE_TYPE.CONTROL
  
  const reactFlowBounds = (event.target as Element).getBoundingClientRect()
  const position = {
    x: event.clientX - reactFlowBounds.left,
    y: event.clientY - reactFlowBounds.top,
  }

  // Customize style for control nodes
  let nodeStyle = {}
  if (isControl) {
    if (item.controlType === WORKFLOW.CONTROL_TYPE.DELAY) {
      nodeStyle = { background: '#fef3c7', border: '1px solid #f59e0b', color: '#92400e' }
    } else if (item.controlType === WORKFLOW.CONTROL_TYPE.CONDITION) {
      nodeStyle = { background: '#eff6ff', border: '1px solid #3b82f6', color: '#1e40af' }
    } else if (item.controlType === WORKFLOW.CONTROL_TYPE.LOOP) {
      nodeStyle = { background: '#faf5ff', border: '1px solid #a855f7', color: '#6b21a8' }
    }
  }

  const newNode = {
    id: getId(),
    type: isControl ? 'output' : 'default',
    position,
    label: item.name,
    data: { 
      taskId: item.id,
      nodeType: item.type || WORKFLOW.NODE_TYPE.TASK,
      controlType: item.controlType || ''
    },
    style: nodeStyle
  }

  addNodes([newNode])
}

async function saveFlow() {
  const wid = route.params.id as string
  const flowData = JSON.stringify({
    nodes: elements.value.filter(el => !el.source),
    edges: elements.value.filter(el => el.source),
  })

  try {
    const current = await api.workflows.get(wid)
    await api.workflows.update(wid, { ...current, flow_data: flowData })
    toast.success('工作流配置已保存')
  } catch {
    toast.error('保存失败')
  }
}

const isRunning = ref(false)
async function triggerRun() {
  const wid = route.params.id as string
  isRunning.value = true
  try {
    await api.workflows.run(wid, globalEnvs.value)
    toast.success('工作流已触发，请观察轨迹')
    loadRuns()
  } catch (err: any) {
    toast.error('运行失败: ' + err.message)
  } finally {
    isRunning.value = false
  }
}

// Global Envs
const globalEnvs = ref<string[]>([])
const newEnv = ref('')
function addEnv() {
  if (newEnv.value.includes('=')) {
    globalEnvs.value.push(newEnv.value)
    newEnv.value = ''
  } else {
    toast.error('格式必须为 KEY=VALUE')
  }
}
function removeEnv(idx: number) {
  globalEnvs.value.splice(idx, 1)
}

// History Tracking
const runHistory = ref<any[]>([])
const selectedRunId = ref<string | null>(null)

async function loadRuns() {
  const wid = route.params.id as string
  try {
    const res = await api.logs.list({ workflow_id: wid, page: 1, page_size: 50 })
    // Group by run_id
    const groups: Record<string, any> = {}
    res.data.forEach((log: any) => {
      if (!log.workflow_run_id) return
      if (!groups[log.workflow_run_id]) {
        groups[log.workflow_run_id] = {
          runId: log.workflow_run_id,
          startTime: log.start_time,
          status: 'running',
          logs: []
        }
      }
      groups[log.workflow_run_id].logs.push(log)
      if (log.status === WORKFLOW.RUN_STATUS.FAILED) groups[log.workflow_run_id].status = WORKFLOW.RUN_STATUS.FAILED
    })
    
    runHistory.value = Object.values(groups).sort((a, b) => b.runId.localeCompare(a.runId))
  } catch(e) { console.error(e) }
}

function getRunStatus(run: any) {
  const hasFailed = run.logs.some((l: any) => l.status === LOG_STATUS.FAILED)
  if (hasFailed) return WORKFLOW.RUN_STATUS.FAILED
  const allSuccess = run.logs.length > 0 && run.logs.every((l: any) => l.status === LOG_STATUS.SUCCESS)
  if (allSuccess) return WORKFLOW.RUN_STATUS.SUCCESS
  return WORKFLOW.RUN_STATUS.RUNNING
}

async function selectRun(runId: string | null) {
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
    const log = run.logs.find((l: any) => String(l.task_id) === String(el.data?.taskId))
    if (!log) {
      el.class = 'opacity-40 grayscale transition-all duration-500'
    } else if (log.status === LOG_STATUS.SUCCESS) {
      el.class = 'border-2 border-[#10b981] bg-[#10b981]/10 shadow-[0_0_15px_rgba(16,185,129,0.3)] transition-all duration-500'
    } else if (log.status === LOG_STATUS.FAILED) {
      el.class = 'border-2 border-[#ef4444] bg-[#ef4444]/10 shadow-[0_0_15px_rgba(239,68,68,0.3)] transition-all duration-500'
    } else {
      el.class = 'border-2 border-[#3b82f6] bg-[#3b82f6]/10 animate-pulse transition-all duration-500'
    }
  })
}

const showMiniMap = ref(false)

onMounted(() => {
  loadTasks()
  loadWorkflow()
  loadRuns()
})
</script>

<template>
  <div class="fixed inset-0 flex flex-col bg-background select-none overflow-hidden z-[50]">
    <!-- Top Header Navigation -->
    <header class="h-12 border-b flex items-center justify-between px-3 bg-background/80 backdrop-blur shrink-0 z-20">
      <div class="flex items-center gap-1.5 md:gap-3">
        <Button variant="ghost" size="icon" class="h-8 w-8" @click="router.push('/workflows')">
          <ArrowLeft class="h-4 w-4" />
        </Button>
        <div class="hidden sm:block">
          <h1 class="font-semibold text-xs md:text-sm truncate max-w-[120px] md:max-w-none">{{ workflow?.name || '加载中...' }}</h1>
          <p class="text-[9px] md:text-[10px] text-muted-foreground truncate">{{ workflow?.description || '构建流水线' }}</p>
        </div>
        
        <div class="flex items-center gap-1 ml-1 md:ml-2">
          <Button variant="ghost" size="icon" class="h-8 w-8" @click="isLeftVisible = !isLeftVisible" :class="isLeftVisible ? 'text-primary bg-primary/5' : ''">
            <PanelLeft class="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" class="h-8 w-8" @click="isRightVisible = !isRightVisible" :class="isRightVisible ? 'text-primary bg-primary/5' : ''">
            <PanelRight class="h-4 w-4" />
          </Button>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="ghost" size="sm" @click="triggerRun" :disabled="isRunning" class="gap-1 md:gap-1.5 h-8 text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 text-[10px] md:text-xs px-2 md:px-3">
          <Loader2 v-if="isRunning" class="h-3.5 w-3.5 animate-spin" />
          <Play v-else class="h-3.5 w-3.5 fill-current" />
          <span class="hidden xs:inline">{{ isRunning ? '执行中' : '立即触发' }}</span>
        </Button>
        <Button variant="outline" size="sm" @click="saveFlow" class="gap-1 md:gap-1.5 h-8 text-[10px] md:text-xs px-2 md:px-3">
          <Save class="h-3.5 w-3.5" />
          <span class="hidden xs:inline">保存</span>
        </Button>
      </div>
    </header>

    <!-- Main Workspace -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Left Sidebar: Task List -->
      <aside 
        v-show="isLeftVisible"
        class="w-56 md:w-64 border-r bg-background flex flex-col pt-4 overflow-hidden z-20 shrink-0 absolute inset-y-0 left-0 md:relative shadow-xl md:shadow-none"
      >
        <div class="px-4 pb-2 mb-2 border-b space-y-3 relative">
          <button @click="isLeftVisible = false" class="md:hidden absolute right-2 top-0 p-1 rounded-full hover:bg-muted">
            <X class="h-4 w-4" />
          </button>
          <h2 class="text-sm font-semibold flex items-center gap-2">
            <GripVertical class="h-4 w-4 text-muted-foreground" />
            系统可用脚本
          </h2>
          <div class="relative">
            <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input 
              v-model="scriptSearch" 
              placeholder="搜索脚本..." 
              class="h-8 pl-8 text-xs bg-muted/5 focus-visible:bg-background transition-colors" 
            />
          </div>
        </div>
        <!-- Task list that is scrollable -->
        <div 
          class="flex-1 overflow-y-auto px-2 space-y-1 pb-4" 
          @scroll="handleTaskScroll"
          ref="taskListContainer"
        >
          <div 
            v-for="task in tasks" 
            :key="task.id"
            :draggable="true"
            @dragstart="onDragStart($event, task)"
            class="px-2 py-1.5 text-[13px] border bg-card rounded cursor-grab active:cursor-grabbing hover:border-primary transition-all flex items-center justify-between group"
          >
            <span class="truncate pr-2 font-medium">{{ task.name }}</span>
            <GripHorizontal class="h-3.5 w-3.5 text-muted-foreground opacity-30 group-hover:opacity-100" />
          </div>
          <div v-if="tasks.length === 0 && !isLoadingTasks" class="py-10 text-center">
            <p class="text-xs text-muted-foreground">未找到可用脚本</p>
          </div>
          <div v-if="isLoadingTasks" class="py-4 text-center">
              <Loader2 class="h-4 w-4 animate-spin mx-auto text-muted-foreground" />
          </div>
        </div>

        <!-- Flow control components pinned to bottom -->
        <div class="shrink-0 flex flex-col">
          <div class="px-4 py-2 bg-muted/20 border-y">
            <h2 class="text-[10px] uppercase font-bold text-muted-foreground tracking-widest flex items-center gap-2">
              <Settings2 class="h-3 w-3" />
              流程控制组件
            </h2>
          </div>
          <div class="px-2 py-2 space-y-1.5 bg-background">
            <div 
              :draggable="true"
              @dragstart="onDragStart($event, { id: WORKFLOW.VIRTUAL_TASK_ID, name: '延时等待', type: WORKFLOW.NODE_TYPE.CONTROL, controlType: WORKFLOW.CONTROL_TYPE.DELAY } as any)"
              class="px-3 py-2 text-[13px] border bg-amber-50/50 border-amber-200 rounded cursor-grab active:cursor-grabbing hover:border-amber-400 transition-colors flex items-center justify-between group"
            >
              <div class="flex items-center gap-2 text-amber-900 font-medium">
                <Clock class="h-3.5 w-3.5 text-amber-600" />
                <span>延时等待</span>
              </div>
              <GripHorizontal class="h-3 w-3 text-amber-400 opacity-50 group-hover:opacity-100" />
            </div>

            <div 
              :draggable="true"
              @dragstart="onDragStart($event, { id: WORKFLOW.VIRTUAL_TASK_ID, name: '条件分支', type: WORKFLOW.NODE_TYPE.CONTROL, controlType: WORKFLOW.CONTROL_TYPE.CONDITION } as any)"
              class="px-3 py-2 text-[13px] border bg-blue-50/50 border-blue-200 rounded cursor-grab active:cursor-grabbing hover:border-blue-400 transition-colors flex items-center justify-between group"
            >
              <div class="flex items-center gap-2 text-blue-900 font-medium">
                <Network class="h-3.5 w-3.5 text-blue-600" />
                <span>条件分支</span>
              </div>
              <GripHorizontal class="h-3 w-3 text-blue-400 opacity-50 group-hover:opacity-100" />
            </div>

            <div 
              :draggable="true"
              @dragstart="onDragStart($event, { id: WORKFLOW.VIRTUAL_TASK_ID, name: '次数循环', type: WORKFLOW.NODE_TYPE.CONTROL, controlType: WORKFLOW.CONTROL_TYPE.LOOP } as any)"
              class="px-3 py-2 text-[13px] border bg-purple-50/50 border-purple-200 rounded cursor-grab active:cursor-grabbing hover:border-purple-400 transition-colors flex items-center justify-between group"
            >
              <div class="flex items-center gap-2 text-purple-900 font-medium">
                <History class="h-3.5 w-3.5 text-purple-600" />
                <span>次数循环</span>
              </div>
              <GripHorizontal class="h-3 w-3 text-purple-400 opacity-50 group-hover:opacity-100" />
            </div>
          </div>
        </div>
      </aside>

      <!-- Vue Flow Canvas -->
      <div class="flex-1 relative" @drop="onDrop" @dragover="onDragOver">
        <VueFlow v-model="elements" :default-zoom="1.2" :min-zoom="0.2" :max-zoom="4" fit-view-on-init>
          <Background pattern-color="#aaa" :gap="20" />
          <MiniMap v-if="showMiniMap" pannable zoomable />
          <Controls />
          <div class="absolute bottom-4 right-4 z-40 flex flex-col gap-2">
            <Button variant="outline" size="icon" class="h-8 w-8 bg-background" @click="showMiniMap = !showMiniMap">
              <MapIcon class="h-4 w-4" :class="showMiniMap ? 'text-primary' : 'text-muted-foreground'" />
            </Button>
          </div>
        </VueFlow>
      </div>

      <!-- Right Sidebar: Properties -->
      <aside 
        v-show="isRightVisible"
        class="w-64 border-l bg-background flex flex-col pt-4 z-20 shrink-0 absolute inset-y-0 right-0 md:relative shadow-xl md:shadow-none"
      >
        <div class="px-4 pb-2 mb-4 border-b relative">
          <button @click="isRightVisible = false" class="md:hidden absolute right-2 top-0 p-1 rounded-full hover:bg-muted">
            <X class="h-4 w-4" />
          </button>
          <h2 class="text-xs font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
            <template v-if="selectedElement">
              <Settings2 class="h-4 w-4" />
              {{ selectedElementType === 'node' ? '节点设置' : '分支属性' }}
            </template>
            <template v-else>
              <History class="h-4 w-4" />
              运行概览
            </template>
          </h2>
        </div>
        
        <div class="flex-1 overflow-y-auto px-4 pb-10">
          <div v-if="selectedElementType === 'node'" class="space-y-4">
            <div class="space-y-2">
              <Label class="text-xs">展示名称</Label>
              <Input v-model="selectedElement.label" class="h-8 text-xs" />
            </div>

            <!-- Configuration fields for control nodes -->
            <div v-if="selectedElement.data?.controlType === WORKFLOW.CONTROL_TYPE.DELAY" class="space-y-2 pt-4 border-t border-dashed">
                <Label class="text-xs font-bold flex items-center gap-1 text-amber-700">
                  <Clock class="h-3 w-3" /> 延时秒数
                </Label>
                <Input type="number" :model-value="selectedElement.data?.config || '5'" @input="(e: any) => selectedElement.data.config = e.target.value" class="h-8 text-xs" />
            </div>

            <div v-if="selectedElement.data?.controlType === WORKFLOW.CONTROL_TYPE.CONDITION" class="space-y-2 pt-4 border-t border-dashed">
                <Label class="text-xs font-bold flex items-center gap-1 text-blue-700">
                  <Network class="h-3 w-3" /> 判断表达式
                </Label>
                <Input :model-value="selectedElement.data?.config || ''" @input="(e: any) => selectedElement.data.config = e.target.value" placeholder="{{status}} == 200" class="h-8 text-xs font-mono" />
            </div>

            <div v-if="selectedElement.data?.controlType === WORKFLOW.CONTROL_TYPE.LOOP" class="space-y-2 pt-4 border-t border-dashed">
                <Label class="text-xs font-bold flex items-center gap-1 text-purple-700">
                  <History class="h-3 w-3" /> 循环总次数
                </Label>
                <Input type="number" :model-value="selectedElement.data?.config || '3'" @input="(e: any) => selectedElement.data.config = e.target.value" class="h-8 text-xs" />
            </div>

            <div class="pt-4">
              <Button variant="destructive" size="sm" class="w-full h-8 text-xs" @click="deleteSelected">移除节点</Button>
            </div>
          </div>
          
          <div v-if="selectedElementType === 'edge'" class="space-y-4">
            <div class="space-y-2">
              <Label class="text-xs">触发连线条件</Label>
              <Select :model-value="selectedElement.data?.condition || WORKFLOW.CONDITION.ALWAYS" @update:model-value="updateEdgeCondition">
                <SelectTrigger class="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem :value="WORKFLOW.CONDITION.ALWAYS">无条件联动 (Always)</SelectItem>
                  <SelectItem :value="WORKFLOW.CONDITION.SUCCESS">成功时触发 (Exit 0)</SelectItem>
                  <SelectItem :value="WORKFLOW.CONDITION.FAILED">失败时触发 (Exit !0)</SelectItem>
                </SelectContent>
              </Select>
              <div class="pt-4">
                <Button variant="destructive" size="sm" class="w-full h-8 text-xs" @click="deleteSelected">移除连线</Button>
              </div>
            </div>
          </div>
          
          <div v-if="!selectedElement" class="space-y-6">
            <!-- Workflow Variable Rules Help (Condensed) -->
            <div class="p-2.5 bg-amber-950/20 border border-amber-500/30 rounded-lg space-y-2 shadow-sm">
              <h3 class="text-[11px] uppercase font-extrabold text-amber-500 flex items-center gap-1.5 tracking-wider">
                <Info class="h-3 w-3" /> 数据传递规范
              </h3>
              <ul class="text-[10px] text-amber-200/90 space-y-1.5 leading-tight pl-0.5">
                <li class="flex gap-1.5 align-top">
                  <span class="text-amber-500 font-bold shrink-0">•</span>
                  <span>识别日志 <code class="bg-amber-500/20 text-amber-400 px-1 py-0 rounded border border-amber-500/20 font-mono text-[9px]">output.KEY=VAL</code></span>
                </li>
                <li class="flex gap-1.5 align-top">
                  <span class="text-amber-500 font-bold shrink-0">•</span>
                  <span>下游通过环境变量 <code class="bg-emerald-500/20 text-emerald-400 px-1 py-0 rounded border border-emerald-500/20 font-mono text-[9px]">input.KEY</code> 接收</span>
                </li>
                <li class="flex gap-1.5 align-top">
                  <span class="text-amber-500 font-bold shrink-0">•</span>
                  <span>支持宽容模式，如 <code class="text-amber-300/80 font-mono">output.name = val</code></span>
                </li>
                <li class="flex gap-1.5 align-top">
                  <span class="text-amber-500 font-bold shrink-0">•</span>
                  <span>变量冲突时采用“最后一次覆盖”策略</span>
                </li>
              </ul>
            </div>

            <div class="space-y-3">
              <Label class="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">初始环境变量</Label>
              <div class="space-y-2">
                <div v-for="(env, idx) in globalEnvs" :key="idx" class="flex items-center gap-2 bg-muted/40 p-1.5 rounded text-[10px] font-mono group">
                  <span class="flex-1 truncate">{{ env }}</span>
                  <button @click="removeEnv(idx)" class="opacity-0 group-hover:opacity-100 hover:text-destructive">
                    <XCircle class="h-3 w-3" />
                  </button>
                </div>
                <div class="flex gap-1">
                  <Input v-model="newEnv" placeholder="KEY=VALUE" class="h-7 text-[10px] font-mono" />
                  <Button variant="outline" size="sm" class="h-7 px-2 text-[10px]" @click="addEnv">添加</Button>
                </div>
              </div>
            </div>

            <div class="border-t pt-4">
              <div class="flex items-center justify-between mb-2">
                <Label class="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">运行回放</Label>
                <Button variant="ghost" size="icon" class="h-5 w-5" @click="loadRuns">
                  <History class="h-3 w-3" />
                </Button>
              </div>
              <div class="space-y-1.5 pr-1 pb-10">
                <div v-for="run in runHistory" :key="run.runId" 
                  @click="selectRun(run.runId === selectedRunId ? null : run.runId)"
                  :class="['p-2 rounded-sm border text-[11px] cursor-pointer transition-all hover:border-primary/40', run.runId === selectedRunId ? 'border-primary bg-primary/5 ring-1 ring-primary/20' : 'bg-card']">
                  <div class="flex items-center justify-between">
                    <span class="font-mono text-muted-foreground">#{{ run.runId.substring(run.runId.length - 6) }}</span>
                    <component :is="getRunStatus(run) === WORKFLOW.RUN_STATUS.SUCCESS ? CheckCircle2 : (getRunStatus(run) === WORKFLOW.RUN_STATUS.FAILED ? XCircle : Loader2)" 
                      :class="['h-3 w-3', getRunStatus(run) === WORKFLOW.RUN_STATUS.SUCCESS ? 'text-emerald-500' : (getRunStatus(run) === WORKFLOW.RUN_STATUS.FAILED ? 'text-rose-500' : 'text-blue-500 animate-spin')]" />
                  </div>
                  <div class="text-[10px] mt-1 text-muted-foreground/60">{{ run.startTime ? run.startTime.substring(11, 16) : '--:--' }} · {{ run.logs.length }} 节点</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.vue-flow__node-default, .vue-flow__node-output {
  background: var(--background);
  border: 1px solid rgba(var(--primary), 0.1);
  color: var(--foreground);
  font-weight: 500;
  border-radius: 6px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  font-size: 12px;
  padding: 8px 12px;
  width: 150px;
  text-align: center;
  transition: all 0.2s;
}

.vue-flow__node-default:hover, .vue-flow__node-output:hover {
  border-color: hsl(var(--primary));
  box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
}

.vue-flow__edge-path {
  stroke: #94a3b8;
  stroke-width: 2px;
}

.vue-flow__handle {
  width: 8px;
  height: 8px;
  background: hsl(var(--primary));
  border: 2px solid var(--background);
}

.vue-flow__minimap {
  background-color: var(--background);
}
</style>
