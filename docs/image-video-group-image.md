---
title: 安装生图工具
---

# 安装生图工具

MyAutoAPI 中转站生图工具让 Codex 直接生成、修改并展示图片，适合已经完成[快速开始](/quick-start)、希望在 Codex 对话中使用图片模型的用户。

安装后，Codex 会自动调用工具并把成品图显示在当前对话中。普通任务默认使用 `gpt-image-2.5-flare` 生成 2K 图片，明确要求高清、精修或重要成品时使用 `gpt-image-2.5-sunburst`；安装器会写入 `$CODEX_HOME/tools/myautoapi-imagines-tool/`、`$CODEX_HOME/skills/myautoapi-image-generation/` 和 `$CODEX_HOME/AGENTS.md`，用户不需要手动修改这些文件。

## 1. 创建专门画图分组的 API Key

进入 MyAutoAPI 管理后台的 **API 密钥** 页面，额外创建一个 API Key，名称可以填写“画图专用”。

把这个 Key 的分组设置为 **专门画图分组**，保存后点击复制。这里使用的不是快速开始中配置 Codex 对话模型的普通 Key，两个 Key 不要复制反。

安装器无法判断 Key 是否来自正确分组，复制前必须自己核对分组名称。

<a class="doc-image-link" href="/images/画图/画图专用分组.jpg" target="_blank" rel="noopener">
  <img src="/images/画图/画图专用分组.jpg" alt="在 API 密钥页面核对专门画图分组并复制画图专用 API Key">
</a>

## 2. 在终端安装生图工具

Windows 按 `Win + X`，点击 **终端** 或 **Windows PowerShell**，粘贴运行：

```powershell
irm https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1 | iex
```

每次执行命令都会输入并覆盖画图 Key。看到“请输入专门画图分组 Key”时，粘贴第 1 步复制的 Key，然后按回车。安装器只会下载 Windows x64 程序，不需要 Python、Node 或管理员权限。

<details>
<summary>Linux 和 Mac 用户看这里</summary>

64 位 x64 Linux、Apple Silicon Mac 和 Intel Mac 使用同一条命令，安装脚本会自动识别系统和芯片。打开终端运行：

```bash
curl -fsSL https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh | bash
```

每次执行命令都会输入并覆盖画图 Key。看到 Key 输入提示后，粘贴第 1 步复制的专门画图分组 Key，然后按回车。安装器只下载当前 Linux 或 Mac 对应的一个程序。

</details>

## 3. 打开 Codex 开始生图

安装完成后打开 Codex，新建一个任务。Codex 已经打开时不用退出应用，直接新建任务即可；不要沿用安装前的旧对话。

先发送下面这句话了解入口和可用模型：

```text
介绍一下中转站生图工具怎么使用，有哪些模型。
```

随后直接描述要画的内容：

```text
画一张白底红苹果。
```

生成成功后，图片会直接显示在当前 Codex 对话中，不会只返回文件路径。

<details>
<summary>查看常用说法</summary>

直接生图：

```text
画一张雨后城市街道的人物照片，电影感，自然光影，不要文字和水印。
```

指定模型：

```text
使用 nano-banana-pro 画一张中式庭院夜景。
```

使用参考图或修改图片：

```text
参考当前文件夹里的产品照片，生成一张白底电商主图。
```

```text
把这张图片的背景改成浅灰色，保留主体不变。
```

一次生成多张：

```text
按同一个要求生成 3 张图片，并全部显示在对话中。
```

对比模型：

```text
用同一个要求分别调用 gpt-image-2.5-flare 和 nano-banana-pro，各生成一张并显示出来。
```

</details>

<details>
<summary>查看支持的模型、尺寸和参数</summary>

