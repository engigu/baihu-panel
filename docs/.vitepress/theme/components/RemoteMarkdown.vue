<template>
  <div class="remote-markdown-container">
    <!-- 顶部状态栏与控制面板 -->
    <div class="remote-header-bar">
      <div class="status-indicator">
        <!-- 加载中状态 -->
        <div v-if="loading" class="pill-badge pill-loading">
          <svg class="ico ico-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" stroke-opacity="0.25" stroke="currentColor"></circle>
            <path d="M12 2a10 10 0 0 1 10 10" stroke-linecap="round"></path>
          </svg>
          <span>正在拉取最新规范...</span>
        </div>

        <!-- 错误状态 -->
        <div v-else-if="error" class="pill-badge pill-error">
          <svg class="ico" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
          </svg>
          <span>同步失败</span>
        </div>

        <!-- 成功状态 -->
        <div v-else class="pill-badge pill-success">
          <svg class="ico" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
          </svg>
          <span>实时同步就绪</span>
        </div>

        <!-- 来源与同步时间元数据 -->
        <div v-if="!loading && !error" class="meta-info">
          <span class="meta-item">
            <span class="meta-label">源:</span> {{ activeSourceName }}
          </span>
          <span class="meta-dot">·</span>
          <span class="meta-item">
            <span class="meta-label">时间:</span> {{ syncTime }}
          </span>
        </div>
      </div>

      <!-- 操作按钮组 -->
      <div class="action-buttons">
        <button class="btn-action" :disabled="loading" @click="fetchMarkdown" title="重新从远程仓库拉取最新文档">
          <svg class="ico-btn" :class="{ 'ico-spin': loading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"></path>
          </svg>
          <span>刷新文档</span>
        </button>

        <a v-if="repoUrl" :href="repoUrl" target="_blank" rel="noopener noreferrer" class="btn-action btn-repo" title="前往 GitHub 官方仓库">
          <svg class="ico-btn" viewBox="0 0 24 24" fill="currentColor">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"></path>
          </svg>
          <span>访问仓库</span>
          <svg class="ico-external" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6M15 3h6v6M10 14L21 3"></path>
          </svg>
        </a>
      </div>
    </div>

    <!-- 加载骨架屏 -->
    <div v-if="loading" class="skeleton-wrapper">
      <div class="skeleton-line skeleton-title"></div>
      <div class="skeleton-line skeleton-p"></div>
      <div class="skeleton-line skeleton-p short"></div>
      <div class="skeleton-line skeleton-card"></div>
      <div class="skeleton-line skeleton-p"></div>
    </div>

    <!-- 加载失败提示 -->
    <div v-else-if="error" class="error-wrapper">
      <div class="error-icon-box">
        <svg class="error-big-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
      </div>
      <div class="error-content">
        <h3>在线文档加载失败</h3>
        <p class="error-msg">{{ errorMessage }}</p>
        <p class="error-tip">由于网络环境或 CDN 限制，未能成功拉取上游 Markdown。您可以点击重试或直接访问 GitHub 仓库。</p>
        <div class="error-actions">
          <button class="retry-btn" @click="fetchMarkdown">
            <svg class="ico-btn" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"></path>
            </svg>
            <span>重新尝试</span>
          </button>
          <a v-if="repoUrl" :href="repoUrl" target="_blank" class="fallback-link">
            <span>浏览 GitHub 仓库</span>
            <svg class="ico-external" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14M12 5l7 7-7 7"></path>
            </svg>
          </a>
        </div>
      </div>
    </div>

    <!-- 成功渲染内容 (绑定点击事件拦截，支持目录锚点平滑滚动) -->
    <div
      v-else
      class="vp-doc remote-markdown-body"
      v-html="renderedHtml"
      @click="handleContentClick"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { Marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/monokai-sublime.css'

const props = withDefaults(
  defineProps<{
    urls?: string[]
    repoUrl?: string
    defaultBranch?: string
    rawBase?: string
  }>(),
  {
    urls: () => [
      'https://fastly.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://cdn.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://gcore.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://raw.githubusercontent.com/engigu/baihu-appstore/main/README.md'
    ],
    repoUrl: 'https://github.com/engigu/baihu-appstore',
    defaultBranch: 'main',
    rawBase: 'https://raw.githubusercontent.com/engigu/baihu-appstore/main/'
  }
)

const loading = ref(true)
const error = ref(false)
const errorMessage = ref('')
const activeSourceName = ref('')
const syncTime = ref('')
const renderedHtml = ref('')

// 格式化源显示名
function getSourceName(url: string): string {
  try {
    const parsed = new URL(url)
    return parsed.hostname
  } catch {
    return '远程镜像'
  }
}

// 符合 GitHub 官方目录规则的 Slugger 算法 (用于生成标题与锚点唯一 ID)
function githubSlug(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/<[^>]+>/g, '') // 移除 HTML 标签
    // 移除标点（中文标点：、，。！？；：（）等；英文标点：().,/:;"'?!~`@#$%^&*+=）
    // 保留字、字母、数字、中文、连字符 - 和下划线 _，空格保留用于转换为 -
    .replace(/[^\w\u4e00-\u9fa5\s\-_]/g, '')
    // 空格替换为连字符 -
    .replace(/\s/g, '-')
    // 去除开头和末尾多余的连字符
    .replace(/^-+|-+$/g, '')
}

