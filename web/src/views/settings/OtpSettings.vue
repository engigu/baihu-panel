<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ShieldCheck, ShieldAlert, Key, Smartphone } from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import QrcodeVue from 'qrcode.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const otpEnabled = ref(false)
const loading = ref(false)
const demoMode = ref(false)

// 绑定相关状态
const showBindSection = ref(false)
const otpSecret = ref('')
const otpUrl = ref('')
const bindCode = ref('')

// 解绑相关状态
const showDisableDialog = ref(false)
const disableCode = ref('')
const isDisabling = ref(false)

async function loadData() {
  loading.value = true
  try {
    const [publicSite, statusRes] = await Promise.all([
      api.settings.getPublicSite(),
      api.auth.getOtpStatus()
    ])
    demoMode.value = publicSite.demo_mode || false
    otpEnabled.value = statusRes.otp_enabled
  } catch (e: any) {
    toast.error('加载两步验证状态失败')
  } finally {
    loading.value = false
  }
}

// 开始启用绑定流程，生成密钥和二维码
async function startBind() {
  if (demoMode.value) {
    toast.error('演示模式下无法操作两步验证')
    return
  }
  loading.value = true
  try {
    const res = await api.auth.generateOtp()
    otpSecret.value = res.secret
    otpUrl.value = res.url
    bindCode.value = ''
    showBindSection.value = true
  } catch (e: any) {
    toast.error('生成验证密钥失败')
  } finally {
    loading.value = false
  }
}

// 取消绑定流程
function cancelBind() {
  showBindSection.value = false
  otpSecret.value = ''
  otpUrl.value = ''
  bindCode.value = ''
}

// 提交绑定
async function handleBind() {
  if (!bindCode.value || bindCode.value.length !== 6) {
    toast.error('请输入 6 位数字验证码')
    return
  }
  loading.value = true
  try {
    await api.auth.enableOtp({ secret: otpSecret.value, code: bindCode.value })
    toast.success('开启两步验证成功')
    otpEnabled.value = true
    cancelBind()
  } catch (e: any) {
    toast.error(e.message || '绑定失败，请检查验证码是否正确')
  } finally {
    loading.value = false
  }
}

// 点击解绑按钮
function triggerDisable() {
  if (demoMode.value) {
    toast.error('演示模式下无法操作两步验证')
    return
  }
  disableCode.value = ''
  showDisableDialog.value = true
}

