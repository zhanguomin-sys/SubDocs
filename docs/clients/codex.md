# Codex 接入

Codex 是 OpenAI 的编程助手，适合阅读代码、修改文件、解释报错和整理项目。接入后，Codex 的模型请求会通过 MyAutoAPI 发出。

## 一键配置 MyAutoAPI

已经安装 Codex App 时，直接按 [快速开始](/quick-start) 运行一键脚本。

配置完成后，彻底退出 Codex，再重新打开并新建任务，发送：

```text
测试
```

能正常回复，就说明 Codex 已经接入 MyAutoAPI。

<details>
<summary>不想运行脚本</summary>

进入 [Codex CC Switch 接入](/codex-cc-switch)，用图形工具导入配置。

</details>

<details>
<summary>还没有安装 Codex</summary>

### Windows

打开 OpenAI 官方 ChatGPT 桌面应用下载页：

```text
https://chatgpt.com/download/
```

选择 Windows，并在 Microsoft Store 完成安装。安装后打开 ChatGPT，在左上角切换到 **ChatGPT Codex**。

只在需要终端命令时，再安装 Codex CLI：

```powershell
winget install OpenJS.NodeJS.LTS
npm install -g @openai/codex
codex --version
```

### macOS

```bash
npm install -g @openai/codex
codex --version
```

### Linux / WSL

```bash
sudo apt update
sudo apt install -y nodejs npm
npm install -g @openai/codex
codex --version
```

在 WSL 里运行 `codex` 时，配置放进 WSL 的 `~/.codex`，不是 Windows 用户目录。

</details>

<details>
<summary>只想手动配置</summary>

进入 [Codex 手动接入](/clients/codex-manual-config)，把 MyAutoAPI 管理后台生成的 `config.toml` 和 `auth.json` 分别写入本机 `.codex` 文件夹。

</details>

<details>
<summary>关于 /v1</summary>

Codex 优先复制 MyAutoAPI 管理后台生成的配置，不要自己额外加 `/v1`。

只有普通 OpenAI-compatible 客户端手动填写 `Base URL` 时，才填写：

```bash
https://api.myautoapi.com/v1
```

</details>

<details>
<summary>接入常见问题</summary>

| 现象 | 处理方式 |
| --- | --- |
| `codex` 命令不存在 | 重新打开终端；确认 `npm install -g @openai/codex` 执行成功。 |
| 后台没有调用记录 | 检查 `config.toml` 和 `auth.json` 是否放在当前运行环境的 `.codex` 目录，文件名不要带 `.txt` 后缀。 |
| Windows 配了但 WSL 不生效 | WSL 读取的是 WSL 里的 `~/.codex`，不是 Windows 的 `%USERPROFILE%\.codex`。 |
| 401 / Unauthorized | 重新复制 `auth.json`，检查 API Key 是否完整、前后有没有多余空格。 |
| 403 / model not found | 检查 API Key 所属分组是否支持 `config.toml` 里的模型。 |

更多复杂报错看 [报错与踩坑](/errors/)。

</details>

<details>
<summary>其它 Codex 用法</summary>

| 需要 | 入口 |
| --- | --- |
| 保留 ChatGPT 登录或手机连接 | [保留 ChatGPT 登录](/clients/chatgpt-login-myautoapi) |
| 使用 Codex 内置生图 | [Codex 内置生图](/images/codex-image-direct) |
| 学习 profile、会话和权限 | [Codex 进阶](/external/codex-advanced) |

</details>
