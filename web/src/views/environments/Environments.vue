<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import { Plus, Pencil, Trash2, Eye, EyeOff, Search, AlertTriangle, Terminal, Zap, ZapOff, Shield, Tag, Link, X } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import TagInput from '@/components/TagInput.vue'
import { api, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ENV_TYPE } from '@/constants'
import { format } from 'date-fns'
import { Badge } from '@/components/ui/badge'

import EditEnvDialog from './components/EditEnvDialog.vue'
import DeleteEnvDialog from './components/DeleteEnvDialog.vue'
import DependentTasksDialog from './components/DependentTasksDialog.vue'

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy-MM-dd HH:mm:ss')
  } catch {
    return dateStr
  }
}

const { pageSize } = useSiteSettings()

const envVars = ref<EnvVar[]>([])
const showValues = ref<Record<string, boolean>>({})

const filterName = ref('')
const filterTags = ref('')
const currentPage = ref(1)
const total = ref(0)
const activeTab = ref<string>(ENV_TYPE.NORMAL)
const isSecretSet = ref(true)
const showSecretTip = ref(localStorage.getItem('bhp_hide_secret_tip') !== 'true')

function closeSecretTip() {
  showSecretTip.value = false
  localStorage.setItem('bhp_hide_secret_tip', 'true')
}
let searchTimer: ReturnType<typeof setTimeout> | null = null

const editDialogRef = ref<InstanceType<typeof EditEnvDialog> | null>(null)
const deleteDialogRef = ref<InstanceType<typeof DeleteEnvDialog> | null>(null)
const dependentTasksDialogRef = ref<InstanceType<typeof DependentTasksDialog> | null>(null)

async function checkSecretStatus() {
  try {
    isSecretSet.value = await api.env.secretStatus()
    if (!isSecretSet.value) {
      toast.warning('未检测到加密机密秘钥，请在启动时配置 BAIHU_SECRET_KEY 环境变量')
    }
  } catch (error) {
    console.error('检查秘钥状态失败', error)
  }
}

async function loadEnvVars() {
  try {
    const res = await api.env.list({ 
      page: currentPage.value, 
      page_size: pageSize.value, 
      name: filterName.value || undefined, 
      type: activeTab.value,
      tags: filterTags.value || undefined
    })
    envVars.value = res.data
    total.value = res.total
    // 初始化显示状态，根据数据库的 hidden 状态同步显示
    res.data.forEach(env => {
      showValues.value[env.id] = !env.hidden
    })
  } catch { toast.error('加载环境变量失败') }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadEnvVars()
  }, 300)
}

watch(activeTab, (val) => {
  currentPage.value = 1
  if (val === ENV_TYPE.SECRET) {
    checkSecretStatus()
  }
  loadEnvVars()
})

function handlePageChange(page: number) {
  currentPage.value = page
  loadEnvVars()
}

function openCreate() {
  editDialogRef.value?.openCreate(activeTab.value)
}

function openEdit(env: EnvVar) {
  editDialogRef.value?.openEdit(env)
}

function openDependentTasks(env: EnvVar) {
  dependentTasksDialogRef.value?.open(env)
}

function confirmDelete(id: string) {
  deleteDialogRef.value?.confirmDelete(id, activeTab.value)
}

function toggleShow(id: string) {
  showValues.value[id] = !showValues.value[id]
}

async function toggleEnabled(env: EnvVar) {
  try {
    await api.env.update(env.id, { ...env, enabled: !env.enabled })
    env.enabled = !env.enabled
    toast.success(env.enabled ? '变量已启用' : '变量已禁用')
  } catch {
    toast.error('操作失败')
  }
}

function maskValue(value: string) {
  return '•'.repeat(Math.min(value.length, 20))
}

const NOTIFY_ENV_KEYS = ['BHPKG_NOTIFY_TOKEN', 'BHPKG_NOTIFY_CHANNEL', 'BHPKG_NOTIFY_URL']
function isNotifyEnv(name: string) {
  return NOTIFY_ENV_KEYS.includes(name)
}

