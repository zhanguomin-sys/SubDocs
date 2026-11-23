const find = (id) => document.getElementById(id)
const app = () => window.go.main.App
const result = find('result')
const key = find('key')
const clipboardNotice = find('clipboard-notice')
const submit = find('submit')
const switchGPT62 = find('switch-gpt62')
const tabs = Array.from(document.querySelectorAll('[role=tab]'))
const toolButtons = Array.from(document.querySelectorAll('[data-tool]'))
const toolDialog = find('tool-panel')
const toolClose = find('tool-close')
const connectionResult = find('connection-result')
const keyHelpDialog = find('key-help-dialog')
const keyHelpClose = find('key-help-close')
const imageKey = find('image-key')
const names = { codex: 'Codex', 'claude-code': 'Claude' }
let target = 'codex'
let codexConfigured = false
let busy = false
let ready = false
let selectedTool = ''
let toolStatus
let lastWindowHeight = 0
let clipboardRequest = 0

function fitWindow() {
  if (!ready) return
  const contentHeight = document.querySelector('main').getBoundingClientRect().height
  const height = Math.ceil(contentHeight * window.devicePixelRatio)
  if (height === lastWindowHeight) return
  lastWindowHeight = height
  app().FitWindow(height).catch(() => { lastWindowHeight = 0 })
}

function canFillFromClipboard() {
  return ready && !busy && !document.hidden && !key.value.trim() &&
    !toolDialog.open && !connectionResult.open && !keyHelpDialog.open
}

async function fillKeyFromClipboard() {
  const request = ++clipboardRequest
  if (!canFillFromClipboard()) return
  try {
    const candidate = await app().ClipboardAPIKey()
    if (candidate && request === clipboardRequest && canFillFromClipboard()) {
      key.value = candidate
      clipboardNotice.hidden = false
    }
  } catch {
    // A busy or unavailable clipboard should not interrupt manual entry.
  }
}

function resetClipboardNotice() {
  clipboardRequest++
  clipboardNotice.hidden = true
}

key.addEventListener('input', resetClipboardNotice)
window.addEventListener('focus', fillKeyFromClipboard)
window.addEventListener('blur', () => { clipboardRequest++ })
document.addEventListener('visibilitychange', () => {
  if (document.hidden) clipboardRequest++
  else fillKeyFromClipboard()
})

function showResult(text, state = '') {
  result.textContent = text
  result.dataset.state = state
  result.hidden = !text
}

function showToolResult(text, state = '') {
  const box = find('tool-result')
  box.textContent = text
  box.dataset.state = state
  box.hidden = !text
}

function openConnectionDialog(dialog, button) {
  if (dialog.open) return
  dialog.onclose = () => {
    if (!dialog.open) button.focus()
  }
  dialog.showModal()
}

function showConnectionResult(button, text, state) {
  if (!text || !state) return
  find('connection-result-title').textContent = state === 'ok' ? '连接成功' : '连接失败'
  const message = find('connection-result-message')
  message.textContent = text
  message.dataset.state = state
  openConnectionDialog(connectionResult, button)
}

function isKeyRejected(error) {
  const message = typeof error === 'string' ? error : error?.message || ''
  return /请粘贴 MyAutoAPI 对应分组的完整 API Key|没有识别这个密钥|拒绝使用此密钥|已过期|已用完设定额度|当前不可用|尚未配置 MyAutoAPI|没有找到已保存的 GPT 分组 API Key/.test(message)
}

async function run(button, message, operation, announce = showResult) {
  if (busy || !ready) return
  clipboardRequest++
  busy = true
  button.disabled = true
  submit.disabled = true
  switchGPT62.disabled = true
  if (toolDialog.open) toolClose.setAttribute('aria-disabled', 'true')
  announce(message)
  try {
    announce(await operation() || '', 'ok')
  } catch (error) {
    const detail = typeof error === 'string' ? error : error?.message
    announce(detail?.trim() || '暂时无法完成，请重试。', 'error')
  } finally {
    busy = false
    button.disabled = false
    submit.disabled = false
    switchGPT62.disabled = false
    toolClose.removeAttribute('aria-disabled')
  }
}

