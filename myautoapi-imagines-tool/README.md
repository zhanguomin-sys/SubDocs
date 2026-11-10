# myautoapi-imagines-tool

这是供 Codex 使用的 MyAutoAPI 原生图片工具，支持文生图、参考图、修图、多图和两个 2.6 模型的 GIF 文生图。普通任务默认使用 `gpt-image-2.6-flare`，明确要求高清、精修或重要成品时使用 `gpt-image-2.6-sunburst`。

## 当前发布来源

当前正式制品是对方仓库 [youdaowudao/subdocs](https://github.com/youdaowudao/subdocs) 固定提交 `c28a91174db4f065ff4c4121e348f20afebda85c` 发布的 V0.9.2 原生程序，经 MyAutoAPI 品牌和域名迁移后发布。对方仓库未提供该工具的当前源码；本目录保留的 Go/Python 源码是此前的独立移植实现，供历史参考，**不是当前正式制品的构建来源**。

迁移脚本为 `release/port_upstream.py`。它先核对四个平台的固定上游 SHA-256，再等长替换 `usegoodai`、`UseGoodAI`、`USEGOODAI`，同时迁移工具名、安装目录、Skill、全局规则和固定网络入口。Apple Silicon 程序原有的 ad-hoc 签名分页哈希会重新计算，文件结构和长度保持不变。`release/upstream-port.json` 记录来源提交、输入和输出哈希、替换数量及签名分页数量，便于复现和核对。

固定 API 已迁移为 `https://api.myautoapi.com`；原生更新清单为 `https://docs.myautoapi.com/install/myautoapi-imagines-tool/latest.json`。工具仍可能下载接口返回的外部 HTTPS 图片 URL。版本号仍为 `V0.9.2`，不能仅凭版本号判断是否已经取得当前制品，还应核对 SHA-256。

## 安装与更新

公开安装入口保持不带版本号：

- Windows：`https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1`
- Linux/macOS：`https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh`（自动选择 x64 Linux、Apple Silicon Mac 或 Intel Mac 的原生程序）

Windows 的 `.exe` 只适用于 Windows。macOS 不运行 `.exe`，安装器会根据 `uname` 和 Rosetta 状态选择 `darwin-arm64`（Apple 芯片）或 `darwin-amd64`（Intel 芯片），两种程序都从无版本号的 `current/` 发布目录下载并校验 SHA-256。

入口按当前平台只下载一个原生程序，并校验 SHA-256。首次安装或用户明确要求重新配置画图 Key 时，入口先显示“请输入专门画图分组 Key”，读取非空的一行后，通过标准输入交给 `install --key-stdin` 保存。该原生选项本身不显示输入提示，不能直接连接交互终端，否则会停在“正在检查安装位置……”等待输入。已有正确制品时可以跳过下载，继续显式配置 Key。不要在聊天、命令参数、日志或 Git 中收集或保存真实 Key。

安装后，Codex 按程序写入的绝对路径直接调用原生命令。普通更新由已安装程序的 `update` 子命令完成，沿用已保存的专门画图 Key，不要求用户再次输入。普通 `update` 先比较版本；版本相同时直接提示已是最新版，不会因 SHA-256 变化重新下载。同版本制品替换需要 `update --reinstall`，也可重新使用稳定安装入口；原生重装保留 Key，稳定安装入口会显式配置 Key。下载后的程序仍须通过清单中的 SHA-256 和程序版本校验。

用户文件默认位于 `~/.codex/`；设置 `CODEX_HOME` 时改用该目录：

- 程序、版本、安装清单和专用 Key：`tools/myautoapi-imagines-tool/`。
- Skill：`skills/myautoapi-image-generation/`。
- 原生命令放行规则：`rules/myautoapi-imagines-tool.rules`。
- 全局生图规则块：`AGENTS.md`。

安装或更新完成后需要完全退出并重新打开 Codex，再开始新任务，以加载新的规则。工具没有后台服务、计划任务或开机启动项，也不在普通生图时自动更新。

## 调用行为

当前发布程序保留上游 V0.9.2 的 `generate`、`edit`、`models`、`paths`、`doctor`、`recover`、`update` 和 `version` 等原生子命令。Codex 使用安装器生成的 Skill 中的绝对工具路径，一次提交一条原生命令；提示词直接通过 `--prompt` 传递，不创建 PowerShell 提示词临时文件，因此不需要额外清理临时提示词文件。具体引号和命令格式以当前安装的 Skill 为准。

API Key 由原生程序内部读取。排错先用 `doctor`；查看模型用 `models`；找回已完成调用的结果用 `recover`。用户要求修复时先诊断真实原因，用户明确要求更新或重装时才运行相应子命令。

默认模型为 `gpt-image-2.6-flare`，两个 2.6 模型默认 `2048x2048`、`high`、`png`。共支持以下十个模型：

| 模型 | 用途 |
| --- | --- |
| `gpt-image-2.6-flare` | 普通生图、参考图、修图和多图的默认模型 |
| `gpt-image-2.6-sunburst` | 高清、精修或重要成品 |
| `gpt-image-2.5-flare` | 用户明确指定时使用的静态图模型 |
| `gpt-image-2.5-sunburst` | 用户明确指定时使用的静态图模型 |
| `gpt-image-2` | 文生图、参考图、修图、多图 |
| `gpt-image-1k-th` | 文生图、参考图、修图、多图 |
| `gpt-image-2-adobe` | 文生图、多图 |
| `grok-imagine-image` | 文生图、参考图、修图、多图 |
| `nano-banana-2` | 文生图、参考图、修图、多图 |
| `nano-banana-pro` | 文生图、参考图、修图、多图 |

GIF 仅使用两个 2.6 模型的文生图模式，支持 `--frame-count` 和 `--frame-duration-ms` 请求参数；尺寸、帧数和时长以实际返回文件与 stdout 的动画信息为准。多图由工具内部拆成独立单张请求，并发上限为 3，保留成功结果。完整参数和限制以当前原生程序生成的 Skill 为准，不以保留源码中的旧模板代替。

## 发布复现

在文档仓库根目录，向迁移脚本传入固定上游仓库的检出目录：

```text
python myautoapi-imagines-tool/release/port_upstream.py <上游仓库目录>
```

脚本只接受已固定校验值的上游程序，生成：

- `docs/public/install/myautoapi-imagines-tool/releases/v0.9.2-upstream-c28a911/`：本次上游迁移制品的独立归档和 `SHA256SUMS`。旧 `releases/v0.9.2/` 保留原字节，不代表本次制品。
- `docs/public/install/myautoapi-imagines-tool/current/`：不带版本号的四平台文件和 `SHA256SUMS`。
- `docs/public/install/myautoapi-imagines-tool/latest.json`：原生更新使用的当前平台路径、大小与哈希。
- `release/install.ps1`、`release/install.sh` 及公开副本：同步新的制品哈希。
- `release/upstream-port.json`：可追溯的迁移记录。

安装脚本、更新清单、程序与校验文件必须一同发布；用户继续使用现有稳定入口，不需要追加版本号或查询参数。

`release/build_release.py` 是保留 Go 实现的历史构建器，默认拒绝运行并提示使用 `port_upstream.py`。只有明确传入 `--legacy-source` 才会构建历史源码；输出隔离在 `dist/legacy-v0.9.2/`，不再修改正式安装脚本中的哈希。它的输出不得覆盖当前正式制品、安装脚本、校验清单或 `latest.json`。不要将旧 Go/Python 测试通过等同于当前发布程序已经验证。

## 验证边界

正式制品应核对固定上游哈希、迁移后哈希、品牌残留、固定 API/更新域名、公开脚本和 `latest.json` 的一致性，并在隔离 `CODEX_HOME` 中用合成 Key 验证 Windows、Linux 安装、保留 Key 的更新和重装流程。Linux 本地 HTTPS 夹具已验证：普通更新跳过同版本不同哈希，`update --reinstall` 正确解析 `current/...` 相对路径并保留 Key。测试不得读取或提交真实 Key，也不以诊断测试发起付费生图请求。

Apple Silicon 程序重新计算现有 ad-hoc 签名的 1560 个 SHA-256 分页哈希；Intel Mac 程序没有该签名。两个 Mac 制品尚未实机验证，签名结构和校验值核验不能替代 Mac 上的安装及运行验证。远端生图是否成功仍取决于 MyAutoAPI 的模型、分组权限和接口服务，不能仅由本地安装测试推断。
