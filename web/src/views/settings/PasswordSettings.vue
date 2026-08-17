<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AlertTriangle, ShieldCheck } from 'lucide-vue-next'
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
    toast.error('演示模式下不能修改')
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
      toast.error('新密码至少6位')
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
    toast.success(res || '修改成功')
    
    // 如果修改了用户名或密码，后端会返回“请重新登录”的信息
    if (res && res.includes('重新登录')) {
      setTimeout(() => {
        window.location.reload()
      }, 1500)
    }

    // 重置密码框
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e: any) {
    toast.error(e.message || '验证失败')
  } finally {
    isSubmitting.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->


    <div v-if="demoMode" class="flex items-center gap-2 p-3 rounded-md bg-destructive/10 text-destructive text-sm">
      <AlertTriangle class="h-4 w-4 shrink-0" />
      <span>演示模式下禁止修改任何敏感信息</span>
    </div>
    
    <div class="space-y-6">
      <!-- 基础设置展示 -->
      <div class="space-y-4 py-2">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">新账户名</Label>
          <Input v-model="username" placeholder="输入新的登录账号" class="h-9" :disabled="demoMode" />
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <Label class="text-xs font-medium text-foreground">新密码</Label>
            <Input v-model="newPassword" type="password" placeholder="若不修改密码请留空" class="h-9" :disabled="demoMode" autocomplete="new-password" />
          </div>
          <div class="space-y-1.5">
            <Label class="text-xs font-medium text-foreground">确认新密码</Label>
            <Input v-model="confirmPassword" type="password" placeholder="请再次输入新密码" class="h-9" :disabled="demoMode" autocomplete="new-password" />
          </div>
        </div>
      </div>

      <div class="flex justify-end pt-2">
        <Button @click="prepareUpdate" :disabled="demoMode" class="shadow-md">
          提交修改
        </Button>
      </div>
    </div>

    <!-- 身份验证弹窗 -->
    <Dialog v-model:open="showVerifyDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-primary" />
            身份验证
          </DialogTitle>
          <DialogDescription>
            为了您的账户安全，修改账号或密码需要验证您当前的凭据。
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="old-user" class="text-right">原账号</Label>
            <Input id="old-user" v-model="verifyOldUsername" placeholder="请输入当前使用的账号" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="old-pwd" class="text-right">原密码</Label>
            <Input id="old-pwd" v-model="verifyOldPassword" type="password" placeholder="请输入当前账号的密码" class="col-span-3" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showVerifyDialog = false" :disabled="isSubmitting">取消</Button>
          <Button @click="handleFinalUpdate" :disabled="isSubmitting">
            <span v-if="isSubmitting">验证中...</span>
            <span v-else>确认修改</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<style scoped>
/* 确保对话框在自动填充时颜色正常 */
:deep(input:-webkit-autofill) {
  -webkit-box-shadow: 0 0 0 1000px hsl(var(--input)) inset !important;
  -webkit-text-fill-color: hsl(var(--foreground)) !important;
  transition: background-color 5000s ease-in-out 0s;
}
</style>
