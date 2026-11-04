<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { copyToClipboard } from '@/utils/clipboard'
import {
  Loader2,
  Fingerprint,
  Globe,
  KeyRound,
  ShieldCheck,
  Copy,
  Check,
  Eye,
  EyeOff,
  ExternalLink,
  AlertTriangle,
  Sparkles,
  HelpCircle,
  Lock,
  Zap,
  ChevronDown,
  ChevronUp,
  Link2
} from 'lucide-vue-next'
import { useRegisterSettingSection, useFieldDiff } from './composables/useSettingsTracker'
import ModifiedBadge from './components/ModifiedBadge.vue'
import SSOBindingsCard from './SSOBindingsCard.vue'

const loading = ref(false)
const saving = ref(false)
const testingOidc = ref(false)
const testingOAuth2 = ref(false)
const demoMode = ref(false)
const showSecretOidc = ref(false)
const showSecretOAuth2 = ref(false)
const copiedOidc = ref(false)
const copiedOAuth2 = ref(false)
const showGuide = ref(false)

// 内部主模式选项卡 ('bindings' 绑定列表 | 'config' 通道配置)
const mainTab = ref<'bindings' | 'config'>('bindings')

// 绑定记录数与引用
const bindingsCount = ref(0)
const bindingsCardRef = ref<any>(null)

function onBindingsCountUpdate(count: number) {
  bindingsCount.value = count
}

// 展开折叠状态（优先检查回调参数展开，其次读取本地偏好）
const CONFIG_EXPANDED_KEY = 'baihu_sso_expanded'
const hasBindCallback = typeof window !== 'undefined' && (
  window.location.search.includes('sso_bind_success') ||
  window.location.search.includes('sso_bind_error')
)
const configExpanded = ref(
  hasBindCallback ||
  localStorage.getItem(CONFIG_EXPANDED_KEY) === 'true'
)

function toggleExpanded() {
  configExpanded.value = !configExpanded.value
  try {
    localStorage.setItem(CONFIG_EXPANDED_KEY, configExpanded.value ? 'true' : 'false')
  } catch {}
}

// 当前激活的通道配置选项卡 ('oidc' | 'oauth2')
const activeTab = ref<'oidc' | 'oauth2'>('oidc')

// 数据库实际保存的已启用状态
const savedOidcEnabled = ref(false)
const savedOAuth2Enabled = ref(false)

// 初始快照表单（用于变动侦测与放弃修改还原）
const initialForm = ref<Record<string, any> | null>(null)

const form = ref({
  // 1. 标准 OIDC 通道独立配置
  oidc_enabled: false,
  oidc_issuer: '',
  oidc_client_id: '',
  oidc_client_secret: '',
  oidc_scopes: '',

  // 2. 通用 OAuth 2.0 通道独立配置
  oauth2_enabled: false,
  oauth2_name: '',
  oauth2_auth_url: '',
  oauth2_token_url: '',
  oauth2_userinfo_url: '',
  oauth2_user_key: '',
  oauth2_client_id: '',
  oauth2_client_secret: '',
  oauth2_scopes: '',
})

// 推导标准回调地址
const oidcRedirectUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  const base = window.location.origin
  const prefix = (window as any).__BASE_URL__ || ''
  return `${base}${prefix}/api/v1/auth/oidc/callback`
})

const oauth2RedirectUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  const base = window.location.origin
  const prefix = (window as any).__BASE_URL__ || ''
  return `${base}${prefix}/api/v1/auth/oauth2/callback`
})

function cleanOidcIssuer() {
  if (form.value.oidc_issuer) {
    form.value.oidc_issuer = form.value.oidc_issuer
      .trim()
      .replace(/\/+\.well-known\/openid-configuration\/?$/i, '')
      .replace(/\/+$/, '')
  }
}

// 探测文档地址（OIDC）
const discoveryUrl = computed(() => {
  let iss = form.value.oidc_issuer.trim()
  if (!iss || (!iss.startsWith('http://') && !iss.startsWith('https://'))) return ''
  iss = iss.replace(/\/+\.well-known\/openid-configuration\/?$/i, '').replace(/\/+$/, '')
  return `${iss}/.well-known/openid-configuration`
})

// 常用 OAuth 2.0 平台预设模版
interface OAuth2Preset {
  name: string
  displayName: string
  authUrl: string
  tokenUrl: string
  userinfoUrl: string
  scopes: string
  userKey?: string
}

const oauth2Presets: OAuth2Preset[] = [
  {
    name: 'GitHub',
    displayName: 'GitHub',
    authUrl: 'https://github.com/login/oauth/authorize',
    tokenUrl: 'https://github.com/login/oauth/access_token',
    userinfoUrl: 'https://api.github.com/user',
    scopes: 'read:user user:email',
    userKey: 'id',
  },
  {
    name: 'Gitee (码云)',
    displayName: 'Gitee',
    authUrl: 'https://gitee.com/oauth/authorize',
    tokenUrl: 'https://gitee.com/oauth/token',
    userinfoUrl: 'https://gitee.com/api/v5/user',
    scopes: 'user_info emails',
    userKey: 'id',
  },
  {
    name: 'GitLab',
    displayName: 'GitLab',
    authUrl: 'https://gitlab.com/oauth/authorize',
    tokenUrl: 'https://gitlab.com/oauth/token',
    userinfoUrl: 'https://gitlab.com/api/v4/user',
    scopes: 'read_user',
    userKey: 'id',
  },
]

function applyOAuth2Preset(p: OAuth2Preset) {
  form.value.oauth2_name = p.displayName
  form.value.oauth2_auth_url = p.authUrl
  form.value.oauth2_token_url = p.tokenUrl
  form.value.oauth2_userinfo_url = p.userinfoUrl
  form.value.oauth2_scopes = p.scopes
  if (p.userKey) form.value.oauth2_user_key = p.userKey
  toast.success(`已自动填充 ${p.name} 平台端点参数`)
}

