/**
 * 白虎面板统一镜像源与 GitHub 代理加速工具模块
 * 前后端对齐定义，提供全局统一的加速代理、CDN 矩阵、应用市场资源解析及多源回退拉取能力
 */

export const DEFAULT_APPSTORE_OWNER = 'engigu'
export const DEFAULT_APPSTORE_REPO = 'baihu-appstore'
export const DEFAULT_APPSTORE_BRANCH = 'main'

// 推荐的代理节点前缀 (按可用性与速度排序，gh-proxy.com 首选)
export const GH_PROXY_ENDPOINTS = [
  'https://gh-proxy.com/',
  'https://ghfast.top/',
  'https://ghproxy.net/',
  'https://mirror.ghproxy.com/',
  'https://ghp.ci/'
] as const

// jsDelivr 免费 CDN 节点矩阵 (适合公开仓库静态文件)
export const JSDELIVR_ENDPOINTS = [
  'https://fastly.jsdelivr.net/gh/',
  'https://cdn.jsdelivr.net/gh/',
  'https://gcore.jsdelivr.net/gh/'
] as const

// 仓库同步代理选项 (统一供表单下拉框使用)
export const PROXY_TYPE_OPTIONS = [
  { label: '不使用代理 (直连)', value: 'none' },
  { label: 'gh-proxy.com (推荐)', value: 'ghproxy' },
  { label: 'mirror.ghproxy.com', value: 'mirror' },
  { label: '自定义代理', value: 'custom' }
]

/**
 * 为指定 Git / 文件 URL 注入加速代理前缀
 */
export function buildProxyUrl(rawUrl: string, proxyType: string, customProxy?: string): string {
  const cleanUrl = (rawUrl || '').trim()
  if (!cleanUrl || !proxyType || proxyType === 'none') {
    return cleanUrl
  }

  // 容错：若 URL 已经自带代理前缀，则跳过注入
  if (
    cleanUrl.includes('gh-proxy.com') ||
    cleanUrl.includes('ghproxy') ||
    cleanUrl.includes('ghfast.top') ||
    cleanUrl.includes('ghp.ci') ||
    cleanUrl.includes('googo.win') ||
    (proxyType === 'custom' && customProxy && cleanUrl.startsWith(customProxy))
  ) {
    return cleanUrl
  }

  let base = ''
  switch (proxyType) {
    case 'ghproxy':
      base = GH_PROXY_ENDPOINTS[0]
      break
    case 'mirror':
      base = 'https://mirror.ghproxy.com/'
      break
    case 'custom':
      if (customProxy) {
        base = customProxy.replace(/\/+$/, '') + '/'
      }
      break
    default:
      if (proxyType.startsWith('http://') || proxyType.startsWith('https://')) {
        base = proxyType.replace(/\/+$/, '') + '/'
      }
  }

  if (base && cleanUrl.startsWith('http') && !cleanUrl.startsWith(base)) {
    return `${base}${cleanUrl}`
  }
  return cleanUrl
}

/**
 * 生成指定 GitHub 仓库文件的多源容灾候选列表 (CDN + 代理 + 直连)
 */
export function getGitHubRawCandidates(
  owner: string,
  repo: string,
  branch: string = 'main',
  filePath: string
): string[] {
  const cleanPath = filePath.replace(/^\/+/, '')
  const candidates: string[] = []

  // 1. jsDelivr CDN 矩阵
  for (const cdn of JSDELIVR_ENDPOINTS) {
    candidates.push(`${cdn}${owner}/${repo}@${branch}/${cleanPath}`)
  }

  // 2. 原生 Raw 直连
  const directRaw = `https://raw.githubusercontent.com/${owner}/${repo}/${branch}/${cleanPath}`
  candidates.push(directRaw)

  // 3. ghproxy 代理矩阵
  for (const proxy of GH_PROXY_ENDPOINTS) {
    candidates.push(`${proxy}${directRaw}`)
  }

  return candidates
}

/**
 * 获取白虎应用市场指定应用 app.yaml 清单的多源候选列表
 */
export function getAppStoreYamlCandidates(
  appId: string,
  customManifestUrl?: string,
  branch: string = DEFAULT_APPSTORE_BRANCH
): string[] {
  const cleanAppId = (appId || '').trim()
  const candidates: string[] = []

  // 1. 若应用提供了明确的 manifest_url，作为最高优先级
  if (customManifestUrl && customManifestUrl.trim()) {
    candidates.push(customManifestUrl.trim())
  }

  if (!cleanAppId) {
    return candidates
  }

  // 2. GitHub Pages (若开启，支持快速跨域)
  candidates.push(`https://${DEFAULT_APPSTORE_OWNER}.github.io/${DEFAULT_APPSTORE_REPO}/apps/${cleanAppId}/app.yaml`)

  // 3. 全局统一 GitHub Raw 镜像矩阵
  const rawCandidates = getGitHubRawCandidates(
    DEFAULT_APPSTORE_OWNER,
    DEFAULT_APPSTORE_REPO,
    branch,
    `apps/${cleanAppId}/app.yaml`
  )
  candidates.push(...rawCandidates)

  // 去重返回
  return Array.from(new Set(candidates))
}

/**
 * 获取白虎应用市场 apps.json 索引的多源候选列表
 */
export function getAppStoreAppsJsonCandidates(branch: string = DEFAULT_APPSTORE_BRANCH): string[] {
  const candidates: string[] = [
    `https://${DEFAULT_APPSTORE_OWNER}.github.io/${DEFAULT_APPSTORE_REPO}/apps.json`,
    ...getGitHubRawCandidates(DEFAULT_APPSTORE_OWNER, DEFAULT_APPSTORE_REPO, branch, 'apps.json')
  ]
  return Array.from(new Set(candidates))
}

/**
 * 带有超时控制的多候选源顺序轮询容灾拉取器
 */
export async function fetchWithFallback(
  candidates: string[],
  options?: { timeoutMs?: number; cacheBust?: boolean }
): Promise<{ text: string; usedUrl: string }> {
  if (!candidates || candidates.length === 0) {
    throw new Error('候选镜像源列表为空')
  }

  const timeoutMs = options?.timeoutMs ?? 6000
  const cacheBust = options?.cacheBust ?? true
  const nowTs = Date.now()
  let lastError: any = null

  for (const targetUrl of candidates) {
    try {
      const urlObj = new URL(targetUrl)
      if (cacheBust) {
        urlObj.searchParams.set('t', String(nowTs))
      }

      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), timeoutMs)

      const resp = await fetch(urlObj.toString(), {
        signal: controller.signal
      })
      clearTimeout(timer)

      if (resp.ok) {
        const text = await resp.text()
        if (text && text.trim()) {
          return { text, usedUrl: targetUrl }
        }
      }
      lastError = new Error(`HTTP 状态码: ${resp.status} (${targetUrl})`)
    } catch (err: any) {
      lastError = err
    }
  }

  throw lastError || new Error('所有候选镜像源均拉取失败')
}
