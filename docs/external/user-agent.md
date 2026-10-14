# User-Agent

`User-Agent` 是客户端发请求时带上的请求头，用来说明“这次请求来自什么客户端”。多数客户端会自动带上自己的 `User-Agent`，用户不需要手动填写。

本页用于判断外接调用时是否需要处理 `User-Agent`，以及遇到 `403 block` / `403 Forbidden` 时把它放在排查链路的哪一步。

| 问题 | 结论 |
| --- | --- |
| 正常能用要不要改？ | 不要改。能正常聊天、写代码、调用模型时，保持客户端默认值。 |
| 403 时先查 UA 吗？ | 不是。先查 Key、Base URL、模型、接口路径，再查 `User-Agent`。 |
| 能不能随便复制别人的 UA？ | 不要复制。确实需要自定义 Header 时，填写稳定、可识别、和当前客户端匹配的标识。 |

::: warning 不要为了“更像某个客户端”而改 UA
`User-Agent` 不是鉴权信息，也不能替代正确的 API Key、模型权限和接口路径。随意伪装成不相关客户端，反而会让排查更困难。
:::

## 速查表

| 接入类型 | 常见工具 | `User-Agent` 处理口径 | 常见优先检查项 |
| --- | --- | --- | --- |
| Codex / Responses | Codex CLI、ChatGPT 桌面应用中的 Codex、支持 Responses API 的代码 Agent | 默认使用 Codex 自带 UA；不要改成普通聊天客户端 UA | 是否使用 Responses，使用记录页面是否出现 `/v1/responses` |
| 普通 OpenAI-compatible | Cherry Studio、Open WebUI、LibreChat、普通聊天客户端 | 能用就不改；需要时填写客户端名和版本 | Key、`https://api.myautoapi.com/v1`、模型名 |
| OpenClaw / Hermes / Agent | OpenClaw、Hermes、其他 Agent CLI | 优先保留工具默认 UA；需要时填写工具真实标识 | API mode、模型名、代理是否转发 Header |
| 自建代理 / 脚本 | Python、Node.js、curl、内部网关、反向代理 | 明确写自己的脚本或服务名，便于日志定位 | 代理最终请求是否覆盖 Authorization 和 UA |
| 浏览器型客户端 | Web 面板、浏览器扩展、自建前端 | 使用浏览器真实 UA；前端未必能手动设置 | 浏览器限制、后端代理、CORS 和最终 Header |

## 按接入类型处理

### Codex / Responses

适用于 Codex CLI、ChatGPT 桌面应用中的 Codex，以及明确使用 OpenAI Responses API 的客户端。

优先按这个顺序看：

1. 确认当前客户端确实支持 Responses。
2. 确认配置没有被改成普通 Chat Completions 兼容模式。
3. 确认使用记录页面里的接口路径是 `/v1/responses`。
4. 前面都正确仍然 `403 block` 时，再看最终请求里是否带有 Codex 自带的 `User-Agent`。

示例格式：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "codex_cli_rs/0.77.0"
}
```

版本号只是格式示例，不要求照抄。实际值以你本机正在使用的 Codex 版本为准。

### 普通 OpenAI-compatible

适用于 Cherry Studio、Open WebUI、LibreChat 这类手动填写 `Base URL`、`API Key`、`Model` 的普通聊天客户端。

普通客户端优先排查连接参数，不要一上来改 `User-Agent`：

1. API Key 是否从 MyAutoAPI 后台复制完整。
2. Base URL 是否填 `https://api.myautoapi.com/v1`。
3. 模型名是否和后台当前可用模型一致。
4. 客户端是否按普通 Chat Completions 路径请求。
5. 以上都正确但仍然 403，再看是否需要补 `User-Agent`。

需要填写时，写成能识别来源的稳定格式：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "CherryStudio/1.0 OpenAI-compatible"
}
```

不要把 Codex、Agent 或浏览器的 UA 复制到普通聊天客户端里。

### OpenClaw / Hermes / Agent

适用于 OpenClaw、Hermes，以及其他带工具调用、文件操作或长任务能力的 Agent / CLI。

这类工具最容易混淆的是协议和代理层，而不是 UA 本身。先确认：

| 检查项 | 说明 |
| --- | --- |
| API mode | 该使用 Responses 就使用 Responses；普通兼容端点才使用 Chat Completions |
| Base URL | 按当前客户端教程填写，不要自己拼完整接口路径 |
| 模型名 | 必须和 MyAutoAPI 后台当前可用模型一致 |
| Header 转发 | 经过网关或代理时，确认最终请求没有丢掉 Header |

只有在使用记录页面或客服排查提示外接识别异常，或 `403 block` 排查到 Header 层时，再手动补：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "OpenClaw/0.1 external-cli"
}
```

