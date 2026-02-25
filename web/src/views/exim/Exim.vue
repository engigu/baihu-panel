<script setup lang="ts">
import { ref } from 'vue'
import { Download, Upload } from 'lucide-vue-next'
import ExportData from './ExportData.vue'
import ImportData from './ImportData.vue'

const activeTab = ref('export')
const exportComp = ref<InstanceType<typeof ExportData> | null>(null)

const handleImported = () => {
    if (exportComp.value) {
        exportComp.value.fetchData()
    }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold tracking-tight">导出与导入</h2>
      <p class="text-muted-foreground mt-2 text-sm">此功能主要用于灵活打包和分享部分数据。<span class="text-muted-foreground/80">如果您需要对整个站点的重要数据进行系统级备份，请前往使用 <router-link to="/settings" class="text-primary hover:underline font-medium">系统设置 - 备份恢复</router-link> 功能。</span></p>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-border/50 text-sm overflow-x-auto scrollbar-hide">
      <button :class="['px-6 py-3 font-semibold border-b-2 flex items-center gap-2', activeTab === 'export' ? 'border-primary text-primary bg-primary/5 shadow-[inset_0_-2px_0_hsl(var(--primary))]' : 'border-transparent text-muted-foreground hover:bg-muted/30 hover:text-foreground']"
        @click="activeTab = 'export'">
        <Upload class="w-4 h-4" /> 数据导出
      </button>
      <button :class="['px-6 py-3 font-semibold border-b-2 flex items-center gap-2', activeTab === 'import' ? 'border-primary text-primary bg-primary/5 shadow-[inset_0_-2px_0_hsl(var(--primary))]' : 'border-transparent text-muted-foreground hover:bg-muted/30 hover:text-foreground']"
        @click="activeTab = 'import'">
        <Download class="w-4 h-4" /> 数据导入
      </button>
    </div>

    <!-- Export Content -->
    <div v-show="activeTab === 'export'">
       <ExportData ref="exportComp" />
    </div>

    <!-- Import Content -->
    <div v-show="activeTab === 'import'">
       <ImportData @imported="handleImported"/>
    </div>
  </div>
</template>

<style scoped>
.animate-bounce-slow {
    animation: bounce 2s infinite;
}
</style>
