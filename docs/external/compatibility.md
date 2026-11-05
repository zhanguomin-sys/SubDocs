---
sidebarTitle: Responses 与兼容模式
---

# 选择 Responses 还是兼容模式

MyAutoAPI 同时支持 **Responses 模式** 和 **OpenAI-compatible 兼容模式**。协议决定接口路径、请求体结构、工具调用和客户端怎么解析回复，按客户端支持的协议选，不按模型名选：

- Codex、支持 Responses API 的 Agent、新项目和偏工具调用的场景，用 **Responses 模式**。
- Cherry Studio、Open WebUI、LibreChat 这类普通聊天客户端，用 **兼容模式**。

## 两种模式的区别

| 对比项 | Responses 模式 | 兼容模式 |
| --- | --- | --- |
| 实际接口路径 | `/v1/responses` | `/v1/chat/completions` |
| 请求体 | 用 `input`、`instructions` 提交，结果在 `output`，用 `previous_response_id` 接续上下文 | 用 `messages` 对话列表提交，从 `choices[].message` 读回复 |
| 工具 / Agent 能力 | 适合工具调用、MCP、Web search、File search、Computer use、推理摘要 | 适合基础聊天和常见函数调用；复杂工具能力看客户端适配 |
| 模型与上下文 | 新模型、推理模型和 Agent 场景优先用 | 普通聊天够用；客户端截断上下文或不传相关字段时，只换模型名不会生效 |
| 后台日志特征 | `/v1/responses`、`output`、`previous_response_id` | `/v1/chat/completions`、`messages`、`choices` |

上面的接口路径由客户端自动拼接。多数客户端只填 Base URL `https://api.myautoapi.com/v1`，地址怎么填看 [Base URL 怎么填](/external/base-url)。

## 按客户端选择

| 客户端 / 场景 | 模式 | 怎么填 |
| --- | --- | --- |
| Codex | Responses | 复制后台或 [Codex 教程](/codex/) 给的配置（配置里有 `wire_api = "responses"`）；后台日志应出现 `/v1/responses` |
| 支持 Responses API 的 Agent / SDK | Responses | API 类型选 Responses，不要误选 Chat Completions；Base URL `https://api.myautoapi.com/v1` |
| Cherry Studio、Open WebUI、LibreChat | 兼容 | 按 OpenAI-compatible 添加，地址填法看 [Base URL 怎么填](/external/base-url) |
| 普通聊天、翻译、摘要 | 兼容即可 | 客户端只提供 OpenAI-compatible / Chat Completions 配置时按兼容模式 |
| 长上下文分组 | 看客户端协议 | 还要看客户端是否传出上下文、截断、缓存等配置 |
| 自写脚本 / 自定义前端 | 按代码 | 代码直接请求 `/v1/responses` 或 `/v1/chat/completions`，自己维护请求体、输出解析、工具调用和错误处理 |

## 不要强行切换

- 客户端只支持 Chat Completions 时，不要改成 Responses：请求体不匹配，工具调用失效，或客户端解析不了回复。
- Codex 和支持 Responses 的 Agent 不要退回兼容模式：还能聊天，但工具调用、状态管理、推理摘要、上下文透传会不完整。

## 排查

| 现象 | 检查 |
| --- | --- |
| Codex 能启动但能力不完整 | 后台日志是否出现 `/v1/responses`，本地配置是否是 Responses 模式 |
| 模型列表为空 | 手动填后台分组里的可用模型名，新开对话发送 `测试` |
| 长上下文没生效 | 客户端是否传出了上下文相关配置，是否在本地被截断 |
| 工具调用、联网、MCP 不生效 | 客户端是否支持这些能力，是否用的是 Responses 模式 |
