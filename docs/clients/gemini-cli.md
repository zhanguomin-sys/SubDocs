---
sidebarTitle: Gemini CLI
---

# Gemini CLI 接入

Gemini CLI 是 Google 开源的终端编程 Agent，能读项目、改文件、跑命令，用的是 Gemini 原生协议。本文在 `C:\Users\你的用户名\.gemini\.env` 里写三行，让它不登录 Google 账号，直接用 MyAutoAPI 的 Gemini 模型。

## 1. 安装 Gemini CLI

需要 Node.js 20 以上（没装看 [周边工具](/codex/tools)）。在 PowerShell 运行：

```powershell
npm install -g @google/gemini-cli
```

装完运行 `gemini --version`，能看到版本号就装好了。

## 2. 准备 Gemini 分组的 Key

Key 的分组选 **Gemini 分组（反重力 Antigravity 反代）**（创建方法看 [快速开始](/quick-start)）。

Gemini CLI 后台的提示补全、分类等请求固定用 `gemini-2.5-flash-lite`，Gemini 分组里有这个模型；`gemini-4-argon` 分组只有一个模型，不要用来接 Gemini CLI。

| 用途 | 模型 ID |
| --- | --- |
| 重要任务 | `gemini-4-barium`、`gemini-3.1-pro-preview` |
| 日常编程（推荐默认） | `gemini-3.8-flash` |
| 简单、高频操作 | `gemini-3.1-flash-lite` |

## 3. 写入 `.env`

打开配置文件（没有会新建）：

::: code-group

```powershell [Windows]
mkdir -Force $env:USERPROFILE\.gemini | Out-Null; notepad $env:USERPROFILE\.gemini\.env
```

```bash [Mac / Linux]
mkdir -p ~/.gemini && nano ~/.gemini/.env
```

:::

粘贴下面三行，换成你的 Key。记事本保存时，保存类型选 **所有文件**，文件名 `.env`，编码 **UTF-8**：

```dotenv
GOOGLE_GEMINI_BASE_URL=https://api.myautoapi.com
GEMINI_API_KEY=你的 Gemini 分组 Key
GEMINI_MODEL=gemini-3.8-flash
```

| 变量 | 作用 |
| --- | --- |
| `GOOGLE_GEMINI_BASE_URL` | 请求发到 MyAutoAPI，只填域名，不加 `/v1` 或 `/v1beta` |
| `GEMINI_API_KEY` | MyAutoAPI API Key |
| `GEMINI_MODEL` | 默认模型。不填时默认是 `auto`，会按 Google 的规则自动挑模型，可能挑到分组里没有的 |

## 4. 首次启动

进入项目目录运行 `gemini`：

1. 询问是否信任当前文件夹时，选 **Trust folder**。选 Don't trust 时 Gemini CLI 不读 `.gemini/.env`。
2. 选择登录方式时，选 **Use Gemini API key**（不要选 Login with Google）。
3. 新开对话，发送 `测试`，能正常回复就接入完成。

以后改了 `.env`，退出并重新运行 `gemini` 才生效。

换模型：对话里输入 `/model set gemini-4-barium`，或启动时用 `gemini -m gemini-4-barium`。长期换默认模型，改 `.env` 里的 `GEMINI_MODEL`（它优先于 `settings.json` 里保存的模型）。

## 排查

| 现象 | 处理 |
| --- | --- |
| 一直让你登录 Google | 输入 `/auth`，改选 **Use Gemini API key**；之前用 Google 账号登录过的，登录状态会绕过 MyAutoAPI |
| 能用但后台使用记录里没有 | 用 `/auth` 确认选的是 **Use Gemini API key**；检查 `GOOGLE_GEMINI_BASE_URL`，终端或系统里设置的同名变量优先于 `.env` |
| 只在某个项目里不生效 | 项目目录或上级目录里有自己的 `.env`，Gemini CLI 只读找到的第一个文件，不合并。把三行加进那个项目的 `.gemini/.env` |
| `401` | 重新复制 Key，确认是 Gemini 分组的 Key |
| `403` / `model not found` | `GEMINI_MODEL` 改成第 2 步表里的模型，确认 Key 是 Gemini 分组 |
| 弹出是否切换到备用模型 | 当前模型请求失败了。同意就改用 Gemini CLI 内置的备用模型；想固定模型就拒绝，稍后重试或用 `/model` 换一个 |
