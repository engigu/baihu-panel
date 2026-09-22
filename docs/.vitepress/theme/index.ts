import DefaultTheme from 'vitepress/theme'
import type { App } from 'vue'
import RemoteMarkdown from './components/RemoteMarkdown.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }: { app: App }) {
    app.component('RemoteMarkdown', RemoteMarkdown)
  }
}

