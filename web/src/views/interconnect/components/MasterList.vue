<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import StatusDot from '@/components/StatusDot.vue'
import { Plus, Edit2, Trash2, RefreshCw, ExternalLink, Eye, Copy } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { toast } from 'vue-sonner'
import * as interconnectApi from '@/api/interconnect'
import { copyToClipboard } from '@/utils/clipboard'
import { setActiveInterconnectNodeId } from '@/api'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

const props = defineProps<{
  nodes: interconnectApi.InterconnectNode[]
  loading: boolean
  searchQuery: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const copied = ref(false)
const detailOpen = ref(false)
const detailLoading = ref(false)
const selectedNodeDetail = ref<any>(null)
const selectedNodeName = ref('')
const nodeStatuses = ref<Record<string, any>>({})

const filteredNodes = computed(() => {
  if (!props.searchQuery) return props.nodes
  const lowerKeyword = props.searchQuery.toLowerCase()
  return props.nodes.filter(node => 
    node.name.toLowerCase().includes(lowerKeyword) || 
    node.url.toLowerCase().includes(lowerKeyword) || 
    (node.remark && node.remark.toLowerCase().includes(lowerKeyword))
  )
})

const dialogOpen = ref(false)
const isEditing = ref(false)
const currentForm = ref<Partial<interconnectApi.InterconnectNode>>({
  name: '',
  url: '',
  token: '',
  remark: ''
})

const showDeleteConfirm = ref(false)
const deleteId = ref('')

function formatUptime(seconds: number | undefined): string {
  if (!seconds) return '-'
  const days = Math.floor(seconds / (24 * 3600))
  const hours = Math.floor((seconds % (24 * 3600)) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  const parts = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}分钟`)
  return parts.join('')
}

function formatBytes(bytes: number | undefined): string {
  if (bytes === undefined) return '-'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(2)} GB`
  const mb = bytes / (1024 * 1024)
  return `${mb.toFixed(2)} MB`
}

function getLoadColor(percent: number | undefined): string {
  if (percent === undefined) return 'text-muted-foreground'
  if (percent < 50) return 'text-green-500'
  if (percent < 80) return 'text-yellow-500'
  return 'text-destructive'
}

function generateRandomToken(length = 32) {
  const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
  let res = ""
  for (let i = 0; i < length; ++i) {
    res += charset[Math.floor(Math.random() * charset.length)]
  }
  return res
}

async function handleCopy(text: string) {
  const success = await copyToClipboard(text)
  if (success) {
    copied.value = true
    toast.success('已复制到剪贴板')
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } else {
    toast.error('复制失败，请手动复制')
  }
}

async function fetchNodeStatus(id: string) {
  try {
    nodeStatuses.value[id] = { status: 'loading' }
    const res = await interconnectApi.getNodeStatus(id)
    nodeStatuses.value[id] = Object.assign({ status: 'online' }, res)
  } catch (error: any) {
    nodeStatuses.value[id] = { status: 'offline', error: error.message || '离线' }
  }
}

watch(() => props.nodes, (newNodes) => {
  newNodes.forEach(node => {
    if (!nodeStatuses.value[node.id] || nodeStatuses.value[node.id].status === 'offline') {
      fetchNodeStatus(node.id)
    }
  })
}, { immediate: true })

async function showDetail(node: interconnectApi.InterconnectNode) {
  selectedNodeName.value = node.name
  selectedNodeDetail.value = null
  detailLoading.value = true
  detailOpen.value = true
  try {
    const res = await interconnectApi.getNodeStatus(node.id)
    selectedNodeDetail.value = res
  } catch (error: any) {
    toast.error('获取子节点详细信息失败')
  } finally {
    detailLoading.value = false
  }
}

function handleTravel(nodeId: string) {
  setActiveInterconnectNodeId(nodeId)
  window.location.href = '/'
}

function openAddDialog() {
  isEditing.value = false
  currentForm.value = { name: '', url: '', token: generateRandomToken(), remark: '' }
  dialogOpen.value = true
}

function openEditDialog(node: interconnectApi.InterconnectNode) {
  isEditing.value = true
  currentForm.value = { ...node }
  dialogOpen.value = true
}

async function handleSave() {
  if (!currentForm.value.name || (!isEditing.value && !currentForm.value.token)) {
    toast.error('请填写必要信息')
    return
  }

  try {
    if (isEditing.value && currentForm.value.id) {
      await interconnectApi.updateNode(currentForm.value.id, currentForm.value)
      toast.success('更新成功')
    } else {
      await interconnectApi.createNode(currentForm.value)
      toast.success('添加成功')
    }
    dialogOpen.value = false
    emit('refresh')
  } catch (error: any) {
    toast.error(isEditing.value ? '更新失败' : '添加失败')
  }
}

function confirmDelete(id: string) {
  deleteId.value = id
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteId.value) return
  showDeleteConfirm.value = false
  try {
    await interconnectApi.deleteNode(deleteId.value)
    toast.success('删除成功')
    emit('refresh')
  } catch (error: any) {
    toast.error('删除失败')
  }
}

