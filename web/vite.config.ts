import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8052',
        changeOrigin: true,
        ws: true
      },
      '/openapi': {
        target: 'http://localhost:8052',
        changeOrigin: true
      }
    }
  },
  // 使用相对路径，这样动态导入的模块也会使用相对路径
  // 浏览器会根据当前页面 URL 解析相对路径
  base: './',
  build: {
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
        }
      }
    }
  }
})
