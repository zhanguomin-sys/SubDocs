---
name: myautoapi-image-generation
description: "Use when a user asks to generate, draw, make, regenerate, compare, edit, retouch, or modify images, use reference images, create multiple image variants, asks which image models are currently available, asks to refresh the relay model list, or asks to repair or diagnose the installed image tool. Route supported work through the installed local MyAutoAPI image tool, invoke different requested models concurrently, explicitly forward Linux and macOS Code Mode results, recover only attributable local outputs when that forwarding is lost, label results with exact model IDs, and display every successful local result in the conversation."
---

# 中转站生图

使用本机已安装的图片工具生成、参考或修改图片，并把每张结果显示在当前对话。用户不需要说出 MyAutoAPI、中转站、模型名或 Skill 名。

## 调用边界

- 只调用 `$CODEX_HOME/tools/myautoapi-imagines-tool/` 下已安装的原生工具；未设置 `CODEX_HOME` 时使用 `~/.codex/tools/myautoapi-imagines-tool/`。
- API Key 由工具从 `$CODEX_HOME/tools/myautoapi-imagines-tool/api-key` 读取。绝不打开、读取、列出、复制、打印、传递或扫描 `auth.json`，也不把 Key 放进命令、提示词或回答。
- 除“低频模型探测”一节规定的单次 `GET /v1/models` 外，不自行发送 `curl`、SDK 或其它 HTTP 请求。GPT Image、Adobe 和 Grok 由工具使用 `https://api.myautoapi.com/v1/images/generations` 与 `https://api.myautoapi.com/v1/images/edits`；Nano Banana 由工具使用 `https://api.myautoapi.com/v1/responses`，不手动换接口。
- 当前可调用模型是 `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`、`gpt-image-2`、`gpt-image-1k-th`、`gpt-image-2-adobe`、`grok-imagine-image`、`nano-banana-2` 和 `nano-banana-pro`。工具按响应中的真实图片格式保存最终结果。
- 生图任务本身不安装、更新、删除或替换工具和 Skill。用户明确要求更新或修复时按“更新与修复的区分”一节处理，两者不可互相代替。

## 模型选择

| 模型 | 已验证能力 | 默认参数 |
| --- | --- | --- |
| `gpt-image-2.5-flare` | 文生图、参考图、修图、多图 | `2048x2048`、`high`；普通任务默认 |
| `gpt-image-2.5-sunburst` | 文生图、参考图、修图、多图 | `2048x2048`、`high`；高质量与精细编辑 |
| `gpt-image-2` | 文生图、参考图、修图、多图 | `1024x1024`、`high` |
| `gpt-image-1k-th` | 文生图、参考图、修图、多图 | `1024x1024`、`high`（也支持 `low`） |
| `gpt-image-2-adobe` | 文生图、多图 | `1024x1536`、`1536x2304`、`2304x3456`、`low` |
| `grok-imagine-image` | 文生图、参考图、修图、多图 | `1k`、`1:1`；可选 `2k` 和常用横竖比例 |
| `nano-banana-2` | 文生图、参考图、修图、多图 | `1k`、`1:1`；可选 `512`、`2k`、`4k` 和多种比例 |
| `nano-banana-pro` | 文生图、参考图、修图、多图 | `1k`、`1:1`；可选 `2k`、`4k` 和多种比例 |

- 模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；其余未指定模型的任务默认使用 `gpt-image-2.5-flare`。
- 快速生成、普通生图、普通参考图、普通修图和批量生成都属于 Flare 的默认范围；“批量生成”不会覆盖同一句要求中的高清、精修或重要成品条件。
- 用户问有哪些模型时，直接给出上表。可以询问是否需要用同一个提示词分别生成，但只有用户明确同意后才并发调用这些模型；展示前标出准确模型名，不自动评价或排名。
- 用户明确指定的模型不静默替换。`gpt-image-1k-th` 固定 `1024x1024`，质量使用 `low` 或 `high`；所有模型的多图都由工具内部拆成独立单张请求，并发上限为 3。`gpt-image-2-adobe` 使用上表中的尺寸和 `low`。
- 参考图和修图可以使用 `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`、`gpt-image-2`、`gpt-image-1k-th`、`grok-imagine-image`、`nano-banana-2` 或 `nano-banana-pro`。Adobe 用于文生图和多图。

## 低频模型探测

