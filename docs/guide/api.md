---
layout: false
---

<script setup>
import { ApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'
</script>

<div class="scalar-container">
  <ClientOnly>
    <ApiReference 
      :configuration="{ 
        spec: { 
          url: '/baihu-panel/swagger.json' 
        },
        theme: 'alternate',
        showSidebar: true,
        withDefaultFonts: false,
        servers: [
          { 
            url: '{protocol}://{host}:{port}/open2api/v1',
            description: '可编辑的服务器地址',
            variables: {
              protocol: { default: 'http', enum: ['http', 'https'] },
              host: { default: 'localhost' },
              port: { default: '8052' }
            }
          }
        ]
      }" 
    />
  </ClientOnly>
</div>

<style>
:root, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
}

.scalar-container {
  height: 100vh;
  width: 100vw;
  --scalar-font: "Noto Sans SC", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  --scalar-font-code: "Ubuntu Mono", monospace;
}

/* 覆盖 VitePress 可能存在的样式干扰 */
.scalar-container :deep(.scalar-api-reference) {
  min-height: 100vh;
}
</style>
