<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Plus, Trash2, TestTube, Pencil, Bell, Calendar, Copy, Check } from 'lucide-vue-next'
import type { NotifyChannel, ChannelType } from '@/api'
import { ref } from 'vue'
import { toast } from 'vue-sonner'

defineProps<{
  channels: NotifyChannel[]
  channelTypes: ChannelType[]
}>()

const emit = defineEmits<{
  add: []
  edit: [channel: NotifyChannel]
  delete: [id: string]
  test: [channel: NotifyChannel]
}>()

function getChannelTypeName(type: string, channelTypes: ChannelType[]): string {
  const found = channelTypes.find(t => t.type === type)
  return found ? found.label : type
}

const copiedBlock = ref<string | null>(null)

function copyToClipboard(text: string, blockId: string) {
  navigator.clipboard.writeText(text).then(() => {
    copiedBlock.value = blockId
    toast.success('已复制长 ID')
    setTimeout(() => {
      copiedBlock.value = null
    }, 2000)
  })
}
</script>

<template>
  <Card>
    <CardHeader>
      <div class="flex items-center justify-between">
        <div>
          <CardTitle>通知渠道</CardTitle>
          <CardDescription>管理消息推送渠道配置</CardDescription>
        </div>
        <Button size="sm" @click="emit('add')">
          <Plus class="w-4 h-4 mr-1" />
          添加渠道
        </Button>
      </div>
    </CardHeader>
    <CardContent>
      <div v-if="channels.length === 0" class="text-center py-12 text-muted-foreground">
        <Bell class="w-12 h-12 mx-auto mb-3 opacity-30" />
        <p class="text-sm">暂无通知渠道</p>
        <p class="text-xs mt-1">点击"添加渠道"开始配置</p>
      </div>
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="(ch, index) in channels" :key="ch.id"
          class="flex flex-col p-4 rounded-xl border bg-card hover:bg-accent/30 hover:shadow-md transition-all group relative overflow-hidden">
          <!-- 装饰性背景序号 -->
          <div
            class="absolute -right-2 -top-4 text-6xl font-bold text-primary/5 select-none transition-colors group-hover:text-primary/10">
            {{ index + 1 }}
          </div>

          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <Bell class="w-5 h-5" />
              </div>
              <div class="flex flex-col">
                <span class="font-bold text-sm truncate max-w-[150px]">{{ ch.name }}</span>
                <div class="flex items-center gap-1.5 mt-0.5">
                  <Badge variant="outline" class="text-[9px] px-1 h-4 font-normal">{{ getChannelTypeName(ch.type,
                    channelTypes) }}</Badge>
                  <Badge
                    :class="ch.enabled ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20' : 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20'"
                    variant="secondary" class="text-[9px] px-1 h-4 font-normal">
                    {{ ch.enabled ? '启用' : '禁用' }}
                  </Badge>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-auto pt-4 border-t flex items-center justify-between">
            <div v-if="ch.created_at" class="flex items-center gap-1 text-[10px] text-muted-foreground opacity-60">
              <Calendar class="w-3 h-3" />
              <span>{{ ch.created_at.split(' ')[0] }}</span>
            </div>
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="icon"
                class="h-8 w-8 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                @click="copyToClipboard(ch.id, 'channel-' + ch.id)" title="复制 ID">
                <Check v-if="copiedBlock === 'channel-' + ch.id" class="w-4 h-4 text-emerald-500" />
                <Copy v-else class="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-8 w-8 rounded-full hover:bg-primary/10 hover:text-primary transition-colors"
                @click="emit('test', ch)" title="测试发送">
                <TestTube class="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-8 w-8 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                @click="emit('edit', ch)" title="编辑">
                <Pencil class="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="icon"
                class="h-8 w-8 rounded-full text-destructive hover:bg-destructive/10 transition-colors"
                @click="emit('delete', ch.id)" title="删除">
                <Trash2 class="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
