---
sidebarTitle: Cherry Studio
---

# Cherry Studio 接入

Cherry Studio 是桌面聊天客户端，用图形界面管理模型、直接聊天。在设置里新增 `MyAutoAPI` 服务商后，聊天请求走 `https://api.myautoapi.com/v1`。

## 1. 安装 Cherry Studio

从 [下载页](https://www.cherry-ai.com/download) 下载安装：

| 系统 | 安装包 |
| --- | --- |
| Windows | x64 或 ARM64 安装版 |
| macOS | 按芯片选 Intel 或 Apple Silicon 版，下载后拖进 Applications |
| Linux | 页面提供的 x86_64 或 ARM64 包 |

## 2. 添加 MyAutoAPI 服务商

1. 进入 **设置 / Settings -> 模型服务 / Model Services**，点击 **添加 / + Add**。
2. 服务商名称填 `MyAutoAPI`，服务商类型选 `OpenAI`，保存。
3. 在服务商列表打开 `MyAutoAPI`，填写：

| 字段 | 填写 |
| --- | --- |
| API Key | MyAutoAPI API Key |
| API 地址 / API address | `https://api.myautoapi.com/v1`，只填到 `/v1` |
| 启用 / Enable | 打开 |

4. 打开 **模型管理 / Model management**，点击 **添加 / + Add**，填一个当前 Key 分组可用的完整模型名，例如 `claude-fable-5-1`、`claude-opus-5-5`、`gpt-6.1-sol`、`gemini-3.8-flash` 或 `grok-4.7`。Claude 模型要用 **CC MAX 外接分组** 的 Key。

## 3. 测试

回到聊天界面，新建对话，在模型选择器里选 `MyAutoAPI` 和刚添加的模型，发送 `测试`。能正常回复就接入完成。

## 常见错误

| 现象 | 处理 |
| --- | --- |
| `401 Unauthorized` | 重新复制 API Key，填在 API Key 字段 |
| 路径错误 / Not Found | API 地址改回 `https://api.myautoapi.com/v1` |
| 模型不存在 | 换成当前 Key 分组里的完整模型名 |
| 模型列表没有自动出现 | 在模型管理里手动添加模型 ID |
| `403 Forbidden` / `block` | 核对 Key、模型名，确认服务商已启用；仍失败看 [User-Agent 说明](/external/user-agent) |
