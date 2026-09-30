<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  AlertTriangle,
  ShieldCheck,
  KeyRound,
  User,
  Lock,
  Eye,
  EyeOff,
  UserCheck
} from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const initialUsername = ref('')
const username = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const demoMode = ref(false)

const showNewPwd = ref(false)
const showConfirmPwd = ref(false)

// 校验弹窗状态
const showVerifyDialog = ref(false)
const verifyOldUsername = ref('')
const verifyOldPassword = ref('')
const isSubmitting = ref(false)

async function loadData() {
  try {
    const [publicSite, me] = await Promise.all([
      api.settings.getPublicSite(),
      api.auth.me()
    ])
    demoMode.value = publicSite.demo_mode || false
    username.value = me.username
    initialUsername.value = me.username
  } catch {
    // ignore
  }
}

// 点击保存，先进行前置校验
function prepareUpdate() {
  if (demoMode.value) {
    toast.error('演示模式下禁止修改凭据')
    return
  }

  // 1. 基础合法性校验
  if (!username.value.trim()) {
    toast.error('账户名不能为空')
    return
  }

  // 2. 检查是否有实质性变更
  const isUsernameChanged = username.value !== initialUsername.value
  const isPasswordChanged = !!newPassword.value

  if (!isUsernameChanged && !isPasswordChanged) {
    toast.info('未检测到任何修改内容')
    return
  }

  // 3. 密码一致性和长度校验（如果尝试修改密码）
  if (isPasswordChanged) {
    if (newPassword.value.length < 6) {
      toast.error('新密码长度至少需要 6 位')
      return
    }
    if (newPassword.value !== confirmPassword.value) {
      toast.error('两次输入的新密码不一致')
      return
    }
  }

  // 所有前置校验通过，重置并打开验证弹窗
  verifyOldUsername.value = ''
  verifyOldPassword.value = ''
  showVerifyDialog.value = true
}

