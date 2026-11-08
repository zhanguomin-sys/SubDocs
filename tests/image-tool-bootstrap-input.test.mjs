import assert from 'node:assert/strict'
import { spawn, spawnSync } from 'node:child_process'
import { mkdtemp, readFile, rm } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const entry = new URL('../docs/public/install/myautoapi-imagines-tool/', import.meta.url)
const prompt = '请输入专门画图分组 Key'
const initialKey = 'sk-' + '3'.repeat(48)
const replacementKey = 'sk-' + '8'.repeat(48)

test('public bootstrap entries explicitly prompt before using silent key input', async () => {
  for (const extension of ['ps1', 'sh']) {
    const source = await readFile(new URL('install.' + extension, entry), 'utf8')
    const release = await readFile(new URL('../myautoapi-imagines-tool/release/install.' + extension, import.meta.url), 'utf8')
    assert.equal(source, release, 'published and maintained entry must match')
    assert.ok(source.includes(prompt), extension + ' must display an explicit key prompt')
    assert.ok(source.indexOf(prompt) < source.indexOf('install --key-stdin'), extension + ' must prompt before entering silent native input')
  }
})

function isolatedEnv(temporary) {
  return { ...process.env, CODEX_HOME: path.join(temporary, 'codex'), HOME: temporary, USERPROFILE: temporary }
}

function seed(program, env) {
  const result = spawnSync(program, ['install', '--key-stdin'], { env, input: initialKey + '\n', encoding: 'utf8', timeout: 10000, windowsHide: true })
  assert.ifError(result.error)
  assert.equal(result.status, 0, 'isolated fixture installation failed')
}

function interactive(program, args, env) {
  const child = spawn(program, args, { env, stdio: ['pipe', 'pipe', 'pipe'], windowsHide: true })
  let output = ''
  let ended = false
  let error
  const observers = new Set()
  const changed = () => { for (const observer of observers) observer() }
  for (const stream of [child.stdout, child.stderr]) {
    stream.setEncoding('utf8')
    stream.on('data', (text) => { output += text; changed() })
  }
  child.on('error', (cause) => { error = cause; changed() })
  const completion = new Promise((resolve, reject) => {
    child.on('error', reject)
    child.on('close', (code) => { ended = true; changed(); resolve(code) })
  })
  function waitForText(expected) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { observers.delete(check); reject(new Error('expected bootstrap message was not displayed: ' + output.replaceAll(initialKey, '[fixture]').replaceAll(replacementKey, '[fixture]'))) }, 10000)
      function check() {
        if (output.includes(expected)) {
          clearTimeout(timer); observers.delete(check); resolve()
        } else if (error || ended) {
          clearTimeout(timer); observers.delete(check); reject(new Error('bootstrap ended before requesting key input: ' + output.replaceAll(initialKey, '[fixture]').replaceAll(replacementKey, '[fixture]')))
        }
      }
      observers.add(check); check()
    })
  }
  return { child, completion, waitForText, output: () => output }
}