| 模型 | 使用场景 | 默认规格 | 专属限制 |
| --- | --- | --- | --- |
| `gpt-image-2.5-flare` | 普通生图、快速生成和批量任务 | `2048x2048`、`high` | 支持 `xhigh`、`max` |
| `gpt-image-2.5-sunburst` | 高清、精修和重要成品 | `2048x2048`、`high` | 支持 `xhigh`、`max` |
| `gpt-image-2` | 用户明确指定的旧版兼容任务 | `1024x1024`、`high` | 质量最高到 `high` |
| `gpt-image-1k-th` | 固定 1K 方图 | `1024x1024`、`high` | 只使用 `low` 或 `high` |
| `gpt-image-2-adobe` | Adobe 竖图 | `1024x1536`、`low` | 只支持生图和多图 |
| `grok-imagine-image` | Grok 图片任务 | `1k`、`1:1` | 支持 7 种比例 |
| `nano-banana-2` | 多规格 Banana 图片任务 | `1k`、`1:1` | 支持 `512`、2K、4K 和 15 种比例 |
| `nano-banana-pro` | Banana Pro 图片任务 | `1k`、`1:1` | 支持 2K、4K 和 15 种比例 |

Codex 会按照用户明确提出的模型、数量、尺寸和画面要求调用工具，不会替用户决定创作内容。价格见[模型价格](/models)。

</details>

<details>
<summary>查看更新、安装位置和卸载方式</summary>

需要更新时，直接对 Codex 说：

```text
更新生图工具。
```

每次执行安装或更新命令都会要求重新粘贴专门画图分组 Key，并覆盖旧 Key。显示“已是最新版”时只跳过下载，仍继续配置 Key。

默认安装位置：

| 内容 | 位置 |
| --- | --- |
| 工具和专门画图 Key | `$CODEX_HOME/tools/myautoapi-imagines-tool/` |
| 生图 Skill | `$CODEX_HOME/skills/myautoapi-image-generation/` |
| Codex 全局调用规则 | `$CODEX_HOME/AGENTS.md` |

未设置 `CODEX_HOME` 时，Windows 使用 `C:\Users\你的用户名\.codex\`，Linux 和 Mac 使用 `~/.codex/`。生成图片保存在当前项目的 `images` 文件夹；没有打开项目时保存在桌面的 `images` 文件夹。

需要卸载时，直接对 Codex 说“卸载中转站生图工具”。专门画图 Key 默认保留，永久删除前需要单独确认。

</details>

<details>
<summary>安装或生图失败先看这里</summary>

| 遇到的问题 | 检查动作 |
| --- | --- |
| 安装器提示 Key 无效或生图返回鉴权错误 | 回到 API 密钥页面复制 **专门画图分组** 的 Key，重新执行上面的安装命令，粘贴后覆盖旧 Key |
| 安装后 Codex 不知道生图工具 | 新建一个 Codex 任务，不要继续使用安装前的旧对话 |
| Codex 只回复图片路径或图片只出现在折叠的中间过程 | 确认当前任务已经读取生图 Skill；最终回复必须用绝对路径 Markdown 图片语法再次嵌入图片 |
| 生图接口返回错误 | 本次请求会直接停止，不会自动重试或更换模型；稍后重新发起一次新请求 |
| 图片无法写入 | 打开一个有写入权限的项目后再试，或让 Codex 使用桌面 `images` 文件夹 |

</details>

<details>
<summary>不用一键安装：让 Codex 创建 Python 脚本</summary>

这个方法适合不使用一键安装器、希望把生图脚本和规则只放在当前项目中的用户。脚本和规则只对当前项目生效，不会安装全局 Skill。

把下面整段提示词发给 Codex：

```text
请完整阅读并严格按照这个页面操作：
https://docs.myautoapi.com/image-video-group-image.html

