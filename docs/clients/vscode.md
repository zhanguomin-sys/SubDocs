---
sidebarTitle: VS Code
---

# VS Code 接入

VS Code 里有两个互不相关的 AI 入口，配置各填各的：

| 目标 | 入口 | 配置位置 |
| --- | --- | --- |
| 用 Codex 读项目、改代码、跑任务 | OpenAI Codex 扩展 | Codex 的 `.codex` 配置 |
| 让 VS Code Chat / Copilot 用第三方模型 | VS Code BYOK | VS Code 的 `chatLanguageModels.json` |

## Codex 扩展

1. 按 [Codex 接入](/codex/) 把 MyAutoAPI 的模型配置填进 Codex。
2. 在 VS Code 扩展市场搜索安装 `OpenAI Codex`。
3. 打开项目根目录（包含代码、文档和 Git 信息，不要只打开单个文件），在 Codex 面板新建任务，发一个小代码任务测试。

## VS Code Chat（BYOK）

BYOK（Bring Your Own Key）把你自己的 API Key 接进 VS Code 的 Chat / Agent。

1. 打开命令面板，运行 `Chat: Manage Language Models`。
2. 选择 **Add Models -> Custom Endpoint**，表单里填：

| 项目 | 填写 |
| --- | --- |
| Group name | `MyAutoAPI` |
| API Key | MyAutoAPI API Key |
| API type | `Responses` |

3. VS Code 打开 `chatLanguageModels.json` 后，按下面写好关键字段并保存：

```json
[
  {
    "name": "MyAutoAPI",
    "vendor": "customendpoint",
    "apiKey": "这里完整填写你的中转站 API Key",
    "apiType": "responses",
    "models": [
      {
        "id": "claude-fable-5-1",
        "name": "claude-fable-5-1",
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

4. 新建 Chat 对话，在模型选择器里选中 `MyAutoAPI / claude-fable-5-1`，发送 `测试`。没选中这个模型时，Chat 请求不走 MyAutoAPI。

换模型时只改 `id` 和 `name`，例如当前 Key 分组里的 `gpt-6.1-sol`、`claude-opus-5-5`、`gemini-3.8-flash`、`grok-4.7`；`url` 和 `apiKey` 不变。示例里的 `claude-fable-5-1` 要用 **CC MAX 外接分组** 的 Key。模型不在当前分组时，VS Code 显示模型不可用。

BYOK 覆盖 Chat、工具调用和工具任务。代码补全、语义搜索和依赖 embeddings 的功能，仍受 GitHub 登录、Copilot 支持和组织策略控制。

当前入口明确要求旧的聊天兼容接口时，把 `apiType` 和 `url` 改成：

```json
"apiType": "chat-completions",
"url": "https://api.myautoapi.com/v1/chat/completions"
```
