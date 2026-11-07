import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../docs/images/params.md', import.meta.url), 'utf8')

const modelSections = [
  {
    heading: '## GPT Image 2.6 {#gpt-image-2-6}',
    required: ['`size`、`quality` 和 `output_format`', '| 3:4 |', '`768x1024`', '`xhigh`', '`max`'],
  },
  {
    heading: '## GPT Image 2',
    required: ['`low`、`medium`、`high`、`auto`'],
  },
  {
    heading: '## GPT Image 1K TH',
    required: ['`1024x1024`', '`low` 或 `high`'],
  },
  {
    heading: '## GPT Image 2 Adobe',
    required: ['`1024x1536`', '`1536x2304`', '`2304x3456`', '只能 `low`', '不支持参考图和修图'],
  },
  {
    heading: '## Grok Imagine',
    required: ['`resolution`', '`aspect_ratio`', '`1k`、`2k`', '`low`、`medium`、`high`', '`768x1152`'],
  },
  {
    heading: '## Nano Banana 2 和 Nano Banana Pro',
    required: ['`response_format.image_size`','`512`、`1K`、`2K`、`4K`', '`1K`、`2K`、`4K`', '`1:4`、`4:1`', '`1024x1024`'],
  },
]

function sectionContent(heading) {
  const start = source.indexOf(`${heading}\n`)
  assert.notEqual(start, -1, `missing model section: ${heading}`)
  const contentStart = start + heading.length
  const next = source.indexOf('\n## ', contentStart)
  return source.slice(contentStart, next === -1 ? undefined : next)
}

test('gives every image model its own top-level section on the params page', () => {
  for (const modelSection of modelSections) {
    const content = sectionContent(modelSection.heading)
    for (const required of modelSection.required) {
      assert.ok(content.includes(required), `${modelSection.heading} missing ${required}`)
    }
  }
})

test('lists every model endpoint in one table', () => {
  const content = sectionContent('## 接口和鉴权')
  const models = [
    'gpt-image-2.6-flare',
    'gpt-image-2.6-sunburst',
    'gpt-image-2.5-flare',
    'gpt-image-2.5-sunburst',
    'gpt-image-2',
    'gpt-image-1k-th',
    'gpt-image-2-adobe',
    'grok-imagine-image',
    'nano-banana-2',
    'nano-banana-pro',
  ]
  const tableModels = [...content.matchAll(/^\| `([^`]+)` \|/gm)].map((match) => match[1])
  assert.deepEqual(tableModels, models)
  assert.equal(new Set(tableModels).size, 10)
  for (const model of models) {
    assert.ok(content.includes(`| \`${model}\` |`), `missing endpoint row: ${model}`)
  }
  assert.match(content, /Authorization: Bearer <API_KEY>/)
  assert.match(content, /`image\[\]`/)
})
