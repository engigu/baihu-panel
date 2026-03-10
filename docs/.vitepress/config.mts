import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
    title: "白虎面板",
    description: "极致轻量、高性能的自动化任务调度平台",
    base: '/baihu-panel/', // 👈 如果您的 GitHub Pages 路径是 /baihu-panel/，请保留此项；若是自定义域名或根目录，请删除或设为 '/'
    lang: 'zh-CN',
    themeConfig: {
        // https://vitepress.dev/reference/default-theme-config
        logo: '/logo.svg',
        nav: [
            { text: '指南', link: '/guide/introduction' },
            { text: '部署', link: '/guide/deployment' },
            { text: 'API接口', link: '/guide/api' }
        ],

        sidebar: [
            {
                text: '快速开始',
                items: [
                    { text: '项目介绍', link: '/guide/introduction' },
                    { text: '快速部署', link: '/guide/deployment' },
                    { text: '访问面板', link: '/guide/getting-started' },
                    { text: 'API 接口', link: '/guide/api' }
                ]
            },
            {
                text: '使用指南',
                items: [
                    { text: '功能特性', link: '/guide/usage' },
                    { text: '编程语言与依赖管理', link: '/guide/languages' },
                    { text: '命令行工具 (CLI)', link: '/guide/cli' }
                ]
            },
            {
                text: '配置参考',
                items: [
                    { text: '系统配置', link: '/guide/configuration' },
                    { text: '反向代理 (Nginx)', link: '/guide/nginx' }
                ]
            },
            {
                text: '其他',
                items: [
                    { text: '更新日志', link: '/guide/changelog' },
                    { text: '免责声明', link: '/guide/disclaimer' }
                ]
            }
        ],

        socialLinks: [
            { icon: 'github', link: 'https://github.com/engigu/baihu-panel' }
        ],

        footer: {
            message: 'Released under the MIT License.',
            copyright: 'Copyright © 2026-present engigu'
        },

        search: {
            provider: 'local'
        }
    },
    vite: {
        ssr: {
            noExternal: ['@scalar/api-reference']
        }
    }
})
