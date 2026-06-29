import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
    title: 'baihu-panel',
    description: '轻量易用的定时任务面板，支持多语言脚本、依赖管理与日志查看',
    base: '/baihu-panel/',
    lang: 'zh-CN',
    head: [
        ['link', { rel: 'stylesheet', href: 'https://fonts.loli.net/css2?family=Noto+Sans+SC:wght@400;500;700&family=Ubuntu+Mono:ital,wght@0,400;0,700;1,400;1,700&display=swap' }],
        ['script', {}, `if (navigator.userAgent.indexOf('Windows') !== -1) document.documentElement.classList.add('is-windows');`]
    ],
    themeConfig: {
        logo: '/logo.svg',
        nav: [
            { text: '快速开始', link: '/guide/introduction' },
            { text: '部署指南', link: '/guide/deployment' },
            { text: 'API 文档', link: '/guide/api' }
        ],

        sidebar: [
            {
                text: '基础指南',
                items: [
                    { text: '项目介绍', link: '/guide/introduction' },
                    { text: '部署说明', link: '/guide/deployment' },
                    { text: '开始使用', link: '/guide/getting-started' },
                    { text: 'API 文档', link: '/guide/api' }
                ]
            },
            {
                text: '功能指南',
                items: [
                    { text: '数据仪表', link: '/guide/dashboard' },
                    { text: '定时任务', link: '/guide/tasks' },
                    { text: '远程执行', link: '/guide/agents' },
                    { text: '面板互联', link: '/guide/interconnect' },
                    { text: '脚本管理', link: '/guide/scripts' },
                    { text: '执行历史', link: '/guide/history' },
                    { text: '变量机密', link: '/guide/environments' },
                    { text: '语言依赖', link: '/guide/languages' },
                    { text: '终端命令', link: '/guide/terminal' },
                    { text: '消息中心', link: '/guide/notify' },
                    { text: '仓库同步', link: '/guide/sync' },
                    { text: '命令行(CLI)', link: '/guide/cli' },
                    {
                        text: '脚本示例',
                        link: '/guide/examples/',
                        items: [
                            { text: '浏览器示例', link: '/guide/examples/browser' },
                            { text: '内置库示例', link: '/guide/examples/builtin' },
                            { text: 'Linux 环境依赖', link: '/guide/examples/linux-deps' }
                        ]
                    }
                ]
            },
            {
                text: '部署配置',
                items: [
                    { text: '系统配置', link: '/guide/configuration' },
                    { text: '前端定制(WebUI)', link: '/guide/webui' },
                    { text: '反向代理', link: '/guide/nginx' }
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
