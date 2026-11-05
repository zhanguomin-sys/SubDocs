import { defineConfig } from 'vitepress'
import { withSidebar } from 'vitepress-sidebar'

const config = defineConfig({
  lang: 'zh-CN',
  title: 'MyAutoAPI 中转站教程',
  description: 'MyAutoAPI 中转站接入、客户端配置与生图工具教程',
  sitemap: { hostname: 'https://docs.myautoapi.com' },
  themeConfig: {
    nav: [
        { text: '快速开始', link: '/quick-start' },
        { text: 'Codex', link: '/codex/' },
        { text: 'Claude', link: '/claude/' },
        { text: '国产工具', link: '/domestic/' },
        { text: '其他工具', link: '/clients/' },
        { text: '生图', link: '/images/' },
        { text: '遇到问题', link: '/errors/' },
        { text: '联系客服', link: '/contact' }
    ],

     search: {
        provider: 'local',
        options: {
          miniSearch: {
            options: {
              // 默认分词只按空格和标点切开，中文整句会变成一个词；改按中文词语切分。
              tokenize: (text) => {
                if (typeof Intl === 'undefined' || !Intl.Segmenter) return text.split(/[\s\p{P}]+/u).filter(Boolean)
                const segmenter = new Intl.Segmenter('zh-CN', { granularity: 'word' })
                return Array.from(segmenter.segment(text)).filter((part) => part.isWordLike).map((part) => part.segment)
              }
            }
          }
        }
    }
}

})

export default withSidebar(config, {
  documentRootPath: 'docs',
  useTitleFromFileHeading: true,
  useTitleFromFrontmatter: true,
  frontmatterTitleFieldName: 'sidebarTitle',
  useFolderTitleFromIndexFile: true,
  useFolderLinkFromIndexFile: true,
  collapsed: true,
  includeRootIndexFile: false,
  includeFolderIndexFile: false,
  excludeByGlobPattern: ['buy.md', 'models.md', 'radar.md'],
  manualSortFileNameByPriority: [
    'quick-start.md',
    'codex-cc-switch.md',
    'image-video-group-image.md',
    'codex-models.md',
    'models.md',
    'codex',
    'manual-config.md',
    'chatgpt-login.md',
    'commands.md',
    'advanced.md',
    'skills.md',
    'superpowers.md',
    'tools.md',
    'git.md',
    'log-sqlite.md',
    'claude',
    'claude-code.md',
    'desktop.md',
    'domestic',
    'kimi-code.md',
    'qwen-code.md',
    'workbuddy.md',
    'trae.md',
    'cherry-studio.md',
    'clients',
    'vscode.md',
    'jetbrains.md',
    'gemini-cli.md',
    'open-webui.md',
    'librechat.md',
    'openclaw.md',
    'hermes.md',
    'images',
    'infinite-canvas.md',
    'image-generation.md',
    'codex-image.md',
    'params.md',
    'external',
    'base-url.md',
    'compatibility.md',
    'user-agent.md',
    'index.md',
    'errors',
    'contact.md'
  ]
})
