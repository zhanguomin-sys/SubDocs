# GPT 生图 API

自己写脚本或服务端流程时，直接调用 `gpt-image-2` 生成并保存图片。需要让 Codex 安装中转站生图工具并在对话中显示图片，查看[安装生图工具](/image-video-group-image)；需要用画布连续创作图片，查看[无限画布](/images/infinite-canvas)。

## 懒人方法：让 AI 帮你快速测试

把下面这段话复制给能操作本机文件的 AI。它会帮你做成一个可重复运行的单文件脚本。

```text
请帮我用最少步骤测试 GPT 生图 API。

要求：
1. 写一个可运行脚本，文件名用中文，例如 `生成图片.sh`、`生成图片.ps1` 或 `生成图片.py`。
2. 可以用 `curl` 发送请求；不要额外安装依赖，本机确实缺少工具时先等我确认。
3. 调用接口：POST https://api.myautoapi.com/v1/images/generations，API Key 让我自己填。
4. 请求体参考 `gpt-image-2` 的 1K 方图调用方法，例如：
   {"model":"gpt-image-2","prompt":"生成一张白底玻璃杯产品图","size":"1024x1024"}
5. 把原始返回保存为 response.json，同时兼容 data[].b64_json 和 data[].url；有 b64_json 时先解码，有 url 时下载图片；两者都没有有效图片数据时，保留原始返回并明确报错。
```

测试成功后，再根据自己的业务改提示词、输出文件名、批量读取方式和错误处理。

## 接口地址

程序或 HTTP 请求的完整端点是：

```text
POST https://api.myautoapi.com/v1/images/generations
```

客户端要求填写 `Base URL` 时，不要把完整端点填进去。按客户端说明填写根地址或 `/v1` 地址：

```text
https://api.myautoapi.com
```

或：

```text
https://api.myautoapi.com/v1
```

普通聊天客户端会自己拼接 `/chat/completions`、`/models` 等路径。不要把 `/v1/images/generations` 填进普通聊天客户端的 `Base URL`。

## 生成一张 1K 方图

先把 MyAutoAPI 管理后台创建的 API Key 放到环境变量里。

macOS / Linux / WSL：

```bash
export MYAUTOAPI_API_KEY="你的 API Key"
```

Windows PowerShell：

```powershell
$env:MYAUTOAPI_API_KEY="你的 API Key"
```

发送图片生成请求，并把原始返回保存为 `response.json`：

```bash
curl https://api.myautoapi.com/v1/images/generations \
  -H "Authorization: Bearer $MYAUTOAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "为一家现代咖啡店生成一张温暖明亮的产品宣传图，画面包含一杯拿铁、一块可颂和简洁的中文标题区域。",
    "size": "1024x1024"
  }' \
  -o response.json
```

请求成功后，图片通常在返回 JSON 的 `data[].b64_json` 或 `data[].url` 里。脚本要同时兼容这两种结果。

## 保存返回的图片

保存图片时按下面顺序处理：

1. 解析整个 JSON，不要只硬取 `data[0]`。
2. 先读取有效的 `data[].b64_json`，做 Base64 解码。
3. 没有有效 `b64_json` 时，读取有效的 `data[].url` 并下载图片。
4. 两者同时存在时，优先使用 Base64，URL 作为备用。
5. 确认拿到图片二进制后再写入文件；没有有效图片数据时，保留 `response.json` 并报错。

下面用 Python 标准库把 `response.json` 里的图片保存为 `outputs/image.png`：

```python
import base64
import json
import os
import urllib.request

with open("response.json", "r", encoding="utf-8") as file:
    response = json.load(file)

def read_image_bytes(item):
    image_base64 = item.get("b64_json")
    image_url = item.get("url")

    if isinstance(image_base64, str) and image_base64.strip():
        try:
            return base64.b64decode(image_base64)
        except Exception:
            if not isinstance(image_url, str) or not image_url.strip():
                raise

    if isinstance(image_url, str) and image_url.strip():
        with urllib.request.urlopen(image_url, timeout=60) as remote:
            return remote.read()

    return None

image_bytes = None
for item in response.get("data", []):
    if isinstance(item, dict):
        image_bytes = read_image_bytes(item)
        if image_bytes:
            break

if not image_bytes:
    raise RuntimeError("response.json 里没有有效的 data[].b64_json 或 data[].url")

os.makedirs("outputs", exist_ok=True)
with open("outputs/image.png", "wb") as file:
    file.write(image_bytes)
```

