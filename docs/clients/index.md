---
sidebarTitle: 其他工具
---

# 其他工具接入

本目录是 VS Code、JetBrains、Gemini CLI、Open WebUI、LibreChat、OpenClaw、Hermes 的接入教程。Codex、Claude 和国产工具各有目录：[Codex](/codex/)、[Claude](/claude/)、[国产工具](/domestic/)。

## 模型怎么填

普通 OpenAI-compatible 客户端（Cherry Studio、Open WebUI、Trae、LibreChat 等）都填这三项，换模型时只改 Model，Base URL 和 API Key 不动：

| 字段 | 填写 |
| --- | --- |
| Base URL | `https://api.myautoapi.com/v1` |
| API Key | MyAutoAPI API Key |
| Model | 当前 Key 分组可用的一个模型名 |

| 想用的模型 | Model 示例 |
| --- | --- |
| GPT | `gpt-6.1-sol` |
| Claude | `claude-opus-5-5`、`claude-fable-5-1` |
| Gemini | `gemini-3.8-flash` |
| Grok | `grok-4.7` |

第三方客户端用 Claude 模型，Key 必须选 **CC MAX 外接分组**；其它 Claude 分组只能接 Claude Code / Claude Desktop，在这里会被拒绝。Key 所属分组里没有的模型，会返回 `model not found` 或 `403`。完整模型 ID 和价格看 [模型价格](/models)。

Codex（Responses 配置）和 Claude Code / Desktop（原生 Anthropic 配置）按各自教程填写，不要把这里的 `/v1` 地址直接复制过去。

## 按工具选择

| 工具 | 教程 | 配置入口 |
| --- | --- | --- |
| VS Code | [VS Code](/clients/vscode) | IDE 内的 AI 设置 |
| JetBrains IDE | [JetBrains](/clients/jetbrains) | 进入页面按目标选择 |
| 自部署聊天面板（需要 Docker） | [Open WebUI](/clients/open-webui) / [LibreChat](/clients/librechat) | 管理后台或项目配置文件 |
| 终端编程 Agent（Gemini 模型） | [Gemini CLI](/clients/gemini-cli) | `.gemini/.env` |
| Agent / Gateway | [OpenClaw](/clients/openclaw) / [Hermes](/clients/hermes) | Gateway 配置或终端向导 |
| 图片创作 | [安装生图工具](/image-video-group-image) / [无限画布](/images/infinite-canvas) | 生图工具或画布设置 |
