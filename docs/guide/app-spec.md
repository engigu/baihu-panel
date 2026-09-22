# 白虎应用市场与规范 (App Specification)

> 💡 **实时动态同步**：本页面直接在线拉取并渲染官方应用商店仓库 [engigu/baihu-appstore](https://github.com/engigu/baihu-appstore) 的最新 `README.md` 文档。任何应用商店规范、环境变量契约或目录结构的最新改动均在此实时同步呈现。

<ClientOnly>
  <RemoteMarkdown 
    :urls="[
      'https://fastly.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://cdn.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://gcore.jsdelivr.net/gh/engigu/baihu-appstore@main/README.md',
      'https://raw.githubusercontent.com/engigu/baihu-appstore/main/README.md'
    ]"
    repoUrl="https://github.com/engigu/baihu-appstore"
    rawBase="https://raw.githubusercontent.com/engigu/baihu-appstore/main/"
  />
  <template #fallback>
    <div style="padding: 2rem 0; color: var(--vp-c-text-2);">
      正在从官方仓库加载最新应用规范...
    </div>
  </template>
</ClientOnly>
