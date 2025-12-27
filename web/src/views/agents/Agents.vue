<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Trash2, Edit, Copy, Key, Server, Search, Check, X, Download, RotateCw } from 'lucide-vue-next'
import { api, type Agent } from '@/api'
import { toast } from 'vue-sonner'
import TextOverflow from '@/components/TextOverflow.vue'

const agents = ref<Agent[]>([])
const pendingAgents = ref<Agent[]>([])
const loading = ref(false)
const searchQuery = ref('')
const activeTab = ref('approved')
const agentVersion = ref('')
const platforms = ref<{ os: string; arch: string; filename: string }[]>([])
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const showTokenDialog = ref(false)
const showDownloadDialog = ref(false)
const formData = ref({ name: '', description: '' })
const editingAgent = ref<Agent | null>(null)
const deletingAgent = ref<Agent | null>(null)
const currentToken = ref('')
let refreshTimer: ReturnType<typeof setInterval> | null = null

const filteredAgents = computed(() => {
  if (!searchQuery.value) return agents.value
  const q = searchQuery.value.toLowerCase()
  return agents.value.filter(a => 
    a.name.toLowerCase().includes(q) || 
    a.hostname?.toLowerCase().includes(q) ||
    a.ip?.toLowerCase().includes(q)
  )
})

