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
      }
    }
  },
  // 使用相对路径，这样动态导入的模块也会使用相对路径
  // 浏览器会根据当前页面 URL 解析相对路径
  base: './'
})
