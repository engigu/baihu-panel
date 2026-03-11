import { createApp } from 'vue'
import { install as VueMonacoEditorPlugin } from '@guolao/vue-monaco-editor'
import './assets/index.css'
import 'vue-sonner/style.css'
import App from './App.vue'
import router from './router'

window.addEventListener('vite:preloadError', () => {
  console.log('Detected vite preload error. Reloading page...')
  window.location.reload()
})

const BASE_URL = (window as any).__BASE_URL__ || ''

createApp(App)
  .use(router)
  .use(VueMonacoEditorPlugin, {
    paths: {
      vs: BASE_URL + '/vs'
    }
  })
  .mount('#app')
