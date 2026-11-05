---
sidebarTitle: Skills
---

# Codex Skills

Skill 是给 Codex 保存的固定工作方法：经常重复的写作、审查、调试、批量处理流程做成 Skill 后，以后直接调用，不用每次重新交代。Skill 不用自己写，让 Codex 创建、修改和测试。

## Skill 放在哪里

每个 Skill 是 `.codex/skills` 下的一个文件夹，例如 `.codex/skills/docs-writer`。

| 系统 | Skill 目录 |
| --- | --- |
| Windows | `C:\Users\你的用户名\.codex\skills` |
| Mac / Linux / WSL | `~/.codex/skills` |

设置过 `CODEX_HOME` 的，放在 `CODEX_HOME/skills` 下。

以后还会反复用的工作方法才做成 Skill。只是这一次的要求，直接写在对话里；只属于当前项目的规则，写进项目的 `AGENTS.md`。

## 让 Codex 创建 Skill

把提示词发给 Codex，按需要改名称和要求：

```text
请帮我创建一个 Codex Skill。

目标：把我经常重复的文档写作流程沉淀成 Skill。
保存位置：当前用户的 .codex/skills 目录。
Skill 名称：docs-writer。

要求：
1. 先确认 .codex/skills 目录是否存在，不存在就创建。
2. 在 .codex/skills 下创建 docs-writer 文件夹。
3. 只创建必要文件，不要创建 README、安装说明、更新日志等无关文件。
4. Skill 内容用中文写清楚：什么时候使用、工作步骤、输出要求、验证方式。
5. 让 Codex 以后写文档时先读附近已有文档，保持标题、语气、链接和步骤风格一致。
6. 完成后告诉我 Skill 放在哪个目录，以及在 Codex CLI 和 ChatGPT 桌面应用中的 Codex 里怎么调用。
```

<details>
<summary>更多示例：代码审查、API 排错、批量处理</summary>

代码审查：

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

API 排错：

```text
请帮我创建一个 API 排错用的 Codex Skill。

保存位置：当前用户的 .codex/skills 目录。
Skill 名称：api-debugging。

要求：
1. Skill 用来排查 HTTP API、OpenAI-compatible 接入、Responses 接入和中转站调用问题。
2. 排查顺序固定为：请求地址、请求方法、认证、请求体、模型名、后台记录、客户端报错、本地网络。
3. 遇到 401、403、404、429、499、502、503、504 时，先解释含义，再给检查动作。
4. 涉及日志或请求头时，必须提醒先隐藏 API Key、Token、Cookie 等敏感信息。
5. 只保留排错必须用到的内容，不要写成长篇百科。
6. 创建完成后，告诉我这个 Skill 的调用方法。
```

批量处理：

```text
请帮我创建一个批量处理用的 Codex Skill。

保存位置：当前用户的 .codex/skills 目录。
Skill 名称：batch-runner。

要求：
1. Skill 用来处理批量文件、批量提示词、批量结果记录和失败重试。
2. 开始前先读取当前目录，识别输入文件、输出目录和已有结果记录。
3. 不要要求我手动粘贴已经在当前目录里的文件内容。
4. 批量任务必须限制范围，说明会改哪些文件、不会改哪些文件。
5. 处理结果要记录成功项、失败项、失败原因和下一轮建议。
6. 需要自动化处理时，只做最小必要实现，不要搭建网页、数据库、队列或复杂工程。
```

</details>

## 调用 Skill

Codex 会自动发现新建和修改的 Skill，创建后直接调用；列表里没出现时重启 Codex。

| 入口 | 调用方式 |
| --- | --- |
| Codex CLI | 输入 `/skills` 从列表选择，或直接说“使用 docs-writer Skill 写这篇文档” |
| ChatGPT 桌面应用中的 Codex、VS Code 插件 | 输入 `$` 从列表选择，例如 `$docs-writer` |

```text
$docs-writer
请重写这篇快速开始文档，面向第一次使用的用户，删掉重复解释。
```

## 测试 Skill

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

## 修改和更新 Skill

Skill 啰嗦、误触发、漏步骤或流程变了，都让 Codex 改，不要自己去文件里找格式问题：

```text
请帮我修改已有 Codex Skill。

Skill 位置：当前用户的 .codex/skills/docs-writer

修改目标：
1. 删除啰嗦解释，只保留 Codex 执行任务必须知道的步骤。
2. 触发说明写清楚，避免所有写作任务都误触发。
3. 工作流程用中文写，除文件名、命令名、配置字段外不要夹杂英文。
4. 检查是否有 README、CHANGELOG、安装说明等无关文件，有就删除。
5. 修改后用一个真实文档任务测试调用效果。
```

给其它 Skill 加流程时，同时改“Skill 位置”和“修改目标”。例如更新 API 排错 Skill：位置改成 `.codex/skills/api-debugging`，修改目标写“增加 499 context canceled 的排查流程；增加后台有记录和没有记录时的不同检查路径”。