根据页面中的“自建 Python 脚本的通用规则”和各模型参数折叠，完成以下任务：
1. 在当前文件夹创建一个可运行的单文件生图脚本，文件名为“生成图片.py”。
2. 创建或更新当前项目根目录的 AGENTS.md，保留原有内容，并写入页面规定的“项目生图规则”。
3. 除“生成图片.py”和 AGENTS.md 外，不创建其它工程文件，也不自动安装依赖。
4. 为 MyAutoAPI API Key 保留一个明显的配置项，创建完成后只让我填写这个值。
5. 脚本是给 Codex 调用的，不要让我手工修改图片描述、模型或运行命令。
6. 脚本必须接收 Codex 每次传入的完整图片描述、模型、数量、规格和可选原图文件，不能把图片描述写死在脚本里。
7. 模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 gpt-image-2.5-sunburst；其余未指定模型的任务默认使用 gpt-image-2.5-flare。
8. 每个模型请求只发送一次，禁止自动重试或静默更换模型。
9. 成功后只保存最终图片，并立即使用 view_image 打开每张图片，确认图片能够正常显示。
10. view_image 成功后，最终回复仍必须用绝对路径 Markdown 图片语法再次嵌入每张图片，例如：`![生成结果](/absolute/path/image.png)`。不得只依赖中间工具输出，也不得只回复普通文件路径或普通 Markdown 文件链接。失败时显示脱敏错误，不创建空图片或运行记录。
```

以后在这个项目中直接对 Codex 说“画一张……”即可。需要比较模型时，明确说出模型名；Codex 应使用同一要求逐个调用，每个模型只请求一次。

</details>

<span id="gpt-image-2-5"></span>

<details>
<summary>GPT Image 2.5：当前主模型、尺寸、质量和比例</summary>

模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；其余未指定模型的任务默认使用 `gpt-image-2.5-flare`。

快速生成、普通生图、普通参考图、普通修图和批量生成都属于 Flare 的默认范围；“批量生成”不会覆盖同一句要求中的高清、精修或重要成品条件。

两个模型都支持文生图、参考图、修图和多图。文生图使用 `POST /v1/images/generations`，参考图和修图使用 `POST /v1/images/edits`；使用 `size`、`quality` 和 `output_format`，不使用 `resolution` 或 `aspect_ratio`。

| 参数 | 写法 |
| --- | --- |
| `model` | `gpt-image-2.5-flare` 或 `gpt-image-2.5-sunburst` |
| `prompt` | 必填，写完整图片描述或修改指令 |
| `size` | `auto` 或 `WIDTHxHEIGHT`；宽高为 16 的倍数，单边不超过 3840，长短边比例不超过 3:1，总像素为 655360 至 8294400；本站工具默认 `2048x2048` |
| `quality` | `low`、`medium`、`high`、`xhigh`、`max`、`auto`；默认 `high` |
| `output_format` | 当前工具使用 `png` |
| `n` | `1` 至 `10`；工具把多图拆成独立单张请求 |
| 参考图 | 使用 `edit`，每张图片重复传入 `--image` |

本站工具主动传入 `2048x2048` 和 `high`；直接调用官方接口并省略 `size` 或 `quality` 时，这两个字段默认使用 `auto`。使用 `size=auto` 时，模型会根据提示词选择尺寸。

`xhigh` 和 `max` 只适用于两个 GPT Image 2.5 模型。工具默认保持 `high`，只有用户明确指定对应质量档位时才传入 `xhigh` 或 `max`。

### 按 1K、2K、4K 选择尺寸

1K、2K、4K 是分辨率档位，不是 `size` 的参数值。必须把档位和方向换成具体像素，不能填写 `size=1k`、`size=2k` 或 `size=4k`。

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

`2880x2880` 的总像素为上限 `8294400`，可作为最大像素方图，但不是官方命名的 4K 方图。超过 `2560x1440` 的输出属于实验性范围，生成时间和稳定性可能与较小尺寸不同。

实际档位和扣费以后台 **使用记录** 为准，价格查看[模型价格](/models)。

### 使用其它比例

用户说“3:4”时转换为 `--size 768x1024`，不要把 `3:4` 直接填进 `size`。

| 比例 | `size` 示例 |
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

比例超过 3:1 的 `4:1`、`1:4`、`8:1`、`1:8` 不支持。`--aspect-ratio 3:4` 也不支持。

</details>

<span id="gpt-image-2"></span>

<details>
<summary>GPT Image 2：旧版兼容差异</summary>

`gpt-image-2` 只用于用户明确指定该模型或已有项目需要继续保持原模型的任务，不再作为默认模型。

| 项目 | 设置 |
| --- | --- |
| 模型名 | `gpt-image-2` |
| 本站工具默认值 | `1024x1024`、`high` |
| `quality` | `low`、`medium`、`high`、`auto` |

</details>

### 其它模型的 1K、2K、4K 实测

以下结果来自 2026-09-02 使用提示词“画一个古装美女”对当前中转路径的单张文生图测试。表中的“实际文件”读取保存图片的像素头；请求被接口接受、后台是否有调用记录、最终文件像素是三个不同的验证点。测试结果只描述当前中转路径，不能据此改写上游模型的官方能力或后台计费档位。

<details>
<summary>GPT Image 1K TH：固定尺寸、质量和接口</summary>

| 参数 | 写法 |
| --- | --- |
| `model` | 固定为 `gpt-image-1k-th` |
| `prompt` | 必填，写完整图片描述或修改指令 |
| 文生图接口 | `POST /v1/images/generations` |
| 参考图和修图接口 | `POST /v1/images/edits` |
| `size` | 只使用 `1024x1024` |
| `quality` | `low` 或 `high`；默认 `high` |
| `output_format` | 当前工具使用 `png` |
| `n` | `1` 至 `10`；多图由工具拆成独立单张请求 |
| 参考图 | 使用 `edit`，每张图片重复传入 `--image` |

这个模型只有固定的 1K 方图：不使用 `--aspect-ratio`、`--resolution` 或 2K/4K 尺寸。需要横图、竖图或其它比例时，使用 `gpt-image-2`。

#### 1K、2K、4K 实测

| 请求 | 本轮结果 | 实际文件 |
| --- | --- | --- |
| `size: 1024x1024`、`quality: high` | 成功返回图片 | `1024x1024` |
| `size: 2048x2048` | 本地参数校验拒绝：模型当前只验证了 `1024x1024` | 无 |
| `size: 3840x2160` | 本地参数校验拒绝：模型当前只验证了 `1024x1024` | 无 |

2K 和 4K 请求都在本地参数校验阶段被拒绝。因此这里不存在“同一模型切换 1K、2K、4K”的调度方法；`1024x1024` 是唯一可用输出规格。

</details>

<details>
<summary>GPT Image 2 Adobe：竖图尺寸、质量和接口</summary>

| 参数 | 写法 |
| --- | --- |
| `model` | 固定为 `gpt-image-2-adobe` |
| `prompt` | 必填，写完整图片描述 |
| 文生图接口 | `POST /v1/images/generations` |
| `size` | `1024x1536`、`1536x2304`、`2304x3456` |
| `quality` | 只使用 `low` |
| `output_format` | 当前工具使用 `png` |
| `n` | `1` 至 `10`；多图由工具拆成独立单张请求 |
| 参考图和修图 | 不开放；需要参考图或修图时使用 `gpt-image-2` |

Adobe 的三个尺寸都是 2:3 竖图，不使用 `--aspect-ratio` 或 `--resolution`。它没有 `1K`、`2K`、`4K` 档位参数，只能按三个精确像素尺寸发请求。

#### 三个固定尺寸实测

| 请求尺寸 | 首次结果 | 重试结果 | 实际文件 |
| --- | --- | --- | --- |
| `1024x1536` | 上游 HTTP 502 | 上游 HTTP 502 | 无 |
| `1536x2304` | 上游 HTTP 503 | 上游 HTTP 502 | 无 |
| `2304x3456` | 上游 HTTP 503 | 上游 HTTP 502 | 无 |

重试后三个尺寸仍均为 HTTP 502，三种允许尺寸都未返回图片。因此不能把三个固定尺寸写成已经在当前中转路径成功交付的 1K、2K、4K 档位。服务恢复后，重新请求并以实际文件像素核验结果。

</details>

<details>
<summary>Grok Imagine：分辨率、比例、质量和接口</summary>

`grok-imagine-image` 使用 Images 接口，尺寸通过 `resolution` 和 `aspect_ratio` 表达，不能发送 GPT Image 的 `size`。

| 参数 | 写法 |
| --- | --- |
| `model` | 固定为 `grok-imagine-image` |
| `prompt` | 必填，写完整图片描述或修改指令 |
| 文生图接口 | `POST /v1/images/generations` |
| 参考图和修图接口 | `POST /v1/images/edits` |
| `resolution` | `1k`、`2k`；默认 `1k` |
| `aspect_ratio` | `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3` |
| `quality` | 可选 `low`、`medium`、`high`；未指定时使用模型默认值 |
| `n` | `1` 至 `9`；工具把多图拆成独立单张请求 |
| 请求体输出字段 | `response_format: "b64_json"`；保存扩展名按返回图片实际格式判断 |
| 参考图和修图 | 使用 `edit` 和重复的 `--image`；直接使用原图规格，不传 `resolution`、`aspect_ratio` 或 `quality` |

#### 1K、2K、4K 实测

| 请求 | 首次结果 | 重试结果 | 实际文件 |
| --- | --- | --- | --- |
| `1k` + `1:1` | 上游 HTTP 400 | 上游 HTTP 400 | 无 |
| `1k` + `16:9` | 成功返回图片 | 再次成功返回图片 | 两次都是 `768x1152` |
| `1k` + `9:16` | 成功返回图片 | 上游 HTTP 400 | 首次为 `768x1152` |
| `2k` + `1:1` | 成功返回图片 | 上游 HTTP 400 | 首次为 `768x1152` |
| `2k` + `4:3` | 成功返回图片 | 未重复 | `768x1152` |
| `2k` + `16:9` | 上游 HTTP 400 | 上游 HTTP 400 | 无 |
| `2k` + `9:16` | 上游 HTTP 400 | 上游 HTTP 400 | 无 |
| `4k` + `1:1` | 本地参数校验拒绝：只允许 `1k`、`2k` | 不适用 | 无 |

Grok 的请求参数只有 `1k`、`2k`，4K 在本地参数校验阶段被拒绝。首次四个成功请求的实际文件均为 `768x1152`，没有按请求的 1K/2K 或横竖比例交付；重试后只有 `1k` + `16:9` 再次成功，且实际文件仍是 `768x1152`。当前中转路径修复前，不把 Grok 用于需要确定输出档位或比例的任务。

</details>

<details>
<summary>Nano Banana 2：分辨率、比例、参考图和接口</summary>

`nano-banana-2` 使用 Responses 接口。文生图、参考图和修图都使用同一个请求入口，不使用 Images 接口。

请求地址是 `POST https://api.myautoapi.com/v1/responses`，请求头使用 `Authorization: Bearer <API_KEY>` 和 `Content-Type: application/json`。

