// 扩展 Window 接口，添加后端注入的全局变量
declare global {
  interface Window {
    __BASE_URL__?: string
    __API_VERSION__?: string
  }
}

export {}
