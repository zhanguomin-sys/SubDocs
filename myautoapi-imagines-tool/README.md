# myautoapi-imagines-tool

这是一个给 Codex 使用的本地图片生成工具。它适合需要直接用自然语言生图、又希望请求统一经过 MyAutoAPI 中转站的用户；安装后，普通任务默认调用 `gpt-image-2.5-flare` 生成 2K 图片，明确要求高清、精修或重要成品时调用 `gpt-image-2.5-sunburst`，成功后把本地图片显示在当前对话中。

当前源码版本是 `V0.8.1`。每次执行安装或更新命令都要求粘贴专门画图分组 Key，并覆盖工具目录中的旧 Key；同版本只跳过下载，仍继续配置。图片模型沿用 V0.8.0 的能力。

`V0.8.1` 发布包已放入文档站的 `docs/public/install/myautoapi-imagines-tool/`，随文档站部署上线。这里的更新是用户明确对 Codex 说“更新生图工具”后，由 Codex 在当前任务中完成检查、下载、安装和验证；修复问题时先捕捉本机真实原因，不把更新当成修复。

## 一键安装

运行对应系统的一键安装命令。每次运行时，安装器都会要求粘贴“专门画图分组”的 Key，并覆盖工具目录中已有的 Key。输入内容会正常显示在终端中；安装器不会把 Key 再打印到后续提示、日志或安装清单。

Linux x64、Apple Silicon 或 Intel Mac：

```bash
curl -fsSL https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh | bash
```

Windows PowerShell：

```powershell
irm https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1 | iex
```

V0.8.1 发布资产支持 64 位 x64 Windows、64 位 x64 Linux、Apple Silicon 64 位 Mac 和 Intel 64 位 Mac。公开入口仍只有 Windows `install.ps1` 和 Linux/macOS `install.sh` 两个；Shell 入口自动判断当前系统和架构，只下载对应的一个 `V0.8.1` 原生程序，核对内置 SHA-256 和 ELF 或 Mach-O 架构后运行。Apple Silicon 在 Rosetta 下运行时优先使用 arm64 程序。客户端不需要 Python、Node、Git、SDK 或管理员权限；安装器不继续下载代码，只会写入以下用户目录：

- 工具：`$CODEX_HOME/tools/myautoapi-imagines-tool/`；其中 `api-key` 保存“专门画图分组”的 Key，`key-source` 保存当前 Key 来源；未设置 `CODEX_HOME` 时使用 `~/.codex/tools/myautoapi-imagines-tool/`。
- Skill：`$CODEX_HOME/skills/myautoapi-image-generation/`；未设置时使用 `~/.codex/skills/myautoapi-image-generation/`。
- 全局默认规则：`$CODEX_HOME/AGENTS.md`；未设置时使用 `~/.codex/AGENTS.md`。

安装器不会修改当前项目的 `AGENTS.md`，不会写入或修改 `$CODEX_HOME/auth.json`，也不会写入 `.agents/skills/` 或 `skill-packs/`。已有 V0.8.1 且架构、SHA-256 匹配时直接复用程序，继续输入并覆盖画图 Key。V0.2 至 V0.8.0 原生安装可以升级到 V0.8.1；架构或 SHA-256 不匹配时重新下载正式程序。损坏的托管文件会修复。目标目录无法确认归属、版本不受支持、规则标记破损或存在非空 `AGENTS.override.md` 时会停止，不覆盖用户文件。

## 第一次使用

安装完成后重新打开 Codex，或开始一个新任务，再发送一条短句，例如：

```text
画一张白底红苹果
```

不需要安装或手工运行 Python，不需要在提示词里写 MyAutoAPI。普通任务默认使用 `gpt-image-2.5-flare`，明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；也可以明确指定原有六个模型。图片生成完全失败时，Codex 会提醒客户到后台查看是否有生成记录，或让 AI 修复生图工具；不自动重试或更换模型。

Linux 或 macOS Code Mode 会每 5 秒收取同一个前台会话；进程退出时立即取得结果，不等待用户再次发消息。生成超过 30 秒时，Codex 先报告仍在生成，再继续等待同一个调用。

## 更新工具

稳定的一键安装入口同时作为更新入口。用户只需对 Codex 说：

```text
更新生图工具
```

Codex 给出当前系统的一键安装命令，用户在终端执行并粘贴专门画图分组 Key。每次执行都会覆盖旧 Key，不需要额外参数。输入为空时重新输入；没有输入或保存失败时保留原有 Key。

更新入口只在用户明确要求时联网。它先比较本地版本与在线版本；已经是最新版且本机文件完整时不下载原生程序。发现新版本时只下载当前平台的一个程序，校验 SHA-256 后执行事务更新，每次执行都继续输入并保存画图 Key；失败时恢复原有工具、Skill、规则和 Key。工具不创建后台服务、计划任务或开机启动项，也不在正常生图时检查更新。

## 卸载工具

不提供公网卸载脚本。用户只需对 Codex 说“卸载中转站生图工具”，由 Codex 调用已安装原生程序的 `uninstall` 命令。卸载范围固定为：

