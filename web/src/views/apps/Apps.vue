<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Store,
  Package,
  Plus,
  Search,
  RefreshCw,
  Sparkles,
  Loader2,
  ArrowRight,
  User,
  Download,
  Eye,
  Github,
  ExternalLink
} from 'lucide-vue-next'
import { api, type MarketplaceApp } from '@/api'
import ApplyDialog from './ApplyDialog.vue'
import { toast } from 'vue-sonner'
import { useRouter } from 'vue-router'

const router = useRouter()
const searchQuery = ref('')
const imageErrorMap = ref<Record<string, boolean>>({})

// 数据状态
const loadingMarketplace = ref(false)
const marketplaceApps = ref<MarketplaceApp[]>([])

// 统计热度数据
interface MarketStats {
  downloads: Record<string, number>
  pv: Record<string, number>
}
const statsData = ref<MarketStats>({ downloads: {}, pv: {} })

// 弹窗
const showApplyDialog = ref(false)
const selectedMarketApp = ref<MarketplaceApp | null>(null)

const isSearchReadonly = ref(true)

onMounted(async () => {
  await Promise.all([fetchMarketplace(), fetchMarketStats()])
})

function handleImgError(id: string) {
  imageErrorMap.value[id] = true
}

async function refreshMarketplace() {
  await Promise.all([fetchMarketplace(), fetchMarketStats()])
  toast.success('应用市场数据已刷新')
}

// 获取热度统计数据
async function fetchMarketStats() {
  try {
    const res = await fetch('https://baihu-appstore-stats.qwapi.eu.org/stats')
    if (res.ok) {
      const data = await res.json()
      statsData.value = data
    }
  } catch (e) {
    // 忽略统计获取异常
  }
}

// 获取应用市场
async function fetchMarketplace() {
  loadingMarketplace.value = true
  try {
    const res = await api.apps.marketplace()
    if (Array.isArray(res.apps)) {
      marketplaceApps.value = res.apps
    } else if (res.apps && typeof res.apps === 'object' && Array.isArray((res.apps as any).apps)) {
      marketplaceApps.value = (res.apps as any).apps
    } else {
      marketplaceApps.value = []
    }
  } catch (err: any) {
    toast.error('拉取应用市场失败: ' + (err.message || '未知错误'))
  } finally {
    loadingMarketplace.value = false
  }
}

// 过滤后的应用市场列表
const filteredMarketplaceApps = computed(() => {
  const list = Array.isArray(marketplaceApps.value) ? marketplaceApps.value : []
  return list.filter(app => {
    // 搜索关键词（支持名称、ID、分类、描述、作者模糊搜索）
    if (!searchQuery.value.trim()) return true
    const q = searchQuery.value.toLowerCase()
    return (
      app.name.toLowerCase().includes(q) ||
      app.id.toLowerCase().includes(q) ||
      (app.category && app.category.toLowerCase().includes(q)) ||
      (app.description && app.description.toLowerCase().includes(q)) ||
      (app.author && app.author.toLowerCase().includes(q))
    )
  })
})

// 打开应用市场安装弹窗
function openMarketApply(app: MarketplaceApp) {
  selectedMarketApp.value = app
  showApplyDialog.value = true
}

// 打开自定义导入弹窗
function openCustomApply() {
  selectedMarketApp.value = null
  showApplyDialog.value = true
}

// 部署成功后，引导跳转至【调度实体】
function handleApplySuccess() {
  toast.success('应用部署成功！已将该实体及受控任务直接合并至【调度实体】页面展示与管控。', {
    action: {
      label: '前往调度实体',
      onClick: () => router.push('/tasks?type=app')
    }
  })
}
</script>

