# V0.1 Python 参考实现

本目录保留早期 Python 参考实现和兼容测试源码，不是当前客户端安装入口。正式工具是项目根目录的 Go 原生程序，客户端不需要 Python、Node 或 SDK。

正式 V0.7.6 工具支持 `gpt-image-2`、`gpt-image-1k-th`、`gpt-image-2-adobe`、`grok-imagine-image`、`nano-banana-2` 和 `nano-banana-pro` 文生图；除 Adobe 外，其余模型还支持参考图和修图。工具从 `$CODEX_HOME/tools/myautoapi-imagines-tool/api-key` 读取“专门画图分组 Key”，命令不接受 `--api-key` 或 `--base-url`。

## 正式调用约定

Apple Silicon 或 Intel Mac：

```bash
tool_codex_home="${CODEX_HOME:-$HOME/.codex}"
"$tool_codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool" generate \
  --prompt "一只红色小鸟站在冬日树枝上，柔和自然光" \
  --model gpt-image-2 \
  --size 1024x1024 \
  --quality high \
  --output-name "冬日树枝上的红色小鸟" \
  --output-dir "$PWD/images"
```

Windows PowerShell：

```powershell
$ToolCodexHome = if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) { Join-Path $HOME ".codex" } else { $env:CODEX_HOME }
$ToolPath = Join-Path $ToolCodexHome "tools\myautoapi-imagines-tool\myautoapi-imagines-tool.exe"
& $ToolPath generate `
  --prompt "一只红色小鸟站在冬日树枝上，柔和自然光" `
  --model gpt-image-2 `
  --size 1024x1024 `
  --quality high `
  --output-name "冬日树枝上的红色小鸟" `
  --output-dir (Join-Path $PWD.Path "images")
```

参考图和修图把命令改成 `edit`，并通过一个或多个 `--image "/输入图片路径.png"` 传入图片。同一提示词需要多张结果时使用 `--n 3`。

默认模型参数是 `1024x1024`、`high` 和 `n=1`。Grok Imagine 与 Nano Banana 使用 `--resolution` 和 `--aspect-ratio`；Nano Banana 不使用 `--quality`。正式 Skill 根据用户要求提供尺寸、质量、数量、输出目录和准确文件名。成功后 `images` 目录只出现按真实格式保存的最终图片；单图 stdout 返回 `output_file`，多图返回 `output_files`，失败时 stderr 返回脱敏错误。
