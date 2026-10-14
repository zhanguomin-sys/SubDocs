# Codex Skills

Skill 是给 Codex 准备的固定工作方法，适合把经常重复的写作、审查、调试、排错、批量处理流程保存下来，以后直接调用。

用户不需要手写 Skill 文件。正常做法是让 Codex 帮你创建、修改和测试 Skill；你只需要说清楚要沉淀什么工作流，并确认文件放在正确位置。

## Skill 放在哪里

每个 Skill 是 `.codex/skills` 下面的一个文件夹。

| 系统 | Skill 目录 |
| --- | --- |
| Windows | `C:\Users\你的用户名\.codex\skills` |
| macOS / Linux / WSL | `~/.codex/skills` |

一个 Skill 单独放一个文件夹，例如：

```text
~/.codex/skills/docs-writer
~/.codex/skills/api-debugging
```

设置过 `CODEX_HOME` 的用户，把 Skill 放到对应的 `CODEX_HOME/skills` 下面。普通用户没有设置过 `CODEX_HOME`，就按上表位置放。

## 什么时候做成 Skill

| 场景 | 是否适合 Skill |
| --- | --- |
| 每次都要重复提醒 Codex 同一套步骤 | 适合 |
| 写文档、审查代码、排查 API、批量处理文件都有固定流程 | 适合 |
| 只是这一次任务的要求 | 不适合，直接写在当前对话 |
| 只是当前项目规则 | 不适合，写进项目的 `AGENTS.md` |

判断一句话：以后还会反复用的工作方法，才做成 Skill。

## 让 AI 创建 Skill

把下面提示词复制给 Codex。它会在你的 `.codex/skills` 目录下创建 Skill，并处理必要文件。

```text
请帮我创建一个 Codex Skill。

目标：把我经常重复的文档写作流程沉淀成 Skill。
保存位置：当前用户的 .codex/skills 目录。
Skill 名称：docs-writer。

要求：
1. 先确认 .codex/skills 目录是否存在，不存在就创建。
2. 在 .codex/skills 下创建 docs-writer 文件夹。
3. 创建必要文件，不要创建 README、安装说明、更新日志等无关文件。
4. Skill 内容用中文写清楚：什么时候使用、工作步骤、输出要求、验证方式。
5. 让 Codex 以后写文档时先读附近已有文档，保持标题、语气、链接和步骤风格一致。
6. 完成后告诉我 Skill 放在哪个目录，以及在 Codex CLI 和 ChatGPT 桌面应用中的 Codex 里怎么调用。
```

## 创建代码审查 Skill

适合经常让 Codex 检查代码、文档、配置改动的用户。

```text
请帮我创建一个代码审查用的 Codex Skill。

保存位置：当前用户的 .codex/skills 目录。
Skill 名称：code-review-style。

要求：
1. Skill 用来审查当前改动、补丁、PR 或未提交文件。
2. 审查时优先找 bug、回归风险、安全问题、数据丢失、缺少验证，不要泛泛夸改动。
3. 输出时先列问题，按严重程度排序，并带上文件路径和行号。
4. 没有发现问题时，也要说明剩余风险和未运行的验证。
5. 只创建必要文件，不要创建无关说明文档。
6. 创建完成后，用一个模拟审查任务测试这个 Skill 是否能被调用。
```

## 创建 API 排错 Skill

适合经常排查 401、403、429、超时、流式输出中断、模型不可用等 API 问题的用户。

```text
请帮我创建一个 API 排错用的 Codex Skill。

保存位置：当前用户的 .codex/skills 目录。
Skill 名称：api-debugging。

要求：
1. Skill 用来排查 HTTP API、OpenAI-compatible 接入、Responses 接入和中转站调用问题。
2. 排查顺序固定为：请求地址、请求方法、认证、请求体、模型名、后台记录、客户端报错、本地网络。
3. 遇到 401、403、404、429、499、502、503、504 时，先解释含义，再给检查动作。
4. 涉及日志或请求头时，必须提醒先隐藏 API Key、Token、Cookie 等敏感信息。
5. 只保留排错必须用到的内容，不要把正文写成长篇百科。
6. 创建完成后，告诉我这个 Skill 的调用方法。
```

