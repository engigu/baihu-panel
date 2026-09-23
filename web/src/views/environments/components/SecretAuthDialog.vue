<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { ShieldCheck, KeyRound, Loader2, Eye, EyeOff } from 'lucide-vue-next'
import { api, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import {
  isWebCryptoSupported,
  generateEcdhKeyPair,
  exportEcdhPublicKey,
  decryptEcdhPayload
} from '@/utils/crypto'

const emit = defineEmits<{
  decrypted: [payload: { id: string; value: string }]
}>()

const isOpen = ref(false)
const isLoadingOtpStatus = ref(false)
const isDecrypting = ref(false)

const currentEnv = ref<EnvVar | null>(null)
const otpEnabled = ref(false)
const otpCode = ref('')
const password = ref('')
const showPassword = ref(false)

const inputRef = ref<HTMLInputElement | null>(null)

async function open(env: EnvVar) {
  currentEnv.value = env
  otpCode.value = ''
  password.value = ''
  isOpen.value = true
  isLoadingOtpStatus.value = true

  try {
    const res = await api.auth.getOtpStatus()
    otpEnabled.value = !!res?.otp_enabled
  } catch {
    otpEnabled.value = false
  } finally {
    isLoadingOtpStatus.value = false
    nextTick(() => {
      const el = (inputRef.value as any)?.$el || inputRef.value
      if (typeof el?.focus === 'function') {
        el.focus()
      }
    })
  }
}

async function handleDecrypt() {
  if (!currentEnv.value) return

  if (otpEnabled.value && !otpCode.value.trim()) {
    toast.error('请输入两步验证动态码')
    return
  }
  if (!otpEnabled.value && !password.value) {
    toast.error('请输入登录密码')
    return
  }

  isDecrypting.value = true
  try {
    let plainValue = ''
    const hasWebCrypto = isWebCryptoSupported()

    if (hasWebCrypto) {
      // 1. 本地生成临时的 ECDH (P-256) 密钥对，毫秒级点乘计算，零阻塞
      const keyPair = await generateEcdhKeyPair()
      const publicKey = await exportEcdhPublicKey(keyPair.publicKey)

      // 2. 发起安全解密请求
      const res = await api.env.decryptSecret(currentEnv.value.id, {
        otp_code: otpEnabled.value ? otpCode.value.trim() : undefined,
        password: !otpEnabled.value ? password.value : undefined,
        public_key: publicKey
      })

      if (res.raw_value !== undefined) {
        plainValue = res.raw_value
      } else if (res.server_public_key && res.ciphertext && res.nonce) {
        // 3. ECDH (P-256) 派生 256 位密钥 + AES-GCM 本地解密
        plainValue = await decryptEcdhPayload(keyPair.privateKey, {
          server_public_key: res.server_public_key,
          ciphertext: res.ciphertext,
          nonce: res.nonce
        })
      } else {
        throw new Error('服务端返回的密文载荷异常')
      }
    } else {
      // 局域网纯 HTTP IP 访问等环境无 Web Crypto 支持，平滑降级（在服务端二次核验后直接返回明文）
      const res = await api.env.decryptSecret(currentEnv.value.id, {
        otp_code: otpEnabled.value ? otpCode.value.trim() : undefined,
        password: !otpEnabled.value ? password.value : undefined
      })
      if (res.raw_value !== undefined) {
        plainValue = res.raw_value
      } else {
        throw new Error('未能在当前网络环境下解密机密')
      }
    }

    console.log('[SecretAuth] 解密成功，明文长度:', plainValue.length, '值:', plainValue)
    toast.success('机密解密成功')
    emit('decrypted', { id: currentEnv.value.id, value: plainValue })
    isOpen.value = false
  } catch (err: any) {
    const msg = err?.msg || err?.message || '解密失败，请检查验证码或密码'
    toast.error(msg)
  } finally {
    isDecrypting.value = false
  }
}

watch(isOpen, (val) => {
  if (!val) {
    otpCode.value = ''
    password.value = ''
    currentEnv.value = null
  }
})

defineExpose({
  open
})
</script>

<template>
  <BaihuDialog
    v-model:open="isOpen"
    title="机密身份核验"
    description="查看机密明文前需进行安全身份二次核验，数据通过本地临时 ECDH 椭圆曲线与 AES-GCM 端到端协商加密传输。"
    class="sm:max-w-[420px]"
  >
    <div class="space-y-4 py-2">
      <div v-if="currentEnv" class="rounded-lg bg-muted/50 p-3 border text-xs space-y-1">
        <div class="text-muted-foreground">待解密机密名称:</div>
        <div class="font-mono font-semibold text-foreground truncate">{{ currentEnv.name }}</div>
      </div>

      <div v-if="isLoadingOtpStatus" class="flex items-center justify-center py-6 text-muted-foreground gap-2 text-sm">
        <Loader2 class="h-4 w-4 animate-spin" />
        <span>正在检查两步验证状态...</span>
      </div>

      <!-- OTP 验证码模式 -->
      <div v-else-if="otpEnabled" class="space-y-2">
        <label class="text-xs font-medium flex items-center gap-1.5 text-foreground">
          <ShieldCheck class="h-4 w-4 text-emerald-500" />
          <span>两步验证动态码 (TOTP)</span>
        </label>
        <Input
          ref="inputRef"
          v-model="otpCode"
          type="text"
          inputmode="numeric"
          pattern="[0-9]*"
          maxlength="6"
          placeholder="请输入 6 位动态验证码"
          class="font-mono text-center tracking-widest text-base"
          :disabled="isDecrypting"
          @keydown.enter.prevent="handleDecrypt"
        />
        <p class="text-[11px] text-muted-foreground">已开启两步验证，请打开身份验证器 App 查看验证码</p>
      </div>

      <!-- 登录密码模式 -->
      <div v-else class="space-y-2">
        <label class="text-xs font-medium flex items-center gap-1.5 text-foreground">
          <KeyRound class="h-4 w-4 text-primary" />
          <span>登录密码</span>
        </label>
        <div class="relative">
          <Input
            ref="inputRef"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            placeholder="请输入当前账号登录密码"
            class="pr-10"
            :disabled="isDecrypting"
            @keydown.enter.prevent="handleDecrypt"
          />
          <button
            type="button"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            tabindex="-1"
            @click="showPassword = !showPassword"
          >
            <Eye v-if="!showPassword" class="h-4 w-4" />
            <EyeOff v-else class="h-4 w-4" />
          </button>
        </div>
        <p class="text-[11px] text-muted-foreground">未开启两步验证，请输入当前登录管理员的登录密码</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button variant="outline" :disabled="isDecrypting" @click="isOpen = false">
          取消
        </Button>
        <Button :disabled="isDecrypting || isLoadingOtpStatus" @click="handleDecrypt">
          <Loader2 v-if="isDecrypting" class="mr-2 h-4 w-4 animate-spin" />
          <span>确认解密</span>
        </Button>
      </div>
    </template>
  </BaihuDialog>
</template>