async function refreshConnection() {
  const current = target
  const card = find('connection-card')
  const text = find('connection-text')
  const mark = find('connection-mark')
  const update = (state, message, icon) => {
    if (target !== current) return
    card.dataset.state = state
    text.textContent = message
    mark.textContent = icon
  }
  update('idle', '正在读取客户端配置…', 'i')
  try {
    const configured = await app().ConnectionStatus(current)
    if (current === 'codex') {
      codexConfigured = configured
      key.placeholder = configured
        ? '已配置 MyAutoAPI，留空以使用当前 GPT 分组 API Key'
        : '请粘贴 API 密钥页面 GPT 分组的 API Key'
    }
    if (configured) {
      update('connected', '已配置 MyAutoAPI', '✓')
    } else {
      update('idle', '尚未配置 MyAutoAPI', 'i')
    }
  } catch {
    if (current === 'codex') codexConfigured = false
    update('idle', '无法读取客户端配置', 'i')
  }
}

function selectTab(tab, focus = false) {
  if (busy || target === tab.dataset.target) return
  target = tab.dataset.target
  const isCodex = target === 'codex'
  for (const item of tabs) {
    item.setAttribute('aria-selected', String(item === tab))
    item.tabIndex = item === tab ? 0 : -1
  }
  if (focus) tab.focus()
  find('connect').setAttribute('aria-labelledby', tab.id)
  find('connection-label').textContent = names[target] + ' 状态'
  find('key-label').textContent = (isCodex ? 'GPT' : 'Claude') + ' 分组 API 密钥'
  key.placeholder = '请粘贴 API 密钥页面 ' + (isCodex ? 'GPT' : 'Claude') + ' 分组的 API Key'
  if (isCodex && codexConfigured) key.placeholder = '已配置 MyAutoAPI，留空以使用当前 GPT 分组 API Key'
  submit.textContent = '一键连接 ' + names[target]
  switchGPT62.hidden = !isCodex
  find('codex-tools').hidden = !isCodex
  key.value = ''
  resetClipboardNotice()
  showResult('')
  if (ready) {
    refreshConnection()
    fillKeyFromClipboard()
  }
}

for (const tab of tabs) {
  tab.addEventListener('click', () => selectTab(tab))
  tab.addEventListener('keydown', (event) => {
    if (['ArrowRight', 'ArrowLeft', 'Home', 'End'].includes(event.key)) {
      event.preventDefault()
      selectTab(event.key === 'Home' ? tabs[0] : event.key === 'End' ? tabs[1] : tabs.find(item => item !== tab), true)
    }
  })
}

const toolNames = { image: '安装绘图模型', models: '导入国产模型', locale: '设为简体中文' }

function closeTool() {
  if (busy) return
  selectedTool = ''
  toolStatus = undefined
  imageKey.value = ''
  toolDialog.close()
}

function renderToolStatus(status) {
  toolStatus = status
  const isImage = selectedTool === 'image'
  find('tool-installed').textContent = status.installedVersion ? 'V' + status.installedVersion : status.installed ? '已导入 · 版本未知' : '未导入'
  find('tool-included').textContent = 'V' + status.bundledVersion + (isImage ? '' : ' · ' + status.modelCount + ' 个模型')
  find('tool-latest').textContent = status.onlineVersion ? 'V' + status.onlineVersion : '未查询'
  find('tool-versions').hidden = false
  find('tool-description').textContent = isImage
    ? '安装 MyAutoAPI 绘图工具和 Codex 技能。首次安装请填写专门画图分组的密钥。安装成功后会自动关闭 ChatGPT/Codex。'
    : '将国产模型加到 Codex，保留现有 GPT、默认模型和密钥。导入成功后会自动关闭 ChatGPT/Codex。'
  find('tool-form').hidden = false
  find('image-key-section').hidden = !isImage
  imageKey.required = isImage && !status.installed
  imageKey.placeholder = status.installed ? '留空会保留原画图密钥' : '粘贴画图分组的密钥'
  const primary = find('tool-bundle')
  primary.textContent = status.installed ? '使用内置包' : isImage ? '安装内置版' : '导入内置模型'
  primary.hidden = !status.canUseBundled
  const online = find('tool-online')
  online.textContent = status.onlineVersion ? '安装 V' + status.onlineVersion : ''
  online.hidden = !status.onlineAvailable
}

for (const button of toolButtons) {
  button.addEventListener('click', () => {
    if (!ready || busy || toolDialog.open) return
    const kind = button.dataset.tool
    selectedTool = kind
    find('tool-title').textContent = toolNames[kind]
    const isLocale = kind === 'locale'
    find('tool-description').textContent = isLocale
      ? '仅在 Codex 已配置 MyAutoAPI 时才能设置。将关闭 Codex、更新中文语言缓存，再重新打开 Codex。请先保存工作。'
      : '正在读取本机和内置版本。'
    find('tool-help').dataset.guide = kind === 'image' ? 'images' : 'models'
    find('tool-versions').hidden = true
    find('tool-form').hidden = true
    find('locale-actions').hidden = !isLocale
    find('locale-confirm').hidden = false
    showToolResult('')
    button.focus()
    toolDialog.showModal()
    if (!isLocale) {
      run(button, '正在读取安装状态…', async () => {
        renderToolStatus(await app().GetToolStatus(kind))
        return ''
      }, showToolResult)
    }
  })
}

