# Base URL 与兼容模式

核心结论：**支持 Responses 的客户端，优先按 Responses 接入；只有普通 OpenAI-compatible 客户端需要手动配置时，才使用兼容模式并填写 `https://api.myautoapi.com/v1`。**

后台、客户端教程或 CC Switch 已经生成配置时，直接使用生成内容，不要自己额外补 `/v1`，也不要把 `/chat/completions`、`/responses` 手动拼进 `Base URL`。

配置具体客户端时，再回到对应客户端教程填写字段。

## 按接入类型填写

| 接入类型 | 优先协议 | Base URL / 地址写法 | 说明 |
| --- | --- | --- | --- |
| 后台生成配置 | 按后台生成配置 | 按后台生成配置 | 直接复制生成结果，不要手动改地址或补 `/v1` |
| CC Switch / 配置工具 | 按工具生成内容 | 按工具生成内容 | 只使用明确来自 MyAutoAPI 后台或当前教程的配置；不要套用其它站点的配置模板 |
| 支持 Responses 的客户端 | Responses | 按后台生成配置 | Codex 等支持 Responses 的工具优先使用 Responses，不要降级成普通兼容模式 |
| 普通 OpenAI-compatible 客户端 | OpenAI-compatible / Chat Completions | `https://api.myautoapi.com/v1` | 适合 Cherry Studio、Open WebUI、LibreChat 等只能手动填 OpenAI-compatible 地址的客户端 |
| 客户端要求完整接口地址 | 按客户端说明 | 按客户端说明填写完整路径 | 这不是常规 `Base URL` 场景，只有客户端明确要求完整接口时才这样填 |

::: tip 判断方法
能用后台生成配置，就先用后台生成配置；客户端明确支持 Responses，就优先 Responses；只有客户端只提供普通 OpenAI-compatible 手动配置入口时，才填写 `https://api.myautoapi.com/v1`。
:::

## 通用接入流程

### 第一步：优先获取生成配置

先看 MyAutoAPI 后台或当前客户端教程是否提供生成配置。生成配置通常已经包含正确的协议、地址和必要字段，复制后不要再手动补 `/v1`。

使用 CC Switch 时，只使用明确来自 MyAutoAPI 后台或当前教程的配置。不要照搬其它站点的配置模板，也不要在现成模板基础上随意改地址。

### 第二步：确认客户端是否支持 Responses

客户端支持 Responses，并且当前教程或后台配置也按 Responses 提供时，优先使用 Responses。

Responses 和普通 OpenAI-compatible 不是同一个接入层。支持 Responses 的工具通常会使用 `/v1/responses`，请求体和输出结构也不同；不要只因为看到 OpenAI-compatible 字样，就把它改成普通 `/v1` 兼容模式。

### 第三步：普通兼容客户端再手动填 `/v1`

只有客户端要求你手动填写 OpenAI-compatible `Base URL` 时，才填写：

```text
https://api.myautoapi.com/v1
```

这类客户端会自己在 `Base URL` 后面拼接 `/chat/completions`、`/models` 等路径，所以不要把完整接口路径写进 `Base URL`。

### 第四步：填写 API Key、模型名并测试

`API Key` 从 MyAutoAPI 后台复制。模型名以后台当前分组显示的可用模型为准，不要照抄其它教程里的模型名。

客户端允许手动添加模型时，先填一个后台明确可用的模型。配置完成后，在客户端发送一句简单测试：

```text
你好，请回复一句话。
```

使用记录页面出现调用记录，说明请求已经到达 MyAutoAPI。后续问题再按模型、分组、客户端功能继续排查。

使用记录页面没有记录时，优先检查客户端当前配置是否真的生效、CC Switch 是否已应用到目标客户端、Base URL 是否填在正确位置。

## 常见错误

### 所有场景都填 `https://api.myautoapi.com/v1`

不要这样理解。`https://api.myautoapi.com/v1` 只适合普通 OpenAI-compatible 客户端手动配置。

后台生成配置、CC Switch 等配置工具和 Responses 接入都应按生成内容填写。

### 把完整接口写进 Base URL

普通客户端的 `Base URL` 不要写成：

```text
https://api.myautoapi.com/v1/chat/completions
```

也不要自己猜成：

```text
https://api.myautoapi.com/v1/responses
```

除非客户端明确要求填写完整接口地址，否则 `Base URL` 只填到对应根地址。普通 OpenAI-compatible 手动配置时填到 `/v1` 即可。

### 支持 Responses 却改成普通兼容模式

Codex 这类支持 Responses 的工具，优先使用后台生成的 Responses 配置。改成普通 OpenAI-compatible 兼容模式后，可能导致协议字段、工具调用、上下文状态或输出解析不符合客户端预期。

### 后台生成配置后又手动补 `/v1`

生成配置已经包含当前接入方式需要的地址。复制后再额外补 `/v1`，可能会变成重复路径或错误路径。

### CC Switch 里只有其它站点配置模板

不同站点的协议、Base URL、模型名和请求处理可能不同。接 MyAutoAPI 时不要套用其它站点的配置模板；按目标客户端是否支持 Responses 或普通 OpenAI-compatible 来配置。

### 客户端报错但后台没有记录

这通常说明请求没有真正发到 MyAutoAPI。优先检查：

- 当前客户端是否正在使用你刚保存的配置。
- CC Switch 是否已经切换并应用到目标客户端。
- Base URL 是否填在 API 地址位置，而不是后台页面地址或其它配置项。
- 终端环境变量、旧配置文件或客户端缓存是否覆盖了新配置。

### `401 Unauthorized`

优先检查 API Key 是否复制完整、是否仍有效、是否来自 MyAutoAPI 后台。也要确认客户端没有被旧 OAuth 登录状态或旧环境变量覆盖。

更多排查见 [报错与踩坑](/errors/)。
