# Codex 周边工具

这页不是工具大全，也不是必须安装清单。

先把 Codex 本身和 MyAutoAPI 配好，再按实际需要补工具。多数用户先装 **Git、Node.js / npm、ripgrep** 就够了；需要经常处理 GitHub、MCP 或规范化开发流程时，再考虑后面的增强工具。

## 先装哪些

| 顺序 | 工具 | 什么时候装 | 安装后检查 |
| --- | --- | --- | --- |
| 1 | Git | 让 Codex 修改代码仓库前先装 | `git --version` |
| 2 | Node.js / npm | 用 npm 安装 Codex、前端工具、MCP 工具或流程插件时 | `node -v`、`npm -v` |
| 3 | ripgrep | 项目文件多，想让 Codex 快速搜索代码和报错文本时 | `rg --version` |
| 4 | GitHub CLI | 经常处理 GitHub issue、PR、release、CI 时 | `gh auth login` |
| 5 | MCPHub | 已经在用多个 MCP 服务，手动管理配置开始变乱时 | 进入 MCPHub 文档继续配置 |
| 6 | Superpowers for Codex | 经常做复杂功能、排 bug、写测试，需要固定工作流程时 | 在 Codex 里确认能调用 Superpowers |

## 让 AI 帮你安装

把工具名改成自己要装的那一个，不要一次让 AI 装一堆。

```text
请帮我在当前电脑安装【工具名】。

要求：
1. 先判断当前系统，并检查这个工具是否已经安装；已安装就告诉我版本，不要重复安装。
2. 没安装时，优先使用当前系统常见安装方式。
3. 需要管理员权限、下载安装包、改系统环境变量时，先说明要做什么，等我确认。
4. 安装完成后运行验证命令，确认这个工具可以在终端里使用。
5. 不要顺手安装其它无关工具。
```

