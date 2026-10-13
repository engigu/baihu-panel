<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { Loader2, Save, Fingerprint } from 'lucide-vue-next'

const loading = ref(false)
const saving = ref(false)
const demoMode = ref(false)

const form = ref({
  enabled: false,
  issuer: '',
  client_id: '',
  client_secret: '',
  redirect_url: '',
  scopes: '',
  auto_create: false,
  default_role: 'user',
})

const redirectPlaceholder = `${window.location.origin}${(window as any).__BASE_URL__ || ''}/api/v1/auth/oidc/callback`

async function loadSection() {
  loading.value = true
  try {
    const res = await api.settings.getSection('security')
    form.value = {
      enabled: res.oidc_enabled === 'true',
      issuer: res.oidc_issuer || '',
      client_id: res.oidc_client_id || '',
      client_secret: res.oidc_client_secret || '',
      redirect_url: res.oidc_redirect_url || '',
      scopes: res.oidc_scopes || '',
      auto_create: res.oidc_auto_create === 'true',
      default_role: res.oidc_default_role === 'admin' ? 'admin' : 'user',
    }
  } catch (e: any) {
    toast.error(e.message || '读取 OIDC 配置失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (form.value.enabled && (!form.value.issuer || !form.value.client_id)) {
    toast.error('启用 OIDC 前请填写 Issuer 与 Client ID')
    return
  }
  saving.value = true
  try {
    await api.settings.setSection('security', {
      oidc_enabled: form.value.enabled ? 'true' : 'false',
      oidc_issuer: form.value.issuer.trim(),
      oidc_client_id: form.value.client_id.trim(),
      oidc_client_secret: form.value.client_secret,
      oidc_redirect_url: form.value.redirect_url.trim(),
      oidc_scopes: form.value.scopes.trim(),
      oidc_auto_create: form.value.auto_create ? 'true' : 'false',
      oidc_default_role: form.value.default_role,
    })
    toast.success('OIDC 配置已保存')
    await loadSection()
  } catch (e: any) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
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
  <Card>
    <CardHeader>
      <CardTitle class="flex items-center gap-2">
        <Fingerprint class="h-5 w-5" />
        OIDC 单点登录
      </CardTitle>
      <CardDescription>
        接入 Pocket ID 等标准 OIDC 提供方，实现免密登录与统一身份管理
      </CardDescription>
    </CardHeader>
    <CardContent class="space-y-5">
      <div class="flex items-center justify-between rounded-lg border p-4">
        <div class="space-y-0.5">
          <Label>启用 OIDC 登录</Label>
          <p class="text-xs text-muted-foreground">启用后登录页会显示「使用 OIDC 单点登录」按钮</p>
        </div>
        <Switch v-model="form.enabled" :disabled="demoMode || loading" />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2 sm:col-span-2">
          <Label>Issuer（发现文档地址）</Label>
          <Input v-model="form.issuer" placeholder="https://id.example.com" :disabled="loading" />
        </div>
        <div class="space-y-2">
          <Label>Client ID</Label>
          <Input v-model="form.client_id" :disabled="loading" />
        </div>
        <div class="space-y-2">
          <Label>Client Secret</Label>
          <Input v-model="form.client_secret" type="password" :disabled="loading" placeholder="留空表示不修改" />
        </div>
        <div class="space-y-2 sm:col-span-2">
          <Label>回调地址（Redirect URI）</Label>
          <Input v-model="form.redirect_url" :placeholder="redirectPlaceholder" :disabled="loading" />
          <p class="text-xs text-muted-foreground">需与 IdP 客户端登记的回调地址完全一致</p>
        </div>
        <div class="space-y-2 sm:col-span-2">
          <Label>附加 Scope（可选，逗号或空格分隔）</Label>
          <Input v-model="form.scopes" placeholder="groups" :disabled="loading" />
        </div>
      </div>

      <div class="flex items-center justify-between rounded-lg border p-4">
        <div class="space-y-0.5">
          <Label>首次登录自动创建账号</Label>
          <p class="text-xs text-muted-foreground">关闭时仅允许已绑定账号登录，未绑定用户将被拒绝</p>
        </div>
        <Switch v-model="form.auto_create" :disabled="demoMode || loading" />
      </div>

      <div class="space-y-2" v-if="form.auto_create">
        <Label>自动创建账号的角色</Label>
        <div class="flex gap-2">
          <Button type="button" size="sm" :variant="form.default_role === 'user' ? 'default' : 'outline'"
            @click="form.default_role = 'user'">普通用户</Button>
          <Button type="button" size="sm" :variant="form.default_role === 'admin' ? 'default' : 'outline'"
            @click="form.default_role = 'admin'">管理员</Button>
        </div>
      </div>

      <div class="flex justify-end">
        <Button @click="save" :disabled="saving || loading || demoMode">
          <Loader2 v-if="saving" class="mr-2 h-4 w-4 animate-spin" />
          <Save v-else class="mr-2 h-4 w-4" />
          保存
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
