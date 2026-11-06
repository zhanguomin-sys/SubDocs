import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const imageToolDocSource = readFileSync(
  new URL('../docs/image-video-group-image.md', import.meta.url),
  'utf8',
)
const apiDocSource = readFileSync(
  new URL('../docs/images/image-generation.md', import.meta.url),
  'utf8',
)

function foldContent(summary) {
  const marker = `<summary>${summary}</summary>`
  const start = imageToolDocSource.indexOf(marker)
  assert.notEqual(start, -1, `missing model fold: ${summary}`)
  const contentStart = start + marker.length
  const end = imageToolDocSource.indexOf('</details>', contentStart)
  assert.notEqual(end, -1, `unclosed model fold: ${summary}`)
  return imageToolDocSource.slice(contentStart, end)
}

test('makes GPT Image 2.5 the complete primary GPT Image guide', () => {
  const gptImage25 = foldContent('GPT Image 2.5：当前主模型、尺寸、质量和比例')

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

  assert.match(gptImage25, /使用 `size`、`quality` 和 `output_format`/)
  assert.match(gptImage25, /宽高为 16 的倍数/)
  assert.match(gptImage25, /单边不超过 3840/)
  assert.match(gptImage25, /总像素为 655360 至 8294400/)
  assert.match(gptImage25, /`size=auto`[\s\S]{0,80}根据提示词选择尺寸/)
  for (const [label, size] of resolutionMappings) {
    assert.ok(gptImage25.includes(`| ${label} | \`${size}\` |`), `missing resolution mapping: ${label}`)
  }
  for (const [ratio, size] of ratioMappings) {
    assert.ok(gptImage25.includes(`| ${ratio} | \`${size}\` |`), `missing ratio mapping: ${ratio}`)
  }
  assert.match(gptImage25, /`2880x2880`/)
  assert.match(gptImage25, /比例超过 3:1 的 `4:1`、`1:4`、`8:1`、`1:8` 不支持/)
  assert.match(gptImage25, /超过 `2560x1440` 的输出属于实验性范围/)
  assert.doesNotMatch(
    gptImage25,
    /GPT Image 2(?!\.5)|gpt-image-2(?!\.5)|旧模型|旧版|区别|相比|沿用|继承|复用/i,
  )
  assert.doesNotMatch(gptImage25, /^\+\|/m)
})

test('links the API guide to the primary GPT Image 2.5 section', () => {
  assert.match(imageToolDocSource, /<span id="gpt-image-2-5"><\/span>/)
  assert.match(
    apiDocSource,
    /前往\[中转站生图工具的 GPT Image 2.5 说明\]\(\/image-video-group-image#gpt-image-2-5\).*点击“GPT Image 2.5：当前主模型、尺寸、质量和比例”展开查看/,
  )
  assert.match(apiDocSource, /宽高必须是 16 的倍数/)
  assert.match(apiDocSource, /长边与短边的比例不超过 3:1/)
  assert.match(apiDocSource, /总像素在 655360 到 8294400 之间/)
})

test('keeps only GPT Image 2 differences after the primary 2.5 guide', () => {
  const routingRule = '模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；其余未指定模型的任务默认使用 `gpt-image-2.5-flare`。'
  const gptImage25Marker = '<summary>GPT Image 2.5：当前主模型、尺寸、质量和比例</summary>'
  const gptImage2Marker = '<summary>GPT Image 2：旧版兼容差异</summary>'
  const gptImage25 = foldContent('GPT Image 2.5：当前主模型、尺寸、质量和比例')
  const gptImage2 = foldContent('GPT Image 2：旧版兼容差异')

  assert.ok(imageToolDocSource.includes(routingRule))
  assert.ok(imageToolDocSource.indexOf(gptImage25Marker) < imageToolDocSource.indexOf(gptImage2Marker))
  assert.match(imageToolDocSource, /普通任务默认使用 `gpt-image-2\.5-flare`/)
  assert.match(
    imageToolDocSource,
    /高清、高质量、精细细节、精修、严格保留原图或重要成品[\s\S]{0,80}`gpt-image-2\.5-sunburst`/,
  )
  assert.match(imageToolDocSource, /`gpt-image-2\.5-flare`[\s\S]{0,160}`2048x2048`/)
  assert.match(imageToolDocSource, /`gpt-image-2\.5-sunburst`[\s\S]{0,160}`2048x2048`/)
  assert.match(gptImage25, /`xhigh` 和 `max` 只适用于两个 GPT Image 2\.5 模型/)
  assert.match(gptImage2, /本站工具默认值 \| `1024x1024`、`high`/)
  assert.match(gptImage2, /`low`、`medium`、`high`、`auto`/)
  assert.doesNotMatch(
    gptImage2,
    /\/v1\/images\/(?:generations|edits)|1536x1024|1024x1536|2048x2048|2048x1152|1152x2048|3840x2160|2160x3840|2880x2880|1152x768|768x1152|768x1024|1024x768|768x960|960x768|720x1280|1280x720|1344x576|576x1344|1536x512|512x1536|文生图|参考图|修图|多图|`size`|`output_format`|`n`|`aspect_ratio`|`resolution`/,
  )
})