| 参数 | 写法 |
| --- | --- |
| `model` | 固定为 `nano-banana-2` |
| `prompt` | 放在 `input[0].content` 的 `input_text.text` |
| 接口 | `POST /v1/responses` |
| `input` | 数组；文本放在 `input_text.text`，图片放在 `input_image.image_url` |
| `stream` | 固定为 `false` |
| `response_format.type` | 固定为 `image` |
| `response_format.mime_type` | 当前固定为 `image/png` |
| `resolution` | 工具参数为 `512`、`1k`、`2k`、`4k`；请求体字段为 `image_size` |
| `aspect_ratio` | 工具参数和请求体字段都使用比例字符串 |
| `quality`、`size`、`output_format` | 不使用 |
| 多图数量 | 工具内部拆成多个单张 Responses 请求，不发送顶层 `n`；同一画面要求 3 张时发送 3 个单张请求 |
| 参考图和修图 | 使用 `edit` 并重复传入 `--image`；PNG、JPEG、WebP 各 1 张以及 2 张、4 张不同图片已验证 |
| 参考图数量 | 本地工具最多 14 张；14 张远端请求尚未形成支持结论 |

工具的 `--resolution 512|1k|2k|4k` 会分别写入请求体的 `image_size`：`512`、`1K`、`2K`、`4K`。

