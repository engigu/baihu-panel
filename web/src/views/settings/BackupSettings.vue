<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
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
  Download,
  UploadCloud,
  Archive,
  TriangleAlert,
  ShieldAlert,
  Clock,
  FileArchive,
  Info
} from 'lucide-vue-next'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { formatDateTime } from '@/utils/date'

const { siteSettings, loadSettings } = useSiteSettings()
const isDemoMode = computed(() => siteSettings.value.demo_mode === 'true' || (siteSettings.value as any).demo_mode === true)

const hasBackup = ref(false)
const backupTime = ref('')
const backupLoading = ref(false)
const restoreLoading = ref(false)
const fileInput = ref<HTMLInputElement>()
const showConfirm = ref(false)

const backupItems = [
  '定时任务编排',
  '环境变量与密钥',
  '历史执行日志',
  '代码仓库与脚本',
  '核心系统设置'
]

async function checkBackupStatus() {
  try {
    const res = await api.settings.getBackupStatus()
    hasBackup.value = res.has_backup
    backupTime.value = res.backup_time || ''
  } catch {}
}

async function createBackup() {
  if (isDemoMode.value) {
    toast.error('演示模式下禁止创建备份')
    return
  }
  backupLoading.value = true
  try {
    await api.settings.createBackup()
    toast.success('备份创建成功')
    await checkBackupStatus()
  } catch (e: any) {
    toast.error(e.message || '备份失败')
  } finally {
    backupLoading.value = false
  }
}

function downloadBackup() {
  if (isDemoMode.value) {
    toast.error('演示模式下禁止下载备份')
    return
  }
  window.open(api.settings.downloadBackup(), '_blank')
  setTimeout(checkBackupStatus, 6000)
}

function showRestoreConfirm() {
  if (isDemoMode.value) {
    toast.error('演示模式下禁止恢复备份数据')
    return
  }
  showConfirm.value = true
}

function confirmRestore() {
  showConfirm.value = false
  fileInput.value?.click()
}

function cancelRestore() {
  showConfirm.value = false
}

async function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  if (!file.name.endsWith('.zip')) {
    toast.error('请选择 .zip 格式备份文件')
    target.value = ''
    return
  }

  restoreLoading.value = true
  try {
    await api.settings.restoreBackup(file)
    toast.success('恢复成功，页面即将刷新')
    setTimeout(() => window.location.reload(), 1500)
  } catch (e: any) {
    toast.error(e.message || '恢复失败')
  } finally {
    restoreLoading.value = false
    target.value = ''
  }
}

