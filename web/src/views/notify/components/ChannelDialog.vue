<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Search,
  ArrowLeft,
  Building2,
  AppWindow,
  MessageSquare,
  Send,
  MessageCircle,
  Zap,
  Smartphone,
  PlusCircle,
  BellRing,
  QrCode,
  Radio,
  Layers,
  Mail,
  PhoneCall,
  Webhook,
  Bell,
  X
} from 'lucide-vue-next'
import type { NotifyChannel, ChannelType, SelectItemProps } from '@/api'

const props = defineProps<{
  open: boolean
  isEditing: boolean
  channel: Partial<NotifyChannel>
  channelTypes: ChannelType[]
  configFields: Record<string, { key: string; label: string; required: boolean; placeholder?: string; type?: string; items?: Array<SelectItemProps>; defaultValue?: string }[]>
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:channel': [channel: Partial<NotifyChannel>]
  'type-change': [type: string]
  save: []
}>()

// 1: 选择渠道类型, 2: 配置参数
const currentStep = ref<number>(props.isEditing ? 2 : 1)

// 渠道元数据配置
interface ChannelMeta {
  category: 'im' | 'push' | 'basic'
  desc: string
  icon: any
  iconColor: string
  bgColor: string
}

const CHANNEL_METAS: Record<string, ChannelMeta> = {
  // 办公协作
  QyWeiXin: { category: 'im', desc: '群机器人 Webhook', icon: Building2, iconColor: 'text-emerald-500', bgColor: 'bg-emerald-500/10' },
  QyWeiXinApp: { category: 'im', desc: '应用消息直发', icon: AppWindow, iconColor: 'text-teal-500', bgColor: 'bg-teal-500/10' },
  Dtalk: { category: 'im', desc: '钉钉群机器人', icon: MessageSquare, iconColor: 'text-blue-500', bgColor: 'bg-blue-500/10' },
  Feishu: { category: 'im', desc: '飞书群机器人', icon: Send, iconColor: 'text-cyan-500', bgColor: 'bg-cyan-500/10' },
  VoceChat: { category: 'im', desc: '轻量自建聊天', icon: MessageCircle, iconColor: 'text-indigo-500', bgColor: 'bg-indigo-500/10' },

  // 移动 / 多端推送
  ServerChan: { category: 'push', desc: '微信/多端推送', icon: Zap, iconColor: 'text-amber-500', bgColor: 'bg-amber-500/10' },
  Bark: { category: 'push', desc: 'iOS 极速通知', icon: Smartphone, iconColor: 'text-sky-500', bgColor: 'bg-sky-500/10' },
  PushPlus: { category: 'push', desc: '微信公众号推送', icon: PlusCircle, iconColor: 'text-emerald-600', bgColor: 'bg-emerald-600/10' },
  PushMe: { category: 'push', desc: '轻量多端推送', icon: BellRing, iconColor: 'text-yellow-500', bgColor: 'bg-yellow-500/10' },
  WxPusher: { category: 'push', desc: '微信模板消息', icon: QrCode, iconColor: 'text-green-600', bgColor: 'bg-green-600/10' },
  Telegram: { category: 'push', desc: 'TG 机器人', icon: Send, iconColor: 'text-blue-400', bgColor: 'bg-blue-400/10' },
  Ntfy: { category: 'push', desc: 'Pub/Sub 极简推送', icon: Radio, iconColor: 'text-rose-500', bgColor: 'bg-rose-500/10' },
  Gotify: { category: 'push', desc: '私有消息推送', icon: Layers, iconColor: 'text-blue-500', bgColor: 'bg-blue-500/10' },

  // 基础与自建
  Email: { category: 'basic', desc: 'SMTP 邮件', icon: Mail, iconColor: 'text-red-500', bgColor: 'bg-red-500/10' },
  AliyunSMS: { category: 'basic', desc: '短信服务 SMS', icon: PhoneCall, iconColor: 'text-orange-500', bgColor: 'bg-orange-500/10' },
  Custom: { category: 'basic', desc: '自定义 Webhook', icon: Webhook, iconColor: 'text-purple-500', bgColor: 'bg-purple-500/10' },
}

const searchQuery = ref('')
const selectedCategory = ref<'all' | 'im' | 'push' | 'basic'>('all')

const categoryTabs = [
  { key: 'all', label: '全部' },
  { key: 'im', label: '办公协作' },
  { key: 'push', label: '移动推送' },
  { key: 'basic', label: '基础服务' },
] as const

function getCategoryName(cat?: string): string {
  switch (cat) {
    case 'im': return '办公协作'
    case 'push': return '移动推送'
    case 'basic': return '基础服务'
    default: return '消息通知'
  }
}

