import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const pagePath = new URL('../docs/radar.md', import.meta.url)
const configPath = new URL('../docs/.vitepress/config.mts', import.meta.url)

test('radar page only frames the radar site without theme chrome', async () => {
  const page = await readFile(pagePath, 'utf8')

  assert.match(page, /^layout:\s*false$/m)
  assert.match(page, /<iframe[^>]+src="https:\/\/radar\.usegoodai\.com\/"/)
  assert.equal(page.match(/<iframe/g).length, 1)
  assert.match(page, /name: robots\s+content: noindex/)
  assert.match(page, /^search:\s*false$/m)
})

test('radar page is kept out of the sidebar', async () => {
  const config = await readFile(configPath, 'utf8')

  assert.match(config, /excludeByGlobPattern:\s*\[[^\]]*'radar\.md'/)
})
