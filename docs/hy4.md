# 腾讯混元模型 hy4 使用指南

腾讯混元模型 `hy4-preview` 适合长程编程、复杂办公任务和工具调用。本文主要讲如何在 WorkBuddy 中接入本站并使用该模型，文末另附手动 API 请求参数。WorkBuddy 配置保存在本机 `workbuddy/models.json`，无需手动修改文件。

## 在 WorkBuddy 使用 hy4-preview

### 1. 安装并打开 WorkBuddy

打开 [WorkBuddy 官网](https://www.codebuddy.cn/work/) 下载并安装，完成后登录。已经安装则直接打开。

按 [快速开始](/quick-start) 创建 MyAutoAPI API Key，分组选择 **国产模型大全 0.45x**。

### 2. 添加模型

打开 **设置 -> 模型**，点击 **添加模型**，提供商选择 **自定义 / Custom**。

| 字段 | 填写 |
| --- | --- |
| URL | <code>https://api.myautoapi.com/<wbr>v1/<wbr>chat/<wbr>completions</code> |
| API Key | MyAutoAPI API Key |
| 模型名称 | `hy4-preview` |

关闭 **自定义协议**。在高级配置中打开 **工具调用** 和 **推理模式**，关闭 **图片输入**；输入和输出上限留空。

### 3. 保存并测试

点击 **保存**，关闭设置后新建任务，不要沿用旧任务。

在模型选择器中选择 `hy4-preview`，发送：

```text
测试
```

能正常回复即接入完成。返回 `model not found` 时，确认当前 API Key 属于 **国产模型大全 0.45x** 分组，并检查模型名是否为 `hy4-preview`。

<details>
<summary>手动组装 API 请求参数（非基础内容）</summary>

WorkBuddy 已接入的用户无需继续操作。下面的参数用于脚本调用和接口调试。

| 项目 | 值 |
| --- | --- |
| 请求方法 | `POST` |
| 接口地址 | <code>https://api.myautoapi.com/<wbr>v1/<wbr>chat/<wbr>completions</code> |
| 鉴权 | `Authorization: Bearer <MyAutoAPI API Key>` |
| 内容类型 | `Content-Type: application/json` |

最小请求：

```bash
curl -X POST 'https://api.myautoapi.com/v1/chat/completions' \
  -H 'Authorization: Bearer <MyAutoAPI API Key>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "hy4-preview",
    "messages": [
      { "role": "user", "content": "测试" }
    ],
    "stream": false
  }'
```

### 请求体参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `model` | 是 | 固定填写 `hy4-preview` |
| `messages` | 是 | 对话消息数组，每项包含 `role` 和 `content` |
| `stream` | 否 | `true` 返回流式结果，`false` 返回完整结果 |
| `temperature` | 否 | 控制输出随机性 |
| `max_tokens` | 否 | 限制本次输出的 Token 数量 |
| `tools` | 否 | 提供可调用的工具定义 |
| `tool_choice` | 否 | 控制是否调用工具，可填 `auto` |

非流式文本读取 `choices[0].message.content`，用量读取 `usage`。请求失败时记录 HTTP 状态码和响应中的 `error.message`。

</details>
