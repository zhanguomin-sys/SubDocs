import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('macOS discovery covers current Codex App layout and system Bash keeps the JavaScript heredoc outside command substitution', async () => {
  const template = await readFile(
    new URL('../myautoapi-codex-installer/codex-domestic-models/src/installer.sh.in', import.meta.url),
    'utf8',
  )

  for (const path of [
    '$app_root/Contents/Resources/codex-cli/CodexCLI.app/Contents/MacOS/codex',
    '$app_root/Contents/Resources/codex-cli/bin/codex',
    '$app_root/Contents/Resources/codex',
    '$app_root/Contents/Resources/bin/codex',
  ]) {
    assert.ok(template.includes(path), 'macOS backend discovery must include ' + path)
  }
  assert.match(
    template,
    /invoke_model_import\(\)\s*\{\s*run_model_import[\s\S]*?<<'MYAUTOAPI_JAVASCRIPT'[\s\S]*?MYAUTOAPI_JAVASCRIPT\s*\}\s*if import_message="\$\(invoke_model_import\)"/,
  )
  assert.doesNotMatch(template, /if import_message="\$\(run_model_import/)
})

for (const extension of ['ps1', 'sh']) {
  test(`published domestic ${extension} installer matches the reviewed subproject`, async () => {
    const paths = [
      `../myautoapi-codex-installer/codex-domestic-models/install/codex-models.${extension}`,
      `../myautoapi-codex-installer/install/codex-models.${extension}`,
      `../docs/public/install/codex-models.${extension}`,
    ]
    const [source, upstream, published] = await Promise.all(
      paths.map(path => readFile(new URL(path, import.meta.url))),
    )
    assert.deepEqual(upstream, source)
    if (extension === 'ps1') {
      // Local PowerShell 5.1 files need a BOM for Chinese text; HTTP + iex must not receive it.
      assert.deepEqual(source.subarray(0, 3), Buffer.from([0xef, 0xbb, 0xbf]))
      assert.deepEqual(published, source.subarray(3))
    } else {
      assert.deepEqual(published, source)
    }
  })

  test(`domestic ${extension} installer uses UTF-8 text and revalidates updates`, async () => {
    const headers = await readFile(new URL('../docs/public/_headers', import.meta.url), 'utf8')
    assert.match(
      headers,
      new RegExp(
        `/install/codex-models\\.${extension}\\s+` +
          'Content-Type:\\s*text/plain;\\s*charset=utf-8\\s+' +
          'Cache-Control:\\s*public,\\s*max-age=0,\\s*must-revalidate',
        'i',
      ),
    )
  })
}
