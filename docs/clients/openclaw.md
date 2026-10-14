# OpenClaw 接入

OpenClaw 官方页面里的“龙虾”指 Molty，是 OpenClaw 的产品形象；看到龙虾或 Molty，说的就是 OpenClaw。

OpenClaw 是自托管 AI Agent Gateway，一个 Gateway 可以同时接 Telegram、WhatsApp、Discord、Slack、iMessage、WebChat 和移动节点。添加 MyAutoAPI model provider 后，这些入口会统一通过 MyAutoAPI 使用模型。

## 安装 OpenClaw

macOS / Linux：

```bash
curl -fsSL https://openclaw.ai/install.sh | bash
```

Windows PowerShell：

```powershell
iwr -useb https://openclaw.ai/install.ps1 | iex
```

已经自己管理 Node 时，用 npm 安装：

```bash
npm install -g openclaw@latest
```

OpenClaw 推荐 Node 24，Node 22.19+ 也支持。

## 启动 Gateway

运行初始向导：

```bash
openclaw onboard --install-daemon
```

第一次使用按默认或 QuickStart 路径完成向导。模型和渠道可以先跳过，后面用浏览器 Control UI 验证。

打开控制台：

```bash
openclaw dashboard
```

能打开页面，说明 OpenClaw Gateway 已经跑起来。

## 添加 MyAutoAPI provider

打开配置文件：

```text
~/.openclaw/openclaw.json
```

把下面这段合并进去。不要覆盖你已有的 channel、gateway、agent 等配置。

```json
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "myautoapi/claude-fable-5"
      }
    }
  },
  "models": {
    "mode": "merge",
    "providers": {
      "myautoapi": {
        "baseUrl": "https://api.myautoapi.com/v1",
        "apiKey": "sk-把这里换成你的MyAutoAPI API Key",
        "api": "openai-completions",
        "models": [
          {
            "id": "claude-fable-5",
            "name": "claude-fable-5",
            "reasoning": true,
            "input": ["text"],
            "contextWindow": 200000,
            "maxTokens": 8192,
            "cost": {
              "input": 0,
              "output": 0,
              "cacheRead": 0,
              "cacheWrite": 0
            }
          }
        ]
      }
    }
  }
}
```

必须改的字段：

| 字段 | 怎么改 |
| --- | --- |
| `apiKey` | 换成 MyAutoAPI API Key |
| `agents.defaults.model.primary` | `myautoapi/` 后面的模型名换成当前 Key 分组可用模型 |
| `models.providers.myautoapi.models[0].id` | 换成同一个模型名 |
| `models.providers.myautoapi.models[0].name` | 和 `id` 保持一致 |

保留 `baseUrl: "https://api.myautoapi.com/v1"` 和 `api: "openai-completions"`。Base URL 只填到 `/v1`，不要追加其它路径。

## 重启并测试

保存配置后重启 Gateway：

```bash
openclaw gateway restart
```

查看 OpenClaw 是否读到新 provider：

```bash
openclaw models list --provider myautoapi
```

打开 Control UI：

```bash
openclaw dashboard
```

发送：

```text
测试
```

能正常回复，说明当前对话已使用 `myautoapi/模型名`。

## 进阶配置

不想把 Key 明文写进配置文件时，把 `apiKey` 改成环境变量引用：

```json
{
  "apiKey": "${MYAUTOAPI_API_KEY}"
}
```

然后在启动 OpenClaw 的环境里设置变量，并重启 Gateway。

macOS / Linux：

```bash
export MYAUTOAPI_API_KEY="sk-你的MyAutoAPI_API_Key"
```

Windows PowerShell：

```powershell
$env:MYAUTOAPI_API_KEY="sk-你的MyAutoAPI_API_Key"
```

OpenClaw 支持多个 API adapter。普通 MyAutoAPI 接入保持 `openai-completions`；只有明确要切换 adapter 时，才改同一个 provider 里的 `api` 字段，不要同时改 Base URL、模型路径和 provider 名。

## 排查

| 现象 | 处理 |
| --- | --- |
| 看不到 `myautoapi` provider | 检查 `openclaw.json` 是否合并成功，再运行 `openclaw models list --provider myautoapi` |
| 模型不可用 | 把 `claude-fable-5` 换成当前 Key 分组里的完整模型 ID |
| `401 Unauthorized` | 重新复制 API Key；使用环境变量时，重启 Gateway |
| Base URL 路径错误 | `baseUrl` 改回 `https://api.myautoapi.com/v1` |
| 配置没有生效 | 运行 `openclaw gateway restart`，确认当前对话选择的是 `myautoapi/模型名` |
| `403 Forbidden` / `block` | 先确认 Key、模型、Base URL；仍失败时看 [外接调用 User-Agent 说明](/external/user-agent) |

需要排查 Header 时，再给 provider 添加稳定的客户端标识。普通接入不要伪装成 Codex、Claude CLI 或浏览器。