onMounted(() => {
  if (activeTab.value === ENV_TYPE.SECRET) {
    checkSecretStatus()
  }
  loadEnvVars()
})
</script>

<template>
  <Tabs v-model="activeTab" class="space-y-6">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="flex flex-col shrink-0">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">{{ activeTab === ENV_TYPE.SECRET ? '机密管理' : '环境变量' }}</h2>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">
          {{ activeTab === ENV_TYPE.SECRET ? '管理加密敏感数据' : '管理脚本执行时的环境变量' }}
        </p>
      </div>

      <div class="flex flex-row items-center flex-wrap gap-2 w-full md:w-auto md:ml-auto md:justify-end">
        <!-- 搜索与操作 -->
        <div class="flex flex-row items-center gap-2 w-full sm:flex-1 md:flex-none md:w-auto text-sm">
          <div class="relative flex-1 md:flex-none md:w-[150px] lg:w-[200px] group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input v-model="filterName" placeholder="搜索名称..." class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm" @input="handleSearch" />
          </div>
          <div class="w-[120px] lg:w-[150px] shrink-0">
            <TagInput v-model="filterTags" placeholder="标签过滤..." :icon="Tag" multiple :fetchTags="api.env.tags" class="h-9 bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm" @enter="handleSearch" @update:modelValue="handleSearch" />
          </div>
          
          <Button variant="outline" class="h-9 px-3 shrink-0 shadow-sm" @click="openCreate" :disabled="activeTab === ENV_TYPE.SECRET && !isSecretSet">
            <Plus class="h-4 w-4 md:mr-2" /> <span class="hidden md:inline">新建{{ activeTab === ENV_TYPE.SECRET ? '机密' : '变量' }}</span>
          </Button>
        </div>

        <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 rounded-lg w-full sm:w-auto">
          <TabsTrigger value="normal" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
            <Terminal class="w-3.5 h-3.5 opacity-70" />
            <span>变量</span>
          </TabsTrigger>
          <TabsTrigger value="secret" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
            <Shield class="w-3.5 h-3.5 opacity-70" />
            <span>机密</span>
          </TabsTrigger>
        </TabsList>
      </div>
    </div>

    <div v-if="activeTab === ENV_TYPE.SECRET && isSecretSet && showSecretTip" class="bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 text-xs px-4 py-2.5 rounded-lg flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <AlertTriangle class="h-4 w-4 shrink-0" />
        <span><strong>使用说明：</strong>机密数据仅在正式任务执行中被解密并注入为环境变量，<strong>测试运行及临时终端无法获取</strong>。</span>
      </div>
      <Button variant="ghost" size="icon" class="h-5 w-5 rounded-full hover:bg-amber-500/20 text-amber-600 dark:text-amber-400 shrink-0" @click="closeSecretTip">
        <X class="h-3 w-3" />
      </Button>
    </div>

    <div v-if="activeTab === 'secret' && !isSecretSet" class="flex flex-col items-center justify-center p-12 text-center rounded-lg border bg-card border-dashed">
      <div class="h-12 w-12 rounded-full bg-destructive/10 flex items-center justify-center mb-4">
        <AlertTriangle class="h-6 w-6 text-destructive" />
      </div>
      <h3 class="text-lg font-bold mb-2">服务未配置加密秘钥</h3>
      <p class="text-sm text-muted-foreground max-w-md leading-relaxed">
        必须在程序启动时通过环境变量 <code class="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">BAIHU_SECRET_KEY</code> 配置秘钥，才能启用机密管理功能。秘钥将仅存在于内存中，为您提供强安全的数据落盘加密保护。
      </p>
    </div>

    <div v-else class="rounded-lg border bg-card overflow-hidden">
      <!-- ========== 1. 大屏布局 (Large >= 1280px) ========== -->
      <div class="hidden xl:block">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-48 shrink-0">名称</span>
          <span class="flex-1 min-w-0">值 / 内容</span>
          <span class="w-48 shrink-0">备注说明</span>
          <span class="w-40 shrink-0">创建时间</span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-24 shrink-0 text-center">操作</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y text-sm">
          <div v-if="envVars.length === 0" class="text-sm text-muted-foreground text-center py-12">
            {{ activeTab === ENV_TYPE.SECRET ? '暂无机密' : '暂无环境变量' }}
          </div>
          <div v-for="(env, index) in envVars" :key="`large-${env.id}`"
            class="flex items-center gap-4 px-4 py-1.5 hover:bg-muted/30 transition-colors">
            <div class="w-12 shrink-0 pl-1 text-muted-foreground tabular-nums text-[11px]">#{{ total - (currentPage - 1) * pageSize - index }}</div>
            
            <div class="w-48 shrink-0 flex flex-col gap-1 justify-center overflow-hidden">
              <div class="flex items-center gap-1.5 overflow-hidden">
                <code class="font-bold truncate text-[11px] bg-muted/60 px-2 py-0.5 rounded text-zinc-700 dark:text-zinc-200">{{ env.name }}</code>
                <Badge v-if="isNotifyEnv(env.name)" variant="secondary" class="text-[9px] h-3.5 px-1 rounded-sm uppercase font-bold tracking-tighter shrink-0 leading-none">内置</Badge>
              </div>
              <div v-if="env.tags" class="flex items-center gap-1 overflow-hidden">
                <span v-for="tag in env.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag" class="truncate text-[9px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">{{ tag }}</span>
              </div>
            </div>

            <div class="flex-1 min-w-0 text-muted-foreground truncate text-xs px-1">
              <TextOverflow :text="showValues[env.id] ? env.value : maskValue(env.value)" :title="activeTab === ENV_TYPE.SECRET ? '机密内容' : '变量值'" />
            </div>

            <div class="w-48 shrink-0 text-muted-foreground truncate text-xs">
              <TextOverflow :text="env.remark || '-'" title="备注描述" />
            </div>

            <div class="w-40 shrink-0 text-muted-foreground tabular-nums text-[11px] opacity-70">
              {{ formatDate(env.created_at) }}
            </div>

            <div class="w-8 shrink-0 flex justify-center">
              <span class="cursor-pointer" @click="toggleEnabled(env)">
                <div v-if="env.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center">
                  <Zap class="h-3 w-3 text-green-500 fill-green-500" />
                </div>
                <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
                  <ZapOff class="h-3 w-3 text-muted-foreground" />
                </div>
              </span>
            </div>

            <div class="w-24 shrink-0 flex justify-center">
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="toggleShow(env.id)" :title="showValues[env.id] ? '隐藏' : '显示'">
                <Eye v-if="!showValues[env.id]" class="h-3 w-3" />
                <EyeOff v-else class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openDependentTasks(env)" title="依赖任务">
                <Link class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEdit(env)" title="编辑">
                <Pencil class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6 text-destructive" @click="confirmDelete(env.id)" title="删除">
                <Trash2 class="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 2. 中屏布局 (Small to Large) ========== -->
      <div class="hidden sm:block xl:hidden">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-48 shrink-0">名称</span>
          <span class="flex-1 min-w-0">值 / 内容</span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-24 shrink-0 text-center">操作</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y text-sm">
          <div v-for="(env, index) in envVars" :key="`med-${env.id}`"
            class="flex items-center gap-4 px-4 py-2 hover:bg-muted/30 transition-colors">
            <div class="w-12 shrink-0 pl-1 text-muted-foreground tabular-nums text-[10px]">#{{ total - (currentPage - 1) * pageSize - index }}</div>
            
            <div class="w-48 shrink-0 flex flex-col gap-1 justify-center overflow-hidden">
              <div class="flex items-center gap-1.5 overflow-hidden">
                <code class="font-bold truncate text-[11px] bg-muted/60 px-2 py-0.5 rounded text-zinc-700 dark:text-zinc-200">{{ env.name }}</code>
                <Badge v-if="isNotifyEnv(env.name)" variant="secondary" class="text-[9px] h-3.5 px-1 rounded-sm uppercase font-bold tracking-tighter shrink-0 leading-none">内置</Badge>
              </div>
              <div v-if="env.tags" class="flex items-center gap-1 overflow-hidden">
                <span v-for="tag in env.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag" class="truncate text-[9px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">{{ tag }}</span>
              </div>
            </div>

            <div class="flex-1 min-w-0 text-muted-foreground truncate text-xs">
               <TextOverflow :text="showValues[env.id] ? env.value : maskValue(env.value)" />
            </div>

            <div class="w-8 shrink-0 flex justify-center">
              <span class="cursor-pointer" @click="toggleEnabled(env)">
                <div v-if="env.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center">
                  <Zap class="h-3 w-3 text-green-500 fill-green-500" />
                </div>
                <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
                  <ZapOff class="h-3 w-3 text-muted-foreground" />
                </div>
              </span>
            </div>

            <div class="w-24 shrink-0 flex justify-center">
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="toggleShow(env.id)">
                <Eye v-if="!showValues[env.id]" class="h-3 w-3" />
                <EyeOff v-else class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openDependentTasks(env)" title="依赖任务">
                <Link class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEdit(env)">
                <Pencil class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6 text-destructive" @click="confirmDelete(env.id)">
                <Trash2 class="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 3. 小屏布局 (Small < 640px) ========== -->
      <div class="divide-y sm:hidden">
        <div v-if="envVars.length === 0" class="text-sm text-muted-foreground text-center py-12">
          {{ activeTab === ENV_TYPE.SECRET ? '暂无机密' : '暂无环境变量' }}
        </div>
        <div v-for="(env, index) in envVars" :key="`small-${env.id}`" class="p-3 hover:bg-muted/50 transition-colors">
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex flex-col gap-1 flex-1 min-w-0 pr-2">
              <div class="flex items-center gap-2">
                <span class="text-[10px] text-muted-foreground tabular-nums flex-shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                <code class="font-bold text-xs bg-muted/60 px-2 py-0.5 rounded truncate text-zinc-700 dark:text-zinc-200">{{ env.name }}</code>
                <Badge v-if="isNotifyEnv(env.name)" variant="secondary" class="text-[8px] h-3.5 px-1 rounded-sm uppercase font-bold tracking-tighter leading-none shrink-0">内置</Badge>
              </div>
              <div v-if="env.tags" class="flex items-center gap-1 pl-6 overflow-hidden">
                <span v-for="tag in env.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag" class="truncate text-[9px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">{{ tag }}</span>
              </div>
            </div>
            <span @click="toggleEnabled(env)" class="cursor-pointer">
              <div v-if="env.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center">
                <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
              </div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
                <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </span>
          </div>
          
          <!-- 详情信息 -->
          <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
            <div class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">内容:</span>
              <div class="flex-1 min-w-0 text-foreground break-all line-clamp-2">
                <TextOverflow :text="showValues[env.id] ? env.value : maskValue(env.value)" />
              </div>
            </div>
            <div v-if="env.remark" class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">备注:</span>
              <span class="flex-1 text-[11px] line-clamp-1">{{ env.remark }}</span>
            </div>
          </div>

          <div class="grid grid-cols-4 items-center pt-2 mt-2 border-t border-border/40 -mx-1">
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none" @click="toggleShow(env.id)">
              <Eye v-if="!showValues[env.id]" class="h-3.5 w-3.5" />
              <EyeOff v-else class="h-3.5 w-3.5" />
              {{ showValues[env.id] ? '隐藏' : '显示' }}
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="openDependentTasks(env)">
              <Link class="h-3.5 w-3.5" />任务
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="openEdit(env)">
              <Pencil class="h-3.5 w-3.5" />编辑
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-destructive/5 text-destructive rounded-none border-l border-border/10" @click="confirmDelete(env.id)">
              <Trash2 class="h-3.5 w-3.5" />删除
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="mt-4">
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <EditEnvDialog ref="editDialogRef" @saved="loadEnvVars" />
    <DeleteEnvDialog ref="deleteDialogRef" @deleted="loadEnvVars" />
    <DependentTasksDialog ref="dependentTasksDialogRef" />
  </Tabs>
</template>
