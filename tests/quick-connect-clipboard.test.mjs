import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'

const source = readFileSync(new URL('../myautoapi-codex-installer/quick-connect/frontend/app.js', import.meta.url), 'utf8')
const flush = () => new Promise(resolve => setImmediate(resolve))
const fixtures = ['sk-clipboard-gpt-fixture', 'sk-clipboard-claude-fixture']

function element(extra = {}) {
  const events = new Map()
  return {
    value: '', dataset: {}, hidden: true, open: false,
    setAttribute() {}, removeAttribute() {},
    addEventListener(name, listener) { events.set(name, listener) },
    emit(name, event) { return events.get(name)?.(event) },
    ...extra
  }
}

async function launch(readClipboard) {
  const fields = new Map()
  const codex = element({ id: 'tab-codex', dataset: { target: 'codex' } })
  const claude = element({ id: 'tab-claude', dataset: { target: 'claude-code' } })
  const document = element({
    hidden: false,
    getElementById(id) {
      if (!fields.has(id)) fields.set(id, element())
      return fields.get(id)
    },
    querySelectorAll(selector) { return selector === '[role=tab]' ? [codex, claude] : [] },
    querySelector() { return element() }
  })
  let reads = 0
  let configures = 0
  const window = element({
    go: { main: { App: {
      Started: async () => ({}),
      ConnectionStatus: async () => false,
      ClipboardAPIKey: () => readClipboard(reads++),
      Configure: async () => { configures++ }
    } } }
  })
  runInNewContext(source, { document, window, ResizeObserver: class { observe() {} } })
  await window.emit('load')
  await flush()
  return {
    document, window, claude, get: id => document.getElementById(id),
    clipboardReads: () => reads, connectCalls: () => configures
  }
}

test('fills a masked field at startup, preserves typed input, and never starts a connection', async () => {
  const ui = await launch(async () => fixtures[0])
  const field = ui.get('key')
  assert.equal(field.value, fixtures[0])
  assert.equal(ui.get('clipboard-notice').hidden, false)
  field.value = 'sk-user-entered-fixture'
  field.emit('input')
  await ui.window.emit('focus')
  assert.equal(field.value, 'sk-user-entered-fixture')
  assert.equal(ui.get('clipboard-notice').hidden, true)
  assert.equal(ui.clipboardReads(), 1)
  assert.equal(ui.connectCalls(), 0)
})

test('a late clipboard reply cannot overwrite the selection or re-fill after an edit', async () => {
  const pending = []
  const ui = await launch(() => new Promise(resolve => pending.push(resolve)))
  ui.claude.emit('click')
  pending[1](fixtures[1])
  await flush()
  pending[0](fixtures[0])
  await flush()
  const field = ui.get('key')
  assert.equal(field.value, fixtures[1])
  field.value = ''
  field.emit('input')
  const returned = ui.window.emit('focus')
  field.value = 'user edit'
  field.emit('input')
  field.value = ''
  field.emit('input')
  pending[2](fixtures[1])
  await returned
  assert.equal(field.value, '')
  assert.equal(ui.connectCalls(), 0)
})

test('does not read in the background and can retry an unavailable clipboard when visible', async () => {
  const ui = await launch(async i => {
    if (i === 0) throw new Error('clipboard busy')
    return fixtures[0]
  })
  assert.equal(ui.get('key').value, '')
  ui.document.hidden = true
  ui.document.emit('visibilitychange')
  await ui.window.emit('focus')
  assert.equal(ui.clipboardReads(), 1)
  ui.document.hidden = false
  ui.document.emit('visibilitychange')
  await flush()
  assert.equal(ui.get('key').value, fixtures[0])
})
