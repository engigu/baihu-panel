<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Trash2, Edit, Copy, Server, Search, Download, RotateCw, Plus, Ticket, Power, PowerOff } from 'lucide-vue-next'
import { api, type Agent, type AgentRegCode } from '@/api'
import { toast } from 'vue-sonner'
import TextOverflow from '@/components/TextOverflow.vue'

const agents = ref<Agent[]>([])
const regCodes = ref<AgentRegCode[]>([])
const loading = ref(false)
const searchQuery = ref('')
const activeTab = ref('agents')
const agentVersion = ref('')
const platforms = ref<{ os: string; arch: string; filename: string }[]>([])
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const showDownloadDialog = ref(false)
const showRegCodeDialog = ref(false)
const formData = ref({ name: '', description: '' })
const regCodeForm = ref({ remark: '', max_uses: 0, expires_at: '' })
const editingAgent = ref<Agent | null>(null)
const deletingAgent = ref<Agent | null>(null)
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

// 判断 Agent 是否在线（last_seen 在 2 分钟内）
function isOnline(agent: Agent): boolean {
  if (!agent.last_seen) return false
  const lastSeen = new Date(agent.last_seen)
  const now = new Date()
  const diffMs = now.getTime() - lastSeen.getTime()
  return diffMs < 2 * 60 * 1000 // 2 分钟
}

