import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const imageToolDocSource = readFileSync(
  new URL('../docs/image-video-group-image.md', import.meta.url),
  'utf8',
)
const quickStartSource = readFileSync(new URL('../docs/quick-start.md', import.meta.url), 'utf8')
const configSource = readFileSync(new URL('../docs/.vitepress/config.mts', import.meta.url), 'utf8')
const customCssSource = readFileSync(new URL('../docs/.vitepress/theme/custom.css', import.meta.url), 'utf8')
const ccSwitchSource = readFileSync(new URL('../docs/codex-cc-switch.md', import.meta.url), 'utf8')

test('puts the image tool installation on one three-step customer path', () => {
  const stepOne = imageToolDocSource.indexOf('## 1. 创建专门画图分组的 API Key')
  const stepTwo = imageToolDocSource.indexOf('## 2. 在终端安装生图工具')
  const stepThree = imageToolDocSource.indexOf('## 3. 打开 Codex 开始生图')

  assert.ok(stepOne > 0)
  assert.ok(stepTwo > stepOne)
  assert.ok(stepThree > stepTwo)
  assert.match(imageToolDocSource, /专门画图分组/)
  assert.match(imageToolDocSource, /\/images\/画图\/画图专用分组\.jpg/)
  assert.match(imageToolDocSource, /irm https:\/\/docs\.myautoapi\.com\/install\/myautoapi-imagines-tool\/install\.ps1 \| iex/)
  assert.match(imageToolDocSource, /curl -fsSL https:\/\/docs\.myautoapi\.com\/install\/myautoapi-imagines-tool\/install\.sh \| bash/)
  assert.match(imageToolDocSource, /已经打开时不用退出应用，直接新建任务/)
  assert.match(imageToolDocSource, /介绍一下中转站生图工具怎么使用，有哪些模型/)
  assert.match(imageToolDocSource, /生成成功后，图片会直接显示在当前 Codex 对话中/)
})

test('keeps secondary information collapsed and retains the DIY Python path at the end', () => {
  const customerPath = imageToolDocSource.slice(
    0,
    imageToolDocSource.indexOf('不用一键安装：让 Codex 创建 Python 脚本'),
  )

  assert.ok((imageToolDocSource.match(/<details>/g) || []).length >= 6)
  assert.match(imageToolDocSource, /查看常用说法/)
  assert.match(imageToolDocSource, /查看支持的模型/)
  assert.match(imageToolDocSource, /查看更新、安装位置和卸载方式/)
  assert.match(imageToolDocSource, /安装或生图失败先看这里/)
  const diySection = imageToolDocSource.indexOf('不用一键安装：让 Codex 创建 Python 脚本')
  const troubleshootingSection = imageToolDocSource.indexOf('安装或生图失败先看这里')
  assert.ok(diySection > troubleshootingSection)
  assert.match(imageToolDocSource, /在当前文件夹创建一个可运行的单文件生图脚本/)
  assert.match(imageToolDocSource, /自建 Python 脚本的通用规则/)
  assert.match(imageToolDocSource, /创建或更新当前项目根目录的 AGENTS\.md/)
  assert.doesNotMatch(customerPath, /POST https:\/\/api\.myautoapi\.com/)
})

test('documents the verified Nano Banana 2 Responses request contract', () => {
  assert.match(imageToolDocSource, /POST https:\/\/api\.myautoapi\.com\/v1\/responses/)
  assert.match(imageToolDocSource, /Authorization: Bearer <API_KEY>/)
  assert.match(imageToolDocSource, /"stream": false/)
  assert.match(imageToolDocSource, /"type": "input_text"/)
  assert.match(imageToolDocSource, /"type": "input_image"/)
  assert.match(imageToolDocSource, /`image_size`/)
  assert.match(imageToolDocSource, /`aspect_ratio`/)
  assert.match(imageToolDocSource, /`quality`、`size`、`output_format`.*不使用/)
  assert.match(imageToolDocSource, /同一画面要求 3 张时发送 3 个单张请求/)
  assert.match(imageToolDocSource, /PNG、JPEG、WebP 各 1 张以及 2 张、4 张不同图片已验证/)
  assert.match(imageToolDocSource, /data:image\/png;base64/)
  assert.match(imageToolDocSource, /16:9、21:9、9:21/)
  assert.match(imageToolDocSource, /本地工具最多 14 张；14 张远端请求尚未形成支持结论/)
  assert.match(imageToolDocSource, /只扫描 `output`/)
})

test('keeps the three primary sidebar paths in the requested order and visually numbered', () => {
  assert.match(
    configSource,
    /'quick-start\.md',\s*'codex-cc-switch\.md',\s*'codex-models\.md',\s*'image-video-group-image\.md',[\s\S]*?'clients'/,
  )
  assert.match(configSource, /frontmatterTitleFieldName:\s*'sidebarTitle'/)
  assert.match(ccSwitchSource, /^sidebarTitle:\s*CC Switch 接入$/m)
  assert.match(customCssSource, /\.link\[href='\/quick-start\.html'\]::before\s*\{[\s\S]*?content:\s*'1'/)
  assert.match(customCssSource, /\.link\[href='\/codex-cc-switch\.html'\]::before\s*\{[\s\S]*?content:\s*'2'/)
  assert.match(customCssSource, /\.link\[href='\/image-video-group-image\.html'\]::before\s*\{[\s\S]*?content:\s*'3'/)
  assert.match(customCssSource, /\.VPSidebar \.link\[href='\/codex-cc-switch\.html'\] \.text/)
  assert.match(customCssSource, /white-space:\s*nowrap/)
  assert.match(quickStartSource, /\[安装生图工具\]\(\/image-video-group-image\)/)
})
