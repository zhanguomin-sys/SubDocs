---
sidebarTitle: User-Agent
---

# User-Agent

`User-Agent`（UA）是客户端发请求时自动带上的请求头，说明“这次请求来自什么客户端”。多数客户端会自己带，不用手动填。本页讲遇到 `403 block` / `403 Forbidden` 时，什么时候才轮到查 UA、怎么填。

- 能正常用就不改，保持客户端默认值。
- 403 先查 Key、Base URL、模型、接口路径，最后才查 UA（顺序见 [403 排查顺序](#_403-时的排查顺序)）。
- 不要复制别人的 UA，也不要伪装成不相关的客户端。UA 不是鉴权信息，替代不了正确的 Key、模型权限和接口路径，乱改只会让排查更难。确实要填时，写稳定、可识别、和当前客户端匹配的标识。

## 按接入类型处理

| 接入类型 | 常见工具 | UA 怎么处理 | 先查什么 |
| --- | --- | --- | --- |
| Codex / Responses | Codex CLI、ChatGPT 桌面应用中的 Codex、支持 Responses API 的代码 Agent | 用 Codex 自带 UA，不改成普通聊天客户端 UA | 是否是 Responses 模式（没被改成 Chat Completions），使用记录里接口是否是 `/v1/responses` |
| 普通 OpenAI-compatible | Cherry Studio、Open WebUI、LibreChat 等 | 能用就不改；需要时填客户端名和版本 | Key 是否完整、Base URL 是否 `https://api.myautoapi.com/v1`、模型名是否和后台一致、是否走 `/v1/chat/completions` |
| OpenClaw / Hermes / 其它 Agent | OpenClaw、Hermes、其它 Agent CLI | 保留工具默认 UA；需要时填工具自己的真实标识 | API mode（该用 Responses 就用 Responses）、Base URL 按教程填、模型名、网关或代理是否丢了 Header |
| 自建代理 / 脚本 | Python、Node.js、curl、内部网关、反向代理 | 主动写清自己的脚本或服务名和版本，方便在日志里定位；很多 HTTP 库默认 UA 很短，代理也会覆盖上游 Header | 代理最终发出的请求是否覆盖了 `Authorization` 和 UA |
| 浏览器型客户端 | Web 面板、浏览器扩展、自建前端 | 浏览器自动带真实 UA，前端 JavaScript 一般改不了 | 浏览器限制、后端代理、CORS 和最终 Header；看使用记录和代理日志，不只看浏览器控制台里的本地配置 |

OpenClaw、Hermes 只在使用记录或客服提示外接识别异常、或 `403 block` 已经排查到 Header 层时才手动补 UA。

UA 格式示例（版本号只是示例，填你本机实际使用的客户端和版本）：

```text
Codex:          codex_cli_rs/0.77.0
普通客户端:      CherryStudio/1.0 OpenAI-compatible
Agent:          OpenClaw/0.1 external-cli
自建代理/脚本:   my-proxy/0.1 openai-compatible
浏览器:          Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36
```

请求头里和 `Authorization: Bearer sk-xxx` 一起发出：

```json
{
  "Authorization": "Bearer sk-xxx",
  "User-Agent": "my-proxy/0.1 openai-compatible"
}
```

## 403 时的排查顺序

| 顺序 | 检查项 | 怎么判断 |
| --- | --- | --- |
| 1 | Key | API Key 复制完整、没过期、来自 MyAutoAPI 后台，请求头是 `Bearer sk-xxx` |
| 2 | Base URL | 普通兼容客户端填 `https://api.myautoapi.com/v1`；Responses 客户端按教程或生成配置填 |
| 3 | 模型 | 模型名和后台可用模型逐字一致，Key 所属分组支持这个模型 |
| 4 | 接口路径 | Codex / Responses 请求到 `/v1/responses`；普通聊天客户端请求到 `/v1/chat/completions` |
| 5 | UA 是否存在 | 使用记录或代理日志里能看到最终发出的 UA |
| 6 | UA 是否匹配 | Codex 用 Codex 自带标识；普通客户端用自己的标识；代理脚本用自己的服务标识 |
| 7 | 代理是否改写 Header | 自建代理、网关、CLI wrapper 可能覆盖 `Authorization` 或 UA |

经过代理、网关或自建后端时，客户端里填了 UA 不代表 MyAutoAPI 最终收到了它，以使用记录或代理发出的最终 Header 为准。

## 联系客服时提供

| 信息 | 示例 |
| --- | --- |
| 客户端名称和版本 | Codex CLI、Cherry Studio、OpenClaw、Hermes |
| 模型名 | 后台可用模型里的完整模型 ID |
| Base URL | 客户端里填的地址 |
| 错误信息 | `403 block`、`403 Forbidden` 或完整报错截图 |
| 使用记录 | 接口路径、状态码、有没有调用记录 |
| 手动设置的 UA | 没设置就写“使用客户端默认 UA” |

确认请求已到达 MyAutoAPI 仍返回 403 时，看 [报错与踩坑](/errors/)。
