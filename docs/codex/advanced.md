---
sidebarTitle: 进阶用法
---

# Codex 进阶

本页讲在系统终端里运行的 `codex ...` 命令、配置分层、工作区和权限、会话管理和多任务并行。进入 Codex 后输入框里的 `/goal`、`/fork` 等命令看 [常用命令](/codex/commands)。

## 外部命令

```bash
codex --version
codex --help
codex features list
```

| 命令 | 用途 |
| --- | --- |
| `codex` | 进入交互式 Codex |
| `codex exec` | 非交互执行一次任务 |
| `codex review` | 非交互代码审查 |
| `codex mcp` | 管理 MCP server |
| `codex plugin` | 管理 Codex 插件 |
| `codex doctor` | 诊断安装、配置、认证和运行环境 |
| `codex debug` | 查看调试信息，例如模型目录和模型可见输入 |
| `codex apply` | 应用 Codex 任务产生的 diff |
| `codex features` | 查看、启用或禁用功能开关 |

没有 `codex goal` 命令；`/goal`、`/resume`、`/fork` 在 Codex 输入框里用。

## 配置分层与模型切换

| 层级 | 放什么 | 示例 |
| --- | --- | --- |
| `config.toml` | 长期默认值：默认模型、provider、sandbox、MCP | 默认使用 MyAutoAPI |
| profile（`<profile>.config.toml`） | 不同任务的模型、权限、工具组合 | `fast`、`review`、`deep` |
| 命令行参数 | 单次任务临时覆盖 | `-m`、`-p`、`-c` |

```bash
codex -m gpt-6.1-sol
codex -p review
codex -c model='"gpt-6.1-sol"'
codex exec -m gpt-6.1-sol "检查当前改动是否有明显 bug"
```

provider 部分直接复制后台生成的 Codex 配置，不要自己推导。`base_url` 填 `https://api.myautoapi.com`，Codex 走 Responses；不要把它改成 Responses 或 Chat Completions 的完整请求路径。

## Codex Home 目录

Codex Home 保存用户级配置、认证、会话和日志。Windows 原生是 `%USERPROFILE%\.codex`，WSL、Mac、Linux 是 `~/.codex`。

| 项目 | 作用 |
| --- | --- |
| `config.toml` | 用户级默认配置 |
| `<profile>.config.toml` | profile 叠加配置，例如 `review.config.toml` |
| `auth.json` | API Key 或认证信息 |
| `sessions/` | 历史会话，供 `/resume`、`/fork` 使用 |
| `logs/` 等 | 日志和诊断记录 |

不要写脚本依赖内部目录结构，排查时用 `codex doctor --summary` 或 `codex doctor --json`。

不同项目、客户或账号要隔离配置时，指定不同的 `CODEX_HOME`：

::: code-group

```powershell [Windows]
$env:CODEX_HOME = "$env:USERPROFILE\.codex-myautoapi"; codex
$env:CODEX_HOME = "$env:USERPROFILE\.codex-review"; codex -p review
```

```bash [Mac / Linux / WSL]
CODEX_HOME="$HOME/.codex-myautoapi" codex
CODEX_HOME="$HOME/.codex-review" codex -p review
```

:::

PowerShell 里设置的 `CODEX_HOME` 对当前窗口后续命令一直有效，换配置时重新设置或新开窗口。

## 工作区、上下文和权限

控制 Codex 能看到什么、能改什么，比写更长的提示词更有效。

| 控制点 | 命令或文件 | 作用 |
| --- | --- | --- |
| 工作根目录 | `-C <DIR>` | 指定 Codex 在哪个目录工作 |
| 额外可写目录 | `--add-dir <DIR>` | 精确授权主工作区之外的目录，不要直接放开整个用户目录 |
| 仓库规则 | `AGENTS.md` | 项目约定、验证命令、禁止事项 |
| 临时约束 | 启动提示词 | 当前任务的目标、范围和验收标准 |
| 沙箱 | `-s` | `read-only` 只读分析；`workspace-write` 改当前项目（常规开发用这个）；`danger-full-access` 完整系统访问 |

```bash
codex -C C:\work\app --add-dir C:\work\shared
codex -s read-only "只阅读代码，解释支付流程，不要修改文件"
codex -s workspace-write "修复登录页表单校验，并运行相关测试"
```

`codex --dangerously-bypass-approvals-and-sandbox` 跳过所有确认并取消沙箱，只在容器、虚拟机或一次性环境里使用，不要在存有真实密钥、生产配置或重要文件的机器上用。

稳定规则写进 `AGENTS.md`，单次目标写在提示词里。不用 `/goal` 时，提示词开头这样写：

```text
目标：修复登录页提交后不显示错误提示的问题。
范围：只允许修改 src/pages/login 和相关测试。
验收：相关测试通过；不要改 API 层；最后说明改动和未验证项。
```

多轮任务先让 Codex 列清单再逐项执行：

```text
先不要改代码。请阅读当前模块，列出最小任务清单，每项都写明完成标准和验证命令。
```

