import DefaultTheme from 'vitepress/theme'
import BookmarkButton from './components/BookmarkButton.vue'
import CopyButton from './components/CopyButton.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('BookmarkButton', BookmarkButton)
    app.component('CopyButton', CopyButton)
  }
}