- `$CODEX_HOME/tools/myautoapi-imagines-tool/` 内的受管程序、版本和来源标记；默认保留 `api-key`，永久删除前必须单独确认。
- `$CODEX_HOME/skills/myautoapi-image-generation/` 内的本工具 Skill。
- `$CODEX_HOME/AGENTS.md` 中由本工具维护的规则块，不改动用户其它内容。

未设置 `CODEX_HOME` 时，上述位置使用当前用户的 `~/.codex/`。安装到用户机器的全局规则和生图 Skill 不暴露卸载命令；卸载说明只保留在本项目文档中。

## 工具边界

- 固定接口：Images 模型使用 `https://api.myautoapi.com/v1/images/generations` 和 `/v1/images/edits`；Nano Banana 使用 `https://api.myautoapi.com/v1/responses`
- 支持模型：`gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`、`gpt-image-2`、`gpt-image-1k-th`、`gpt-image-2-adobe`、`grok-imagine-image`、`nano-banana-2`、`nano-banana-pro`
- `gpt-image-2.5-flare` 支持文生图、参考图、修图和多图；普通任务默认使用，默认 `2048x2048`、`high`
- `gpt-image-2.5-sunburst` 支持文生图、参考图、修图和多图；明确的高质量与精细编辑任务使用，默认 `2048x2048`、`high`
- `gpt-image-2` 支持文生图、参考图、修图和多图；默认 `1024x1024`、`high`
- `gpt-image-1k-th` 支持文生图、参考图、修图和多图；固定 `1024x1024`，已验证 `low` 和 `high`，多图固定拆成单张请求
- `gpt-image-2-adobe` 开放已验证的 `1024x1536`、`1536x2304`、`2304x3456`、`low` 文生图；多图由工具内部拆成多次单张请求
- `grok-imagine-image` 支持文生图、参考图、修图和多图；支持 `1k`、`2k`，以及 `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3`
- Grok Imagine 质量支持 `low`、`medium`、`high`；未指定时使用模型默认质量
- `nano-banana-2` 支持文生图、参考图、修图和多图；本轮真实验证分辨率 `512`、`1k`、`2k`、`4k`、15 种比例、PNG/JPEG/WebP 各 1 张参考图，以及 2 张和 4 张不同参考图
- `nano-banana-pro` 支持文生图、参考图、修图和多图；本地参数使用 `1k`、`2k` 或 `4k`，本轮不把 Nano Banana 2 的远端验证结果套用到 Pro
- Nano Banana 参考图的本地校验上限仍为 14 个 `--image`；14 张远端请求返回 HTTP 502，尚未形成远端支持结论
- 默认每次请求：`n=1`；同一画面要求通用支持 `n=1` 到 `n=10`，Grok Imagine 支持 `n=1` 到 `n=9`
- 默认参数：`gpt-image-2.5-flare`、`2048x2048`、`high`、`png`
- 质量支持：通用为 `low`、`medium`、`high`、`auto`；两个 2.5 模型另外支持 `xhigh` 和 `max`
- 常用尺寸建议：`1024x1024`、`1536x1024`、`1024x1536`、`2048x2048`、`2048x1152`、`1152x2048`、`3840x2160`、`2160x3840`、`auto`
- 精确尺寸支持 `WIDTHxHEIGHT`：宽高为 16 的倍数，单边不超过 3840，长短边比例不超过 3:1，总像素为 655360 至 8294400
- 支持接口返回的 `b64_json`，也支持 HTTPS 图片 URL 回退；每次 URL 只下载一次
- GPT Image 和 Grok 的参考图、修图通过固定 `/v1/images/edits` 提交；Nano Banana 把文本和参考图一起提交到 `/v1/responses`；输入图片不写入用户输出目录
- 多图在一次工具调用内拆成独立的单张请求，并发上限为 3；任意请求失败时保留其它成功图片，最终数量不超过用户请求的 `n`
- 图片按接口返回的真实格式保存；当前实测结果为 PNG 或 JPEG
- 成功时只在 `images` 下保存最终图片；文件名由 Skill 根据画面主体提供，重名自动追加数字，不覆盖旧图
- stdout 使用 `requested_count` 和 `output_count` 标明请求数量与实际成功数量；单图返回 `output_file`，多图返回 `output_files`；API 原始响应只在内存中处理，不写入用户目录
- 用户明确要求多个不同模型时，Codex 为每个模型提交一个独立前台调用并在同一批中并发启动；工具返回路径后，Codex 不重新读取图片，直接用绝对路径 Markdown 图片语法展示，并在最终回复中再次嵌入全部成功图片
- API Key 默认由工具从 `$CODEX_HOME/tools/myautoapi-imagines-tool/api-key` 读取，调用方不传 `--api-key` 或 `--base-url`。旧版 `auth.json` 读取只保留为工具内部回退模式。

客户端使用预编译原生程序，不安装运行环境或额外依赖。安装器不会强制 `0600` 或重写 Windows ACL；普通文件继承当前操作系统和用户目录的正常默认权限，Linux 和 Mac 工具文件只增加运行程序必需的可执行属性。

## 验证状态

