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
        theme: 'purple',
        showSidebar: true
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
}

/* 覆盖 VitePress 可能存在的样式干扰 */
.scalar-container :deep(.scalar-api-reference) {
  min-height: 100vh;
}
</style>
