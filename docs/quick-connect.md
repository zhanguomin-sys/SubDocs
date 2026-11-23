---
title: Windows 接入助手
---

# Windows 接入助手

MyAutoAPI 中转快速接入助手适用于 Windows x64。可以连接 Codex、Claude 桌面版和 Claude Code，以及在助手内为 Codex 安装绘图模型和导入国产模型。

助手窗口宽 720 像素，高度随内容调整。绘图模型、国产模型和简体中文操作均在居中弹窗中进行。

<p><a href="/download/quick-connect/myautoapi-quick-connect-1.2.25-windows-x64.exe" download>下载 Windows 接入助手 V1.2.25（x64）</a> · <a href="/download/myautoapi-quick-connect-windows-x64.json">查看版本及 SHA-256</a></p>

这是免安装程序，需要 Windows 上的 Microsoft Edge WebView2。当前文件未签名；如果你的设备禁止运行未签名软件，可以继续 [用 CC Switch 图形化接入](/codex-cc-switch)。

## Codex 和 Claude

1. 按 [快速开始](/quick-start#_1-创建本中转站的-api-密钥) 创建对应分组的 API Key：Codex 用 GPT 分组，Claude 用 Claude 分组。
2. 打开接入助手，选择客户端。可以点「获取 MyAutoAPI 密钥」打开本站密钥管理页。复制对应分组的 Key。助手启动、切回前台或切换页签时，空白输入框会自动填入剪贴板中的完整 Key；也可以手动粘贴。确认分组后，点击「一键连接 Codex」或「一键连接 Claude」。
3. 保存接入配置后，重新打开对应客户端。有旧配置时，完成提示会给出备份位置。

一键连接 Codex、切换至 GPT-6.2、安装绘图模型、导入国产模型和设为简体中文成功后，助手会自动结束当前 Windows 会话里的 ChatGPT/Codex 进程（含后台进程）。正在运行的任务会中断，请先保存工作。关闭不完全时，按助手提示完全退出客户端后再打开。

如果「Codex 状态」显示「已配置 MyAutoAPI」，连接和切换 GPT-6.2 时可以留空使用已保存的 GPT 分组 API Key，助手仍会向 MyAutoAPI 验证。首次接入时留空，或输入的密钥无效时，会弹出提示并提供打开「API 密钥」页的按钮。

助手向 MyAutoAPI 的用量接口验证密钥，通过后才会保存配置。Codex 会切换到 MyAutoAPI 服务商，Key 存在当前 Windows 用户的环境变量中。如果你在用 Codex 命名配置，可继续按 [手动接入指南](/codex/manual-config) 配置相应 Profile。Claude Code 使用自己的用户设置。要恢复旧接入，可以关闭对应客户端并用助手提示的备份恢复配置。

使用支持 GPT-6.2 的 MyAutoAPI 密钥时，也可以点击「切换至 GPT-6.2」。助手按同样的方式完成接入，并把默认模型设为 `gpt-6.2-sol`、推理力度设为 High。完全退出并重开 ChatGPT/Codex 后，新建任务使用。

## 安装绘图模型和导入国产模型

- **安装绘图模型**：助手会安装 MyAutoAPI 绘图工具及 Codex 技能。首次安装时，从 MyAutoAPI 密钥管理页创建「专门画图分组」的密钥，粘贴后点「安装内置版」。更新时可以留空，继续使用已保存的画图密钥。安装完成后重新打开 Codex。详见 [生图工具](/image-video-group-image)。
- **导入国产模型**：点「导入内置模型」，即可使用仓库发布的模型列表。助手保留原有 GPT、默认模型和接入密钥。导入完成后重新打开 Codex。可用分组和模型列表见 [导入国产模型](/codex-models)。
- 点击对应按钮后，在弹窗内选择安装方式并查看结果。两项均可使用内置包离线导入。「联网检查更新」会显示 MyAutoAPI 官网版本；有新版时，点击对应按钮下载、校验并导入。

## Claude 桌面版

1. 在助手里选「Claude」，填入 Claude 分组的 API Key，点「一键连接 Claude」。
2. 看到配置完成后，从 Windows 托盘完全退出 Claude，再重新打开。桌面版会使用 MyAutoAPI 网关，无需登录 Claude.ai。

如果仍然显示登录页，先确认已完全退出旧进程，并使用新版 Claude 桌面版。由单位统一管理的设备以其管理员设置为准。详见 [Claude 官方接入说明](https://claude.com/docs/third-party/claude-desktop/in-app-configuration)。

## 简体中文与更新

- 在「Codex」页点击「设为简体中文」，再在弹窗内确认。助手会重新检查配置；如果尚未接入 MyAutoAPI 或无法读取配置，不会更改语言设置。保存成功后重开 Codex；若仍显示英文，查看 [Codex 界面语言指南](/codex-chinese)。
- 点击「检查助手更新」。发现较新版本后，可点击「下载新版」；助手校验文件大小和 SHA-256，保存后双击新版即可。
- macOS 和 Linux 沿用 [现有客户端接入方式](/clients/)。