## 常用参数

下面的尺寸规则针对本站 GPT Image 像素尺寸接口。1K、2K、4K 是分辨率档位，不是 `size` 可以直接填写的值；请求必须使用 `1024x1024`、`2048x2048`、`3840x2160` 这类具体像素尺寸。方图、横图和竖图的完整对应关系，前往[中转站生图工具的 GPT Image 2.5 说明](/image-video-group-image#gpt-image-2-5)，点击“GPT Image 2.5：当前主模型、尺寸、质量和比例”展开查看。

| 参数 | 说明 |
| --- | --- |
| `size` | 填 `auto` 或 `WIDTHxHEIGHT`。宽高必须是 16 的倍数，单边不超过 3840，长边与短边的比例不超过 3:1，总像素在 655360 到 8294400 之间。 |
| `quality` | 图片质量选项。不同质量档位是否可用，以实际支持为准。 |
| `output_format` | 输出格式选项，例如需要 PNG 时可尝试设置为 `png`，以实际支持为准。 |

不要填写 `size: "1k"`、`size: "2k"`、`size: "4k"`，也不要给 `gpt-image-2` 发送 `resolution` 或 `aspect_ratio`。需要控制后台档位时，使用三档尺寸表中的明确像素值；使用 `size: "auto"` 时，输出尺寸和后台档位由本次生成结果决定。

如果接口返回参数不支持、请求体无效或模型不可用，先删掉可选参数，只保留 `model`、`prompt` 和一个确认可用的 `size` 再测试。实际档位和扣费进入后台 **使用记录** 核对。

## 图片编辑

图片编辑接口是：

```text
POST https://api.myautoapi.com/v1/images/edits
```

如果账号和接口支持图片编辑，按 `multipart/form-data` 形式提交图片文件和提示词。这个接口需要单独测试，不要直接假定所有账号、客户端或参数都已支持。

请求形态示例：

```bash
curl https://api.myautoapi.com/v1/images/edits \
  -H "Authorization: Bearer $MYAUTOAPI_API_KEY" \
  -F "model=gpt-image-2" \
  -F "image=@input.png" \
  -F "prompt=在保持主体不变的前提下，把背景改成明亮的办公室场景。"
```

如果编辑请求失败，先确认账号权限、模型分组、文件字段名、图片格式和接口返回错误。稳定上线前，先用最小图片和最短提示词做一次独立验证。

## 常见问题

### 浏览器直连失败或 CORS 报错

不要把 API Key 放在浏览器前端直接请求图片接口。浏览器直连可能遇到 CORS 限制，也会暴露密钥。由你的后端服务请求 MyAutoAPI，再把生成结果或保存后的图片地址返回给前端。

### Base URL 填错

完整生图端点是 `https://api.myautoapi.com/v1/images/generations`，但普通客户端的 `Base URL` 只填根地址或 `/v1` 地址。不要把 `/v1/images/generations` 填进普通聊天客户端的 `Base URL`。

更多说明见 [外接兼容与 Base URL 说明](/external/base-url)。

### 模型列表能拉取，不代表生图一定能用

`GET /v1/models` 能返回模型列表，只能说明模型列表接口可访问。图片生成还要看当前 API Key 的分组、账号权限、模型可用状态和图片端点是否允许调用。

模型和分组说明见 [模型与分组](/models)。

### 保存后的文件打不开

检查是不是把 `data[].b64_json` 原样写进了图片文件，或者在字段为空时写出了空文件。正确做法是先解析 JSON：有 `b64_json` 就 Base64 解码，有 `url` 就下载图片；两者都没有有效图片数据时，保留原始返回并报错。

### 429 或 5xx 怎么处理

`429` 表示当前请求过快、额度不足或触发限流；`5xx` 表示服务端或上游临时异常。先做短暂退避后重试，避免高频并发重放同一个请求。如果持续失败，保留请求时间、模型名、端点、错误码和返回内容，再按 [报错与踩坑](/errors/) 排查。
