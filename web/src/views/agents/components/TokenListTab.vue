<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Plus, Ticket, Check, X, Copy, Pencil, Trash2 } from 'lucide-vue-next'
import { type AgentToken, api } from '@/api'
import { toast } from 'vue-sonner'
import { copyToClipboard } from '@/utils/clipboard'

const props = defineProps<{
  tokens: AgentToken[]
}>()

const emit = defineEmits<{
  (e: 'create-token'): void
  (e: 'edit-token', token: AgentToken): void
  (e: 'refresh'): void
}>()

function isTokenExpired(token: AgentToken) {
  if (!token.expires_at) return false
  const dateStr = token.expires_at.replace(' ', 'T')
  return new Date(dateStr) < new Date()
}

function isTokenExhausted(token: AgentToken) {
  return token.max_uses > 0 && token.used_count >= token.max_uses
}

function copyToken(token: string) {
  copyToClipboard(token).then((success) => {
    if (success) toast.success('已复制')
  })
}

function openTokenDialog() {
  emit('create-token')
}

function openEditToken(token: AgentToken) {
  emit('edit-token', token)
}

async function deleteToken(id: string) {
  try {
    await api.agents.deleteToken(id)
    emit('refresh')
    toast.success('删除成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div class="rounded-lg border bg-card overflow-hidden">
    <!-- 表头 -->
    <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
      <span class="w-8 shrink-0">状态</span>
      <span class="flex-1 min-w-0">令牌</span>
      <span class="w-32 shrink-0 hidden sm:block">备注</span>
      <span class="w-16 shrink-0 text-center hidden sm:block">次数</span>
      <span class="w-32 shrink-0 hidden md:block">过期时间</span>
      <span class="w-24 shrink-0 flex justify-end">
        <Button size="sm" class="h-6 px-2 text-[10px]" @click="openTokenDialog">
          <Plus class="h-3 w-3 mr-1" />生成
        </Button>
      </span>
    </div>
    <!-- 数据行 -->
    <div class="divide-y text-sm">
      <div v-if="tokens.length === 0" class="text-center py-12 text-muted-foreground">
        <Ticket class="h-8 w-8 mx-auto mb-2 opacity-50" />暂无令牌
      </div>
      <div v-for="token in tokens" :key="token.id"
        class="flex items-center gap-4 px-4 py-1.5 hover:bg-muted/30 transition-colors">
        <!-- 状态 -->
        <span class="w-8 shrink-0 flex justify-center">
          <div v-if="!isTokenExpired(token) && !isTokenExhausted(token)"
            class="h-5 w-5 rounded-full bg-green-500/10 flex items-center justify-center">
            <Check class="h-3 w-3 text-green-500 stroke-[3]" />
          </div>
          <div v-else class="h-5 w-5 rounded-full bg-red-500/10 flex items-center justify-center">
            <X class="h-3 w-3 text-red-500 stroke-[3]" />
          </div>
        </span>
        <!-- Token -->
        <code class="flex-1 min-w-0 font-mono text-xs bg-muted/40 px-2 py-0.5 rounded truncate text-muted-foreground">{{ token.token }}</code>
        <!-- 备注 -->
        <span class="w-32 shrink-0 text-xs text-muted-foreground truncate hidden sm:block">{{ token.remark || '-' }}</span>
        <!-- 使用次数 -->
        <span class="w-16 shrink-0 text-xs text-muted-foreground text-center hidden sm:block tabular-nums">
          {{ token.used_count }}/{{ token.max_uses === 0 ? '∞' : token.max_uses }}
        </span>
        <!-- 过期时间 -->
        <span class="w-32 shrink-0 text-[11px] text-muted-foreground truncate hidden md:block tabular-nums">
          {{ token.expires_at || '永不过期' }}
        </span>
        <!-- 操作 -->
        <span class="w-24 shrink-0 flex justify-end items-center">
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="copyToken(token.token)" title="复制">
            <Copy class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEditToken(token)" title="编辑">
            <Pencil class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6 text-destructive" @click="deleteToken(token.id)" title="删除">
            <Trash2 class="h-3 w-3" />
          </Button>
        </span>
      </div>
    </div>
  </div>
</template>