- 模型探测不是默认生图流程。只有用户明确询问“当前/最新有哪些模型”、要求刷新中转站模型列表、要求确认某个模型是否仍可用，或工具返回 `model_not_found`/模型不受路由支持时才探测。
- 同一轮任务只探测一次；探测失败不自动重试。普通生图、Skill 加载、启动 Codex、列出本地已支持模型时都不探测。
- 探测只用于报告中转站当前暴露的模型 ID，不等于本工具支持。不得自动加入白名单、改默认模型、替换用户指定模型或自动发起对比生图。
- 只访问固定的 `GET https://api.myautoapi.com/v1/models`。命令读取已安装工具目录中的 `api-key`，不读取 `auth.json`；Key 只能放在进程内存或 curl 配置管道中，不能出现在命令参数、终端输出、文件或回答里。原始 JSON 只保留在临时文件并在命令结束时删除，只输出 HTTP 状态和 `id`/`display_name`。

macOS Bash 或 Linux shell：

```bash
tool_codex_home="${CODEX_HOME:-$HOME/.codex}"
key_file="$tool_codex_home/tools/myautoapi-imagines-tool/api-key"
response_file="$(mktemp)"
key=""
trap 'rm -f "$response_file"; unset key' EXIT
if [ ! -f "$key_file" ]; then
  printf '模型探测失败：未找到已安装工具的 Key。\n'
else
  key="$(tr -d '\r\n' < "$key_file")"
  http_code="$(curl --silent --show-error --max-time 60 \
    --config <(printf 'header = "Authorization: Bearer %s"\n' "$key") \
    --output "$response_file" --write-out '%{http_code}' \
    https://api.myautoapi.com/v1/models || true)"
  printf 'HTTP %s\n' "$http_code"
  if [ "$http_code" = 200 ]; then
    jq -r '.data[] | [(.id // ""), (.display_name // "")] | @tsv' "$response_file"
  else
    printf '模型探测失败，未更新模型列表。\n'
  fi
fi
```

Windows PowerShell：

```powershell
$ToolCodexHome = if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) { Join-Path $HOME ".codex" } else { $env:CODEX_HOME }
$KeyPath = Join-Path $ToolCodexHome "tools\myautoapi-imagines-tool\api-key"
$Key = $null
try {
  if (-not (Test-Path -LiteralPath $KeyPath -PathType Leaf)) { throw "未找到已安装工具的 Key" }
  $Key = (Get-Content -Raw -LiteralPath $KeyPath).Trim()
  $Headers = @{ Authorization = "Bearer $Key" }
  $Response = Invoke-RestMethod -Method Get -Uri "https://api.myautoapi.com/v1/models" -Headers $Headers -TimeoutSec 60
  $Response.data | Select-Object id, display_name
} catch {
  Write-Output "模型探测失败，未更新模型列表。"
} finally {
  $Key = $null
  $Headers = $null
  $Response = $null
}
```

## 选择调用方式

1. 用户没有提供图片：使用 `generate`，并按“模型选择”确定 `--model`。
2. 用户提供图片作为人物、物体、风格、构图或其它参考：使用 `edit`，把图片路径按用户说明的顺序通过重复的 `--image` 传入。
3. 用户要求修改已有图片：使用 `edit`，第一张 `--image` 是修改目标；提示词保留用户要求，并明确只修改指定部分、其余保持不变。
4. 用户要求同一画面要求的多张结果或多个方案：使用一次 `generate --n N`。通用数量为 1 到 10，Grok Imagine 为 1 到 9；用户没有说数量时只补问数量。不要为同一模型的一组多图拆成多个前台工具命令；工具内部会拆成独立的单张请求，并保留所有成功结果，最终数量不超过 `N`。
5. 用户要求内容彼此不同的多张图片时，每个不同画面要求使用各自的工具调用；`--n` 只表示同一提示词的多张结果。

不要把参考图自动当成修改目标。多张输入图片的用途不明确时，只补问每张图的角色。不要擅自增加人物、物体、风格、构图、配色、文字或负面提示词。

## 参数

- 用户明确给出尺寸、方向或质量时，将其转换为工具参数。下表是建议值，不是替用户作创作决定。
- 用户没有指定尺寸或质量时，使用所选模型表中的默认参数。
- 工具按接口返回的真实图片格式保存，不向用户追加格式选择。

下面的像素尺寸建议适用于 `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst` 和 `gpt-image-2`。两个 2.5 模型默认使用 2K，`gpt-image-2` 继续默认使用 1K；`gpt-image-1k-th` 和 Adobe 使用“模型选择”中的固定值，Grok Imagine 和 Nano Banana 使用后面的分辨率和比例表。

