---
sidebarTitle: Base URL 怎么填
---

# Base URL 怎么填

手动配置客户端时，按下表判断地址怎么填。后台、教程或 CC Switch 已经给出配置的，直接复制，不要自己补 `/v1`，也不要把 `/chat/completions`、`/responses` 拼进 Base URL。

## 按接入类型填写

| 接入类型 | 协议 | 地址写法 |
| --- | --- | --- |
| 后台生成配置 / CC Switch 导入 | 按生成内容 | 原样复制，不改地址、不补 `/v1`；只用 MyAutoAPI 后台或本站教程给的配置，不套用其它站点的模板 |
| Codex 等支持 Responses 的客户端 | Responses | 按后台生成配置或对应教程，不要降级成普通兼容模式 |
| 普通 OpenAI-compatible 客户端（Cherry Studio、Open WebUI、LibreChat 等） | Chat Completions | `https://api.myautoapi.com/v1` |
| 客户端字段明确要求完整接口地址 | 按客户端说明 | 填完整路径，例如 `https://api.myautoapi.com/v1/chat/completions` |

普通兼容客户端会自己在 Base URL 后拼 `/chat/completions`、`/models` 等路径，所以 Base URL 只填到 `/v1`。两种协议的区别看 [Responses 与兼容模式](/external/compatibility)。

## 填完测试

API Key 从 MyAutoAPI 后台复制，模型名填当前 Key 分组里的一个可用模型。在客户端新开对话，发送 `测试`，使用记录页面出现这次调用，就说明请求已经到达 MyAutoAPI。

## 常见错误

| 现象 | 原因和处理 |
| --- | --- |
| 普通客户端报路径错误 / 404 | Base URL 写成了 `.../v1/chat/completions` 或自己猜的 `.../v1/responses`，改回 `https://api.myautoapi.com/v1` |
| 复制后台配置后又出错 | 复制后又手动补了 `/v1`，变成重复路径；改回生成内容 |
| Codex 能聊天但工具调用、上下文不正常 | 检查是否误改成了普通兼容模式，换回后台生成的 Responses 配置；协议区别看 [Responses 与兼容模式](/external/compatibility) |
| CC Switch 里只有其它站点的模板 | 不同站点的协议、地址、模型名都不同；从 MyAutoAPI 后台导入，或按本站教程手动填 |
| 客户端报错但使用记录没有记录 | 检查客户端是否在用刚保存的配置、CC Switch 是否已启用到目标客户端、Base URL 是否填在 API 地址栏而不是其它字段、终端环境变量 / 旧配置文件 / 客户端缓存是否覆盖了新配置 |
| `401 Unauthorized` | 重新复制 MyAutoAPI API Key，确认没过期；检查旧 OAuth 登录状态或旧环境变量是否覆盖了新配置 |

更多排查见 [报错与踩坑](/errors/)。
