---
sidebarTitle: Qwen Code
---

# Qwen Code 接入国产模型

Qwen Code 是阿里千问团队开源的终端编程 Agent，能读项目、改文件、跑命令。它走 OpenAI 兼容接口，所以不只千问，DeepSeek、GLM、Kimi、MiniMax、小米 MiMo、腾讯混元都能用。本文把这些模型写进 `C:\Users\你的用户名\.qwen\settings.json`：每个模型带自己分组的 Key，在 Qwen Code 里用 `/model` 直接切换，不用去后台改分组。

## 1. 安装 Qwen Code

::: code-group

```powershell [Windows]
irm https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen-standalone.ps1 | iex
```

```bash [Mac / Linux]
curl -fsSL https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen-standalone.sh | bash
```

:::

装完关闭并重新打开 PowerShell，运行 `qwen --version`，能看到版本号就装好了。

## 2. 每个分组建一个 Key

一个 Key 只属于一个分组，要用几个分组就建几个 Key（创建方法看 [快速开始](/quick-start)）。

| 分组 | 模型 ID | 识图 |
| --- | --- | --- |
| Qwen 千问分组 | `qwen3.8-max`（复杂编程）、`qwen3.8-flash`（高频辅助） | 支持 |
| DeepSeek 分组 | `deepseek-flash`、`deepseek-v4.1-flash-1017` | 支持 |
| 国产之光 | `glm-5.3`（不识图）、`glm-5.3-flash`（识图） | 看模型 |
| Kimi 分组 | `kimi-k3` | 支持 |
| MiniMax 分组 | `minimax-m3` | 支持 |
| 小米 MiMo 分组 | `mimo-v2.6-pro`、`mimo-v2.6-flash` | 支持 |
| 腾讯混元分组 | `hy4-preview` | 不支持 |

价格表里的其它模型（如 `deepseek-v4-pro`、`glm-5.2`、`LongCat-2.0`）按同样格式添加，`modalities` 那一项先删掉。价格看 [模型价格](/models)。

## 3. 写入配置

打开配置文件（没有会新建）：

::: code-group

```powershell [Windows]
mkdir -Force $env:USERPROFILE\.qwen | Out-Null; notepad $env:USERPROFILE\.qwen\settings.json
```

```bash [Mac / Linux]
mkdir -p ~/.qwen && nano ~/.qwen/settings.json
```

:::

粘贴下面的内容，把三个 Key 换成对应分组的 Key。记事本保存时，保存类型选 **所有文件**，文件名 `settings.json`，编码 **UTF-8**。

文件里原来有内容时先复制一份备份，再把本文的模型合并进 `modelProviders.openai`、Key 合并进 `env`，改 `security.auth.selectedType` 和 `model.name`，其它配置保留。

```json
{
  "modelProviders": {
    "openai": [
      {
        "id": "qwen3.8-max",
        "baseUrl": "https://api.myautoapi.com/v1",
        "envKey": "UGA_QWEN_KEY",
        "generationConfig": { "contextWindowSize": 1000000, "modalities": { "image": true } }
      },
      {
        "id": "deepseek-flash",
        "baseUrl": "https://api.myautoapi.com/v1",
        "envKey": "UGA_DEEPSEEK_KEY",
        "generationConfig": { "contextWindowSize": 1000000, "modalities": { "image": true } }
      },
      {
        "id": "kimi-k3",
        "baseUrl": "https://api.myautoapi.com/v1",
        "envKey": "UGA_KIMI_KEY",
        "generationConfig": { "contextWindowSize": 1000000, "modalities": { "image": true } }
      }
    ]
  },
  "env": {
    "UGA_QWEN_KEY": "千问分组的 Key",
    "UGA_DEEPSEEK_KEY": "DeepSeek 分组的 Key",
    "UGA_KIMI_KEY": "Kimi 分组的 Key"
  },
  "security": { "auth": { "selectedType": "openai" } },
  "model": { "name": "qwen3.8-max" }
}
```

| 字段 | 作用 |
| --- | --- |
| `id` | 发给 MyAutoAPI 的模型 ID，和上表一致 |
| `envKey` | 这个模型用 `env` 里的哪个 Key；同一分组的模型填同一个名字 |
| `contextWindowSize` | 上下文长度。国产模型都在百万级，填 `1000000`，自动压缩按这个长度计算 |
| `modalities.image` | 明确告诉 Qwen Code 这个模型能看图；不识图的模型删掉这一项 |
| `selectedType` | 启动时直接用这份配置，不弹登录菜单 |
| `model.name` | 默认模型，必须是上面某个 `id` |

加别的模型：复制一段 `{ ... }` 改 `id`；换了分组就在 `env` 里加一个新 Key，`envKey` 指向它。Qwen Code 运行中改这个文件会自动生效，重新打开 `/model` 就能看到。

## 4. 测试

进入项目目录运行 `qwen`，新开会话发送 `测试`，能正常回复就接入完成。输入 `/model` 切到 DeepSeek、Kimi 再各发一次 `测试`，确认每个 Key 都填对了。

## 排查

| 现象 | 处理 |
| --- | --- |
| 启动后弹出登录 / 选择服务商菜单 | `security.auth.selectedType` 没写成 `openai`，或 JSON 格式有错（少逗号、多逗号） |
| 某个模型 `401` | 它的 `envKey` 和 `env` 里的名字对不上，或 Key 复制不完整 |
| 某个模型 `403` / `model not found` | `env` 里填的 Key 不是这个模型所在的分组；按第 2 步的表换成对应分组的 Key |
| 发图片后模型说看不到 | 这个模型没加 `"modalities": { "image": true }`，或模型本身不识图（`glm-5.3`、`hy4-preview`） |
| 长对话很快就被压缩 | 这个模型没设 `contextWindowSize`，或设小了 |
| 改了配置但 `/model` 里没出现 | 重新打开 `/model`；仍没有就退出重开 `qwen` |

Key 明文保存在 `settings.json` 里，不要把这个文件发给别人或提交到仓库。
