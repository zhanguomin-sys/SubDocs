# 客户端接入

第一次使用 Codex App，先看 [快速开始](/quick-start)；不想运行脚本，看 [Codex CC Switch 接入](/codex-cc-switch)。

已经完成 Codex 接入，或者要把 MyAutoAPI 接到其它工具，再按工具类型进入对应教程。

## 模型怎么填

下面适用于 Cherry Studio、Open WebUI、Trae、LibreChat 等普通 OpenAI-compatible 客户端。Codex、Claude Desktop 和 Claude Code CLI 有专门配置方式，按各自教程填写。

普通 OpenAI-compatible 客户端只需要改模型名，不需要为每个模型重新创建地址和 Key：

| 字段 | 填写 |
| --- | --- |
| Base URL | `https://api.myautoapi.com/v1` |
| API Key | MyAutoAPI 管理后台创建的 API Key |
| Model | 当前 Key 分组可用模型名 |

模型名填一个，不要把下面几项一起填进去：

| 想使用的模型 | Model 示例 |
| --- | --- |
| GPT | `gpt-5.5` |
| Claude | `claude-opus-5`、`claude-fable-5` |
| Gemini | `gemini-3.6-flash` |
| Grok | `grok-4.5` |

上面的模型名只有在当前 API Key 所属分组支持时才能使用。后台分组里没有的模型，会返回 `model not found` 或 `403`。换模型时保留 Base URL 和 API Key，只改 Model。完整价格和模型 ID 看 [模型价格](/models)。

Codex 的 Responses 配置和 Claude Code / Desktop 的原生 Anthropic 配置，按各自教程填写，不要把普通客户端的 `/v1` 地址直接复制过去。

## 按工具类型选择

| 你要接入的工具 | 进入 | 配置入口 |
| --- | --- | --- |
| 第一次接入 Codex | [快速开始](/quick-start) / [CC Switch 导入（Codex 用）](/codex-cc-switch) | 一键脚本或图形工具 |
| 手动配置 Codex | [Codex 接入](/clients/codex) / [Codex 手动接入](/clients/codex-manual-config) / [保留 ChatGPT 登录](/clients/chatgpt-login-myautoapi) | Codex 配置文件 |
| 桌面聊天客户端 | [Cherry Studio](/clients/cherry-studio) | 模型服务设置 |
| JetBrains IDE | [JetBrains 接入](/clients/jetbrains) | 进入页面按目标选择 |
| 其它 IDE / 编辑器 | [VS Code](/clients/vscode) / [Trae](/clients/trae) | IDE 内的 AI 设置 |
| Kimi Code Agent（Windows） | [Kimi Code 接入 Kimi K3](/kimi-code) | Kimi Web 的供应商设置 |
| 自部署聊天面板（需要 Docker） | [Open WebUI](/clients/open-webui) / [LibreChat](/clients/librechat) | 管理后台或项目配置文件 |
| Agent / Gateway | [OpenClaw](/clients/openclaw) / [Hermes](/clients/hermes) | Gateway 配置或命令行向导 |
| Claude Code CLI | [后台一键导入到 CC Switch](/clients/claude-code-desktop#claude-code-cli-ccswitch) / [CC Switch 导入确认](/clients/cc-switch#ccswitch-claude-code-import) | Claude Code 供应商 |
| Claude Desktop APP | [优先用 CC Switch 手动配置](/clients/claude-code-desktop#claude-desktop-ccswitch) / [Gateway 备用配置](/clients/claude-code-desktop#claude-desktop-gateway) | CC Switch 或 APP 内第三方推理设置 |
| 图片创作 | [Codex 内置生图](/images/codex-image-direct) / [无限画布](/images/infinite-canvas) | 生图工具或画布设置 |
