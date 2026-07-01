<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Network } from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'

const emit = defineEmits<{
  (e: 'select', role: 'master' | 'child'): void
}>()

const demoMode = ref(false)

onMounted(async () => {
  try {
    const publicSite = await api.settings.getPublicSite()
    demoMode.value = publicSite.demo_mode || false
  } catch (e) {
    // ignore
  }
})

function handleSelect(role: 'master' | 'child') {
  if (demoMode.value) {
    toast.error('演示模式下禁止修改互联角色')
    return
  }
  emit('select', role)
}
</script>

<template>
  <div class="flex flex-col items-center justify-center py-6 gap-4">
    <div class="text-center space-y-1.5">
      <div class="inline-flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary mb-1">
        <Network class="h-4 w-4" />
      </div>
      <h2 class="text-lg font-bold tracking-tight">选择面板的互联角色</h2>
      <p class="text-muted-foreground text-xs max-w-md mx-auto">
        请根据集群架构分配互斥角色，避免循环嵌套。
      </p>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 w-full max-w-3xl px-4 mt-2">
      <!-- Master Card -->
      <div class="border rounded-xl p-5 hover:border-amber-500 hover:ring-1 hover:ring-amber-500/30 cursor-pointer transition-all hover:shadow-sm bg-card space-y-2.5 group" @click="handleSelect('master')">
        <div class="h-10 w-10 rounded-lg bg-amber-500/10 flex items-center justify-center text-amber-500 group-hover:scale-110 transition-transform duration-300">
          <Network class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold">我是主节点 (Master)</h3>
        <p class="text-xs text-muted-foreground leading-relaxed">
          集中监控其他面板的状态，并可无缝穿越到子节点进行管理。<strong class="font-medium text-foreground">节点分为子面板与 Agent，主节点作为核心控制端，负责对子面板与 Agent 进行集中管理与任务调度。</strong>即使子节点处于无公网 IP 的深层内网，主节点依然可以通过反向隧道直连穿透。
        </p>
      </div>
      <!-- Child Card -->
      <div class="border rounded-xl p-5 hover:border-green-500 hover:ring-1 hover:ring-green-500/30 cursor-pointer transition-all hover:shadow-sm bg-card space-y-2.5 group" @click="handleSelect('child')">
        <div class="h-10 w-10 rounded-lg bg-green-500/10 flex items-center justify-center text-green-500 group-hover:scale-110 transition-transform duration-300">
          <Network class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold">我是子节点 (Child)</h3>
        <p class="text-xs text-muted-foreground leading-relaxed">
          向主节点报告运行状态，并允许主节点穿越到本面板进行管理。<strong class="font-medium text-foreground">互联遵循单机选主与角色互斥原则，子面板不能配置作为 Agent 的控制端。</strong>所有 Agent 节点需统一连接至主面板。非常适合部署在无公网 IP 的受控环境中。
        </p>
      </div>
    </div>
  </div>
</template>
