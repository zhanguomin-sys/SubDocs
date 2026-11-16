---
sidebarTitle: JetBrains
---

# JetBrains 接入

适用于 IntelliJ IDEA、PyCharm、WebStorm、GoLand、PhpStorm 等 JetBrains IDE。三种用法入口不同，MyAutoAPI 的配置填的位置也不同：

| 要做什么 | 入口 | MyAutoAPI 配在哪 |
| --- | --- | --- |
| 在 JetBrains 里用 Claude Code | **Claude Code [Beta]** 插件 | 本机 Claude Code（CC Switch），插件里不填 |
| 在 JetBrains 里用 Codex | IDE 内置终端里的 Codex CLI | 本机 Codex 配置；AI Assistant 里的 Codex 不能接 MyAutoAPI |
| 让 JetBrains AI Chat 用 MyAutoAPI | **JetBrains AI Assistant** | AI Assistant 的 OpenAI-compatible 设置 |

## Claude Code

1. 先按 [Claude Code 接入](/claude/claude-code) 在本机安装 Claude Code 并接入 MyAutoAPI（也可以 [手动添加](/claude/claude-code#ccswitch-claude-code-manual)）。
2. 进入 **Settings / Preferences -> Plugins -> Marketplace**，搜索 **Claude Code [Beta]**。
3. 认准发布方 **Anthropic PBC**、带 Verified 标记的插件，安装后按提示重启 IDE。

## Codex

JetBrains 新版 AI Assistant 的 **AI Chat** 里能直接选 **Codex**，但 JetBrains 官方说明：Codex 走 BYOK 时只接受 OpenAI 直接签发的 API Key，不支持第三方中转站。不要在 AI Assistant 的 Codex 设置里填 MyAutoAPI 的 Base URL 或 API Key。

在 JetBrains 项目里用 MyAutoAPI 的 Codex：打开 IDE 内置终端，运行已经按 [快速开始](/quick-start) 或 [手动配置](/codex/manual-config) 配好的 Codex CLI。Marketplace 里发布方不明确的 Codex Launcher、Codex CLI、Codex GUI 插件不是 OpenAI 官方插件。

## AI Assistant

1. 进入 **Settings / Preferences -> Plugins -> Marketplace**，搜索安装 **JetBrains AI Assistant**，重启 IDE。
2. 进入 **Settings / Preferences -> Tools -> AI Assistant -> Providers & API keys**，选择 **OpenAI-compatible**，填写：

| 字段 | 填写 |
| --- | --- |
| URL | `https://api.myautoapi.com/v1` |
| API key | MyAutoAPI API Key |
| Model | 当前 Key 分组可用的一个模型名，例如 `claude-opus-5-5`、`claude-fable-5-1`、`gemini-3.8-flash`、`grok-4.7`、`gpt-6.1-sol`；Claude 模型要用 **CC MAX 外接分组** 的 Key |

3. 点击 **Test Connection**，通过后保存。
4. 打开 **AI Chat** 新建对话，在顶部模型选择器选刚添加的 MyAutoAPI 模型，发送 `测试`。

这条配置只用于 AI Chat：不会把 JetBrains 里的 Codex 换成 MyAutoAPI，代码补全和下一步编辑也不一定用这里的模型。
