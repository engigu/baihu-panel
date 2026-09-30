<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import {
  Cpu,
  Layers,
  Activity,
  Server,
  Sliders,
  Workflow,
  Sparkles,
  Zap,
  CheckCircle2
} from 'lucide-vue-next'

interface SchedulerSettings {
  worker_count: string
  queue_size: string
  rate_interval: string
}

const form = ref<SchedulerSettings>({
  worker_count: '4',
  queue_size: '100',
  rate_interval: '200'
})
const loading = ref(false)
const showConfirm = ref(false)

async function loadSettings() {
  try {
    const res = await api.settings.getScheduler()
    form.value = res
  } catch {}
}

function confirmSave() {
  showConfirm.value = true
}

function applyPreset(name: string, workers: string, queue: string, rate: string) {
  form.value.worker_count = workers
  form.value.queue_size = queue
  form.value.rate_interval = rate
  toast.info(`已快速填入「${name}」参数预设，点击保存即可生效`)
}

async function saveSettings() {
  showConfirm.value = false
  loading.value = true
  try {
    await api.settings.updateScheduler({
      worker_count: String(form.value.worker_count),
      queue_size: String(form.value.queue_size),
      rate_interval: String(form.value.rate_interval)
    })
    toast.success('调度参数保存成功，引擎已热重载')
  } catch {
    toast.error('保存调度设置失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-6">
    <!-- 左右双列平衡卡片 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
      <!-- 左列：调度引擎参数配置 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-3 pt-0 px-5">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <CardTitle class="text-base flex items-center gap-2">
                  <Sliders class="w-4 h-4 text-primary" />
                  <span>调度引擎参数</span>
                </CardTitle>
                <Badge
                  class="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-[10px] px-1.5 py-0 font-normal shadow-none flex items-center gap-1"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span>调度器就绪</span>
                </Badge>
              </div>
              <CardDescription class="text-xs leading-relaxed pt-0.5">
                控制主服务后台子进程并发池、任务排队缓冲深度与防抖启动策略。
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
          <div class="space-y-3.5">
            <!-- 参数 1 & 2：并发 Worker 与 队列深度 -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
              <!-- 并发限制数 -->
              <div class="p-3.5 rounded-xl border border-border/70 bg-muted/20 space-y-2">
                <div class="flex items-center justify-between">
                  <Label class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                    <Cpu class="w-3.5 h-3.5 text-primary" />
                    <span>并发池容量 (Workers)</span>
                  </Label>
                </div>
                <div class="relative">
                  <Input
                    type="number"
                    v-model="form.worker_count"
                    :min="1"
                    class="h-9 text-xs font-bold font-inter pr-12 no-spin"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[11px] text-muted-foreground pointer-events-none">
                    进程
                  </span>
                </div>
                <p class="text-[10px] text-muted-foreground leading-relaxed">
                  全局同时并行执行的任务子进程上限
                </p>
              </div>

              <!-- 最大队列数 -->
              <div class="p-3.5 rounded-xl border border-border/70 bg-muted/20 space-y-2">
                <div class="flex items-center justify-between">
                  <Label class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                    <Layers class="w-3.5 h-3.5 text-blue-500" />
                    <span>缓冲队列 (Queue Size)</span>
                  </Label>
                </div>
                <div class="relative">
                  <Input
                    type="number"
                    v-model="form.queue_size"
                    :min="1"
                    class="h-9 text-xs font-bold font-inter pr-14 no-spin"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[11px] text-muted-foreground pointer-events-none">
                    个任务
                  </span>
                </div>
                <p class="text-[10px] text-muted-foreground leading-relaxed">
                  超出并发时进入内存 FIFO 排队的等待深度
                </p>
              </div>
            </div>

            <!-- 参数 3：频控防抖间隔 -->
            <div class="p-3.5 rounded-xl border border-border/70 bg-muted/20 space-y-2">
              <div class="flex items-center justify-between">
                <Label class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                  <Activity class="w-3.5 h-3.5 text-emerald-500" />
                  <span>调度频控间隔 (Rate Interval)</span>
                </Label>
                <span class="text-[10px] text-muted-foreground">防瞬间 CPU/IO 尖刺</span>
              </div>
              <div class="relative">
                <Input
                  type="number"
                  v-model="form.rate_interval"
                  :min="0"
                  class="h-9 text-xs font-bold font-inter pr-12 no-spin"
                />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[11px] text-muted-foreground pointer-events-none">
                  ms
                </span>
              </div>
              <p class="text-[10px] text-muted-foreground leading-relaxed">
                连续子进程启动的物理间隔时间（如设为 200ms 即系统每秒最多启动 5 个任务进程）
              </p>
            </div>

            <!-- 节点作用域提示条 -->
            <div class="p-3 rounded-xl bg-muted/30 border border-border/60 text-xs text-muted-foreground leading-relaxed flex items-start gap-2.5">
              <Server class="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
              <div>
                <span class="font-medium text-foreground">作用域约束：</span>
                <span>此处配置仅对<strong>主服务 (Master)</strong> 调度引擎生效；分布式 Agent 节点的并发参数请在「节点管理」中独立设定。</span>
              </div>
            </div>
          </div>

          <!-- 底部保存按钮栏 -->
          <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
            <span class="text-[11px] text-muted-foreground">
              保存后即刻热重载生效，运行中的任务不受影响
            </span>
            <Button
              @click="confirmSave"
              :disabled="loading"
              class="h-8.5 px-4 text-xs font-medium shadow-sm gap-1.5"
            >
              <Zap class="w-3.5 h-3.5" />
              <span>{{ loading ? '应用配置中...' : '保存调度配置' }}</span>
            </Button>
          </div>
        </CardContent>
      </Card>

      <!-- 右列：调度机制与调优指南 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-3 pt-0 px-5">
          <div class="flex items-center justify-between">
            <div class="space-y-1">
              <CardTitle class="text-base flex items-center gap-2">
                <Workflow class="w-4 h-4 text-primary" />
                <span>核心调度机制与调优</span>
              </CardTitle>
              <CardDescription class="text-xs">Go 协程调度模型、内存队列与平滑防抖运转机制</CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
          <div class="space-y-3">
            <!-- 机制 1：并发池 -->
            <div class="p-3 rounded-xl border border-border/70 bg-muted/20 flex items-start gap-2.5">
              <div class="p-1.5 rounded-lg bg-primary/10 text-primary shrink-0 mt-0.5">
                <Cpu class="w-3.5 h-3.5" />
              </div>
              <div class="space-y-0.5">
                <div class="text-xs font-bold text-foreground">并发池 (Workers Pool)</div>
                <p class="text-[11px] text-muted-foreground leading-relaxed">
                  控制系统全局并行运行的任务进程上限。建议按服务器 CPU 物理核心数 1~2 倍进行配置，防止瞬时过多子进程打满物理内存。
                </p>
              </div>
            </div>

            <!-- 机制 2：缓冲队列 -->
            <div class="p-3 rounded-xl border border-border/70 bg-muted/20 flex items-start gap-2.5">
              <div class="p-1.5 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400 shrink-0 mt-0.5">
                <Layers class="w-3.5 h-3.5" />
              </div>
              <div class="space-y-0.5">
                <div class="text-xs font-bold text-foreground">FIFO 内存缓冲队列 (Queue)</div>
                <p class="text-[11px] text-muted-foreground leading-relaxed">
                  当同时触发的任务数超出 Workers 容量时，任务将按时间戳有序排入先进先出队列，前序任务释放后毫秒级唤醒消费。
                </p>
              </div>
            </div>

            <!-- 机制 3：频控防抖 -->
            <div class="p-3 rounded-xl border border-border/70 bg-muted/20 flex items-start gap-2.5">
              <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5">
                <Activity class="w-3.5 h-3.5" />
              </div>
              <div class="space-y-0.5">
                <div class="text-xs font-bold text-foreground">频控防抖 (Rate Limiting)</div>
                <p class="text-[11px] text-muted-foreground leading-relaxed">
                  设置每次子进程启动之间的物理间隔时间，防止整点秒级（如 00:00:00）多任务并发打桩引发的系统短时冷启动雪崩。
                </p>
              </div>
            </div>

            <!-- 规格配置速查矩阵 -->
            <div class="space-y-1.5 pt-1">
              <div class="flex items-center justify-between text-[11px] font-medium text-foreground">
                <span class="flex items-center gap-1.5">
                  <Sparkles class="w-3.5 h-3.5 text-primary" />
                  <span>硬件推荐方案（点击可一键套用）</span>
                </span>
              </div>
              <div class="grid grid-cols-3 gap-2">
                <!-- 基础轻量型 -->
                <div
                  @click="applyPreset('基础轻量型', '6', '100', '150')"
                  class="p-2 rounded-lg border border-border/60 bg-muted/20 hover:border-primary/40 hover:bg-muted/30 transition-all text-center cursor-pointer group"
                  title="点击套用基础轻量型预设 (6 Workers)"
                >
                  <div class="text-[10px] text-muted-foreground group-hover:text-foreground">轻量 1C/1G-2G</div>
                  <div class="text-xs font-bold text-foreground font-inter pt-0.5">4 ~ 8 Workers</div>
                  <div class="text-[9px] text-muted-foreground">队列 100 · 150ms</div>
                </div>

                <!-- 主流云主机推荐型 -->
                <div
                  @click="applyPreset('主流推荐型', '12', '200', '100')"
                  class="p-2 rounded-lg border border-primary/30 bg-primary/5 hover:border-primary/60 hover:bg-primary/10 transition-all text-center cursor-pointer group ring-1 ring-primary/20"
                  title="点击套用主流推荐型预设 (12 Workers)"
                >
                  <div class="text-[10px] text-primary font-medium">推荐型 2C/2G-4G</div>
                  <div class="text-xs font-bold text-foreground font-inter pt-0.5">8 ~ 16 Workers</div>
                  <div class="text-[9px] text-muted-foreground">队列 200 · 100ms</div>
                </div>

                <!-- 进阶多任务高并发型 -->
                <div
                  @click="applyPreset('高并发旗舰型', '24', '500', '50')"
                  class="p-2 rounded-lg border border-border/60 bg-muted/20 hover:border-primary/40 hover:bg-muted/30 transition-all text-center cursor-pointer group"
                  title="点击套用高并发旗舰型预设 (24 Workers)"
                >
                  <div class="text-[10px] text-muted-foreground group-hover:text-foreground">旗舰 4C+/4G+</div>
                  <div class="text-xs font-bold text-foreground font-inter pt-0.5">16 ~ 32 Workers</div>
                  <div class="text-[9px] text-muted-foreground">队列 500 · 50ms</div>
                </div>
              </div>
            </div>
          </div>

          <!-- 右卡片底栏补充说明 -->
          <div class="p-3 rounded-xl bg-muted/20 border border-border/60 text-xs text-muted-foreground flex items-center gap-2">
            <CheckCircle2 class="w-4 h-4 text-emerald-500 shrink-0" />
            <span>核心调度器由 Go 原生极轻协程支撑，常驻开销小于 0.1% CPU。</span>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- 确认保存弹窗 -->
    <AlertDialog :open="showConfirm" @update:open="showConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="flex items-center gap-2">
            <Sliders class="w-5 h-5 text-primary" />
            <span>确认更新调度引擎参数？</span>
          </AlertDialogTitle>
          <AlertDialogDescription class="text-xs leading-relaxed pt-1">
            更新后新的并发数与队列规则将立即生效并重新编排后续触发的任务，当前正在执行中的任务不受影响。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter class="gap-2 sm:gap-0">
          <AlertDialogCancel class="text-xs h-8">取消</AlertDialogCancel>
          <AlertDialogAction @click="saveSettings" class="text-xs h-8">
            确认应用配置
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<style scoped>
.font-inter {
  font-family: 'Inter Variable', 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif !important;
  font-feature-settings: "cv05", "cv08", "cv11", "ss01", "ss03", "tnum" !important;
}

/* 隐藏原生数字输入框箭头 */
.no-spin::-webkit-inner-spin-button,
.no-spin::-webkit-outer-spin-button {
  -webkit-appearance: none !important;
  margin: 0 !important;
  background: none !important;
  opacity: 0 !important;
  display: none !important;
}
.no-spin {
  -moz-appearance: textfield !important;
  appearance: textfield !important;
}
</style>
