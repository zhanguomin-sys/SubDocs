# Codex Superpowers

Superpowers 是给 Codex 用的一组工作流插件。它的作用不是增加一个新模型，而是让 Codex 在写代码、修 bug、做调研、写文档时按更稳的步骤工作，比如先澄清需求、先写测试、先找根因、完成前先验证。

## 为什么要用

AI 编程最大的问题，不是它不会写代码，而是它太容易直接开写。需求没问清、根因没找到、测试没先写、改完没验证，最后看起来进展很快，实际可能是在制造返工。

Superpowers 解决的是这些问题：

| AI 编程常见问题 | Superpowers 的作用 |
| --- | --- |
| 一上来就写实现 | 先用需求澄清和方案设计把目标说清楚 |
| 修 bug 靠猜 | 先复现、读日志、找根因，再动手改 |
| 写完才补测试 | 用测试驱动开发，先让失败测试证明问题存在 |
| 多任务混在一起 | 用计划和子代理把任务拆开，主控负责审查 |
| 代理说完成就信 | 完成前必须跑测试、构建或检查命令 |
| 文档和调研混入猜测 | 先分清事实、假设和建议，再输出结论 |

所以它适合用在“出错会返工”的任务上：新功能、复杂 bug、多文件修改、资料调研、文档整理、提交前审查。小问题不必硬套流程；但只要任务需要质量和可验证结果，就应该让 Codex 用 Superpowers 的方法做。

第一次使用时，顺序是：

1. 先安装 Superpowers 插件。
2. 重启 Codex CLI、ChatGPT 桌面应用中的 Codex 或 VS Code 插件，或者新开一个任务。
3. 在对话里用对应入口的触发符号，或者直接用中文提示把工作流叫出来。

## 安装方式

### Codex CLI

进入 Codex CLI 后输入：

```text
/plugins
```

搜索：

```text
superpowers
```

找到 **Superpowers** 后安装。安装后退出当前会话，重新打开 Codex CLI。

### ChatGPT 桌面应用中的 Codex / VS Code 插件

在 ChatGPT 桌面应用中切换到 Codex 或 Work，打开 **Plugins**，找到 **Superpowers**，点击 `+` 或安装按钮。

VS Code 插件从 **Plugins** 或 **Skills** 入口搜索并启用 **Superpowers**。安装或更新后，重载 VS Code 窗口，或者新开一个 Codex 会话再测试。

App、VS Code 插件和 CLI 是不同入口。在哪个入口使用，就在哪个入口安装和重启。

## 平时怎么叫出来

不同入口的触发符号不一样：

| 入口 | 怎么叫 |
| --- | --- |
| Codex CLI | 用 `/`，例如 `/brainstorming`；也可以输入 `/b` 让 CLI 筛选候选 |
| ChatGPT 桌面应用中的 Codex | 用 `$` 选择 Skill，例如 `$brainstorming`、`$test-driven-development` |
| VS Code 插件 | 用 `$` 选择 Skill，和 ChatGPT 桌面应用中的 Codex 一样 |
| 不确定当前入口 | 直接用中文说“用 Superpowers 的方法解决当前问题” |

`/` 和 `$` 都不是外部终端命令，而是 Codex 输入框里的触发方式。候选太多时，继续多输入几个字母，或者写完整 Skill 名。

也可以直接用中文叫法：

```text
用 Superpowers 的方法解决当前问题。
```

更明确一点可以这样说：

```text
用 Superpowers 的测试驱动开发方法修这个 bug。
```

```text
用 Superpowers 的子代理驱动开发方法，把能独立处理的任务交给子代理，主控负责审查。
```

```text
用 Superpowers 的系统化调试方法，先找根因，不要直接猜着改。
```

英文 Skill 名主要是检索关键词，不需要用户按英文理解。文档里保留英文，是为了你在 `/` 候选、`$` 候选、插件列表或错误信息里能对得上。

## 常用工作流

