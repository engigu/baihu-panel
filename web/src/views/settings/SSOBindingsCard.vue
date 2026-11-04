<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Link2,
  Trash2,
  Fingerprint,
  Zap,
  UserCheck,
  RefreshCw,
  ShieldAlert,
  Loader2,
  Copy,
  Check
} from 'lucide-vue-next'
import { api, type UserSSOBinding } from '@/api'
import { toast } from 'vue-sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { withDemoGuard } from '@/utils/demo'

const props = withDefaults(
  defineProps<{
    defaultChannel?: 'oidc' | 'oauth2'
  }>(),
  {
    defaultChannel: 'oidc'
  }
)

const emit = defineEmits<{
  (e: 'update:count', count: number): void
}>()

const bindings = ref<UserSSOBinding[]>([])
const loading = ref(false)
const demoMode = ref(false)
const copiedSub = ref<string | null>(null)

function copySub(sub: string) {
  if (!navigator.clipboard) return
  navigator.clipboard.writeText(sub).then(() => {
    copiedSub.value = sub
    toast.success('已复制身份标识 (Sub)')
    setTimeout(() => {
      if (copiedSub.value === sub) copiedSub.value = null
    }, 2000)
  })
}

// 解绑确认弹窗
const showUnbindDialog = ref(false)
const targetBinding = ref<UserSSOBinding | null>(null)
const isUnbinding = ref(false)

async function loadData() {
  loading.value = true
  try {
    const [publicSite, list] = await Promise.all([
      api.settings.getPublicSite(),
      api.auth.getSSOBindings()
    ])
    demoMode.value = publicSite.demo_mode || false
    bindings.value = list || []
    emit('update:count', bindings.value.length)
  } catch (e: any) {
    toast.error(e.message || '加载绑定列表失败')
  } finally {
    loading.value = false
  }
}

function confirmUnbind(item: UserSSOBinding) {
  targetBinding.value = item
  showUnbindDialog.value = true
}

const handleUnbind = withDemoGuard(async () => {
  if (!targetBinding.value) return
  isUnbinding.value = true
  try {
    await api.auth.deleteSSOBinding(targetBinding.value.id)
    toast.success('解除绑定成功')
    showUnbindDialog.value = false
    targetBinding.value = null
    await loadData()
  } catch (e: any) {
    toast.error(e.message || '解绑失败')
  } finally {
    isUnbinding.value = false
  }
}, '演示模式下禁止解除绑定')

// 发起当前账号的一键授权绑定
const startAuthBind = withDemoGuard((type?: 'oidc' | 'oauth2') => {
  const channel = type || props.defaultChannel || 'oidc'
  window.location.href = api.auth.getSSOBindUrl(channel)
}, '演示模式下禁止发起授权绑定')

// 检查回调 URL 参数以呈现提示
function checkCallbackFeedback() {
  if (typeof window === 'undefined') return
  const url = new URL(window.location.href)
  const bindSuccess = url.searchParams.get('sso_bind_success')
  const bindError = url.searchParams.get('sso_bind_error')

  if (bindSuccess === '1') {
    toast.success('第三方账号授权绑定成功！已加入绑定列表')
    url.searchParams.delete('sso_bind_success')
    window.history.replaceState({}, '', url.toString())
  } else if (bindError) {
    toast.error(decodeURIComponent(bindError))
    url.searchParams.delete('sso_bind_error')
    window.history.replaceState({}, '', url.toString())
  }
}

defineExpose({
  bindingsCount: computed(() => bindings.value.length),
  loadData
})

onMounted(() => {
  loadData()
  checkCallbackFeedback()
})
</script>

