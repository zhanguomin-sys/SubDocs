---
title: 快速开始
---

# 快速开始

先装好 Codex App（[安装方法](/codex/)），再按下面 4 步接入。

<a id="_1-创建-api-key"></a>

## 1. 创建密钥

后台 **API 密钥** → **创建密钥**。名称随意填，比如「测试」；分组选 GPT 开头的。

<details>
<summary>截图</summary>

<a class="doc-image-link" href="/images/quick-start/create-api-key-1.jpg" target="_blank" rel="noopener">
  <img src="/images/quick-start/create-api-key-1.jpg" alt="进入 API 密钥页面并点击创建密钥">
</a>

<a class="doc-image-link" href="/images/quick-start/create-api-key-2.jpg" target="_blank" rel="noopener">
  <img src="/images/quick-start/create-api-key-2.jpg" alt="填写密钥名称并选择正确分组">
</a>

<a class="doc-image-link" href="/images/quick-start/create-api-key-3.jpg" target="_blank" rel="noopener">
  <img src="/images/quick-start/create-api-key-3.jpg" alt="在 API 密钥列表中复制 API Key">
</a>

</details>

## 2. 充值

后台首页点 **去充值**。新人加 <span style="color: #d93025;"><strong>QQ 群：685861462</strong></span> 领 2 元。

<a id="quick-connect-helper"></a>

## 3. 连接 Codex

1. [下载 Windows 接入助手](/download/myautoapi-quick-connect-windows-x64.exe)，双击打开一次。Mac 用 [Mac 版](/download/myautoapi-quick-connect-mac-arm64.zip)（Apple 芯片）。
2. 回后台 **API 密钥** 页，复制第 1 步创建的完整密钥。
3. Windows 将密钥粘贴到助手，点击 **一键连接 Codex**；Mac 双击助手，按终端提示使用剪贴板密钥或粘贴密钥。

导入时会先自动关闭 Codex，先保存正在进行的任务。权限默认设为 **完全访问**。

打不开助手时：Windows 点 **更多信息** → **仍要运行**；Mac 到 **系统设置 → 隐私与安全性** 点 **仍要打开**，问到控制“终端”时点 **允许**。

不用网页也行：复制密钥后打开助手，Windows 点 **一键连接 Codex**，Mac 会直接读取剪贴板。

<a id="_3-重新打开-codex-测试"></a>

## 4. 发送第一条消息

重新打开 Codex，新建任务，发送 `测试`。能回复就接入完成了。

<a id="_2-运行接入命令"></a>
<a id="command-line"></a>

## 没反应？用 PowerShell 命令行

按 `Win + X`，打开 **终端**，粘贴命令回车（Mac 打开 **终端**，用 Mac 那行）：

::: code-group

```powershell [Windows]
irm https://docs.myautoapi.com/install/myautoapi-codex-installer/install.ps1 | iex
```

```bash [Mac]
curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/install.sh | bash
```

:::

提示输入 API Key 时，粘贴第 1 步的密钥，回车。完成后做第 4 步。

<a id="运行脚本遇到问题"></a>

## 遇到问题

| 提示或现象 | 怎么办 |
| --- | --- |
| 打不开助手 | 重新下载并解压；还不行用 [命令行](#command-line) |
| 要安装 WebView2 运行库 | 点 **确定** 下载安装，或改用 [命令行](#command-line) |
| 密钥格式不对 | 回后台重新复制完整密钥（带 `****` 的是遮挡过的，不能用） |
| 密钥无效、已停用、过期或额度用完 | 换一把可用的密钥 |
| 网络超时、连不上服务器、服务器出错 | 检查网络或代理后重试。命令行这时仍会写入，可以直接做第 4 步试试 |
| 命令提示无法下载脚本 | 检查网络后重试，或改用 [CC Switch](/codex-cc-switch) |
| Mac：另一个窗口正在配置 Codex | 先把那个终端窗口走完或关掉 |
| 没能自动关闭 Codex | 手动退出 Codex 再打开，不行就重启电脑 |
| Codex 报 `INSUFFICIENT_BALANCE` 或 `Insufficient account balance` | 充值后重试 |
| 其他报错 | 看 [报错与踩坑](/errors/) |

<details>
<summary>改了哪些文件</summary>

只改 `C:\Users\你的用户名\.codex\`（Mac 为 `~/.codex/`）里的三个文件：

- `config.toml`：接口地址、模型和完全访问权限；Windows 原有的沙箱设置（`[windows]` 段）保留
- `auth.json`：密钥
- `.codex-global-state.json`：Codex App 输入框下方的权限选择，改成完全访问

旧文件备份在同目录，文件名带 `.bak.时间戳`。原来配过 MCP 的，从备份里复制回 `config.toml`。

</details>

<details>
<summary>改回官方账号登录</summary>

Windows 助手点右下角 **恢复官方登录**；或者运行：

::: code-group

```powershell [Windows]
irm https://docs.myautoapi.com/install/myautoapi-codex-installer/restore.ps1 | iex
```

```bash [Mac]
curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/restore.sh | bash
```

:::

会清空登录凭据和中转站的接口配置，MCP 等其它配置不动。之后重新打开 Codex，用官方账号登录。

</details>

## 其他方式

- 工具合集：[下载 ZIP](/download/myautoapi-tools.zip)，解压后打开 `1.MyAutoAPI中转快速接入助手`。
- 要在官方账号和中转站之间来回切：[CC Switch 接入](/codex-cc-switch)。
- 接入 Claude 桌面版（Windows 助手也能一键连接）：[Claude Desktop 接入](/claude/desktop)。
- 自己改配置文件：[手动配置](/codex/manual-config)。

## 接下来

- [模型价格](/models)
- 在 Codex 里生图：另建一把 **专门画图分组** 的密钥，按 [安装生图工具](/image-video-group-image) 安装。
