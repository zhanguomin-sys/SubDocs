# 上游同步记录

## 2026-11-21

- 按 [上游 0.1.33](https://github.com/youdaowudao/subdocs/commit/44f424414dcbc5368bcda01ace6826bca3b353f0) 同步选定内容。Gemini 文档和页面采用本站 0.8 token 倍率，Argon、Barium、Carbon、Hydrogen 和 Neon 同组；Neon 的分组缓存读取价待确认。本站未配置上游的按次分组。
- 补充 GPT-6.2 Astra / Sol 官方参考价，本站分组价格及可用性以实际配置为准。
- 国产模型导入脚本更新为 1.0.3；Codex 官方快照为 0.170.0，本站十个国产模型和 DeepSeek 1017 的请求 ID 保持原样。
- Windows 接入助手 V1.2.2 从本站源码构建。接入 Codex、Claude Code，以及通过 Claude 桌面应用自身的导入功能配置网关；语言功能只保存 Codex 的简体中文偏好。macOS 沿用现有接入方式。
- 发布生图工具 V0.9.7，来源为上游 `af1c363`；四平台制品按固定校验值迁移，保留旧发布归档和本站生图分组价格。

验证覆盖 VitePress 构建、两种价格模式、官方 Codex 0.170.0 和 Shell / PowerShell 下的模型导入、Windows 接入助手启动及配置与更新检查。生图工具在 Windows 和 Linux 使用测试 Key 安装，Linux 在本地 HTTPS 服务验证 0.9.2 升级保留 Key 与 0.9.6 残留 Skill 恢复；Mac 制品核对文件结构和哈希，未实机运行。

## 2026-11-16

- 基线：MyAutoAPI `f205770d63d56b075c919ac9ae6b7edbc845e77a`，备份分支 `backup/before-upstream-20261116`。
- 来源：[youdaowudao/subdocs 2f835e2](https://github.com/youdaowudao/subdocs/commit/2f835e2913cf4379c80cdc2f0424221c38ff716f)，文档版本 0.1.18。
- 同步 Codex、Claude、国产工具教程分类，新增 Qwen Code、Gemini CLI 教程；增加中文搜索分词、旧网址重定向与移动端价格卡片。
- 同步 Codex 一键配置和恢复脚本，默认模型跟随上游为 `gpt-6.1-sol`，新增 Key 有效性检查。配置只写入接口和默认模型字段，清理旧 provider 与历史根级设置，保留其它配置段，旧文件先备份；详见脚本维护说明。所有验证在隔离测试目录使用假 Key 和本地假服务运行。
- 国产模型导入脚本版本 1.0.2，内置 GPT 快照对应 Codex 0.165.0。保留本站 10 个国产模型及 `deepseek-v4.1-flash-1017`，继续移除本站不可用的 `deepseek-flash` 别名。
- 保留 MyAutoAPI 域名、群号 685861462、联系图片、模型价格和分组。智商雷达继续使用此前确认的外部服务。
- 生图工具当前正式版本仍为 V0.9.2。V0.9.6 于基线提交前已被回退，本次不重新启用；生图源码、移植记录、安装入口、清单和原生程序均未修改。
- 本次不发布图形化接入助手及 Mac App 合集；相应安装引导未引入，继续使用本站现有命令安装入口。

### 本地验证

- Node 文档与组件测试：97 项，95 通过，2 项平台跳过，0 失败。
- Linux 隔离环境中的 Shell / PowerShell 配置脚本测试：126 通过，0 失败。
- Shell / PowerShell 恢复脚本：12 通过，0 失败。
- 国产模型脚本：53 项通过，含导入、重复执行、异常回滚、卸载、GPT 保留和进程控制模拟。
- VitePress 1.6.4 构建通过；浏览器核验中文搜索、教程分类和四个入口按钮，390px 手机端价格页无横向溢出。发布脚本副本、BOM/LF、10 模型名单、品牌和旧网址重定向校验通过；生图制品与价格数据和基线逐字节一致。
- 未进行真实 Key 调用、计费测试或原生 macOS 桌面实机验证。
