# JetBrains 接入

适用于 IntelliJ IDEA、PyCharm、WebStorm、GoLand、PhpStorm 等 JetBrains IDE。

JetBrains 里先按目标选入口。Claude Code 可以安装 Anthropic 的插件；Codex 可以在 JetBrains 的 AI Assistant 中使用，但不能把 MyAutoAPI API Key 填给 JetBrains 的 Codex 集成；只有普通 AI Chat 才使用下面的 OpenAI-compatible 配置。

| 你要做什么 | 推荐入口 | 说明 |
| --- | --- | --- |
| 在 JetBrains 里用 Claude Code | **Claude Code [Beta]** | Marketplace 里认准 Anthropic PBC 发布、Verified 的插件；本机仍要先安装 Claude Code |
| 在 JetBrains 里用 Codex | JetBrains AI Assistant 里的 **Codex** | 可以使用，但 BYOK 只接受 OpenAI 直接签发的 API Key，不能接 MyAutoAPI |
| 只想让 JetBrains AI Chat 使用 MyAutoAPI | **JetBrains AI Assistant** | 这是通用 AI Chat / BYOK 入口，不是 Claude Code，也不是 Codex |

## Claude Code

1. 打开 JetBrains IDE。
2. 进入 **Settings / Preferences -> Plugins -> Marketplace**。
3. 搜索 **Claude Code [Beta]**。
4. 确认发布方是 **Anthropic PBC**，并且插件标记为 Verified。
5. 安装后按提示重启 IDE。

这个插件要求本机已经安装 Claude Code。MyAutoAPI 的配置不要在 JetBrains AI Assistant 里填，先按 [Claude Code CLI 一键导入](/clients/claude-code-desktop#claude-code-cli-ccswitch) 或 [CC Switch 手动配置](/clients/cc-switch#ccswitch-claude-code-manual) 配好 Claude Code，再回到 JetBrains 里使用插件入口。

## Codex

JetBrains 新版 AI Assistant 里可以直接选择 **Codex**。打开 **AI Chat**，在智能体或模型选择器里选择 Codex，再按 JetBrains 的提示完成激活。

这条入口不支持 MyAutoAPI：JetBrains 官方说明，Codex 使用 BYOK 时必须填写 OpenAI 直接签发的 API Key，第三方中转站的 Key 不支持。不要在 AI Assistant 的 Codex 设置里填写 MyAutoAPI 的 Base URL 或 API Key。

需要在 JetBrains 项目里使用 MyAutoAPI 的 Codex 时，打开 JetBrains 的内置终端，使用已经按 [快速开始](/quick-start) 或 [Codex 手动接入](/clients/codex-manual-config) 配好的 Codex CLI。不要把 Marketplace 里发布方不明确的 Codex Launcher、Codex CLI、Codex GUI 插件当成 OpenAI 官方插件。

## AI Assistant

只需要 JetBrains 自带 AI Chat 接入 OpenAI-compatible 服务时，用 AI Assistant。

1. 进入 **Settings / Preferences -> Plugins -> Marketplace**。
2. 搜索并安装 **JetBrains AI Assistant**。
3. 重启 IDE 后，进入 **Settings / Preferences -> Tools -> AI Assistant -> Providers & API keys**。
4. 选择 **OpenAI-compatible**。

填写：

| 字段 | 填写 |
| --- | --- |
| URL | `https://api.myautoapi.com/v1` |
| API key | MyAutoAPI API Key |
| Model | 当前 Key 分组可用模型 |

模型可以填写：

```text
claude-opus-5
```

也可以改成当前分组里的 `claude-fable-5`、`gemini-3.6-flash`、`grok-4.5` 或 `gpt-5.5`。一次只填一个模型名，模型名必须和 MyAutoAPI 管理后台分组里的 ID 完全一致。

点击 **Test Connection**，测试通过后保存。打开 **AI Chat**，在顶部模型选择器里选择刚添加的 MyAutoAPI 模型，再发送 `测试`。

这条配置用于 AI Chat，不会把 JetBrains 里的 Codex 换成 MyAutoAPI。代码补全和下一步编辑也不一定会使用这里配置的通用聊天模型。

## 不要装这些

| 插件类型 | 处理 |
| --- | --- |
| 发布方不明确的 Claude / Codex GUI 插件 | 不作为默认推荐 |
| 要求单独安装官方 CLI 的插件 | 先安装官方 Claude Code 或 Codex CLI，再配置对应工具 |
| 想让 JetBrains 里的 Codex 使用 MyAutoAPI | 不使用 AI Assistant 的 Codex 集成，改用内置终端里的 Codex CLI |
| 只是 OpenAI-compatible AI Chat 的需求 | 用 AI Assistant，不要把它写成 Claude Code 或 Codex 接入 |
