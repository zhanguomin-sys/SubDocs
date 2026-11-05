---
sidebarTitle: Hermes
---

# Hermes Agent 接入

Hermes Agent 是能在终端和聊天软件里处理项目任务的 AI Agent，适合长期管理文件、执行工具和跑自动化任务。在 `hermes model` 向导里添加 MyAutoAPI 的 `Custom endpoint` 后，Hermes 通过 `https://api.myautoapi.com/v1` 使用模型。

## 1. 安装 Hermes

::: code-group

```powershell [Windows]
iex (irm https://hermes-agent.nousresearch.com/install.ps1)
```

```bash [Mac / Linux / WSL2]
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
```

:::

关闭并重新打开 PowerShell 或终端，运行 `hermes --version`，能看到版本号就装好了。

## 2. 填写 MyAutoAPI

运行 `hermes model`，Provider 选择 `Custom endpoint`，按提示填写：

| Hermes 提示 | 填写 |
| --- | --- |
| API Base URL / Base URL | `https://api.myautoapi.com/v1`，只填到 `/v1` |
| API Key | MyAutoAPI API Key |
| Model name | 当前 Key 分组可用模型，例如 `claude-fable-5-1`、`gemini-3.8-flash`、`grok-4.7`；Claude 模型要用 **CC MAX 外接分组** 的 Key |
| Context window | 能跳过就跳过；必须填时按模型上下文填写 |

## 3. 启动并测试

运行 `hermes`（新开会话，不用 `--continue`），发送 `测试`。能正常回复就接入完成。

## 排查

先运行 `hermes config` 看 Hermes 当前读取的配置：

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
| `403 block` / `403 Forbidden` | 核对 Key 分组；仍失败看 [User-Agent 说明](/external/user-agent) |
| 安装出错或依赖缺失 | 运行 `hermes doctor` 诊断安装、依赖和环境 |

## 进阶命令

| 命令 | 什么时候用 |
| --- | --- |
| `hermes model` | 换模型或换接口 |
| `hermes --continue` | 继续最近一次会话，例如误关终端后接着做 |
| `hermes sessions list` | 找以前的会话 |
| `hermes dashboard` | 在网页控制台里管理会话、配置和任务 |
| `hermes --tui` | 想用终端图形界面操作 |
| `hermes config set terminal.backend docker` | 让 Hermes 执行的终端命令跑在 Docker 隔离环境里，避免改到本机 |
| `hermes gateway status` | 接了聊天软件后，查看消息网关是否在运行 |
| `hermes update` | 更新 Hermes |
