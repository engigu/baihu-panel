<script setup lang="ts">
import { computed } from 'vue'
import { Input } from '@/components/ui/input'
import { Zap, Clock } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { getCronDescription } from '@/utils/cron'

const props = defineProps<{
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const schedule = computed({
  get: () => props.modelValue || '',
  set: (val) => emit('update:modelValue', val)
})

const cronPresets = [
  { label: '每5秒', value: '*/5 * * * * *' },
  { label: '每30秒', value: '*/30 * * * * *' },
  { label: '每分钟', value: '0 * * * * *' },
  { label: '每5分钟', value: '0 */5 * * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每天0点', value: '0 0 0 * * *' },
  { label: '每天8点', value: '0 0 8 * * *' },
  { label: '每周一', value: '0 0 0 * * 1' },
  { label: '每月1号', value: '0 0 0 1 * *' },
]

const cronDescription = computed(() => {
  if (!schedule.value) return ''
  return getCronDescription(schedule.value, (navigator as any).language)
})
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
    <div class="sm:text-right flex sm:block items-center gap-2">
      <slot name="label">
        <label class="text-xs text-foreground/70 uppercase tracking-wider font-bold">定时规则</label>
      </slot>
    </div>
    <div class="sm:col-span-3">
      <Input v-model="schedule" placeholder="秒 分 时 日 月 周 (必须 6 位)" :class="cn('h-9 bg-muted/30 border-muted-foreground/20 transition-all focus:ring-1 focus:ring-primary/40 focus:border-primary/40', schedule ? 'font-mono text-sm tracking-[0.1em] font-medium' : 'text-[11px] font-normal')" autocomplete="off" />
      <div v-if="cronDescription" class="mt-2.5 p-2 rounded-lg bg-primary/5 border border-primary/10 text-[11px] text-primary font-medium flex items-center gap-2 animate-in fade-in slide-in-from-top-1 duration-300">
        <Zap class="h-3 w-3 shrink-0" />
        {{ cronDescription }}
      </div>
      <div class="mt-2.5 space-y-2">
        <div class="flex items-center gap-1.5 text-[10px] text-muted-foreground/70 uppercase font-bold tracking-tighter">
          <Clock class="h-3 w-3" /> 格式指导: 秒 分 时 日 月 周
        </div>
        <div class="flex flex-wrap gap-1.5">
          <button v-for="preset in cronPresets" :key="preset.value"
            class="px-2 py-1 text-[10px] rounded-md bg-muted/50 border border-muted-foreground/10 hover:border-primary/50 hover:bg-primary/5 hover:text-primary transition-all font-medium"
            @click.prevent="schedule = preset.value">
            {{ preset.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
