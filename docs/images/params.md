---
title: 生图参数汇总
---

# 生图参数汇总

本文对应 MyAutoAPI 生图工具 **V0.8.3**。默认使用 `gpt-image-2.6-flare`，高清、精修或重要成品使用 `gpt-image-2.6-sunburst`。现有工具支持以下十个模型；本版工具不支持 GIF 输出。

| 模型 | 尺寸与质量 | 参考图和修图 |
| --- | --- | --- |
| `gpt-image-2.6-flare`、`gpt-image-2.6-sunburst` | 默认 `2048x2048`、`high`；横图 `2048x1152`，竖图 `1152x2048`；质量支持 `low`、`medium`、`high`、`xhigh`、`max`、`auto` | 支持 |
| `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst` | 静态图参数与 2.6 相同；仅在明确指定时调用 | 支持 |
| `gpt-image-2` | 使用具体像素尺寸；质量支持 `low`、`medium`、`high`、`auto` | 支持 |
| `gpt-image-1k-th` | 默认 `1024x1024` | 支持 |
| `gpt-image-2-adobe` | 默认 `1024x1536`，文生图 | 不支持 |
| `grok-imagine-image` | `--resolution 1k / 2k`；比例支持 `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3` | 支持 |
| `nano-banana-2` | `--resolution` 支持 `512`、`1k`、`2k`、`4k`，配合 `--aspect-ratio`；不使用 `--size`、`--quality` | 支持，最多传入 14 张参考图；远端实际支持以请求结果为准 |
| `nano-banana-pro` | `--resolution` 支持 `1k`、`2k`、`4k`，配合 `--aspect-ratio`；不使用 `--size`、`--quality` | 支持 |

2.6 和 2.5 模型的 2K 尺寸要通过 `--size` 传具体像素，不使用 `--resolution 2k`。输出使用 PNG；Grok、Nano Banana 按接口返回的真实图片格式保存。

多张输入图片使用重复的 `--image`，并说明每张图的用途。多张结果使用 `--n N`，工具按独立单张请求生成，最多并发 3 张，保留全部成功结果。失败时不会自动重试或更换模型。

安装和自然语言用法看 [安装生图工具](/image-video-group-image)，HTTP 接口示例看 [生图 API](/images/image-generation)，价格看 [模型价格](/models)。
