---
title: Windows 接入助手
---

# Windows 接入助手

MyAutoAPI 接入助手 V1.2.2 适用于 Windows x64。可以配置 Codex 或 Claude Code、为 Claude 桌面版导出接入模板、设置 Codex 的简体中文偏好，以及检查助手更新。

<p><a href="/download/quick-connect/myautoapi-quick-connect-1.2.2-windows-x64.exe" download>下载 Windows 接入助手 V1.2.2（x64）</a> · <a href="/download/myautoapi-quick-connect-windows-x64.json">查看版本及 SHA-256</a></p>

这是免安装程序，需要 Windows 上的 Microsoft Edge WebView2。当前文件未签名；如果你的设备禁止运行未签名软件，可以继续 [用 CC Switch 图形化接入](/codex-cc-switch)。

## Codex 和 Claude Code

1. 按 [快速开始](/quick-start#_1-创建本中转站的-api-密钥) 创建对应分组的 API Key：Codex 用 GPT 分组，Claude Code 用 Claude 分组。
2. 打开接入助手，选择客户端，粘贴对应的 Key，点击「验证并保存配置」。
3. 完全退出并重新打开对应客户端。有旧配置时，完成提示会给出备份位置。

Codex 会切换到 MyAutoAPI 服务商，Key 存在当前 Windows 用户的环境变量中。如果你在用 Codex 命名配置，可继续按 [手动接入指南](/codex/manual-config) 配置相应 Profile。Claude Code 使用自己的用户设置。要恢复旧接入，可以关闭对应客户端并用助手提示的备份恢复配置。

## Claude 桌面版

1. 点击助手中的「保存接入模板」。模板只包含网关地址和认证方式。
2. 在 Claude 桌面版选择 **Help → Troubleshooting → Enable Developer Mode**，再选择 **Developer → Configure Third-Party Inference**。
3. 在配置窗口右上角打开 **Import configuration**，选择刚保存的模板。填入 Claude 分组的 Key，在 Models 中添加本分组支持的模型，然后点击 **Apply Changes → Relaunch now**。

如果找不到该菜单，更新 Claude 桌面版后再试。由单位统一管理的设备以其管理员设置为准。详见 [Claude 官方接入说明](https://claude.com/docs/third-party/claude-desktop/in-app-configuration)。

## 简体中文与更新

- 点击「设为简体中文」并重开 Codex。若仍显示英文，查看 [Codex 界面语言指南](/codex-chinese)。
- 点击「检查更新」。发现较新版本后，可点击「下载新版」；助手校验文件大小和 SHA-256，保存后双击新版即可。
- macOS 和 Linux 沿用 [现有客户端接入方式](/clients/)。