工具允许传入的 `--aspect-ratio`：

```text
1:1、3:2、2:3、3:4、1:4、4:1、4:3、4:5、
5:4、1:8、8:1、9:16、16:9、21:9、9:21
```

#### 1K、2K、4K 实测

| 请求 | 首次结果 | 重试结果 | 实际文件 |
| --- | --- | --- | --- |
| `512` + `1:1` | 未测试 | 上游 HTTP 502 | 无 |
| `1k` + `1:1` | 上游 HTTP 502 | 上游 HTTP 502 | 无 |
| `2k` + `1:1` | 上游 HTTP 502 | 上游 HTTP 502 | 无 |
| `4k` + `1:1` | 上游 HTTP 502 | 上游 HTTP 502 | 无 |
| `1k` + `16:9` | 上游 HTTP 502 | 未重复 | 无 |
| `1k` + `9:16` | 上游 HTTP 502 | 未重复 | 无 |
| `4k` + `16:9` | 上游 HTTP 502 | 未重复 | 无 |

本轮覆盖的 `512`、1K、2K、4K 请求均返回 HTTP 502，没有取得可测量的图片文件。工具能把这些参数发送到服务端，但不能把 HTTP 502 写成“不支持该档位”；它表示当前这条中转路径没有完成生成。

Nano Banana 2 不接受任意 `--size` 像素尺寸。`--size 2048x2048` 会在本地参数校验阶段被拒绝；可传的只有离散的 `--resolution 512|1k|2k|4k`，再配合 `--aspect-ratio`。服务恢复后，再用每个离散档位的实际文件像素确认调度是否生效。

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

