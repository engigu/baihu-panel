<script setup lang="ts">
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Sun, Moon, Monitor } from 'lucide-vue-next'
import { useTheme } from '@/composables/useTheme'

const { theme, setTheme } = useTheme()
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" class="h-8 w-8">
        <Sun v-if="theme === 'light'" class="h-4 w-4" />
        <Moon v-else-if="theme === 'dark'" class="h-4 w-4" />
        <Monitor v-else class="h-4 w-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="min-w-[90px]">
      <DropdownMenuItem @click="setTheme('light')" class="gap-2">
        <Sun class="h-4 w-4" />
        <span>浅色</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="setTheme('dark')" class="gap-2">
        <Moon class="h-4 w-4" />
        <span>深色</span>
      </DropdownMenuItem>
      <DropdownMenuItem @click="setTheme('system')" class="gap-2">
        <Monitor class="h-4 w-4" />
        <span>系统</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
