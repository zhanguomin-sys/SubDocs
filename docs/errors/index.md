# 报错与踩坑

遇到报错时，先判断请求有没有到达 MyAutoAPI 后台。

| 后台情况 | 先查什么 |
| --- | --- |
| 没有调用记录 | Base URL、客户端是否保存配置、本地代理、环境变量、旧配置覆盖 |
| 有调用记录 | API Key、模型名、分组权限、额度、后台返回的具体错误 |

不要只看状态码。把客户端完整报错、后台记录里的接口路径和错误信息一起对照，通常能更快定位。

## `401 Unauthorized` / `Incorrect API key provided` / `Invalid API Key`

- 用户看到：`401 Unauthorized`、`Incorrect API key provided`、`Invalid API Key`、`Invalid API Key format`。
- 通常表示：API Key 无效、没被客户端读取，或被旧配置覆盖。
- 先查什么：Key 是否复制完整，前后有没有空格、换行、引号或不可见字符；客户端是否真的读取了最新 Key。
- 怎么处理：重新从后台复制 Key，填到 `API Key` / `Bearer Token` 字段；重启客户端或终端；Key 已删除、禁用或过期时，新建一个 Key。

## `403 Forbidden` / `403 block` / `Permission denied` / `model not allowed`

- 用户看到：`403 Forbidden`、`403 block`、`Permission denied`、`model not allowed`、`not allowed to use this model`。
- 通常表示：请求到了后台，但当前 Key、分组或模型不允许这次调用。
- 先查什么：Key 所属分组是否支持当前模型，模型名是否和后台一致，账号或分组是否被停用。
- 怎么处理：换成该分组可用的模型；把 Key 切到正确分组；分组停用时联系后台处理。

## `403` / `INSUFFICIENT_BALANCE` / `Insufficient account balance`

- 用户看到：`INSUFFICIENT_BALANCE`、`Insufficient account balance`。
- 通常表示：MyAutoAPI 账户余额不足，当前请求被后台拒绝。
- 先查什么：登录 MyAutoAPI 后台查看账户余额。
- 怎么处理：充值后重新测试。充值后仍然出现这条提示，再确认当前客户端使用的是最新 API Key。

## `404 Not Found` / `Cannot POST /v1/chat/completions`

- 用户看到：`404 Not Found`、`Cannot POST /v1/chat/completions`、`Cannot POST /v1/responses`、`Not Found`。
- 通常表示：请求路径不对，或客户端把 Base URL 和完整接口路径拼错了。
- 先查什么：Base URL 是否填成了完整接口地址；客户端最终请求的是 `/v1/chat/completions` 还是 `/v1/responses`。
- 怎么处理：普通 OpenAI-compatible 客户端通常填 `https://api.myautoapi.com/v1`；Codex 等 Responses 接入按后台生成配置或对应教程填写，不要手动追加 `/chat/completions` 或 `/responses`。

## `405 Method Not Allowed`

- 用户看到：`405 Method Not Allowed`、`Method Not Allowed`。
- 通常表示：请求方法不对，常见于把 API 地址放进浏览器打开，或客户端用错接口类型。
- 先查什么：客户端是否在用 POST 请求；Base URL 是否填到了客户端要求的位置。
- 怎么处理：不要用浏览器直接测试聊天接口；回到客户端配置页，只填写 Base URL、API Key 和模型名，让客户端自己拼接请求路径。

## `429 Too Many Requests` / `rate limit` / `rate_limit_exceeded`

- 用户看到：`429 Too Many Requests`、`rate limit`、`Rate Limit Exceeded`、`rate_limit_exceeded`。
- 通常表示：触发了并发、频率或 API Key 限制。
- 先查什么：API Key 是否设置了限额金额、有效期、速率或次数限制；是否有多个客户端同时请求。
- 怎么处理：降低并发和重试频率，等待限制窗口恢复；确认没有人为限制后仍持续出现，再联系后台处理。

## `500 Internal Server Error`

- 用户看到：`500 Internal Server Error`、`API Error 500`。
- 通常表示：上游或中转侧临时错误，也可能是请求体触发了服务端异常。
- 先查什么：后台记录里的具体错误、同一模型是否连续失败、换一个短输入是否正常。
- 怎么处理：先重试一次；仍失败时换模型或稍后再试；如果只有某段输入失败，缩短上下文后再发。

## `502 Bad Gateway` / `upstream_error` / `Your input exceeds the context window of this model`

