# MyAutoAPI Codex 一键配置

一条命令把 Codex App、ChatGPT 和 Codex CLI 的模型请求切换到 MyAutoAPI中转站。脚本会写入当前用户的 `~/.codex/config.toml` 和 `~/.codex/auth.json`，写入后自动关闭正在运行的 ChatGPT 软件和 Codex。

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

## 使用过程

1. 在终端明文输入 MyAutoAPI中转站API Key。Key 必须以 `sk-` 开头；输入为空或格式错误时会提示重新输入。
2. 已有 `config.toml` 或 `auth.json` 时，脚本自动创建时间戳备份，再写入新配置；没有旧文件时直接创建。
3. 写入完成后，脚本自动关闭当前用户下名称为 `codex` 和 `chatgpt` 的进程，包括 ChatGPT 软件、Codex CLI 和 VS Code 的 Codex 插件，正在运行的任务会中断。脚本只关闭，不重新打开；关闭失败时提示重启电脑，未检测到时提示启动软件测试。
4. 重新打开 App 并新建任务，或启动新的 Codex CLI 会话，发送 `测试`。

## 写入结果

- API 地址：`https://api.myautoapi.com`
- 默认模型：`gpt-5.6-sol`
- 复核模型：`gpt-5.6-sol`
- 推理强度：`high`
- provider：`OpenAI`
- 鉴权：`requires_openai_auth = true`，使用 `auth.json` 中的 API Key
- Goal：`goals = true`
- 图片生成：`image_generation = true`
- 网络访问：`network_access = "enabled"`
- 不写入沙盒或审批字段

## 仓库结构

```text
install/
  install.ps1
  install.sh
tests/
  installers.sh
```

运行本地检查：

```bash
bash tests/installers.sh
```

## 部署

脚本只在 `install/` 里修改。本地检查通过后，把两个脚本复制到文档站的部署位置，再提交并推送：

```bash
cp install/install.sh install/install.ps1 ../docs/public/install/myautoapi-codex-installer/
```

线上下发的是 `docs/public/install/` 里的那一份。文档站的测试会检查两处内容一致，忘记复制时测试不通过。
