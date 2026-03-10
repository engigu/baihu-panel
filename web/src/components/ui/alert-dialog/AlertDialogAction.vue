<script setup lang="ts">
import type { AlertDialogActionProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { AlertDialogAction } from "reka-ui"
import { cn } from "@/lib/utils"
import { buttonVariants, type ButtonVariants } from '@/components/ui/button'

const props = defineProps<AlertDialogActionProps & {
  class?: HTMLAttributes["class"]
  variant?: ButtonVariants['variant']
  size?: ButtonVariants['size']
}>()

const delegatedProps = reactiveOmit(props, "class", "variant", "size")
</script>

<template>
  <AlertDialogAction v-bind="delegatedProps"
    :class="cn(buttonVariants({ variant: props.variant, size: props.size }), props.class)">
    <slot />
  </AlertDialogAction>
</template>
