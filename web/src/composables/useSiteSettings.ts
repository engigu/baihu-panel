import { ref } from 'vue'
import { api, type SiteSettings } from '@/api'

const siteSettings = ref<SiteSettings>({
  title: '白虎面板',
  subtitle: '轻量级定时任务管理系统',
  icon: '',
  page_size: '10',
  cookie_days: '7'
})

let loaded = false

function updateFavicon(svgContent: string) {
  if (!svgContent || !svgContent.trim().startsWith('<svg')) return

  const blob = new Blob([svgContent], { type: 'image/svg+xml' })
  const url = URL.createObjectURL(blob)

  let link = document.querySelector("link[rel*='icon']") as HTMLLinkElement
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = 'image/svg+xml'
  link.href = url
}

export function useSiteSettings() {
  async function loadSettings() {
    if (loaded) return
    try {
      const res = await api.settings.getSite()
      siteSettings.value = res
      document.title = res.title || '白虎面板'
      if (res.icon) updateFavicon(res.icon)
      loaded = true
    } catch {
      // 使用默认值
    }
  }

  function refreshSettings() {
    loaded = false
    return loadSettings()
  }

  return {
    siteSettings,
    loadSettings,
    refreshSettings,
    updateFavicon
  }
}
