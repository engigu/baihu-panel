<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  ShieldCheck,
  ShieldAlert,
  Key,
  Smartphone,
  CheckCircle2,
  Lock,
  Shield,
  Fingerprint
} from 'lucide-vue-next'
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

const supportedApps = [
  'Google Authenticator',
  'Microsoft Authenticator',
  'Bitwarden',
  '1Password'
]

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
    toast.success('两步验证 (2FA) 开启成功')
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
  <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
    <CardHeader class="pb-3 pt-0 px-5">
      <div class="flex items-start justify-between gap-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <CardTitle class="text-base flex items-center gap-2">
              <ShieldCheck class="w-4 h-4 text-emerald-500" />
              <span>两步验证 (2FA) 安全盾</span>
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
            基于 TOTP 动态口令机制，为管理权限提供强力双因子防护。
          </CardDescription>
        </div>
      </div>
    </CardHeader>

    <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
      <!-- 状态 1：未开启状态（展示防护评估与推荐） -->
      <template v-if="!otpEnabled && !showBindSection">
        <div class="space-y-3.5">
          <!-- 安全状态评级指示卡片 -->
          <div class="p-3.5 rounded-xl border border-yellow-500/20 bg-yellow-500/5 flex items-center justify-between">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 flex items-center justify-center shrink-0">
                <ShieldAlert class="w-4 h-4" />
              </div>
              <div>
                <div class="text-xs font-semibold text-yellow-600 dark:text-yellow-500">
                  防护状态：未开启保护
                </div>
                <div class="text-[10px] text-muted-foreground pt-0.5">当前仅依赖单一账号密码验证</div>
              </div>
            </div>

            <Badge variant="outline" class="text-[10px] px-1.5 py-0 border-yellow-500/30 text-yellow-600 dark:text-yellow-400">
              安全等级：中
            </Badge>
          </div>

          <!-- 核心安全收益清单 -->
          <div class="space-y-2 pt-0.5">
            <div class="text-xs font-medium text-foreground flex items-center gap-1.5">
              <Fingerprint class="w-3.5 h-3.5 text-primary" />
              <span>双因子安全特性</span>
            </div>
            <div class="space-y-1.5 text-xs text-muted-foreground leading-relaxed">
              <div class="flex items-start gap-2">
                <CheckCircle2 class="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5" />
                <span>防范凭据撞库：即便登录密码泄露，未知攻击者依然无法登入面板。</span>
              </div>
              <div class="flex items-start gap-2">
                <CheckCircle2 class="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5" />
                <span>实时动态口令：基于通用 TOTP 算法，手机端每 30 秒独立刷新一次。</span>
              </div>
              <div class="flex items-start gap-2">
                <CheckCircle2 class="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5" />
                <span>离线密钥存储：验证器运行于移动设备本地，不依赖短信与公网服务。</span>
              </div>
            </div>
          </div>

          <!-- 支持的验证器应用徽章 -->
          <div class="space-y-1.5">
            <div class="text-[11px] text-muted-foreground">兼容主流认证客户端：</div>
            <div class="flex flex-wrap gap-1.5">
              <Badge
                v-for="app in supportedApps"
                :key="app"
                class="text-[10px] bg-muted/60 text-foreground/80 border-0 font-normal px-2 py-0.5"
              >
                {{ app }}
              </Badge>
            </div>
          </div>
        </div>

        <!-- 底部配置按钮栏 -->
        <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
          <span class="text-[11px] text-muted-foreground">
            强烈建议管理员开启两步验证
          </span>
          <Button
            @click="startBind"
            :disabled="loading || demoMode"
            class="h-8.5 px-4 text-xs font-medium shadow-sm gap-1.5"
          >
            <Shield class="w-3.5 h-3.5" />
            <span>立即配置两步验证</span>
          </Button>
        </div>
      </template>

      <!-- 状态 2：配置激活向导 (Wizard) -->
      <template v-else-if="!otpEnabled && showBindSection">
        <div class="space-y-4">
          <div class="flex items-center justify-between pb-2 border-b border-border/60">
            <div class="text-xs font-bold text-foreground">绑定验证器向导</div>
            <button
              @click="cancelBind"
              class="text-xs text-muted-foreground hover:text-foreground"
            >
              取消
            </button>
          </div>

          <div class="space-y-4">
            <!-- 步骤 1: 扫码 -->
            <div class="space-y-1.5">
              <div class="flex items-center gap-1.5 text-xs font-medium text-foreground">
                <span class="w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] flex items-center justify-center font-bold">1</span>
                <span>手机扫码绑定</span>
              </div>
              <p class="text-[11px] text-muted-foreground pl-5.5 leading-relaxed">
                打开验证器 App，扫描下方二维码建立安全绑定：
              </p>
              <div v-if="otpUrl" class="pl-5.5 pt-1">
                <qrcode-vue :value="otpUrl" :size="130" level="H" class="rounded-xl border p-2 bg-white shadow-sm" />
              </div>
            </div>

            <!-- 步骤 2: 手动密钥 -->
            <div class="space-y-1.5">
              <div class="flex items-center gap-1.5 text-xs font-medium text-foreground">
                <span class="w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] flex items-center justify-center font-bold">2</span>
                <span>备用手动密钥（可选）</span>
              </div>
              <div class="pl-5.5">
                <div class="flex items-center gap-2 bg-muted/40 border border-border/80 rounded-lg p-2 max-w-sm font-mono text-xs">
                  <Key class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  <span class="select-all tracking-wider break-all text-[11px]">{{ otpSecret }}</span>
                </div>
              </div>
            </div>

            <!-- 步骤 3: 校验激活 -->
            <div class="space-y-1.5">
              <div class="flex items-center gap-1.5 text-xs font-medium text-foreground">
                <span class="w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] flex items-center justify-center font-bold">3</span>
                <span>输入 6 位动态验证码</span>
              </div>
              <div class="pl-5.5 max-w-xs">
                <Input
                  v-model="bindCode"
                  placeholder="000000"
                  maxlength="6"
                  class="h-9 text-center tracking-[0.4em] font-mono font-bold text-sm"
                  @keyup.enter="handleBind"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="pt-3 border-t border-border/50 flex justify-end gap-2.5 mt-1">
          <Button variant="outline" size="sm" class="h-8 text-xs" @click="cancelBind">取消</Button>
          <Button
            size="sm"
            class="h-8 text-xs font-medium"
            @click="handleBind"
            :disabled="loading || bindCode.length !== 6"
          >
            校验并完成绑定
          </Button>
        </div>
      </template>

      <!-- 状态 3：已开启保护状态 -->
      <template v-else-if="otpEnabled">
        <div class="space-y-3.5">
          <!-- 绿色最高防护卡片 -->
          <div class="p-3.5 rounded-xl border border-emerald-500/20 bg-emerald-500/5 flex items-center justify-between">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
                <ShieldCheck class="w-4 h-4" />
              </div>
              <div>
                <div class="text-xs font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span>防护状态：已全面生效</span>
                </div>
                <div class="text-[10px] text-muted-foreground pt-0.5">管理员登录受 TOTP 双因子全面保护</div>
              </div>
            </div>

            <Badge variant="outline" class="text-[10px] px-1.5 py-0 border-emerald-500/30 text-emerald-600 dark:text-emerald-400">
              安全等级：最高
            </Badge>
          </div>

          <!-- 激活状态说明 -->
          <div class="space-y-2 pt-0.5">
            <div class="text-xs font-medium text-foreground flex items-center gap-1.5">
              <Lock class="w-3.5 h-3.5 text-emerald-500" />
              <span>当前安全策略运行中</span>
            </div>
            <p class="text-xs text-muted-foreground leading-relaxed">
              每次从新设备访问或登录凭据失效时，系统除验证账号密码外，必须校验您绑定的移动端实时动态口令。
            </p>
          </div>

          <div class="p-3 rounded-xl bg-muted/20 border border-border/60 text-[11px] text-muted-foreground flex items-start gap-2">
            <ShieldCheck class="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5" />
            <span>若更换手机或重装验证器，建议先关闭两步验证再重新绑定。</span>
          </div>
        </div>

        <!-- 底部关闭按钮栏 -->
        <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
          <span class="text-[11px] text-muted-foreground">
            关闭保护将降低账户防御风险等级
          </span>
          <Button
            variant="destructive"
            size="sm"
            @click="triggerDisable"
            :disabled="loading || demoMode"
            class="h-8.5 px-3.5 text-xs font-medium shadow-sm gap-1.5"
          >
            <ShieldAlert class="w-3.5 h-3.5" />
            <span>关闭两步验证</span>
          </Button>
        </div>
      </template>
    </CardContent>

    <!-- 关闭二次确认验证弹窗 -->
    <Dialog v-model:open="showDisableDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <Smartphone class="w-5 h-5 text-destructive" />
            <span>关闭两步验证保护</span>
          </DialogTitle>
          <DialogDescription class="text-xs pt-1">
            为了防止恶意关闭，执行解绑操作前需要输入您手机验证器上的 6 位当前动态口令。
          </DialogDescription>
        </DialogHeader>
        <div class="py-3 space-y-2">
          <Label for="disable-code" class="text-xs font-medium">当前 6 位动态验证码</Label>
          <Input
            id="disable-code"
            v-model="disableCode"
            placeholder="000000"
            class="h-9 text-center tracking-[0.4em] font-mono font-bold text-sm"
            maxlength="6"
            @keyup.enter="handleDisable"
          />
        </div>
        <DialogFooter class="gap-2 sm:gap-0">
          <Button variant="outline" @click="showDisableDialog = false" :disabled="isDisabling" class="text-xs h-8">取消</Button>
          <Button
            variant="destructive"
            @click="handleDisable"
            :disabled="isDisabling || disableCode.length !== 6"
            class="text-xs h-8"
          >
            <span v-if="isDisabling">正在验证...</span>
            <span v-else>确认关闭保护</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </Card>
</template>
