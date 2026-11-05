---
sidebarTitle: Codex
---

# Codex

Codex 是 OpenAI 的编程助手，能读代码、改文件、运行命令。第一次接入 MyAutoAPI 按 [快速开始](/quick-start) 操作。

## 安装 Codex

Windows 打开 [ChatGPT 下载页](https://chatgpt.com/download/)，选择 Windows，在 Microsoft Store 安装。打开 ChatGPT，在左上角切换到 **ChatGPT Codex**。

需要在终端里使用 Codex CLI 时，再运行：

::: code-group

```powershell [Windows]
winget install OpenJS.NodeJS.LTS
npm install -g @openai/codex
codex --version
```

```bash [Mac]
npm install -g @openai/codex
codex --version
```

```bash [Linux / WSL]
sudo apt update
sudo apt install -y nodejs npm
npm install -g @openai/codex
codex --version
```

:::

WSL 里的 Codex 读取 WSL 的 `~/.codex`，不读取 Windows 用户目录。

## 本目录

| 你要做什么 | 进入 |
| --- | --- |
| 不运行脚本，手动写配置 | [手动配置](/codex/manual-config) |
| 接入后保留 ChatGPT 登录和手机连接 | [保留 ChatGPT 登录](/codex/chatgpt-login) |
| 查 `/goal`、`/resume` 等命令什么时候用 | [常用命令](/codex/commands) |
| 用 profile、多会话、权限和项目规则 | [进阶用法](/codex/advanced) |
| 创建和复用 Skill | [Skills](/codex/skills) |
| 用 Superpowers 固定开发流程 | [Superpowers](/codex/superpowers) |
| 安装 Git、ripgrep 等配套工具 | [周边工具](/codex/tools) |
| 让 AI 帮你用 Git 提交、推送和备份 | [Git 妙用](/codex/git) |
| Windows 上 Codex 日志库频繁写盘 | [日志写盘](/codex/log-sqlite) |

## 常见问题

| 遇到的问题 | 处理方法 |
| --- | --- |
| `codex` 命令不存在 | 重新打开终端；确认 `npm install -g @openai/codex` 执行成功 |
| Windows 配好了，WSL 里不生效 | 在 WSL 里重新配置 `~/.codex` |
| `401 Unauthorized` | 重新复制 `auth.json`，检查 Key 是否完整、前后有没有空格 |
| `403` 或 `model not found` | Key 换成支持 `config.toml` 中模型的 GPT 分组 |
| 自己填写 `base_url` 后报错 | Codex 填 `https://api.myautoapi.com`，不加 `/v1` |

其他报错看 [报错与踩坑](/errors/)。
