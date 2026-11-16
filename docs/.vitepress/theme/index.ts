import { inBrowser } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import BookmarkButton from './components/BookmarkButton.vue'
import CopyButton from './components/CopyButton.vue'
import './custom.css'

// 旧 /clients/claude-code-desktop 页里的 Claude Code CLI 章节已移到 /claude/claude-code。
// _redirects 只能跳页面，旧链接带的章节锚点在这里转到新页面对应章节。
const MOVED_ANCHORS: Record<string, Record<string, string>> = {
  '/claude/desktop': {
    'claude-code-cli-ccswitch': '/claude/claude-code#claude-code-cli-ccswitch',
    'claude-code-cli-manual': '/claude/claude-code#claude-code-cli-manual',
    '配置-claude-code-cli': '/claude/claude-code',
    'claude-模型怎么切换': '/claude/claude-code#_2-重新打开-claude-code-测试'
  }
}

function redirectMovedAnchor() {
  const path = location.pathname.replace(/\.html$/, '')
  const anchor = decodeURIComponent(location.hash.slice(1))
  const target = MOVED_ANCHORS[path]?.[anchor]
  if (target) location.replace(target)
}

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('BookmarkButton', BookmarkButton)
    app.component('CopyButton', CopyButton)
    if (inBrowser) redirectMovedAnchor()
  }
}