// 快捷 Scope 列表
const oidcCommonScopes = ['groups', 'roles', 'offline_access']
const oauth2CommonScopes = ['read:user', 'user:email', 'profile', 'email']

function addOidcScope(scope: string) {
  const current = form.value.oidc_scopes.trim()
  const list = current ? current.split(/[\s,]+/).filter(Boolean) : []
  if (!list.includes(scope)) {
    list.push(scope)
    form.value.oidc_scopes = list.join(' ')
  }
}

function addOAuth2Scope(scope: string) {
  const current = form.value.oauth2_scopes.trim()
  const list = current ? current.split(/[\s,]+/).filter(Boolean) : []
  if (!list.includes(scope)) {
    list.push(scope)
    form.value.oauth2_scopes = list.join(' ')
  }
}

async function copyUrl(url: string, type: 'oidc' | 'oauth2') {
  if (!url) return
  const success = await copyToClipboard(url)
  if (success) {
    if (type === 'oidc') {
      copiedOidc.value = true
      setTimeout(() => (copiedOidc.value = false), 2000)
    } else {
      copiedOAuth2.value = true
      setTimeout(() => (copiedOAuth2.value = false), 2000)
    }
    toast.success('回调地址已复制到剪贴板')
  } else {
    toast.error('复制失败，请手动选择复制')
  }
}

async function loadSection() {
  loading.value = true
  try {
    const res = await api.settings.getSection('security')

    // 判断各自通道的激活状态（兼容历史单一开关）
    const isOidc = res.oidc_enabled === 'true' && res.oidc_type !== 'oauth2'
    let isOAuth2 = res.oauth2_enabled === 'true'
    if (!isOAuth2 && res.oidc_enabled === 'true' && res.oidc_type === 'oauth2') {
      isOAuth2 = true
    }

    savedOidcEnabled.value = isOidc
    savedOAuth2Enabled.value = isOAuth2

    form.value = {
      // OIDC
      oidc_enabled: isOidc,
      oidc_issuer: res.oidc_issuer || '',
      oidc_client_id: res.oidc_client_id || '',
      oidc_client_secret: res.oidc_client_secret || '',
      oidc_scopes: res.oidc_scopes || '',

      // OAuth 2.0
      oauth2_enabled: isOAuth2,
      oauth2_name: res.oauth2_name || '',
      oauth2_auth_url: res.oauth2_auth_url || '',
      oauth2_token_url: res.oauth2_token_url || '',
      oauth2_userinfo_url: res.oauth2_userinfo_url || '',
      oauth2_user_key: res.oauth2_user_key || '',
      oauth2_client_id: res.oauth2_client_id || '',
      oauth2_client_secret: res.oauth2_client_secret || '',
      oauth2_scopes: res.oauth2_scopes || '',
    }

    // 记录初始快照以供变动追踪
    initialForm.value = JSON.parse(JSON.stringify(form.value))

    // 若系统从未配置过任何 SSO 通道且无绑定记录，默认切至配置页引导接入
    if (!hasBindCallback && !isOidc && !isOAuth2 && bindingsCount.value === 0) {
      mainTab.value = 'config'
    }
  } catch (e: any) {
    toast.error(e.message || '读取单点登录配置失败')
  } finally {
    loading.value = false
  }
}

// 变动项侦测与单字段追踪
const { isDiff, getOriginal, resetField, changedFields, diffClass } = useFieldDiff(form, initialForm)
const isDirty = computed(() => changedFields.value.length > 0)
const dirtyCount = computed(() => changedFields.value.length)

// 检测特定 Tab 内部是否有改动
const hasOidcDiff = computed(() => {
  return [
    'oidc_enabled',
    'oidc_issuer',
    'oidc_client_id',
    'oidc_client_secret',
    'oidc_scopes',
  ].some(k => isDiff(k))
})

const hasOAuth2Diff = computed(() => {
  return [
    'oauth2_enabled',
    'oauth2_name',
    'oauth2_auth_url',
    'oauth2_token_url',
    'oauth2_userinfo_url',
    'oauth2_user_key',
    'oauth2_client_id',
    'oauth2_client_secret',
    'oauth2_scopes',
  ].some(k => isDiff(k))
})

function resetForm() {
  if (!initialForm.value) return
  form.value = JSON.parse(JSON.stringify(initialForm.value))
  toast.info('已还原单点登录配置')
}

// 注册至全局设置变动追踪器
useRegisterSettingSection({
  id: 'oidc',
  tab: 'security',
  title: '单点登录 SSO',
  isDirty,
  dirtyCount,
  saving,
  save,
  reset: resetForm
})

