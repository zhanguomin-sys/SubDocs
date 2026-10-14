import { defineConfig } from 'vitepress'
import { withSidebar } from 'vitepress-sidebar'

const config = defineConfig({
  lang: 'zh-CN',
  title: 'MyAutoAPI 中转站教程',
  description: 'MyAutoAPI 中转站接入、客户端配置与生图工具教程',
  sitemap: {
    hostname: 'https://docs.myautoapi.com'
  },
  rewrites: {
    'clients/codex-cc-switch.md': 'codex-cc-switch.md',
    'chatgpt-login-myautoapi.md': 'clients/chatgpt-login-myautoapi.md',
    'clients/infinite-canvas.md': 'images/infinite-canvas.md',
    'external/image-generation.md': 'images/image-generation.md',
    'images/codex-image-misunderstanding.md': 'images/codex-image-direct.md'
  },

  themeConfig: {
    nav: [
        { text: '快速开始', link: '/quick-start' },
        { text: 'CC Switch', link: '/codex-cc-switch' },
        { text: '客户端', link: '/clients/' },
        { text: '生图', link: '/images/' },
        { text: '遇到问题', link: '/errors/' },
        { text: '联系客服', link: '/contact' }
    ],

     search: {
        provider: 'local'
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
  excludeByGlobPattern: ['buy.md', 'models.md'],
  manualSortFileNameByPriority: [
    'quick-start.md',
    'codex-cc-switch.md',
    'image-video-group-image.md',
    'models.md',
    'clients',
    'kimi-code.md',
    'hy4.md',
    'codex.md',
    'chatgpt-login-myautoapi.md',
    'codex-manual-config.md',
    'vscode.md',
    'claude-code-desktop.md',
    'cc-switch.md',
    'jetbrains.md',
    'trae.md',
    'cherry-studio.md',
    'open-webui.md',
    'openclaw.md',
    'hermes.md',
    'librechat.md',
    'images',
    'infinite-canvas.md',
    'image-generation.md',
    'codex-image-direct.md',
    'codex-image.md',
    'external',
    'base-url.md',
    'compatibility.md',
    'user-agent.md',
    'git-practical.md',
    'codex-cli-internal-commands.md',
    'codex-skills.md',
    'codex-superpowers.md',
    'codex-advanced.md',
    'codex-log-sqlite.md',
    'codex-tools.md',
    'index.md',
    'errors',
    'contact.md'
  ]
})