async function loadAgents() {
  loading.value = true
  try {
    const [agentList, versionInfo, codeList] = await Promise.all([
      api.agents.list(),
      api.agents.getVersion(),
      api.agents.listRegCodes()
    ])
    agents.value = agentList
    agentVersion.value = versionInfo.version || ''
    platforms.value = versionInfo.platforms || []
    regCodes.value = codeList
  } catch {
    toast.error('加载失败')
  } finally {
    loading.value = false
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
    const newEnabled = !agent.enabled
    await api.agents.update(agent.id, { name: agent.name, description: agent.description, enabled: newEnabled })
    await loadAgents()
    toast.success(`${agent.name} 已${newEnabled ? '启用' : '禁用'}`)
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

async function forceUpdate(agent: Agent) {
  try {
    await api.agents.forceUpdate(agent.id)
    toast.success('已标记强制更新')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

function copyRegCode(code: string) {
  navigator.clipboard.writeText(code)
  toast.success('已复制')
}

async function createRegCode() {
  try {
    await api.agents.createRegCode({
      remark: regCodeForm.value.remark,
      max_uses: regCodeForm.value.max_uses,
      expires_at: regCodeForm.value.expires_at || undefined
    })
    showRegCodeDialog.value = false
    regCodeForm.value = { remark: '', max_uses: 0, expires_at: '' }
    await loadAgents()
    toast.success('创建成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '创建失败')
  }
}

async function deleteRegCode(id: number) {
  try {
    await api.agents.deleteRegCode(id)
    await loadAgents()
    toast.success('删除成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '删除失败')
  }
}

function isRegCodeExpired(code: AgentRegCode) {
  if (!code.expires_at) return false
  return new Date(code.expires_at) < new Date()
}

function isRegCodeExhausted(code: AgentRegCode) {
  return code.max_uses > 0 && code.used_count >= code.max_uses
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
        <TabsTrigger value="agents">Agent 列表</TabsTrigger>
        <TabsTrigger value="regcodes">
          <Ticket class="h-4 w-4 mr-1" />令牌
        </TabsTrigger>
      </TabsList>

      <TabsContent value="agents" class="mt-4">
        <div class="rounded-lg border bg-card overflow-x-auto">
          <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[900px]">
            <span class="w-6"></span>
            <span class="w-28">名称</span>
            <span class="w-24">IP</span>
            <span class="w-24">主机名</span>
            <span class="w-16">版本</span>
            <span class="w-28">构建时间</span>
            <span class="w-36">心跳时间</span>
            <span class="flex-1">描述</span>
            <span class="w-28 text-center">操作</span>
          </div>
          <div class="divide-y min-w-[900px]">
            <div v-if="filteredAgents.length === 0" class="text-center py-8 text-muted-foreground">
              <Server class="h-8 w-8 mx-auto mb-2 opacity-50" />
              {{ searchQuery ? '无匹配结果' : '暂无 Agent' }}
            </div>
            <div v-for="agent in filteredAgents" :key="agent.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
            <span class="w-6 flex justify-center">
              <span 
                class="relative flex h-2.5 w-2.5" 
                :title="isOnline(agent) ? '在线' : '离线'"
              >
                <span v-if="isOnline(agent)" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span :class="isOnline(agent) ? 'bg-green-500' : 'bg-gray-400'" class="relative inline-flex rounded-full h-2.5 w-2.5"></span>
              </span>
            </span>
              <span class="w-28 font-medium text-sm truncate" :title="agent.machine_id ? '机器ID: ' + agent.machine_id.slice(0, 16) + '...' : ''">{{ agent.name }}</span>
              <span class="w-24 text-sm text-muted-foreground truncate">{{ agent.ip || '-' }}</span>
              <span class="w-24 text-sm text-muted-foreground truncate">{{ agent.hostname || '-' }}</span>
              <span class="w-16 text-sm text-muted-foreground">{{ agent.version || '-' }}</span>
              <span class="w-28 text-sm text-muted-foreground truncate">{{ agent.build_time || '-' }}</span>
              <span class="w-36 text-sm text-muted-foreground">{{ agent.last_seen || '-' }}</span>
              <span class="flex-1 text-sm text-muted-foreground truncate">
                <TextOverflow :text="agent.description || '-'" title="描述" />
              </span>
              <span class="w-28 flex justify-center gap-1">
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="toggleEnabled(agent)" :title="agent.enabled ? '禁用' : '启用'">
                  <Power v-if="agent.enabled" class="h-3.5 w-3.5 text-green-600" />
                  <PowerOff v-else class="h-3.5 w-3.5 text-gray-400" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="forceUpdate(agent)" title="强制更新">
                  <RotateCw class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEditDialog(agent)" title="编辑">
                  <Edit class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="confirmDelete(agent)" title="删除">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </span>
            </div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="regcodes" class="mt-4">
        <div class="rounded-lg border bg-card overflow-x-auto">
          <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[800px]">
            <span class="w-6"></span>
            <span class="w-[420px]">令牌</span>
            <span class="w-32">备注</span>
            <span class="w-20 text-center">使用次数</span>
            <span class="flex-1">过期时间</span>
            <span class="w-20 flex justify-center">
              <Button size="sm" class="h-7" @click="showRegCodeDialog = true">
                <Plus class="h-3.5 w-3.5 mr-1" />生成
              </Button>
            </span>
          </div>
          <div class="divide-y min-w-[800px]">
            <div v-if="regCodes.length === 0" class="text-center py-8 text-muted-foreground">
              <Ticket class="h-8 w-8 mx-auto mb-2 opacity-50" />暂无令牌
            </div>
            <div v-for="code in regCodes" :key="code.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
              <span class="w-6 flex justify-center">
                <span class="relative flex h-2.5 w-2.5">
                  <span v-if="!isRegCodeExpired(code) && !isRegCodeExhausted(code)" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span :class="!isRegCodeExpired(code) && !isRegCodeExhausted(code) ? 'bg-green-500' : 'bg-gray-400'" class="relative inline-flex rounded-full h-2.5 w-2.5"></span>
                </span>
              </span>
              <code class="w-[420px] font-mono text-xs bg-muted px-2 py-0.5 rounded truncate">{{ code.code }}</code>
              <span class="w-32 text-sm text-muted-foreground truncate">{{ code.remark || '-' }}</span>
              <span class="w-20 text-sm text-muted-foreground text-center">
                {{ code.used_count }}/{{ code.max_uses === 0 ? '∞' : code.max_uses }}
              </span>
              <span class="flex-1 text-sm text-muted-foreground">
                {{ code.expires_at || '永不过期' }}
              </span>
              <span class="w-20 flex justify-center gap-1">
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="copyRegCode(code.code)" title="复制">
                  <Copy class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive" @click="deleteRegCode(code.id)" title="删除">
                  <Trash2 class="h-3.5 w-3.5" />
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

    <!-- 下载对话框 -->
    <Dialog v-model:open="showDownloadDialog">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>下载 Agent</DialogTitle>
          <DialogDescription v-if="agentVersion">当前版本: {{ agentVersion }}</DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-4">
          <div v-if="platforms.length === 0" class="text-center py-4 text-muted-foreground">
            暂无可用的 Agent 程序
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
              <p>3. 在"令牌"标签页生成令牌，填入 config.ini 的 token</p>
              <p>4. 运行 <code class="bg-muted px-1 rounded">./baihu-agent start</code> 启动</p>
              <p>5. 可选: <code class="bg-muted px-1 rounded">./baihu-agent install</code> 设置开机自启</p>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button @click="showDownloadDialog = false">关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- 创建令牌对话框 -->
    <Dialog v-model:open="showRegCodeDialog">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle>生成令牌</DialogTitle>
          <DialogDescription>Agent 使用令牌可直接注册，无需审核</DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">备注</Label>
            <Input v-model="regCodeForm.remark" class="col-span-3" placeholder="可选" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">使用次数</Label>
            <Input v-model.number="regCodeForm.max_uses" type="number" min="0" class="col-span-3" placeholder="0 表示无限制" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label class="text-right">过期时间</Label>
            <Input v-model="regCodeForm.expires_at" type="datetime-local" class="col-span-3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showRegCodeDialog = false">取消</Button>
          <Button @click="createRegCode">生成</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
