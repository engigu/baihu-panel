<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { api, type NotifyChannel, type ChannelType, type EventType, type NotifyBinding, type Task } from '@/api'
import { toast } from 'vue-sonner'
import ChannelList from './components/ChannelList.vue'
import EventBinding from './components/EventBinding.vue'
import ApiUsage from './components/ApiUsage.vue'
import ChannelDialog from './components/ChannelDialog.vue'

const activeTab = ref('channels')

// 渠道数据
const channels = ref<NotifyChannel[]>([])
const channelTypes = ref<ChannelType[]>([])
const eventTypes = ref<EventType[]>([])
const loading = ref(false)

// API Token
const apiToken = ref('')

// 编辑弹窗
const showDialog = ref(false)
const editingChannel = ref<Partial<NotifyChannel>>({
  name: '',
  type: '',
  enabled: true,
  config: {}
})
const isEditing = ref(false)

// 删除确认
const showDeleteConfirm = ref(false)
const deletingChannelId = ref('')

// 事件绑定
const bindings = ref<NotifyBinding[]>([])
const allTasks = ref<Task[]>([])

// 渠道配置模板
const channelConfigFields: Record<string, { key: string; label: string; required: boolean; placeholder?: string; type?: string }[]> = {
  Telegram: [
    { key: 'bot_token', label: 'Bot Token', required: true, placeholder: '从 @BotFather 获取' },
    { key: 'chat_id', label: 'Chat ID', required: true, placeholder: '聊天/群组 ID' },
    { key: 'api_host', label: 'API 地址', required: false, placeholder: '自定义 API 地址，留空使用官方' },
    { key: 'proxy_url', label: '代理地址', required: false, placeholder: 'http/https/socks5 代理' },
  ],
  Bark: [
    { key: 'push_key', label: 'Push Key', required: true, placeholder: 'Bark Push Key' },
    { key: 'sound', label: '推送声音', required: false, placeholder: '留空使用默认' },
    { key: 'group', label: '推送分组', required: false },
    { key: 'icon', label: '推送图标', required: false, placeholder: '图标 URL' },
    { key: 'level', label: '时效性', required: false, placeholder: 'active / timeSensitive / passive' },
    { key: 'url', label: '跳转URL', required: false },
  ],
  Dtalk: [
    { key: 'access_token', label: 'Access Token', required: true, placeholder: '钉钉机器人 access_token' },
    { key: 'secret', label: '加签秘钥', required: false, placeholder: '可选' },
  ],
  QyWeiXin: [
    { key: 'access_token', label: 'Access Token', required: true, placeholder: '企业微信机器人 Key' },
  ],
  Feishu: [
    { key: 'access_token', label: 'Access Token', required: true, placeholder: '飞书机器人 access_token' },
    { key: 'secret', label: '加签秘钥', required: false, placeholder: '可选' },
  ],
  Custom: [
    { key: 'webhook', label: 'Webhook URL', required: true, placeholder: 'https://...' },
    { key: 'headers', label: '请求头', required: false, placeholder: 'JSON格式，如 {"Authorization": "Bearer ..."}', type: 'textarea' },
    { key: 'body', label: '请求体模板', required: false, placeholder: '使用 TEXT 作为消息内容占位符', type: 'textarea' },
  ],
  Ntfy: [
    { key: 'topic', label: 'Topic', required: true },
    { key: 'url', label: 'API 地址', required: false, placeholder: '默认 https://ntfy.sh' },
    { key: 'priority', label: '优先级', required: false, placeholder: '1-5' },
    { key: 'icon', label: '图标 URL', required: false },
    { key: 'token', label: 'Token', required: false },
    { key: 'username', label: '用户名', required: false },
    { key: 'password', label: '密码', required: false },
  ],
  Gotify: [
    { key: 'url', label: '服务地址', required: true, placeholder: 'https://gotify.example.com' },
    { key: 'token', label: 'Token', required: true },
    { key: 'priority', label: '优先级', required: false, placeholder: '0-10' },
  ],
  PushMe: [
    { key: 'push_key', label: 'Push Key', required: true },
    { key: 'url', label: 'API 地址', required: false, placeholder: '默认 https://push.i-i.me' },
    { key: 'type', label: '类型', required: false },
  ],
  Email: [
    { key: 'server', label: 'SMTP 服务器', required: true, placeholder: 'smtp.example.com' },
    { key: 'port', label: '端口', required: true, placeholder: '465' },
    { key: 'account', label: '邮箱账号', required: true },
    { key: 'passwd', label: '邮箱密码', required: true },
    { key: 'from_name', label: '发信人名称', required: false },
    { key: 'to_account', label: '收件邮箱', required: true },
  ],
  AliyunSMS: [
    { key: 'access_key_id', label: 'AccessKeyId', required: true },
    { key: 'access_key_secret', label: 'AccessKeySecret', required: true },
    { key: 'sign_name', label: '短信签名', required: true },
    { key: 'region_id', label: '区域ID', required: false, placeholder: '默认 cn-hangzhou' },
    { key: 'phone_number', label: '手机号码', required: true },
    { key: 'template_code', label: '短信模板 CODE', required: true },
  ],
  PushPlus: [
    { key: 'token', label: 'Token', required: true, placeholder: 'PushPlus Token' },
    { key: 'topic', label: '群组编码', required: false, placeholder: '可选，一对多推送' },
    { key: 'template', label: '推送模板', required: false, placeholder: 'html, txt, json, markdown' },
    { key: 'channel', label: '推送渠道', required: false, placeholder: 'wechat, dingding, feishu, mail等' },
    { key: 'webhook', label: 'Webhook', required: false },
    { key: 'callback_url', label: '回调地址', required: false },
    { key: 'to', label: '好友令牌', required: false },
  ],
}

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const [typesRes, channelsRes, tasksRes] = await Promise.all([
      api.notify.getTypes(),
      api.notify.getChannels(),
      api.tasks.list({ page: 1, page_size: 1000 })
    ])
    channelTypes.value = typesRes.channel_types
    eventTypes.value = typesRes.event_types
    channels.value = channelsRes
    allTasks.value = tasksRes.data
  } catch (e: any) {
    toast.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function loadEvents() {
  try {
    bindings.value = await api.notify.getBindings()
  } catch (e: any) {
    toast.error('加载事件绑定失败: ' + e.message)
  }
}

// 渠道操作
function openNewChannel() {
  editingChannel.value = { name: '', type: '', enabled: true, config: {} }
  isEditing.value = false
  showDialog.value = true
}

function openEditChannel(ch: NotifyChannel) {
  editingChannel.value = { ...ch, config: { ...ch.config } }
  isEditing.value = true
  showDialog.value = true
}

function onTypeChange(val: string) {
  editingChannel.value.type = val
  const existing = editingChannel.value.config || {}
  const fields = channelConfigFields[val] || []
  const newConfig: Record<string, string> = {}
  for (const f of fields) {
    newConfig[f.key] = existing[f.key] || ''
  }
  editingChannel.value.config = newConfig
}

async function saveChannel() {
  if (!editingChannel.value.name || !editingChannel.value.type) {
    toast.error('请填写渠道名称和类型')
    return
  }

  // 确保 enabled 字段有值
  const channelData = {
    ...editingChannel.value,
    enabled: editingChannel.value.enabled ?? true
  }

  try {
    await api.notify.saveChannel(channelData)
    toast.success('保存成功')
    showDialog.value = false
    await loadData()
  } catch (e: any) {
    toast.error('保存失败: ' + e.message)
  }
}

function confirmDelete(id: string) {
  deletingChannelId.value = id
  showDeleteConfirm.value = true
}

async function deleteChannel() {
  showDeleteConfirm.value = false
  try {
    await api.notify.deleteChannel(deletingChannelId.value)
    toast.success('删除成功')
    await loadData()
  } catch (e: any) {
    toast.error('删除失败: ' + e.message)
  }
}

async function testChannel(ch: NotifyChannel) {
  try {
    const result = await api.notify.testChannel(ch)
    if (result.success) {
      toast.success('测试发送成功！')
    } else {
      toast.error('测试发送失败: ' + (result.error || '未知错误'))
    }
  } catch (e: any) {
    toast.error('测试失败: ' + e.message)
  }
}

// 事件绑定
async function saveBindings(newBindings: Partial<NotifyBinding>[]) {
  try {
    for (const binding of newBindings) {
      await api.notify.saveBinding(binding)
    }
    toast.success('绑定保存成功')
    await loadEvents()
  } catch (e: any) {
    toast.error('保存失败: ' + e.message)
  }
}

async function deleteBinding(id: string) {
  try {
    await api.notify.deleteBinding(id)
    toast.success('绑定已删除')
    await loadEvents()
  } catch (e: any) {
    toast.error('删除失败: ' + e.message)
  }
}

// API Token
async function loadApiToken() {
  try {
    const token = await api.settings.get('notify', 'notify_token')
    console.log('Loaded API Token:', token, 'Type:', typeof token, 'Length:', token?.length)
    apiToken.value = token || ''
  } catch (e: any) {
    console.error('加载 API Token 失败:', e)
    // 如果是404或者其他错误，token保持为空字符串
    apiToken.value = ''
  }
}

async function generateApiToken() {
  try {
    const newToken = await api.settings.generateToken('notify', 'notify_token')
    apiToken.value = newToken
    toast.success('API Token 已生成')
  } catch (e: any) {
    toast.error('生成失败: ' + e.message)
  }
}

async function copyApiToken() {
  if (!apiToken.value) {
    toast.error('请先生成 Token')
    return
  }
  try {
    await navigator.clipboard.writeText(apiToken.value)
    toast.success('Token 已复制到剪贴板')
  } catch {
    toast.error('复制失败')
  }
}

async function copyApiExample() {
  if (channels.value.length === 0) {
    toast.error('请先添加一个通知渠道')
    return
  }
  const ch = channels.value[0]
  if (!ch) return
  const token = apiToken.value || 'YOUR_API_TOKEN'
  const example = `curl -X POST "{{API_URL}}/api/v1/notify/send" \\
  -H "Content-Type: application/json" \\
  -H "notify-token: ${token}" \\
  -d '{"channel_id": "${ch.id}", "title": "测试通知", "text": "来自脚本的通知"}'`
  try {
    await navigator.clipboard.writeText(example)
    toast.success('API 调用示例已复制到剪贴板')
  } catch {
    toast.error('复制失败')
  }
}

onMounted(() => {
  loadData()
  loadEvents()
  loadApiToken()
})
</script>

<template>
  <div class="space-y-6">
    <Tabs v-model="activeTab" class="w-full">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h2 class="text-xl sm:text-2xl font-bold tracking-tight">消息推送</h2>
          <p class="text-muted-foreground text-sm">配置通知渠道，绑定系统事件实现自动推送</p>
        </div>
        <TabsList class="grid grid-cols-3 w-full sm:w-auto min-w-[300px]">
          <TabsTrigger value="channels">渠道管理</TabsTrigger>
          <TabsTrigger value="events">事件绑定</TabsTrigger>
          <TabsTrigger value="api">脚本调用</TabsTrigger>
        </TabsList>
      </div>

      <!-- 渠道管理 -->
      <TabsContent value="channels">
        <ChannelList :channels="channels" :channel-types="channelTypes" @add="openNewChannel" @edit="openEditChannel"
          @delete="confirmDelete" @test="testChannel" />
      </TabsContent>

      <!-- 事件绑定 -->
      <TabsContent value="events">
        <EventBinding :channels="channels" :channel-types="channelTypes" :event-types="eventTypes" :bindings="bindings"
          :tasks="allTasks" @save="saveBindings" @delete="deleteBinding" />
      </TabsContent>

      <!-- 脚本调用 -->
      <TabsContent value="api">
        <ApiUsage :channels="channels" :channel-types="channelTypes" :api-token="apiToken"
          @generate-token="generateApiToken" @copy-token="copyApiToken" @copy-example="copyApiExample" />
      </TabsContent>


    </Tabs>

    <!-- 添加/编辑渠道弹窗 -->
    <ChannelDialog v-model:open="showDialog" :is-editing="isEditing" v-model:channel="editingChannel"
      :channel-types="channelTypes" :config-fields="channelConfigFields" @type-change="onTypeChange"
      @save="saveChannel" />

    <!-- 删除确认 -->
    <AlertDialog :open="showDeleteConfirm" @update:open="showDeleteConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>
            删除后将无法恢复，同时会取消该渠道的所有事件绑定。确定要删除吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="deleteChannel">确认删除</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
