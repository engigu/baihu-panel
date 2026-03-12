import { computed, onMounted, ref, watch } from 'vue'

export type Theme = 'light' | 'dark' | 'system'

const theme = ref<Theme>('system')
const systemTheme = ref<'light' | 'dark'>('light')

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(t: Theme) {
  const root = document.documentElement
  const effectiveTheme = t === 'system' ? getSystemTheme() : t

  root.classList.remove('light', 'dark')
  root.classList.add(effectiveTheme)
}

export function useTheme() {
  onMounted(() => {
    systemTheme.value = getSystemTheme()

    const saved = localStorage.getItem('theme') as Theme | null
    if (saved && ['light', 'dark', 'system'].includes(saved)) {
      theme.value = saved
    }
    applyTheme(theme.value)

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      systemTheme.value = getSystemTheme()
      if (theme.value === 'system') {
        applyTheme('system')
      }
    })
  })

  watch(theme, (newTheme) => {
    localStorage.setItem('theme', newTheme)
    applyTheme(newTheme)
  })

  const resolvedTheme = computed<'light' | 'dark'>(() => {
    return theme.value === 'system' ? systemTheme.value : theme.value
  })

  function setTheme(t: Theme) {
    theme.value = t
  }

  return {
    theme,
    resolvedTheme,
    setTheme,
  }
}