| 用户要求 | `--size` 建议值 |
| --- | --- |
| 方图、1K 方图 | `1024x1024` |
| 横图、1K 横图 | `1536x1024` |
| 竖图、1K 竖图 | `1024x1536` |
| 2K 方图 | `2048x2048` |
| 2K 横图 | `2048x1152` |
| 2K 竖图 | `1152x2048` |
| 4K 横图 | `3840x2160` |
| 4K 竖图 | `2160x3840` |
| 由模型决定 | `auto` |

| 用户要求 | `--quality` 建议值 |
| --- | --- |
| 草稿、快速预览 | `low` |
| 普通质量 | `medium` |
| 高质量、最终图片 | `high` |
| 更高质量（两个 2.5 模型） | `xhigh` |
| 最高质量（两个 2.5 模型） | `max` |
| 由模型决定 | `auto` |

`xhigh` 和 `max` 只适用于 `gpt-image-2.5-flare` 和 `gpt-image-2.5-sunburst`。两个 2.5 模型仍默认使用 `high`；只有用户明确要求对应质量档位时才传入 `xhigh` 或 `max`。

其它精确尺寸必须使用 `WIDTHxHEIGHT`：宽高都是 16 的倍数，任一边不超过 3840，长边不超过短边 3 倍，总像素在 655360 到 8294400 之间。

Grok Imagine 使用 `--resolution` 和 `--aspect-ratio`，不使用 `--size`：

| 用户要求 | 参数 |
| --- | --- |
| 1K | `--resolution 1k` |
| 2K | `--resolution 2k` |
| 方图 | `--aspect-ratio 1:1` |
| 横图 | `--aspect-ratio 16:9`、`4:3` 或 `3:2` |
| 竖图 | `--aspect-ratio 9:16`、`3:4` 或 `2:3` |

Grok Imagine 的质量可以使用 `--quality low`、`--quality medium` 或 `--quality high`；用户没有指定时不传质量参数。参考图和修图直接使用原图规格，只传原图和修改要求。

Nano Banana 使用 `--resolution` 和 `--aspect-ratio`，不使用 `--size` 或 `--quality`：

| 用户要求 | 参数 |
| --- | --- |
| 512 图（仅 `nano-banana-2`） | `--resolution 512` |
| 1K、2K、4K | `--resolution 1k`、`--resolution 2k`、`--resolution 4k` |
| 方图 | `--aspect-ratio 1:1` |
| 常用横图 | `--aspect-ratio 16:9`、`4:3`、`3:2`、`5:4` 或 `21:9` |
| 常用竖图 | `--aspect-ratio 9:16`、`3:4`、`2:3`、`4:5` 或 `9:21` |
| 超长图 | `--aspect-ratio 1:4`、`4:1`、`1:8` 或 `8:1` |

Nano Banana 的参考图和修图使用重复的 `--image` 说明人物、物体、风格、构图或修改目标。本轮真实上游已验证 PNG、JPEG、WebP 各 1 张参考图，以及 2 张和 4 张不同参考图；14 张尚未形成远端支持结论。输出规格仍由 `--resolution` 和 `--aspect-ratio` 决定。

## 输出位置和文件名

- 已打开项目或工作区：保存到项目根目录的 `images`。
- 没有项目或工作区：保存到桌面的 `images`。
- 用户指定其它目录或文件名时使用用户给出的值。
- 用户没有指定文件名时，根据画面主体给出简短、准确的中文名；不要使用时间戳、模型名或泛化的 `result`。
- 工具只在 `images` 下保存最终图片，不创建图片专属子目录。单图重名时自动追加数字；多图使用 `名称-1`、`名称-2`、`名称-3`，均不覆盖旧图。
- 修改图片始终另存为新图片，不覆盖输入图片。
- 桌面 `images` 或输入图片读取权限超出当前沙箱时，与联网许可放在同一次工具调用中申请；用户不批准时停止，不改写到 `$CODEX_HOME`。

直接根据当前会话选择输出目录，不预检、创建或列出目录：

| 当前会话 | macOS Bash | Windows PowerShell |
| --- | --- | --- |
| 已打开项目或工作区 | `output_dir="$PWD/images"` | `$OutputDir = Join-Path $PWD.Path "images"` |
| 没有打开项目或工作区 | `output_dir="$HOME/Desktop/images"` | `$OutputDir = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::Desktop)) "images"` |