参考图或改图时，在同一个 `content` 数组中增加：

```json
{
  "type": "input_image",
  "image_url": "data:image/png;base64,<图片的Base64内容>"
}
```

`image_url` 必须是 PNG、JPEG 或 WebP 的 Data URL，例如 `data:image/png;base64,<...>`。文本、参考图和修改指令都放在 `input[0].content`，不能写成顶层 `prompt` 或 `images` 字段。

</details>

<details>
<summary>Nano Banana Pro：分辨率、比例、参考图和接口</summary>

`nano-banana-pro` 使用与 Nano Banana 2 相同的 Responses 请求结构，但分辨率从 `1k` 开始，不支持 `512`。

| 参数 | 写法 |
| --- | --- |
| `model` | 固定为 `nano-banana-pro` |
| `prompt` | 放在 `input[0].content` 的 `input_text.text` |
| 接口 | `POST /v1/responses` |
| `input` | 数组；文本使用 `input_text`，图片使用 `input_image` |
| `stream` | 固定为 `false` |
| `response_format.type` | 固定为 `image` |
| `response_format.mime_type` | 当前固定为 `image/png` |
| `resolution` | `1k`、`2k`、`4k`；请求体字段为 `image_size` |
| `aspect_ratio` | `1:1`、`3:2`、`2:3`、`3:4`、`1:4`、`4:1`、`4:3`、`4:5`、`5:4`、`1:8`、`8:1`、`9:16`、`16:9`、`21:9`、`9:21` |
| `quality`、`size`、`output_format` | 不使用 |
| 多图数量 | 工具内部拆成多个单张 Responses 请求，不发送顶层 `n` |
| 参考图和修图 | 使用 `edit` 并重复传入 `--image`；最多 14 张由本地参数校验限制 |

#### 1K、2K、4K 实测

