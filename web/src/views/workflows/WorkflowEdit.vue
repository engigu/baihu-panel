<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type Workflow, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { ArrowLeft, Save, GripVertical, GripHorizontal, Settings2 } from 'lucide-vue-next'
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
      selectedElement.value.style = { stroke: 'hsl(var(--primary))', strokeWidth: 2 }
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
    type: 'smoothstep',
    markerEnd: MarkerType.ArrowClosed,
    label: '总是 (Always)',
    data: { condition: 'always' },
    style: { stroke: 'hsl(var(--primary))', strokeWidth: 2 }
  }])
})

async function loadWorkflow() {
  const wid = parseInt(route.params.id as string)
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

  const task: Task = JSON.parse(data)
  
  // Calculate relative position based on mouse drop
  const reactFlowBounds = (event.target as Element).getBoundingClientRect()
  const position = {
    x: event.clientX - reactFlowBounds.left - 50,
    y: event.clientY - reactFlowBounds.top - 20,
  }

  const newNode = {
    id: getId(),
    type: 'default',
    position,
    label: task.name,
    data: { taskId: task.id },
  }

  addNodes([newNode])
}

async function saveFlow() {
  if (!workflow.value) return
  
  // Get all current nodes and edges from VueFlow instance
  // Note: Here we are storing elements.value directly,
  const payload = {
    flow_data: JSON.stringify({
      nodes: elements.value.filter(el => !el.source),
      edges: elements.value.filter(el => el.source)
    })
  }
  
  try {
    await api.workflows.update(workflow.value.id, payload)
    toast.success('工作流编排保存成功')
  } catch {
    toast.error('保存失败')
  }
}

onMounted(() => {
  loadTasks()
  loadWorkflow()
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
      </aside>

      <!-- Vue Flow Canvas -->
      <div class="flex-1 relative" @drop="onDrop" @dragover="onDragOver">
        <VueFlow v-model="elements" :default-zoom="1.5" :min-zoom="0.2" :max-zoom="4" fit-view-on-init>
          <Background pattern-color="#aaa" :gap="20" />
          
          <MiniMap />
          <Controls />
        </VueFlow>
      </div>

      <!-- Right Sidebar: Properties -->
      <aside v-if="selectedElement" class="w-64 border-l bg-background flex flex-col pt-4 z-10">
        <div class="px-4 pb-2 mb-4 border-b">
          <h2 class="text-sm font-semibold flex items-center gap-2">
            <Settings2 class="h-4 w-4 text-muted-foreground" />
            {{ selectedElementType === 'node' ? '节点设置' : '连线分支设置' }}
          </h2>
        </div>
        
        <div class="px-4 flex-1">
          <!-- Node Properties -->
          <div v-if="selectedElementType === 'node'" class="space-y-4">
            <div class="space-y-2">
              <Label>展示名称</Label>
              <Input v-model="selectedElement.label" />
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
            </div>
            <div class="pt-4">
              <Button variant="destructive" class="w-full" @click="deleteSelected">删除关联连线</Button>
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
  @apply stroke-primary/50 stroke-2 transition-all;
}
.vue-flow__edge:hover .vue-flow__edge-path {
  @apply stroke-primary stroke-[3px];
}
/* 强化连接点 (Handles) 的视觉大小和颜色，方便用户拖拽 */
.vue-flow__handle {
  @apply w-3 h-3 bg-primary border-2 border-background shadow-sm transition-transform;
}
.vue-flow__handle:hover {
  @apply scale-150 bg-primary/80;
}
</style>