## 非交互任务：`codex exec`

适合边界清楚的一次性任务、脚本化检查和 CI 前的本地检查。需求不明确或要多轮判断的任务，先用交互式 `codex` 梳理，再拆成小任务。

```bash
codex exec "阅读当前仓库，指出最可能影响启动的配置问题"
codex exec -C C:\work\app "运行测试并修复一个最小失败点"
cat task.md | codex exec -                          # 从标准输入读任务
codex exec -o codex-result.md "总结当前未提交改动"    # 最后回复写入文件
codex exec --json "检查当前仓库的测试入口"            # 输出 JSONL 事件流
```

## 会话管理

旧对话越长，每次继续带的上下文越多，费用越高，Codex 也越容易被旧信息干扰。

- 换项目、换目标、换模型、改配置后，新开对话。改完配置不要用旧对话测试。
- 方向变了也新开对话，例如从修 bug 换成写文档、从接入配置换成排错。
- 断网、关错窗口、电脑睡眠后马上回来，用 `/resume` 恢复。超过约 2 小时的旧对话，缓存大概率失效，继续会明显更贵，直接新开。
- 一个阶段做完，让 Codex 总结目标、已完成事项、已改文件、已运行命令、未解决问题和下一步，存成本地文档；下一阶段新开对话，让 Codex 先读这份文档。
- 同一份分析要试两个方向时，用 `/fork`。
- 不要的会话用 `codex archive <SESSION>` 归档；确定不要时再 `codex delete <SESSION>`，删除前确认 session。

手机查看电脑上的 Codex 任务，看 [保留 ChatGPT 登录](/codex/chatgpt-login#手机连接)。手机适合看进度、补一句要求；大段 diff、密钥、删除文件、发布上线回到电脑上确认。

## 多任务并行

`codex features list` 里 `multi_agent` 显示为 stable，但 Codex CLI 没有公开的 `codex subagent` 命令。并行工作靠隔离上下文和工作区：

| 方法 | 适合场景 |
| --- | --- |
| 多个终端 | 同时处理不同任务 |
| 不同工作目录 | 分离前端、后端、文档、测试 |
| 不同 `CODEX_HOME` | 分离账号、配置、插件和会话 |
| 不同 profile | 分离模型、权限、MCP 和任务偏好 |
| `/fork` | 从同一分析拆出多个方向 |

::: code-group

```powershell [Windows]
# 窗口 1
$env:CODEX_HOME = "$env:USERPROFILE\.codex-docs"; codex -C C:\work\docs -p writing
# 窗口 2
$env:CODEX_HOME = "$env:USERPROFILE\.codex-api"; codex -C C:\work\api -p review
```

```bash [Mac / Linux / WSL]
CODEX_HOME="$HOME/.codex-docs" codex -C ~/work/docs -p writing
CODEX_HOME="$HOME/.codex-api" codex -C ~/work/api -p review
```

:::

并行会增加消耗和冲突。多个 Codex 改同一个仓库时，先按目录或文件划好边界，每条线结束后统一检查 diff。

## MCP、插件和联网搜索

```bash
codex mcp list
codex mcp add --help
codex mcp get --help
codex mcp login --help

codex plugin list
codex plugin marketplace list
codex plugin add --help
```

添加外部插件市场时，`owner/repo`、`plugin-name`、`marketplace-name` 按插件维护者的说明填写；普通 GitHub 仓库不是插件市场。

```bash
codex plugin marketplace add owner/repo --ref main
codex plugin add plugin-name@marketplace-name
```

MCP 用来连接明确的数据源或工具，例如内部文档、Issue 系统、只读数据库、设计稿系统；插件用来打包一组可复用能力，例如 Skill、MCP 配置、hooks、脚本。

联网搜索按单次运行打开（`codex exec` 不支持 `--search`，用 `-c web_search=live`）：

```bash
codex --search "查找这个报错的最新官方说明，并给出修复建议"
codex exec -c web_search=live "基于最新官方文档，检查当前配置是否过时"
```

联网搜索会增加消耗。涉及账号、账单、私有仓库、客户资料时，用受控 MCP 或本地文档，不要把敏感信息写进搜索提示词。

## 诊断排错

```bash
codex doctor
codex doctor --summary
codex doctor --json        # 脱敏的机器可读报告，可以发给技术支持
codex debug models
codex debug models --bundled
codex debug prompt-input   # 可能包含任务上下文，分享前检查敏感信息
```

| 现象 | 先检查 |
| --- | --- |
| `401` | `auth.json` 是否为 MyAutoAPI API Key，Key 是否完整 |
| `403` | Key 所属分组是否支持当前模型，额度和权限是否允许 |
| 后台没有记录 | Codex 是否读取了正确的 Codex Home 和 provider 配置 |
| 模型不存在 | `model` 是否和 Key 分组支持的模型名一致 |
| 配置不生效 | 是否把 Windows 原生和 WSL 的配置混用 |
| 行为和预期不一致 | 是否被 profile、命令行 `-c` 或项目 `AGENTS.md` 覆盖 |
