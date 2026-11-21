const find = (id) => document.getElementById(id)
const status = find('status')
const version = find('version')
const key = find('key')
const check = find('check')
const download = find('download')
const api = () => window.go.main.App
let busy = false

const run = async (button, message, operation) => {
  if (busy) return
  busy = true
  button.disabled = true
  status.classList.remove('error')
  status.textContent = message
  try {
    status.textContent = (await operation()) || ''
  } catch (error) {
    status.classList.add('error')
    status.textContent = typeof error === 'string' ? error : error.message || '暂时无法完成，请重试。'
  } finally {
    busy = false
    button.disabled = false
  }
}

window.addEventListener('load', async () => {
  try {
    version.textContent = 'V' + await api().Started()
  } catch {
    status.textContent = '接入助手没有完成启动，请关闭后重新打开。'
  }
})

find('configure').addEventListener('submit', (event) => {
  event.preventDefault()
  run(event.submitter, '正在验证 Key…', async () => {
    const message = await api().Configure(find('target').value, key.value)
    key.value = ''
    return message
  })
})

find('target').addEventListener('change', () => { key.value = '' })

find('claude').addEventListener('click', (event) => {
  run(event.currentTarget, '请选择保存位置…', () => api().SaveClaudeTemplate())
})
find('locale').addEventListener('click', (event) => {
  run(event.currentTarget, '正在保存语言偏好…', () => api().SetChinese())
})
for (const button of document.querySelectorAll('[data-guide]')) {
  button.addEventListener('click', () => api().OpenGuide(button.dataset.guide))
}
check.addEventListener('click', () => run(check, '正在检查更新…', async () => {
  download.hidden = true
  const update = await api().CheckUpdate()
  download.hidden = !update.available
  return update.message
}))
download.addEventListener('click', () => run(download, '请选择保存位置，然后等待下载和校验…', () => api().DownloadUpdate()))