async function loadAgents() {
  loading.value = true
  try {
    const [agentList, pendingList, versionInfo] = await Promise.all([
      api.agents.list(),
      api.agents.listPending(),
      api.agents.getVersion()
    ])
    agents.value = agentList
    pendingAgents.value = pendingList
    agentVersion.value = versionInfo.version || ''
    platforms.value = versionInfo.platforms || []
  } catch {
    toast.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function approveAgent(agent: Agent) {
  try {
    const approved = await api.agents.approve(agent.id)
    currentToken.value = approved.token
    showTokenDialog.value = true
    await loadAgents()
    toast.success('已通过审核')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

async function rejectAgent(agent: Agent) {
  try {
    await api.agents.reject(agent.id)
    await loadAgents()
    toast.success('已拒绝')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

function openEditDialog(agent: Agent) {
  editingAgent.value = agent
  formData.value = { name: agent.name, description: agent.description }
  showEditDialog.value = true
}

async function updateAgent() {
  if (!editingAgent.value || !formData.value.name.trim()) return
  try {
    await api.agents.update(editingAgent.value.id, { ...formData.value, enabled: editingAgent.value.enabled })
    showEditDialog.value = false
    await loadAgents()
    toast.success('更新成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '更新失败')
  }
}

async function toggleEnabled(agent: Agent) {
  try {
    await api.agents.update(agent.id, { name: agent.name, description: agent.description, enabled: !agent.enabled })
    await loadAgents()
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

function confirmDelete(agent: Agent) {
  deletingAgent.value = agent
  showDeleteDialog.value = true
}

async function deleteAgent() {
  if (!deletingAgent.value) return
  try {
    await api.agents.delete(deletingAgent.value.id)
    showDeleteDialog.value = false
    await loadAgents()
    toast.success('删除成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '删除失败')
  }
}

async function regenerateToken(agent: Agent) {
  try {
    const res = await api.agents.regenerateToken(agent.id)
    currentToken.value = res.token
    showTokenDialog.value = true
    toast.success('Token 已重新生成')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

async function forceUpdate(agent: Agent) {
  try {
    await api.agents.forceUpdate(agent.id)
    toast.success('已标记强制更新')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

function copyToken() {
  navigator.clipboard.writeText(currentToken.value)
  toast.success('已复制')
}

function downloadAgent(os: string, arch: string) {
  window.open(api.agents.downloadUrl(os, arch), '_blank')
}

function getPlatformLabel(os: string, arch: string) {
  const osLabels: Record<string, string> = { linux: 'Linux', windows: 'Windows', darwin: 'macOS' }
  const archLabels: Record<string, string> = { amd64: 'x64', arm64: 'ARM64', '386': 'x86' }
  return `${osLabels[os] || os} ${archLabels[arch] || arch}`
}

onMounted(() => {
  loadAgents()
  refreshTimer = setInterval(loadAgents, 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>


<template>
  <div class="space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">Agent 管理</h2>
        <p class="text-muted-foreground text-sm">管理远程执行代理</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="searchQuery" placeholder="搜索..." class="h-9 pl-8 w-full sm:w-48 text-sm" />
        </div>
        <Button variant="outline" size="sm" class="h-9" @click="showDownloadDialog = true">
          <Download class="h-4 w-4 mr-1.5" />下载
        </Button>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadAgents" :disabled="loading">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        </Button>
      </div>
    </div>

    <Tabs v-model="activeTab">
      <TabsList>
        <TabsTrigger value="approved">已注册</TabsTrigger>
        <TabsTrigger value="pending" class="relative">
          未注册
          <Badge v-if="pendingAgents.length > 0" variant="destructive" class="ml-1.5 h-5 min-w-5 px-1">{{ pendingAgents.length }}</Badge>
        </TabsTrigger>
      </TabsList>

      <TabsContent value="approved" class="mt-4">
        <div class="rounded-lg border bg-card overflow-x-auto">
          <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[800px]">
            <span class="w-28">名称</span>
            <span class="w-16 text-center">状态</span>
            <span class="w-24">IP</span>
            <span class="w-24">主机名</span>
            <span class="w-16">版本</span>
            <span class="w-28">构建时间</span>
            <span class="flex-1">描述</span>
            <span class="w-14 text-center">启用</span>
            <span class="w-28 text-center">操作</span>
          </div>
          <div class="divide-y min-w-[800px]">
            <div v-if="filteredAgents.length === 0" class="text-center py-8 text-muted-foreground">
              <Server class="h-8 w-8 mx-auto mb-2 opacity-50" />
              {{ searchQuery ? '无匹配结果' : '暂无 Agent' }}
            </div>
            <div v-for="agent in filteredAgents" :key="agent.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
              <span class="w-28 font-medium text-sm truncate">{{ agent.name }}</span>
              <span class="w-16 flex justify-center">
                <Badge :variant="agent.status === 'online' ? 'default' : 'secondary'" class="text-xs">{{ agent.status === 'online' ? '在线' : '离线' }}</Badge>
              </span>
              <span class="w-24 text-sm text-muted-foreground truncate">{{ agent.ip || '-' }}</span>
              <span class="w-24 text-sm text-muted-foreground truncate">{{ agent.hostname || '-' }}</span>
              <span class="w-16 text-sm text-muted-foreground">{{ agent.version || '-' }}</span>
              <span class="w-28 text-sm text-muted-foreground truncate">{{ agent.build_time || '-' }}</span>
              <span class="flex-1 text-sm text-muted-foreground truncate">
                <TextOverflow :text="agent.description || '-'" title="描述" />
              </span>
              <span class="w-14 flex justify-center">
                <Switch :checked="agent.enabled" @update:checked="toggleEnabled(agent)" />
              </span>
              <span class="w-28 flex justify-center gap-1">
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="forceUpdate(agent)" title="强制更新">
                  <RotateCw class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="regenerateToken(agent)" title="重新生成 Token">
                  <Key class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEditDialog(agent)">
                  <Edit class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(agent)">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </span>
            </div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="pending" class="mt-4">
        <div class="rounded-lg border bg-card overflow-x-auto">
          <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[500px]">
            <span class="w-32">名称</span>
            <span class="w-28">IP</span>
            <span class="w-28">主机名</span>
            <span class="w-20">版本</span>
            <span class="flex-1">注册时间</span>
            <span class="w-24 text-center">操作</span>
          </div>
          <div class="divide-y min-w-[500px]">
            <div v-if="pendingAgents.length === 0" class="text-center py-8 text-muted-foreground">
              <Server class="h-8 w-8 mx-auto mb-2 opacity-50" />暂无未注册的 Agent
            </div>
            <div v-for="agent in pendingAgents" :key="agent.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
              <span class="w-32 font-medium text-sm truncate">{{ agent.name }}</span>
              <span class="w-28 text-sm text-muted-foreground truncate">{{ agent.ip || '-' }}</span>
              <span class="w-28 text-sm text-muted-foreground truncate">{{ agent.hostname || '-' }}</span>
              <span class="w-20 text-sm text-muted-foreground">{{ agent.version || '-' }}</span>
              <span class="flex-1 text-sm text-muted-foreground">{{ agent.created_at }}</span>
              <span class="w-24 flex justify-center gap-1">
                <Button variant="ghost" size="icon" class="h-7 w-7 text-green-600" @click="approveAgent(agent)" title="通过">
                  <Check class="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="rejectAgent(agent)" title="拒绝">
                  <X class="h-4 w-4" />
                </Button>
              </span>
            </div>
          </div>
        </div>
      </TabsContent>
    </Tabs>

    <!-- 编辑对话框 -->
    <Dialog v-model:open="showEditDialog">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle>编辑 Agent</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">名称</Label>
            <Input v-model="formData.name" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">描述</Label>
            <Input v-model="formData.description" class="col-span-3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showEditDialog = false">取消</Button>
          <Button @click="updateAgent">保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 删除确认 -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>确定要删除 "{{ deletingAgent?.name }}" 吗？此操作不可恢复。</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteAgent">删除</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Token 对话框 -->
    <Dialog v-model:open="showTokenDialog">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Agent Token</DialogTitle>
          <DialogDescription>Token 已下发给 Agent，此处仅供查看</DialogDescription>
        </DialogHeader>
        <div class="py-4">
          <div class="flex items-center gap-2">
            <code class="flex-1 bg-muted px-3 py-2 rounded text-xs font-mono break-all">{{ currentToken }}</code>
            <Button variant="outline" size="icon" class="shrink-0" @click="copyToken">
              <Copy class="h-4 w-4" />
            </Button>
          </div>
        </div>
        <DialogFooter>
          <Button @click="showTokenDialog = false">关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 下载对话框 -->
    <Dialog v-model:open="showDownloadDialog">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>下载 Agent</DialogTitle>
          <DialogDescription v-if="agentVersion">当前版本: {{ agentVersion }}</DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-4">
          <div v-if="platforms.length === 0" class="text-center py-4 text-muted-foreground">
            暂无可用的 Agent 程序，请先构建并上传到 data/agent 目录
          </div>
          <div v-else class="grid gap-2">
            <Button v-for="p in platforms" :key="p.filename" variant="outline" class="justify-start" @click="downloadAgent(p.os, p.arch)">
              <Download class="h-4 w-4 mr-2" />{{ getPlatformLabel(p.os, p.arch) }}
            </Button>
          </div>
          <div class="border-t pt-4">
            <p class="text-sm font-medium mb-2">使用说明</p>
            <div class="text-xs text-muted-foreground space-y-1.5">
              <p>1. 下载对应平台的 Agent 压缩包并解压</p>
              <p>2. 修改 config.example.ini 为 config.ini，设置 server_url</p>
              <p>3. 运行 <code class="bg-muted px-1 rounded">./baihu-agent start</code> 启动</p>
              <p>4. 在本页面"未注册"标签中审核通过</p>
              <p>5. 可选: <code class="bg-muted px-1 rounded">./baihu-agent install</code> 设置开机自启</p>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button @click="showDownloadDialog = false">关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