## Git

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 保存修改记录、对比改动、回退错误修改、切换分支、配合远程仓库。 |
| 什么时候需要 | 只要让 Codex 修改代码仓库，就优先准备 Git。 |
| 安装入口 | Git 官网：[git-scm.com/install](https://git-scm.com/install) |
| 与 Codex 的配合方式 | 修改前后可以让 Codex 查看 `git diff`、整理提交说明、解释某次改动、按分支隔离任务。 |
| 使用提醒 | 不熟悉 Git 时，先只掌握 `status`、`diff`、`add`、`commit`。不要随便执行 `reset --hard`、强制覆盖、强推等破坏性命令。 |

Git 是 Codex 工作流里最基础的安全垫。Codex 能改文件，但 Git 负责留下修改边界；一旦改错，可以更容易看清楚哪里变了。

要直接复制提示词，让 AI 帮你初始化仓库、提交、推送或把日记、知识库纳入版本管理，继续看 [Git 妙用](/external/git-practical)。

## Node.js / npm

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 安装 Codex CLI、运行 `npx`、安装前端项目依赖、运行很多 JavaScript / TypeScript 工具。 |
| 什么时候需要 | 用 `npm install -g @openai/codex` 安装 Codex、运行 VitePress / Next.js / React 项目、安装 Superpowers 或部分 MCP 工具时。 |
| 安装入口 | Node.js 官网：[nodejs.org/en/download](https://nodejs.org/en/download)，npm 文档：[docs.npmjs.com](https://docs.npmjs.com/) |
| 与 Codex 的配合方式 | Codex 可以读取 `package.json`，帮你安装依赖、运行脚本、排查 npm 报错、解释前端构建失败。 |
| 使用提醒 | 新手优先安装 LTS 版本。macOS / Linux 遇到全局安装权限问题时，优先查看 npm 官方的版本管理器说明，不要直接用 `sudo npm install -g` 反复尝试。 |

只使用 Codex 官方安装脚本时，可能不需要先处理 npm；用 npm 安装 Codex 或周边工具时，Node.js / npm 就是基础依赖。

## ripgrep

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 在大型项目里快速搜索函数名、配置项、报错文本、接口路径。 |
| 什么时候需要 | 项目文件多、普通编辑器搜索太慢、需要让 Codex 快速定位上下文时。 |
| 安装入口 | GitHub 仓库与 release：[github.com/BurntSushi/ripgrep](https://github.com/BurntSushi/ripgrep) |
| 与 Codex 的配合方式 | 可以让 Codex 用 `rg "关键词"` 搜索代码，再根据结果决定读哪些文件，减少无关文件读取。 |
| 使用提醒 | ripgrep 默认会遵守 `.gitignore`，跳过隐藏目录和二进制文件。搜索隐藏文件或被忽略文件时，要明确告诉 Codex。 |

ripgrep 的命令名是 `rg`。它不是必须安装，但对中大型仓库很有价值，尤其适合让 Codex 先定位再修改。

## GitHub CLI

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 在终端查看 issue、PR、release、GitHub Actions 状态，创建 PR 或切换 PR 分支。 |
| 什么时候需要 | 项目托管在 GitHub，且经常需要 Codex 帮你看 PR、整理提交、检查 CI 或处理 issue。 |
| 安装入口 | GitHub CLI 官网：[cli.github.com](https://cli.github.com/)，手册：[cli.github.com/manual](https://cli.github.com/manual) |
| 与 Codex 的配合方式 | Codex 可以调用 `gh pr view`、`gh pr diff`、`gh issue view`、`gh run list` 等命令，减少在浏览器和终端之间切换。 |
| 使用提醒 | 安装后需要 `gh auth login` 登录。企业 GitHub 或私有仓库要控制权限范围，避免把 token 暴露到日志或文档里。 |

只在本地写小项目时，GitHub CLI 可以暂时不装。它的价值主要出现在 PR、issue、CI 和远程协作场景。

## MCPHub

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 集中管理多个 MCP 服务，把不同工具按统一端点、分组或单服务暴露给 AI 客户端。 |
| 什么时候需要 | 已经在使用多个 MCP 服务，例如搜索、数据库、文件系统、浏览器、内部工具，并且手动管理配置开始变乱。 |
| 安装入口 | 官网文档：[docs.mcphub.app](https://docs.mcphub.app)，GitHub 仓库：[github.com/samanhappy/mcphub](https://github.com/samanhappy/mcphub) |
| 与 Codex 的配合方式 | 可以把常用 MCP 服务集中起来，再让支持 MCP 的客户端连接统一地址，减少每个客户端重复配置工具。 |
| 使用提醒 | 这属于进阶配置，不是 Codex 入门必需。MCP 端点涉及外部工具调用，必须控制鉴权、网络暴露和敏感数据权限。 |

MCP 的目标是让大模型应用以标准方式连接外部数据和工具。MCPHub 适合“工具已经多起来”的阶段；目前只用 Codex 写代码时，先不用搭。

## Superpowers for Codex

| 项目 | 说明 |
| --- | --- |
| 适合解决什么问题 | 给 Codex 增加更明确的工作流程，例如先讨论需求、再写计划、再实现、再验证。 |
| 什么时候需要 | 已经频繁让 Codex 做中等复杂度开发任务，发现它容易跳过设计、测试或验证步骤。 |
| 安装入口 | 进入 Codex 的 `/plugins` 搜索 `superpowers` |
| 与 Codex 的配合方式 | CLI 里用 `/brainstorming`、`/writing-plans`、`/systematic-debugging` 等命令触发；ChatGPT 桌面应用中的 Codex 和 VS Code 插件里用 `$brainstorming`、`$writing-plans` 等 Skill 触发；也可以用中文说“用 Superpowers 的方法解决当前问题”。 |
| 使用提醒 | Superpowers 是流程约束工具，不是模型能力开关。第一次配置 Codex 时先不要同时安装，避免把简单任务变复杂。 |

Superpowers 更像一套工作纪律，不是模型能力开关。它适合已经知道自己需要“流程约束”的用户；只是让 Codex 改小文件、写简单脚本时，先不用装。
