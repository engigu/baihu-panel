<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { Home } from 'lucide-vue-next'

const router = useRouter()


function goHome() {
    router.push('/')
}
</script>

<template>
    <div class="fixed inset-0 flex items-center justify-center p-6 sm:p-12 overflow-hidden bg-background">
        <!-- Subtle Background Decoration -->
        <div class="absolute inset-0 z-0 pointer-events-none overflow-hidden">
            <div
                class="absolute -top-[10%] -left-[5%] w-[40%] h-[40%] rounded-full bg-primary/5 blur-[120px] animate-pulse" />
            <div
                class="absolute -bottom-[10%] -right-[5%] w-[40%] h-[40%] rounded-full bg-blue-500/5 blur-[120px] delay-1000 animate-pulse" />
            <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-full h-full opacity-[0.03] dark:opacity-[0.05]"
                style="background-image: radial-gradient(circle at 2px 2px, currentColor 1px, transparent 0); background-size: 40px 40px;">
            </div>
        </div>

        <div class="relative z-10 max-w-lg w-full text-center space-y-8 animate-in fade-in zoom-in-95 duration-700">
            <!-- 404 Illustration/Text -->
            <div class="relative inline-block mt-4">
                <h1
                    class="text-[12rem] sm:text-[15rem] font-black leading-none tracking-tighter text-transparent bg-clip-text bg-gradient-to-b from-primary via-primary/80 to-transparent opacity-10 select-none">
                    404
                </h1>
            </div>

            <!-- Text Content -->
            <div class="space-y-4 px-4">
                <h2 class="text-3xl sm:text-4xl font-extrabold tracking-tight text-foreground">看起来迷路了</h2>
                <p class="text-lg text-muted-foreground max-w-sm mx-auto leading-relaxed">
                    抱歉，您访问的页面不存在或已被移除。您可以尝试返回上一页或回到首页继续。
                </p>
            </div>

            <!-- Actions -->
            <div class="flex justify-center pt-4">
                <Button variant="default" size="lg" @click="goHome"
                    class="w-full sm:w-auto px-12 gap-2 active:scale-95 transition-all shadow-lg shadow-primary/20">
                    <Home class="w-4 h-4" />
                    返回首页
                </Button>
            </div>

            <!-- Footer Help -->
            <div class="pt-12 text-sm text-zinc-400 dark:text-zinc-600 animate-in slide-in-from-bottom-4 duration-1000">
                <p>如果有疑问，请联系系统管理员</p>
            </div>
        </div>
    </div>
</template>

<style scoped>
@keyframes pulse {

    0%,
    100% {
        opacity: 0.3;
        transform: scale(1);
    }

    50% {
        opacity: 0.5;
        transform: scale(1.05);
    }
}

.animate-pulse {
    animation: pulse 8s ease-in-out infinite;
}
</style>
