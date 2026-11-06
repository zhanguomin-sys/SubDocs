# Codex 进阶

本页适合已经完成 Codex 基础安装和 MyAutoAPI 接入的用户。首次配置 API Key、安装 CLI、Windows / WSL / macOS / Linux 路径排查，请先看 [Codex 接入](/clients/codex)。

本页主要讲系统终端里的外部 `codex ...` 命令、高级配置、多会话工作流和 ChatGPT 桌面应用中 Codex 的手机远程查看。已经进入 Codex 交互界面后，输入框里的 `/goal`、`/fork`、`/review`、`/model` 等内部命令，请看 [Codex CLI 内部命令](/external/codex-cli-internal-commands)。

## 查看当前版本和命令

```bash
codex --version
codex --help
codex features list
```

常用外部命令包括：

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

这些是外部 CLI 命令，不等于进入 Codex 之后输入的内部 `/...` 命令。例如 `/goal` 是交互界面里的内部命令，不是 `codex goal`；恢复对话和分叉对话，优先进入 Codex 后使用 `/resume`、`/fork`。

## 配置分层与模型切换

Codex 的配置可以来自基础配置、profile 和命令行参数。高级用户最常用的是“长期默认值放配置文件，临时选择放命令行”。

| 层级 | 适合放什么 | 示例 |
| --- | --- | --- |
| `config.toml` | 默认模型、provider、sandbox、MCP 等长期配置 | 默认使用 MyAutoAPI |
| profile | 不同任务的模型、权限、工具组合 | `fast`、`review`、`deep` |
| 命令行参数 | 单次任务临时覆盖 | `-m`、`-p`、`-c` |

常用切换方式：

```bash
codex -m gpt-5.5
codex -p review
codex -c model='"gpt-5.5"'
codex exec -m gpt-5.5 "检查当前改动是否有明显 bug"
```

MyAutoAPI 用户优先复制后台生成的 Codex 配置，不要在高级配置里重新推导 provider 字段。MyAutoAPI API 地址通常是 `https://api.myautoapi.com`；Codex 接入优先使用 Responses。手动排障时，检查后台生成配置和基础接入页，不要把 Base URL 改成 Responses 或 Chat Completions 的最终请求路径。

## Codex Home 目录解释

Codex Home 是 Codex 保存用户级配置、认证、会话和运行记录的目录。常见位置如下：

| 环境 | 常见 Codex Home |
| --- | --- |
| Windows 原生 PowerShell / CMD | `%userprofile%\.codex` |
| Windows WSL | `~/.codex` |
| macOS | `~/.codex` |
| Linux | `~/.codex` |

常见文件和目录：

| 项目 | 作用 |
| --- | --- |
| `config.toml` | 用户级默认配置 |
| `<profile>.config.toml` | profile 叠加配置，例如 `review.config.toml` |
| `auth.json` | API Key 或认证信息 |
| `sessions/` | 历史会话记录，供 `/resume`、`/fork` 等功能使用 |
| `logs/` 或类似目录 | 日志、诊断或运行记录 |

不同版本的本地目录结构可能略有差异，不建议写脚本依赖某个固定内部目录。需要排查时，优先运行：

```bash
codex doctor --summary
codex doctor --json
```

为不同项目、不同客户或不同账号隔离配置时，可以显式指定不同的 `CODEX_HOME`：

```bash
CODEX_HOME="$HOME/.codex-myautoapi" codex
CODEX_HOME="$HOME/.codex-review" codex -p review
```

## 工作区 / 上下文控制

Codex 对代码仓库的理解高度依赖工作目录、可读写目录和仓库内指导文件。高级用法的核心不是给更长提示词，而是控制它能看到什么、能改什么。

| 控制点 | 命令或文件 | 作用 |
| --- | --- | --- |
| 工作根目录 | `-C <DIR>` | 指定 Codex 在哪个目录工作 |
| 额外可写目录 | `--add-dir <DIR>` | 允许 Codex 访问主工作区之外的目录 |
| 仓库规则 | `AGENTS.md` | 写入项目约定、验证命令、禁止事项 |
| 临时约束 | 启动提示词 | 给当前任务的目标、范围和验收标准 |
| 沙箱 | `-s read-only/workspace-write/danger-full-access` | 限制命令和文件系统权限 |

示例：

```bash
codex -C ~/work/app
codex -C ~/work/app --add-dir ~/work/shared
codex -s read-only "只阅读代码，解释支付流程，不要修改文件"
codex -s workspace-write "修复登录页表单校验，并运行相关测试"
```

稳定规则写进仓库里的 `AGENTS.md`，单次任务目标写在提示词里。不要把所有背景一次性塞进提示词；上下文越混杂，Codex 越难判断优先级。

## 非交互任务

`codex exec` 适合一次性任务、脚本化检查、批量说明生成和 CI 前的本地辅助检查。

```bash
codex exec "阅读当前仓库，指出最可能影响启动的配置问题"
codex exec -C ~/work/app "运行测试并修复一个最小失败点"
```

