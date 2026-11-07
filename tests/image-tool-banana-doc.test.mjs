import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const imageToolDocSource = readFileSync(
  new URL('../docs/image-video-group-image.md', import.meta.url),
  'utf8',
)
const paramsSource = readFileSync(new URL('../docs/images/params.md', import.meta.url), 'utf8')
const quickStartSource = readFileSync(new URL('../docs/quick-start.md', import.meta.url), 'utf8')
const configSource = readFileSync(new URL('../docs/.vitepress/config.mts', import.meta.url), 'utf8')
const customCssSource = readFileSync(new URL('../docs/.vitepress/theme/custom.css', import.meta.url), 'utf8')
const ccSwitchSource = readFileSync(new URL('../docs/codex-cc-switch.md', import.meta.url), 'utf8')

test('puts the image tool installation on one three-step customer path', () => {
  const stepOne = imageToolDocSource.indexOf('## 1. 创建专门画图分组的 API Key')
  const stepTwo = imageToolDocSource.indexOf('## 2. 安装生图工具')
  const stepThree = imageToolDocSource.indexOf('## 3. 打开 Codex 开始生图')

  assert.ok(stepOne > 0)
  assert.ok(stepTwo > stepOne)
  assert.ok(stepThree > stepTwo)
  assert.match(imageToolDocSource, /专门画图分组/)
  assert.match(imageToolDocSource, /\/images\/画图\/画图专用分组\.jpg/)
  assert.match(imageToolDocSource, /irm https:\/\/docs\.myautoapi\.com\/install\/myautoapi-imagines-tool\/install\.ps1 \| iex/)
  assert.match(imageToolDocSource, /curl -fsSL https:\/\/docs\.myautoapi\.com\/install\/myautoapi-imagines-tool\/install\.sh \| bash/)
  assert.match(imageToolDocSource, /完全退出 Codex 再重新打开（放行规则在启动时加载），新建任务/)
  assert.match(imageToolDocSource, /图片会直接显示在对话中/)
})

test('keeps secondary information collapsed and retains the DIY Python path at the end', () => {
  const diySection = imageToolDocSource.indexOf('不用一键安装：让 Codex 创建 Python 脚本')
  const troubleshootingSection = imageToolDocSource.indexOf('## 安装或生图失败')
  const customerPath = imageToolDocSource.slice(0, diySection)

  assert.match(imageToolDocSource, /查看常用说法/)
  assert.match(imageToolDocSource, /查看支持的模型/)
  assert.match(imageToolDocSource, /更新、卸载和文件位置/)
  assert.ok(troubleshootingSection > 0)
  assert.ok(diySection > troubleshootingSection)
  assert.match(imageToolDocSource, /在当前文件夹创建一个可运行的单文件生图脚本/)
  assert.match(imageToolDocSource, /#### 自建脚本规则/)
  assert.match(imageToolDocSource, /创建或更新当前项目根目录的 AGENTS\.md/)
  assert.match(imageToolDocSource, /https:\/\/docs\.myautoapi\.com\/images\/params\.html/)
  assert.doesNotMatch(customerPath, /POST https:\/\/api\.myautoapi\.com/)
})

test('documents the verified Nano Banana Responses request contract on the params page', () => {
  assert.match(paramsSource, /`POST \/v1\/responses`/)
  assert.match(paramsSource, /Authorization: Bearer <API_KEY>/)
  assert.match(paramsSource, /"stream": false/)
  assert.match(paramsSource, /"type": "input_text"/)
  assert.match(paramsSource, /"type": "input_image"/)
  assert.match(paramsSource, /data:image\/png;base64/)
  assert.match(paramsSource, /`response_format\.image_size`/)
  assert.match(paramsSource, /`response_format\.aspect_ratio`/)
  assert.match(paramsSource, /不使用 `quality`、`size` 和 `output_format`/)
  assert.match(paramsSource, /每张图单独发一次请求/)
  assert.match(paramsSource, /只在 `output` 里找/)
})

test('keeps the V0.9.1 upgrade scoped to MyAutoAPI image tooling', () => {
  assert.match(imageToolDocSource, /V0\.9\.1/)
  assert.match(imageToolDocSource, /\$CODEX_HOME/)
  assert.match(imageToolDocSource, /rules\/myautoapi-imagines-tool\.rules/)
  assert.match(imageToolDocSource, /安装或更新后完全退出 Codex 再重新打开/)
  assert.match(imageToolDocSource, /当前支持上表 10 个模型/)
  assert.match(imageToolDocSource, /不再调用 `view_image` 重新读取/)
  assert.doesNotMatch(imageToolDocSource, /工具合集|QuickConnect|quick-connect|\/download\//i)
  assert.doesNotMatch(imageToolDocSource + paramsSource, /usegoodai|本站已验证/i)
})

test('keeps the four primary sidebar paths in the requested order and visually numbered', () => {
  assert.match(
    configSource,
    /'quick-start\.md',\s*'codex-cc-switch\.md',\s*'image-video-group-image\.md',\s*'codex-models\.md',[\s\S]*?'clients'/,
  )
  assert.match(configSource, /frontmatterTitleFieldName:\s*'sidebarTitle'/)
  assert.match(ccSwitchSource, /^sidebarTitle:\s*CC Switch 接入$/m)
  assert.match(customCssSource, /\.link\[href='\/quick-start\.html'\]::before\s*\{[\s\S]*?content:\s*'1\.'/)
  assert.match(customCssSource, /\.link\[href='\/codex-cc-switch\.html'\]::before\s*\{[\s\S]*?content:\s*'2\.'/)
  assert.match(customCssSource, /\.link\[href='\/image-video-group-image\.html'\]::before\s*\{[\s\S]*?content:\s*'3\.'/)
  assert.match(customCssSource, /\.link\[href='\/codex-models\.html'\]::before\s*\{[\s\S]*?content:\s*'4\.'/)
  assert.match(customCssSource, /\.VPSidebar \.link\[href='\/codex-cc-switch\.html'\] \.text/)
  assert.match(customCssSource, /white-space:\s*nowrap/)
  assert.match(quickStartSource, /\[安装生图工具\]\(\/image-video-group-image\)/)
})