// 确认解绑
async function handleDisable() {
  if (!disableCode.value || disableCode.value.length !== 6) {
    toast.error('请输入 6 位数字验证码')
    return
  }
  isDisabling.value = true
  try {
    await api.auth.disableOtp({ code: disableCode.value })
    toast.success('两步验证已成功关闭')
    otpEnabled.value = false
    showDisableDialog.value = false
  } catch (e: any) {
    toast.error(e.message || '验证失败，关闭两步验证失败')
  } finally {
    isDisabling.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-6 max-w-2xl">
    <!-- 未开启状态且不在绑定流程中 -->
    <div v-if="!otpEnabled && !showBindSection" class="space-y-4 pt-2">
      <div class="flex items-center gap-2">
        <ShieldAlert class="h-5 w-5 text-amber-500 shrink-0" />
        <span class="text-sm font-semibold text-amber-500">当前状态：未启用保护</span>
      </div>
      <p class="text-xs text-muted-foreground leading-relaxed">
        两步验证（2FA）为您的账户提供了额外的安全屏障。在开启后，除了用户名和密码，您在每次登录时还需要输入智能手机应用程序（如 Google Authenticator 等）生成的 6 位实时动态验证码。
      </p>
      <div class="flex justify-end pt-2">
        <Button @click="startBind" :disabled="loading" class="shadow-md">
          立即配置两步验证
        </Button>
      </div>
    </div>

    <div v-if="!otpEnabled && showBindSection" class="space-y-6 pt-2">
      <div class="space-y-1 pb-3 border-b">
        <h4 class="text-sm font-semibold">配置两步验证</h4>
        <p class="text-xs text-muted-foreground">请按照以下步骤完成您的安全设置</p>
      </div>

      <div class="space-y-6">
        <!-- 步骤1: 扫码 -->
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-sm font-semibold">
            <span class="flex items-center justify-center w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs">1</span>
            <span>在您的手机上扫描二维码</span>
          </div>
          <p class="text-xs text-muted-foreground pl-7">
            使用您的两步验证 APP（如 Google Authenticator、Microsoft Authenticator、Bitwarden 等）扫描下方二维码。
          </p>
          <div v-if="otpUrl" class="pl-7 pt-2">
            <qrcode-vue :value="otpUrl" :size="160" level="H" class="rounded-xl border p-2 bg-white shadow-sm" />
          </div>
        </div>

        <!-- 步骤2: 密钥备份 -->
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-sm font-semibold">
            <span class="flex items-center justify-center w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs">2</span>
            <span>手动添加密钥（可选）</span>
          </div>
          <p class="text-xs text-muted-foreground pl-7">
            如果您的设备无法扫描二维码，也可以在应用中选择手动添加，并输入以下密钥：
          </p>
          <div class="pl-7">
            <div class="flex items-center gap-2 bg-muted/50 border rounded-xl p-3 max-w-md font-mono text-sm">
              <Key class="w-4 h-4 text-muted-foreground shrink-0" />
              <span class="select-all tracking-wider break-all text-xs">{{ otpSecret }}</span>
            </div>
          </div>
        </div>

        <!-- 步骤3: 校验并激活 -->
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-sm font-semibold">
            <span class="flex items-center justify-center w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs">3</span>
            <span>输入动态验证码以激活</span>
          </div>
          <p class="text-xs text-muted-foreground pl-7">
            请输入您在手机 App 中看到的 6 位实时验证码以完成校验。
          </p>
          <div class="pl-7 max-w-xs space-y-2">
            <Label for="bind-code" class="sr-only">验证码</Label>
            <Input id="bind-code" v-model="bindCode" placeholder="请输入 6 位动态验证码" maxlength="6" class="h-10 text-center tracking-[0.5em] rounded-xl font-bold" @keyup.enter="handleBind" />
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3 pt-4 border-t">
        <Button variant="outline" @click="cancelBind">取消</Button>
        <Button @click="handleBind" :disabled="loading || bindCode.length !== 6">确认绑定并激活</Button>
      </div>
    </div>

    <div v-if="otpEnabled" class="space-y-4 pt-2">
      <div class="flex items-center gap-2">
        <ShieldCheck class="h-5 w-5 text-emerald-500 shrink-0" />
        <span class="text-sm font-semibold text-emerald-500">当前状态：已启用保护</span>
      </div>
      <p class="text-xs text-muted-foreground leading-relaxed">
        您的账户已受到两步验证（2FA）的安全保护。每次从新设备登录或登录会话失效时，均需要输入您移动设备上 App 生成的实时动态密码。
      </p>
      <div class="flex justify-end pt-2">
        <Button variant="destructive" @click="triggerDisable" :disabled="loading" class="shadow-md">
          关闭两步验证
        </Button>
      </div>
    </div>

    <!-- 关闭二次确认弹窗 -->
    <Dialog v-model:open="showDisableDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Smartphone class="w-5 h-5 text-destructive" />
            关闭两步验证
          </DialogTitle>
          <DialogDescription>
            为了您的账户安全，关闭两步验证需要验证您当前手机应用上的动态验证码。
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="disable-code" class="text-right">动态验证码</Label>
            <Input id="disable-code" v-model="disableCode" placeholder="6 位验证码" class="col-span-3 font-bold text-center tracking-[0.5em]" maxlength="6" @keyup.enter="handleDisable" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDisableDialog = false" :disabled="isDisabling">取消</Button>
          <Button variant="destructive" @click="handleDisable" :disabled="isDisabling || disableCode.length !== 6">
            <span v-if="isDisabling">正在验证...</span>
            <span v-else>确认关闭</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
