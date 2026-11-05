---
sidebarTitle: WorkBuddy
---

# WorkBuddy 接入腾讯混元 hy4

WorkBuddy 是 AI 办公 Agent。腾讯混元模型 `hy4-preview` 擅长长程编程、复杂办公任务和工具调用。本文在 WorkBuddy 里添加 `hy4-preview`，让请求走 MyAutoAPI；配置保存在本机 `workbuddy/models.json`，不用手动改文件。文末附手动 API 请求参数。

需要一个 **腾讯混元分组** 的 MyAutoAPI API Key，创建方法看 [快速开始](/quick-start)。

## 1. 安装并打开 WorkBuddy

从 [WorkBuddy 官网](https://www.codebuddy.cn/work/) 下载安装，完成后登录。

## 2. 添加模型

打开 **设置 -> 模型**，点击 **添加模型**，提供商选择 **自定义 / Custom**，填写：

| 字段 | 填写 |
| --- | --- |
| URL | <code>https://api.myautoapi.com/<wbr>v1/<wbr>chat/<wbr>completions</code> |
| API Key | 腾讯混元分组的 MyAutoAPI API Key |
| 模型名称 | `hy4-preview` |

然后设置开关：

- 关闭 **自定义协议**。
- 高级配置里打开 **工具调用** 和 **推理模式**，关闭 **图片输入**。
- 输入和输出上限留空。

## 3. 保存并测试

点击 **保存**，关闭设置，新建任务（不要沿用旧任务）。在模型选择器选 `hy4-preview`，发送 `测试`，能正常回复就接入完成。

返回 `model not found` 时，换成腾讯混元分组的 Key，模型名改成 `hy4-preview`。

<details>
<summary>手动组装 API 请求参数（脚本调用、接口调试用）</summary>

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

<a id="请求体参数"></a>

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `model` | 是 | 固定填 `hy4-preview` |
| `messages` | 是 | 对话消息数组，每项包含 `role` 和 `content` |
| `stream` | 否 | `true` 返回流式结果，`false` 返回完整结果 |
| `temperature` | 否 | 控制输出随机性 |
| `max_tokens` | 否 | 限制本次输出的 Token 数量 |
| `tools` | 否 | 可调用的工具定义 |
| `tool_choice` | 否 | 控制是否调用工具，可填 `auto` |

非流式结果的文本在 `choices[0].message.content`，用量在 `usage`。请求失败时记录 HTTP 状态码和响应里的 `error.message`。

</details>