toolClose.addEventListener('click', closeTool)
toolDialog.addEventListener('cancel', (event) => {
  event.preventDefault()
  closeTool()
})

find('connection-result-close').addEventListener('click', () => connectionResult.close())
keyHelpClose.addEventListener('click', () => keyHelpDialog.close())

find('locale-confirm').addEventListener('click', (event) => {
  const button = event.currentTarget
  run(button, '', async () => {
    const label = button.textContent
    button.textContent = '正在设置…'
    try {
      const message = await app().SetChinese()
      button.hidden = true
      toolClose.focus()
      return message
    } finally {
      button.textContent = label
    }
  }, showToolResult)
})

find('tool-update').addEventListener('click', (event) => {
  run(event.currentTarget, '正在向 MyAutoAPI 查询版本…', async () => {
    renderToolStatus({ ...toolStatus, onlineVersion: '', onlineAvailable: false })
    const status = await app().CheckToolUpdate(selectedTool)
    renderToolStatus(status)
    return status.message
  }, showToolResult)
})

find('tool-form').addEventListener('submit', (event) => {
  event.preventDefault()
  const source = event.submitter?.dataset.source
  if (!source || (source === 'bundled' && !toolStatus?.canUseBundled) || (source === 'online' && !toolStatus?.onlineAvailable)) return
  const message = source === 'online' ? '正在下载并校验新版，随后导入…' : '正在从内置包导入…'
  run(event.submitter, message, async () => {
    const reply = await app().InstallTool(selectedTool, source, selectedTool === 'image' ? imageKey.value : '')
    imageKey.value = ''
    renderToolStatus(reply.status)
    return reply.message
  }, showToolResult)
})

window.addEventListener('load', async () => {
  try {
    const info = await app().Started()
    find('version').textContent = 'V' + info.version
    find('site').textContent = info.domain + ' ↗'
    find('qq-group').textContent = info.qqGroup
    ready = true
    submit.disabled = false
    switchGPT62.disabled = false
    new ResizeObserver(fitWindow).observe(document.querySelector('main'))
    window.addEventListener('resize', fitWindow)
    refreshConnection()
    fillKeyFromClipboard()
  } catch {
    showResult('接入助手没有完成启动，请关闭后重新打开。', 'error')
  }
})

find('configure').addEventListener('submit', (event) => {
  event.preventDefault()
  if (busy || !ready) return
  const isCodex = target === 'codex'
  const button = isCodex && event.submitter === switchGPT62 ? switchGPT62 : submit
  const action = button === switchGPT62 ? 'codex-gpt62' : target
  showResult('')

  run(button, '', async () => {
    const label = button.textContent
    button.textContent = '正在连接…'
    try {
      if (!isCodex && !key.value.trim()) {
        throw new Error('请粘贴 MyAutoAPI 网站「API 密钥」页中的 Claude 分组 API Key。')
      }
      if (isCodex && !key.value.trim() && !codexConfigured) {
        await refreshConnection()
        if (!codexConfigured) {
          openConnectionDialog(keyHelpDialog, button)
          return ''
        }
      }
      const message = await app().Configure(action, key.value)
      key.value = ''
      resetClipboardNotice()
      await refreshConnection()
      return message
    } catch (error) {
      if (isCodex && isKeyRejected(error)) {
        openConnectionDialog(keyHelpDialog, button)
        return ''
      }
      throw error
    } finally {
      button.textContent = label
    }
  }, (text, state) => showConnectionResult(button, text, state))
})

const bind = (id, message, operation) => {
  const button = find(id)
  button.addEventListener('click', () => run(button, message, operation))
}

bind('copy-group', '正在复制群号…', async () => {
  await app().CopyGroup()
  return '已复制 QQ 群号 ' + find('qq-group').textContent + '。'
})
for (const button of document.querySelectorAll('[data-guide]')) {
  button.addEventListener('click', () => {
    if (ready) app().OpenGuide(button.dataset.guide)
  })
}
bind('check', '正在检查 MyAutoAPI 更新…', async () => {
  find('download').hidden = true
  const update = await app().CheckUpdate()
  find('download').hidden = !update.available
  return update.message
})
bind('download', '请选择保存位置，然后等待下载和校验…', () => app().DownloadUpdate())