## 调用流程

1. 调用前用中文告诉用户本次使用的准确模型名，以及是生成、参考还是修改；多图时同时说明数量。
2. 不要先在无网络权限的沙箱中试跑。当前权限不允许工具联网或读取输入图片时，为已安装原生工具的 `generate` 或 `edit` 子命令申请可复用的最小许可；许可范围不能扩大到整个 PowerShell、Bash 或其它命令，不要求用户开启“完全访问”，也不修改 `config.toml`。宿主只提供单次批准时申请单次；用户拒绝时停止。
3. 单模型普通任务只运行一次工具并提交一个前台调用。用户明确要求两个或更多不同模型分别生成时，为每个模型构造一个独立工具命令，并使用宿主的并发工具调用能力，在同一批并发工具调用中同时提交；三个模型就是一次提交三个工具调用，不能先等第一个完成再提交第二个。每个模型只运行一次。每个命令都是可等待、可收集 stdout 的独立前台调用，不能因为“每个模型需要独立前台调用”而改成顺序执行。禁止用 shell 后台符号、PowerShell `Start-Job`、脱离会话任务或进程轮询模拟并发。不要搜索工具或 Skill，不要预检输出目录。
4. 单模型调用仍在运行时继续等待同一个调用完成；多模型并发时继续等待这一批调用全部完成。保存模型、工具调用和 stdout 结果的对应关系，不使用其它命令检查进程。任一调用未退出时不得结束当前轮次，不得把“正在运行”当成最终结果，也不得声称图片稍后会自动显示。
5. Windows PowerShell 先用单引号 here-string 保存完整描述，再把变量作为 `--prompt` 的独立参数；不要把提示词拼进双引号命令文本。

下面 GPT Image 示例中的 `model`、`size`、`quality` 和 PowerShell 对应变量，必须先按“模型选择”和“参数”确定；示例只引用已经选定的值，不重新写死模型。

macOS 文生图或多图：

```bash
tool_codex_home="${CODEX_HOME:-$HOME/.codex}"
"$tool_codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool" generate \
  --model "$model" \
  --prompt "用户给出的完整图片描述" \
  --n 1 \
  --size "$size" \
  --quality "$quality" \
  --output-name "准确的图片名称" \
  --output-dir "$output_dir"
```

macOS 参考图或修改图：

```bash
"$tool_codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool" edit \
  --model "$model" \
  --image "/输入图片路径.png" \
  --prompt "用户给出的完整图片要求" \
  --size "$size" \
  --quality "$quality" \
  --output-name "准确的图片名称" \
  --output-dir "$output_dir"
```

macOS Grok Imagine 文生图或多图：

```bash
"$tool_codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool" generate \
  --model grok-imagine-image \
  --prompt "用户给出的完整图片描述" \
  --n 1 \
  --resolution 2k \
  --aspect-ratio 16:9 \
  --quality high \
  --output-name "准确的图片名称" \
  --output-dir "$output_dir"
```

macOS Nano Banana 文生图、多图、参考图或修图：

```bash
"$tool_codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool" generate \
  --model nano-banana-2 \
  --prompt "用户给出的完整图片描述" \
  --n 1 \
  --resolution 2k \
  --aspect-ratio 16:9 \
  --output-name "准确的图片名称" \
  --output-dir "$output_dir"
```

用户提供参考图或要求修改图片时，把 `generate` 改为 `edit`，并加入一个或多个 `--image "/输入图片路径.png"`。

### Linux 或 macOS Code Mode 结果转发

Linux 或 macOS Code Mode 把上述 Bash 命令放进嵌套 `exec_command()` 时，调用前为每个命令分别保存 `output_dir`、`output_name`、请求数量 `n`、模型 ID 和调用开始时间。嵌套命令的返回对象不会自动进入外层结果，必须显式转发；不同模型并发时分别保存，不能合并或覆盖：

```javascript
// @exec: {"yield_time_ms": 120000, "max_output_tokens": 4000}
const startedAt = Date.now();
let result = await tools.exec_command({ cmd, workdir, yield_time_ms: 5000 });
let output = result.output ?? "";
let nextStatusAt = startedAt + 30000;
while (result.session_id) {
  result = await tools.write_stdin({
    session_id: result.session_id,
    chars: "",
    yield_time_ms: 5000,
  });
  output += result.output ?? "";
  if (result.session_id && Date.now() >= nextStatusAt) {
    notify("图片仍在生成，继续等待同一个调用。");
    nextStatusAt = Date.now() + 30000;
  }
}
text(JSON.stringify({
  exit_code: result.exit_code ?? null,
  output,
  session_id: null,
}));
```

