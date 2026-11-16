---
sidebarTitle: Trae
---

# Trae 接入

Trae 是字节跳动的 AI 编辑器，自带聊天、代码编辑和 Agent 功能。在 Trae 的模型设置里添加 MyAutoAPI 的自定义模型后，这些功能用的模型就走 MyAutoAPI。

<a id="添加-provider"></a>
<a id="添加模型"></a>

## 1. 添加自定义模型

进入 **设置 -> 模型**，点击 **添加模型**，选择 **自定义模型**，填写：

| 字段 | 填写 |
| --- | --- |
| API 格式 | `OpenAI Chat Completions` |
| 请求地址 | 打开 **完整 URL**，填 `https://api.myautoapi.com/v1/chat/completions` |
| 模型 ID | 当前 Key 分组可用的完整模型名，例如 `claude-fable-5-1`、`claude-opus-5-5`、`gemini-3.8-flash`、`grok-4.7`；Claude 模型要用 **CC MAX 外接分组** 的 Key |
| API Key | MyAutoAPI API Key |

点击 **添加模型** 保存。Trae 会用这组信息请求一次 MyAutoAPI，失败时在窗口里显示错误和服务商日志。

## 2. 选择模型并测试

在聊天或 Agent 的模型选择器里选刚添加的模型，新建对话，发送 `测试`。能正常回复，就说明当前功能已经切到 MyAutoAPI。

先确认聊天能正常回复，再测试 Agent 的工具调用、代码编辑、联网和文件操作。

## 排查

| 现象 | 处理 |
| --- | --- |
| 添加时报地址错误或 404 | 确认 **完整 URL** 已打开，地址是 `https://api.myautoapi.com/v1/chat/completions` |
| 仍然在用官方模型 | 回到当前聊天或 Agent 的模型选择器，重新选刚添加的模型 |
| `401 Unauthorized` | 重新复制 API Key，保存后重新选择模型，仍失败就重启 Trae |
| `model not found` | 换成当前 Key 分组里的完整模型名 |
| Agent 调用工具时报 `400` | 先看返回的错误原文；是接口或工具调用不兼容时，换当前 Key 支持的 Claude 模型，新建对话重试 |
| `403 Forbidden` / `block` | 换成 Key 分组支持的模型；仍失败看 [User-Agent 说明](/external/user-agent) |
| 网络或证书错误 | 检查系统代理、公司网络、证书信任和代理插件 |
