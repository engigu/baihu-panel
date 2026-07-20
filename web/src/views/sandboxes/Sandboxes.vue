<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type SandboxProfile } from '@/api'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2, Search, Shield, ShieldAlert, Wrench } from 'lucide-vue-next'
import { format } from 'date-fns'

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy-MM-dd HH:mm:ss')
  } catch {
    return dateStr
  }
}

const list = ref<SandboxProfile[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const filterName = ref('')

const form = ref<Partial<SandboxProfile>>({
  id: '',
  name: '',
  description: '',
  memory_limit: 128,
  nproc_limit: 64,
  uid: 10001,
  gid: 10001,
  hide_system_etc: true
})

const fetchList = async () => {
  loading.value = true
  try {
    const res = await api.sandboxes.list()
    list.value = res
  } catch (err: any) {
    toast.error('获取列表失败', {
      description: err.message || '未知错误'
    })
  } finally {
    loading.value = false
  }
}

const filteredList = computed(() => {
  return list.value.filter(item => {
    const q = filterName.value.toLowerCase().trim()
    if (!q) return true
    return item.name.toLowerCase().includes(q) || (item.description && item.description.toLowerCase().includes(q))
  })
})

const openCreate = () => {
  isEdit.value = false
  form.value = {
    name: '',
    description: '',
    memory_limit: 128,
    nproc_limit: 64,
    uid: 10001,
    gid: 10001,
    hide_system_etc: true
  }
  dialogVisible.value = true
}

const openEdit = (item: SandboxProfile) => {
  isEdit.value = true
  form.value = { ...item }
  dialogVisible.value = true
}

const handleDelete = async (id: string) => {
  if (!confirm('确认删除该沙箱配置吗？绑定此沙箱的任务将被恢复为不使用沙箱直接运行。')) return
  try {
    await api.sandboxes.delete(id)
    toast.success('删除成功')
    fetchList()
  } catch (err: any) {
    toast.error('删除失败', {
      description: err.message || '未知错误'
    })
  }
}

const save = async () => {
  if (!form.value.name) return
  try {
    if (isEdit.value && form.value.id) {
      await api.sandboxes.update(form.value.id, form.value)
      toast.success('更新成功')
    } else {
      await api.sandboxes.create(form.value)
      toast.success('创建成功')
    }
    dialogVisible.value = false
    fetchList()
  } catch (err: any) {
    toast.error('保存失败', {
      description: err.message || '未知错误'
    })
  }
}

const handleRepair = async () => {
  const loadingToast = toast.loading('正在修复并重新生成沙箱执行目录...')
  try {
    await api.sandboxes.repair()
    toast.success('修复并重新生成所有沙箱目录成功', { id: loadingToast })
    fetchList()
  } catch (err: any) {
    toast.error('修复失败', {
      id: loadingToast,
      description: err.message || '未知错误'
    })
  }
}

onMounted(() => {
  fetchList()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 头部栏 -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="flex flex-col shrink-0">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">沙箱管理</h2>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">配置和分配定时任务脚本专用的轻量沙箱隔离模板</p>
      </div>

      <div class="flex flex-row items-center flex-wrap gap-2 w-full md:w-auto md:ml-auto md:justify-end">
        <div class="flex flex-row items-center gap-2 w-full sm:flex-1 md:flex-none md:w-auto text-sm">
          <!-- 搜索框 -->
          <div class="relative flex-1 md:flex-none md:w-[180px] lg:w-[240px] group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input v-model="filterName" placeholder="搜索名称..." class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm" />
          </div>
          <!-- 新建与修复按钮 -->
          <Button variant="outline" class="h-9 px-3 shrink-0 shadow-sm gap-1.5" @click="handleRepair">
            <Wrench class="h-4 w-4 md:mr-1.5 text-muted-foreground" />
            <span class="hidden md:inline">重新修复目录</span>
            <span class="md:hidden text-xs">修复</span>
          </Button>
          <Button variant="outline" class="h-9 px-3 shrink-0 shadow-sm" @click="openCreate">
            <Plus class="h-4 w-4 md:mr-2" />
            <span class="hidden md:inline">新建沙箱</span>
          </Button>
        </div>
      </div>
    </div>

    <!-- 列表展示卡片 -->
    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- ========== 1. 大屏布局 (Large >= 1024px) ========== -->
      <div class="hidden lg:block">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-48 shrink-0">名称</span>
          <span class="flex-1 min-w-0">描述说明</span>
          <span class="w-32 shrink-0">内存限制</span>
          <span class="w-32 shrink-0">并发进程</span>
          <span class="w-40 shrink-0">创建时间</span>
          <span class="w-20 shrink-0 text-center">操作</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y text-sm">
          <div v-if="filteredList.length === 0" class="text-sm text-muted-foreground text-center py-12">
            暂无沙箱配置
          </div>
          <div v-for="(item, index) in filteredList" :key="`large-${item.id}`"
            class="flex items-center gap-4 px-4 py-1.5 hover:bg-muted/30 transition-colors">
            <div class="w-12 shrink-0 pl-1 text-muted-foreground tabular-nums text-[11px]">#{{ index + 1 }}</div>
            
            <div class="w-48 shrink-0 flex items-center gap-1.5 overflow-hidden">
              <code class="font-bold truncate text-[11px] bg-muted/60 px-2 py-0.5 rounded text-zinc-700 dark:text-zinc-200">{{ item.name }}</code>
              <span class="text-[9px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0 font-mono">UID: {{ item.uid }}</span>
            </div>

            <div class="flex-1 min-w-0 text-muted-foreground truncate text-xs">
              {{ item.description || '-' }}
            </div>

            <div class="w-32 shrink-0 text-muted-foreground text-xs font-mono">
              {{ item.memory_limit ? `${item.memory_limit} MB` : '无限制' }}
            </div>

            <div class="w-32 shrink-0 text-muted-foreground text-xs font-mono">
              {{ item.nproc_limit ? `${item.nproc_limit} 个` : '无限制' }}
            </div>

            <div class="w-40 shrink-0 text-muted-foreground tabular-nums text-[11px] opacity-70">
              {{ formatDate(item.created_at) }}
            </div>

            <div class="w-20 shrink-0 flex justify-center gap-1">
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEdit(item)" title="编辑">
                <Pencil class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6 text-destructive" @click="handleDelete(item.id)" title="删除">
                <Trash2 class="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 2. 小屏布局 (Mobile & Tablet < 1024px) ========== -->
      <div class="divide-y lg:hidden">
        <div v-if="filteredList.length === 0" class="text-sm text-muted-foreground text-center py-12">
          暂无沙箱配置
        </div>
        <div v-for="(item, index) in filteredList" :key="`small-${item.id}`" class="p-3 hover:bg-muted/50 transition-colors">
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex flex-col gap-1 flex-1 min-w-0 pr-2">
              <div class="flex items-center gap-2">
                <span class="text-[10px] text-muted-foreground tabular-nums flex-shrink-0">#{{ index + 1 }}</span>
                <code class="font-bold text-xs bg-muted/60 px-2 py-0.5 rounded truncate text-zinc-700 dark:text-zinc-200">{{ item.name }}</code>
                <span class="text-[9px] px-1.5 py-0.5 rounded bg-secondary text-secondary-foreground shrink-0 font-mono">UID: {{ item.uid }}</span>
              </div>
            </div>
          </div>
          
          <!-- 详情信息 -->
          <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
            <div class="flex items-start gap-3" v-if="item.description">
              <span class="w-16 shrink-0 font-medium opacity-70">描述:</span>
              <span class="flex-1 text-foreground">{{ item.description }}</span>
            </div>
            <div class="flex items-start gap-3">
              <span class="w-16 shrink-0 font-medium opacity-70">配置参数:</span>
              <span class="flex-1 text-[11px] font-mono text-foreground flex flex-wrap gap-x-3 gap-y-1">
                <span>内存: {{ item.memory_limit ? `${item.memory_limit}MB` : '无限制' }}</span>
                <span>进程: {{ item.nproc_limit ? `${item.nproc_limit}个` : '无限制' }}</span>
              </span>
            </div>
          </div>

          <div class="grid grid-cols-2 items-center pt-2 mt-2 border-t border-border/40 -mx-1">
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none" @click="openEdit(item)">
              <Pencil class="h-3.5 w-3.5" />编辑
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-destructive/5 text-destructive rounded-none border-l border-border/10" @click="handleDelete(item.id)">
              <Trash2 class="h-3.5 w-3.5" />删除
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新增/编辑模态框 -->
    <Dialog v-model:open="dialogVisible">
      <DialogContent class="w-[calc(100vw-2rem)] max-w-md min-w-0" @open-auto-focus="(e) => e.preventDefault()">
        <DialogHeader>
          <DialogTitle>{{ isEdit ? '编辑沙箱配置' : '新增沙箱配置' }}</DialogTitle>
          <DialogDescription class="sr-only">设置沙箱的内存限制、最大子进程数、系统UID/GID以及是否隐藏敏感文件。</DialogDescription>
        </DialogHeader>
        
        <div class="space-y-4 py-2 min-w-0">
          <div class="space-y-1.5">
            <Label for="name">模板名称</Label>
            <Input id="name" v-model="form.name" placeholder="例如：超轻量爬虫沙箱" class="h-9" />
          </div>

          <div class="space-y-1.5">
            <Label for="description">描述信息</Label>
            <Input id="description" v-model="form.description" placeholder="对该沙箱适用场景的简要介绍" class="h-9" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label for="memory">内存限制 (MB)</Label>
              <Input id="memory" type="number" v-model.number="form.memory_limit" placeholder="128" class="h-9" />
            </div>
            <div class="space-y-1.5">
              <Label for="nproc">最大子进程数</Label>
              <Input id="nproc" type="number" v-model.number="form.nproc_limit" placeholder="64" class="h-9" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label for="uid">系统 UID</Label>
              <Input id="uid" type="number" v-model.number="form.uid" placeholder="10001" class="h-9" />
            </div>
            <div class="space-y-1.5">
              <Label for="gid">系统 GID</Label>
              <Input id="gid" type="number" v-model.number="form.gid" placeholder="10001" class="h-9" />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="dialogVisible = false">取消</Button>
          <Button @click="save" :disabled="!form.name">保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
