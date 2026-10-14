import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../docs/image-video-group-image.md', import.meta.url), 'utf8')

const modelFolds = [
  {
    summary: 'GPT Image 2.5：当前主模型、尺寸、质量和比例',
    required: ['`size`、`quality` 和 `output_format`', '`3:4`', '`768x1024`', '`xhigh`', '`max`'],
  },
  {
    summary: 'GPT Image 2：旧版兼容差异',
    required: ['`1024x1024`', '`low`、`medium`、`high`、`auto`', '用户明确指定'],
  },
  {
    summary: 'GPT Image 1K TH：固定尺寸、质量和接口',
    required: ['`1024x1024`', '`low` 或 `high`', '/v1/images/edits'],
  },
  {
    summary: 'GPT Image 2 Adobe：竖图尺寸、质量和接口',
    required: ['`1024x1536`', '`1536x2304`', '`2304x3456`', '只使用 `low`'],
  },
  {
    summary: 'Grok Imagine：分辨率、比例、质量和接口',
    required: ['`resolution`', '`aspect_ratio`', '`1k`、`2k`', '`low`、`medium`、`high`', '`768x1152`'],
  },
  {
    summary: 'Nano Banana 2：分辨率、比例、参考图和接口',
    required: ['`image_size`', '`512`、`1k`、`2k`、`4k`', '`input_image.image_url`', '1:4、4:1'],
  },
  {
    summary: 'Nano Banana Pro：分辨率、比例、参考图和接口',
    required: ['`1k`、`2k`、`4k`', '九个成功请求', '`quality`、`size`、`output_format`'],
  },
]

function foldContent(summary) {
  const marker = `<summary>${summary}</summary>`
  const start = source.indexOf(marker)
  assert.notEqual(start, -1, `missing model fold: ${summary}`)
  const contentStart = start + marker.length
  const end = source.indexOf('</details>', contentStart)
  assert.notEqual(end, -1, `unclosed model fold: ${summary}`)
  return source.slice(contentStart, end)
}

test('keeps every image model as a same-level single-click fold', () => {
  const detailsCount = (source.match(/<details>/g) || []).length
  const summaries = (source.match(/<summary>/g) || []).length
  assert.equal(detailsCount, summaries)

  for (const modelFold of modelFolds) {
    assert.equal(source.split(`<summary>${modelFold.summary}</summary>`).length - 1, 1)
    const content = foldContent(modelFold.summary)
    assert.doesNotMatch(content, /<details>|<summary>/, `${modelFold.summary} contains a nested fold`)
    for (const required of modelFold.required) {
      assert.match(content, new RegExp(required.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
    }
  }
})

test('keeps the common Python rules separate from model-specific folds', () => {
  const content = foldContent('自建 Python 脚本的通用规则')
  assert.match(content, /统一接口和鉴权/)
  assert.match(content, /解析和保存/)
  assert.match(content, /Authorization: Bearer <API_KEY>/)
  assert.match(content, /Images 响应检查/)
  assert.match(content, /Responses 只扫描 `output`/)
})