defineExpose({
  openAddDialog
})
</script>

<template>
  <div class="space-y-4">
    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- ========== 1. 大屏布局 (Large >= 1280px) ========== -->
      <div class="hidden xl:block">
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-56 shrink-0">节点名称</span>
          <span class="flex-1 min-w-0">隧道地址 </span>
          <span class="w-48 shrink-0">运行状态 & 负载</span>
          <span class="w-48 shrink-0">备注</span>
          <span class="w-40 shrink-0 text-center">操作</span>
        </div>
        <div class="divide-y text-sm">
          <div v-if="loading" class="py-8 text-center text-muted-foreground">加载中...</div>
          <div v-else-if="filteredNodes.length === 0" class="py-8 text-center text-muted-foreground">暂无连接的子节点</div>
          <div v-for="(node, index) in filteredNodes" :key="`large-${node.id}`" class="flex items-center gap-2 px-4 py-2 hover:bg-muted/30 transition-colors">
            <StatusDot 
              :state="nodeStatuses[node.id]?.status === 'loading' ? 'pending' : (nodeStatuses[node.id]?.status === 'online' || node.status === 'online' ? 'online' : 'failed')"
              :title="nodeStatuses[node.id]?.status === 'loading' ? '检测中' : (nodeStatuses[node.id]?.status === 'online' || node.status === 'online' ? '在线' : '离线')" 
            />
            <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-[11px]">#{{ index + 1 }}</div>
            <div class="w-56 shrink-0 flex items-center">
              <span class="font-medium truncate" :title="node.name">{{ node.name }}</span>
            </div>
            <div class="flex-1 min-w-0 text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded" style="font-family: Inter, sans-serif;" :title="node.url">
              {{ node.url || '-' }}
            </div>
            <div class="w-48 shrink-0 flex flex-col justify-center cursor-pointer gap-1" @click="fetchNodeStatus(node.id)" title="点击刷新状态">
              <template v-if="nodeStatuses[node.id]?.status === 'loading'">
                <div class="flex items-center gap-1.5">
                  <RefreshCw class="h-3.5 w-3.5 animate-spin text-muted-foreground" />
                  <span class="text-xs text-muted-foreground">检测中...</span>
                </div>
              </template>
              <template v-else-if="nodeStatuses[node.id]?.status === 'online' || node.status === 'online'">
                <div class="flex items-center gap-1.5" v-if="nodeStatuses[node.id]?.version">
                  <span class="text-xs text-muted-foreground font-normal">v{{ nodeStatuses[node.id].version }}</span>
                </div>
                <div class="flex items-center gap-2 text-[10px]" v-if="node.metrics?.cpu_percent !== undefined && node.metrics?.mem_percent !== undefined && node.metrics?.cpu_percent !== 0">
                  <span :class="getLoadColor(node.metrics?.cpu_percent)">CPU: {{ node.metrics?.cpu_percent.toFixed(1) }}%</span>
                  <span :class="getLoadColor(node.metrics?.mem_percent)">Mem: {{ node.metrics?.mem_percent.toFixed(1) }}%</span>
                </div>
              </template>
              <template v-else>
                <div class="flex items-center gap-1.5">
                  <span class="text-xs text-destructive font-medium">离线</span>
                </div>
              </template>
            </div>
            <div class="w-48 shrink-0 truncate text-xs text-muted-foreground" :title="node.remark">
              {{ node.remark || '-' }}
            </div>
            <div class="w-40 shrink-0 flex justify-center gap-1">
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="showDetail(node)" title="查看详情">
                <Eye class="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="handleTravel(node.id)" title="穿越到此子节点">
                <ExternalLink class="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEditDialog(node)" title="编辑">
                <Edit2 class="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="confirmDelete(node.id)" title="删除">
                <Trash2 class="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 2. 中屏布局 (Medium 640px - 1280px) ========== -->
      <div class="hidden sm:block xl:hidden">
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-48 shrink-0">节点信息</span>
          <span class="flex-1 min-w-0">隧道地址 </span>
          <span class="w-32 shrink-0">状态与负载</span>
          <span class="w-36 shrink-0 text-center">操作</span>
        </div>
        <div class="divide-y text-sm">
          <div v-if="loading" class="py-8 text-center text-muted-foreground">加载中...</div>
          <div v-else-if="filteredNodes.length === 0" class="py-8 text-center text-muted-foreground">暂无连接的子节点</div>
          <div v-for="(node, index) in filteredNodes" :key="`medium-${node.id}`" class="flex items-center gap-2 px-4 py-2.5 hover:bg-muted/30 transition-colors">
            <StatusDot 
              :state="nodeStatuses[node.id]?.status === 'loading' ? 'pending' : (nodeStatuses[node.id]?.status === 'online' || node.status === 'online' ? 'online' : 'failed')"
            />
            <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-[10px]">#{{ index + 1 }}</div>
            <div class="w-48 shrink-0 flex items-center overflow-hidden">
              <div class="flex flex-col min-w-0">
                <span class="font-medium truncate text-sm">{{ node.name }}</span>
                <span v-if="node.remark" class="text-[10px] text-muted-foreground truncate">{{ node.remark }}</span>
              </div>
            </div>
            <div class="flex-1 min-w-0 text-[11px] text-muted-foreground bg-muted/20 px-2 py-1 rounded truncate" style="font-family: Inter, sans-serif;" :title="node.url">
              {{ node.url || '-' }}
            </div>
            <div class="w-32 shrink-0 flex flex-col justify-center cursor-pointer gap-0.5" @click="fetchNodeStatus(node.id)">
              <template v-if="nodeStatuses[node.id]?.status === 'loading'">
                <RefreshCw class="h-4 w-4 animate-spin text-muted-foreground" />
              </template>
              <template v-else-if="nodeStatuses[node.id]?.status === 'online' || node.status === 'online'">
                <div class="flex items-center gap-1.5" v-if="nodeStatuses[node.id]?.version">
                  <span class="text-[10px] text-muted-foreground font-normal">v{{ nodeStatuses[node.id].version }}</span>
                </div>
                <div class="flex items-center gap-1 text-[9px]" v-if="node.metrics?.cpu_percent !== undefined && node.metrics?.mem_percent !== undefined && node.metrics?.cpu_percent !== 0">
                  <span :class="getLoadColor(node.metrics?.cpu_percent)">C:{{ node.metrics?.cpu_percent.toFixed(0) }}%</span>
                  <span :class="getLoadColor(node.metrics?.mem_percent)">M:{{ node.metrics?.mem_percent.toFixed(0) }}%</span>
                </div>
              </template>
              <template v-else>
                <div class="flex items-center gap-1.5">
                  <span class="text-xs text-destructive font-medium">离线</span>
                </div>
              </template>
            </div>
            <div class="w-36 shrink-0 flex justify-center gap-0.5">
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="showDetail(node)" title="查看详情"><Eye class="h-3.5 w-3.5" /></Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="handleTravel(node.id)" title="穿越到此子节点"><ExternalLink class="h-3.5 w-3.5" /></Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEditDialog(node)" title="编辑"><Edit2 class="h-3.5 w-3.5" /></Button>
              <Button variant="ghost" size="icon" class="h-7 w-7" @click="confirmDelete(node.id)" title="删除"><Trash2 class="h-3.5 w-3.5" /></Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 3. 小屏布局 (Small < 640px) ========== -->
      <div class="divide-y sm:hidden">
        <div v-if="loading" class="text-sm text-muted-foreground text-center py-8">加载中...</div>
        <div v-else-if="filteredNodes.length === 0" class="text-sm text-muted-foreground text-center py-8">暂无连接的子节点</div>
        <div v-for="(node, index) in filteredNodes" :key="`small-${node.id}`" class="p-3 hover:bg-muted/50 transition-colors">
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex items-center gap-2 flex-1 min-w-0 pr-2">
              <StatusDot 
                :state="nodeStatuses[node.id]?.status === 'loading' ? 'pending' : (nodeStatuses[node.id]?.status === 'online' || node.status === 'online' ? 'online' : 'failed')"
                class="mt-0.5"
              />
              <span class="text-[10px] text-muted-foreground tabular-nums flex-shrink-0">#{{ index + 1 }}</span>
              <div class="flex items-center gap-1.5 min-w-0 flex-1">
                <span class="font-bold text-sm truncate">{{ node.name }}</span>
              </div>
            </div>
          </div>
          <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
            <div class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">隧道地址:</span>
              <div class="flex-1 min-w-0 overflow-hidden text-foreground">
                <div class="text-[11px] bg-muted/40 px-1 py-0.5 rounded break-all" style="font-family: Inter, sans-serif;">{{ node.url || '-' }}</div>
              </div>
            </div>
            <div v-if="node.remark" class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">备注:</span>
              <span class="flex-1 text-[11px] truncate">{{ node.remark }}</span>
            </div>
            <div class="flex items-start gap-3 cursor-pointer" @click="fetchNodeStatus(node.id)">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">状态:</span>
              <div class="flex-1 text-[11px] flex items-center gap-2 flex-wrap">
                <template v-if="nodeStatuses[node.id]?.status === 'loading'">
                  <RefreshCw class="h-3 w-3 animate-spin" />检测中...
                </template>
                <template v-else-if="nodeStatuses[node.id]?.status === 'online' || node.status === 'online'">
                  <span v-if="nodeStatuses[node.id]?.version" class="text-muted-foreground">v{{ nodeStatuses[node.id].version }}</span>
                  <span v-if="node.metrics?.cpu_percent !== undefined && node.metrics?.cpu_percent !== 0" :class="getLoadColor(node.metrics?.cpu_percent)">CPU: {{ node.metrics?.cpu_percent.toFixed(1) }}%</span>
                  <span v-if="node.metrics?.mem_percent !== undefined && node.metrics?.mem_percent !== 0" :class="getLoadColor(node.metrics?.mem_percent)">Mem: {{ node.metrics?.mem_percent.toFixed(1) }}%</span>
                </template>
                <template v-else>
                  <span class="text-destructive font-medium">离线</span>
                </template>
              </div>
            </div>
          </div>
          <div class="grid grid-cols-4 items-center pt-2 mt-2 border-t border-border/40 -mx-1">
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1 hover:bg-primary/5 rounded-none" @click="showDetail(node)">
              <Eye class="h-3.5 w-3.5" />详情
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1 hover:bg-primary/5 rounded-none border-l border-border/10" @click="handleTravel(node.id)">
              <ExternalLink class="h-3.5 w-3.5" />穿越
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1 hover:bg-primary/5 rounded-none border-l border-border/10" @click="openEditDialog(node)">
              <Edit2 class="h-3.5 w-3.5" />编辑
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1 hover:bg-primary/5 rounded-none border-l border-border/10" @click="confirmDelete(node.id)">
              <Trash2 class="h-3.5 w-3.5" />删除
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ isEditing ? '编辑子节点' : '生成子节点专属接入密钥' }}</DialogTitle>
          <DialogDescription>
            {{ isEditing ? '修改节点的备注名称信息。' : '保存后请将此密钥粘贴到子节点的配置界面中建立连接。' }}
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid gap-2">
            <Label for="name">节点名称 <span class="text-destructive">*</span></Label>
            <Input id="name" v-model="currentForm.name" placeholder="例如：海外节点-洛杉矶" autocomplete="off" />
          </div>
          <div class="grid gap-2" v-if="!isEditing">
            <Label>专属互联密钥 (Token) <span class="text-destructive">*</span></Label>
            <div class="flex items-center gap-2">
              <Input v-model="currentForm.token" readonly class="font-mono text-sm bg-muted/30 placeholder:font-sans" />
              <Button variant="outline" size="icon" @click="currentForm.token = generateRandomToken()" title="重新生成">
                <RefreshCw class="h-4 w-4" />
              </Button>
              <Button variant="outline" size="icon" @click="handleCopy(currentForm.token || '')" :title="copied ? '已复制' : '复制'">
                <Copy class="h-4 w-4" :class="{ 'text-green-500': copied }" />
              </Button>
            </div>
            <p class="text-xs text-orange-500 mt-1">请务必复制上方密钥，一旦关闭窗口将无法再次查看完整密钥。</p>
          </div>
          <div class="grid gap-2">
            <Label for="remark">备注</Label>
            <Input id="remark" v-model="currentForm.remark" placeholder="选填，关于该节点的附加说明" autocomplete="off" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">取消</Button>
          <Button @click="handleSave">保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 删除确认弹窗 -->
    <AlertDialog :open="showDeleteConfirm" @update:open="showDeleteConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确定删除该互联节点？</AlertDialogTitle>
          <AlertDialogDescription>
            此操作不可恢复，节点连接信息将被永久删除。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="handleDelete" class="bg-destructive text-destructive-foreground hover:bg-destructive/90">确认删除</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- 详情弹窗 -->
    <Dialog :open="detailOpen" @update:open="detailOpen = $event">
      <DialogContent class="sm:max-w-[600px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <span>节点详情</span>
            <span class="text-sm font-normal text-muted-foreground">({{ selectedNodeName }})</span>
          </DialogTitle>
          <DialogDescription>
            查看子节点的系统环境、硬件指标和任务调度器运行状态。
          </DialogDescription>
        </DialogHeader>

        <!-- 加载中 -->
        <div v-if="detailLoading" class="py-12 flex flex-col items-center justify-center space-y-4 animate-in fade-in duration-300">
          <div class="relative flex items-center justify-center">
            <!-- 外圈渐变呼吸环 -->
            <div class="absolute h-10 w-10 rounded-full border border-primary/25 animate-ping"></div>
            <!-- 旋转环 -->
            <div class="h-10 w-10 rounded-full border-2 border-primary/10 border-t-primary animate-spin"></div>
          </div>
          <span class="text-xs font-medium text-muted-foreground/80 tracking-wider animate-pulse">正在获取远程数据...</span>
        </div>

        <!-- 加载失败或无数据 -->
        <div v-else-if="!selectedNodeDetail" class="py-12 flex flex-col items-center justify-center gap-2 text-destructive">
          <p class="text-sm font-medium">获取数据失败</p>
          <p class="text-xs text-muted-foreground">子节点可能处于离线状态，或者网络连接超时。</p>
        </div>

        <!-- 数据展示 -->
        <div v-else class="space-y-6 py-2">
          <!-- 1. 硬件状态 -->
          <div class="space-y-3">
            <h3 class="text-sm font-semibold border-b pb-1">系统资源负载</h3>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <!-- CPU -->
              <div class="rounded-lg border p-3 bg-muted/10 space-y-2">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">CPU 使用率</span>
                  <span class="font-medium" :class="getLoadColor(selectedNodeDetail.host?.cpu_percent)">{{ selectedNodeDetail.host?.cpu_percent?.toFixed(1) }}%</span>
                </div>
                <div class="w-full bg-muted rounded-full h-2">
                  <div class="h-2 rounded-full transition-all duration-300" :class="selectedNodeDetail.host?.cpu_percent >= 80 ? 'bg-destructive' : selectedNodeDetail.host?.cpu_percent >= 50 ? 'bg-yellow-500' : 'bg-green-500'" :style="{ width: `${selectedNodeDetail.host?.cpu_percent || 0}%` }"></div>
                </div>
              </div>
              <!-- 内存 -->
              <div class="rounded-lg border p-3 bg-muted/10 space-y-2">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">内存使用率</span>
                  <span class="font-medium" :class="getLoadColor(selectedNodeDetail.host?.mem_percent)">{{ selectedNodeDetail.host?.mem_percent?.toFixed(1) }}%</span>
                </div>
                <div class="w-full bg-muted rounded-full h-2">
                  <div class="h-2 rounded-full transition-all duration-300" :class="selectedNodeDetail.host?.mem_percent >= 80 ? 'bg-destructive' : selectedNodeDetail.host?.mem_percent >= 50 ? 'bg-yellow-500' : 'bg-green-500'" :style="{ width: `${selectedNodeDetail.host?.mem_percent || 0}%` }"></div>
                </div>
                <div class="text-[10px] text-muted-foreground flex justify-between">
                  <span>{{ formatBytes(selectedNodeDetail.host?.mem_used) }}</span>
                  <span>{{ formatBytes(selectedNodeDetail.host?.mem_total) }}</span>
                </div>
              </div>
              <!-- 磁盘 -->
              <div class="rounded-lg border p-3 bg-muted/10 space-y-2">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">磁盘使用率</span>
                  <span class="font-medium" :class="getLoadColor(selectedNodeDetail.host?.disk_percent)">{{ selectedNodeDetail.host?.disk_percent?.toFixed(1) }}%</span>
                </div>
                <div class="w-full bg-muted rounded-full h-2">
                  <div class="h-2 rounded-full transition-all duration-300" :class="selectedNodeDetail.host?.disk_percent >= 80 ? 'bg-destructive' : selectedNodeDetail.host?.disk_percent >= 50 ? 'bg-yellow-500' : 'bg-green-500'" :style="{ width: `${selectedNodeDetail.host?.disk_percent || 0}%` }"></div>
                </div>
                <div class="text-[10px] text-muted-foreground flex justify-between">
                  <span>{{ formatBytes(selectedNodeDetail.host?.disk_used) }}</span>
                  <span>{{ formatBytes(selectedNodeDetail.host?.disk_total) }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 2. 环境信息 -->
          <div class="space-y-3">
            <h3 class="text-sm font-semibold border-b pb-1">运行环境与系统</h3>
            <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">操作系统</span>
                <span class="font-medium uppercase">{{ selectedNodeDetail.env?.os }} ({{ selectedNodeDetail.env?.arch }})</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">系统平台</span>
                <span class="font-medium truncate max-w-[160px]" :title="selectedNodeDetail.host?.platform">{{ selectedNodeDetail.host?.platform || '-' }}</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">CPU 核心数</span>
                <span class="font-medium">{{ selectedNodeDetail.env?.num_cpu }} 核</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">Goroutine 数量</span>
                <span class="font-medium">{{ selectedNodeDetail.env?.goroutines }}</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">Go 编译版本</span>
                <span class="font-medium">{{ selectedNodeDetail.env?.go_version }}</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40">
                <span class="text-muted-foreground">节点运行时间</span>
                <span class="font-medium">{{ formatUptime(selectedNodeDetail.host?.uptime) }}</span>
              </div>
            </div>
          </div>

          <!-- 3. 物理隧道连接状态 -->
          <div class="space-y-3" v-if="selectedNodeDetail.tunnel_connected">
            <h3 class="text-sm font-semibold border-b pb-1">物理隧道连接状态</h3>
            <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
              <div class="flex justify-between py-1 border-b border-border/40 col-span-2">
                <span class="text-muted-foreground">隧道内部通讯地址</span>
                <span class="font-medium truncate max-w-[280px]" :title="selectedNodeDetail.tunnel_url">{{ selectedNodeDetail.tunnel_url }}</span>
              </div>
              <div class="flex justify-between py-1 border-b border-border/40 col-span-2" v-if="selectedNodeDetail.host?.tx_bytes !== undefined || selectedNodeDetail.host?.rx_bytes !== undefined">
                <span class="text-muted-foreground">实时累加隧道流量</span>
                <span class="font-medium">
                  <span class="text-green-500" title="发送 (TX)">{{ formatBytes(selectedNodeDetail.host?.tx_bytes) }}</span>
                  <span class="mx-1 text-muted-foreground">/</span>
                  <span class="text-blue-500" title="接收 (RX)">{{ formatBytes(selectedNodeDetail.host?.rx_bytes) }}</span>
                </span>
              </div>
            </div>
          </div>

          <!-- 4. 调度器与任务统计 -->
          <div class="space-y-3" v-if="selectedNodeDetail.scheduler">
            <h3 class="text-sm font-semibold border-b pb-1">任务调度统计</h3>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
              <div class="rounded border p-2 bg-muted/5">
                <div class="text-[10px] text-muted-foreground">计划中任务</div>
                <div class="text-lg font-bold tabular-nums mt-0.5">{{ selectedNodeDetail.scheduler.scheduled ?? 0 }}</div>
              </div>
              <div class="rounded border p-2 bg-muted/5">
                <div class="text-[10px] text-muted-foreground">正在运行任务</div>
                <div class="text-lg font-bold tabular-nums mt-0.5 text-green-500">{{ selectedNodeDetail.scheduler.running ?? 0 }}</div>
              </div>
              <div class="rounded border p-2 bg-muted/5">
                <div class="text-[10px] text-muted-foreground">队列任务积压</div>
                <div class="text-lg font-bold tabular-nums mt-0.5">{{ selectedNodeDetail.scheduler.queue_size ?? 0 }}</div>
              </div>
              <div class="rounded border p-2 bg-muted/5">
                <div class="text-[10px] text-muted-foreground">并发工作协程</div>
                <div class="text-lg font-bold tabular-nums mt-0.5">{{ selectedNodeDetail.scheduler.worker_count ?? 0 }}</div>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
