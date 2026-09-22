<template>
  <div class="remote-markdown-container">
    <!-- 顶部状态栏与控制面板 -->
    <div class="remote-header-bar">
      <div class="status-indicator">
        <span v-if="loading" class="badge loading">
          <span class="spinner"></span> 正在从远程仓库获取最新文档...
        </span>
        <span v-else-if="error" class="badge error">
          ⚠️ 获取失败 (所有镜像源重试完毕)
        </span>
        <span v-else class="badge success">
          ✅ 实时在线同步成功 (源: {{ activeSourceName }})
        </span>
        <span v-if="syncTime" class="sync-time">同步时间: {{ syncTime }}</span>
      </div>
      <div class="action-buttons">
        <button class="btn-action" :disabled="loading" @click="fetchMarkdown" title="重新从远程仓库拉取">
          🔄 刷新文档
        </button>
        <a v-if="repoUrl" :href="repoUrl" target="_blank" rel="noopener noreferrer" class="btn-action link-btn">
          📦 访问上游仓库
        </a>
      </div>
    </div>

    <!-- 加载骨架屏 -->
    <div v-if="loading" class="skeleton-wrapper">
      <div class="skeleton-line title"></div>
      <div class="skeleton-line paragraph"></div>
      <div class="skeleton-line paragraph short"></div>
      <div class="skeleton-line code-box"></div>
      <div class="skeleton-line paragraph"></div>
    </div>

    <!-- 加载失败提示 -->
    <div v-else-if="error" class="error-wrapper">
      <div class="error-content">
        <h3>无法加载在线文档</h3>
        <p class="error-msg">{{ errorMessage }}</p>
        <p class="error-tip">可能是由于当前网络环境阻断了 GitHub / CDN 的访问，您可以尝试刷新或直接访问上游仓库。</p>
        <div class="error-actions">
          <button class="retry-btn" @click="fetchMarkdown">立即重试</button>
          <a v-if="repoUrl" :href="repoUrl" target="_blank" class="fallback-link">前往 GitHub 仓库查看 &rarr;</a>
        </div>
      </div>
    </div>

    <!-- 成功渲染内容 -->
    <div
      v-else
      class="vp-doc remote-markdown-body"
      v-html="renderedHtml"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { marked } from 'marked'

const props = withDefaults(
  defineProps<{
    urls?: string[]
    repoUrl?: string
    defaultBranch?: string
    rawBase?: string
  }>(),
  {
    urls: () => [
      'https://raw.githubusercontent.com/engigu/baihu-appstore/main/README.md',
      'https://fastly.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://cdn.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://ghproxy.net/https://raw.githubusercontent.com/engigu/baihu-appstore/main/README.md'
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

  return fixed
}

async function fetchMarkdown() {
  loading.value = true
  error.value = false
  errorMessage.value = ''

  let content = ''
  let successUrl = ''

  for (const url of props.urls) {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 8000)

      const res = await fetch(url, {
        signal: controller.signal,
        headers: {
          'Cache-Control': 'no-cache'
        }
      })
      clearTimeout(timeoutId)

      if (res.ok) {
        content = await res.text()
        successUrl = url
        break
      }
    } catch {
      // 忽略单次失败，尝试下一轮镜像
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

    // 解析 markdown
    const parsed = await marked.parse(normalizedMarkdown, {
      gfm: true,
      breaks: false
    })

    // 尝试 dompurify 消毒
    if (typeof window !== 'undefined') {
      try {
        const DOMPurifyModule = await import('dompurify')
        const DOMPurify = DOMPurifyModule.default || DOMPurifyModule
        renderedHtml.value = DOMPurify.sanitize(parsed)
      } catch {
        renderedHtml.value = parsed
      }
    } else {
      renderedHtml.value = parsed
    }

    activeSourceName.value = getSourceName(successUrl)
    syncTime.value = new Date().toLocaleTimeString()
    loading.value = false
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
  margin-top: 1rem;
}

.remote-header-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1.5rem;
  background-color: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

.status-indicator {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-weight: 500;
}

.badge.loading {
  color: var(--vp-c-brand-1);
}

.badge.success {
  color: var(--vp-c-green-1, #10b981);
}

.badge.error {
  color: var(--vp-c-danger-1, #ef4444);
}

.sync-time {
  color: var(--vp-c-text-2);
  font-size: 0.8rem;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-action {
  display: inline-flex;
  align-items: center;
  font-size: 0.8rem;
  padding: 0.25rem 0.65rem;
  background-color: var(--vp-c-bg-mute);
  border: 1px solid var(--vp-c-border);
  border-radius: 6px;
  color: var(--vp-c-text-1);
  cursor: pointer;
  text-decoration: none;
  transition: all 0.2s ease;
}

.btn-action:hover:not(:disabled) {
  background-color: var(--vp-c-brand-1);
  color: #fff;
  border-color: var(--vp-c-brand-1);
}

.btn-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Spinner */
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
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
  gap: 1rem;
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
  border-radius: 4px;
}

.skeleton-line.title {
  height: 32px;
  width: 50%;
}

.skeleton-line.paragraph {
  height: 18px;
  width: 100%;
}

.skeleton-line.paragraph.short {
  width: 75%;
}

.skeleton-line.code-box {
  height: 120px;
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

/* 错误提示 */
.error-wrapper {
  padding: 2rem;
  border: 1px dashed var(--vp-c-danger-2, #fca5a5);
  background-color: var(--vp-c-danger-soft, rgba(239, 68, 68, 0.08));
  border-radius: 8px;
  text-align: center;
}

.error-content h3 {
  margin-top: 0;
  color: var(--vp-c-danger-1, #ef4444);
}

.error-msg {
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
}

.error-tip {
  font-size: 0.9rem;
  color: var(--vp-c-text-2);
  margin: 0.75rem 0 1.25rem;
}

.error-actions {
  display: flex;
  justify-content: center;
  gap: 1rem;
}

.retry-btn {
  padding: 0.4rem 1.2rem;
  background-color: var(--vp-c-brand-1);
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.fallback-link {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 1rem;
  color: var(--vp-c-brand-1);
  text-decoration: underline;
  font-size: 0.875rem;
}

/* Markdown 内容融合 */
.remote-markdown-body {
  margin-top: 1rem;
}
</style>
