# 生图参数汇总

自己写脚本或调用接口时查本页。API Key 使用 **专门画图分组** 的 Key，价格看 [模型价格](/models)。

## 接口和鉴权

根地址 `https://api.myautoapi.com`，请求头 `Authorization: Bearer <API_KEY>`。

| 模型 | 文生图 | 参考图和修图 | 尺寸参数 |
| --- | --- | --- | --- |
| `gpt-image-2.6-flare` | `POST /v1/images/generations` | `POST /v1/images/edits` | `size` |
| `gpt-image-2.6-sunburst` | `POST /v1/images/generations` | `POST /v1/images/edits` | `size` |
| `gpt-image-2.5-flare` | `POST /v1/images/generations` | `POST /v1/images/edits` | `size` |
| `gpt-image-2.5-sunburst` | `POST /v1/images/generations` | `POST /v1/images/edits` | `size` |
| `gpt-image-2` | `POST /v1/images/generations` | `POST /v1/images/edits` | `size` |
| `gpt-image-1k-th` | `POST /v1/images/generations` | `POST /v1/images/edits` | 固定 `1024x1024` |
| `gpt-image-2-adobe` | `POST /v1/images/generations` | 不支持 | 三个固定尺寸 |
| `grok-imagine-image` | `POST /v1/images/generations` | `POST /v1/images/edits` | `resolution` + `aspect_ratio` |
| `nano-banana-2` | `POST /v1/responses` | `POST /v1/responses` | `image_size` + `aspect_ratio` |
| `nano-banana-pro` | `POST /v1/responses` | `POST /v1/responses` | `image_size` + `aspect_ratio` |

`/v1/images/edits` 用 `multipart/form-data` 上传：一张图字段名 `image`，多张图用 `image[]`。Nano Banana 把图片放进请求体，见下文。

## GPT Image 2.6 {#gpt-image-2-6}

`gpt-image-2.6-flare` 适合普通生图和批量任务，`gpt-image-2.6-sunburst` 适合高清、精修和重要成品。两个模型参数相同，都支持文生图、参考图、修图和多图，使用 `size`、`quality` 和 `output_format`，不使用 `resolution` 或 `aspect_ratio`。

下表列出一键生图工具开放的静态图参数范围。GIF 使用后面的动画参数。

| 参数 | 写法 |
| --- | --- |
| `prompt` | 必填，完整图片描述或修改指令 |
| `size` | `auto` 或 `WIDTHxHEIGHT`：宽高为 16 的倍数，单边不超过 3840，长短边比例不超过 3:1，总像素为 655360 至 8294400 |
| `quality` | `low`、`medium`、`high`、`xhigh`、`max`、`auto` |
| `output_format` | 静态图使用 `png`；文字生成 GIF 使用 `gif` |
| `n` | `1` 至 `10`；工具把多图拆成独立单张请求 |

一键工具的静态图默认传 `size=2048x2048`、`quality=high`；`size=auto` 时模型根据提示词选择尺寸。`xhigh` 和 `max` 适用于 GPT Image 2.6 和 2.5，不用于 GPT Image 2。

### 1K、2K、4K 对应的 `size`

1K、2K、4K 不能直接填进 `size`，按下表换成像素：

| 用户要求 | `size` |
| --- | --- |
| 1K 方图 | `1024x1024` |
| 1K 横图 | `1536x1024` |
| 1K 竖图 | `1024x1536` |
| 2K 方图 | `2048x2048` |
| 2K 横图 | `2048x1152` |
| 2K 竖图 | `1152x2048` |
| 4K 横图 | `3840x2160` |
| 4K 竖图 | `2160x3840` |

工具允许的最大方图是 `2880x2880`（总像素上限）。表中尺寸是请求值，实际交付尺寸以返回文件为准。

### 按比例对应的 `size`