// 弹窗状态重置
watch(() => props.open, (val) => {
  if (val) {
    currentStep.value = props.isEditing ? 2 : 1
    searchQuery.value = ''
    selectedCategory.value = 'all'
  }
})

// 合并后端返回的 channelTypes 与前端元数据
const enrichedChannelTypes = computed(() => {
  return props.channelTypes.map(ct => {
    const meta = CHANNEL_METAS[ct.type] || {
      category: 'basic' as const,
      desc: '消息通道',
      icon: Bell,
      iconColor: 'text-muted-foreground',
      bgColor: 'bg-muted',
    }
    return {
      type: ct.type,
      label: ct.label,
      ...meta,
    }
  })
})

// 根据搜索与分类筛选出来的渠道列表
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return enrichedChannelTypes.value.filter(item => {
    if (selectedCategory.value !== 'all' && item.category !== selectedCategory.value) {
      return false
    }
    if (!q) return true
    return (
      item.label.toLowerCase().includes(q) ||
      item.type.toLowerCase().includes(q) ||
      item.desc.toLowerCase().includes(q)
    )
  })
})

// 当前选中的渠道元数据
const currentSelectedMeta = computed(() => {
  if (!props.channel.type) return null
  return enrichedChannelTypes.value.find(item => item.type === props.channel.type) || null
})

function handleSelectType(type: string, label: string) {
  emit('type-change', type)
  // 如果名称为空，自动填充默认渠道名
  if (!props.channel.name) {
    updateChannelField('name', label)
  }
  currentStep.value = 2
}

const currentConfigFields = computed(() => {
  return props.configFields[props.channel.type || ''] || []
})

const enabledModel = computed({
  get: () => props.channel.enabled ?? true,
  set: (value) => updateChannelField('enabled', value)
})

function updateChannelField(field: string, value: any) {
  emit('update:channel', { ...props.channel, [field]: value })
}