Hermes 也同理，写 Hermes 自己的稳定标识，不要伪装成 Codex 或普通浏览器。

### 自建代理 / 脚本

适用于自己写的 Python、Node.js、curl 脚本，或者公司内部网关、反向代理、二次封装服务。

这类场景要主动检查 `User-Agent`，因为很多 HTTP 客户端默认值很短，或者代理会覆盖上游 Header。

示例：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "my-proxy/0.1 openai-compatible"
}
```

排查重点不是“UA 像不像某个官方客户端”，而是日志里能不能稳定看出请求来自哪个脚本、哪个代理、哪个版本。

### 浏览器型客户端

适用于 Web 面板、浏览器扩展、自建前端，或其它从浏览器环境发起请求的客户端。

浏览器会自动带自己的 `User-Agent`。前端 JavaScript 一般不能随意修改浏览器的真实 UA；看到客户端里有 Header 设置时，也要确认最终请求是不是经过后端代理转发。

浏览器型 UA 示例只用于说明格式：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36"
}
```

浏览器扩展或自建前端优先看使用记录页面和代理日志，不要只看浏览器控制台里的本地配置。

## 403 时的排查顺序

外接请求返回 `403 block`、`403 Forbidden` 或类似拦截提示时，按这个顺序查：

| 顺序 | 检查项 | 怎么判断 |
| --- | --- | --- |
| 1 | Key | API Key 是否复制完整、没有过期、来自 MyAutoAPI 后台，并且请求使用 `Bearer sk-xxx` |
| 2 | Base URL | 普通 OpenAI-compatible 手动配置时填 `https://api.myautoapi.com/v1`；Responses 客户端按当前教程或生成配置填写 |
| 3 | 模型 | 模型名是否和后台当前可用模型逐字一致，Key 所属权限是否支持这个模型 |
| 4 | 接口路径 | Codex / Responses 看实际请求是否到 `/v1/responses`；普通聊天客户端看是否到 `/v1/chat/completions` |
| 5 | `User-Agent` 是否存在 | 使用记录页面或代理日志里能否看到最终发出的 UA |
| 6 | `User-Agent` 是否匹配 | Codex 用 Codex 自带标识；普通客户端用自己的客户端标识；代理脚本用自己的服务标识 |
| 7 | 代理是否改写 Header | 自建代理、网关、CLI wrapper 可能覆盖 `Authorization` 或 `User-Agent` |

::: tip 先看最终请求
使用代理、网关或自建后端时，客户端里填了 `User-Agent` 不代表 MyAutoAPI 最终收到了它。排查时优先看 MyAutoAPI 使用记录页面，或者看代理发出请求时的最终 Header。
:::

## 给客服的信息

排查后仍不能解决时，把这些信息一起发给客服：

| 信息 | 示例 |
| --- | --- |
| 客户端名称和版本 | Codex CLI、Cherry Studio、OpenClaw、Hermes |
| 使用的模型名 | 后台当前可用模型里的完整模型 ID |
| Base URL | 你在客户端里填写的地址 |
| 错误信息 | `403 block`、`403 Forbidden` 或完整报错截图 |
| 使用记录页面 | 接口路径、状态码、是否有调用记录 |
| 手动设置的 `User-Agent` | 没有手动设置时，说明“使用客户端默认 UA” |

## 相关页面

| 页面 | 什么时候看 |
| --- | --- |
| [外接兼容与 Base URL 说明](./base-url) | 不确定 Base URL、协议或 `/v1` 应该怎么填 |
| [Responses 与兼容模式说明](./compatibility) | 不确定应该使用 Responses 还是普通 OpenAI-compatible |
| [报错与踩坑](/errors/) | 已经确认请求到达 MyAutoAPI，但仍然返回 403 |
