import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const imageToolDocSource = readFileSync(
  new URL('../docs/image-video-group-image.md', import.meta.url),
  'utf8',
)
const paramsSource = readFileSync(new URL('../docs/images/params.md', import.meta.url), 'utf8')
const apiDocSource = readFileSync(
  new URL('../docs/images/image-generation.md', import.meta.url),
  'utf8',
)

function sectionContent(heading) {
  const start = paramsSource.indexOf(`${heading}\n`)
  assert.notEqual(start, -1, `missing section: ${heading}`)
  const contentStart = start + heading.length
  const next = paramsSource.indexOf('\n## ', contentStart)
  return paramsSource.slice(contentStart, next === -1 ? undefined : next)
}

test('makes GPT Image 2.6 the complete primary GPT Image guide', () => {
  const gptImage26 = sectionContent('## GPT Image 2.6 {#gpt-image-2-6}')

  const resolutionMappings = [
    ['1K 方图', '1024x1024'],
    ['1K 横图', '1536x1024'],
    ['1K 竖图', '1024x1536'],
    ['2K 方图', '2048x2048'],
    ['2K 横图', '2048x1152'],
    ['2K 竖图', '1152x2048'],
    ['4K 横图', '3840x2160'],
    ['4K 竖图', '2160x3840'],
  ]
  const ratioMappings = [
    ['1:1', '1024x1024'],
    ['3:2', '1152x768'],
    ['2:3', '768x1152'],
    ['3:4', '768x1024'],
    ['4:3', '1024x768'],
    ['4:5', '768x960'],
    ['5:4', '960x768'],
    ['9:16', '720x1280'],
    ['16:9', '1280x720'],
    ['21:9', '1344x576'],
    ['9:21', '576x1344'],
    ['3:1', '1536x512'],
    ['1:3', '512x1536'],
  ]

  assert.match(gptImage26, /使用 `size`、`quality` 和 `output_format`/)
  assert.match(gptImage26, /宽高为 16 的倍数/)
  assert.match(gptImage26, /单边不超过 3840/)
  assert.match(gptImage26, /总像素为 655360 至 8294400/)
  assert.match(gptImage26, /`size=auto` 时模型根据提示词选择尺寸/)
  for (const [label, size] of resolutionMappings) {
    assert.ok(gptImage26.includes(`| ${label} | \`${size}\` |`), `missing resolution mapping: ${label}`)
  }
  for (const [ratio, size] of ratioMappings) {
    assert.ok(gptImage26.includes(`| ${ratio} | \`${size}\` |`), `missing ratio mapping: ${ratio}`)
  }
  assert.match(gptImage26, /`2880x2880`/)
  assert.match(gptImage26, /比例超过 3:1 的 `4:1`、`1:4`、`8:1`、`1:8` 不支持/)
  assert.match(gptImage26, /实际交付尺寸以返回文件为准/)
  assert.match(gptImage26, /`xhigh` 和 `max` 适用于 GPT Image 2\.6 和 2\.5/)
})

test('documents GIF controls and distinguishes requested values from actual output', () => {
  const gptImage26 = sectionContent('## GPT Image 2.6 {#gpt-image-2-6}')
  for (const field of ['output_format', 'frame_count', 'frame_duration_ms']) {
    assert.ok(gptImage26.includes(`"${field}":`), field)
  }
  assert.match(gptImage26, /帧数和时长目前不能精确控制/)
  assert.match(gptImage26, /`animations`/)
  assert.match(gptImage26, /`warnings`/)
  assert.match(gptImage26, /512x512.*失败.*自动尺寸/)
  assert.match(gptImage26, /GIF 只支持文字生成，不接参考图或修图，不批量生成/)
  assert.match(gptImage26, /`--n 1`/)
  assert.match(gptImage26, /工具接受 2 至 16/)
  assert.match(gptImage26, /工具接受 20 至 2000/)
  const gifRequest = JSON.parse(gptImage26.match(/```json\n([\s\S]*?)\n```/)[1])
  assert.equal(gifRequest.model, 'gpt-image-2.6-flare')
  assert.equal(gifRequest.output_format, 'gif')
  assert.equal(gifRequest.size, 'auto')
  assert.equal(gifRequest.quality, 'low')
  assert.equal(gifRequest.n, 1)
  assert.match(sectionContent('## GPT Image 2.5 {#gpt-image-2-5}'), /不开放 GIF 参数/)
})

test('points the API guide and the tool page to the params page', () => {
  assert.match(apiDocSource, /\[生图参数汇总\]\(\/images\/params\)/)
  assert.match(apiDocSource, /"model": "gpt-image-2\.6-flare"/)
  assert.match(imageToolDocSource, /\[生图参数汇总\]\(\/images\/params\)/)
})

test('keeps the GPT Image 2 defaults and quality difference', () => {
  const gptImage2 = sectionContent('## GPT Image 2')
  assert.match(gptImage2, /工具默认 `1024x1024`、`high`/)
  assert.match(gptImage2, /`low`、`medium`、`high`、`auto`/)
  assert.match(gptImage2, /不支持 `xhigh` 和 `max`/)
  assert.doesNotMatch(gptImage2, /\/v1\/images|`size`|`n`|`aspect_ratio`|`resolution`/)
})

test('keeps the default model routing rule in the DIY prompt', () => {
  const routingRule = '模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 gpt-image-2.6-sunburst；其余未指定模型的任务默认使用 gpt-image-2.6-flare。'
  assert.ok(imageToolDocSource.includes(routingRule))
  assert.match(imageToolDocSource, /`gpt-image-2\.6-flare` \| 默认模型/)
  assert.match(imageToolDocSource, /`gpt-image-2\.6-sunburst` \| 说“高清”“精修”“重要成品”时自动使用/)
})
