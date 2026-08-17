<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { getLangIcon } from '@/utils/icons'
import type { MiseLanguage } from '@/api'

interface DisplayLanguage extends Omit<MiseLanguage, 'source'> {
    source: string
    isGlobal: boolean
}

const props = defineProps<{
    open: boolean,
    lang: DisplayLanguage | null
}>()

const emit = defineEmits<{
    (e: 'update:open', val: boolean): void
}>()

const isOpen = computed({
    get: () => props.open,
    set: (val) => emit('update:open', val)
})
</script>

<template>
    <BaihuDialog v-model:open="isOpen" title="运行环境详情" icon="Info">
        <template v-if="lang">
            <div class="space-y-4">
                <div class="flex items-center gap-4 p-3 rounded-lg bg-muted/30 border border-border/40">
                    <div
                        class="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center font-bold text-primary uppercase overflow-hidden shrink-0 border border-primary/10">
                        <template v-if="getLangIcon(lang.plugin)">
                            <div class="w-full h-full bg-white/90 p-2.5">
                                <img :src="getLangIcon(lang.plugin)" class="w-full h-full object-contain" />
                            </div>
                        </template>
                        <template v-else>
                            <span class="text-lg">{{ lang.plugin.substring(0, 2) }}</span>
                        </template>
                    </div>
                    <div>
                        <div class="flex items-center gap-2">
                            <span class="font-bold text-lg capitalize">{{ lang.plugin }}</span>
                            <Badge variant="outline" class="font-mono text-xs">{{ lang.version }}</Badge>
                        </div>
                        <div class="text-xs text-muted-foreground mt-1">
                            {{ lang.isGlobal ? '系统全局默认版本' : '普通已安装版本' }}
                        </div>
                    </div>
                </div>

                <div class="space-y-3">
                    <div class="space-y-1.5">
                        <div class="text-xs font-bold text-muted-foreground uppercase tracking-wider px-1">
                            完整安装路径</div>
                        <div
                            class="p-3 bg-muted/50 rounded-lg border border-border/40 break-all text-xs font-mono leading-relaxed select-all">
                            {{ lang.source }}
                        </div>
                    </div>
                    <div v-if="lang.installed_at" class="flex flex-row items-center justify-between px-1">
                        <div class="text-xs font-bold text-muted-foreground uppercase tracking-wider">安装日期</div>
                        <div class="text-xs font-mono text-muted-foreground">{{ lang.installed_at }}</div>
                    </div>
                </div>
            </div>
        </template>
    </BaihuDialog>
</template>