首行 `@exec` 把外层 Code Mode 调用的最长等待时间设为 120 秒；进程提前退出时仍立即返回。外层仍返回 `Script running with cell ID ...` 时，立即对同一个 `cell_id` 调用 `functions.wait`，并继续使用 `yield_time_ms: 120000`，直到外层脚本完成，不等待用户再次发消息。

内层 `yield_time_ms: 5000` 也是每次最长等待时间，不是固定睡眠；进程退出时会立即返回。返回 `session_id` 说明命令仍在运行，必须每 5 秒收取同一个 `session_id` 并累积每次返回的 `output`，不等待用户再次发消息或催促；每累计 30 秒仍未退出时，用 Code Mode 的 `notify(...)` 发出中文状态，再继续收取。命令完成后只用一次 `text(...)` 显式转发最终退出码和完整输出。宿主实际返回独立 stderr 字段时一并转发；没有该字段时不得虚构。Windows PowerShell 保持前述直接前台调用，不使用本节的 Code Mode 兜底。

Windows PowerShell：

```powershell
$ToolCodexHome = if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) { Join-Path $HOME ".codex" } else { $env:CODEX_HOME }
$ToolPath = Join-Path $ToolCodexHome "tools\myautoapi-imagines-tool\myautoapi-imagines-tool.exe"
$Prompt = @'
用户给出的完整图片描述或修改要求
'@
& $ToolPath generate `
  --model $Model `
  --prompt $Prompt `
  --n 1 `
  --size $Size `
  --quality $Quality `
  --output-name "准确的图片名称" `
  --output-dir "$OutputDir"