用户可能看到：

```json
{
  "error": {
    "message": "Your input exceeds the context window of this model. Please adjust your input and try again.",
    "type": "upstream_error"
  }
}
```

- 状态码：`502`
- 错误类型：`upstream_error`
- 核心含义：请求已经到达 MyAutoAPI，并且已经转发到上游；上游拒绝了这次请求，原因是本次输入超过了当前模型的上下文窗口。
- 不是额度问题，也不是 Key 失效；不要让用户去反复换 Key 或重复重试同一段超长输入。

中转站后台先看这些信息：

- 这次请求是否有后台调用记录。
- 记录里的模型名是否就是用户当前客户端填写的模型名。
- 上游返回的原始错误里是否包含 `exceeds the context window`、`context length`、`maximum context length`、`too many tokens` 这类信息。
- 用户是否在长会话里连续追问，或者一次性塞入了大文件、长日志、网页全文、PDF 内容、代码仓库片段。

处理方法：

- 让用户新开会话后重试同一个问题，先排除历史上下文堆积导致的超限。
- 删除无关历史、长日志、重复文件内容，只保留当前问题必须用到的片段。
- 客户端支持“压缩上下文”“总结会话”“清理历史消息”时，先压缩再发送。
- 做代码、日志、文档分析时，不要一次性整包提交；按文件、模块或错误片段分批发送。
- 确实需要长上下文时，换成后台模型列表里明确支持更大上下文的模型，并确认当前 Key 所属分组允许使用该模型。
- 换大上下文模型后仍报错时，继续减少输入和预留输出长度；文本生成模型限制的是输入和输出合计长度，不只是用户当前打出来的那一段文字。

简短说法：

> 这次请求已经到达中转站，但上游返回“输入超过当前模型上下文窗口”。请新开会话，删掉不必要的历史和大段文件内容后再发；如果必须处理长文档，请换后台列表里支持更大上下文的模型。

## `499` / `api_error` / `context canceled`

用户可能看到：

```json
{
  "error": {
    "message": "context canceled",
    "type": "api_error"
  }
}
```

- 状态码：`499`
- 错误类型：`api_error`
- 核心含义：请求在服务端完成响应前被取消。常见原因是客户端主动中止、客户端超时、本地网络断开、浏览器或应用关闭连接，或者上游响应太慢导致调用链某一层放弃等待。
- `499` 更接近“客户端/调用方提前断开”，不能直接当成上游模型报错，也不能直接判断为中转站配置错误。

中转站后台先看这些信息：

- 后台是否有这次请求记录；完全没有记录时，优先查 Base URL、代理、客户端配置和本地网络。
- 有记录时，看请求耗时是在很短时间内取消，还是等待很久后取消。
- 同一用户是否只在长输出、长上下文、流式输出时出现；短问题是否正常。
- 同一时间同一模型是否大量出现慢响应、超时或取消。

处理方法：

- 让用户先用一个很短的问题测试同一个 Key、同一个模型；短问题正常，说明基础配置通常没问题。
- 只在长问题或长输出时出现时，减少输入长度，降低输出长度，或关闭客户端里的自动重试/并发请求后再试。
- 客户端有超时设置时，把请求超时时间调长；代理、网关、反代、Docker、浏览器插件也可能有自己的超时。
- 只在流式输出时出现时，先关闭 stream 验证非流式请求是否稳定；非流式稳定时，再检查客户端或代理是否会中断 SSE 流。
- 后台显示上游长时间无响应时，可以临时切换到响应更快的模型，或稍后再试。
- 短问题也稳定出现 `499` 时，把客户端完整报错、后台请求记录、请求时间点、模型名和是否使用代理一起反馈，重点排查客户端到中转站这一段连接是否提前断开。

简短说法：

> 这条 `499 context canceled` 表示请求还没等到完整响应就被取消了。请先用短问题测试；如果短问题正常，就减少上下文和输出长度，检查客户端/代理超时设置，必要时关闭流式输出再试。

## `503 Service Unavailable`

- 用户看到：`503 Service Unavailable`、`Overloaded`、`model_not_found`。
- 通常表示：上游服务过载、模型暂不可用、该渠道下模型已下线，或本地网络/代理连接不稳定。
- 先查什么：后台记录中的模型名和上游错误；模型是否仍在当前分组可用列表里；本机是否开了代理、加速器或公司网络。
- 怎么处理：先换成后台明确可用的模型；开了代理时先关闭代理重试，或切换节点后再试；没开代理时换一个网络再试。遇到 `model_not_found` 时，不要反复重试同一个模型名。

