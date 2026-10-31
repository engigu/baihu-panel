import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import type { Directive } from 'vue'

/**
 * 检测当前是否处于演示模式
 */
export function isDemoModeActive(): boolean {
  const { siteSettings } = useSiteSettings()
  const v = siteSettings.value?.demo_mode
  return v === 'true' || (v as any) === true
}

/**
 * 前端演示模式高阶函数包装器（函数级装饰器）
 * 在演示模式下自动拦截并弹出 toast 提示，阻止业务逻辑执行
 */
export function withDemoGuard<T extends (...args: any[]) => any>(
  fn: T,
  msg = '演示模式下禁止此操作'
): T {
  return ((...args: any[]) => {
    if (isDemoModeActive()) {
      toast.error(msg)
      return
    }
    return fn(...args)
  }) as T
}

/**
 * Vue 自定义指令 v-demo-guard（模板级装饰器）
 * 在元素点击时（捕获阶段）优先拦截并弹出 toast
 * 用法: <Button v-demo-guard="'演示模式下禁止操作'">保存</Button>
 */
export const vDemoGuard: Directive<HTMLElement, string | undefined> = {
  mounted(el, binding) {
    el.addEventListener(
      'click',
      (e) => {
        if (isDemoModeActive()) {
          e.preventDefault()
          e.stopPropagation()
          e.stopImmediatePropagation()
          toast.error(binding.value || '演示模式下禁止此操作')
        }
      },
      true // 捕获阶段优先拦截
    )
  }
}
