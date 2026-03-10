<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Plus, X, AlertCircle } from 'lucide-vue-next'

interface Env {
  plugin: string
  version: string
}

const props = defineProps<{
  open: boolean
  selectedEnvs: Env[]
  langGroups: Record<string, string[]>
  getLangIcon: (plugin: string) => string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:selectedEnvs': [value: Env[]]
  'confirm': []
}>()

function addEnv() {
  const newEnvs = [...props.selectedEnvs, { plugin: '', version: '' }]
  emit('update:selectedEnvs', newEnvs)
}

function removeEnv(index: number) {
  const newEnvs = [...props.selectedEnvs]
  newEnvs.splice(index, 1)
  emit('update:selectedEnvs', newEnvs)
}

function updateEnvPlugin(index: number, plugin: string) {
  const newEnvs = [...props.selectedEnvs]
  const group = props.langGroups ? props.langGroups[plugin] : null
  newEnvs[index] = {
    plugin,
    version: (group && group.length > 0) ? (group[0] as string) : ''
  }
  emit('update:selectedEnvs', newEnvs)
}

function updateEnvVersion(index: number, version: string) {
  const newEnvs = [...props.selectedEnvs]
  if (newEnvs[index]) {
    newEnvs[index].version = version
    emit('update:selectedEnvs', newEnvs)
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="val => emit('update:open', val)">
    <DialogContent class="max-w-md" @openAutoFocus.prevent>
      <DialogHeader>
        <DialogTitle class="text-sm">运行配置</DialogTitle>
      </DialogHeader>
      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label class="text-xs font-medium">运行环境 (Mise)</Label>
          <div class="space-y-3">
            <div v-for="(env, index) in selectedEnvs" :key="index" class="flex gap-1.5 items-center group bg-muted/20 hover:bg-muted/40 p-1 rounded-lg border border-transparent hover:border-border/50 transition-all duration-200">
              <!-- 语言选择 -->
              <div class="flex-1 min-w-0">
                <Select :modelValue="env.plugin" @update:model-value="newVal => updateEnvPlugin(index, newVal as string)">
                  <SelectTrigger class="w-full bg-background/50 hover:bg-background border-border/50 h-9 transition-colors">
                    <div class="flex items-center gap-2 overflow-hidden">
                      <div class="h-5 w-5 rounded-full overflow-hidden shrink-0 bg-white flex items-center justify-center p-0.5" v-if="getLangIcon(env.plugin)">
                        <img :src="getLangIcon(env.plugin)" :alt="env.plugin" class="w-full h-full object-contain" />
                      </div>
                      <SelectValue placeholder="语言" class="text-xs" />
                    </div>
                  </SelectTrigger>
                  <SelectContent>
                    <template v-for="(_, lang) in langGroups" :key="lang">
                      <SelectItem v-if="lang !== 'none'" :value="lang as string">
                        <div class="flex items-center gap-2">
                          <img v-if="getLangIcon(lang as string)" :src="getLangIcon(lang as string)" class="h-3 w-3" />
                          <span class="text-xs">{{ lang }}</span>
                        </div>
                      </SelectItem>
                    </template>
                  </SelectContent>
                </Select>
              </div>

              <!-- 版本选择 -->
              <div class="w-28 shrink-0">
                 <Select :modelValue="env.version" @update:model-value="newVal => updateEnvVersion(index, newVal as string)" :disabled="!env.plugin">
                  <SelectTrigger class="w-full bg-background/50 hover:bg-background border-border/50 h-9 text-xs">
                    <SelectValue placeholder="版本" />
                  </SelectTrigger>
                  <SelectContent v-if="env.plugin && langGroups && langGroups[env.plugin]">
                    <SelectItem v-for="v in langGroups[env.plugin]" :key="v" :value="v" class="text-xs">
                      {{ v }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <!-- 删除按钮 -->
              <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground hover:text-destructive shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" @click="removeEnv(index)">
                <X class="h-3.5 w-3.5" />
              </Button>
            </div>

            <!-- 添加按钮 -->
            <Button variant="outline" class="w-full border-dashed h-9 text-xs font-normal text-muted-foreground hover:text-foreground" @click="addEnv">
              <Plus class="h-3 w-3 mr-2" /> 添加语言环境
            </Button>
          </div>
        </div>

        <div class="text-[10px] text-amber-500 bg-amber-500/10 p-3 rounded-lg border border-amber-500/20 flex gap-2">
          <AlertCircle class="h-3 w-3 shrink-0 mt-0.5" />
          <p>请先在「语言依赖」中安装所需的运行时。任务执行时将使用该环境，确保所有依赖已正确配置（如果是执行 <span class="font-bold">bash</span> 脚本，可随便选择一个环境即可）。</p>
        </div>

        <div class="text-[10px] text-muted-foreground bg-muted/40 p-3 rounded-lg border border-border/50">
          <p class="font-medium mb-1">提示：</p>
          <template v-if="selectedEnvs.length === 0">
            <p>使用服务器默认环境运行脚本。</p>
          </template>
          <template v-else>
            <p>将在此 Mise 隔离环境中运行脚本：</p>
            <div class="mt-1 flex flex-wrap gap-1">
              <span v-for="env in selectedEnvs" :key="env.plugin" class="bg-primary/10 text-primary px-1.5 py-0.5 rounded-sm" v-show="env.plugin && env.version">
                {{ env.plugin }}@{{ env.version }}
              </span>
            </div>
          </template>
        </div>
      </div>
      <DialogFooter class="gap-2 sm:gap-0">
        <Button variant="outline" size="sm" class="h-8 text-xs px-4" @click="emit('update:open', false)">取消</Button>
        <Button size="sm" class="h-8 text-xs px-4" @click="emit('confirm')">确认运行</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