| 请求 | 本轮结果 | 实际文件 |
| --- | --- | --- |
| `1k` + `1:1` | 成功返回图片 | `1024x1024` |
| `2k` + `1:1` | 成功返回图片 | `1024x1024` |
| `4k` + `1:1` | 成功返回图片 | `1024x1024` |
| `1k` + `16:9` | 成功返回图片 | `1024x1024` |
| `2k` + `16:9` | 成功返回图片 | `1024x1024` |
| `4k` + `16:9` | 成功返回图片 | `1024x1024` |
| `1k` + `9:16` | 成功返回图片 | `1024x1024` |
| `2k` + `9:16` | 成功返回图片 | `1024x1024` |
| `4k` + `9:16` | 成功返回图片 | `1024x1024` |

九个成功请求的实际文件均为 `1024x1024`。这表示当前路径接受 `resolution` 和 `aspect_ratio`，但没有把 2K、4K、横图或竖图落实到交付文件；请求成功不等于实际按请求的档位和比例交付。当前中转路径修复前，不把 Nano Banana Pro 用于需要确定 2K、4K 或非方图输出的任务。

请求体的关键字段：

```json
{
  "model": "nano-banana-pro",
  "input": [{
    "role": "user",
    "content": [{
      "type": "input_text",
      "text": "用户给出的完整图片描述"
    }]
  }],
  "stream": false,
  "response_format": {
    "type": "image",
    "mime_type": "image/png",
    "aspect_ratio": "16:9",
    "image_size": "1K"
  }
}
```

</details>

<details>
<summary>自建 Python 脚本的通用规则</summary>

### 统一接口和鉴权

| 模型 | 生成接口 | 参考图和修图 |
| --- | --- | --- |
| `gpt-image-2.5-flare` | `/v1/images/generations` | `/v1/images/edits` |
| `gpt-image-2.5-sunburst` | `/v1/images/generations` | `/v1/images/edits` |
| `gpt-image-2` | `/v1/images/generations` | `/v1/images/edits` |
| `gpt-image-1k-th` | `/v1/images/generations` | `/v1/images/edits` |
| `gpt-image-2-adobe` | `/v1/images/generations` | 不开放 |
| `grok-imagine-image` | `/v1/images/generations` | `/v1/images/edits` |
| `nano-banana-2` | `/v1/responses` | `/v1/responses` |
| `nano-banana-pro` | `/v1/responses` | `/v1/responses` |

API 根地址固定为 `https://api.myautoapi.com`，鉴权使用 `Authorization: Bearer <API_KEY>`。脚本不得读取或覆盖 Codex 的 `auth.json`，也不得把 Key 写入请求记录、错误信息或生成图片目录。

### 解析和保存

1. Images 响应检查 `data[].b64_json` 和 `data[].url`；URL 只下载一次。
2. Responses 只扫描 `output`，解析结构化图片、Markdown Data URL 或 HTTPS 图片链接。
3. 只有图片字节非空且文件签名有效时才算成功，扩展名按照实际 PNG 或 JPEG 内容决定。
4. 输出目录使用当前项目的 `images`；没有项目时使用桌面的 `images`。
5. 只保留最终图片，不创建请求、响应、摘要或图片专属子目录。
6. 单图向 Codex 返回 `output_file`，多图返回 `output_files`；Codex 必须逐张调用 `view_image`。
7. `view_image` 成功后，最终回复必须用每张图片的绝对路径和 Markdown 图片语法再次嵌入全部图片；普通路径和普通文件链接不能代替图片嵌入。
8. HTTP 非 2xx、解码失败或保存失败时立即停止，不自动重试、不更换模型。

写入当前项目 `AGENTS.md` 的规则必须包含：固定使用本项目的 `生成图片.py`；模型只能从“查看支持的模型、尺寸和参数”表中选择；每次调用前说明模型；不读取或显示 Key；失败不重试；成功后逐张调用 `view_image`，并在最终回复中用绝对路径 Markdown 图片语法再次嵌入所有成功图片。

</details>