| 想做什么 | 工作流 | CLI | App / VS Code |
| --- | --- | --- | --- |
| 先确认需求和方案 | Brainstorming（需求澄清与方案设计） | `/brainstorming` 或 `/b` | `$brainstorming` |
| 把方案拆成执行步骤 | Writing Plans（写执行计划） | `/writing-plans` | `$writing-plans` |
| 多任务开发 | Subagent-Driven Development（子代理驱动开发） | `/subagent-driven-development` 或 `/sub` | `$subagent-driven-development` |
| 多路资料或问题并行调查 | Dispatching Parallel Agents（并行代理调研） | `/dispatching-parallel-agents` | `$dispatching-parallel-agents` |
| 写功能或修 bug | Test-Driven Development（测试驱动开发） | `/test-driven-development` 或 `/t` | `$test-driven-development` |
| 排查复杂问题 | Systematic Debugging（系统化调试） | `/systematic-debugging` 或 `/sys` | `$systematic-debugging` |
| 完成后叫代理审查 | Requesting Code Review（请求代码审查） | `/requesting-code-review` | `$requesting-code-review` |
| 处理审查意见 | Receiving Code Review（处理审查反馈） | `/receiving-code-review` | `$receiving-code-review` |
| 完成前验证 | Verification Before Completion（完成前验证） | `/verification-before-completion` 或 `/v` | `$verification-before-completion` |
| 写新的 Skill | Writing Skills（编写 Skill） | `/writing-skills` | `$writing-skills` |
| 开隔离工作区 | Using Git Worktrees（使用 Git worktree） | `/using-git-worktrees` | `$using-git-worktrees` |
| 开发分支收尾 | Finishing a Development Branch（开发分支收尾） | `/finishing-a-development-branch` | `$finishing-a-development-branch` |

## 常见组合

做一个新功能：

CLI 可以这样依次选择：

```text
/brainstorming
/writing-plans
/test-driven-development
/requesting-code-review
/verification-before-completion
```

在 ChatGPT 桌面应用中的 Codex 或 VS Code 插件里，把 `/` 换成 `$`：

```text
$brainstorming
$writing-plans
$test-driven-development
$requesting-code-review
$verification-before-completion
```

也可以直接用中文说：

```text
用 Superpowers 的方法做这个功能：先确认需求，再写计划，开发时用测试驱动，完成后审查和验证。
```

修一个复杂 bug 时，CLI 依次选择：

```text
/systematic-debugging
/test-driven-development
/verification-before-completion
```

App / VS Code 把 `/` 换成 `$`。对应中文说法：

```text
先用系统化调试找根因，再写复现测试，修完后跑验证命令。
```

做资料调研时，按入口选择并行代理工作流：

```text
/dispatching-parallel-agents
```

App / VS Code 把 `/` 换成 `$`。对应中文说法：

```text
用并行代理做资料搜集：一个读官方文档，一个读本地资料，一个按项目联网规则搜索，最后由主控合并结论。
```

写或改文档：

```text
用 Superpowers 的文档验收清单方式处理：先列必须覆盖点，再写正文，最后叫审查代理检查遗漏、误导性命令和链接。
```

## 关键规则

Superpowers 有用，但不要把所有小事都变成大流程。

| 规则 | 说明 |
| --- | --- |
| 小问题直接处理 | 改错别字、解释一条命令，不必完整走计划和子代理 |
| 新功能先澄清 | 需求没定清楚前，不要直接写实现 |
| 修 bug 先找根因 | 不要只看报错猜一个补丁 |
| 测试驱动不是补测试 | 必须先看到测试因正确原因失败，再写实现 |
| 子代理只做独立任务 | 不要让多个子代理同时改同一文件同一区域 |
| 联网要按项目规则 | 需要搜索时先走项目要求的联网路由 |
| 完成前必须验证 | 没有新鲜构建、测试或检查结果，就不要说已经完成 |
| 项目规则优先 | 用户要求、`AGENTS.md`、生产系统规则优先于 Skill 说明 |

如果只记一句：先安装插件；CLI 里用 `/`，ChatGPT 桌面应用中的 Codex 和 VS Code 插件里用 `$`，也可以直接中文点名；复杂任务用 Superpowers，简单任务别过度流程化。
