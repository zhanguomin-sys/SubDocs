# VS Code 接入

VS Code 有两种接入方式，按你的需求选择。

## 两条路线

| 目标 | 入口 | 配置位置 |
| --- | --- | --- |
| 用 Codex 扩展读项目、改代码、跑任务 | OpenAI Codex 扩展 | Codex 的 `.codex` 配置 |
| 让 VS Code Chat / Copilot 使用第三方模型 | VS Code BYOK | VS Code 的 Language Models 配置 |

这两条路线互不影响。Codex 扩展不会读取 VS Code BYOK；VS Code Chat 也不会自动读取 Codex 的 `.codex` 配置。

## Codex 扩展

先把本中转站的模型配置填进 Codex，让 Codex 请求通过 MyAutoAPI 发出。已经按 [Codex 接入](/clients/codex) 配好后，直接在 VS Code 扩展市场安装：

```text
OpenAI Codex
```

安装后打开项目根目录，在 Codex 面板里直接使用。项目根目录要包含代码、文档和 Git 信息，不要只打开单个文件。

## BYOK

BYOK 是 **Bring Your Own Key**。它的作用是把你自己的模型 API Key 接进 VS Code 的 Chat / Agent 体验。

VS Code Chat 使用本中转站模型时，需要配置 BYOK。它和 Codex 扩展没有配置联动。

打开命令面板：

```text
Chat: Manage Language Models
```

选择：

```text
Add Models -> Custom Endpoint
```

MyAutoAPI 按 Custom Endpoint 配。表单里先填：

| 项目 | 填什么 |
| --- | --- |
| Group name | `MyAutoAPI` |
| API Key | MyAutoAPI 管理后台创建的 API Key |
| API type | `Responses` |

VS Code 打开 `chatLanguageModels.json` 后，关键字段按下面写：

```json
[
  {
    "name": "MyAutoAPI",
    "vendor": "customendpoint",
    "apiKey": "这里完整填写你的中转站 API Key",
    "apiType": "responses",
    "models": [
      {
        "id": "claude-fable-5",
        "name": "claude-fable-5",
        "url": "https://api.myautoapi.com/v1/responses",
        "toolCalling": true,
        "vision": true,
        "maxInputTokens": 128000,
        "maxOutputTokens": 64000
      }
    ]
  }
]
```

配置后，在 VS Code Chat 的模型选择器里选中刚添加的模型。只有选中了这个 BYOK 模型，VS Code Chat 的请求才会通过对应的第三方 API 发出。

`id` 和 `name` 也可以改成当前 Key 分组里的 `gpt-5.5`、`claude-opus-5`、`gemini-3.6-flash` 或 `grok-4.5`；`url` 和 `apiKey` 不变。模型不在当前分组时，VS Code 会显示模型不可用。

BYOK 主要覆盖 VS Code 的 Chat 体验、工具调用和工具任务。代码补全、语义搜索、依赖 embeddings 的能力，仍受 GitHub 登录、Copilot 支持或组织策略影响。

## 兼容接口

只有当前入口明确要求旧聊天兼容接口时，再把 `apiType` 和 `url` 改成：

```json
"apiType": "chat-completions",
"url": "https://api.myautoapi.com/v1/chat/completions"
```

## 验证

| 要验证什么 | 怎么测 |
| --- | --- |
| Codex 扩展是否使用 MyAutoAPI | 在 Codex 面板发起一个小代码任务 |
| VS Code Chat 是否使用 MyAutoAPI | 在 Chat 模型选择器选中 `MyAutoAPI / 当前模型名` 后发一条短消息 |
