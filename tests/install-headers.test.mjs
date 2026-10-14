import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const headersPath = new URL('../docs/public/_headers', import.meta.url)

async function readHeaders() {
  return readFile(headersPath, 'utf8').catch(() => '')
}

test('serves the PowerShell installer as UTF-8 text', async () => {
  const headers = await readHeaders()

  assert.match(
    headers,
    /\/install\/myautoapi-codex-installer\/install\.ps1\s+Content-Type:\s*text\/plain;\s*charset=utf-8/i,
  )
})

test('serves the shell installer as UTF-8 text', async () => {
  const headers = await readHeaders()

  assert.match(
    headers,
    /\/install\/myautoapi-codex-installer\/install\.sh\s+Content-Type:\s*text\/plain;\s*charset=utf-8/i,
  )
})

test('serves both restore scripts as UTF-8 text', async () => {
  const headers = await readHeaders()

  for (const extension of ['ps1', 'sh']) {
    assert.match(
      headers,
      new RegExp(
        `/install/myautoapi-codex-installer/restore\\.${extension}\\s+` +
          'Content-Type:\\s*text/plain;\\s*charset=utf-8',
        'i',
      ),
    )
  }
})