## 创建批量处理 Skill

适合经常让 Codex 批量改文件、批量生成图片提示词、批量整理结果的用户。

```text
请帮我创建一个批量处理用的 Codex Skill。

保存位置：当前用户的 .codex/skills 目录。
Skill 名称：batch-runner。

要求：
1. Skill 用来处理批量文件、批量提示词、批量结果记录和失败重试。
2. 开始前先让 Codex 读取当前目录，识别输入文件、输出目录和已有结果记录。
3. 不要要求我手动粘贴已经在当前目录里的文件内容。
4. 批量任务必须限制范围，说明会改哪些文件、不会改哪些文件。
5. 处理结果要记录成功项、失败项、失败原因和下一轮建议。
6. 需要自动化处理时，只做最小必要实现，不要搭建网页、数据库、队列或复杂工程。
```

## 修改已有 Skill

后续发现 Skill 太啰嗦、误触发、漏步骤，也让 AI 改。

```text
请帮我修改已有 Codex Skill。

Skill 位置：~/.codex/skills/docs-writer

修改目标：
1. 删除啰嗦解释，只保留 Codex 执行任务必须知道的步骤。
2. 触发说明要写清楚，避免所有写作任务都误触发。
3. 工作流程用中文写，除文件名、命令名、配置字段外不要夹杂英文。
4. 检查是否有 README、CHANGELOG、安装说明等无关文件，有就删除。
5. 修改后用一个真实文档任务测试调用效果。
```

Windows 用户把路径改成：

```text
C:\Users\你的用户名\.codex\skills\docs-writer
```

## 更新 Skill

工作流程变了，继续让 AI 更新。

```text
请帮我更新这个 Codex Skill。

Skill 位置：~/.codex/skills/api-debugging

新增要求：
1. 增加 499 context canceled 的排查流程。
2. 增加后台有记录和没有记录时的不同检查路径。
3. 删除重复解释，保持正文短。
4. 更新后检查 Skill 是否还能正常调用。
```

## 创建后检查什么

用户不用打开 Skill 文件，只看三件事：

| 检查项 | 正确状态 |
| --- | --- |
| 文件夹位置 | 放在 `.codex/skills` 下面 |
| Skill 名称 | 文件夹名和调用名一致，例如 `docs-writer` |
| 无关文件 | 没有 README、安装说明、更新日志等给人看的多余文件 |

发现不对时，把问题发给 Codex，让它修改这个 Skill。

## 调用 Skill

创建后要新开一个 Codex 会话，或者重启对应入口，让 Codex 重新加载 Skill。

| 入口 | 调用方式 |
| --- | --- |
| Codex CLI | 在输入框里输入 `/skills`，从列表选择；也可以直接说“使用 docs-writer Skill 写这篇文档” |
| ChatGPT 桌面应用中的 Codex | 在输入框里输入 `$`，从 Skill 列表选择；也可以直接点对应 Skill |
| VS Code 插件 | 在输入框里输入 `$`，和 ChatGPT 桌面应用中的 Codex 类似 |

CLI 示例：

```text
使用 docs-writer Skill 重写当前文档，保持现有文档风格，完成后检查链接和命令。
```

App 示例：

```text
$docs-writer
请重写这篇快速开始文档，面向第一次使用的用户，删掉重复解释。
```

排错示例：

```text
使用 api-debugging Skill 排查这个 403 报错。请先看请求、模型名、API Key 分组和后台记录，不要直接猜原因。
```

## 创建后怎么确认能用

让 AI 做一次真实任务测试：

```text
请测试刚创建的 docs-writer Skill。

用它处理当前目录里一篇文档，只做小改动。
测试时观察：
1. 是否能正确调用 Skill。
2. 是否会先读附近文档风格。
3. 是否按 Skill 要求输出。
4. 是否没有乱改无关文件。
测试完成后告诉我结果和需要调整的地方。
```

Skill 不好用时，直接让 AI 改触发说明、删掉多余规则、补上遗漏步骤。不要自己手动在文件里找格式问题。

Skill 的价值不是多一个配置文件，而是让 Codex 少忘步骤、少重复解释、少靠临场发挥。
