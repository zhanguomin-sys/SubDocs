# 选择 Responses 还是兼容模式

MyAutoAPI 同时支持 **Responses 模式** 和常见 **OpenAI-compatible 兼容模式**。协议会影响接口路径、请求体结构、工具调用和客户端解析方式，选择时先看客户端支持什么协议，不要只看模型名。

::: tip 最短结论
Codex、支持 Responses API 的 Agent、新项目和偏工具调用的场景，优先使用 **Responses 模式**。

Cherry Studio、Open WebUI、LibreChat 这类普通 OpenAI-compatible 客户端，按 **兼容模式** 配置。
:::

## 两种模式有什么区别

| 对比项 | Responses 模式 | 兼容模式 |
| --- | --- | --- |
| 实际接口路径 | `/v1/responses` | `/v1/chat/completions` |
| 请求体概念 | 以 `input`、`instructions`、`output`、`previous_response_id` 等结构组织请求和结果 | 以 `messages` 对话列表提交请求，从 `choices[].message` 读取回复 |
| 适合客户端 | Codex、支持 Responses API 的 Agent、需要工具调用或更完整状态管理的客户端 | Cherry Studio、Open WebUI、LibreChat 等普通 OpenAI-compatible 聊天客户端 |
| 工具调用 / Agent 能力 | 更适合承载工具调用、MCP、Web search、File search、Computer use、推理摘要等新能力 | 适合基础聊天和常见函数调用；复杂工具能力取决于客户端是否做了适配 |
| 模型与上下文影响 | 新模型、推理模型和 Agent 场景优先按 Responses 接入；长上下文也要看客户端是否真实传出对应配置 | 普通聊天足够使用；如果客户端本身截断上下文或不传相关字段，只换模型名不会自动生效 |
| 调试时看什么 | 后台日志里看 `/v1/responses`、`output`、`previous_response_id` 等特征 | 后台日志里看 `/v1/chat/completions`、`messages`、`choices` 等特征 |

::: warning 不要把“实际接口路径”当成普通 Base URL 填写
多数客户端只需要填写 `Base URL`，例如：

```text
https://api.myautoapi.com/v1
```

`/v1/responses` 和 `/v1/chat/completions` 是客户端最终请求到 MyAutoAPI 时的实际接口路径，由客户端自动拼接。只有客户端明确要求填写完整接口地址时，才按它的说明填写完整路径。
:::

## 按模型 / 能力类型选择

| 模型 / 能力类型 | 使用模式 | 怎么判断 |
| --- | --- | --- |
| Codex、代码 Agent、支持 Responses 的 Agent | Responses 模式 | 客户端文档或配置里明确支持 Responses API，或配置项里有类似 `wire_api = "responses"` |
| 推理模型、新文本生成项目 | Responses 模式 | 客户端支持 Responses 时优先使用 Responses，推理模型尤其适合使用 Responses |
| 普通聊天、翻译、摘要、日常问答 | 兼容模式也可以 | 客户端只提供 OpenAI-compatible / Chat Completions 配置时，按兼容模式接入 |
| 长上下文分组 | 先看客户端协议 | 长上下文不是只换模型名就生效，还要看客户端是否传出上下文、截断、缓存等相关配置 |
| 工具调用、搜索、MCP、文件检索、Computer use | 优先 Responses 模式 | 这些能力更接近 Responses / Agent 体系；普通聊天客户端可能只支持其中一部分 |

## 按接入类型填写

| 接入类型 | 使用模式 | 常见填写方式 | 说明 |
| --- | --- | --- | --- |
| Codex | Responses 模式 | 优先复制 MyAutoAPI / Codex 教程生成的配置；需要手动时按 Codex 页面配置 `Base URL` 和 Responses 模式 | 后台日志应出现 `/v1/responses` |
| 支持 Responses API 的 Agent / SDK | Responses 模式 | 选择 Responses API 或对应的 Agent 接入方式，`Base URL` 使用 `https://api.myautoapi.com/v1` | 客户端允许选择 API 类型时，不要误选成 Chat Completions |
| Cherry Studio | 兼容模式 | 服务商类型选 `OpenAI`，API 地址填 `https://api.myautoapi.com/v1` | 客户端会自己请求 Chat Completions 路径，不要手动补 `/chat/completions` |
| Open WebUI | 兼容模式 | 按 OpenAI-compatible 方式添加服务，Base URL 填 `https://api.myautoapi.com/v1` | 适合网页聊天面板和多人使用场景 |
| LibreChat | 兼容模式 | 按 OpenAI-compatible endpoint 配置，baseURL 填 `https://api.myautoapi.com/v1` | 适合已有 LibreChat 部署接入 MyAutoAPI |
| 自写脚本 / 自定义前端 | 按代码选择 | 代码里实际请求 `/v1/responses` 或 `/v1/chat/completions` | 需要自己维护请求体、输出解析、工具调用和错误处理 |

## 什么时候不该切换模式

客户端本身只支持 OpenAI-compatible / Chat Completions 时，不要为了“更高级”强行改成 Responses。结果不是能力变强，而是请求体不匹配、工具调用失效或客户端无法解析回复。

使用 Codex 或明确支持 Responses 的 Agent 时，不要退回普通兼容模式。退回后可能还能聊天，但工具调用、状态管理、推理摘要、上下文透传等能力可能不完整。

## 常见判断方法

| 现象 | 优先检查 |
| --- | --- |
| Codex 能启动但能力不完整 | 后台日志是否出现 `/v1/responses`，本地配置是否启用了 Responses 模式 |
| Cherry Studio / Open WebUI / LibreChat 报路径错误 | Base URL 是否误填成了 `https://api.myautoapi.com/v1/chat/completions` |
| 模型列表为空 | 不一定代表聊天不可用，可以手动填写后台分组里实际可用的模型名 |
| 长上下文没有生效 | 客户端是否真的传出了上下文相关配置，是否在本地被截断 |
| 工具调用、联网、MCP 不生效 | 当前客户端是否支持这些能力，以及是否使用了 Responses 模式 |