test('PowerShell bootstrap prompts, rejects empty input and completes without stdin EOF', { timeout: 25000 }, async (t) => {
  if (process.platform !== 'win32') return t.skip('requires Windows PowerShell')
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'myautoapi-key-prompt-'))
  t.after(() => rm(temporary, { recursive: true, force: true }))
  const env = isolatedEnv(temporary)
  env.PSModulePath = path.join(process.env.SystemRoot || 'C:\Windows', 'System32/WindowsPowerShell/v1.0/Modules')
  const program = fileURLToPath(new URL('current/myautoapi-imagines-tool-windows-amd64.exe', entry))
  seed(program, env)
  const script = fileURLToPath(new URL('install.ps1', entry)).replaceAll("'", "''")
  // Invoke the UTF-8 body like irm | iex; Windows PowerShell 5 otherwise guesses
  // an ANSI encoding for a BOM-less local script.
  const command = "[Console]::OutputEncoding = [Text.UTF8Encoding]::new(); $OutputEncoding = [Text.UTF8Encoding]::new(); & ([scriptblock]::Create([IO.File]::ReadAllText('" + script + "', [Text.Encoding]::UTF8)))"
  const run = interactive('powershell.exe', ['-NoLogo', '-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', command], env)
  t.after(() => { if (run.child.exitCode === null) run.child.kill() })
  // Read-Host's prompt goes to the console host, not the redirected stream.
  // The preceding visible instruction and empty-input retry still exercise
  // the real script and real Read-Host without replacing either in a mock.
  await run.waitForText('现在需要填写')
  assert.ok(!run.output().includes('正在检查安装位置'), 'script must request input before invoking the silent native installer')
  run.child.stdin.write('\n')
  await run.waitForText('Key 不能为空')
  run.child.stdin.write(replacementKey + '\n')
  // Deliberately keep stdin open: consume one line without waiting for EOF.
  let timer
  try {
    const result = await Promise.race([run.completion, new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('bootstrap waited for stdin EOF')), 10000) })])
    assert.equal(result, 0, 'PowerShell bootstrap failed')
  } finally { clearTimeout(timer) }
  assert.ok(run.output().includes('完成：V0.9.2'), 'installation did not complete')
  assert.ok((await readFile(path.join(env.CODEX_HOME, 'tools/myautoapi-imagines-tool/api-key'), 'utf8')).trim() === replacementKey, 'explicit bootstrap input must replace the saved synthetic key')
})

// Python supplies the controlling terminal required by /dev/tty. Captured terminal
// text and the synthetic key are never emitted by the driver.
const shellDriver = String.raw`
import errno, os, pathlib, pty, select, signal, sys, time
script, home = sys.argv[1:]
prompt = '请输入专门画图分组 Key'.encode()
replacement = ('sk-' + '8' * 48).encode()
pid, terminal = pty.fork()
if pid == 0:
    os.execlp('bash', 'bash', script)
captured = bytearray()
sent_empty = False
sent_key = False
status = None
deadline = time.monotonic() + 12
try:
    while time.monotonic() < deadline:
        if select.select([terminal], [], [], 0.1)[0]:
            try:
                chunk = os.read(terminal, 65536)
                if chunk:
                    captured.extend(chunk)
            except OSError as error:
                if error.errno != errno.EIO:
                    raise
        count = captured.count(prompt)
        if count >= 1 and not sent_empty:
            assert '正在检查安装位置'.encode() not in captured, 'silent installer ran before key prompt'
            os.write(terminal, b'\n')
            sent_empty = True
        if count >= 2 and not sent_key:
            os.write(terminal, replacement + b'\n')
            sent_key = True
        finished, candidate = os.waitpid(pid, os.WNOHANG)
        if finished:
            status = candidate
            break
    assert sent_empty and sent_key, 'bootstrap did not explicitly prompt and reject empty input'
    assert status is not None, 'bootstrap waited for terminal EOF after a complete key line'
    assert os.waitstatus_to_exitcode(status) == 0, 'shell bootstrap failed'
    assert '完成：V0.9.2'.encode() in captured, 'shell installation did not complete'
    assert (pathlib.Path(home) / 'tools/myautoapi-imagines-tool/api-key').read_bytes().strip() == replacement, 'explicit bootstrap input did not replace the synthetic key'
finally:
    if status is None:
        os.kill(pid, signal.SIGKILL)
        os.waitpid(pid, 0)
    os.close(terminal)
`

test('shell bootstrap prompts, rejects empty input and completes with terminal open', { timeout: 25000 }, async (t) => {
  if (process.platform !== 'linux' || process.arch !== 'x64') return t.skip('requires Linux x64 and a PTY')
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'myautoapi-key-prompt-'))
  t.after(() => rm(temporary, { recursive: true, force: true }))
  const env = isolatedEnv(temporary)
  const program = fileURLToPath(new URL('current/myautoapi-imagines-tool-linux-amd64', entry))
  seed(program, env)
  const result = spawnSync('python3', ['-c', shellDriver, fileURLToPath(new URL('install.sh', entry)), env.CODEX_HOME], { env, encoding: 'utf8', timeout: 18000 })
  assert.ifError(result.error)
  assert.equal(result.status, 0, result.stderr)
})
