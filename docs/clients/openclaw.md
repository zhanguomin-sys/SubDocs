---
sidebarTitle: OpenClaw
---

# OpenClaw 接入

OpenClaw 是自托管的 AI Agent Gateway（官方页面的龙虾形象叫 Molty），一个 Gateway 同时接 Telegram、WhatsApp、Discord、Slack、iMessage、WebChat 和移动节点。在 `~/.openclaw/openclaw.json` 里添加 MyAutoAPI provider 后，这些入口统一通过 MyAutoAPI 使用模型。

## 1. 安装 OpenClaw

::: code-group

```powershell [Windows]
iwr -useb https://openclaw.ai/install.ps1 | iex
```

```bash [Mac / Linux]
curl -fsSL https://openclaw.ai/install.sh | bash
```

:::

自己管理 Node 时改用 `npm install -g openclaw@latest`，Node 推荐 24，22.19+ 也支持。

## 2. 启动 Gateway

```bash
openclaw onboard --install-daemon
```

向导按默认或 QuickStart 走完，模型和渠道先跳过。

## 3. 添加 MyAutoAPI provider

打开 `~/.openclaw/openclaw.json`，把下面这段合并进去，保留已有的 channel、gateway、agent 等配置：

```json
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "myautoapi/claude-fable-5-1"
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
            "id": "claude-fable-5-1",
            "name": "claude-fable-5-1",
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

| 要改的字段 | 改成 |
| --- | --- |
| `apiKey` | MyAutoAPI API Key |
| `agents.defaults.model.primary` | `myautoapi/` + 当前 Key 分组可用的模型名 |
| `models.providers.myautoapi.models[0].id` | 同一个模型名 |
| `models.providers.myautoapi.models[0].name` | 和 `id` 一致 |

`baseUrl` 保持 `https://api.myautoapi.com/v1`（只到 `/v1`），`api` 保持 `openai-completions`。示例里的 `claude-fable-5-1` 要用 **CC MAX 外接分组** 的 Key。

## 4. 重启并测试

```bash
openclaw gateway restart
openclaw models list --provider myautoapi
openclaw dashboard
```

列表里出现 `myautoapi` 的模型后，在 Control UI 里新建对话，发送 `测试`。能正常回复，就说明当前对话用的是 `myautoapi/模型名`。

## 进阶：Key 不写进配置文件

把 `apiKey` 改成 `"${MYAUTOAPI_API_KEY}"`，在启动 OpenClaw 的环境里设置变量，再重启 Gateway：

::: code-group

```powershell [Windows]
$env:MYAUTOAPI_API_KEY="sk-你的MyAutoAPI_API_Key"
```

```bash [Mac / Linux]
export MYAUTOAPI_API_KEY="sk-你的MyAutoAPI_API_Key"
```

:::

OpenClaw 支持多个 API adapter。普通接入保持 `openai-completions`；确实要换 adapter 时，只改同一个 provider 里的 `api` 字段，Base URL、模型路径和 provider 名不动。

## 排查

| 现象 | 处理 |
| --- | --- |
| 看不到 `myautoapi` provider | 检查 `openclaw.json` 是否合并成功，再运行 `openclaw models list --provider myautoapi` |
| 模型不可用 | 把 `claude-fable-5-1` 换成当前 Key 分组里的完整模型 ID |
| `401 Unauthorized` | 重新复制 API Key；用环境变量时设置后重启 Gateway |
| Base URL 路径错误 | `baseUrl` 改回 `https://api.myautoapi.com/v1` |
| 配置没有生效 | 运行 `openclaw gateway restart`，确认当前对话选的是 `myautoapi/模型名` |
| `403 Forbidden` / `block` | 核对 Key、模型、Base URL；仍失败看 [User-Agent 说明](/external/user-agent) |

排查 Header 时，给 provider 加一个稳定的客户端标识即可，不要伪装成 Codex、Claude CLI 或浏览器。
