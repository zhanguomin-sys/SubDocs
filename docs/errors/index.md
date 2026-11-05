---
sidebarTitle: 遇到问题
---

# 报错与踩坑

遇到报错，先看后台 **使用记录** 里有没有这次请求，再按下面的报错原文查：

| 使用记录 | 先查什么 |
| --- | --- |
| 没有记录 | Base URL、客户端是否保存了配置、本地代理、环境变量、旧配置覆盖 |
| 有记录 | API Key、模型名、分组权限、额度、记录里的具体错误 |

把客户端的完整报错和使用记录里的接口路径、错误信息对照着看，比只看状态码定位快。

## `401 Unauthorized` / `Incorrect API key provided` / `Invalid API Key`

也会显示为 `Invalid API Key format`。API Key 无效、没被客户端读取，或被旧配置覆盖。

- 检查 Key 是否复制完整，前后有没有空格、换行、引号或不可见字符。
- 重新从后台复制 Key，填到 `API Key` / `Bearer Token` 字段，重启客户端或终端。
- Key 已删除、禁用或过期时，新建一个 Key。

## `403 Forbidden` / `403 block` / `Permission denied` / `model not allowed`

也会显示为 `not allowed to use this model`。请求到了后台，但当前 Key、分组或模型不允许这次调用。

