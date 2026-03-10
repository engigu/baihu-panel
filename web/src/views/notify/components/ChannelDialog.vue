<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { NotifyChannel, ChannelType } from '@/api'

const props = defineProps<{
  open: boolean
  isEditing: boolean
  channel: Partial<NotifyChannel>
  channelTypes: ChannelType[]
  configFields: Record<string, { key: string; label: string; required: boolean; placeholder?: string; type?: string }[]>
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:channel': [channel: Partial<NotifyChannel>]
  'type-change': [type: string]
  save: []
}>()

const currentConfigFields = computed(() => {
  return props.configFields[props.channel.type || ''] || []
})

const enabledModel = computed({
  get: () => props.channel.enabled ?? true,
  set: (value) => updateChannelField('enabled', value)
})

function updateChannelField(field: string, value: any) {
  emit('update:channel', { ...props.channel, [field]: value })
}

function updateConfigField(key: string, value: string) {
  const newConfig = { ...props.channel.config, [key]: value }
  emit('update:channel', { ...props.channel, config: newConfig })
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-lg max-h-[80vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ isEditing ? '编辑渠道' : '添加渠道' }}</DialogTitle>
        <DialogDescription>配置消息推送渠道</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="grid grid-cols-4 items-center gap-3">
          <Label class="text-right text-sm">名称</Label>
          <Input 
            :model-value="channel.name" 
            @update:model-value="updateChannelField('name', $event)"
            placeholder="给渠道起个名字" 
            class="col-span-3" 
          />
        </div>

        <div class="grid grid-cols-4 items-center gap-3">
          <Label class="text-right text-sm">类型</Label>
          <div class="col-span-3">
            <Select 
              :model-value="channel.type" 
              @update:model-value="(val: any) => emit('type-change', String(val))" 
              :disabled="isEditing"
            >
              <SelectTrigger>
                <SelectValue placeholder="选择渠道类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="ct in channelTypes" :key="ct.type" :value="ct.type">
                  {{ ct.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-4 items-center gap-3">
          <Label class="text-right text-sm">启用</Label>
          <Switch v-model="enabledModel" />
        </div>

        <div v-if="currentConfigFields.length > 0" class="border-t pt-4 mt-4">
          <h4 class="text-sm font-medium mb-3 text-muted-foreground">渠道配置</h4>
          <div class="space-y-3">
            <div v-for="field in currentConfigFields" :key="field.key" class="grid grid-cols-4 items-start gap-3">
              <Label class="text-right text-sm pt-2">
                {{ field.label }}
                <span v-if="field.required" class="text-destructive">*</span>
              </Label>
              <div class="col-span-3">
                <Input
                  v-if="!field.type || field.type !== 'textarea'"
                  :model-value="channel.config?.[field.key] || ''"
                  @update:model-value="updateConfigField(field.key, String($event))"
                  :placeholder="field.placeholder || ''"
                  class="text-sm"
                />
                <textarea
                  v-else
                  :value="channel.config?.[field.key] || ''"
                  @input="(e: Event) => updateConfigField(field.key, (e.target as HTMLTextAreaElement).value)"
                  :placeholder="field.placeholder || ''"
                  class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring min-h-[80px]"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button @click="emit('save')">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
