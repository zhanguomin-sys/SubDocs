# Cherry Studio 接入

Cherry Studio 是桌面聊天客户端，适合想用图形界面管理模型、直接聊天的个人用户。新增 `MyAutoAPI` 服务商后，聊天请求会通过 `https://api.myautoapi.com/v1` 发出。

## 安装 Cherry Studio

打开下载页：

```text
https://www.cherry-ai.com/download
```

| 系统 | 安装方式 |
| --- | --- |
| Windows | 下载 x64 或 ARM64 安装版 |
| macOS | 按芯片选择 Intel 或 Apple Silicon 版本，下载后拖入 Applications |
| Linux | 按页面提供的 x86_64 或 ARM64 包安装 |

安装完成后打开 Cherry Studio。

## 添加服务商

进入 **Settings / 设置** -> **Model Services / 模型服务**，点击 **+ Add / 添加**。

| 字段 | 填写 |
| --- | --- |
| Provider name / 服务商名称 | `MyAutoAPI` |
| Provider type / 服务商类型 | `OpenAI` |

保存后，在服务商列表里打开 `MyAutoAPI`。

## 填写 Key 和地址

| 字段 | 填写 |
| --- | --- |
| API Key | MyAutoAPI API Key |
| API address / API 地址 | `https://api.myautoapi.com/v1` |
| Enable / 启用 | 打开 |

Base URL 只填到 `/v1`，不要追加其它路径。

## 添加模型

在 `MyAutoAPI` 服务商里打开 **Model management / 模型管理**，点击 **+ Add / 添加**，填入当前 Key 分组可用模型。

示例：

```text
claude-fable-5
```

也可以填 `gpt-5.5`、`claude-opus-5`、`gemini-3.6-flash` 或 `grok-4.5`。模型名要和当前 Key 分组里的模型完全一致。

## 测试

回到聊天界面：

1. 在模型选择器里选择 `MyAutoAPI`。
2. 选择刚刚添加的模型。
3. 发送：

```text
测试
```

能正常回复，说明 Cherry Studio 已经通过 MyAutoAPI 发起请求。

## 常见错误

| 现象 | 处理 |
| --- | --- |
| `401 Unauthorized` | 重新复制 API Key，确认填在 API Key 字段 |
| 路径错误 / Not Found | API 地址改回 `https://api.myautoapi.com/v1` |
| 模型不存在 | 换成当前 Key 分组里的完整模型名 |
| 模型列表没有自动出现 | 在模型管理里手动添加模型 ID |
| `403 Forbidden` / `block` | 先确认 Key、模型名和服务商启用状态；仍失败时看 [User-Agent 说明](/external/user-agent) |
