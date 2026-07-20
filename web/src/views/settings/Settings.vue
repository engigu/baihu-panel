<script setup lang="ts">
import { ref } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { UploadCloud, ExternalLink } from 'lucide-vue-next'
import PasswordSettings from './PasswordSettings.vue'
import SiteSettings from './SiteSettings.vue'
import SchedulerSettings from './SchedulerSettings.vue'
import BackupSettings from './BackupSettings.vue'
import AboutSettings from './AboutSettings.vue'
import WebUISettings from './WebUISettings.vue'
import Sandboxes from '../sandboxes/Sandboxes.vue'

const activeTab = ref('password')
const webuiRef = ref<any>(null)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-xl sm:text-2xl font-bold tracking-tight">系统设置</h2>
      <p class="text-muted-foreground text-sm">管理系统配置和账户安全</p>
    </div>

    <Tabs v-model="activeTab" class="max-w-2xl">
      <TabsList class="w-full grid grid-cols-3 sm:grid-cols-7 gap-y-1 sm:gap-y-0 h-auto p-1 bg-muted/50 rounded-lg">
        <TabsTrigger value="password" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">密码修改</TabsTrigger>
        <TabsTrigger value="site" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">站点设置</TabsTrigger>
        <TabsTrigger value="webui" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">前端定制</TabsTrigger>
        <TabsTrigger value="sandbox" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">沙箱管理</TabsTrigger>
        <TabsTrigger value="scheduler" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">调度设置</TabsTrigger>
        <TabsTrigger value="backup" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">备份恢复</TabsTrigger>
        <TabsTrigger value="about" class="text-xs sm:text-sm px-1 sm:px-3 py-1.5 whitespace-nowrap">关于</TabsTrigger>
      </TabsList>

      <TabsContent value="password" class="mt-6">
        <Card>
          <CardHeader>
            <CardTitle>修改密码</CardTitle>
            <CardDescription>更新您的账户密码</CardDescription>
          </CardHeader>
          <CardContent>
            <PasswordSettings />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="site" class="mt-6">
        <Card>
          <CardHeader>
            <CardTitle>站点设置</CardTitle>
            <CardDescription>配置站点标题、图标和系统常规参数</CardDescription>
          </CardHeader>
          <CardContent>
            <SiteSettings />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="webui" class="mt-6">
        <Card>
          <CardHeader class="flex flex-row items-start justify-between space-y-0 gap-4">
            <div class="space-y-1.5 flex-1 min-w-0">
              <CardTitle class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                <span>自定义前端包 (WebUI)</span>
                <a href="https://engigu.github.io/baihu-panel/guide/webui" target="_blank" class="flex items-center gap-1 text-xs text-blue-600 hover:underline font-normal shrink-0">
                  开发文档
                  <ExternalLink class="w-3 h-3 shrink-0" />
                </a>
              </CardTitle>
              <CardDescription class="line-clamp-2 sm:line-clamp-none">完全接管和替换默认的系统面板界面，实现深度定制</CardDescription>
            </div>
            <div class="shrink-0 mt-0 sm:mt-1">
              <Button 
                @click="webuiRef?.triggerUpload" 
                :disabled="webuiRef?.uploading" 
                variant="outline"
                size="sm" 
                class="h-8 px-2 sm:px-3"
                title="上传前端资源包"
              >
                <span v-if="webuiRef?.uploading" class="flex items-center justify-center">
                  <svg class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span class="hidden sm:inline ml-2">上传中...</span>
                </span>
                <span v-else class="flex items-center">
                  <UploadCloud class="w-4 h-4" />
                  <span class="hidden sm:inline ml-2">上传资源包</span>
                </span>
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <WebUISettings ref="webuiRef" />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="sandbox" class="mt-6">
        <Card>
          <CardHeader>
            <CardTitle>沙箱管理</CardTitle>
            <CardDescription>配置和分配定时任务脚本专用的轻量沙箱隔离模板</CardDescription>
          </CardHeader>
          <CardContent>
            <Sandboxes />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="scheduler" class="mt-6">
        <Card>
          <CardHeader>
            <CardTitle>调度设置</CardTitle>
            <CardDescription>配置任务调度器参数</CardDescription>
          </CardHeader>
          <CardContent>
            <SchedulerSettings />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="backup" class="mt-6">
        <Card>
          <CardHeader>
            <CardTitle>备份恢复</CardTitle>
            <CardDescription>备份和恢复系统数据</CardDescription>
          </CardHeader>
          <CardContent>
            <BackupSettings />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="about" class="mt-6">
        <Card>
          <CardContent class="pt-6">
            <AboutSettings />
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>
