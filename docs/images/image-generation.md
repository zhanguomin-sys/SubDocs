# 生图 API

自己写程序生成图片时看本页：用 `gpt-image-2.6-flare` 生成一张图并保存到本地。API Key 使用 **专门画图分组** 的 Key；其它模型和尺寸看 [生图参数汇总](/images/params)。

制作循环动画时，使用参数页的 [GIF 请求体](/images/params#gif)，将解码后的结果保存为 `.gif`。

## 懒人方法：让 AI 写一个测试脚本

把下面这段发给能操作本机文件的 AI：

```text
请帮我写一个测试 MyAutoAPI 生图接口的可运行脚本。

要求：
1. 只写一个脚本文件，文件名用中文，例如 `生成图片.py`、`生成图片.ps1` 或 `生成图片.sh`。
2. 不安装依赖；本机确实缺少工具时先告诉我缺什么，等我确认。
3. 调用接口：POST https://api.myautoapi.com/v1/images/generations，请求头 Authorization: Bearer <API_KEY>，API Key 留一个配置项让我自己填。
4. 请求体参考 gpt-image-2.6-flare 的 1K 方图调用方法，例如：
   {"model":"gpt-image-2.6-flare","prompt":"生成一张白底玻璃杯产品图","size":"1024x1024"}
5. 原始返回保存为 response.json；有 data[].b64_json 时先解码，没有时下载 data[].url；都没有有效图片时保留 response.json 并显示接口返回的错误信息。
6. 图片保存为 outputs/image.png。
```

## 生成一张图

把专门画图分组的 Key 放进环境变量：

::: code-group

```bash [Mac / Linux]
export MYAUTOAPI_API_KEY="你的 API Key"
```

```powershell [Windows PowerShell]
$env:MYAUTOAPI_API_KEY="你的 API Key"
```

:::

发送请求，原始返回保存为 `response.json`：

```bash
curl https://api.myautoapi.com/v1/images/generations \
  -H "Authorization: Bearer $MYAUTOAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2.6-flare",
    "prompt": "为一家现代咖啡店生成一张温暖明亮的产品宣传图，画面包含一杯拿铁、一块可颂和简洁的中文标题区域。",
    "size": "1024x1024"
  }' \
  -o response.json
```

图片在返回的 `data[].b64_json` 或 `data[].url` 里。下面用 Python 标准库把图片保存为 `outputs/image.png`：

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

## 修改图片

修改图片和参考图生成使用 `POST /v1/images/edits`，以表单上传原图：

```bash
curl https://api.myautoapi.com/v1/images/edits \
  -H "Authorization: Bearer $MYAUTOAPI_API_KEY" \
  -F "model=gpt-image-2.6-flare" \
  -F "image=@input.png" \
  -F "prompt=保持主体不变，把背景改成明亮的办公室场景。" \
  -o response.json
```

返回格式和生成接口相同，用上面的 Python 代码保存。

## 常见问题

| 遇到的问题 | 处理方法 |
| --- | --- |
| 返回 `401` | 检查环境变量里的 Key 是否完整，重新复制后再试 |
| 返回 `403` 或模型不可用 | 换成 **专门画图分组** 的 Key |
| 返回参数错误 | 删掉可选参数，只保留 `model`、`prompt`、`size` 再试；尺寸规则看 [生图参数汇总](/images/params) |
| 返回 `429` 或 `5xx` | 等几秒再重试，不要高频并发重放同一个请求 |
| 保存的图片打不开 | 先对 `b64_json` 做 Base64 解码再写文件，不要把原文直接写进图片 |
| 网页前端直接请求报 CORS 错误 | 不要在浏览器前端直接调用，Key 会泄露；由后端请求 MyAutoAPI，再把图片返回给前端 |
