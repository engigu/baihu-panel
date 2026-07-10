<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Zap } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import type { Task } from '@/api'

const props = withDefaults(defineProps<{
  modelValue: Partial<Task>
  showRetry?: boolean
}>(), {
  showRetry: true
})

const emit = defineEmits<{
  'update:modelValue': [value: Partial<Task>]
}>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// === 日志清理配置 ===
const cleanType = ref('none')
const cleanKeep = ref(30)

// 解析初始清理配置
function parseCleanConfig() {
  if (form.value.clean_config) {
    try {
      const config = JSON.parse(form.value.clean_config)
      cleanType.value = config.type || 'none'
      cleanKeep.value = config.keep || 30
    } catch {
      cleanType.value = 'none'
      cleanKeep.value = 30
    }
  } else {
    if (!form.value.id) {
      cleanType.value = 'count'
      cleanKeep.value = 30
    } else {
      cleanType.value = 'none'
      cleanKeep.value = 30
    }
  }
}

// 保存清理配置
function updateCleanConfig() {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) {
    form.value.clean_config = ''
  } else {
    form.value.clean_config = JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
  }
}

watch(cleanType, updateCleanConfig)
watch(cleanKeep, updateCleanConfig)

// === 并发控制配置 ===
const concurrencyEnabled = ref(false)

// 解析初始并发配置
function parseConcurrencyConfig() {
  let configStr = form.value.config
  if (!configStr) {
    concurrencyEnabled.value = true // 默认开启
    return
  }
  try {
    const parsed = JSON.parse(configStr)
    if (parsed && typeof parsed === 'object') {
      const val = parsed['$task_concurrency']
      if (typeof val === 'number') {
        concurrencyEnabled.value = val === 1
      } else {
        concurrencyEnabled.value = true
      }
    }
  } catch {
    concurrencyEnabled.value = true
  }
}

// 保存并发配置
function updateConcurrencyConfig(enabled: boolean) {
  concurrencyEnabled.value = enabled
  let config: Record<string, any> = {}
  if (form.value.config) {
    try {
      const parsed = JSON.parse(form.value.config)
      if (parsed && typeof parsed === 'object') {
        config = parsed
      }
    } catch { }
  }
  config['$task_concurrency'] = enabled ? 1 : 0
  form.value.config = JSON.stringify(config)
}

// 初始化解析
watch(() => form.value.id, () => {
  parseCleanConfig()
  parseConcurrencyConfig()
}, { immediate: true })

</script>

<template>
  <!-- 随机延迟 -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">随机延迟</Label>
    <div class="sm:col-span-3 flex items-center gap-4">
      <div class="flex items-center gap-2">
        <Input :model-value="form.random_range" @update:model-value="(v: string | number) => form.random_range = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center font-semibold text-xs" />
        <span class="text-xs font-semibold text-muted-foreground">秒</span>
      </div>
      <div class="flex-1 text-[11px] text-muted-foreground leading-snug p-2 rounded-lg bg-blue-500/5 border border-blue-500/10 italic">
        基准时间后随机延迟 0~{{ form.random_range || 0 }}s
      </div>
    </div>
  </div>

  <!-- 执行超时 -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">执行超时</Label>
    <div class="sm:col-span-3">
      <div class="flex items-center gap-2">
        <Input :model-value="form.timeout" @update:model-value="(v: string | number) => form.timeout = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center font-semibold text-xs" />
        <span class="text-[11px] font-semibold text-muted-foreground">分钟超时</span>
      </div>
    </div>
  </div>

  <!-- 失败策略 -->
  <div v-if="showRetry" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">失败策略</Label>
    <div class="sm:col-span-3 flex items-center gap-4">
      <div class="flex items-center gap-2">
        <span class="text-[11px] text-muted-foreground font-semibold">重试</span>
        <Input :model-value="form.retry_count" @update:model-value="(v: string | number) => form.retry_count = Number(v || 0)" type="number" :min="0" class="w-16 h-9 bg-muted/30 text-center font-semibold text-xs" />
        <span class="text-[11px] text-muted-foreground font-semibold">次，间隔</span>
        <Input :model-value="form.retry_interval" @update:model-value="(v: string | number) => form.retry_interval = Number(v || 0)" type="number" :min="0" class="w-16 h-9 bg-muted/30 text-center font-semibold text-xs" />
        <span class="text-[11px] text-muted-foreground font-semibold">秒</span>
      </div>
    </div>
  </div>

  <!-- 日志清理 -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">日志清理</Label>
    <div class="sm:col-span-3">
      <div class="flex items-center gap-2">
        <Select v-model="cleanType">
          <SelectTrigger class="w-28 h-9 text-xs bg-muted/10">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">保留日志</SelectItem>
            <SelectItem value="day">按天清理</SelectItem>
            <SelectItem value="count">按条清理</SelectItem>
          </SelectContent>
        </Select>
        <div v-if="cleanType && cleanType !== 'none'" class="flex items-center gap-2">
          <Input v-model="cleanKeep" type="number" class="w-20 h-9 bg-muted/30 text-center font-semibold text-xs" />
          <span class="text-[11px] font-semibold text-muted-foreground">{{ cleanType === 'day' ? '天' : '条' }}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- 运行策略 (包含并发控制，支持外部通过 slot 注入自动添加任务等) -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">运行策略</Label>
    <div class="sm:col-span-3 space-y-4">
      <!-- 供 RepoDialog 插入特有的自动添加任务开关等 -->
      <slot name="run-strategy-prepend"></slot>

      <div class="p-3 rounded-xl bg-muted/20 border border-muted-foreground/10 space-y-2.5">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2 text-xs font-semibold">
            <Zap :class="cn('h-3.5 w-3.5', concurrencyEnabled ? 'text-primary' : 'text-muted-foreground')" /> 
            并发控制
          </div>
          <Switch :model-value="concurrencyEnabled" @update:model-value="updateConcurrencyConfig" />
        </div>
        <p class="text-[11px] text-muted-foreground leading-relaxed">
          {{ concurrencyEnabled ? '允许同时开启多个副本。' : '当前任务/同步未结束时，新触发将被静默忽略。' }}
        </p>
      </div>

      <slot name="run-strategy-append"></slot>
    </div>
  </div>
</template>