V0.8.1 的 Go 测试和 74 项 Python 测试通过，独立审查未发现阻塞问题。Linux x64 使用真实 `curl | bash`、本地 HTTPS 和控制伪终端验证了 12 个场景，包括重复换 Key、取消输入、缓存损坏、下载失败、V0.8.0 升级和卸载后重装；独立审查再次运行全部 12 项，结果通过。本轮尚未在 Windows、Apple Silicon Mac 和 Intel Mac 实机运行。

V0.1 的 Python 参考实现已完成 `gpt-image-2` 请求、结果保存和无密钥契约测试；V0.3-r5 的原生安装、专门 Key、更新和 Windows 生图流程已经验证。V0.4.0 已真实验证 `gpt-image-2` 的 1K、2K、4K、横图、竖图、方图、自动尺寸和四档质量，并验证 `b64_json` 与 HTTPS URL 两种图片返回。V0.4.1 已真实验证参考图、图片修改和三张多图结果；测试确认三张结果都会逐张进入当前对话显示流程。V0.5.0 已验证 `gpt-image-1k-th` 和 `gpt-image-2-adobe`，V0.6.0 已验证 `grok-imagine-image`。V0.7.0 已真实验证两个 Nano Banana 模型均能返回有效 PNG，工具可完成解析、保存和显示；参考图、多图、嵌套图片解析及逐张展示由本地契约测试覆盖。V0.7.1 的多模型并发和结果绑定由本地 Skill 契约测试覆盖。V0.7.2 的 macOS Code Mode 结果转发与受限目录恢复已在 Linux 使用假工具和临时图片目录模拟验证。V0.7.3 增加 Intel Mac 构建和自动分流。V0.7.4 修复多图独立请求、部分成功保留和直接 Markdown 展示。V0.7.5 已验证 Nano Banana 2 的 512/1K/2K/4K、比例、PNG/JPEG/WebP 各 1 张参考图及 2/4 张不同参考图，并明确 14 张远端请求仍无支持结论；同时修正空参考图提示、V0.7.4 升级和发布测试隔离。V0.7.6 在 Linux 实测 `gpt-image-2` 完成后可立即取得结果，并用本地契约覆盖 5 秒连续收取和 30 秒中文状态通知。V0.8.0 已从子项目源码真实验证 Flare 默认 2K 文生图和 Sunburst 默认 2K 修图，两者均返回实际 `2048x2048` PNG；六档质量和旧模型不变由本地契约覆盖。

原有 `V0.2-r1` 依赖客户端 Python，不能作为通用一键安装发布。`V0.2-r2` 首次改为原生程序，`V0.2-r3` 收缩为 Windows x64 和 Apple Silicon Mac；`V0.3-r3` 在这两个目标上加入专门画图分组 Key，并把生图输出移出 Codex Home；`V0.3-r4` 加入用户主动触发的更新入口；`V0.3-r5` 收敛为两个公开入口并修正 Windows 提示词传参；`V0.7.3` 增加 Intel Mac 原生程序，Mac 仍只保留一条自动分流的安装入口。Windows 已完成真实安装和生图验证，Apple Silicon 与 Intel Mac 实机验证仍待分别完成。

## 发布维护

本子项目使用独立 Git 仓库，父级 `subdocs` 仓库不会自动收录这里的源码或 `dist/`。每次修改安装器、正式工具、Skill、清单或 README 后，维护者必须完成以下动作：

1. 运行全部无密钥测试和 Skill 校验。
2. 运行 `python3 release/build_release.py`，重新生成 Windows x64、Linux x64、Apple Silicon Mac、Intel Mac 四个程序和 `SHA256SUMS`。
3. 把新 SHA-256 同步写入 `release/` 下 Windows、Linux/macOS 两份公开安装脚本，重新运行发布契约测试。
4. 手动把两份公开安装脚本、四个平台程序和 `SHA256SUMS` 迁移到父仓库 `docs/public/install/myautoapi-imagines-tool/` 对应版本目录。
5. 在父仓库完成站点构建、公网哈希核对和隔离 `CODEX_HOME` 安装测试后再推送。

发布脚本不得自动修改、提交或推送父仓库。手动迁移是两个仓库之间固定的发布边界。

`V0.3-r5` 是旧版发布编号。V0.4 开始使用 `V0.4.0`、`V0.4.1` 这样的版本号，不再为普通发布增加 `rN`。任何受管文件变化都必须重新构建原生程序、SHA-256 和两份公开安装脚本。

## 项目目录

```text
myautoapi-imagines-tool/
  *.go              原生安装、卸载和生图程序源码
  installer/        旧 Python 安装兼容代码和全局规则资源
  release/          原生发布构建脚本和公网一键安装入口
  tool/             V0.1 Python 参考实现
  skill-template/  安装包内的“中转站生图” Skill
  tests/            无密钥契约测试和安装器测试
  test-records/     脱敏后的真实测试记录
  skill-evaluations/ Skill 基线、前向和安装流程演练
  plans/            版本实施计划
```

父项目的 `subdocs/docs/`、站点页面、导航和其它多模型文档不属于本子项目的 V0.8.1 发布资产范围；生图分组方法页需要随已支持模型同步维护。
