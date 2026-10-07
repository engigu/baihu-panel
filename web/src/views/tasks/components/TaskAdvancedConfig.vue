<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ShieldCheck, Zap, Timer, Shuffle, RotateCcw, History } from 'lucide-vue-next'
import type { Task } from '@/api'

const props = withDefaults(defineProps<{
  modelValue: Partial<Task>
  showRetry?: boolean
  showRandom?: boolean
}>(), {
  showRetry: true,
  showRandom: true
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
// 限制并发运行：true = 限制并发(单实例运行，默认推荐)；false = 允许并发多实例
const limitConcurrency = ref(true)

// 解析初始并发配置
function parseConcurrencyConfig() {
  const configStr = form.value.unified_config
  if (!configStr) {
    limitConcurrency.value = true // 默认限制并发
    return
  }
  try {
    const parsed = JSON.parse(configStr)
    if (parsed && typeof parsed === 'object' && parsed.common) {
      const val = parsed.common.task_concurrency
      // 0 表示限制并发(单实例)，1 表示允许并发多副本。未配置或非 1 均默认为限制并发
      limitConcurrency.value = val !== 1
      return
    }
  } catch { }
  limitConcurrency.value = true // 默认限制并发
}

// 保存并发配置
function updateConcurrencyConfig(limited: boolean) {
  limitConcurrency.value = limited
  let config: Record<string, any> = {}
  const rawStr = form.value.unified_config
  if (rawStr) {
    try {
      const parsed = JSON.parse(rawStr)
      if (parsed && typeof parsed === 'object') {
        config = parsed
      }
    } catch { }
  }
  config.common = config.common || {}
  // limited 为 true 表示限制并发(task_concurrency = 0)，为 false 表示允许并发(task_concurrency = 1)
  config.common.task_concurrency = limited ? 0 : 1
  form.value.unified_config = JSON.stringify(config)
}

// 初始化解析
watch(() => form.value.id, () => {
  parseCleanConfig()
  parseConcurrencyConfig()
}, { immediate: true })

</script>

<template>
  <!-- 任务运行与控制参数 (一行展示一个，开阔舒展，左文右控) -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold pt-1">控制参数</Label>
    <div class="sm:col-span-3 space-y-2.5">
      <!-- 1. 执行超时 -->
      <div class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <Timer class="h-4 w-4 text-muted-foreground shrink-0" />
            <span class="text-xs font-bold text-foreground">执行超时</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-medium">超限终止</span>
          </div>
          <p class="text-[11px] text-muted-foreground">
            单次运行最长耗时，超时将被系统强制结束
          </p>
        </div>
        <div class="flex items-center gap-2 self-start sm:self-auto shrink-0">
          <Input
            :model-value="form.timeout"
            @update:model-value="(v: string | number) => form.timeout = Number(v || 0)"
            type="number"
            :min="1"
            class="w-24 h-8 text-xs font-semibold text-center bg-background"
          />
          <span class="text-xs text-muted-foreground font-medium shrink-0">分钟超时</span>
        </div>
      </div>

      <!-- 2. 随机延迟 (仅计划定时或支持随机模式时展示) -->
      <div v-if="showRandom" class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <Shuffle class="h-4 w-4 text-muted-foreground shrink-0" />
            <span class="text-xs font-bold text-foreground">随机延迟</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-medium">防并发聚集</span>
          </div>
          <p class="text-[11px] text-muted-foreground">
            {{ form.random_range && form.random_range > 0 ? `计划基准时间后随机延迟 0~${form.random_range} 秒执行` : '不开启延迟，到达定时时间后立即执行' }}
          </p>
        </div>
        <div class="flex items-center gap-2 self-start sm:self-auto shrink-0">
          <Input
            :model-value="form.random_range"
            @update:model-value="(v: string | number) => form.random_range = Number(v || 0)"
            type="number"
            :min="0"
            class="w-24 h-8 text-xs font-semibold text-center bg-background"
          />
          <span class="text-xs text-muted-foreground font-medium shrink-0">秒延迟</span>
        </div>
      </div>

      <!-- 3. 失败策略 -->
      <div v-if="showRetry" class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <RotateCcw class="h-4 w-4 text-muted-foreground shrink-0" />
            <span class="text-xs font-bold text-foreground">失败策略</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-medium">异常重试</span>
          </div>
          <p class="text-[11px] text-muted-foreground">
            {{ form.retry_count && form.retry_count > 0 ? `任务异常退出时自动重试 ${form.retry_count} 次，每次间隔 ${form.retry_interval || 0} 秒` : '失败后不重试，直接记录失败状态' }}
          </p>
        </div>
        <div class="flex items-center gap-2 self-start sm:self-auto shrink-0">
          <span class="text-xs text-muted-foreground font-medium">重试</span>
          <Input
            :model-value="form.retry_count"
            @update:model-value="(v: string | number) => form.retry_count = Number(v || 0)"
            type="number"
            :min="0"
            placeholder="0"
            class="w-16 h-8 text-xs font-semibold text-center bg-background"
          />
          <span class="text-xs text-muted-foreground font-medium">次，间隔</span>
          <Input
            :model-value="form.retry_interval"
            @update:model-value="(v: string | number) => form.retry_interval = Number(v || 0)"
            type="number"
            :min="0"
            placeholder="0"
            class="w-16 h-8 text-xs font-semibold text-center bg-background"
          />
          <span class="text-xs text-muted-foreground font-medium">秒</span>
        </div>
      </div>

      <!-- 4. 日志清理 -->
      <div class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <History class="h-4 w-4 text-muted-foreground shrink-0" />
            <span class="text-xs font-bold text-foreground">日志清理</span>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-medium">自动回收</span>
          </div>
          <p class="text-[11px] text-muted-foreground">
            {{ cleanType === 'none' ? '保留所有历史日志，不执行自动清除' : `执行完成后自动保留最近 ${cleanKeep} ${cleanType === 'day' ? '天' : '条'} 日志` }}
          </p>
        </div>
        <div class="flex items-center gap-2 self-start sm:self-auto shrink-0">
          <Select v-model="cleanType">
            <SelectTrigger class="w-28 h-8 text-xs bg-background">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none" class="text-xs">永久保留</SelectItem>
              <SelectItem value="count" class="text-xs">按条清理</SelectItem>
              <SelectItem value="day" class="text-xs">按天清理</SelectItem>
            </SelectContent>
          </Select>
          <div v-if="cleanType && cleanType !== 'none'" class="flex items-center gap-1.5">
            <Input
              v-model.number="cleanKeep"
              type="number"
              :min="1"
              class="w-20 h-8 text-xs font-semibold text-center bg-background"
            />
            <span class="text-xs text-muted-foreground font-medium shrink-0">
              {{ cleanType === 'day' ? '天' : '条' }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 运行策略 (双卡片单选：限制并发 vs 允许并发) -->
  <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-semibold">并发策略</Label>
    <div class="sm:col-span-3 space-y-3">
      <!-- 供 RepoDialog 插入特有的自动添加任务开关等 -->
      <slot name="run-strategy-prepend"></slot>

      <!-- 双选项卡片 -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <!-- 选项 1: 限制并发 (单实例推荐) -->
        <div
          class="p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between select-none"
          :class="limitConcurrency
            ? 'border-emerald-500/40 bg-emerald-500/5 dark:bg-emerald-500/[0.07] shadow-xs'
            : 'border-border/60 bg-muted/10 hover:bg-muted/20 opacity-70 hover:opacity-100'"
          @click="updateConcurrencyConfig(true)"
        >
          <div class="flex items-center justify-between mb-1.5">
            <div class="flex items-center gap-2">
              <ShieldCheck class="h-4 w-4" :class="limitConcurrency ? 'text-emerald-500/80 dark:text-emerald-400/80' : 'text-muted-foreground'" />
              <span class="text-xs font-semibold" :class="limitConcurrency ? 'text-foreground' : 'text-foreground/80'">限制并发</span>
            </div>
            <span
              class="text-[10px] px-1.5 py-0.5 rounded font-mono font-medium"
              :class="limitConcurrency
                ? 'bg-emerald-500/10 text-emerald-600/90 dark:text-emerald-400/80 border border-emerald-500/20'
                : 'bg-muted text-muted-foreground'"
            >
              单实例 · 推荐
            </span>
          </div>
          <p class="text-[11px] leading-relaxed" :class="limitConcurrency ? 'text-foreground/75 dark:text-foreground/70' : 'text-muted-foreground/80'">
            同一时间只跑一个实例。执行未结束时，新触发将被直接拦截跳过，防止冲突。
          </p>
        </div>

        <!-- 选项 2: 允许并发 (多副本并行) -->
        <div
          class="p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between select-none"
          :class="!limitConcurrency
            ? 'border-amber-500/40 bg-amber-500/5 dark:bg-amber-500/[0.07] shadow-xs'
            : 'border-border/60 bg-muted/10 hover:bg-muted/20 opacity-70 hover:opacity-100'"
          @click="updateConcurrencyConfig(false)"
        >
          <div class="flex items-center justify-between mb-1.5">
            <div class="flex items-center gap-2">
              <Zap class="h-4 w-4" :class="!limitConcurrency ? 'text-amber-500/80 dark:text-amber-400/80' : 'text-muted-foreground'" />
              <span class="text-xs font-semibold" :class="!limitConcurrency ? 'text-foreground' : 'text-foreground/80'">允许并发</span>
            </div>
            <span
              class="text-[10px] px-1.5 py-0.5 rounded font-mono font-medium"
              :class="!limitConcurrency
                ? 'bg-amber-500/10 text-amber-600/90 dark:text-amber-400/80 border border-amber-500/20'
                : 'bg-muted text-muted-foreground'"
            >
              多副本并行
            </span>
          </div>
          <p class="text-[11px] leading-relaxed" :class="!limitConcurrency ? 'text-foreground/75 dark:text-foreground/70' : 'text-muted-foreground/80'">
            不做任何限制。无论是否有任务在跑，每次触发都创建新的独立进程同时执行。
          </p>
        </div>
      </div>

      <!-- 选中策略详细说明 -->
      <div
        class="p-2.5 rounded-lg border text-[11px] leading-relaxed transition-all"
        :class="limitConcurrency
          ? 'bg-emerald-500/[0.03] border-emerald-500/20 text-muted-foreground'
          : 'bg-amber-500/[0.03] border-amber-500/20 text-muted-foreground'"
      >
        <template v-if="limitConcurrency">
          <span class="text-foreground/85 font-medium">当前已锁定「单实例保护」</span>：任务在执行尚未结束前，任何新的触发信号（无论是定时计划、文件热同步、还是手动点击执行）均会自动忽略并跳过，确保数据安全，杜绝多进程抢占与重复运行。
        </template>
        <template v-else>
          <span class="text-foreground/85 font-medium">当前已开启「多副本并发」</span>：每次触发均会独立启动新的进程/协程同时运行。请确保脚本或任务原生具备并发幂等性，否则可能导致数据覆盖或资源竞争。
        </template>
      </div>

      <slot name="run-strategy-append"></slot>
    </div>
  </div>
</template>