- 换成 Key 所属分组支持的模型，模型名和后台逐字一致。
- 把 Key 切到正确分组；账号或分组被停用时联系客服。
- 在 Claude Code、Claude Desktop 以外的客户端用 Claude 模型，Key 要选 **CC MAX 外接分组**。
- 以上都对仍是 `403 block`，按 [User-Agent](/external/user-agent#_403-时的排查顺序) 的顺序继续查。

## `403` / `INSUFFICIENT_BALANCE` / `Insufficient account balance`

账户余额不足。登录后台查看余额，充值后重新测试；充值后仍提示，确认客户端用的是最新 API Key。

## `404 Not Found` / `Cannot POST /v1/chat/completions`

也会显示为 `Cannot POST /v1/responses`、`Not Found`。请求路径不对，通常是 Base URL 和接口路径拼错了。

- 普通 OpenAI-compatible 客户端填 `https://api.myautoapi.com/v1`。
- Codex 等 Responses 接入按后台生成配置或对应教程填写。
- 不要手动追加 `/chat/completions` 或 `/responses`。详见 [Base URL 怎么填](/external/base-url)。

## `405 Method Not Allowed`

请求方法不对，常见于把 API 地址放进浏览器打开，或客户端用错接口类型。不要用浏览器直接测试聊天接口；回到客户端配置页，只填 Base URL、API Key 和模型名，让客户端自己拼路径、用 POST 请求。

## `429 Too Many Requests` / `rate limit` / `rate_limit_exceeded`

触发了并发、频率或 API Key 限制。

- 检查 API Key 是否设置了限额金额、有效期、速率或次数限制，是否有多个客户端同时在用这个 Key。
- 降低并发和重试频率，等限制窗口恢复。
- 没有人为限制仍持续出现，联系客服。

## `500 Internal Server Error`

也会显示为 `API Error 500`。上游或中转临时错误，也可能是请求体触发了服务端异常。先重试一次；仍失败就换模型或稍后再试。只有某段输入失败时，缩短上下文后再发。

<a id="context-window"></a>

## `502 Bad Gateway` / `upstream_error` / `context_length_exceeded` / 上下文超限

常见原文：

```json
{
  "error": {
    "message": "Your input exceeds the context window of this model. Please adjust your input and try again.",
    "type": "upstream_error"
  }
}
```

也会显示为 `context_length_exceeded`、`maximum context length`、`This model's maximum context length is`、`context length`、`too many tokens`；不同上游可能包装成 `400`、`502` 或其它状态码，以错误原文为准。

请求已经到达 MyAutoAPI 并转给了上游，上游拒绝的原因是：输入、历史消息、工具内容和预留输出合计超过了模型上下文上限。这不是额度或 Key 问题，换 Key、重复发送同一段超长输入都没用。

1. 核对使用记录里的模型名和客户端填的模型名一致。
2. 新开会话后重发同一个问题，排除历史上下文堆积。长会话连续追问、一次塞进大文件 / 长日志 / 网页全文 / PDF / 代码仓库片段，最容易超限。
3. 删掉无关历史、长日志和重复文件内容，只留当前问题必须的片段；客户端有“压缩上下文”“总结会话”“清理历史消息”时先压缩。
4. 分析代码、日志、文档时，按文件、模块或错误片段分批发送，不要整包提交。
5. 确实需要长上下文时，换后台模型列表里支持更大上下文、且当前 Key 分组能用的模型。
6. 换了大上下文模型仍报错，继续减少输入，并调低输出长度。

## `499` / `api_error` / `context canceled`

常见原文：

```json
{
  "error": {
    "message": "context canceled",
    "type": "api_error"
  }
}
```

请求在完整响应前被取消：客户端主动中止、客户端超时、本地网络断开、应用或浏览器关闭连接，或上游太慢导致调用链某一层放弃等待。它更接近“调用方提前断开”，不能直接当成上游模型报错或中转站配置错误。

1. 在使用记录里看这次请求是很快就被取消，还是等了很久才取消；等了很久才取消的，重点做第 4 步。
2. 用很短的问题测试同一个 Key 和模型。短问题正常，基础配置就没问题。
3. 只在长问题、长输出时出现：减少输入、调低输出长度，关闭客户端的自动重试和并发请求。
4. 调长客户端的请求超时；代理、网关、反代、Docker、浏览器插件也有各自的超时。
5. 只在流式输出时出现：先关闭 stream 测非流式；非流式稳定时，检查客户端或代理是否中断了 SSE 流。
6. 同一模型普遍响应慢时，换响应更快的模型或稍后再试。
7. 短问题也稳定 `499`：把完整报错、这条使用记录、请求时间点、模型名、是否用代理发给 [客服](/contact)，重点查客户端到中转站这段连接。

## `503 Service Unavailable`

也会显示为 `Overloaded`。上游过载、模型暂不可用或已下线，或本地网络 / 代理不稳定。

- 换成后台明确可用的模型。
- 开了代理、加速器时，关掉代理重试或换节点；没开代理就换一个网络试。

## `504 Gateway Timeout`

请求已发出，但上游或中转等待响应超时。缩短上下文或新开会话，调低输出长度；稍后重试或换响应更快的模型。

## `ECONNRESET` / `ETIMEDOUT` / `ENOTFOUND` / `Connection refused`

也会显示为 `ECONNREFUSED`、`API Error (Connection error.)`。本机到服务器的网络不通，或域名、代理、网关配置有问题，这时使用记录里通常没有记录。

- 核对 Base URL 拼写。
- 关闭或更换代理后重试；检查公司网络、防火墙是否放行接口域名。
- Docker / WSL / 远程服务器上运行的客户端，要在它实际运行的环境里测网络。

## `model not found` / `The model does not exist`

也会显示为 `Invalid model`、`model_not_found`。模型名写错，或当前 Key 分组没有这个模型。从 [模型价格](/models) 或后台复制准确的模型名，不要手写猜测；先用一个确定可用的模型跑通，再换别的。遇到 `model_not_found` 不要反复重试同一个模型名。

## `invalid_request_error`

也会显示为 `invalid request`、`bad request`。请求体不符合当前协议，常见于 Responses 和 Chat Completions 混用。Codex 等支持 Responses 的工具用 Responses 配置，普通聊天客户端用 OpenAI-compatible 配置，两种协议的字段不要混在一起。详见 [Responses 与兼容模式](/external/compatibility)。

## `cyber safety` / `safety policy` / `content blocked`

也会显示为 `request blocked`、`disallowed content`。请求内容触发了 OpenAI 上游的网络安全风控，不是 Key、额度、模型名或 Base URL 的问题；中转站只转发请求，绕不过上游策略。

以下内容会被拦截，不要提交：

- 破解账号、破解软件、绕过授权、绕过付费限制
- 逆向破解、绕过 license、绕过风控
- 漏洞利用、攻击步骤、提权、持久化、横向移动
- 木马、后门、恶意脚本、免杀、绕过检测
- 钓鱼、撞库、爆破、抓取 Cookie、Token、密码
- 扫描、攻击、入侵不属于自己的系统
- 生成可直接用于攻击的 payload、POC、执行命令

做安全相关的工作时，改成安全审查、风险解释、日志分析、漏洞修复或防护加固方向来提问。

## JSON / stream / SSE 解析错误

也会显示为 `Unexpected token`、`JSON parse error`、`stream error`、`SSE error`、`Invalid event stream`、`Premature close`。客户端预期的响应格式和实际返回不一致，或流式响应被网络中断。

- 先看使用记录是否返回了非 JSON 的错误页，按其中的原始错误排查。
- 按客户端教程重新选择 Responses 或普通兼容模式。
- 关闭会改写流量的代理。
- 非流式可用、流式失败时，先关闭 stream 验证基础配置。