```

参考图或修改图时把最后一段改为 `& $ToolPath edit`，并在 `--prompt` 前加入一个或多个独立的 `--image "$InputImage"` 参数。Windows 调用 Grok Imagine 或 Nano Banana 时使用对应模型名，并用独立的 `--resolution` 和 `--aspect-ratio` 参数代替 `--size`；Nano Banana 不传 `--quality`。

工具是独立程序，不安装或调用 Python、Node 或其它运行环境。宿主为加载本 `SKILL.md` 进行的一次读取属于 Skill 加载；加载后不要再次搜索或读取 Skill。生图调用不要使用 `Get-ChildItem`、`Get-Content`、`Get-Process`、`Test-Path` 或 `mkdir` 做调用前检查或运行中轮询；模型探测只按上一节的固定命令读取专用 Key。

## 结果处理

- 工具在保存前已经验证图片格式。正常取得 stdout 时不扫描 `images` 目录，也不调用 `view_image` 或其它工具重新读取图片。
- `output_count` 是实际成功数量，`requested_count` 是用户请求数量。单张成功使用 `output_file`，多张成功使用 `output_files`；按照 stdout 路径顺序处理，最终展示数量不得超过 `requested_count`。
- 取得路径后立即使用绝对路径 Markdown 图片语法直接展示，例如 `![生成结果](/absolute/path/image.png)`。不得只回复普通路径，也不得只提供 `[查看图片](/absolute/path/image.png)` 这类普通文件链接。
- 同一模型有多张图时写明 `模型：<准确模型 ID>` 和 `第 i/output_count 张`。多模型任务必须沿用调用阶段保存的模型与 stdout 对应关系，不能根据文件名或完成顺序猜模型。
- `output_count` 小于 `requested_count` 时，展示全部成功图片，只补充一句 `已生成 x/N 张。`，不展开上游数据异常。一个模型完全失败时仍展示其它成功模型的全部图片，不自动重试或换模型。
- 最终回复必须再次使用绝对路径 Markdown 图片语法嵌入 stdout 返回的全部成功图片。最终回复中的嵌入数量与 `output_count` 一致后才能报告完成。
- 只有 Linux 或 macOS Code Mode 中嵌套命令已经结束、外层却没有取得可解析的成功或失败 JSON 时，才允许对本次已知的 `output_dir` 做一次只读恢复核查。不得重新生图、更新或重装工具，也不得检查其它目录。
- 恢复核查只接受该目录顶层的 PNG、JPEG 或 WebP 普通文件，同时满足：修改时间不早于本次调用开始时间，文件名符合工具清洗后的本次 `output_name` 原名、工具追加的重名数字后缀，或多图的 `名称-1`、`名称-2` 等编号形式。不能只按“最近 N 张”选择文件；旧图、名称不符和调用前已有的文件全部排除。
- 恢复结果为 1 到 `n` 张时，按修改时间和文件名稳定排序，直接使用绝对路径 Markdown 图片语法逐张展示，并说明“工具结果已从本次输出目录恢复；Code Mode 未返回命令结果”。少于 `n` 时只补充“已生成 x/n 张”；多于 `n` 时归属不明确，不选择其中最近的图片冒充本次结果。
- 多模型并发时分别按各自记录恢复。多个调用使用相同目录和相同文件名前缀、且 stdout 同时丢失时，不根据完成时间猜模型；只能说明图片属于本批但具体模型归属无法确认。无法确认属于本批时不展示。
- 没有匹配文件时，只能说明“没有取得命令结果，输出目录也没有发现本次新图片”，不得据此归因于中转站、macOS 权限或工具保存失败。目录读取被沙箱拒绝时只申请一次该目录的最小只读权限；用户拒绝后停止核查。
- 已取得明确的非零退出码或 `ok` 为 false 的错误 JSON 时，以 `error` 字段的简短提示为准，不进入目录恢复。只有用户要求修复时才使用 `diagnostic` 字段定位问题。
- 请求完全失败时提示用户到后台查看是否有生成记录，或让 AI 修复生图工具；不换模型、不伪造图片、不自动重复付费请求。
- 调用前提示、等待状态、成功结果和失败说明全部使用中文，不输出英文进度说明。

## 更新与修复的区分

用户说“更新生图工具”“升级生图工具”时是**更新**：按项目既定的一键安装入口取得新版本并安装，本 Skill 不描述该流程，也不把更新当成排错手段。

用户说“修复生图工具”“生图工具有问题”“调试一下生图工具”，或本次生图没有拿到 `output_file` 时是**修复**：在本机定位真实故障并说明，不重新安装、不换模型、不重复同一条失败命令，也不把“已是最新版”当成修复结论。已确认工具文件缺失或损坏时才提出重新安装，并说明这是重新安装而不是修复。

## 修复流程

按顺序推进，每步都用中文说明这一步在查什么、看到了什么；不要跳到结论。

1. 先保留当前这次调用的完整证据，不自动重复付费生图：收下宿主实际提供的 stdout、stderr、退出码和 `session_id`；缺少哪些字段就准确说明哪些字段未被宿主转发。
2. 按已取得的退出码、输出和本次目录恢复结果分流故障层：
   - 工具打印了 `ok` 为 false 的 JSON 或非零退出：普通生成只显示 `error`；修复任务读取宿主已经返回的 `diagnostic`，据此定位读取 Key、请求接口、下载图片或保存图片环节。
   - 工具无输出且非零退出：多为程序无法执行，检查工具文件是否存在、在 macOS 或 Linux 上是否有可执行属性、架构是否匹配当前机器。
   - 工具打印了 `output_file` 或 `output_files` 而对话没有图片：故障在绝对路径 Markdown 嵌入或宿主回传，不是生成失败；核对最终回复是否直接嵌入了 stdout 返回的全部路径。
   - Linux 或 macOS Code Mode 外层没有 JSON，但本次已知输出目录存在符合时间、名称和数量条件的新图片：按嵌套命令结果未转发处理，恢复并展示图片，不把工具说成静默退出。
   - 外层没有 JSON且恢复核查也没有匹配文件：证据不足，报告缺失的调用通道和核查结果，不猜测保存失败。
3. 只针对定位到的环节做最小验证，不做无关检查：保存失败就验证输出目录能否创建和写入；Key 环节失败就确认 `api-key` 是否存在且非空（不打印内容）；接口失败就按返回的 HTTP 状态判断是凭据问题还是服务端问题。
4. 向用户完整展示定位结果：说明失败发生在哪一层、依据是哪条输出、是否可在本机修复。脱敏错误原文要给出，不要改写成猜测。
5. 提出并执行最小修复：可在本机解决的（补上可执行属性、改用可写的输出目录）先征得用户同意再执行。不可在本机解决的（服务端错误、Key 无效、宿主吞掉输出）明确说明归属和下一步建议；没有用户确认不再次产生付费请求。

修复期间不打印、不转述 Key，不读取 `auth.json`，诊断产生的临时文件用完即删，不留在用户的 `images` 目录。
