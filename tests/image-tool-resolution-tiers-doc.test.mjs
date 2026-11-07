import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../docs/image-video-group-image.md', import.meta.url), 'utf8')

function foldContent(summary) {
  const marker = `<summary>${summary}</summary>`
  const start = source.indexOf(marker)
  assert.notEqual(start, -1, `missing model fold: ${summary}`)
  const contentStart = start + marker.length
  const end = source.indexOf('</details>', contentStart)
  assert.notEqual(end, -1, `unclosed model fold: ${summary}`)
  return source.slice(contentStart, end)
}

test('documents the observed resolution-tier behavior separately for every model', () => {
  const image1k = foldContent('GPT Image 1K TH：固定尺寸、质量和接口')
  assert.match(image1k, /只有固定的 1K 方图/)
  assert.match(image1k, /2K 和 4K 请求都在本地参数校验阶段被拒绝/)

  const adobe = foldContent('GPT Image 2 Adobe：竖图尺寸、质量和接口')
  assert.match(adobe, /没有 `1K`、`2K`、`4K` 档位参数/)
  assert.match(adobe, /三种允许尺寸都未返回图片/)
  assert.match(adobe, /重试后三个尺寸仍均为 HTTP 502/)

  const grok = foldContent('Grok Imagine：分辨率、比例、质量和接口')
  assert.match(grok, /4K 在本地参数校验阶段被拒绝/)
  assert.match(grok, /四个成功请求的实际文件均为 `768x1152`/)
  assert.match(grok, /重试后只有 `1k` \+ `16:9` 再次成功/)

  const banana2 = foldContent('Nano Banana 2：分辨率、比例、参考图和接口')
  assert.match(banana2, /本轮覆盖的 `512`、1K、2K、4K 请求均返回 HTTP 502/)
  assert.match(banana2, /不能把 HTTP 502 写成“不支持该档位”/)
  assert.match(banana2, /不接受任意 `--size` 像素尺寸/)

  const bananaPro = foldContent('Nano Banana Pro：分辨率、比例、参考图和接口')
  assert.match(bananaPro, /九个成功请求的实际文件均为 `1024x1024`/)
  assert.match(bananaPro, /请求成功不等于实际按请求的档位和比例交付/)
})