// 修复 Markdown 中的相对链接与图片路径
function fixRelativePaths(mdContent: string): string {
  if (!props.rawBase) return mdContent

  // 1. 修复相对图片路径 ![alt](./path/to/img) 或 ![alt](path/to/img)
  let fixed = mdContent.replace(
    /!\[(.*?)\]\((?!https?:\/\/|\/\/|data:)(?:\.\/)?(.*?)\)/g,
    (match, alt, path) => `![${alt}](${props.rawBase}${path})`
  )

  // 2. 修复 <img src="./path" />
  fixed = fixed.replace(
    /<img\s+([^>]*?)src=["'](?!https?:\/\/|\/\/|data:)(?:\.\/)?(.*?)["']([^>]*?)>/gi,
    (match, prefix, path, suffix) => `<img ${prefix}src="${props.rawBase}${path}"${suffix}>`
  )

  // 3. 将连续换行的 Badge 徽标合并至同一行展示
  fixed = fixed.replace(/(\]\([^\)]+\))\s*\n+\s*(\[!\[)/g, '$1 $2')

  return fixed
}

// 锚点平滑滚动定位（自动预留 VitePress 顶部 Header 高度）
function scrollToTargetId(targetId: string, pushHash?: string) {
  if (!targetId) return

  // 尝试按精准 ID 寻找
  let el = document.getElementById(targetId)
  if (!el) {
    try {
      el = document.querySelector(`[id="${CSS.escape(targetId)}"]`)
    } catch {
      // 忽略 CSS selector 异常
    }
  }

  // 兜底模糊匹配：尝试对所有标题进行对比
  if (!el) {
    const headings = document.querySelectorAll('.remote-markdown-body h1, .remote-markdown-body h2, .remote-markdown-body h3, .remote-markdown-body h4, .remote-markdown-body h5, .remote-markdown-body h6')
    for (const h of headings) {
      if (h.id === targetId || githubSlug(h.textContent || '') === targetId) {
        el = h as HTMLElement
        break
      }
    }
  }

  if (el) {
    const navOffset = 80 // 导航栏高度 + 安全边距
    const top = el.getBoundingClientRect().top + window.pageYOffset - navOffset
    window.scrollTo({
      top: Math.max(0, top),
      behavior: 'smooth'
    })

    if (pushHash && window.location.hash !== pushHash) {
      history.pushState(null, '', pushHash)
    }
  }
}

// 处理文档内部点击：拦截内部锚点跳转并触发平滑滚动
function handleContentClick(e: MouseEvent) {
  const anchor = (e.target as HTMLElement).closest('a')
  if (!anchor) return

  const href = anchor.getAttribute('href')
  if (!href) return

  // 内部锚点跳转 (#xxx)
  if (href.startsWith('#')) {
    e.preventDefault()
    const targetId = decodeURIComponent(href.slice(1))
    scrollToTargetId(targetId, href)
  } else if (href.startsWith('http://') || href.startsWith('https://')) {
    // 外部链接一律新标签页安全打开
    anchor.setAttribute('target', '_blank')
    anchor.setAttribute('rel', 'noopener noreferrer')
  }
}

async function fetchMarkdown() {
  loading.value = true
  error.value = false
  errorMessage.value = ''

  let content = ''
  let successUrl = ''

  for (const rawUrl of props.urls) {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 6000)

      // 使用时间戳参数避免缓存，且不带任何自定义 header，以确保为 CORS Simple Request 避免触发 preflight
      const separator = rawUrl.includes('?') ? '&' : '?'
      const fetchUrl = `${rawUrl}${separator}_t=${Date.now()}`

      const res = await fetch(fetchUrl, {
        signal: controller.signal
      })
      clearTimeout(timeoutId)

      if (res.ok) {
        content = await res.text()
        successUrl = rawUrl
        break
      }
    } catch {
      // 忽略单次失败，继续尝试下一个镜像源
      continue
    }
  }

  if (!content) {
    loading.value = false
    error.value = true
    errorMessage.value = '网络请求超时或全部镜像源均不可达'
    return
  }

  try {
    // 路径修正
    const normalizedMarkdown = fixRelativePaths(content)

    // 构建专用的 Marked 实例并注册 GitHub 风格的 Heading Renderer
    const markedInstance = new Marked({
      gfm: true,
      breaks: false
    })

    // 记录已经出现的 slug，防止重复 id
    const slugMap = new Map<string, number>()

    markedInstance.use({
      renderer: {
        heading({ tokens, depth, text }) {
          const inlineHtml = this.parser.parseInline(tokens)
          const rawText = text || tokens.map((t: any) => t.raw || t.text).join('')
          let slug = githubSlug(rawText)

          // 重复 slug 递增（如 #一-仓库目录结构-1）
          const count = slugMap.get(slug) || 0
          if (count > 0) {
            slugMap.set(slug, count + 1)
            slug = `${slug}-${count}`
          } else {
            slugMap.set(slug, 1)
          }

          return `<h${depth} id="${slug}">${inlineHtml}</h${depth}>\n`
        },
        code({ text, lang }) {
          const language = (lang || '').trim().toLowerCase()
          let highlighted = ''
          if (language && hljs.getLanguage(language)) {
            try {
              highlighted = hljs.highlight(text, { language, ignoreIllegals: true }).value
            } catch {
              highlighted = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
            }
          } else {
            try {
              highlighted = hljs.highlightAuto(text).value
            } catch {
              highlighted = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
            }
          }

          const langClass = language ? `language-${language}` : ''
          return `<div class="language-${language || 'text'} vp-adaptive-theme"><span class="lang">${language || ''}</span><pre class="hljs"><code class="${langClass}">${highlighted}</code></pre></div>\n`
        }
      }
    })

    const parsed = await markedInstance.parse(normalizedMarkdown)

    // 尝试 dompurify 消毒，显式保留 id 属性与代码高亮 span/class
    if (typeof window !== 'undefined') {
      try {
        const DOMPurifyModule = await import('dompurify')
        const DOMPurify = DOMPurifyModule.default || DOMPurifyModule
        renderedHtml.value = DOMPurify.sanitize(parsed, {
          ADD_ATTR: ['id', 'target', 'rel', 'class'],
          ADD_TAGS: ['span']
        })
      } catch {
        renderedHtml.value = parsed
      }
    } else {
      renderedHtml.value = parsed
    }

    activeSourceName.value = getSourceName(successUrl)
    syncTime.value = new Date().toLocaleTimeString()
    loading.value = false

    // 渲染完成后，如果 URL 中携带了初始 hash，自动定位到该标题
    nextTick(() => {
      if (typeof window !== 'undefined' && window.location.hash) {
        const initTargetId = decodeURIComponent(window.location.hash.slice(1))
        setTimeout(() => {
          scrollToTargetId(initTargetId)
        }, 350)
      }
    })
  } catch (err: any) {
    loading.value = false
    error.value = true
    errorMessage.value = err?.message || 'Markdown 渲染异常'
  }
}

