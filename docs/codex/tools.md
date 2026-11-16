---
sidebarTitle: 周边工具
---

# Codex 周边工具

多数用户装 Git、Node.js、ripgrep 就够了；经常处理 GitHub、MCP 或复杂开发流程时，再装后面三个。

| 工具 | 什么时候装 | 和 Codex 怎么配合 | 装完验证 |
| --- | --- | --- | --- |
| [Git](https://git-scm.com/install) | 让 Codex 改代码仓库之前 | 让 Codex 看 `git diff`、写提交说明、按分支隔离任务，改错能回退 | `git --version` |
| [Node.js](https://nodejs.org/en/download) / npm | 用 npm 安装 Codex、用 `npx` 运行工具、运行前端项目、安装 MCP 工具时 | Codex 读 `package.json`，安装依赖、运行脚本、排查构建报错 | `node -v`、`npm -v` |
| [ripgrep](https://github.com/BurntSushi/ripgrep) | 项目文件多，想让 Codex 快速搜索时 | Codex 用 `rg "关键词"` 先定位再读文件 | `rg --version` |
| [GitHub CLI](https://cli.github.com/) | 经常处理 issue、PR、release、CI 时 | Codex 调用 `gh pr view`、`gh pr diff`、`gh issue view`、`gh run list` | `gh auth login` 登录成功 |
| [MCPHub](https://docs.mcphub.app) | 已经在用多个 MCP 服务，配置开始变乱时 | 把多个 MCP 服务集中到统一地址，客户端只连一次 | 按 MCPHub 文档完成配置 |
| Superpowers | 经常做复杂功能、排 bug、写测试时 | 让 Codex 按“讨论需求 → 写计划 → 实现 → 验证”的流程工作 | 在 Codex 的 `/plugins` 搜索 `superpowers` 安装后能调用 |

## 让 AI 帮你安装

一次只装一个，把【工具名】换成要装的工具：

```text
请帮我在当前电脑安装【工具名】。

要求：
1. 先判断当前系统，检查这个工具是否已经安装；已安装就告诉我版本，不要重复安装。
2. 没安装时，用当前系统常见的安装方式。
3. 需要管理员权限、下载安装包、改系统环境变量时，先说明要做什么，等我确认。
4. 安装完成后运行验证命令，确认能在终端里使用。
5. 不要顺手安装其它工具。
```

## 使用提醒

- **Git**：新手先掌握 `status`、`diff`、`add`、`commit`。不要随便让 Codex 执行 `reset --hard`、强制覆盖、强推。提示词用法看 [Git 妙用](/codex/git)。
- **Node.js**：安装 LTS 版本。Mac 和 Linux 遇到全局安装权限报错时，按 [npm 官方说明](https://docs.npmjs.com/) 改用版本管理器，不要反复 `sudo npm install -g`。
- **ripgrep**：命令名是 `rg`，默认跳过 `.gitignore` 里的文件、隐藏目录和二进制文件；要搜这些文件时明确告诉 Codex。
- **GitHub CLI**：私有仓库和企业 GitHub 控制好登录权限，不要把 token 写进日志或文档。
- **MCPHub**：MCP 端点会调用外部工具，控制好鉴权、网络暴露和敏感数据权限。
- **Superpowers**：CLI 里用 `/brainstorming`、`/writing-plans`、`/systematic-debugging` 等命令；ChatGPT 桌面应用和 VS Code 插件里用 `$brainstorming`、`$writing-plans`；也可以直接说“用 Superpowers 的方法解决当前问题”。只改小文件、写简单脚本时不用装。详见 [Superpowers](/codex/superpowers)。
