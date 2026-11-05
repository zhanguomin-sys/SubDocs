---
sidebarTitle: 手动配置
---

# Codex 手动接入

不用接入助手或命令时，把后台生成的两段配置手动写进本机 `.codex` 文件夹里的 `config.toml` 和 `auth.json`。需要先按 [快速开始](/quick-start) 创建 API Key。

## 1. 打开 `.codex` 文件夹

| 系统 | 位置 |
| --- | --- |
| Windows | `C:\Users\你的用户名\.codex`，在文件管理器地址栏输入 `%USERPROFILE%\.codex` 可直接打开 |
| Mac / Linux / WSL | `~/.codex` |

文件夹里没有 `config.toml` 或 `auth.json` 时，新建同名文件。Windows 新建后确认文件名没有变成 `.txt` 结尾。不要删除文件夹里的其它文件。

<details>
<summary>Windows 找不到文件夹？看截图</summary>

依次打开 **此电脑 → 本地磁盘 (C:) → 用户 → 你的用户名 → `.codex`**。用户名文件夹不是 **公用** 或 `Administrator`，图中的 `ss` 只是示例。

<a class="doc-image-link" href="/images/连接/c盘1.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/c盘1.jpg" alt="在 Windows 的 C 盘打开用户文件夹">
</a>

<a class="doc-image-link" href="/images/连接/c盘2.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/c盘2.jpg" alt="在用户文件夹中打开自己的 Windows 用户名">
</a>

<a class="doc-image-link" href="/images/连接/c盘3.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/c盘3.jpg" alt="在 Windows 用户目录中打开 .codex 文件夹">
</a>

<a class="doc-image-link" href="/images/连接/c盘4.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/c盘4.jpg" alt="在 .codex 文件夹中找到 auth.json 和 config.toml">
</a>

</details>

## 2. 粘贴后台配置

在管理后台 **API 密钥** 页面，点击 Key 右侧的 **使用密钥**，选择 **Codex CLI** 和本机系统。后台显示两段配置内容：

1. 打开 `config.toml`，先复制一份备份（原来配过 MCP 的，MCP 配置在备份里），再删除原有全部内容，粘贴后台的 `config.toml` 内容，保存。
2. 打开 `auth.json`，删除原有全部内容，粘贴后台的 `auth.json` 内容，保存。

两段不能贴反，也不能合并到一个文件。

<details>
<summary>查看后台截图</summary>

<a class="doc-image-link" href="/images/quick-start/create-api-key-3.jpg" target="_blank" rel="noopener">
  <img src="/images/quick-start/create-api-key-3.jpg" alt="在 API 密钥列表中点击使用密钥">
</a>

<a class="doc-image-link" href="/images/quick-start/create-api-key-4.jpg" target="_blank" rel="noopener">
  <img src="/images/quick-start/create-api-key-4.jpg" alt="复制 config.toml 和 auth.json 内容并分别保存">
</a>

</details>

## 3. 重新打开 Codex 测试

ChatGPT 桌面应用：在右下角托盘的 ChatGPT 图标上点击 **Exit** 彻底退出，再重新打开并切换到 Codex。仍然没读到新配置时，在任务管理器结束所有 ChatGPT 进程，或重启电脑。

Codex CLI：关闭原来的 PowerShell 或终端，重新打开后运行 `codex`。

新建任务，发送 `测试`。能正常回复就接入完成。

<details>
<summary>查看托盘退出截图</summary>

<a class="doc-image-link" href="/images/连接/c盘5.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/c盘5.jpg" alt="从 Windows 右下角托盘退出 ChatGPT 桌面应用">
</a>

</details>

## 常见问题

| 遇到的问题 | 处理方法 |
| --- | --- |
| `401 Unauthorized` | 重新复制后台的 `auth.json` 内容；Key 完整仍报 401 时，按下面 [Key 写在哪个文件](#codex-auth-modes) 检查 |
| `403 Forbidden` | Key 换成支持配置中模型的分组 |
| 修改后不生效 | 确认文件名不是 `.txt` 结尾，彻底退出 Codex 后重新打开 |
| 后台没有使用记录 | 确认两个文件在 Codex 实际读取的 `.codex` 目录；WSL 用 WSL 里的 `~/.codex` |

仍然无法使用时，改用 [快速开始](/quick-start) 的接入助手或命令行。

<a id="codex-auth-modes"></a>

## 进阶：Key 写在哪个文件

`config.toml` 里 `[model_providers.OpenAI]` 下的 `requires_openai_auth` 决定 Codex 发请求时带哪个 Key：

| `requires_openai_auth` | Codex 带哪个 Key |
| --- | --- |
| `true` | `auth.json` 里的 API Key |
| `false` | `config.toml` 里 `experimental_bearer_token` 写的 API Key，不读取 `auth.json` |

2026 年 8 月下旬起的 Codex 版本，`requires_openai_auth = false` 时不再读取 `auth.json`。旧配置把 Key 放在 `auth.json`、同时写 `requires_openai_auth = false`，请求不带 Key，返回 `401 Unauthorized`。

两种可用写法，二选一：

| | 写法一 | 写法二 |
| --- | --- | --- |
| 使用场景 | 一键脚本写入的默认配置 | 需要 Codex 内置生图工具 |
| API Key 位置 | `auth.json` 的 `OPENAI_API_KEY` | `config.toml` 的 `experimental_bearer_token` |
| `requires_openai_auth` | `true` | `false` |
| 生图请求头 `http_headers` | 不写 | 写 |
| Codex 内置生图工具 | 不提供 | 提供 |
| `auth.json` | 写入 API Key，替换原来的 ChatGPT 登录信息 | 不使用 |

写法二在 `[model_providers.OpenAI]` 下写这三行：

```toml
requires_openai_auth = false
experimental_bearer_token = "这里完整填写你的中转站 API Key"
http_headers = { "x-openai-actor-authorization" = "local-image-extension" }
```

`http_headers` 是 Codex 内置生图工具的开关，只在 `requires_openai_auth = false` 时生效，不负责验证 API Key。

写法二的 Key 保存在 `config.toml` 里，把这个文件发给别人或截图前，先删掉 `experimental_bearer_token` 这一行。