| 比例 | `size` |
| --- | --- |
| 1:1 | `1024x1024` |
| 3:2 | `1152x768` |
| 2:3 | `768x1152` |
| 3:4 | `768x1024` |
| 4:3 | `1024x768` |
| 4:5 | `768x960` |
| 5:4 | `960x768` |
| 9:16 | `720x1280` |
| 16:9 | `1280x720` |
| 21:9 | `1344x576` |
| 9:21 | `576x1344` |
| 3:1 | `1536x512` |
| 1:3 | `512x1536` |

比例超过 3:1 的 `4:1`、`1:4`、`8:1`、`1:8` 不支持。

### 生成 GIF 动图 {#gif}

在 Codex 中安装 V0.9.2 生图工具后，直接说 `画一个红色立方体旋转的循环 GIF 动图`。Skill 自动使用 2.6 Flare；说“用 2.6 Sunburst”则使用 Sunburst。GIF 只支持文字生成，不接参考图或修图，不批量生成。

自己调用时，向 `POST https://api.myautoapi.com/v1/images/generations` 发送 JSON，请求头仍为 `Authorization: Bearer <API_KEY>`，并设置 `Content-Type: application/json`：

```json
{
  "model": "gpt-image-2.6-flare",
  "prompt": "一个红色立方体绕竖直轴旋转一周，白色背景，固定镜头，无缝循环，不要文字。",
  "output_format": "gif",
  "size": "auto",
  "quality": "low",
  "n": 1,
  "frame_count": 9,
  "frame_duration_ms": 200
}
```

改用 Sunburst 时只替换模型名。把 `data[].b64_json` 解码后直接保存为 `.gif`，无需本地拼帧；通用解码方法见 [生图 API](/images/image-generation)。

| API 参数 | 原生工具参数 | 填写方法 |
| --- | --- | --- |
| `output_format` | `--output-format gif` | 必须显式开启 GIF，默认仍为静态 PNG |
| `size` | `--size auto` | GIF 默认自动尺寸；固定 `512x512` 在两个模型的上游工具测试中均失败，先用自动尺寸 |
| `quality` | `--quality low` | GIF 默认 `low`，先用来验证动作和循环效果 |
| `n` | `--n 1` | 每次只生成一个 GIF，不批量生成 |
| `frame_count` | `--frame-count` | 可选，默认请求 9 帧；工具接受 2 至 16 |
| `frame_duration_ms` | `--frame-duration-ms` | 可选，默认请求每帧 200 毫秒；工具接受 20 至 2000 |

**帧数和时长目前不能精确控制。** 上游工具文档记录两个模型都能返回真实多帧、无限循环 GIF；实测文件为 8 至 16 帧、每帧 120 或 150 毫秒，与请求的 4 或 9 帧、每帧 200 或 500 毫秒不完全一致。不能把请求值相乘当作成品时长。

工具原样保存文件，stdout 的 `animations` 返回实际尺寸、帧数、每帧毫秒数、总时长与循环信息；参数未完全生效时另有 `warnings`，Codex 会据此说明实际结果。工具不自动重试、改帧率或补帧。

原生工具只发送非流式文字生图请求，不提供 GIF 编辑、局部预览图、自定义压缩或 `prompt_rewriter` 参数；总像素预算由接口检查。

## GPT Image 2.5 {#gpt-image-2-5}

`gpt-image-2.5-flare`、`gpt-image-2.5-sunburst` 继续支持原有静态生图、参考图、修图和六档质量，尺寸按上面的静态图表填写。一键工具不再默认选择它们；明确写出模型名时使用。不开放 GIF 参数。

## GPT Image 2

工具默认 `1024x1024`、`high`。静态图质量可选 `low`、`medium`、`high`、`auto`，不支持 `xhigh` 和 `max`。

## GPT Image 1K TH

| 参数 | 写法 |
| --- | --- |
| `size` | 只能 `1024x1024` |
| `quality` | `low` 或 `high`，工具默认 `high` |
| `n` | `1` 至 `10` |

