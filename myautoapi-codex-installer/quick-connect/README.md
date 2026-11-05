# MyAutoAPI 中转快速接入助手

发布版本：**1.1.4**。Windows 使用窗口助手；Apple 芯片 Mac 使用终端应用。当前后台提供复制 API Key 的流程，本次成品不依赖网页导入链接。

| 平台 | 成品 | 功能 |
| --- | --- | --- |
| Windows x64 | `1.MyAutoAPI中转快速接入助手.exe` | 连接 Codex、连接 Claude Desktop、安装 MyAutoAPI 生图工具、导入或移除 Codex 国产模型 |
| Mac Apple 芯片 | `1.MyAutoAPI中转快速接入助手.app` | 终端中连接 Codex，读取剪贴板密钥或手动粘贴 |

Windows 和 Mac 的独立成品都包含在 [下载合集](../../docs/public/download/myautoapi-tools.zip)；固定下载文件由 `desktop/publish_downloads.py --terminal-mac` 生成。Windows 单文件与 Mac 应用下载地址分别是 `https://docs.myautoapi.com/download/myautoapi-quick-connect-windows-x64.exe` 和 `https://docs.myautoapi.com/download/myautoapi-quick-connect-mac-arm64.zip`。

## 连接流程

从 [MyAutoAPI 网站](https://myautoapi.com) 的 API 密钥页复制完整密钥。Windows 将密钥粘贴到助手后点「一键连接 Codex」；Claude Desktop 在其独立页面配置。Mac 双击助手后按终端提示使用剪贴板中的密钥，或手动粘贴。连接前会检查 `https://api.myautoapi.com/v1/usage`，随后更新 Codex 配置并关闭正在运行的客户端；请先保存任务。

API 地址固定为 `https://api.myautoapi.com`，默认模型 `gpt-6.1-sol`，密钥从 API 密钥页取得。Codex 配置设为完全访问，保留 Windows 原有沙箱选项。详见 [快速开始](../../docs/quick-start.md)。

Mac 应用只配置 Codex；Claude Desktop、图片工具和国产模型导入使用 Mac 合集里的编号 2、3 应用。Mac `.app` 没有开发者签名或 Apple 公证，首次打开时需在系统设置中允许。

## 源码结构

- `desktop/`：Windows、Mac 成品和网站下载文件的统一构建入口。
- `connect/`：密钥检查、配置写入、客户端关闭与 Windows 窗口助手。
- `macapp/`：Mac 终端应用和启动脚本。
- `../install/`：Codex 接入、恢复及国产模型导入的安装脚本。
- `../codex-domestic-models/`：国产模型目录的维护源码与测试。

`myautoapi://` 解析器和 Cocoa 启动器保留为可选开发能力，但当前网站没有导入按钮，本次下载成品也不登记该协议。不要把可选协议构建描述成当前用户流程。

运行 Go 行为测试：

```bash
go test ./connect ./tools/resources
```

Mac 协议启动器的独立测试：

```bash
go test ./macapp
```