onMounted(() => {
  fetchMarkdown()
})
</script>

<style scoped>
.remote-markdown-container {
  margin-top: 1.25rem;
}

/* 顶部状态栏 */
.remote-header-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.65rem 1rem;
  margin-bottom: 1.5rem;
  background-color: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  font-size: 0.85rem;
}

.status-indicator {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
}

/* 状态胶囊 */
.pill-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.2rem 0.65rem;
  border-radius: 9999px;
  font-weight: 500;
  font-size: 0.78rem;
  line-height: 1.2;
}

.pill-loading {
  background-color: var(--vp-c-brand-soft, rgba(16, 185, 129, 0.12));
  color: var(--vp-c-brand-1);
}

.pill-success {
  background-color: var(--vp-c-green-soft, rgba(16, 185, 129, 0.12));
  color: var(--vp-c-green-1, #10b981);
}

.pill-error {
  background-color: var(--vp-c-danger-soft, rgba(239, 68, 68, 0.12));
  color: var(--vp-c-danger-1, #ef4444);
}

/* 元数据标签 */
.meta-info {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--vp-c-text-2);
  font-size: 0.8rem;
  font-family: var(--vp-font-family-mono, monospace);
}

.meta-label {
  color: var(--vp-c-text-3);
}

.meta-dot {
  color: var(--vp-c-text-3);
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-action {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  padding: 0.35rem 0.75rem;
  background-color: var(--vp-c-bg-mute);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  color: var(--vp-c-text-1);
  cursor: pointer;
  text-decoration: none;
  transition: all 0.2s ease;
  user-select: none;
}

.btn-action:hover:not(:disabled) {
  background-color: var(--vp-c-bg-soft);
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.btn-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* SVG 图标通用规范 */
.ico {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.ico-btn {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}

.ico-external {
  width: 11px;
  height: 11px;
  opacity: 0.65;
  margin-left: -0.1rem;
}

.ico-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* 骨架屏 */
.skeleton-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
  padding: 1rem 0;
}

.skeleton-line {
  background: linear-gradient(
    90deg,
    var(--vp-c-bg-soft) 25%,
    var(--vp-c-bg-mute) 37%,
    var(--vp-c-bg-soft) 63%
  );
  background-size: 400% 100%;
  animation: skeleton-loading 1.4s ease infinite;
  border-radius: 6px;
}

.skeleton-title {
  height: 36px;
  width: 45%;
}

.skeleton-p {
  height: 18px;
  width: 100%;
}

.skeleton-p.short {
  width: 70%;
}

.skeleton-card {
  height: 140px;
  width: 100%;
  border-radius: 8px;
}

@keyframes skeleton-loading {
  0% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0 50%;
  }
}

/* 错误提示卡片 */
.error-wrapper {
  display: flex;
  align-items: flex-start;
  gap: 1.25rem;
  padding: 1.5rem;
  border: 1px solid var(--vp-c-danger-soft, rgba(239, 68, 68, 0.25));
  background-color: var(--vp-c-danger-soft, rgba(239, 68, 68, 0.05));
  border-radius: 8px;
  margin: 1.5rem 0;
}

.error-icon-box {
  color: var(--vp-c-danger-1, #ef4444);
  flex-shrink: 0;
  margin-top: 0.15rem;
}

.error-big-icon {
  width: 32px;
  height: 32px;
}

.error-content {
  flex: 1;
}

.error-content h3 {
  margin: 0 0 0.4rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--vp-c-danger-1, #ef4444);
}

.error-msg {
  font-family: var(--vp-font-family-mono, monospace);
  font-size: 0.8rem;
  color: var(--vp-c-text-2);
  margin: 0 0 0.5rem;
}

.error-tip {
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
  margin: 0 0 1rem;
  line-height: 1.5;
}

.error-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.retry-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.85rem;
  background-color: var(--vp-c-brand-1);
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.82rem;
  font-weight: 500;
  transition: opacity 0.2s ease;
}

.retry-btn:hover {
  opacity: 0.9;
}

.fallback-link {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.35rem 0.75rem;
  color: var(--vp-c-brand-1);
  font-size: 0.82rem;
  text-decoration: none;
}

.fallback-link:hover {
  text-decoration: underline;
}

/* Markdown 内容渲染容器 */
.remote-markdown-body {
  margin-top: 1rem;
}

/* 保证所有标题锚点拥有滚动防遮挡预留边距 */
:deep(.remote-markdown-body h1),
:deep(.remote-markdown-body h2),
:deep(.remote-markdown-body h3),
:deep(.remote-markdown-body h4),
:deep(.remote-markdown-body h5),
:deep(.remote-markdown-body h6) {
  scroll-margin-top: calc(var(--vp-nav-height, 64px) + 24px);
}

/* 代码高亮容器 - 经典 Monokai 风格 */
:deep(.vp-adaptive-theme) {
  position: relative;
  margin: 1.25rem 0;
  border-radius: 8px;
  background-color: #23241f !important;
  overflow: hidden;
  border: 1px solid #3e3d32;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
}

:deep(.vp-adaptive-theme .lang) {
  position: absolute;
  top: 6px;
  right: 12px;
  font-size: 11px;
  font-weight: 700;
  color: #75715e;
  text-transform: uppercase;
  user-select: none;
  pointer-events: none;
  letter-spacing: 0.5px;
}

:deep(.vp-adaptive-theme pre) {
  margin: 0;
  padding: 1.25rem 1rem 1rem;
  background: #23241f !important;
  color: #f8f8f2;
  overflow-x: auto;
}

:deep(.vp-adaptive-theme code) {
  font-family: var(--vp-font-family-mono, 'Ubuntu Mono', Consolas, Monaco, monospace);
  font-size: 0.88rem;
  line-height: 1.6;
}

/* 经典 Monokai 专属语法配色 */
:deep(.hljs) {
  background: #23241f !important;
  color: #f8f8f2 !important;
}

/* 属性名 / YAML Key (经典 Monokai 荧光品红粉) */
:deep(.hljs-attr),
:deep(.hljs-attribute) {
  color: #f92672 !important;
  font-weight: 500;
}

/* 字符串 (经典 Monokai 暖麦浅黄) */
:deep(.hljs-string) {
  color: #e6db74 !important;
}

/* 注释 (经典 Monokai 暗灰橄榄金) */
:deep(.hljs-comment),
:deep(.hljs-quote) {
  color: #75715e !important;
  font-style: italic;
}

/* 数字、布尔值与字面量 (经典 Monokai 薰衣草亮紫) */
:deep(.hljs-number),
:deep(.hljs-literal),
:deep(.hljs-boolean) {
  color: #ae81ff !important;
  font-weight: 500;
}

/* 关键字 (经典 Monokai 荧光品红) */
:deep(.hljs-keyword),
:deep(.hljs-selector-tag) {
  color: #f92672 !important;
  font-weight: bold;
}

/* 列表符号与符号元信息 (经典 Monokai 电光青蓝) */
:deep(.hljs-bullet),
:deep(.hljs-symbol),
:deep(.hljs-meta) {
  color: #66d9ef !important;
}

/* 标题、函数名与类别 (经典 Monokai 柠檬嫩绿) */
:deep(.hljs-title),
:deep(.hljs-section),
:deep(.hljs-type),
:deep(.hljs-built_in) {
  color: #a6e22e !important;
  font-weight: bold;
}

/* 变量与模板参数 (经典 Monokai 鲜亮橙色) */
:deep(.hljs-variable),
:deep(.hljs-template-variable),
:deep(.hljs-params) {
  color: #fd971f !important;
}

/* 徽章 (Badge) 图片与链接单行水平并排展示 */
:deep(.remote-markdown-body p a img),
:deep(.remote-markdown-body p > img) {
  display: inline-block !important;
  vertical-align: middle !important;
  margin-top: 0 !important;
  margin-bottom: 6px !important;
  margin-right: 8px !important;
}

:deep(.remote-markdown-body a:has(> img)) {
  display: inline-block !important;
  vertical-align: middle !important;
  margin-right: 8px !important;
  margin-bottom: 6px !important;
  line-height: 1 !important;
  text-decoration: none !important;
}
</style>
