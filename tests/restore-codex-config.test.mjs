import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import {
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  stat,
  writeFile,
} from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { basename, dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const testDir = dirname(fileURLToPath(import.meta.url))
const shellRestorer = join(testDir, '../docs/public/install/myautoapi-codex-installer/restore.sh')
const powershellRestorer = join(testDir, '../docs/public/install/myautoapi-codex-installer/restore.ps1')
const quickStartPath = join(testDir, '../docs/quick-start.md')

function findPowerShell() {
  for (const command of ['pwsh', 'powershell']) {
    const result = spawnSync(command, ['-NoProfile', '-Command', 'exit 0'])
    if (result.status === 0) {
      return command
    }
  }
  return undefined
}

const powershellCommand = findPowerShell()

const fixtures = [
  {
    name: 'clears the current installer config without changing features',
    config: `model_provider = "OpenAI"
model = "gpt-5.6-sol"
review_model = "gpt-5.6-sol"
model_reasoning_effort = "high"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://api.myautoapi.com"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true
image_generation = true
`,
    expected: `model_reasoning_effort = "high"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[features]
goals = true
image_generation = true
`,
  },
  {
    name: 'removes arbitrary provider names and keeps MCP blocks',
    config: `  model_provider = "custom-gateway"
model="another-model"
review_model = "review-model" # root selection
notify = ["terminal-notifier"]

[model_providers.custom-gateway] # first endpoint
name = "First"
base_url = "https://first.example.com"
http_headers = { "X-Test" = "[value]" }

[model_providers.custom-gateway.auth]
command = "credential-helper"

[model_providers.third_party]
name = "Second"
base_url = "https://second.example.com"

[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]

[features]
web_search = true
`,
    expected: `notify = ["terminal-notifier"]

[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]

[features]
web_search = true
`,
  },
  {
    name: 'keeps matching field names outside the root table',
    config: `model_provider = "relay"
model = "relay-model"
review_model = "relay-review"

[profiles.official]
model_provider = "openai"
model = "official-model"
review_model = "official-review"

[model_providers.relay]
name = "Relay"
base_url = "https://relay.example.com"

[projects."/workspace/demo"]
trust_level = "trusted"
`,
    expected: `[profiles.official]
model_provider = "openai"
model = "official-model"
review_model = "official-review"

[projects."/workspace/demo"]
trust_level = "trusted"
`,
  },
  {
    name: 'keeps a config containing only unrelated settings byte for byte',
    config: `[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]

[features]
goals = false
image_generation = false
`,
    expected: `[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]

[features]
goals = false
image_generation = false
`,
  },
  {
    name: 'preserves CRLF line endings while removing provider config',
    config: 'model_provider = "edge"\r\nmodel = "edge-model"\r\nreview_model = "edge-review"\r\n\r\n[model_providers.edge]\r\nbase_url = "https://edge.example.com"\r\n\r\n[mcp_servers.edge]\r\ncommand = "edge-mcp"\r\n',
    expected: '[mcp_servers.edge]\r\ncommand = "edge-mcp"\r\n',
  },
]

async function createCodexHome(options) {
  const { config } = options
  const auth = Object.hasOwn(options, 'auth')
    ? options.auth
    : '{"OPENAI_API_KEY":"sk-test"}\n'
  const home = await mkdtemp(join(tmpdir(), 'codex-restore-'))
  const codexHome = join(home, '.codex')
  await mkdir(codexHome)
  if (config !== undefined) {
    await writeFile(join(codexHome, 'config.toml'), config)
  }
  if (auth !== undefined) {
    await writeFile(join(codexHome, 'auth.json'), auth)
  }
  return { home, codexHome }
}

function runShellRestorer(home) {
  return spawnSync('bash', [shellRestorer], {
    cwd: home,
    encoding: 'utf8',
    env: { ...process.env, HOME: home },
  })
}

async function readBackups(codexHome, filename) {
  const entries = await readdir(codexHome)
  return entries.filter((entry) => entry.startsWith(`${filename}.bak.`))
}

for (const fixture of fixtures) {
  test(`Shell restorer ${fixture.name}`, async () => {
    const { home, codexHome } = await createCodexHome(fixture)
    try {
      const result = runShellRestorer(home)
      assert.equal(result.status, 0, result.stderr || result.stdout)

      const configPath = join(codexHome, 'config.toml')
      const authPath = join(codexHome, 'auth.json')
      assert.equal(await readFile(configPath, 'utf8'), fixture.expected)
      assert.equal((await stat(authPath)).size, 0)

      const configBackups = await readBackups(codexHome, 'config.toml')
      const authBackups = await readBackups(codexHome, 'auth.json')
      assert.equal(configBackups.length, 1)
      assert.equal(authBackups.length, 1)
      assert.equal(
        await readFile(join(codexHome, configBackups[0]), 'utf8'),
        fixture.config,
      )
      assert.equal(
        await readFile(join(codexHome, authBackups[0]), 'utf8'),
        fixture.auth ?? '{"OPENAI_API_KEY":"sk-test"}\n',
      )
    } finally {
      await rm(home, { recursive: true, force: true })
    }
  })
}

test('Shell restorer clears auth when config.toml is missing', async () => {
  const { home, codexHome } = await createCodexHome({ config: undefined })
  try {
    const result = runShellRestorer(home)
    assert.equal(result.status, 0, result.stderr || result.stdout)
    assert.equal((await stat(join(codexHome, 'auth.json'))).size, 0)
    await assert.rejects(stat(join(codexHome, 'config.toml')), { code: 'ENOENT' })
  } finally {
    await rm(home, { recursive: true, force: true })
  }
})

test('Shell restorer cleans config without creating a missing auth.json', async () => {
  const { home, codexHome } = await createCodexHome({
    config: fixtures[0].config,
    auth: undefined,
  })
  try {
    const result = runShellRestorer(home)
    assert.equal(result.status, 0, result.stderr || result.stdout)
    assert.equal(
      await readFile(join(codexHome, 'config.toml'), 'utf8'),
      fixtures[0].expected,
    )
    await assert.rejects(stat(join(codexHome, 'auth.json')), { code: 'ENOENT' })
  } finally {
    await rm(home, { recursive: true, force: true })
  }
})

test('PowerShell restorer carries the same cleanup contract', async () => {
  const script = await readFile(powershellRestorer, 'utf8')

  assert.match(script, /model_provider\|model\|review_model/)
  assert.match(script, /model_providers\\\./)
  assert.match(script, /WriteAllBytes\([^)]*\[byte\[\]\]@\(\)/s)
  assert.doesNotMatch(script, /\[features\]|goals|image_generation/)
})

test('PowerShell restorer applies the same real-file fixtures', {
  skip: powershellCommand === undefined ? 'PowerShell runtime is not installed' : false,
}, async () => {
  for (const fixture of fixtures) {
    const { home, codexHome } = await createCodexHome(fixture)
    try {
      const result = spawnSync(
        powershellCommand,
        ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', powershellRestorer],
        {
          cwd: home,
          encoding: 'utf8',
          env: { ...process.env, HOME: home, USERPROFILE: home },
        },
      )
      assert.equal(result.status, 0, result.stderr || result.stdout)
      assert.equal(
        await readFile(join(codexHome, 'config.toml'), 'utf8'),
        fixture.expected,
      )
      assert.equal((await stat(join(codexHome, 'auth.json'))).size, 0)
    } finally {
      await rm(home, { recursive: true, force: true })
    }
  }
})

test('quick start publishes both one-click restore commands', async () => {
  const quickStart = await readFile(quickStartPath, 'utf8')

  assert.match(
    quickStart,
    /irm https:\/\/docs\.myautoapi\.com\/install\/myautoapi-codex-installer\/restore\.ps1 \| iex/,
  )
  assert.match(
    quickStart,
    /curl -fsSL https:\/\/docs\.myautoapi\.com\/install\/myautoapi-codex-installer\/restore\.sh \| bash/,
  )
})

test('published restore script names are stable', () => {
  assert.equal(basename(shellRestorer), 'restore.sh')
  assert.equal(basename(powershellRestorer), 'restore.ps1')
})