<template>
  <div class="space-y-4 w-full flex-1 flex flex-col">
    <!-- 顶部主标题与操作工具栏 -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <!-- 标题区域 -->
      <div class="flex flex-col shrink-0">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight flex items-center gap-2">
          <Store class="w-6 h-6 text-primary shrink-0" />
          <span>应用市场</span>
        </h2>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <span>浏览并一键部署官方声明式应用</span>
          <span class="hidden sm:inline text-border">|</span>
          <span class="text-[11px] text-emerald-500 font-medium">100% 开源免费</span>
          <span class="hidden sm:inline text-border">|</span>
          <a
            href="https://github.com/engigu/baihu-appstore"
            target="_blank"
            class="text-[11px] text-muted-foreground/90 hover:text-primary transition-colors inline-flex items-center gap-1 font-mono hover:underline"
            title="访问 engigu/baihu-appstore GitHub 开源仓库"
          >
            <Github class="w-3 h-3 text-foreground shrink-0" />
            <span>engigu/baihu-appstore</span>
            <ExternalLink class="w-2.5 h-2.5 opacity-60 shrink-0" />
          </a>
          <span v-if="statsData.pv?.marketplace || statsData.downloads?.global" class="hidden sm:inline text-border">|</span>
          <span v-if="statsData.pv?.marketplace" class="text-[11px] text-muted-foreground/90 flex items-center gap-1" title="应用市场累计浏览次数">
            <Eye class="w-3 h-3 text-primary/80" /> {{ statsData.pv.marketplace }} 浏览
          </span>
          <span v-if="statsData.downloads?.global" class="text-[11px] text-muted-foreground/90 flex items-center gap-1" title="所有应用累计下载安装总数">
            <Download class="w-3 h-3 text-emerald-500/80" /> {{ statsData.downloads.global }} 安装
          </span>
        </p>
      </div>

      <!-- 右侧控制栏：搜索框 + 刷新 + 官方仓库 + 导入 + 已装应用 -->
      <div class="flex items-center flex-wrap gap-2 w-full md:w-auto md:ml-auto md:justify-end">
        <!-- 搜索框 -->
        <div class="relative w-full sm:w-[200px] group text-sm">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input
            v-model="searchQuery"
            :readonly="isSearchReadonly"
            @focus="isSearchReadonly = false"
            @click="isSearchReadonly = false"
            type="search"
            name="app_store_search"
            placeholder="搜索应用、描述..."
            autocomplete="off"
            class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-xs sm:text-sm"
          />
        </div>

        <div class="flex items-center gap-2 w-full sm:w-auto">
          <!-- 刷新按钮 -->
          <Button
            variant="outline"
            size="icon"
            class="h-9 w-9 shrink-0"
            title="刷新应用市场"
            :disabled="loadingMarketplace"
            @click="refreshMarketplace"
          >
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loadingMarketplace }" />
          </Button>

          <!-- 官方 GitHub 仓库按钮 -->
          <a
            href="https://github.com/engigu/baihu-appstore"
            target="_blank"
            class="inline-flex flex-1 sm:flex-none"
            title="前往 baihu-appstore 官方 GitHub 开源仓库"
          >
            <Button variant="outline" size="sm" class="h-9 px-3 text-xs w-full justify-center shadow-sm font-medium gap-1">
              <Github class="h-3.5 w-3.5 shrink-0" />
              <span>应用仓库</span>
              <ExternalLink class="h-3 w-3 opacity-60 shrink-0 ml-0.5" />
            </Button>
          </a>

          <!-- 导入应用 (小屏下 flex-1 铺满) -->
          <Button size="sm" class="h-9 px-3 text-xs flex-1 sm:flex-none justify-center shadow-sm font-medium gap-1" @click="openCustomApply">
            <Plus class="h-3.5 w-3.5 shrink-0" />
            <span>导入应用</span>
          </Button>

          <!-- 前往已装应用 (小屏下 flex-1 铺满) -->
          <Button variant="outline" size="sm" class="h-9 px-3 text-xs flex-1 sm:flex-none justify-center shadow-sm font-medium gap-1" @click="router.push('/tasks?type=app')" title="查看已装应用">
            <Package class="h-3.5 w-3.5 text-emerald-500 shrink-0" />
            <span>已装应用</span>
          </Button>
        </div>
      </div>
    </div>

    <!-- ===================================================================== -->
    <!-- 应用市场主视图 (固定使用网格卡片视图) -->
    <!-- ===================================================================== -->
    <div class="space-y-4 flex-1 flex flex-col w-full">
      <!-- 市场加载中 -->
      <div v-if="loadingMarketplace" class="py-24 flex flex-col items-center justify-center text-muted-foreground gap-3">
        <Loader2 class="w-8 h-8 animate-spin text-primary" />
        <span class="text-xs">正在连接并检索应用市场...</span>
      </div>

      <!-- 市场空状态 -->
      <div
        v-else-if="filteredMarketplaceApps.length === 0"
        class="py-20 text-center border border-dashed border-border/80 rounded-2xl p-8 space-y-4 bg-muted/5 flex-1 flex flex-col items-center justify-center w-full"
      >
        <Store class="w-10 h-10 text-muted-foreground opacity-50 mx-auto" />
        <div class="space-y-1">
          <h3 class="text-sm font-bold text-foreground">没有找到匹配的市场应用</h3>
          <p class="text-xs text-muted-foreground">
            尝试更换搜索关键词，也可点击右上角进行自定义导入。
          </p>
        </div>
      </div>

      <!-- 网格卡片视图 -->
      <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
        <div
          v-for="app in filteredMarketplaceApps"
          :key="app.id"
          class="group rounded-xl border border-border/70 bg-card p-5 hover:border-primary/60 hover:shadow-md transition-all duration-200 flex flex-col justify-between relative overflow-hidden space-y-4"
        >
          <div class="space-y-3.5">
            <div class="flex items-start justify-between gap-3">
              <div class="flex items-center gap-3.5 min-w-0 flex-1">
                <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary/15 via-primary/5 to-muted/20 border border-primary/20 flex items-center justify-center shrink-0 overflow-hidden shadow-xs group-hover:scale-105 transition-transform">
                  <img
                    v-if="app.icon && !imageErrorMap[app.id]"
                    :src="app.icon"
                    class="w-full h-full object-cover"
                    @error="handleImgError(app.id)"
                    alt=""
                  />
                  <Package v-else class="w-6 h-6 text-primary" />
                </div>
                <div class="min-w-0 flex-1 overflow-hidden space-y-1">
                  <h3
                    class="text-sm sm:text-base font-bold text-foreground truncate group-hover:text-primary transition-colors cursor-pointer leading-snug"
                    :title="app.name"
                    @click="openMarketApply(app)"
                  >
                    {{ app.name }}
                  </h3>
                  <div class="text-[11px] sm:text-xs text-muted-foreground flex items-center gap-1.5 font-mono truncate">
                    <span class="text-primary font-semibold">v{{ app.version }}</span>
                    <span v-if="app.category" class="truncate opacity-75">• {{ app.category }}</span>
                  </div>
                </div>
              </div>

              <Badge variant="outline" class="text-[9px] sm:text-[10px] shrink-0 font-mono bg-muted/20 border-border/60">
                商店精选
              </Badge>
            </div>

            <p class="text-xs text-muted-foreground/90 line-clamp-2 leading-relaxed min-h-[2.25rem]">
              {{ app.description || '暂无应用详细描述说明...' }}
            </p>

            <div class="pt-2 border-t border-border/40 flex items-center justify-between text-xs text-muted-foreground gap-2">
              <div class="flex items-center gap-1.5 truncate min-w-0 flex-1">
                <User class="w-3.5 h-3.5 opacity-60 shrink-0" />
                <span class="truncate">{{ app.author || '社区贡献者' }}</span>
              </div>
              <div class="flex items-center gap-2.5 font-mono text-[10px] sm:text-[11px] opacity-75 shrink-0">
                <span v-if="statsData.downloads && statsData.downloads[app.id]" class="flex items-center gap-1 text-emerald-500 font-medium" title="累计部署次数">
                  <Download class="w-3 h-3" /> {{ statsData.downloads[app.id] }}
                </span>
                <span>{{ app.tasks_count || 1 }} 个任务</span>
              </div>
            </div>
          </div>

          <Button class="w-full h-9 font-medium shadow-xs text-xs gap-1.5 shrink-0" @click="openMarketApply(app)">
            <Sparkles class="w-3.5 h-3.5 shrink-0" />
            <span class="truncate">部署应用到调度实体</span>
            <ArrowRight class="w-3.5 h-3.5 ml-auto opacity-70 shrink-0 group-hover:translate-x-0.5 transition-transform" />
          </Button>
        </div>
      </div>
    </div>

    <!-- 应用部署弹窗 -->
    <ApplyDialog
      v-model:open="showApplyDialog"
      :target-app="selectedMarketApp"
      @success="handleApplySuccess"
    />
  </div>
</template>