<template>
  <div class="space-y-3">
    <!-- 列表标题与操作栏 -->
    <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3 sm:gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2 flex-wrap">
          <div class="text-sm font-semibold flex items-center gap-2 text-foreground">
            <Link2 class="w-4 h-4 text-sky-500 shrink-0" />
            <span>第三方身份与本地账号绑定列表</span>
          </div>
          <Badge variant="outline" class="text-[10px] px-1.5 py-0 font-normal">
            {{ bindings.length }} 条记录
          </Badge>
        </div>
        <p class="text-xs text-muted-foreground leading-relaxed pt-0.5">
          维护本地用户与单点登录（OIDC / OAuth 2.0）的映射。仅在此列表中的第三方身份可登录对应本地系统账号。
        </p>
      </div>

      <!-- 顶部操作栏（小屏铺满整行，中大屏自适应右侧） -->
      <div class="grid grid-cols-2 sm:flex sm:items-center gap-2 w-full sm:w-auto shrink-0">
        <Button
          variant="outline"
          size="sm"
          class="h-8 text-xs gap-1.5 w-full sm:w-auto justify-center"
          @click="loadData"
          :disabled="loading"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
          <span>刷新</span>
        </Button>

        <!-- 授权绑定当前登录账号 -->
        <Button
          variant="outline"
          size="sm"
          class="h-8 text-xs gap-1.5 w-full sm:w-auto justify-center hover:bg-sky-500/10 hover:border-sky-500/30 text-foreground"
          @click="() => startAuthBind(props.defaultChannel)"
          :disabled="demoMode"
          title="通过已启用的单点登录提供方授权并绑定至当前登录账号"
        >
          <Fingerprint class="w-3.5 h-3.5 text-sky-500 shrink-0" />
          <span class="truncate">授权绑定当前账号</span>
        </Button>
      </div>
    </div>

    <!-- 表格/卡片容器 -->
    <div class="rounded-lg border bg-card/60 overflow-hidden">
      <!-- ========== 1. 中大屏表格布局 (>= 640px) ========== -->
      <div class="hidden sm:block overflow-x-auto">
        <!-- 表头 -->
        <div class="flex items-center gap-3 md:gap-4 px-4 py-2 border-b bg-muted/30 text-xs text-muted-foreground font-medium min-w-[580px]">
          <span class="w-28 shrink-0">本地账号</span>
          <span class="w-24 shrink-0">认证通道</span>
          <span class="flex-1 min-w-[120px]">第三方身份标识 (Sub)</span>
          <span class="w-40 md:w-48 shrink-0">第三方昵称 / 邮箱</span>
          <span class="w-32 md:w-36 shrink-0 hidden md:block">绑定时间</span>
          <span class="w-16 shrink-0 text-right">操作</span>
        </div>

        <!-- 数据列表 -->
        <div class="divide-y text-xs min-w-[580px]">
          <div v-if="loading && bindings.length === 0" class="text-center py-10 text-muted-foreground">
            <Loader2 class="h-6 w-6 mx-auto mb-2 animate-spin opacity-50" />
            <span>加载绑定记录中...</span>
          </div>
          <div v-else-if="bindings.length === 0" class="text-center py-9 text-muted-foreground">
            <Link2 class="h-8 w-8 mx-auto mb-2 opacity-40 text-muted-foreground" />
            <div class="font-medium">暂无绑定的第三方身份</div>
            <p class="text-[11px] text-muted-foreground/70 mt-1">
              请点击上方【授权绑定当前账号】将当前系统账号与第三方身份进行关联
            </p>
          </div>
          <div
            v-for="item in bindings"
            :key="`desktop-${item.id}`"
            class="flex items-center gap-3 md:gap-4 px-4 py-2 hover:bg-muted/30 transition-colors"
          >
            <!-- 本地账号 -->
            <span class="w-28 shrink-0 flex items-center gap-1.5 font-medium truncate">
              <UserCheck class="w-3.5 h-3.5 text-primary shrink-0" />
              <span class="truncate">{{ item.username }}</span>
            </span>

            <!-- 认证通道 -->
            <span class="w-24 shrink-0">
              <Badge
                v-if="item.channel_type === 'oauth2'"
                variant="secondary"
                class="text-[10px] px-1.5 py-0 font-normal bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 flex items-center gap-1 w-fit"
              >
                <Zap class="w-2.5 h-2.5" />
                <span>OAuth 2.0</span>
              </Badge>
              <Badge
                v-else
                variant="secondary"
                class="text-[10px] px-1.5 py-0 font-normal bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20 flex items-center gap-1 w-fit"
              >
                <Fingerprint class="w-2.5 h-2.5" />
                <span>标准 OIDC</span>
              </Badge>
            </span>

            <!-- 第三方 Sub -->
            <span class="flex-1 min-w-[120px] font-mono text-[11px] truncate text-foreground/90 select-all" :title="item.subject">
              {{ item.subject }}
            </span>

            <!-- 昵称/邮箱 -->
            <span class="w-40 md:w-48 shrink-0 truncate text-muted-foreground text-[11px]" :title="item.identity_name || item.email || '-'">
              {{ item.identity_name || item.email || '-' }}
            </span>

            <!-- 绑定时间 -->
            <span class="w-32 md:w-36 shrink-0 hidden md:block text-[11px] text-muted-foreground truncate">
              {{ item.created_at || '-' }}
            </span>

            <!-- 操作项 -->
            <span class="w-16 shrink-0 flex justify-end">
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7 text-muted-foreground hover:text-destructive transition-colors"
                @click="confirmUnbind(item)"
                :disabled="demoMode"
                title="解除该身份绑定"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </Button>
            </span>
          </div>
        </div>
      </div>

      <!-- ========== 2. 小屏卡片流布局 (< 640px) ========== -->
      <div class="divide-y sm:hidden text-xs">
        <div v-if="loading && bindings.length === 0" class="text-center py-8 text-muted-foreground">
          <Loader2 class="h-5 w-5 mx-auto mb-2 animate-spin opacity-50" />
          <span>加载中...</span>
        </div>
        <div v-else-if="bindings.length === 0" class="text-center py-7 text-muted-foreground px-4">
          <Link2 class="h-7 w-7 mx-auto mb-2 opacity-40 text-muted-foreground" />
          <div class="font-medium">暂无绑定的第三方身份</div>
          <p class="text-[11px] text-muted-foreground/70 mt-1">
            请点击上方【授权绑定当前账号】进行关联
          </p>
        </div>
        <div
          v-for="item in bindings"
          :key="`mobile-${item.id}`"
          class="p-3 space-y-2 hover:bg-muted/20 transition-colors"
        >
          <!-- 顶部行：本地账号 + 认证通道 + 解绑按钮 -->
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-1.5 min-w-0">
              <UserCheck class="w-3.5 h-3.5 text-primary shrink-0" />
              <span class="font-semibold text-xs truncate max-w-[120px]">{{ item.username }}</span>
              <Badge
                v-if="item.channel_type === 'oauth2'"
                variant="secondary"
                class="text-[9px] px-1.5 py-0 font-normal bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 shrink-0 flex items-center gap-0.5"
              >
                <Zap class="w-2.5 h-2.5" />
                <span>OAuth 2.0</span>
              </Badge>
              <Badge
                v-else
                variant="secondary"
                class="text-[9px] px-1.5 py-0 font-normal bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20 shrink-0 flex items-center gap-0.5"
              >
                <Fingerprint class="w-2.5 h-2.5" />
                <span>标准 OIDC</span>
              </Badge>
            </div>

            <!-- 解绑按钮 -->
            <Button
              variant="ghost"
              size="sm"
              class="h-6 px-2 text-[11px] text-muted-foreground hover:text-destructive hover:bg-destructive/10 shrink-0 gap-1"
              @click="confirmUnbind(item)"
              :disabled="demoMode"
            >
              <Trash2 class="w-3 h-3 text-destructive" />
              <span class="text-destructive font-medium">解绑</span>
            </Button>
          </div>

          <!-- Sub 身份标识展示块（支持一键复制与自动折行） -->
          <div class="bg-muted/30 border border-border/50 rounded-md p-2 flex items-start justify-between gap-2">
            <div class="min-w-0 flex-1">
              <div class="text-[10px] text-muted-foreground mb-0.5">第三方身份标识 (Sub):</div>
              <div class="font-mono text-[11px] break-all leading-tight text-foreground select-all">
                {{ item.subject }}
              </div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              class="h-6 w-6 shrink-0 text-muted-foreground hover:text-foreground mt-1"
              @click="copySub(item.subject)"
              title="复制 Sub"
            >
              <Check v-if="copiedSub === item.subject" class="w-3 h-3 text-emerald-500" />
              <Copy v-else class="w-3 h-3" />
            </Button>
          </div>

          <!-- 底部行：第三方昵称/邮箱与绑定时间 -->
          <div class="flex items-center justify-between gap-2 text-[11px] text-muted-foreground pt-0.5">
            <span class="truncate" :title="item.identity_name || item.email || '-'">
              {{ item.identity_name || item.email || '-' }}
            </span>
            <span v-if="item.created_at" class="shrink-0 text-[10px] tabular-nums opacity-75">
              {{ item.created_at }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 解除绑定确认弹窗 -->
    <Dialog v-model:open="showUnbindDialog">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <ShieldAlert class="w-5 h-5 text-destructive" />
            <span>解除身份绑定确认</span>
          </DialogTitle>
          <DialogDescription class="text-xs pt-1 leading-relaxed">
            确定要解除本地账号 <strong class="text-foreground">「{{ targetBinding?.username }}」</strong> 与第三方身份 <strong class="text-foreground font-mono">「{{ targetBinding?.identity_name || targetBinding?.subject }}」</strong> 的绑定关系吗？解除后该第三方账号将无法直接通过单点登录进入此本地账号。
          </DialogDescription>
        </DialogHeader>

        <DialogFooter class="gap-2 sm:gap-0 pt-2">
          <Button variant="outline" size="sm" class="text-xs h-8" @click="showUnbindDialog = false" :disabled="isUnbinding">
            取消
          </Button>
          <Button variant="destructive" size="sm" class="text-xs h-8" @click="handleUnbind" :disabled="isUnbinding">
            <span v-if="isUnbinding">解绑中...</span>
            <span v-else>确认解除绑定</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
