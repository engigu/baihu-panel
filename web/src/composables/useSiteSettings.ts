import { ref, computed } from 'vue'
import { api, type SiteSettings } from '@/api'

const CACHE_KEY = 'site_settings_cache'

// 从 localStorage 加载缓存
function loadFromCache(): SiteSettings | null {
  try {
    const cached = localStorage.getItem(CACHE_KEY)
    if (cached) {
      return JSON.parse(cached)
    }
  } catch {
    // 忽略解析错误
  }
  return null
}

// 保存到 localStorage
function saveToCache(settings: SiteSettings) {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(settings))
  } catch {
    // 忽略存储错误
  }
}

const cachedSettings = loadFromCache()
const siteSettings = ref<SiteSettings>(cachedSettings || {
  title: '白虎面板',
  subtitle: '轻量级定时任务管理系统',
  icon: '',
  page_size: '10',
  cookie_days: '7'
})

// 立即应用缓存的设置，防止标题闪烁或空白
if (cachedSettings) {
  if (cachedSettings.title) {
    document.title = cachedSettings.title
  }
  if (cachedSettings.icon) {
    updateFavicon(cachedSettings.icon)
  }
} else {
  document.title = siteSettings.value.title
}

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

const pageSize = computed(() => parseInt(siteSettings.value.page_size) || 10)

export function useSiteSettings() {
  async function loadSettings() {
    if (loaded) return
    try {
      const res = await api.settings.getSite()
      siteSettings.value = res
      saveToCache(res) // 保存到缓存
      document.title = res.title || '白虎面板'
      if (res.icon) {
        updateFavicon(res.icon)
      }
      loaded = true
    } catch (error) {
      // 使用默认值或缓存值
      document.title = siteSettings.value.title || '白虎面板'
      if (siteSettings.value.icon) {
        updateFavicon(siteSettings.value.icon)
      }
    }
  }

  function refreshSettings() {
    loaded = false
    return loadSettings()
  }

  return {
    siteSettings,
    pageSize,
    loadSettings,
    refreshSettings,
    updateFavicon
  }
}
