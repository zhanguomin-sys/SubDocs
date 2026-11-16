# MyAutoAPI Codex 一键配置

一条命令把 Codex App、ChatGPT 和 Codex CLI 的模型请求切换到 MyAutoAPI 中转站。脚本会备份并更新当前用户的 `~/.codex/config.toml` 和 `~/.codex/auth.json`，写入前自动关闭正在运行的 ChatGPT 软件和 Codex。

## Windows

打开 PowerShell，运行：

```powershell
irm https://docs.myautoapi.com/install/myautoapi-codex-installer/install.ps1 | iex
```

## macOS / Linux / WSL

打开终端，运行：

```bash
curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/install.sh | bash
```

## 给 Codex 增加国产模型

已经接入 Codex 后，运行对应命令可导入 10 个国产模型，同时保留原有 GPT、默认模型、接口和 Key。Windows 只读写 `CODEX_HOME` 中的配置和模型文件，不依赖 Codex CLI；导入成功后各平台都会自动关闭 ChatGPT/Codex。完成后把 MyAutoAPI API Key 切换到 **国产模型大全 0.45x** 分组，重新打开 App 并新建任务选择模型。重复运行同一命令就是更新。

Windows：

```powershell
irm https://docs.myautoapi.com/install/codex-models.ps1 | iex
```

Linux/macOS（同一入口自动识别系统）：

```bash
curl -fsSL https://docs.myautoapi.com/install/codex-models.sh | sh
```

Linux 使用本机已有 Node.js 16+，脚本不会自动安装依赖。模型设定、完整源码、自动测试、独立审查与桌面待验收项目在 [codex-domestic-models](codex-domestic-models/README.md) 内维护。

## 使用过程

1. 在终端明文输入 MyAutoAPI中转站API Key。Key 必须以 `sk-` 开头；输入为空或格式错误时会提示重新输入。
2. 脚本先通过 `/v1/usage` 检查 Key；无效或停用的 Key 需要重新输入。余额为零时提示充值，网络异常时说明未完成在线检查并继续配置。已有文件先创建时间戳备份。
3. 写入配置前，脚本自动关闭当前用户下名称为 `codex` 和 `chatgpt` 的进程，包括 ChatGPT 软件、Codex CLI 和 VS Code 的 Codex 插件，正在运行的任务会中断。脚本只关闭，不重新打开；关闭失败时提示重启电脑，未检测到时提示启动软件测试。
4. 重新打开 App 并新建任务，或启动新的 Codex CLI 会话，发送 `测试`。

## 写入结果

- API 地址：`https://api.myautoapi.com`
- 默认模型：`gpt-6.1-sol`
- provider：`OpenAI`
- 协议：`responses`
- 鉴权：`requires_openai_auth = true`，使用 `auth.json` 中的 API Key

新配置只写入上述接口和默认模型字段。已有配置保留其它内容，但会清理旧 provider、根级复核模型/推理强度/沙盒/审批/网络字段、`model_catalog_json`、`profile` 和强制登录设置；旧文件有备份。`[features]`、`[windows]` 等其它段保留。重新接入后，需要国产模型时再次运行国产模型导入入口。脚本不替用户启用完全访问或额外功能。

## 仓库结构

```text
install/
  install.ps1
  install.sh
  codex-models.ps1
  codex-models.sh
codex-domestic-models/
  README.md
  src/
  tests/
  install/
tests/
  installers.sh
```

运行本地检查：

```bash
bash tests/installers.sh
```

## 部署

原 GPT 脚本在 `install/` 里维护；国产模型导入脚本在 `codex-domestic-models/` 内开发、构建和测试。把发布文件同步到文档站后再提交：

```bash
cp install/install.sh install/install.ps1 ../docs/public/install/myautoapi-codex-installer/
cp codex-domestic-models/install/codex-models.sh codex-domestic-models/install/codex-models.ps1 install/
cp install/codex-models.sh install/codex-models.ps1 ../docs/public/install/
```

线上下发的是 `docs/public/install/` 里的那一份。测试会检查子项目与上级安装目录一致，并检查站点 PowerShell 副本只移除了开头的 UTF-8 BOM。