也可以从标准输入传入复杂说明：

```bash
cat task.md | codex exec -
```

把最后回复写入文件，方便后续脚本读取：

```bash
codex exec -o codex-result.md "总结当前未提交改动"
```

需要结构化事件流时，可以使用 JSONL 输出：

```bash
codex exec --json "检查当前仓库的测试入口"
```

非交互任务更适合边界清楚的工作。对于大范围重构、需求不明确或需要多轮判断的任务，先用交互式 `codex` 梳理方案，再拆成小任务执行。

## 会话管理

Codex 会保存历史会话，但不要把一个旧对话长期当成所有任务的入口。对话越长，历史内容越多，每次继续都会带上更多上下文，token 消耗和费用都会变高，Codex 也更容易被旧信息干扰。

换项目、换目标、换模型、改配置后，都新开对话。改完配置后不要用旧对话测试，新开对话后再发送测试消息。

断网、关错窗口、电脑睡眠、临时退出后马上回来，可以进入 Codex 后用 `/resume` 恢复原对话。超过 2 小时左右的旧对话，缓存大概率已经不稳定；继续旧对话可能吃不到缓存，费用会明显增加，直接新开对话更合适。

方向变化也要新开对话。从修 bug 改成写文档，从接入配置改成排错，从实现功能改成代码审查，都属于方向变化，不要在同一个对话里反复切目标。

一个阶段做完后，让 Codex 总结目标、已完成事项、已改文件、已运行命令、未解决问题和下一步。把总结保存到本地文档，下一阶段新开对话，把总结贴进去。关键经验和交接信息要存到本地文档，不要只留在聊天记录里。

同一份分析要尝试两个方向时，用 `/fork` 分出新对话。比如一条继续修复，一条只做审查或整理方案。

不再需要的历史会话可以用 `codex archive <SESSION>` 归档；确定不要保留时再用 `codex delete <SESSION>` 删除。删除前先确认目标 session，避免误删。

## 在手机端查看桌面 Codex 任务

ChatGPT 桌面应用中的 Codex 保持 ChatGPT 登录后，手机端打开 ChatGPT 的 **Remote** 标签，就能查看受支持的桌面任务。电脑继续保留项目文件、终端环境、权限和长任务；手机负责查看进度、补充要求、让任务继续跑。

完整配置看 [保留 ChatGPT 登录](/clients/chatgpt-login-myautoapi)。进阶用法只记这几步：

1. 电脑端打开 ChatGPT 桌面应用，切换到 Codex 后启动任务。
2. 手机端打开 ChatGPT，进入 **Remote** 标签。
3. 选择当前电脑上支持继续查看的 Codex 任务。

适合场景：

| 场景 | 用法 |
| --- | --- |
| 通勤或外出 | 查看任务进度，补一句要求，让 Codex 继续处理。 |
| 会议中 | 让 Codex 根据当前项目整理清单、复盘结果或下一步计划。 |
| 电脑放在固定工位 | 手机远程控制同一台电脑，不用在新设备重新配项目环境。 |
| 长任务等待中 | 出门后继续看回复，必要时让它跑测试、改文档或整理 diff。 |

手机端适合轻量控制。涉及大段 diff、密钥、删除文件、发布上线时，回到电脑端确认。

## 诊断排错

先用 `doctor` 判断安装、配置、认证和运行环境：

```bash
codex doctor
codex doctor --summary
codex doctor --json
```

`doctor --json` 会输出经过脱敏的机器可读报告，适合发给技术支持或保存到排障记录中。

常见排查路径：

| 现象 | 优先检查 |
| --- | --- |
| 401 | `auth.json` 是否为 MyAutoAPI API Key，Key 是否完整 |
| 403 | Key 所属分组是否支持当前模型，额度或权限是否允许 |
| 后台没有记录 | Codex 是否读取了正确的 Codex Home 和 provider 配置 |
| 模型不存在 | `model` 是否和 MyAutoAPI 后台分组支持的模型名一致 |
| 配置不生效 | 是否在错误环境配置，例如 Windows 原生和 WSL 混用 |
| 行为和预期不一致 | 是否被 profile、命令行 `-c` 或项目规则覆盖 |

需要看模型目录或模型可见输入时，可以使用：

```bash
codex debug models
codex debug models --bundled
codex debug prompt-input
```

`debug prompt-input` 可能包含当前任务上下文。分享给他人前应先检查是否有敏感信息。

## MCP / 插件 / 联网搜索

Codex 支持通过 MCP 接入外部工具，也支持插件管理。当前 CLI 可验证命令：

```bash
codex mcp list
codex mcp add --help
codex mcp get --help
codex mcp login --help

codex plugin list
codex plugin marketplace list
codex plugin add --help
```

需要添加外部插件市场来源时，按插件维护者给出的仓库和市场名执行：

