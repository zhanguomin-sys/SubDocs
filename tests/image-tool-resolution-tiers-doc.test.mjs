import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const params = readFileSync(new URL('../docs/images/params.md', import.meta.url), 'utf8')
const imageTool = readFileSync(new URL('../docs/image-video-group-image.md', import.meta.url), 'utf8')

test('states the observed output-size limits as conclusions, without raw test logs', () => {
  assert.match(params, /只出 1K 方图，2K、4K 和其它比例会被拒绝/)
  assert.match(params, /实测交付文件固定为 `768x1152`/)
  assert.match(params, /Nano Banana Pro 实测交付文件都是 `1024x1024`/)
  assert.match(imageTool, /`nano-banana-pro` \| .*上游工具实测为 1024x1024/)
  assert.doesNotMatch(params, /首次结果|重试结果|本轮结果|HTTP 502/)
})