function updateConfigField(key: string, value: string) {
  const newConfig = { ...props.channel.config, [key]: value }
  emit('update:channel', { ...props.channel, config: newConfig })
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-xl max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>
          {{ isEditing ? '编辑渠道' : (currentStep === 1 ? '选择通知渠道' : '配置通知渠道') }}
        </DialogTitle>
        <DialogDescription>
          {{ currentStep === 1 ? '点击选择您需要接入的消息推送服务' : '配置推送参数并绑定消息通知' }}
        </DialogDescription>
      </DialogHeader>

      <!-- 步骤 1：卡片网格选择渠道类型 -->
      <div v-if="currentStep === 1 && !isEditing" class="space-y-3 py-1">
        <!-- 搜索与分类栏 -->
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
          <!-- 搜索框 -->
          <div class="relative flex-1">
            <Search class="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              v-model="searchQuery"
              placeholder="搜索渠道，如 微信、钉钉、Server酱..."
              class="h-8.5 pl-8 pr-7 text-xs bg-muted/30 focus-visible:ring-1"
            />
            <button
              v-if="searchQuery"
              type="button"
              @click="searchQuery = ''"
              class="absolute right-2.5 top-2.5 text-muted-foreground hover:text-foreground"
            >
              <X class="w-3.5 h-3.5" />
            </button>
          </div>

          <!-- 分类 Tabs 药丸 -->
          <div class="flex items-center gap-1 p-0.5 bg-muted/50 rounded-lg shrink-0">
            <button
              v-for="tab in categoryTabs"
              :key="tab.key"
              type="button"
              @click="selectedCategory = tab.key"
              class="py-1 px-2 text-[11px] font-medium rounded-md transition-all text-center"
              :class="selectedCategory === tab.key 
                ? 'bg-background text-foreground shadow-xs' 
                : 'text-muted-foreground hover:text-foreground hover:bg-background/40'"
            >
              {{ tab.label }}
            </button>
          </div>
        </div>

        <!-- 渠道网格卡片列表 -->
        <div class="pt-1">
          <div v-if="filteredChannels.length === 0" class="py-12 text-center text-xs text-muted-foreground">
            未找到与 "{{ searchQuery }}" 相关的通知渠道
          </div>
          <div v-else class="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
            <div
              v-for="item in filteredChannels"
              :key="item.type"
              @click="handleSelectType(item.type, item.label)"
              class="flex items-center gap-2.5 p-2.5 rounded-xl border border-border/50 bg-card/60 hover:bg-accent/70 hover:border-primary/50 hover:shadow-sm cursor-pointer transition-all duration-200 group text-left relative overflow-hidden"
            >
              <div :class="['w-8.5 h-8.5 rounded-lg flex items-center justify-center shrink-0 transition-transform duration-200 group-hover:scale-110 shadow-2xs', item.bgColor, item.iconColor]">
                <component :is="item.icon" class="w-4.5 h-4.5" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="text-xs font-semibold text-foreground truncate group-hover:text-primary transition-colors">
                  {{ item.label }}
                </div>
                <div class="text-[10px] text-muted-foreground truncate mt-0.5">
                  {{ item.desc }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 步骤 2：配置渠道参数 -->
      <div v-else class="space-y-4 py-1">
        <!-- 顶部选中的渠道卡片横幅 -->
        <div class="flex items-center justify-between p-3 rounded-xl bg-muted/40 border border-border/50">
          <div class="flex items-center gap-3">
            <div :class="['w-9 h-9 rounded-lg flex items-center justify-center shrink-0', currentSelectedMeta?.bgColor, currentSelectedMeta?.iconColor]">
              <component :is="currentSelectedMeta?.icon || Bell" class="w-5 h-5" />
            </div>
            <div>
              <div class="flex items-center gap-2">
                <span class="text-sm font-semibold text-foreground">{{ currentSelectedMeta?.label }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                  {{ getCategoryName(currentSelectedMeta?.category) }}
                </span>
              </div>
              <div class="text-xs text-muted-foreground mt-0.5">{{ currentSelectedMeta?.desc }}</div>
            </div>
          </div>
          <Button 
            v-if="!isEditing" 
            variant="ghost" 
            size="sm" 
            @click="currentStep = 1"
            class="text-xs text-muted-foreground hover:text-foreground h-8 px-2.5"
          >
            <ArrowLeft class="w-3.5 h-3.5 mr-1" />
            更换渠道
          </Button>
        </div>

        <!-- 渠道名称 -->
        <div class="grid grid-cols-[90px_1fr] items-center gap-3">
          <Label class="text-right text-sm whitespace-nowrap">名称</Label>
          <Input 
            :model-value="channel.name" 
            @update:model-value="updateChannelField('name', $event)"
            placeholder="例如：日常告警通知" 
          />
        </div>

        <!-- 渠道开关 -->
        <div class="grid grid-cols-[90px_1fr] items-center gap-3">
          <Label class="text-right text-sm whitespace-nowrap">启用</Label>
          <Switch v-model="enabledModel" />
        </div>

        <!-- 渠道参数配置项 -->
        <div v-if="currentConfigFields.length > 0" class="border-t pt-4 mt-4">
          <div class="flex items-center justify-between mb-3">
            <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              参数配置
            </h4>
          </div>
          <div class="space-y-3">
            <div 
              v-for="field in currentConfigFields" 
              :key="field.key" 
              class="grid grid-cols-[90px_1fr] gap-3" 
              :class="field.type === 'textarea' ? 'items-start' : 'items-center'"
            >
              <Label class="text-right text-sm whitespace-nowrap" :class="field.type === 'textarea' ? 'pt-2' : ''">
                {{ field.label }}<span v-if="field.required" class="text-destructive ml-1">*</span>
              </Label>
              <div>
                <Input
                  v-if="!field.type"
                  :model-value="channel.config?.[field.key] || ''"
                  @update:model-value="updateConfigField(field.key, String($event))"
                  :placeholder="field.placeholder || ''"
                  class="text-sm"
                />
                <div v-else-if="field.type === 'note'" class="text-xs text-muted-foreground py-1">
                  {{ field.placeholder || '' }}
                </div>
                <textarea
                  v-else-if="field.type === 'textarea'"
                  :value="channel.config?.[field.key] || ''"
                  @input="(e: Event) => updateConfigField(field.key, (e.target as HTMLTextAreaElement).value)"
                  :placeholder="field.placeholder || ''"
                  class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring min-h-[80px]"
                />
                <Switch
                  v-else-if="field.type === 'switch'"
                  :model-value="channel.config?.[field.key] === 'true'"
                  @update:model-value="updateConfigField(field.key, String($event))"
                />
                <Select
                  v-else-if="field.type === 'select'"
                  :model-value="channel.config?.[field.key] || field.defaultValue"
                  @update:model-value="updateConfigField(field.key, String($event))"
                >
                  <SelectTrigger>
                    <SelectValue :placeholder="field.placeholder" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="ct in field.items" :key="ct.value" :value="ct.value">
                      {{ ct.label }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <template v-if="currentStep === 1 && !isEditing">
          <Button variant="outline" @click="emit('update:open', false)">取消</Button>
        </template>
        <template v-else>
          <Button v-if="!isEditing" variant="ghost" @click="currentStep = 1">
            <ArrowLeft class="w-4 h-4 mr-1" />
            上一步
          </Button>
          <Button variant="outline" @click="emit('update:open', false)">取消</Button>
          <Button @click="emit('save')">保存</Button>
        </template>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
