# Hermes Agent 接入

Hermes Agent 是可以在终端和聊天软件里处理项目任务的 AI Agent，适合长期管理文件、执行工具和处理自动化任务。

通过 Hermes 的 `Custom endpoint` 添加 MyAutoAPI 后，Hermes 会通过 `https://api.myautoapi.com/v1` 使用模型。配置入口是 `hermes model`。

## 安装 Hermes

Linux、macOS、WSL2 终端运行：

```bash
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
```

重新打开终端，确认 Hermes 可用：

```bash
hermes --version
```

看不到版本号时，先重新打开终端；Windows 原生命令行遇到路径或依赖问题，改用 WSL2 终端安装。

## 填写 MyAutoAPI

打开模型向导：

```bash
hermes model
```

Provider 选择 `Custom endpoint`，然后按提示填写：

| Hermes 提示 | 填写 |
| --- | --- |
| API Base URL / Base URL | `https://api.myautoapi.com/v1` |
| API Key | MyAutoAPI API Key |
| Model name | 当前 Key 分组可用模型，例如 `claude-fable-5`、`gemini-3.6-flash` 或 `grok-4.5` |
| Context window | 不强制；必须填写时按模型上下文填写 |

Base URL 只填到 `/v1`，不要追加其它路径。

## 启动并测试

配置完成后运行：

```bash
hermes
```

进入 Hermes 后发送：

```text
测试
```

能正常回复，说明 Hermes 已经通过 MyAutoAPI 发起请求。

## 排查

先查看 Hermes 当前读取的配置：

```bash
hermes config
```

重点看三项：

| 字段 | 应该看到 |
| --- | --- |
| `provider` | `custom` |
| `default` / `model` | 当前 Key 分组可用模型 |
| `base_url` | `https://api.myautoapi.com/v1` |

| 现象 | 处理 |
| --- | --- |
| `401 Unauthorized` | 重新复制 MyAutoAPI API Key，再跑一次 `hermes model` |
| 模型不存在 | 换成当前 Key 分组里的完整模型名 |
| 路径错误 | Base URL 改回 `https://api.myautoapi.com/v1` |
| Hermes 仍用旧模型 | 退出 Hermes，重新运行 `hermes model` |
| `403 block` / `403 Forbidden` | 先确认 Key 分组；仍失败时看 [外接调用 User-Agent 说明](/external/user-agent) |

## 进阶命令

能正常使用后再看这些命令：

| 命令 | 用途 |
| --- | --- |
| `hermes config` | 查看当前配置 |
| `hermes config set terminal.backend docker` | 让终端工具在 Docker 隔离环境里运行 |
| `hermes dashboard` | 打开网页控制台，管理会话、配置和任务 |
| `hermes --tui` | 使用终端图形界面 |
| `hermes doctor` | 诊断安装、依赖和环境问题 |
| `hermes model` | 重新选择模型和接口 |
| `hermes sessions list` | 查看历史会话 |
| `hermes --continue` | 继续最近一次会话 |
| `hermes gateway status` | 查看消息网关状态 |
| `hermes update` | 更新 Hermes |