只出 1K 方图，2K、4K 和其它比例会被拒绝。需要其它尺寸时换 GPT Image 2.6。

## GPT Image 2 Adobe

| 参数 | 写法 |
| --- | --- |
| `size` | `1024x1536`、`1536x2304`、`2304x3456`（都是 2:3 竖图） |
| `quality` | 只能 `low` |
| `n` | `1` 至 `10` |

工具默认 `1024x1536`、`low`。只能文生图，不支持参考图和修图。

## Grok Imagine

| 参数 | 写法 |
| --- | --- |
| `resolution` | `1k`、`2k` |
| `aspect_ratio` | `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3` |
| `quality` | `low`、`medium`、`high`；不填使用模型默认值 |
| `n` | `1` 至 `9` |
| `response_format` | `b64_json` |

不能发送 `size`。参考图和修图时不传 `resolution`、`aspect_ratio` 和 `quality`，按原图规格输出。

工具默认请求 `1k`、`1:1`；未指定质量时不传 `quality`。

上游工具文档的实测交付文件固定为 `768x1152`，不随 `resolution` 和 `aspect_ratio` 变化。需要确定尺寸或比例时，不用 Grok。

## Nano Banana 2 和 Nano Banana Pro

两个模型都走 `POST /v1/responses`，文字、参考图和修改指令都放在 `input[0].content` 里，不能写成顶层 `prompt` 或 `images` 字段。

| 参数 | 写法 |
| --- | --- |
| `response_format.image_size` | Nano Banana 2：`512`、`1K`、`2K`、`4K`；Nano Banana Pro：`1K`、`2K`、`4K` |
| `response_format.aspect_ratio` | `1:1`、`3:2`、`2:3`、`3:4`、`4:3`、`4:5`、`5:4`、`9:16`、`16:9`、`21:9`、`9:21`、`1:4`、`4:1`、`1:8`、`8:1` |
| `response_format.mime_type` | `image/png` |
| `stream` | `false` |
| 多图 | 不支持 `n`，每张图单独发一次请求 |
| 参考图 | PNG、JPEG、WebP 的 Data URL；Nano Banana 2 上游工具已验证 1、2、4 张，本地参数最多 14 张，14 张远端请求尚未形成支持结论 |

不使用 `quality`、`size` 和 `output_format`。

两个模型的工具默认参数都是 `1k`、`1:1`；`--resolution 1k` 写入请求体时为 `image_size: "1K"`。

最小请求体：

```json
{
  "model": "nano-banana-2",
  "input": [{
    "role": "user",
    "content": [{
      "type": "input_text",
      "text": "一只红苹果放在白色桌面上"
    }]
  }],
  "stream": false,
  "response_format": {
    "type": "image",
    "mime_type": "image/png",
    "aspect_ratio": "1:1",
    "image_size": "1K"
  }
}
```

带参考图或修图时，在同一个 `content` 数组里加：

```json
{
  "type": "input_image",
  "image_url": "data:image/png;base64,<图片的Base64内容>"
}
```

上游工具文档记录 Nano Banana Pro 实测交付文件都是 `1024x1024`，2K、4K、横图和竖图参数不生效。需要确定尺寸或比例时，不用 Nano Banana Pro。

## 读取返回的图片

| 接口 | 图片位置 |
| --- | --- |
| `/v1/images/generations`、`/v1/images/edits` | `data[].b64_json`（Base64 解码）或 `data[].url`（下载一次）；两者都有时用 Base64 |
| `/v1/responses` | 只在 `output` 里找：结构化图片、Markdown 里的 Data URL 或 HTTPS 图片链接 |

图片字节非空，按实际 PNG、JPEG、WebP 或 GIF 格式保存，扩展名按真实格式决定。GIF 还要解码检查帧数、帧间隔和循环信息；没有有效图片时报错，不写空文件。
