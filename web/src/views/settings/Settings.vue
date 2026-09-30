<script setup lang="ts">
import { ref } from 'vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import PasswordSettings from './PasswordSettings.vue'
import OtpSettings from './OtpSettings.vue'
import SiteSettings from './SiteSettings.vue'
import SchedulerSettings from './SchedulerSettings.vue'
import BackupSettings from './BackupSettings.vue'
import AboutSettings from './AboutSettings.vue'
import WebUISettings from './WebUISettings.vue'

const activeTab = ref('security')
</script>

<template>
  <div class="space-y-6 max-w-5xl xl:max-w-6xl mx-auto w-full">
    <div>
      <h2 class="text-xl sm:text-2xl font-bold tracking-tight">系统设置</h2>
      <p class="text-muted-foreground text-sm">管理系统配置和账户安全</p>
    </div>

    <Tabs v-model="activeTab" class="w-full">
      <TabsList class="w-full sm:w-auto inline-grid grid-cols-3 sm:grid-cols-6 gap-y-1 sm:gap-y-0 h-auto p-1 bg-muted/50 rounded-lg">
        <TabsTrigger value="security" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">安全设置</TabsTrigger>
        <TabsTrigger value="site" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">站点设置</TabsTrigger>
        <TabsTrigger value="webui" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">前端定制</TabsTrigger>
        <TabsTrigger value="scheduler" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">调度设置</TabsTrigger>
        <TabsTrigger value="backup" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">备份恢复</TabsTrigger>
        <TabsTrigger value="about" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">关于</TabsTrigger>
      </TabsList>

      <!-- 安全设置：双列卡片并排 -->
      <TabsContent value="security" class="mt-6">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
          <PasswordSettings />
          <OtpSettings />
        </div>
      </TabsContent>

      <TabsContent value="site" class="mt-6">
        <SiteSettings />
      </TabsContent>

      <TabsContent value="webui" class="mt-6">
        <WebUISettings />
      </TabsContent>

      <TabsContent value="scheduler" class="mt-6">
        <SchedulerSettings />
      </TabsContent>

      <TabsContent value="backup" class="mt-6">
        <BackupSettings />
      </TabsContent>

      <TabsContent value="about" class="mt-6">
        <AboutSettings />
      </TabsContent>
    </Tabs>
  </div>
</template>