async function save() {
  if (demoMode.value) {
    toast.error('演示模式下禁止修改单点登录配置')
    return
  }

  cleanOidcIssuer()

  // 校验 OIDC 参数
  if (form.value.oidc_enabled) {
    if (!form.value.oidc_issuer.trim()) {
      toast.error('已开启标准 OIDC，必须填写 Issuer（提供方根地址）')
      mainTab.value = 'config'
      activeTab.value = 'oidc'
      return
    }
    if (!form.value.oidc_client_id.trim()) {
      toast.error('已开启标准 OIDC，必须填写 Client ID（客户端标识）')
      mainTab.value = 'config'
      activeTab.value = 'oidc'
      return
    }
  }

  // 校验 OAuth 2.0 参数
  if (form.value.oauth2_enabled) {
    if (!form.value.oauth2_auth_url.trim() || !form.value.oauth2_token_url.trim() || !form.value.oauth2_userinfo_url.trim()) {
      toast.error('已开启通用 OAuth 2.0，必须完整填写授权、令牌与用户信息端点')
      mainTab.value = 'config'
      activeTab.value = 'oauth2'
      return
    }
    if (!form.value.oauth2_client_id.trim()) {
      toast.error('已开启通用 OAuth 2.0，必须填写 Client ID（客户端标识）')
      mainTab.value = 'config'
      activeTab.value = 'oauth2'
      return
    }
  }

  saving.value = true
  try {
    await api.settings.setSection('security', {
      // OIDC 独立参数
      oidc_enabled: form.value.oidc_enabled ? 'true' : 'false',
      oidc_issuer: form.value.oidc_issuer.trim(),
      oidc_client_id: form.value.oidc_client_id.trim(),
      oidc_client_secret: form.value.oidc_client_secret,
      oidc_scopes: form.value.oidc_scopes.trim(),

      // OAuth 2.0 独立参数
      oauth2_enabled: form.value.oauth2_enabled ? 'true' : 'false',
      oauth2_name: form.value.oauth2_name.trim(),
      oauth2_auth_url: form.value.oauth2_auth_url.trim(),
      oauth2_token_url: form.value.oauth2_token_url.trim(),
      oauth2_userinfo_url: form.value.oauth2_userinfo_url.trim(),
      oauth2_user_key: form.value.oauth2_user_key.trim(),
      oauth2_client_id: form.value.oauth2_client_id.trim(),
      oauth2_client_secret: form.value.oauth2_client_secret,
      oauth2_scopes: form.value.oauth2_scopes.trim(),
    })
    toast.success('单点登录配置已保存，多身份源通道已生效')
    await loadSection()
    bindingsCardRef.value?.loadData?.()
  } catch (e: any) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleTestOidc() {
  if (demoMode.value) {
    toast.error('演示模式下禁止测试')
    return
  }
  cleanOidcIssuer()
  if (!form.value.oidc_issuer.trim()) {
    toast.error('请先填写 Issuer（提供方根地址）')
    return
  }

  testingOidc.value = true
  try {
    const res = await api.auth.oidcTest({
      type: 'oidc',
      issuer: form.value.oidc_issuer.trim(),
    })
    toast.success(res.message || 'OIDC 发现文档解析通过，连通正常')
  } catch (e: any) {
    toast.error(e.message || '连接测试失败，请检查 Issuer 地址或网络')
  } finally {
    testingOidc.value = false
  }
}

async function handleTestOAuth2() {
  if (demoMode.value) {
    toast.error('演示模式下禁止测试')
    return
  }
  if (!form.value.oauth2_auth_url.trim() || !form.value.oauth2_token_url.trim() || !form.value.oauth2_userinfo_url.trim()) {
    toast.error('请完整填写授权、令牌与用户信息端点')
    return
  }

  testingOAuth2.value = true
  try {
    const res = await api.auth.oauth2Test({
      type: 'oauth2',
      oauth2_auth_url: form.value.oauth2_auth_url.trim(),
      oauth2_token_url: form.value.oauth2_token_url.trim(),
      oauth2_userinfo_url: form.value.oauth2_userinfo_url.trim(),
    })
    toast.success(res.message || 'OAuth 2.0 端点网络连通性测试通过')
  } catch (e: any) {
    toast.error(e.message || '连接测试失败，请检查端点配置或网络')
  } finally {
    testingOAuth2.value = false
  }
}

async function loadDemoMode() {
  try {
    const res = await api.settings.getPublicSite()
    demoMode.value = !!res.demo_mode
  } catch {
    demoMode.value = false
  }
}

onMounted(() => {
  loadSection()
  loadDemoMode()
})
</script>

<template>
  <Card class="shadow-sm flex flex-col justify-between pt-4" :class="configExpanded ? 'pb-5' : 'pb-4'">
    <!-- 卡片头部：图标、标题、副标题与全局状态 -->
    <CardHeader class="pb-3 pt-0 px-4 sm:px-5">
      <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-2.5 sm:gap-4">
        <div class="space-y-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <CardTitle class="text-base flex items-center gap-2">
              <Fingerprint class="w-4 h-4 text-sky-500 shrink-0" />
              <span class="whitespace-nowrap font-bold">统一单点登录与第三方认证 (SSO)</span>
            </CardTitle>
            <Badge
              v-if="demoMode"
              variant="destructive"
              class="text-[10px] px-1.5 py-0 font-normal shrink-0"
            >
              演示模式禁用
            </Badge>
            <Badge
              v-else-if="savedOidcEnabled && savedOAuth2Enabled"
              variant="secondary"
              class="text-[10px] px-1.5 py-0 font-normal bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 shrink-0"
            >
              双通道已启用 (OIDC + {{ form.oauth2_name || 'OAuth 2.0' }})
            </Badge>
            <Badge
              v-else-if="savedOidcEnabled"
              variant="secondary"
              class="text-[10px] px-1.5 py-0 font-normal bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 shrink-0"
            >
              OIDC 已激活
            </Badge>
            <Badge
              v-else-if="savedOAuth2Enabled"
              variant="secondary"
              class="text-[10px] px-1.5 py-0 font-normal bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 shrink-0"
            >
              {{ form.oauth2_name || 'OAuth 2.0' }} 已激活
            </Badge>
            <Badge
              v-else-if="form.oidc_enabled || form.oauth2_enabled"
              variant="secondary"
              class="text-[10px] px-1.5 py-0 font-normal bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 shrink-0"
            >
              通道配置待保存
            </Badge>
            <Badge
              v-else
              variant="outline"
              class="text-[10px] px-1.5 py-0 font-normal border-border/80 text-muted-foreground shrink-0"
            >
              未启用
            </Badge>

            <!-- 绑定记录数指示 -->
            <Badge
              v-if="bindingsCount > 0"
              variant="outline"
              class="text-[10px] px-1.5 py-0 font-normal shrink-0 border-border/80 text-muted-foreground"
            >
              {{ bindingsCount }} 条绑定
            </Badge>
          </div>
          <CardDescription class="text-xs leading-relaxed pt-0.5">
            支持标准 OIDC 身份中心与通用 OAuth 2.0 多身份源并行启用，登录页将自动呈现各自快捷登录入口。
          </CardDescription>
        </div>

        <!-- 对接指引与折叠展开按钮 -->
        <div class="grid grid-cols-2 gap-2 w-full sm:w-auto sm:flex sm:items-center sm:gap-2.5 shrink-0 pt-1.5 sm:pt-0">
          <Button
            type="button"
            variant="outline"
            size="sm"
            @click="showGuide = !showGuide"
            class="text-xs h-8 sm:h-7.5 px-3 sm:px-2.5 gap-1.5 shadow-xs bg-background hover:bg-muted/70 transition-all justify-center w-full sm:w-auto text-sky-600 dark:text-sky-400 border-sky-500/25 hover:border-sky-500/40"
          >
            <HelpCircle class="w-3.5 h-3.5 shrink-0" />
            <span class="whitespace-nowrap">{{ showGuide ? '收起指引' : '对接指引' }}</span>
          </Button>
          <Button
            variant="outline"
            size="sm"
            class="text-xs h-8 sm:h-7.5 px-3 sm:px-2.5 gap-1.5 shadow-xs bg-background hover:bg-muted/70 transition-all justify-center w-full sm:w-auto shrink-0"
            @click="toggleExpanded"
          >
            <span class="whitespace-nowrap">{{ configExpanded ? '收起' : '展开' }}</span>
            <ChevronUp v-if="configExpanded" class="h-3.5 w-3.5" />
            <ChevronDown v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </CardHeader>

    <!-- 卡片主体内容（受展开/收起统一控制） -->
    <CardContent v-if="configExpanded" class="flex-1 flex flex-col justify-between px-4 sm:px-5 pb-0 pt-0 space-y-3.5 sm:space-y-4">
      <!-- ==================== 顶部模式 Tab 切换栏 (全屏/小屏均铺满整行) ==================== -->
      <div class="p-1 sm:p-1.5 rounded-xl border border-border/70 bg-muted/30 grid grid-cols-2 gap-1.5 text-xs shadow-xs w-full">
        <!-- Tab 1: 已绑定账号列表 -->
        <button
          type="button"
          @click="mainTab = 'bindings'"
          class="py-2 px-2 sm:px-3 rounded-lg transition-all flex items-center justify-center gap-1.5 sm:gap-2 text-xs select-none whitespace-nowrap min-w-0"
          :class="mainTab === 'bindings' ? 'bg-background text-foreground font-semibold shadow-xs border border-border/60' : 'text-muted-foreground hover:text-foreground hover:bg-background/40'"
        >
          <Link2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-sky-500 shrink-0" />
          <span class="truncate">
            <span class="sm:hidden">绑定列表</span>
            <span class="hidden sm:inline">已绑定账号列表</span>
          </span>
          <Badge
            variant="secondary"
            class="text-[10px] px-1.5 py-0 h-4 min-w-4 justify-center shrink-0"
            :class="mainTab === 'bindings' ? 'bg-muted text-foreground' : 'bg-muted/60 text-muted-foreground'"
          >
            {{ bindingsCount }}
          </Badge>
        </button>

        <!-- Tab 2: 通道接入与参数配置 -->
        <button
          type="button"
          @click="mainTab = 'config'"
          class="py-2 px-2 sm:px-3 rounded-lg transition-all flex items-center justify-center gap-1.5 sm:gap-2 text-xs select-none whitespace-nowrap min-w-0"
          :class="mainTab === 'config' ? 'bg-background text-foreground font-semibold shadow-xs border border-border/60' : 'text-muted-foreground hover:text-foreground hover:bg-background/40'"
        >
          <KeyRound class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-sky-500 shrink-0" />
          <span class="truncate">
            <span class="sm:hidden">通道配置</span>
            <span class="hidden sm:inline">通道接入与参数配置</span>
          </span>
          <span
            v-if="isDirty"
            class="inline-flex items-center gap-1 text-[10px] text-amber-600 dark:text-amber-400 font-medium bg-amber-500/10 px-1 py-0.2 rounded shrink-0"
            title="配置存在未保存变动"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
            <span class="hidden sm:inline">待保存</span>
          </span>
          <span
            v-else-if="savedOidcEnabled || savedOAuth2Enabled"
            class="inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-normal shrink-0"
            title="通道服务在线"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
            <span class="hidden sm:inline">在线</span>
          </span>
        </button>
      </div>

      <!-- ==================== 1. 已绑定账号列表视图 (Tab 1) ==================== -->
      <div v-show="mainTab === 'bindings'" class="space-y-3 sm:space-y-4 animate-in fade-in duration-150">
        <SSOBindingsCard
          ref="bindingsCardRef"
          :default-channel="savedOAuth2Enabled && !savedOidcEnabled ? 'oauth2' : 'oidc'"
          @update:count="onBindingsCountUpdate"
        />
      </div>

      <!-- ==================== 2. 通道参数配置视图 (Tab 2) ==================== -->
      <div v-show="mainTab === 'config'" class="space-y-3.5 animate-in fade-in duration-150">
        <!-- 演示模式警示条 -->
        <div
          v-if="demoMode"
          class="flex items-center gap-2 p-3 rounded-xl bg-destructive/10 text-destructive text-xs"
        >
          <AlertTriangle class="h-4 w-4 shrink-0" />
          <span>演示模式下已锁定单点登录凭据与安全参数配置。</span>
        </div>

        <!-- 对接指引折叠面板 -->
        <div
          v-if="showGuide"
          class="p-3 rounded-xl border bg-gradient-to-br from-sky-500/5 via-background to-purple-500/5 border-sky-500/25 space-y-2 text-xs"
        >
          <div class="flex items-center gap-1.5 font-semibold text-foreground">
            <Sparkles class="w-3.5 h-3.5 text-sky-500" />
            <span>多身份源并行接入速查</span>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] text-muted-foreground">
            <div class="p-2 rounded-lg bg-muted/40 border border-border/40">
              <b class="text-foreground">标准 OIDC:</b> 适用于 Pocket ID、Keycloak、Authentik 等企业身份中心，只需填入 Issuer，发现文档自动嗅探。
            </div>
            <div class="p-2 rounded-lg bg-muted/40 border border-border/40">
              <b class="text-foreground">通用 OAuth 2.0:</b> 适用于 GitHub、Gitee、GitLab 或自建 OAuth App，支持快速套用预设。
            </div>
            <div class="p-2 rounded-lg bg-muted/40 border border-border/40 col-span-1 sm:col-span-2">
              <b class="text-foreground">多源并行:</b> 两套通道可同时独立开启！登录页面将同时展现「Pocket ID」与「GitHub」快捷按钮供用户选择。
            </div>
          </div>
        </div>

        <!-- 顶部通道选择选项卡（清晰展示各自通道在线状态与独立切换入口） -->
        <div class="p-1 sm:p-1.5 rounded-xl border border-border/70 bg-muted/30 grid grid-cols-2 gap-1.5 text-xs shadow-xs">
          <!-- OIDC 选项卡 -->
          <button
            type="button"
            @click="activeTab = 'oidc'"
            class="py-2 px-2 sm:px-3 rounded-lg transition-all flex items-center justify-between gap-1.5 sm:gap-2 text-xs select-none"
            :class="activeTab === 'oidc' ? 'bg-background text-foreground font-semibold shadow-xs border border-border/60' : 'text-muted-foreground hover:text-foreground hover:bg-background/40'"
          >
            <div class="flex items-center gap-1.5 sm:gap-2 truncate">
              <Globe class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-sky-500 shrink-0" />
              <span class="truncate font-medium">OIDC</span>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              <span
                v-if="hasOidcDiff"
                class="inline-flex items-center gap-1 text-[10px] text-amber-600 dark:text-amber-400 font-medium bg-amber-500/10 px-1.5 py-0.2 rounded border border-amber-500/20"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                <span>待保存</span>
              </span>
              <span
                v-else-if="savedOidcEnabled"
                class="inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-normal"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span>在线</span>
              </span>
              <span v-else class="text-[10px] text-muted-foreground/70">未开启</span>
            </div>
          </button>

          <!-- OAuth 2.0 选项卡 -->
          <button
            type="button"
            @click="activeTab = 'oauth2'"
            class="py-2 px-2 sm:px-3 rounded-lg transition-all flex items-center justify-between gap-1.5 sm:gap-2 text-xs select-none"
            :class="activeTab === 'oauth2' ? 'bg-background text-foreground font-semibold shadow-xs border border-border/60' : 'text-muted-foreground hover:text-foreground hover:bg-background/40'"
          >
            <div class="flex items-center gap-1.5 sm:gap-2 truncate">
              <Zap class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-amber-500 shrink-0" />
              <span class="truncate font-medium">OAuth 2.0</span>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              <span
                v-if="hasOAuth2Diff"
                class="inline-flex items-center gap-1 text-[10px] text-amber-600 dark:text-amber-400 font-medium bg-amber-500/10 px-1.5 py-0.2 rounded border border-amber-500/20"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                <span>待保存</span>
              </span>
              <span
                v-else-if="savedOAuth2Enabled"
                class="inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-normal"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span>在线</span>
              </span>
              <span v-else class="text-[10px] text-muted-foreground/70">未开启</span>
            </div>
          </button>
        </div>

        <!-- ==================== 1. 标准 OIDC 通道专属面板 ==================== -->
        <div v-if="activeTab === 'oidc'" class="space-y-3.5 animate-in fade-in duration-200">
          <!-- 通道主开关与在线指示条 -->
          <div
            class="p-3 sm:p-3.5 rounded-xl border flex flex-col sm:flex-row sm:items-center justify-between transition-all gap-2.5 sm:gap-3"
            :class="savedOidcEnabled ? 'border-emerald-500/20 bg-emerald-500/5' : (form.oidc_enabled ? 'border-amber-500/30 bg-amber-500/5' : 'border-border/70 bg-muted/20')"
          >
            <div class="flex items-center gap-2.5 sm:gap-3 min-w-0">
              <div
                class="w-8.5 h-8.5 rounded-lg flex items-center justify-center shrink-0"
                :class="savedOidcEnabled ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20' : 'bg-muted text-muted-foreground border border-border/60'"
              >
                <Globe class="w-4 h-4" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-1.5 flex-nowrap">
                  <span class="text-xs font-bold text-foreground shrink-0">OIDC 通道</span>
                  <Badge
                    variant="outline"
                    class="text-[9px] px-1 py-0 h-4 shrink-0"
                    :class="savedOidcEnabled ? 'border-emerald-500/30 text-emerald-600 dark:text-emerald-400' : 'border-muted-foreground/30 text-muted-foreground'"
                  >
                    {{ savedOidcEnabled ? '服务在线' : (form.oidc_enabled ? '待保存' : '已停用') }}
                  </Badge>
                </div>
                <div class="text-[10px] text-muted-foreground pt-0.5 truncate">
                  支持 Pocket ID / Keycloak / Authentik 等标准身份源
                </div>
              </div>
            </div>

            <!-- 通道操作区：测试按钮与独立启用开关 -->
            <div class="flex items-center gap-2 sm:gap-2.5 shrink-0 justify-between sm:justify-end w-full sm:w-auto pt-2 sm:pt-0 border-t sm:border-t-0 border-border/50">
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="handleTestOidc"
                :disabled="testingOidc || saving || loading || demoMode"
                class="h-8 px-2.5 sm:px-3 text-xs font-medium shadow-xs gap-1.5 bg-background/80 hover:bg-background shrink-0"
              >
                <Loader2 v-if="testingOidc" class="h-3.5 w-3.5 animate-spin" />
                <Globe v-else class="h-3.5 w-3.5 text-sky-500" />
                <span>{{ testingOidc ? '测试中...' : '测试 OIDC 发现文档' }}</span>
              </Button>

              <div class="flex items-center gap-1.5 sm:gap-2 pl-2 sm:pl-2.5 border-l border-border/60 shrink-0">
                <ModifiedBadge
                  v-if="isDiff('oidc_enabled')"
                  :original-value="getOriginal('oidc_enabled')"
                  @reset="resetField('oidc_enabled')"
                />
                <Switch
                  :model-value="form.oidc_enabled"
                  @update:model-value="(val) => (form.oidc_enabled = val)"
                  id="oidc-channel-enabled"
                  :disabled="demoMode || loading"
                />
                <Label for="oidc-channel-enabled" class="text-xs cursor-pointer select-none">
                  {{ form.oidc_enabled ? '已开启' : '开启' }}
                </Label>
              </div>
            </div>
          </div>

          <!-- OIDC 专用回调地址说明条 -->
          <div class="p-3 rounded-xl border border-sky-500/20 bg-sky-500/5 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
            <div class="space-y-0.5 min-w-0">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                <ShieldCheck class="w-3.5 h-3.5 text-sky-500 shrink-0" />
                <span>OIDC 专用授权重定向地址 (Redirect URI)</span>
              </div>
              <p class="text-[11px] text-muted-foreground truncate">
                请在 Pocket ID / Keycloak 应用客户端后台严格登记该回调地址。
              </p>
            </div>
            <div class="flex items-center gap-1.5 shrink-0 w-full sm:w-auto">
              <code class="text-[11px] font-mono px-2 py-1 rounded bg-background border border-border/80 text-foreground truncate flex-1 min-w-0 sm:max-w-[320px]">
                {{ oidcRedirectUrl }}
              </code>
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-7.5 px-2.5 text-xs gap-1 font-medium bg-background shadow-xs shrink-0"
                @click="copyUrl(oidcRedirectUrl, 'oidc')"
              >
                <Check v-if="copiedOidc" class="w-3.5 h-3.5 text-emerald-500" />
                <Copy v-else class="w-3.5 h-3.5" />
                <span>{{ copiedOidc ? '已复制' : '复制' }}</span>
              </Button>
            </div>
          </div>

          <!-- Issuer 地址 -->
          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2 flex-wrap">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Globe class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                <span>Issuer（提供方根地址）</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oidc_issuer')"
                  :original-value="getOriginal('oidc_issuer')"
                  @reset="resetField('oidc_issuer')"
                />
              </Label>
              <a
                v-if="discoveryUrl"
                :href="discoveryUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="text-[11px] text-sky-600 dark:text-sky-400 hover:underline flex items-center gap-0.5 shrink-0"
              >
                <span>检验发现文档</span>
                <ExternalLink class="w-2.5 h-2.5" />
              </a>
            </div>
            <Input
              v-model="form.oidc_issuer"
              placeholder="https://id.example.com"
              class="h-9 text-xs font-mono"
              :class="diffClass('oidc_issuer')"
              :disabled="demoMode || loading"
              @blur="cleanOidcIssuer"
            />
            <p class="text-[11px] text-muted-foreground pt-0.5 flex items-center gap-1.5">
              <HelpCircle class="w-3.5 h-3.5 text-muted-foreground/80 shrink-0" />
              <span>填写身份认证根域名（例如 <code>https://id.example.com</code>），无需手动拼接 <code>/.well-known/...</code> 后缀。</span>
            </p>
          </div>

          <!-- OIDC 客户端凭据 -->
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-3.5">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5 flex-wrap">
                <KeyRound class="w-3.5 h-3.5 text-muted-foreground" />
                <span>Client ID（OIDC 客户端标识）</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oidc_client_id')"
                  :original-value="getOriginal('oidc_client_id')"
                  @reset="resetField('oidc_client_id')"
                />
              </Label>
              <Input
                v-model="form.oidc_client_id"
                placeholder="例如 Pocket ID 或 Keycloak Client ID"
                class="h-9 text-xs font-mono"
                :class="diffClass('oidc_client_id')"
                :disabled="demoMode || loading"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5 flex-wrap">
                <Lock class="w-3.5 h-3.5 text-muted-foreground" />
                <span>Client Secret（OIDC 客户端密钥）</span>
                <ModifiedBadge
                  v-if="isDiff('oidc_client_secret')"
                  :show-reset="true"
                  @reset="resetField('oidc_client_secret')"
                />
              </Label>
              <div class="relative">
                <Input
                  v-model="form.oidc_client_secret"
                  :type="showSecretOidc ? 'text' : 'password'"
                  class="h-9 pr-8 text-xs font-mono"
                  :class="diffClass('oidc_client_secret')"
                  :disabled="demoMode || loading"
                  placeholder="留空保持不变"
                />
                <button
                  type="button"
                  @click="showSecretOidc = !showSecretOidc"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors p-0.5"
                  :title="showSecretOidc ? '隐藏密钥' : '显示明文'"
                >
                  <EyeOff v-if="showSecretOidc" class="w-3.5 h-3.5" />
                  <Eye v-else class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>

          <!-- OIDC Scope -->
          <div class="space-y-1.5">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-1.5 sm:gap-2">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>授权范围 (Scope)</span>
                <span class="text-[10px] text-muted-foreground font-normal">（默认已包含 openid profile email）</span>
                <ModifiedBadge
                  v-if="isDiff('oidc_scopes')"
                  :original-value="getOriginal('oidc_scopes')"
                  @reset="resetField('oidc_scopes')"
                />
              </Label>
              <div class="flex items-center gap-1 flex-wrap">
                <button
                  v-for="s in oidcCommonScopes"
                  :key="s"
                  type="button"
                  @click="addOidcScope(s)"
                  :disabled="demoMode || loading"
                  class="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-foreground border border-border/50 transition-colors disabled:opacity-50 shrink-0"
                >
                  +{{ s }}
                </button>
              </div>
            </div>
            <Input
              v-model="form.oidc_scopes"
              placeholder="例如 groups 或 roles，按需追加"
              class="h-9 text-xs font-mono"
              :class="diffClass('oidc_scopes')"
              :disabled="demoMode || loading"
            />
          </div>
        </div>

        <!-- ==================== 2. 通用 OAuth 2.0 通道专属面板 ==================== -->
        <div v-else class="space-y-3.5 animate-in fade-in duration-200">
          <!-- 通道主开关与在线指示条 -->
          <div
            class="p-3 sm:p-3.5 rounded-xl border flex flex-col sm:flex-row sm:items-center justify-between transition-all gap-2.5 sm:gap-3"
            :class="savedOAuth2Enabled ? 'border-emerald-500/20 bg-emerald-500/5' : (form.oauth2_enabled ? 'border-amber-500/30 bg-amber-500/5' : 'border-border/70 bg-muted/20')"
          >
            <div class="flex items-center gap-2.5 sm:gap-3 min-w-0">
              <div
                class="w-8.5 h-8.5 rounded-lg flex items-center justify-center shrink-0"
                :class="savedOAuth2Enabled ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20' : 'bg-muted text-muted-foreground border border-border/60'"
              >
                <Zap class="w-4 h-4" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-1.5 flex-nowrap">
                  <span class="text-xs font-bold text-foreground shrink-0">OAuth 2.0 通道</span>
                  <Badge
                    variant="outline"
                    class="text-[9px] px-1 py-0 h-4 shrink-0"
                    :class="savedOAuth2Enabled ? 'border-emerald-500/30 text-emerald-600 dark:text-emerald-400' : 'border-muted-foreground/30 text-muted-foreground'"
                  >
                    {{ savedOAuth2Enabled ? '服务在线' : (form.oauth2_enabled ? '待保存' : '已停用') }}
                  </Badge>
                </div>
                <div class="text-[10px] text-muted-foreground pt-0.5 truncate">
                  支持 GitHub / Gitee / GitLab 等第三方应用平台
                </div>
              </div>
            </div>

            <!-- 通道操作区：测试按钮与独立启用开关 -->
            <div class="flex items-center gap-2 sm:gap-2.5 shrink-0 justify-between sm:justify-end w-full sm:w-auto pt-2 sm:pt-0 border-t sm:border-t-0 border-border/50">
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="handleTestOAuth2"
                :disabled="testingOAuth2 || saving || loading || demoMode"
                class="h-8 px-2.5 sm:px-3 text-xs font-medium shadow-xs gap-1.5 bg-background/80 hover:bg-background shrink-0"
              >
                <Loader2 v-if="testingOAuth2" class="h-3.5 w-3.5 animate-spin" />
                <Zap v-else class="h-3.5 w-3.5 text-amber-500" />
                <span>{{ testingOAuth2 ? '测试中...' : '测试 OAuth 2.0 平台端点' }}</span>
              </Button>

              <div class="flex items-center gap-1.5 sm:gap-2 pl-2 sm:pl-2.5 border-l border-border/60 shrink-0">
                <ModifiedBadge
                  v-if="isDiff('oauth2_enabled')"
                  :original-value="getOriginal('oauth2_enabled')"
                  @reset="resetField('oauth2_enabled')"
                />
                <Switch
                  :model-value="form.oauth2_enabled"
                  @update:model-value="(val) => (form.oauth2_enabled = val)"
                  id="oauth2-channel-enabled"
                  :disabled="demoMode || loading"
                />
                <Label for="oauth2-channel-enabled" class="text-xs cursor-pointer select-none">
                  {{ form.oauth2_enabled ? '已开启' : '开启' }}
                </Label>
              </div>
            </div>
          </div>

          <!-- OAuth2 快捷预设模版条 -->
          <div class="p-2.5 px-3 rounded-xl border border-border/70 bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <span class="text-xs text-muted-foreground flex items-center gap-1.5 font-medium shrink-0">
              <Sparkles class="w-3.5 h-3.5 text-amber-500" />
              <span>常用平台一键填充:</span>
            </span>
            <div class="flex items-center gap-2 flex-wrap">
              <button
                v-for="p in oauth2Presets"
                :key="p.name"
                type="button"
                class="h-7 text-xs px-2.5 rounded-md border border-border/80 bg-background hover:bg-muted/80 hover:border-primary/50 text-foreground transition-all flex items-center gap-1 font-medium disabled:opacity-50"
                :disabled="demoMode || loading"
                @click="applyOAuth2Preset(p)"
              >
                <span>{{ p.name }}</span>
              </button>
            </div>
          </div>

          <!-- OAuth2 专用回调地址说明条 -->
          <div class="p-3 rounded-xl border border-sky-500/20 bg-sky-500/5 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
            <div class="space-y-0.5 min-w-0">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                <ShieldCheck class="w-3.5 h-3.5 text-sky-500 shrink-0" />
                <span>OAuth 2.0 专用授权回调地址 (Callback URL)</span>
              </div>
              <p class="text-[11px] text-muted-foreground truncate">
                请在 GitHub / Gitee OAuth App 设置中严格登记该回调地址。
              </p>
            </div>
            <div class="flex items-center gap-1.5 shrink-0 w-full sm:w-auto">
              <code class="text-[11px] font-mono px-2 py-1 rounded bg-background border border-border/80 text-foreground truncate flex-1 min-w-0 sm:max-w-[320px]">
                {{ oauth2RedirectUrl }}
              </code>
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-7.5 px-2.5 text-xs gap-1 font-medium bg-background shadow-xs shrink-0"
                @click="copyUrl(oauth2RedirectUrl, 'oauth2')"
              >
                <Check v-if="copiedOAuth2" class="w-3.5 h-3.5 text-emerald-500" />
                <Copy v-else class="w-3.5 h-3.5" />
                <span>{{ copiedOAuth2 ? '已复制' : '复制' }}</span>
              </Button>
            </div>
          </div>

          <!-- 端点三列并排 -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>授权端点 (Authorize URL)</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_auth_url')"
                  :original-value="getOriginal('oauth2_auth_url')"
                  @reset="resetField('oauth2_auth_url')"
                />
              </Label>
              <Input
                v-model="form.oauth2_auth_url"
                placeholder="https://github.com/login/oauth/authorize"
                class="h-9 text-xs font-mono"
                :class="diffClass('oauth2_auth_url')"
                :disabled="demoMode || loading"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>令牌端点 (Token URL)</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_token_url')"
                  :original-value="getOriginal('oauth2_token_url')"
                  @reset="resetField('oauth2_token_url')"
                />
              </Label>
              <Input
                v-model="form.oauth2_token_url"
                placeholder="https://github.com/login/oauth/access_token"
                class="h-9 text-xs font-mono"
                :class="diffClass('oauth2_token_url')"
                :disabled="demoMode || loading"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>用户信息端点 (Userinfo URL)</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_userinfo_url')"
                  :original-value="getOriginal('oauth2_userinfo_url')"
                  @reset="resetField('oauth2_userinfo_url')"
                />
              </Label>
              <Input
                v-model="form.oauth2_userinfo_url"
                placeholder="https://api.github.com/user"
                class="h-9 text-xs font-mono"
                :class="diffClass('oauth2_userinfo_url')"
                :disabled="demoMode || loading"
              />
            </div>
          </div>

          <!-- 显示名称与唯一用户标识 Key 双列并排 -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>显示名称</span>
                <span class="text-[10px] text-muted-foreground font-normal">（登录页按钮回显，如 GitHub / Gitee）</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_name')"
                  :original-value="getOriginal('oauth2_name')"
                  @reset="resetField('oauth2_name')"
                />
              </Label>
              <Input
                v-model="form.oauth2_name"
                placeholder="例如 GitHub 或 Gitee"
                class="h-9 text-xs"
                :class="diffClass('oauth2_name')"
                :disabled="demoMode || loading"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>唯一用户标识 Key</span>
                <span class="text-[10px] text-muted-foreground font-normal">（留空智能提取）</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_user_key')"
                  :original-value="getOriginal('oauth2_user_key')"
                  @reset="resetField('oauth2_user_key')"
                />
              </Label>
              <Input
                v-model="form.oauth2_user_key"
                placeholder="例如 id 或 sub"
                class="h-9 text-xs font-mono"
                :class="diffClass('oauth2_user_key')"
                :disabled="demoMode || loading"
              />
            </div>
          </div>

          <!-- OAuth 2.0 独立客户端凭据 -->
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-3.5">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5 flex-wrap">
                <KeyRound class="w-3.5 h-3.5 text-muted-foreground" />
                <span>Client ID（OAuth 2.0 客户端标识）</span>
                <span class="text-destructive">*</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_client_id')"
                  :original-value="getOriginal('oauth2_client_id')"
                  @reset="resetField('oauth2_client_id')"
                />
              </Label>
              <Input
                v-model="form.oauth2_client_id"
                placeholder="例如 GitHub OAuth App Client ID"
                class="h-9 text-xs font-mono"
                :class="diffClass('oauth2_client_id')"
                :disabled="demoMode || loading"
              />
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5 flex-wrap">
                <Lock class="w-3.5 h-3.5 text-muted-foreground" />
                <span>Client Secret（OAuth 2.0 客户端密钥）</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_client_secret')"
                  :show-reset="true"
                  @reset="resetField('oauth2_client_secret')"
                />
              </Label>
              <div class="relative">
                <Input
                  v-model="form.oauth2_client_secret"
                  :type="showSecretOAuth2 ? 'text' : 'password'"
                  class="h-9 pr-8 text-xs font-mono"
                  :class="diffClass('oauth2_client_secret')"
                  :disabled="demoMode || loading"
                  placeholder="留空保持不变"
                />
                <button
                  type="button"
                  @click="showSecretOAuth2 = !showSecretOAuth2"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors p-0.5"
                  :title="showSecretOAuth2 ? '隐藏密钥' : '显示明文'"
                >
                  <EyeOff v-if="showSecretOAuth2" class="w-3.5 h-3.5" />
                  <Eye v-else class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>

          <!-- OAuth 2.0 Scope -->
          <div class="space-y-1.5">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-1.5 sm:gap-2">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1 flex-wrap">
                <span>授权范围 (Scope)</span>
                <span class="text-[10px] text-muted-foreground font-normal">（以空格或逗号分隔）</span>
                <ModifiedBadge
                  v-if="isDiff('oauth2_scopes')"
                  :original-value="getOriginal('oauth2_scopes')"
                  @reset="resetField('oauth2_scopes')"
                />
              </Label>
              <div class="flex items-center gap-1 flex-wrap">
                <button
                  v-for="s in oauth2CommonScopes"
                  :key="s"
                  type="button"
                  @click="addOAuth2Scope(s)"
                  :disabled="demoMode || loading"
                  class="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-foreground border border-border/50 transition-colors disabled:opacity-50 shrink-0"
                >
                  +{{ s }}
                </button>
              </div>
            </div>
            <Input
              v-model="form.oauth2_scopes"
              placeholder="例如 read:user user:email"
              class="h-9 text-xs font-mono"
              :class="diffClass('oauth2_scopes')"
              :disabled="demoMode || loading"
            />
          </div>
        </div>

        <!-- 底部提示文字 -->
        <div class="pt-3 border-t border-border/50 text-[11px] text-muted-foreground mt-2">
          多身份源支持同时启用；保存后将在登录页面即时聚合展示对应的登录按钮。
        </div>
      </div>
    </CardContent>
  </Card>
</template>
