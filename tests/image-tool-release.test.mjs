import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFile, readdir } from 'node:fs/promises'
import test from 'node:test'

const publicRoot = new URL('../docs/public/', import.meta.url)
const releaseRoot = new URL(
  'install/myautoapi-imagines-tool/releases/v0.8.1/',
  publicRoot,
)
const artifactNames = [
  'myautoapi-imagines-tool-v0.8.1-windows-amd64.exe',
  'myautoapi-imagines-tool-v0.8.1-darwin-arm64',
  'myautoapi-imagines-tool-v0.8.1-darwin-amd64',
  'myautoapi-imagines-tool-v0.8.1-linux-amd64',
]
const shellUrl = new URL('install/myautoapi-imagines-tool/install.sh', publicRoot)
const powershellUrl = new URL('install/myautoapi-imagines-tool/install.ps1', publicRoot)
const checksumUrl = new URL('SHA256SUMS', releaseRoot)
const headersUrl = new URL('_headers', publicRoot)
const entryRoot = new URL('install/myautoapi-imagines-tool/', publicRoot)
const releasesRoot = new URL('install/myautoapi-imagines-tool/releases/', publicRoot)

function parseChecksums(text) {
  return new Map(
    text.trim().split('\n').map((line) => {
      const match = line.match(/^([0-9a-f]{64})\s{2}(.+)$/)
      assert.ok(match, `invalid checksum line: ${line}`)
      return [match[2], match[1]]
    }),
  )
}

test('publishes matching V0.8.1 native artifacts, scripts and checksums', async () => {
  assert.deepEqual((await readdir(releaseRoot)).sort(), ['SHA256SUMS', ...artifactNames].sort())
  assert.deepEqual((await readdir(entryRoot)).sort(), ['install.ps1', 'install.sh', 'releases'])
  assert.deepEqual(
    (await readdir(releasesRoot)).sort(),
    ['v0.8.1'],
  )
  const [shell, powershell, checksum, ...artifacts] = await Promise.all([
    readFile(shellUrl, 'utf8'),
    readFile(powershellUrl, 'utf8'),
    readFile(checksumUrl, 'utf8'),
    ...artifactNames.map((name) => readFile(new URL(name, releaseRoot))),
  ])
  const checksums = parseChecksums(checksum)
  assert.equal(checksums.size, artifactNames.length)

  for (const [index, name] of artifactNames.entries()) {
    const digest = createHash('sha256').update(artifacts[index]).digest('hex')
    assert.equal(checksums.get(name), digest)
    const installScript = name.includes('windows') ? powershell : shell
    assert.match(installScript, new RegExp(digest))
  }
  for (const script of [shell, powershell]) {
    assert.match(script, /releases\/v0\.8\.1/)
    assert.doesNotMatch(script, /python|Expand-Archive|v0\.2-r1\.zip/i)
  }
  for (const script of [powershell]) {
    assert.match(script, /if\s*\(\$Architecture.*?-ne\s*"AMD64"\).*?throw\s*\(/s)
    assert.match(script, /catch\s*\{\s*throw\s*\(/s)
    assert.doesNotMatch(script, /^\s*exit\b/im)
    assert.match(script, /Download-WithPercent/)
    assert.match(script, /Write-Progress/)
    assert.match(script, /-PercentComplete\s+\$Percent/)
    assert.doesNotMatch(script, /Invoke-WebRequest/)
  }
  const combined = [shell, powershell].join('\n')
  assert.match(combined, /darwin-amd64/)
  assert.match(combined, /linux-amd64/)
  assert.match(shell, /Darwin\/x86_64/)
  assert.match(shell, /sysctl\.proc_translated/)
  assert.match(shell, /expected_binary_format/)
  assert.doesNotMatch(combined, /windows-arm64|linux-arm64/)
})

test('publishes explicit content types and cache policy', async () => {
  const headers = await readFile(headersUrl, 'utf8')

  for (const name of ['install.ps1', 'install.sh']) {
    assert.match(
      headers,
      new RegExp(`/install/myautoapi-imagines-tool/${name.replace('.', '\\.')}` +
        '\\s+Content-Type:\\s*text/plain;\\s*charset=utf-8', 'i'),
    )
  }
  assert.match(
    headers,
    /\/install\/myautoapi-imagines-tool\/releases\/v0\.8\.1\/myautoapi-imagines-tool-\*\s+Content-Type:\s*application\/octet-stream\s+Cache-Control:\s*public,\s*max-age=31536000,\s*immutable/i,
  )
  assert.match(
    headers,
    /\/install\/myautoapi-imagines-tool\/releases\/v0\.8\.1\/SHA256SUMS\s+Content-Type:\s*text\/plain;\s*charset=utf-8\s+Cache-Control:\s*public,\s*max-age=31536000,\s*immutable/i,
  )
})
