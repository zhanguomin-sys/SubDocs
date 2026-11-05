# MyAutoAPI Codex 一键配置

一条命令把 Codex App、ChatGPT 和 Codex CLI 的模型请求切换到 MyAutoAPI中转站。脚本会写入当前用户的 `~/.codex/config.toml` 和 `~/.codex/auth.json`，写入后自动关闭正在运行的 ChatGPT 软件和 Codex。

## 双击运行：MyAutoAPI中转快速接入助手

编号工具 1 是 `1.MyAutoAPI中转快速接入助手`，构建入口是 [desktop/build.py](desktop/build.py)，成品位于 `dist/desktop/`：

- Windows `.exe`：窗口程序，源码在 [quick-connect/](quick-connect/README.md)。粘贴密钥点「一键连接 Codex」；同一窗口也可接入 Claude、安装生图工具和导入国产模型。
- Apple 芯片 Mac `.app`：打开终端运行 `install/codex.sh`。双击打开时读剪贴板里的 `sk-` 密钥，提示直接回车使用，也可手动粘贴。当前发布版使用可跨平台构建的终端启动器，不登记网页导入协议。构建方法见 [desktop/README.md](desktop/README.md)。

网站下载用固定地址，更新时覆盖：

- `https://docs.myautoapi.com/download/myautoapi-quick-connect-windows-x64.exe`
- `https://docs.myautoapi.com/download/myautoapi-quick-connect-mac-arm64.zip`

原有在线命令继续使用下面的入口。

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

在已安装 Codex 的机器上运行对应命令，导入 10 个国产模型；Windows 只读写 `CODEX_HOME` 中的配置和模型文件，Linux 还需要已有 Node.js 16+。官方账号和任意中转站均可导入，默认模型、服务名称、接口和 Key 保留；没有配置时只创建模型目录所需的设置。成功导入后自动关闭 ChatGPT/Codex。重新打开 App 并新建任务选择模型。重复运行同一命令就是更新。

Windows：

```powershell
irm https://docs.myautoapi.com/install/codex-models.ps1 | iex
```

Linux/macOS（同一入口自动识别系统）：

```bash
curl -fsSL https://docs.myautoapi.com/install/codex-models.sh | sh
```

Linux 使用本机已有 Node.js 16+，脚本不会自动安装依赖。模型设定、完整源码、自动测试、独立审查与桌面待验收项目在 [codex-domestic-models](codex-domestic-models/README.md) 内维护。

## 恢复官方登录

Windows 运行 `irm https://docs.myautoapi.com/install/myautoapi-codex-installer/restore.ps1 | iex`，macOS / Linux / WSL 运行 `curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/restore.sh | bash`。脚本先备份 `config.toml` 和 `auth.json`，再删除 `config.toml` 根部的 `model_provider`、`model`、`review_model` 和全部 `[model_providers.*]` 段，并清空 `auth.json`；其它配置保留。完成后退出 Codex，重新打开用官方账号登录。

## 使用过程

1. 在终端明文输入 MyAutoAPI中转站API Key。Key 必须以 `sk-` 开头；输入为空或格式错误时会提示重新输入。自动化调用可通过环境变量 `MYAUTOAPI_KEY` 传入密钥，格式正确时不再询问，格式错误时提示后改为手动输入：

   ```powershell
   $env:MYAUTOAPI_KEY='sk-…'; irm https://docs.myautoapi.com/install/myautoapi-codex-installer/install.ps1 | iex
   ```

   ```bash
   curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/install.sh | MYAUTOAPI_KEY='sk-…' bash
   ```

   写入前请求 `GET https://api.myautoapi.com/v1/usage` 检查密钥：401/403、`isValid` 为 false、密钥过期或额度用完时提示并重新输入；余额为 0 时提示后继续写入。检查不了时说明原因后继续写入：10 秒内没有回应提示“网络超时”（curl 退出码 28；PowerShell 5.1 的 `WebException` 状态为 Timeout，PowerShell 7 为 `TaskCanceledException`/`TimeoutException`），连接失败提示“连不上 MyAutoAPI 服务器”，其它 HTTP 状态或返回不是约定的 JSON 时提示“服务器暂时出错”并带状态码。
2. 已有 `config.toml` 或 `auth.json` 时，脚本自动创建时间戳备份，再写入新配置；没有旧文件时直接创建。
3. 写入完成后，脚本自动关闭当前用户下名称为 `codex` 和 `chatgpt` 的进程，包括 ChatGPT 软件、Codex CLI 和 VS Code 的 Codex 插件，正在运行的任务会中断。脚本只关闭，不重新打开；关闭失败时提示重启电脑，未检测到时提示启动软件测试。
4. 重新打开 App 并新建任务，或启动新的 Codex CLI 会话，发送 `测试`。

## 写入结果

- API 地址：`https://api.myautoapi.com`
- 默认模型：`gpt-6.1-sol`
- 复核模型：`gpt-6.1-sol`
- 推理强度：`high`
- provider：`OpenAI`
- 鉴权：`requires_openai_auth = true`，使用 `auth.json` 中的 API Key
- Goal：`goals = true`
- 图片生成：`image_generation = true`
- 网络访问：`network_access = "enabled"`
- 设置 `approval_policy = "never"`、`sandbox_mode = "danger-full-access"`，并同步 App 完全访问权限；保留 Windows 原有沙箱设置

## 仓库结构

```text
quick-connect/       快速接入助手：Windows 窗口版（Go + Wails）、Mac 启动器 macapp/
desktop/             双击成品的启动器和打包程序
install/
  codex.ps1
  codex.sh
  codex-restore.ps1
  codex-restore.sh
  codex-models.ps1
  codex-models.sh
codex-domestic-models/
  README.md
  src/
  tests/
  install/
tests/
  installers.sh
  fake_usage_server.py   检查密钥用的本地假服务器
```

运行本地检查：

```bash
bash tests/installers.sh
```

检查密钥的请求全部转到 `tests/fake_usage_server.py`，不访问真实接口，也不用真实密钥。

`reference/original/` 是不进 Git 的本机对照材料，没有时跳过两项参考文件检查。临时文件建在 `TMPDIR`（默认系统临时目录）。

## 部署

原 GPT 脚本和还原脚本 `codex-restore.*` 在 `install/` 里维护；国产模型导入脚本只在 `codex-domestic-models/` 内开发、构建和测试，再手动复制交付物到本目录的 `install/`。把要发布的脚本复制到文档站部署位置，再提交并推送：

```bash
cp install/codex.sh install/codex.ps1 install/codex-restore.sh install/codex-restore.ps1 ../docs/public/install/
cp codex-domestic-models/install/codex-models.sh codex-domestic-models/install/codex-models.ps1 install/
cp install/codex-models.sh install/codex-models.ps1 ../docs/public/install/
# 先手动触发 Actions 的 Mac 打包并下载产物，见 desktop/README.md
(cd desktop && python3 build.py --mac-launcher <产物里的 Contents/MacOS/launcher>)
cp dist/desktop/download/myautoapi-quick-connect-windows-x64.exe dist/desktop/download/myautoapi-quick-connect-mac-arm64.zip ../docs/public/download/
```

线上下发的是 `docs/public/install/` 和 `docs/public/download/` 里的那一份。文档站的测试会检查脚本两处一致、Mac 下载包内的 `install.sh` 与线上 `codex.sh` 一致，忘记复制时测试不通过。下载文件的中文文件名由 `docs/public/_headers` 的 `Content-Disposition` 给出。
