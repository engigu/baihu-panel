<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Globe, Check, ArrowRight, Trash2 } from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'

const globalEnvs = ref<Record<string, string>>({})
const loadingEnvs = ref(false)
const showAddEnvDialog = ref(false)
const newEnvKey = ref('')
const newEnvValue = ref('')

const ENV_PRESETS = [
    { label: 'npm (淘宝镜像)', key: 'NPM_CONFIG_REGISTRY', value: 'https://registry.npmmirror.com' },
    { label: 'Python (清华镜像)', key: 'PIP_INDEX_URL', value: 'https://pypi.tuna.tsinghua.edu.cn/simple' },
    { label: 'Go (goproxy.cn)', key: 'GOPROXY', value: 'https://goproxy.cn,direct' },
    { label: 'Rust (Sparse protocol)', key: 'CARGO_REGISTRY_CRATES_IO_PROTOCOL', value: 'sparse' },
    { label: 'Rust (中科大镜像)', key: 'CARGO_REGISTRIES_CRATES_IO_INDEX', value: 'https://mirrors.ustc.edu.cn/crates.io-index' },
]

async function loadGlobalEnvs() {
    loadingEnvs.value = true
    try {
        globalEnvs.value = await api.mise.getEnvs()
    } catch (e) {
        toast.error('获取全局环境变量失败: ' + e)
    } finally {
        loadingEnvs.value = false
    }
}

async function handleSetEnv(key: string, value: string) {
    try {
        await api.mise.setEnv(key, value)
        toast.success(`环境变量 ${key} 设置成功`)
        loadGlobalEnvs()
    } catch (e) {
        toast.error('设置失败: ' + e)
    }
}

async function handleUnsetEnv(key: string) {
    try {
        await api.mise.unsetEnv(key)
        toast.success(`环境变量 ${key} 已移除`)
        loadGlobalEnvs()
    } catch (e) {
        toast.error('移除失败: ' + e)
    }
}

function applyPreset(preset: { key: string, value: string }) {
    newEnvKey.value = preset.key
    newEnvValue.value = preset.value
    showAddEnvDialog.value = true
}

function startAddEnv() {
    if (!newEnvKey.value.trim()) return
    handleSetEnv(newEnvKey.value.trim(), newEnvValue.value.trim())
    showAddEnvDialog.value = false
    newEnvKey.value = ''
    newEnvValue.value = ''
}

onMounted(() => {
    loadGlobalEnvs()
})
</script>

<template>
    <div class="space-y-4">
        <div class="rounded-lg border bg-card p-6">
            <div class="flex items-start justify-between mb-8">
                <div>
                    <h3 class="text-lg font-bold flex items-center gap-2">
                        <Globe class="h-5 w-5 text-primary" />
                        镜像源加速 (Mirror Acceleration)
                    </h3>
                    <p class="text-xs text-muted-foreground mt-1 max-w-2xl">
                        通过配置全局环境变量，加速编程语言环境及依赖包（如 pip, npm, go get）的安装速度。
                    </p>
                </div>
            </div>

            <!-- 常用预设网格 -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <div v-for="preset in ENV_PRESETS" :key="preset.label" :class="cn(
                    'relative p-4 rounded-xl border transition-all group overflow-hidden',
                    globalEnvs[preset.key] ? 'bg-primary/5 border-primary shadow-sm' : 'bg-muted/30 hover:border-primary/50 cursor-pointer'
                )"
                    @click="applyPreset({ key: preset.key, value: globalEnvs[preset.key] || preset.value })">

                    <div class="flex items-start justify-between mb-2">
                        <div class="font-semibold text-sm">{{ preset.label }}</div>
                        <div v-if="globalEnvs[preset.key]"
                            class="h-5 w-5 rounded-full bg-primary text-primary-foreground flex items-center justify-center">
                            <Check class="h-3 w-3" />
                        </div>
                    </div>

                    <div class="text-[10px] font-mono mb-3">
                        <div class="text-muted-foreground truncate">{{ preset.key }}</div>
                        <div v-if="globalEnvs[preset.key]"
                            class="mt-1 text-primary/80 font-medium truncate bg-primary/5 px-1.5 py-0.5 rounded-sm border border-primary/10"
                            :title="globalEnvs[preset.key] as string">
                            {{ globalEnvs[preset.key] }}
                        </div>
                    </div>

                    <div class="flex items-center justify-between mt-auto">
                        <span v-if="globalEnvs[preset.key]" class="text-[10px] font-medium text-primary">
                            已启用 (点击修改)
                        </span>
                        <span v-else
                            class="text-[10px] font-medium text-muted-foreground group-hover:text-primary transition-colors flex items-center">
                            立即启用
                            <ArrowRight class="h-3 w-3 ml-1 group-hover:translate-x-1 transition-transform" />
                        </span>

                        <Button v-if="globalEnvs[preset.key]" variant="ghost" size="icon"
                            class="h-6 w-6 text-destructive hover:bg-destructive/10"
                            @click.stop="handleUnsetEnv(preset.key)">
                            <Trash2 class="h-3.5 w-3.5" />
                        </Button>
                    </div>
                </div>
            </div>
        </div>

        <!-- 启用/编辑镜像加速对话框 -->
        <Dialog v-model:open="showAddEnvDialog">
            <DialogContent class="sm:max-w-[400px]">
                <DialogHeader>
                    <DialogTitle>设置镜像加速</DialogTitle>
                    <DialogDescription>
                        正在配置 <span class="font-bold text-primary">{{ newEnvKey }}</span>。你可以根据需要调整镜像地址。
                    </DialogDescription>
                </DialogHeader>
                <div class="grid gap-4 py-4">
                    <div class="grid gap-2">
                        <Label>环境变量名 (Readonly)</Label>
                        <Input v-model="newEnvKey" readonly class="bg-muted cursor-default" />
                    </div>
                    <div class="grid gap-2">
                        <Label>镜像源地址 (Value)</Label>
                        <Input v-model="newEnvValue" placeholder="请输入镜像地址" @keydown.enter="startAddEnv" />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" @click="showAddEnvDialog = false">取消</Button>
                    <Button @click="startAddEnv">确认启用</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
</template>
