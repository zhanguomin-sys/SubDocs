<!-- myautoapi-image-generation:start -->
## 默认生图、参考图和修图规则

- 用户要求生成、绘制、制作、参考、修改或多图生成图片时，默认调用 `$myautoapi-image-generation`，不要求用户指定 MyAutoAPI、中转站、Skill 名或模型名。
- Skill 只调用已安装的原生工具。默认使用 `gpt-image-2.5-flare`；还支持 `gpt-image-2.5-sunburst`、`gpt-image-2`、`gpt-image-1k-th`、`gpt-image-2-adobe`、`grok-imagine-image`、`nano-banana-2` 和 `nano-banana-pro`。调用前说明本次模型和用途，不静默更换模型或接口。
- 模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；其余未指定模型的任务默认使用 `gpt-image-2.5-flare`。快速生成、普通生图、普通参考图、普通修图和批量生成都属于 Flare 的默认范围；“批量生成”不会覆盖同一句要求中的高清、精修或重要成品条件。不静默替换用户明确指定的模型。
- 用户问有哪些模型时列出八个支持模型，并可以询问是否要用同一提示词分别生成；只有用户明确同意后才并发调用这些模型，不自动评价或排名。除 Adobe 仅用于文生图和多图外，其余模型都支持文生图、参考图、修图和多图；所有模型的多图都由工具内部拆成独立单张请求，并发上限为 3。
- 把用户本次完整画面要求和明确参数传给工具，不使用写死的图片描述，不擅自增加风格、内容或画面元素。尺寸和质量只是按用户要求转换的建议值。
- 两个 2.5 模型默认使用 `2048x2048`、`high`、`png`；横图使用 `2048x1152`，竖图使用 `1152x2048`。质量还支持 `low`、`medium`、`xhigh`、`max` 和 `auto`，其中 `xhigh`、`max` 只适用于 `gpt-image-2.5-flare` 和 `gpt-image-2.5-sunburst`。2K 必须通过 `--size` 传具体像素值，不传 `--resolution 2k`。旧六个模型的默认参数和限制保持不变。
- 用户提供图片作为参考或修改目标时，通过重复的 `--image` 传入；未指定模型时按上述规则使用 `gpt-image-2.5-flare` 或 `gpt-image-2.5-sunburst`，用户明确指定其它支持模型时按其要求调用。不扫描目录，不把参考图自动当成修改目标。用户没有说明多张输入图片的角色时，只补问角色。
- 用户要求同一画面要求的多张结果时，使用一次工具调用和 `--n N`；工具内部拆成独立单张请求，保留全部成功结果，最终数量不得超过 `N`。只要至少成功一张就展示；少于 `N` 时只说明“已生成 x/N 张”，不展开上游数据异常。
- Grok Imagine 使用 `--resolution 1k|2k` 和 `--aspect-ratio 1:1|16:9|9:16|4:3|3:4|3:2|2:3`；质量可以使用 `low`、`medium` 或 `high`。工具按响应中的真实图片格式保存。
- Nano Banana 使用 `--resolution` 和 `--aspect-ratio`，不使用 `--size` 或 `--quality`。`nano-banana-2` 本轮已验证 `512|1k|2k|4k`，以及 `1:1`、`3:2`、`2:3`、`3:4`、`1:4`、`4:1`、`4:3`、`4:5`、`5:4`、`1:8`、`8:1`、`9:16`、`16:9`、`21:9`、`9:21`；还验证了 PNG/JPEG/WebP 各 1 张参考图，以及 2 张和 4 张不同参考图。`nano-banana-pro` 的本地参数沿用 `1k|2k|4k`，本轮不把 Nano Banana 2 的远端验证结果套用到 Pro。参考图本地校验上限为 14 个 `--image`，但 14 张尚未形成远端支持结论。
- 已打开项目或工作区时把图片保存到项目根目录的 `images`；没有项目或工作区时保存到桌面的 `images`。桌面写入超出沙箱时，只为本次工具命令申请最小写入权限；用户不批准就停止。不把图片写入 `$CODEX_HOME`。
- `images` 目录只保存最终图片，不创建图片专属子目录；使用简短准确的图片名称，重名时不覆盖旧图；修图不覆盖输入图片，多图使用编号文件名。
- 生图工具必须访问固定 HTTPS API；不要先在无网络权限的沙箱中试跑。当前权限不允许联网或读取输入图片时，为已安装原生工具的 `generate` 或 `edit` 子命令申请可复用的最小许可；许可范围不能扩大到整个 PowerShell、Bash 或其它命令，不要求用户开启“完全访问”，也不修改 `config.toml`。能否在本会话或以后重复使用由宿主权限界面和用户决定；只支持单次批准时申请单次，用户拒绝就停止。
- 单模型普通任务只运行一次 Skill 给出的前台工具调用。用户明确要求两个或更多不同模型分别生成时，为每个模型构造一个独立命令，并使用宿主的并发工具调用能力在同一批中同时提交；每个模型只运行一次。每个命令仍是可等待、可收集 stdout 的独立前台调用，不得因需要独立调用而改成顺序执行，也不得用 shell 后台符号、`Start-Job`、脱离会话任务或进程轮询模拟并发。
- 单模型时等待同一个调用退出，多模型时等待整批调用全部退出；保存模型、工具调用和 stdout 结果的对应关系，不搜索工具，不预建或探测输出目录，不另查进程。任一调用未退出时不得结束当前轮次，不得声称图片稍后会自动显示。
- Linux 或 macOS Code Mode 使用嵌套 `exec_command()` 调用工具时，外层代码首行设置 `@exec` 的 `yield_time_ms: 120000`；外层返回 `Script running with cell ID ...` 时立即对同一个 `cell_id` 调用 `functions.wait`，不等待用户再次发消息或催促。调用前分别保存每个命令的 `output_dir`、`output_name`、请求数量、模型和开始时间；必须用 `text(...)` 显式转发嵌套返回对象中的退出码、`output` 和 `session_id`。返回 `session_id` 后每 5 秒收取同一个 `session_id`；每累计 30 秒仍未退出时用 Code Mode 的 `notify(...)` 发出中文状态并继续收取。Windows PowerShell 调用流程不变。
- stdout 的 `requested_count` 是请求数量，`output_count` 是实际成功数量；单张使用 `output_file`，多张使用 `output_files`。正常取得 stdout 时不扫描输出目录，也不调用 `view_image` 或其它工具重新读取图片。取得路径后立即使用绝对路径 Markdown 图片语法直接展示，例如 `![生成结果](/absolute/path/image.png)`；最终回复必须再次嵌入全部成功图片，嵌入数量与 `output_count` 一致。多模型结果按保存的对应关系处理，保留准确模型 ID 和 `第 i/output_count 张`。只有 Linux 或 macOS Code Mode 的嵌套命令已经结束、外层没有可解析 JSON 时，才对本次已知的 `output_dir` 做一次只读恢复核查；不能只取最近 N 张，不重新生图、不更新或重装工具。
- 工具完全失败时只显示 `error` 的简短提示，提醒用户到后台查看是否有生成记录，或让 AI 修复生图工具。只有修复任务才使用 `diagnostic` 字段；不自动重试或换模型。
- Windows PowerShell 不把用户画面描述直接拼进 `powershell -Command` 的双引号命令文本；先用单引号 here-string 保存完整描述，再把变量作为 `--prompt` 的独立参数传给工具。
- 调用前提示、等待状态、成功结果和失败说明全部使用中文，不输出英文进度说明。
- 不换模型、不伪造图片。
- API Key 由工具直接从 `$CODEX_HOME/tools/myautoapi-imagines-tool/api-key` 读取。AI 不打开、读取、复制、展示、记录、扫描或提交 `auth.json`；旧版 `auth.json` 读取仅作为工具内部回退模式，不在 Skill 中主动切换。
- 只有用户明确说“更新生图工具”“升级生图工具”或要求重装时才执行更新。用户要求修复时先捕捉本机真实原因，不把更新当成修复。Windows 使用 `irm https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1 | iex`，Linux x64、Apple Silicon 或 Intel Mac 使用 `curl -fsSL https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh | bash`；用户明确要求更新后，给出当前系统的安装命令，让用户在终端执行并粘贴专门画图分组 Key，不在聊天中收集 Key。
- 更新入口会按当前平台选择一个程序、校验 SHA-256；每次执行都输入并覆盖 `api-key`，同版本只跳过下载；不创建后台服务、计划任务、开机启动，也不在正常生图时检查更新。
<!-- myautoapi-image-generation:end -->
