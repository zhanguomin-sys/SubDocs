import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { spawnSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const entryRoot = new URL('../docs/public/install/myautoapi-imagines-tool/', import.meta.url)
const portRelease = 'releases/v0.9.2-upstream-c28a911/'
const platforms = {
  'windows/amd64': 'windows-amd64.exe',
  'darwin/arm64': 'darwin-arm64',
  'darwin/amd64': 'darwin-amd64',
  'linux/amd64': 'linux-amd64',
}
const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex')
const nativePlatform = (process.platform === 'win32' ? 'windows' : process.platform) + '/' + (process.arch === 'x64' ? 'amd64' : process.arch)

test('native updater metadata names only verified MyAutoAPI artifacts', async () => {
  const manifest = JSON.parse(await readFile(new URL('latest.json', entryRoot), 'utf8'))
  assert.equal(manifest.version, '0.9.2')
  assert.equal(manifest.release, 'V0.9.2')
  assert.deepEqual(Object.keys(manifest.files).sort(), Object.keys(platforms).sort())
  for (const [platform, suffix] of Object.entries(platforms)) {
    const file = manifest.files[platform]
    assert.equal(typeof file.path, 'string')
    assert.doesNotMatch(file.path, /usegoodai|https?:|^\/|\\|(?:^|\/)\.\.(?:\/|$)/i)
    assert.ok(file.path.endsWith('/myautoapi-imagines-tool-' + suffix) || file.path.endsWith('/myautoapi-imagines-tool-v0.9.2-' + suffix))
    const bytes = await readFile(new URL(file.path, entryRoot))
    assert.equal(file.sha256, sha256(bytes), platform + ' checksum')
    assert.equal(file.size, bytes.length, platform + ' size')
  }
  const headers = await readFile(new URL('../docs/public/_headers', import.meta.url), 'utf8')
  assert.match(headers, /\/install\/myautoapi-imagines-tool\/latest\.json\s+Content-Type:\s*application\/json;\s*charset=utf-8\s+Cache-Control:\s*no-store/i)
})

test('all native release and stable artifacts contain the MyAutoAPI endpoints', async () => {
  for (const suffix of Object.values(platforms)) {
    const release = await readFile(new URL(portRelease + 'myautoapi-imagines-tool-v0.9.2-' + suffix, entryRoot))
    const current = await readFile(new URL('current/myautoapi-imagines-tool-' + suffix, entryRoot))
    assert.equal(sha256(current), sha256(release), suffix + ' stable and release bytes')
    const content = current.toString('latin1')
    assert.equal(/usegoodai/i.test(content), false, suffix + ' retains an upstream identity')
    for (const expected of [
      'https://api.myautoapi.com/v1/models',
      'https://docs.myautoapi.com/install/myautoapi-imagines-tool/latest.json',
      'myautoapi-imagines-tool',
    ]) assert.ok(content.includes(expected), suffix + ' is missing ' + expected)
  }
})

function verifyMachOSignature(bytes, label) {
  assert.equal(bytes.readUInt32LE(0), 0xfeedfacf, label + ' Mach-O magic')
  const commandCount = bytes.readUInt32LE(16)
  const commandsEnd = 32 + bytes.readUInt32LE(20)
  assert.ok(commandsEnd <= bytes.length, label + ' load command bounds')
  let offset = 32
  for (let index = 0; index < commandCount; index++) {
    assert.ok(offset + 8 <= commandsEnd, label + ' truncated load command')
    const command = bytes.readUInt32LE(offset)
    const length = bytes.readUInt32LE(offset + 4)
    assert.ok(length >= 8 && offset + length <= commandsEnd, label + ' load command length')
    if (command !== 0x1d) {
      offset += length
      continue
    }
    assert.ok(length >= 16, label + ' code signature command length')
    const signatureStart = bytes.readUInt32LE(offset + 8)
    const signatureLength = bytes.readUInt32LE(offset + 12)
    assert.ok(signatureStart + signatureLength <= bytes.length, label + ' signature bounds')
    assert.equal(bytes.readUInt32BE(signatureStart), 0xfade0cc0, label + ' signature container')
    const containerLength = bytes.readUInt32BE(signatureStart + 4)
    const slotCount = bytes.readUInt32BE(signatureStart + 8)
    assert.ok(containerLength >= 20 && containerLength <= signatureLength, label + ' container bounds')
    assert.equal(slotCount, 1, label + ' ad-hoc signature slot count')
    assert.equal(bytes.readUInt32BE(signatureStart + 12), 0, label + ' CodeDirectory slot')
    const directory = signatureStart + bytes.readUInt32BE(signatureStart + 16)
    assert.ok(directory >= signatureStart + 20 && directory + 44 <= signatureStart + containerLength, label + ' directory bounds')
    assert.equal(bytes.readUInt32BE(directory), 0xfade0c02, label + ' CodeDirectory magic')
    const directoryLength = bytes.readUInt32BE(directory + 4)
    assert.ok(directory + directoryLength <= signatureStart + containerLength, label + ' directory length')
    assert.ok(bytes.readUInt32BE(directory + 12) & 2, label + ' ad-hoc flag')
    const hashes = bytes.readUInt32BE(directory + 16)
    const specialSlots = bytes.readUInt32BE(directory + 24)
    const pages = bytes.readUInt32BE(directory + 28)
    const limit = bytes.readUInt32BE(directory + 32)
    const hashSize = bytes[directory + 36]
    const hashType = bytes[directory + 37]
    const pageExponent = bytes[directory + 39]
    assert.equal(specialSlots, 0, label + ' special slots')
    assert.equal(hashSize, 32, label + ' SHA-256 size')
    assert.equal(hashType, 2, label + ' SHA-256 type')
    assert.ok(pageExponent > 0 && pageExponent <= 20, label + ' page size exponent')
    const pageSize = 2 ** pageExponent
    assert.equal(limit, signatureStart, label + ' signed code limit')
    assert.equal(pages, Math.ceil(limit / pageSize), label + ' signed page count')
    assert.ok(hashes >= 44 && hashes + pages * hashSize <= directoryLength, label + ' hash table bounds')
    for (let page = 0; page < pages; page++) {
      const begin = page * pageSize
      const actual = sha256(bytes.subarray(begin, Math.min(begin + pageSize, limit)))
      const expected = bytes.subarray(directory + hashes + page * hashSize, directory + hashes + (page + 1) * hashSize).toString('hex')
      assert.equal(actual, expected, label + ' code page ' + page)
    }
    return pages
  }
  return 0
}

test('Apple Silicon ad-hoc signature hashes match all ported code pages', async () => {
  for (const suffix of ['darwin-arm64', 'darwin-amd64']) {
    const bytes = await readFile(new URL('current/myautoapi-imagines-tool-' + suffix, entryRoot))
    const pages = verifyMachOSignature(bytes, suffix)
    if (suffix === 'darwin-arm64') assert.ok(pages > 0, 'Apple Silicon signature must remain present')
    else assert.equal(pages, 0, 'Intel upstream artifact must remain unsigned')
  }
})

test('macOS codesign accepts the ported signed artifact', async (t) => {
  const probe = spawnSync('codesign', ['--version'], { encoding: 'utf8', windowsHide: true })
  if (probe.error?.code === 'ENOENT') return t.skip('codesign is available only on macOS')
  assert.ifError(probe.error)
  for (const suffix of ['darwin-arm64', 'darwin-amd64']) {
    const binary = fileURLToPath(new URL('current/myautoapi-imagines-tool-' + suffix, entryRoot))
    // Intel Go binaries can be unsigned; do not claim they carry a signature.
    if (!verifyMachOSignature(await readFile(binary), suffix)) continue
    const result = spawnSync('codesign', ['--verify', '--deep', '--strict', binary], { encoding: 'utf8', windowsHide: true })
    assert.ifError(result.error)
    assert.equal(result.status, 0, suffix + ' code signature verification failed')
  }
})

test('ported native installer stays in isolated CODEX_HOME and preserves other tools', { timeout: 30000 }, async (t) => {
  const suffix = platforms[nativePlatform]
  if (!suffix) return t.skip('no published native artifact for ' + nativePlatform)
  const program = fileURLToPath(new URL('current/myautoapi-imagines-tool-' + suffix, entryRoot))
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'myautoapi-port-check-'))
  t.after(() => rm(temporary, { recursive: true, force: true }))
  const home = path.join(temporary, '独立 测试目录')
  const foreign = path.join(home, 'tools', 'unrelated-image-tool')
  await mkdir(foreign, { recursive: true })
  await writeFile(path.join(foreign, 'sentinel.txt'), 'keep unrelated tool\n')
  await writeFile(path.join(home, 'AGENTS.md'), '# Existing rules\n')
  await writeFile(path.join(home, 'config.toml'), '# Existing configuration\n')
  const envFor = (codexHome) => ({ ...process.env, CODEX_HOME: codexHome, HOME: temporary, USERPROFILE: temporary })
  const env = envFor(home)
  const installedName = process.platform === 'win32' ? 'myautoapi-imagines-tool.exe' : 'myautoapi-imagines-tool'
  const installed = path.join(home, 'tools', 'myautoapi-imagines-tool', installedName)
  function run(binary, args, input, runEnv = env) {
    const result = spawnSync(binary, args, { env: runEnv, input, encoding: 'utf8', timeout: 20000, windowsHide: true })
    assert.ifError(result.error)
    assert.equal(result.status, 0, 'native ' + args[0] + ' failed with status ' + result.status)
    return result.stdout
  }
  // A synthetic fixture stays local: install only copies embedded assets.
  // No generate, edit, models, doctor or update request runs in this test.
  run(program, ['install', '--key-stdin'], 'sk-' + '0'.repeat(48) + '\n')
  assert.ok(existsSync(installed))
  assert.equal(run(installed, ['version']).trim(), '0.9.2')
  // A second install must reuse the fixture instead of waiting for key input.
  run(program, ['install'])
  const record = JSON.parse(await readFile(path.join(home, 'tools', 'myautoapi-imagines-tool', 'install-manifest.json'), 'utf8'))
  assert.equal(record.managed_by, 'myautoapi-imagines-tool')
  assert.equal(record.version, '0.9.2')
  assert.equal(record.platform, nativePlatform)
  assert.equal(record.files['tool/' + installedName], sha256(await readFile(installed)))
  assert.equal(sha256(await readFile(installed)), sha256(await readFile(program)), 'installed program must match the ported artifact')
  const skill = await readFile(path.join(home, 'skills', 'myautoapi-image-generation', 'SKILL.md'), 'utf8')
  const rules = await readFile(path.join(home, 'rules', 'myautoapi-imagines-tool.rules'), 'utf8')
  const agents = await readFile(path.join(home, 'AGENTS.md'), 'utf8')
  for (const content of [skill, rules, agents]) assert.doesNotMatch(content, /usegoodai/i)
  const command = skill.match(/本机工具命令 `([^`]+)`/)?.[1]
  assert.ok(command, 'Skill must expose the exact native command')
  const executable = process.platform === 'win32' ? command : command.startsWith("'") && command.endsWith("'") ? command.slice(1, -1).replaceAll("'\"'\"'", "'") : command
  // Windows uses an 8.3 command path to keep native invocations free of spaces.
  // Unix command examples quote a path containing spaces; policy paths are raw.
  assert.equal(await realpath(executable), await realpath(installed))
  assert.ok(skill.includes('https://api.myautoapi.com/'))
  assert.match(skill, /update --reinstall/)
  assert.match(skill, /--prompt/)
  assert.ok(rules.includes(JSON.stringify(executable)), 'allow rule must match the Skill executable')
  assert.doesNotMatch(rules, /pattern\s*=\s*\[\s*"(?:powershell|pwsh|bash|cmd)(?:\.exe)?"/i)
  assert.ok(agents.startsWith('# Existing rules\n'))
  assert.equal(await readFile(path.join(home, 'config.toml'), 'utf8'), '# Existing configuration\n')
  assert.equal(await readFile(path.join(foreign, 'sentinel.txt'), 'utf8'), 'keep unrelated tool\n')
  assert.equal(existsSync(path.join(home, 'tools', 'usegoodai-imagines-tool')), false)

  if (process.platform === 'win32') {
    const legacyHome = path.join(temporary, 'Existing 0.9.2 Installation')
    const legacyEnv = envFor(legacyHome)
    const legacyProgram = fileURLToPath(new URL('releases/v0.9.2/myautoapi-imagines-tool-v0.9.2-' + suffix, entryRoot))
    const legacyKey = 'sk-' + '2'.repeat(48)
    const replacementKey = 'sk-' + '7'.repeat(48)
    const legacyKeyPath = path.join(legacyHome, 'tools', 'myautoapi-imagines-tool', 'api-key')
    const migratedProgram = path.join(legacyHome, 'tools', 'myautoapi-imagines-tool', installedName)

    // Install the prior same-version release, then exercise the ported program against it.
    run(legacyProgram, ['install'], legacyKey + '\n', legacyEnv)
    assert.ok((await readFile(legacyKeyPath, 'utf8')) === legacyKey, 'legacy fixture key was saved')
    assert.notEqual(sha256(await readFile(migratedProgram)), sha256(await readFile(program)), 'legacy fixture must use the original binary')
    run(program, ['install'], undefined, legacyEnv)
    assert.ok((await readFile(legacyKeyPath, 'utf8')) === legacyKey, 'same-version migration preserves the saved key')
    assert.equal(sha256(await readFile(migratedProgram)), sha256(await readFile(program)), 'same-version migration replaces the old binary')
    run(program, ['install', '--key-stdin'], replacementKey + '\n', legacyEnv)
    assert.ok((await readFile(legacyKeyPath, 'utf8')) === replacementKey, '--key-stdin replaces the saved key')
    const legacyRecord = JSON.parse(await readFile(path.join(legacyHome, 'tools', 'myautoapi-imagines-tool', 'install-manifest.json'), 'utf8'))
    assert.equal(legacyRecord.version, '0.9.2')
    assert.equal(legacyRecord.files['tool/' + installedName], sha256(await readFile(path.join(legacyHome, 'tools', 'myautoapi-imagines-tool', installedName))))
  }
})