// 弹窗确认，执行真正的修改流程
async function handleFinalUpdate() {
  if (!verifyOldUsername.value || !verifyOldPassword.value) {
    toast.error('请输入原账号和密码进行验证')
    return
  }

  isSubmitting.value = true
  try {
    const res: any = await api.settings.changePassword({
      old_username: verifyOldUsername.value,
      username: username.value,
      old_password: verifyOldPassword.value,
      new_password: newPassword.value || undefined
    })
    
    showVerifyDialog.value = false
    toast.success(res || '账户凭据修改成功')
    
    if (res && res.includes('重新登录')) {
      setTimeout(() => {
        window.location.reload()
      }, 1500)
    }

    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e: any) {
    toast.error(e.message || '验证失败，无法修改凭据')
  } finally {
    isSubmitting.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
    <CardHeader class="pb-3 pt-0 px-5">
      <div class="flex items-start justify-between gap-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <CardTitle class="text-base flex items-center gap-2">
              <KeyRound class="w-4 h-4 text-primary" />
              <span>账户凭据管理</span>
            </CardTitle>
            <Badge
              v-if="demoMode"
              variant="destructive"
              class="text-[10px] px-1.5 py-0 font-normal"
            >
              演示模式禁用
            </Badge>
          </div>
          <CardDescription class="text-xs leading-relaxed pt-0.5">
            维护系统管理员主账号名称及登录密码，强化访问安全保护。
          </CardDescription>
        </div>
      </div>
    </CardHeader>

    <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
      <div class="space-y-3.5">
        <!-- 演示模式警示 -->
        <div v-if="demoMode" class="flex items-center gap-2 p-3 rounded-xl bg-destructive/10 text-destructive text-xs">
          <AlertTriangle class="h-4 w-4 shrink-0" />
          <span>演示模式下已锁定敏感凭据修改，防止公开访问被恶意篡改。</span>
        </div>

        <!-- 当前账号身份卡片 -->
        <div class="p-3.5 rounded-xl border border-border/70 bg-muted/20 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-primary/10 border border-primary/20 text-primary flex items-center justify-center shrink-0">
              <UserCheck class="w-4 h-4" />
            </div>
            <div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold text-foreground">
                  {{ initialUsername || 'admin' }}
                </span>
                <Badge variant="outline" class="text-[9px] px-1 py-0 h-4 border-primary/30 text-primary">
                  超级管理员
                </Badge>
              </div>
              <div class="text-[10px] text-muted-foreground pt-0.5">系统主管理权限主体</div>
            </div>
          </div>

          <div class="flex items-center gap-1.5 text-[11px] text-emerald-600 dark:text-emerald-400 font-medium">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            <span>当前登录</span>
          </div>
        </div>

        <!-- 凭据修改表单 -->
        <div class="space-y-3 pt-0.5">
          <!-- 账号名输入 -->
          <div class="space-y-1.5">
            <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
              <User class="w-3.5 h-3.5 text-muted-foreground" />
              <span>新账户名</span>
            </Label>
            <Input
              v-model="username"
              placeholder="输入新的管理员登录账号"
              class="h-9 text-xs"
              :disabled="demoMode"
            />
          </div>

          <!-- 双列密码输入 -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Lock class="w-3.5 h-3.5 text-muted-foreground" />
                <span>新登录密码</span>
              </Label>
              <div class="relative">
                <Input
                  v-model="newPassword"
                  :type="showNewPwd ? 'text' : 'password'"
                  placeholder="若不修改请留空"
                  class="h-9 text-xs pr-8"
                  :disabled="demoMode"
                  autocomplete="new-password"
                />
                <button
                  type="button"
                  @click="showNewPwd = !showNewPwd"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <EyeOff v-if="showNewPwd" class="w-3.5 h-3.5" />
                  <Eye v-else class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Lock class="w-3.5 h-3.5 text-muted-foreground" />
                <span>确认新密码</span>
              </Label>
              <div class="relative">
                <Input
                  v-model="confirmPassword"
                  :type="showConfirmPwd ? 'text' : 'password'"
                  placeholder="再次输入新密码"
                  class="h-9 text-xs pr-8"
                  :disabled="demoMode"
                  autocomplete="new-password"
                />
                <button
                  type="button"
                  @click="showConfirmPwd = !showConfirmPwd"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <EyeOff v-if="showConfirmPwd" class="w-3.5 h-3.5" />
                  <Eye v-else class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部操作按钮栏 -->
      <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
        <span class="text-[11px] text-muted-foreground">
          修改凭据需通过原密码安全验证
        </span>
        <Button
          @click="prepareUpdate"
          :disabled="demoMode"
          class="h-8.5 px-4 text-xs font-medium shadow-sm"
        >
          保存凭据修改
        </Button>
      </div>
    </CardContent>

    <!-- 身份验证弹窗 -->
    <Dialog v-model:open="showVerifyDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-primary" />
            <span>安全身份验证</span>
          </DialogTitle>
          <DialogDescription class="text-xs pt-1">
            为了确保系统安全，修改登录凭据前需要验证您当前拥有的有效账号及密码。
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-3.5 py-3">
          <div class="space-y-1.5">
            <Label for="old-user" class="text-xs font-medium">当前有效账号</Label>
            <Input id="old-user" v-model="verifyOldUsername" placeholder="请输入当前使用的登录账号" class="h-9 text-xs" />
          </div>
          <div class="space-y-1.5">
            <Label for="old-pwd" class="text-xs font-medium">当前有效密码</Label>
            <Input id="old-pwd" v-model="verifyOldPassword" type="password" placeholder="请输入当前账号的有效密码" class="h-9 text-xs" />
          </div>
        </div>
        <DialogFooter class="gap-2 sm:gap-0">
          <Button variant="outline" @click="showVerifyDialog = false" :disabled="isSubmitting" class="text-xs h-8">取消</Button>
          <Button @click="handleFinalUpdate" :disabled="isSubmitting" class="text-xs h-8">
            <span v-if="isSubmitting">验证校验中...</span>
            <span v-else>确认提交修改</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </Card>
</template>

<style scoped>
:deep(input:-webkit-autofill) {
  -webkit-box-shadow: 0 0 0 1000px hsl(var(--input)) inset !important;
  -webkit-text-fill-color: hsl(var(--foreground)) !important;
  transition: background-color 5000s ease-in-out 0s;
}
</style>
