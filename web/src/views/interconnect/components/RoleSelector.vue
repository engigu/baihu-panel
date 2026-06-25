<script setup lang="ts">
import { Network } from 'lucide-vue-next'

const emit = defineEmits<{
  (e: 'select', role: 'master' | 'child'): void
}>()
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
      <div class="border rounded-xl p-5 hover:border-amber-500 hover:ring-1 hover:ring-amber-500/30 cursor-pointer transition-all hover:shadow-sm bg-card space-y-2.5 group" @click="emit('select', 'master')">
        <div class="h-10 w-10 rounded-lg bg-amber-500/10 flex items-center justify-center text-amber-500 group-hover:scale-110 transition-transform duration-300">
          <Network class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold">我是主节点 (Master)</h3>
        <p class="text-xs text-muted-foreground leading-relaxed">
          集中监控其他面板的状态，并可无缝穿越到子节点进行管理。<strong class="font-medium text-foreground">即使子节点处于无公网 IP 的深层内网，主节点依然可以通过反向隧道直连穿透。</strong>选择此项后，您将可以生成专属密钥并添加多个子节点。
        </p>
      </div>
      <!-- Child Card -->
      <div class="border rounded-xl p-5 hover:border-green-500 hover:ring-1 hover:ring-green-500/30 cursor-pointer transition-all hover:shadow-sm bg-card space-y-2.5 group" @click="emit('select', 'child')">
        <div class="h-10 w-10 rounded-lg bg-green-500/10 flex items-center justify-center text-green-500 group-hover:scale-110 transition-transform duration-300">
          <Network class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold">我是子节点 (Child)</h3>
        <p class="text-xs text-muted-foreground leading-relaxed">
          向主节点报告运行状态，并允许主节点穿越到本面板进行管理。<strong class="font-medium text-foreground">非常适合部署在家庭宽带或企业内网等无公网 IP 环境中。</strong>选择此项后，本机将主动连接到主节点建立安全的反向穿透隧道。
        </p>
      </div>
    </div>
  </div>
</template>