```bash
codex plugin marketplace add owner/repo --ref main
codex plugin add plugin-name@marketplace-name
```

这里的 `owner/repo`、`plugin-name` 和 `marketplace-name` 必须按插件维护者说明填写。不要把普通 GitHub 仓库当成 Codex 插件市场。

MCP 更适合连接明确的数据源或工具，例如内部文档、Issue 系统、数据库只读查询、设计稿系统。插件更适合打包一组可复用能力，例如技能、MCP 配置、hooks、脚本或本地工具。

联网搜索是单次运行级能力，可以通过 `--search` 打开：

```bash
codex --search "查找这个报错的最新官方说明，并给出修复建议"
codex exec --search "基于最新官方文档，检查当前配置是否过时"
```

联网搜索会增加外部信息摄入，也可能增加消耗。涉及账号、账单、私有仓库、客户资料时，优先使用受控 MCP 或本地文档，不要把敏感信息直接放进联网搜索提示词。

## 安全与权限

Codex 能读写文件、运行命令，因此权限配置应按任务风险选择。

| 模式 | 适合场景 |
| --- | --- |
| `read-only` | 只读分析、解释代码、审查方案 |
| `workspace-write` | 修改当前项目文件，常规开发推荐 |
| `danger-full-access` | 需要完整系统访问的特殊任务 |

常用示例：

```bash
codex -s read-only "解释当前项目结构"
codex -s workspace-write "修复当前测试失败"
```

谨慎使用：

```bash
codex --dangerously-bypass-approvals-and-sandbox
```

这个选项会跳过确认并取消沙箱限制，只适合已经由外部容器、虚拟机或一次性环境隔离的场景。不要在保存真实密钥、生产配置或重要个人文件的机器上随手使用。

让 Codex 读取或修改主项目之外的目录时，优先用 `--add-dir` 精确授权，不要直接扩大到整个用户目录。

## 并行工作 / 多 Codex 工作流

并行使用 Codex 的重点是隔离上下文和工作区，而不是寻找一个不存在的公开 `subagent` 命令。当前 CLI 虽然显示 `multi_agent` feature 为 stable，但主帮助没有公开 `codex subagent` 子命令。不要把内部环境里的 agent 编排能力当成所有 Codex 用户都能直接调用的公开命令。

可用的高级工作流：

| 方法 | 适合场景 |
| --- | --- |
| 多个终端 | 同时处理不同任务 |
| 不同工作目录 | 分离前端、后端、文档、测试 |
| 不同 `CODEX_HOME` | 分离账号、配置、插件和会话 |
| 不同 profile | 分离模型、权限、MCP 和任务偏好 |
| `/fork` / `/resume` | 从同一分析上下文拆出多个方向 |

示例：

```bash
CODEX_HOME="$HOME/.codex-docs" codex -C ~/work/docs -p writing
CODEX_HOME="$HOME/.codex-api" codex -C ~/work/api -p review
```

从同一会话拆任务：

```text
/fork 基于刚才的分析，只修复登录 bug
/fork 基于刚才的分析，只补测试计划，不改代码
```

并发会增加模型调用、工具调用和搜索消耗，也会增加冲突概率。多个 Codex 同时修改同一个仓库时，先按目录或文件划分边界，并在每条线结束后统一检查 diff。

## 目标管理

目标管理的核心是让 Codex 始终知道“完成标准”，不只是知道“要做什么”。`/goal` 是进入 Codex 交互界面后的内部命令，不是系统终端里的 `codex goal` 子命令。内部命令的详细用法看 [Codex CLI 内部命令](/external/codex-cli-internal-commands)。

常用做法：

| 场景 | 方法 |
| --- | --- |
| 交互式长任务 | 在 Codex 输入框里使用 `/goal <目标>` |
| 单次任务 | 在提示词开头写清目标、范围和验收标准 |
| 仓库长期约定 | 写入 `AGENTS.md` |
| 多阶段任务 | 先让 Codex 输出任务清单，再逐项执行 |
| 会话延续 | 用 `/resume` 保留上下文 |
| 方向分叉 | 用 `/fork` 复制上下文后改变目标 |

交互式任务可以先设置目标：

```text
/goal 修复登录页提交后不显示错误提示的问题。范围只包括 src/pages/login 和相关测试。完成标准是相关测试通过，并说明改动和未验证项。
```

不用内部命令时，也可以在普通提示词开头写清目标：

```text
目标：修复登录页提交后不显示错误提示的问题。
范围：只允许修改 src/pages/login 和相关测试。
验收：相关测试通过；不要改 API 层；最后说明改动和未验证项。
```

对于需要多轮推进的任务，可以先让 Codex 建立清单：

```text
先不要改代码。请阅读当前模块，列出最小任务清单，每项都写明完成标准和验证命令。
```

目标管理不需要复杂系统。能稳定复用的规则放进 `AGENTS.md`，临时目标可以用 `/goal` 或写进提示词，历史上下文用 `/resume`，分支探索用 `/fork`。