onMounted(async () => {
  await loadSettings()
  await checkBackupStatus()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 左右双列平衡卡片 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
      <!-- 左列：数据全量备份 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-3 pt-0 px-5">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <CardTitle class="text-base flex items-center gap-2">
                  <Archive class="w-4 h-4 text-primary" />
                  <span>全量数据备份</span>
                </CardTitle>
                <Badge
                  v-if="isDemoMode"
                  variant="destructive"
                  class="text-[10px] px-1.5 py-0 font-normal"
                >
                  演示模式禁用
                </Badge>
              </div>
              <CardDescription class="text-xs leading-relaxed pt-0.5">
                一键打包归档系统所有配置与任务，生成便携式快照压缩包。
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
          <div class="space-y-3.5">
            <!-- 备份包含范围 -->
            <div class="space-y-1.5">
              <div class="text-xs font-medium text-foreground flex items-center gap-1.5">
                <FileArchive class="w-3.5 h-3.5 text-muted-foreground" />
                <span>归档数据范畴</span>
              </div>
              <div class="flex flex-wrap gap-1.5">
                <Badge
                  v-for="item in backupItems"
                  :key="item"
                  class="text-xs bg-muted/60 text-foreground/80 border-0 font-normal px-2.5 py-0.5"
                >
                  {{ item }}
                </Badge>
              </div>
            </div>

            <!-- 当前备份状态卡片 -->
            <div
              v-if="!isDemoMode"
              class="p-3.5 rounded-xl border border-border/70 bg-muted/20 flex flex-col gap-1.5"
            >
              <div class="flex items-center justify-between text-xs">
                <span class="text-muted-foreground font-medium flex items-center gap-1.5">
                  <Clock class="w-3.5 h-3.5 text-muted-foreground/70" />
                  <span>快照状态</span>
                </span>
                <span v-if="hasBackup" class="flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-medium">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                  备份就绪
                </span>
                <span v-else class="text-[11px] text-muted-foreground">
                  尚未生成快照
                </span>
              </div>
              
              <div class="flex items-baseline justify-between pt-0.5">
                <span class="text-xs text-muted-foreground">生成时间戳</span>
                <span
                  class="text-xs font-medium font-inter text-foreground/90 truncate"
                  :title="backupTime || ''"
                >
                  {{ hasBackup && backupTime ? formatDateTime(backupTime) : '暂无备份记录' }}
                </span>
              </div>
            </div>

            <!-- 策略提示说明 -->
            <div class="p-3 rounded-xl bg-muted/30 border border-border/60 text-xs text-muted-foreground leading-relaxed flex items-start gap-2">
              <Info class="w-4 h-4 text-blue-500 shrink-0 mt-0.5" />
              <span>
                <span class="font-medium text-foreground">生命周期策略：</span>
                为保障存储安全，备份文件在首次下载 5 分钟后将被系统自动物理清理以释放空间。
              </span>
            </div>
          </div>

          <!-- 底部操作按钮栏 -->
          <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
            <span class="text-[11px] text-muted-foreground">
              打包归档当前系统完整配置与任务快照
            </span>
            <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 w-full sm:w-auto">
              <Button
                @click="createBackup"
                :disabled="backupLoading || isDemoMode"
                class="h-8.5 px-4 text-xs font-medium shadow-sm gap-1.5 w-full sm:w-auto justify-center"
                :title="isDemoMode ? '演示模式下禁止创建备份' : '创建备份'"
              >
                <Archive class="w-3.5 h-3.5" />
                <span>{{ backupLoading ? '备份生成中...' : '创建新备份' }}</span>
              </Button>

              <Button
                v-if="hasBackup && !isDemoMode"
                @click="downloadBackup"
                variant="outline"
                class="h-8.5 px-3.5 text-xs font-medium shadow-sm gap-1.5 hover:bg-accent w-full sm:w-auto justify-center"
              >
                <Download class="w-3.5 h-3.5" />
                <span>下载备份包 (.zip)</span>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- 右列：数据快照恢复 -->
      <Card class="shadow-sm flex flex-col justify-between pt-4.5 pb-5">
        <CardHeader class="pb-3 pt-0 px-5">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <CardTitle class="text-base flex items-center gap-2">
                  <UploadCloud class="w-4 h-4 text-primary" />
                  <span>系统数据恢复</span>
                </CardTitle>
                <Badge
                  v-if="isDemoMode"
                  variant="destructive"
                  class="text-[10px] px-1.5 py-0 font-normal"
                >
                  演示模式禁用
                </Badge>
              </div>
              <CardDescription class="text-xs leading-relaxed pt-0.5">
                从先前导出的 .zip 格式快照压缩包全面还原系统数据与环境。
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent class="flex-1 flex flex-col justify-between px-5 pb-0 pt-0 space-y-4">
          <div class="space-y-3.5">
            <!-- 高危操作警告提示框 -->
            <div class="p-3.5 rounded-xl border border-destructive/25 bg-destructive/5 space-y-1.5">
              <div class="text-xs font-semibold text-destructive flex items-center gap-1.5">
                <TriangleAlert class="h-3.5 w-3.5 shrink-0" />
                <span>不可逆操作警示</span>
              </div>
              <p class="text-xs text-muted-foreground leading-relaxed">
                恢复操作将以归档包内的快照为准，<strong class="text-destructive font-medium">全量覆写当前系统的所有数据库记录、任务脚本与环境变量</strong>。恢复成功后面板主进程将自动重载。
              </p>
            </div>

            <!-- 上传拖拽交互热区 -->
            <div
              @click="showRestoreConfirm"
              :class="[
                'border-2 border-dashed rounded-xl p-5 flex flex-col items-center justify-center text-center transition-all group',
                isDemoMode || restoreLoading
                  ? 'border-border/40 bg-muted/10 opacity-60 cursor-not-allowed'
                  : 'border-border/80 hover:border-primary/50 bg-muted/15 hover:bg-muted/30 cursor-pointer'
              ]"
            >
              <div class="w-10 h-10 rounded-full bg-primary/10 text-primary flex items-center justify-center mb-2 group-hover:scale-105 transition-transform">
                <UploadCloud class="w-5 h-5" />
              </div>
              <div class="text-xs font-medium text-foreground">
                {{ restoreLoading ? '正在解压并恢复系统数据...' : '点击上传备份文件 (.zip)' }}
              </div>
              <div class="text-[11px] text-muted-foreground pt-1">
                支持标准白虎面板导出的快照归档包
              </div>
            </div>
          </div>

          <!-- 隐藏文件输入框与动作触发区 -->
          <input
            ref="fileInput"
            type="file"
            accept=".zip"
            class="hidden"
            @change="handleFileSelect"
          />

          <!-- 底部操作按钮栏 -->
          <div class="pt-3 border-t border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-1">
            <span class="text-[11px] text-muted-foreground">
              导入并覆写全量快照归档数据
            </span>
            <Button
              @click="showRestoreConfirm"
              :disabled="restoreLoading || isDemoMode"
              variant="outline"
              class="h-8.5 px-4 text-xs font-medium shadow-sm w-full sm:w-auto gap-1.5 justify-center"
              :title="isDemoMode ? '演示模式下禁止恢复' : '选择备份文件并恢复'"
            >
              <UploadCloud class="w-3.5 h-3.5" />
              <span>{{ restoreLoading ? '系统恢复中...' : '选择文件并恢复...' }}</span>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- 危险操作确认对话框 -->
    <AlertDialog :open="showConfirm" @update:open="showConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="flex items-center gap-2 text-destructive">
            <ShieldAlert class="w-5 h-5 text-destructive" />
            <span>确认执行系统快照恢复？</span>
          </AlertDialogTitle>
          <AlertDialogDescription class="text-xs leading-relaxed pt-1 space-y-2">
            <p>恢复备份将全面覆盖当前数据库及所有受控脚本文件，已有未备份的修改将彻底丢失且无法撤销。</p>
            <p class="font-medium text-foreground">建议在恢复前先点击左侧「创建新备份」保留当前状态。</p>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel @click="cancelRestore" class="text-xs h-8">取消</AlertDialogCancel>
          <AlertDialogAction
            @click="confirmRestore"
            class="text-xs h-8 bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            我已知晓风险，继续恢复
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
</style>
