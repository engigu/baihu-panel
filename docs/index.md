---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "白虎面板"
  text: "极致轻量、高性能的自动化任务调度平台"
  tagline: "采用 Go + Vue3 架构，专注于高性能与低系统开销。"
  image:
    src: /logo.svg
    alt: Baihu Logo
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/introduction
    - theme: alt
      text: 查看源码
      link: https://github.com/engigu/baihu-panel

features:
  - icon: 🚀
    title: 极致轻量
    details: Docker/Compose 一键部署，无需复杂配置，开箱即用，资源分配合理。
  - icon: ⏰
    title: 任务调度
    details: 支持标准 Cron 表达式，日志不落文件，规避频繁磁盘 IO 问题。
  - icon: 🛠️
    title: 多语言支持
    details: 深度集成 Mise，支持几乎所有主流编程语言的动态安装、多版本切换及依赖管理。
  - icon: 💻
    title: 在线管理
    details: 现代响应式 UI，集成在线编辑器、实时终端与 WebSocket 日志流。
  - icon: 🔔
    title: 消息推送
    details: 内置主流推送渠道（微信、钉钉、飞书、Telegram 等），支持系统级事件通知。
  - icon: 🔒
    title: 安全稳健
    details: 安全存储敏感配置，任务自动注入，登录防暴力破解，精细权限定制。
---
