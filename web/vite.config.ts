import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { viteStaticCopy } from 'vite-plugin-static-copy'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'
import { writeFileSync, readFileSync, readdirSync, statSync, existsSync, unlinkSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { gzipSync } from 'node:zlib'

// Simple inline compression plugin to avoid extra dependencies
const compressionPlugin = () => ({
  name: 'compression-plugin',
  apply: 'build',
  closeBundle: {
    order: 'post',
    handler: async () => {
      const distDir = resolve(process.cwd(), 'dist')
      if (!existsSync(distDir)) return
      
      const compressDir = (dir: string) => {
        const files = readdirSync(dir)
        for (const file of files) {
          const filePath = join(dir, file)
          const stat = statSync(filePath)
          if (stat.isDirectory()) {
            compressDir(filePath)
          } else if (file.match(/\.(js|css|html|svg|json|vs)$/) && !file.endsWith('.gz')) {
            try {
              const content = readFileSync(filePath)
              if (content.length < 1024) continue // Don't compress very small files
              const gzipped = gzipSync(content, { level: 9 })
              writeFileSync(`${filePath}.gz`, gzipped)
              // Remove original file after successful compression to save space in binary
              unlinkSync(filePath)
            } catch (e) {
              console.error(`Compression failed for ${file}:`, e)
            }
          }
        }
      }
      compressDir(distDir)
    }
  }
} as const)

// Release optimization plugin to offload fonts to CDN
const releaseOptimizePlugin = (isOptimize: boolean) => ({
  name: 'release-optimize-plugin',
  transformIndexHtml(html: string) {
    if (isOptimize) {
      return html.replace(
        '</head>',
        `  <link rel="preconnect" href="https://fonts.geekzu.org">\n  <link rel="stylesheet" href="https://fonts.geekzu.org/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&family=JetBrains+Mono:ital,wght@0,100..800;1,100..800&display=swap">\n  </head>`
      )
    }
    return html
  },
  transform(code: string, id: string) {
    if (isOptimize && id.includes('index.css')) {
      return {
        code: code
          .replace(/@import\s+["']@fontsource-variable\/inter\/index\.css["'];/g, '/* Removed for CDN */')
          .replace(/@import\s+["']@fontsource-variable\/jetbrains-mono\/index\.css["'];/g, '/* Removed for CDN */'),
        map: null
      }
    }
  }
})

const isOptimize = process.env.VITE_RELEASE_OPTIMIZE === 'true'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    !isOptimize && viteStaticCopy({
      targets: [
        {
          src: 'node_modules/monaco-editor/min/vs',
          dest: 'assets'
        }
      ]
    }),
    releaseOptimizePlugin(isOptimize),
    compressionPlugin(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      includeAssets: ['favicon.ico', 'logo.svg', 'pwa-icon-192.png', 'pwa-icon-512.png'],
      manifest: {
        name: 'Baihu Panel',
        short_name: 'Baihu',
        description: '白虎面板 - 现代化的服务器管理面板',
        theme_color: '#ffffff',
        background_color: '#ffffff',
        display: 'standalone',
        icons: [
          {
            src: 'logo.svg',
            sizes: 'any',
            type: 'image/svg+xml',
            purpose: 'any maskable'
          },
        ]
      },
      workbox: {
        maximumFileSizeToCacheInBytes: 10 * 1024 * 1024 // 10MB to allow monaco-editor files
      },
      devOptions: {
        enabled: true
      }
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:8052',
        changeOrigin: true,
        ws: true
      },
      '/openapi': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:8052',
        changeOrigin: true
      }
    }
  },
  // 使用相对路径，这样动态导入的模块也会使用相对路径
  // 浏览器会根据当前页面 URL 解析相对路径
  base: './',
  build: {
    reportCompressedSize: true,
    sourcemap: false,
    rollupOptions: {
      output: {
        // 防止生成以 _ 开头的文件，导致被 Cloudflare Pages 或 Github Pages 等静态托管平台拦截并降级返回 HTML
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
        sanitizeFileName(name) {
          // 仿制 Rollup 的默认 sanitizeFileName，将特殊字符替换为 '-'
          let safeName = name.replace(/[\0?*:|"<>\/\\&=$]/g, '-')
          // 去除开头可能引起静态托管平台屏蔽的下划线 '_'
          return safeName.replace(/^_/, '')
        },
        manualChunks(id) {
          if (id.includes('node_modules')) {
            // 编辑器相关
            if (id.includes('monaco-editor') || id.includes('@guolao/vue-monaco-editor')) {
              return 'vendor-monaco'
            }
            // 图表相关
            if (id.includes('apexcharts') || id.includes('chart.js') || id.includes('vue3-apexcharts') || id.includes('vue-chartjs')) {
              return 'vendor-charts'
            }
            // 终端相关
            if (id.includes('@xterm/xterm') || id.includes('xterm') || id.includes('ansi-to-html') || id.includes('ansi-to-vue3')) {
              return 'vendor-terminal'
            }
            // 基础 UI 库
            if (id.includes('radix-vue') || id.includes('reka-ui') || id.includes('lucide-vue-next') || id.includes('date-fns')) {
              return 'vendor-ui'
            }
            // 其他基础依赖
            return 'vendor'
          }
        }
      }
    }
  },
  define: {
    __MONACO_CDN__: isOptimize ? JSON.stringify('https://registry.npmmirror.com/monaco-editor/0.52.0/files/min/vs') : 'null'
  }
})
