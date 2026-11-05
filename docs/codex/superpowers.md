---
sidebarTitle: Superpowers
---

# Codex Superpowers

Superpowers 是给 Codex 用的一组工作流插件，不是新模型。它让 Codex 按更稳的步骤做事：先澄清需求、先找根因、先写测试、完成前先验证，减少“看起来很快、实际返工”的情况。

适合出错会返工的任务：新功能、复杂 bug、多文件修改、资料调研、文档整理、提交前审查。改错别字、解释一条命令这类小事直接处理，不用走流程。

## 安装

在哪个入口用，就在哪个入口安装：

| 入口 | 安装方法 |
| --- | --- |
| Codex CLI | 输入 `/plugins`，搜索 `superpowers`，安装后退出当前会话，重新打开 Codex CLI |
| ChatGPT 桌面应用中的 Codex | 切换到 Codex 或 Work，打开 **Plugins**，找到 **Superpowers**，点击 `+` 或安装按钮，安装后新开 Codex 任务 |
| VS Code 插件 | 从 **Plugins** 或 **Skills** 入口搜索并启用 **Superpowers**，重载 VS Code 窗口或新开 Codex 会话 |

## 调用

| 入口 | 怎么叫 |
| --- | --- |
| Codex CLI | 用 `/`，例如 `/brainstorming`；输入 `/b` 等前几个字母可以筛选 |
| ChatGPT 桌面应用中的 Codex、VS Code 插件 | 用 `$`，例如 `$brainstorming`、`$test-driven-development` |
| 任何入口 | 直接用中文说“用 Superpowers 的方法解决当前问题” |

`/` 和 `$` 是 Codex 输入框里的触发方式，不是终端命令。中文点名可以更具体：

```text
用 Superpowers 的系统化调试方法，先找根因，不要直接猜着改。
```

```text
用 Superpowers 的子代理驱动开发方法，把能独立处理的任务交给子代理，主控负责审查。
```

## 常用工作流

表里是 CLI 写法；ChatGPT 桌面应用和 VS Code 插件把 `/` 换成 `$`。

| 想做什么 | 工作流 | CLI |
| --- | --- | --- |
| 先确认需求和方案 | Brainstorming（需求澄清与方案设计） | `/brainstorming` 或 `/b` |
| 把方案拆成执行步骤 | Writing Plans（写执行计划） | `/writing-plans` |
| 多任务开发 | Subagent-Driven Development（子代理驱动开发） | `/subagent-driven-development` 或 `/sub` |
| 多路资料或问题并行调查 | Dispatching Parallel Agents（并行代理调研） | `/dispatching-parallel-agents` |
| 写功能或修 bug | Test-Driven Development（测试驱动开发） | `/test-driven-development` 或 `/t` |
| 排查复杂问题 | Systematic Debugging（系统化调试） | `/systematic-debugging` 或 `/sys` |
| 完成后叫代理审查 | Requesting Code Review（请求代码审查） | `/requesting-code-review` |
| 处理审查意见 | Receiving Code Review（处理审查反馈） | `/receiving-code-review` |
| 完成前验证 | Verification Before Completion（完成前验证） | `/verification-before-completion` 或 `/v` |
| 写新的 Skill | Writing Skills（编写 Skill） | `/writing-skills` |
| 开隔离工作区 | Using Git Worktrees（使用 Git worktree） | `/using-git-worktrees` |
| 开发分支收尾 | Finishing a Development Branch（开发分支收尾） | `/finishing-a-development-branch` |

## 常见组合

| 任务 | 依次使用 | 中文说法 |
| --- | --- | --- |
| 做新功能 | `/brainstorming` → `/writing-plans` → `/test-driven-development` → `/requesting-code-review` → `/verification-before-completion` | 用 Superpowers 的方法做这个功能：先确认需求，再写计划，开发时用测试驱动，完成后审查和验证。 |
| 修复杂 bug | `/systematic-debugging` → `/test-driven-development` → `/verification-before-completion` | 先用系统化调试找根因，再写复现测试，修完后跑验证命令。 |
| 资料调研 | `/dispatching-parallel-agents` | 用并行代理做资料搜集：一个读官方文档，一个读本地资料，一个按项目联网规则搜索，最后由主控合并结论。 |
| 写或改文档 | — | 用 Superpowers 的文档验收清单方式处理：先列必须覆盖点，再写正文，最后叫审查代理检查遗漏、误导性命令和链接。 |

## 关键规则

| 规则 | 说明 |
| --- | --- |
| 新功能先澄清 | 需求没定清楚前不写实现 |
| 修 bug 先找根因 | 不要只看报错猜一个补丁 |
| 测试驱动不是补测试 | 先看到测试因正确原因失败，再写实现 |
| 子代理只做独立任务 | 不让多个子代理同时改同一文件的同一区域 |
| 联网按项目规则 | 需要搜索时先走项目要求的联网方式 |
| 完成前必须验证 | 没有新跑的构建、测试或检查结果，不算完成 |
| 项目规则优先 | 用户要求、`AGENTS.md`、生产系统规则优先于 Skill 说明 |
