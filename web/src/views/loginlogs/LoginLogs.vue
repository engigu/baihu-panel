<script setup lang="ts">
import { ref } from 'vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import LoginLogTab from './tabs/LoginLogTab.vue'
import SystemEventTab from './tabs/SystemEventTab.vue'
import PushLog from '@/views/notify/components/PushLog.vue'

const activeTab = ref('system')
</script>

<template>
  <div class="space-y-6">
    <Tabs v-model="activeTab" class="w-full">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h2 class="text-xl sm:text-2xl font-bold tracking-tight">消息日志</h2>
          <p class="text-muted-foreground text-sm">
            {{ activeTab === 'system' ? '查看系统重要运行事件' :
              activeTab === 'push' ? '查看消息推送历史记录' : '查看系统用户登录记录' }}
          </p>
        </div>
        <TabsList class="grid grid-cols-3 w-full sm:w-auto min-w-[300px]">
          <TabsTrigger value="system">系统事件</TabsTrigger>
          <TabsTrigger value="push">推送日志</TabsTrigger>
          <TabsTrigger value="login">登录日志</TabsTrigger>
        </TabsList>
      </div>

      <TabsContent value="system" class="mt-0">
        <SystemEventTab />
      </TabsContent>

      <TabsContent value="push" class="mt-0">
        <PushLog />
      </TabsContent>

      <TabsContent value="login" class="mt-0">
        <LoginLogTab />
      </TabsContent>
    </Tabs>
  </div>
</template>