## `504 Gateway Timeout`

- 用户看到：`504 Gateway Timeout`、`gateway timeout`。
- 通常表示：请求已发出，但上游或中转等待响应超时。
- 先查什么：输入是否过长、模型是否响应慢、本地网络是否稳定。
- 怎么处理：缩短上下文或新开会话；降低输出长度；稍后重试或换响应更快的模型。

## `ECONNRESET` / `ETIMEDOUT` / `ENOTFOUND` / `Connection refused`

- 用户看到：`ECONNRESET`、`ETIMEDOUT`、`ENOTFOUND`、`ECONNREFUSED`、`Connection refused`、`API Error (Connection error.)`。
- 通常表示：本地到服务器的网络链路不通，或域名、代理、网关配置有问题。
- 先查什么：后台有没有调用记录；本地代理是否拦截；域名是否写错；公司网络、防火墙或容器网络是否能访问接口域名。
- 怎么处理：确认 Base URL 拼写；关闭或更换代理后重试；Docker / WSL / 远程服务器场景要在实际运行客户端的环境里测试网络。

## `model not found` / `The model does not exist`

- 用户看到：`model not found`、`The model does not exist`、`Invalid model`、`model_not_found`。
- 通常表示：模型名写错，或当前 Key 分组没有这个模型。
- 先查什么：后台模型列表里的准确模型名；Key 所属分组是否包含该模型。
- 怎么处理：复制后台显示的模型名，不要手写猜测；先用一个确定可用的模型跑通，再切换到其它模型。

## `invalid_request_error`

- 用户看到：`invalid_request_error`、`invalid request`、`bad request`。
- 通常表示：请求体不符合当前接口协议，或客户端把 Responses / Chat Completions 模式混用了。
- 先查什么：客户端当前走的是 `/v1/responses` 还是 `/v1/chat/completions`；配置里的协议模式是否和教程一致。
- 怎么处理：Codex 等支持 Responses 的工具优先用 Responses 配置；普通聊天客户端使用 OpenAI-compatible 配置；不要把两个协议的字段混在一起。

## `cyber safety` / `safety policy` / `content blocked`

用户看到 `cyber safety`、`safety policy`、`content blocked`、`request blocked`、`disallowed content` 等提示时，通常表示请求内容触发了上游网络安全策略。

这类问题一般不是 API Key、额度、模型名或 Base URL 配置错误。这是 OpenAI 上游官方风控限制，中转站只能转发请求，不能绕过上游安全策略。

请不要提交以下内容：

- 破解账号、破解软件、绕过授权、绕过付费限制
- 逆向破解、绕过 license、绕过风控
- 漏洞利用、攻击步骤、提权、持久化、横向移动
- 木马、后门、恶意脚本、免杀、绕过检测
- 钓鱼、撞库、爆破、抓取 Cookie、Token、密码
- 扫描、攻击、入侵不属于自己的系统
- 生成可直接用于攻击的 payload、POC、执行命令

排查安全问题时，请改成安全审查、风险解释、日志分析、漏洞修复或防护加固方向。

## `context_length_exceeded`

- 用户看到：`context_length_exceeded`、`maximum context length`、`This model's maximum context length is`、`exceeds the context window`。
- 核心含义：输入、历史消息、工具内容和预期输出合计超过模型上下文上限。不同上游可能返回 `400`、`502 upstream_error` 或其它包装后的状态码，判断时以错误信息和后台上游原文为准。
- 怎么处理：按上面的 `502 upstream_error` 上下文超限处理，新开会话、压缩历史、拆分大文件、降低输出长度；需要长上下文时，换后台明确支持更大上下文且当前 Key 分组允许使用的模型。

## JSON / stream / SSE 解析类错误

- 用户看到：`Unexpected token`、`JSON parse error`、`stream error`、`SSE error`、`Invalid event stream`、`Premature close`。
- 通常表示：客户端预期的响应格式和实际返回不一致，或流式响应被网络中断。
- 先查什么：客户端是否选错协议模式；后台记录是否返回了非 JSON 错误页；本地代理是否改写响应。
- 怎么处理：按客户端教程重新选择 Responses 或普通兼容模式；关闭会改写流量的代理；非流式可用而流式失败时，先关闭 stream 验证基础配置。
