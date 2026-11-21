# MyAutoAPI 文档站

MyAutoAPI 中转站的接入教程与工具下载站，使用 VitePress 构建。

2026-11-21 同步 Gemini 和 GPT-6.2 价格文档，更新国产模型导入脚本 1.0.3，发布 Windows 接入助手 1.2.2 和生图工具 0.9.7。价格、分组和 DeepSeek 1017 模型 ID 按本站配置。详见 [同步记录](UPSTREAM-SYNC.md)。

- 文档：<https://docs.myautoapi.com>
- 中转站：<https://myautoapi.com>
- API：<https://api.myautoapi.com>
- 无限画布：<https://canvas.myautoapi.com>

## 本地预览

```bash
npm ci
npm run docs:dev
```

## 构建发布

```bash
npm run docs:build
```

Cloudflare Pages 的构建命令为 `npm run docs:build`，产物目录为 `docs/.vitepress/dist`，Node.js 使用 22。

完整操作见 [DEPLOYMENT.md](DEPLOYMENT.md)。

## 子项目与下载文件

| 子项目 | 维护目录 | 随文档站发布的文件 |
| --- | --- | --- |
| Codex 一键配置 | `myautoapi-codex-installer/` | `docs/public/install/myautoapi-codex-installer/` |
| Codex 国产模型导入 | `myautoapi-codex-installer/codex-domestic-models/` | `docs/public/install/codex-models.ps1`、`codex-models.sh` |
| 图片生成工具 V0.9.7 | `myautoapi-imagines-tool/` | `docs/public/install/myautoapi-imagines-tool/` |
| Windows 接入助手 V1.2.2 | `myautoapi-codex-installer/quick-connect/` | `docs/public/download/` |

图片工具发布文件包含 Windows x64、macOS Intel、macOS Apple Silicon 和 Linux x64 原生程序、`SHA256SUMS` 和 `latest.json`。用户的安装入口和 `current/` 下载地址不带版本号。

当前图片工具使用上游 V0.9.7 原生程序，经 `myautoapi-imagines-tool/release/port_upstream.py` 迁移至本站域名。来源和哈希记录在 `myautoapi-imagines-tool/release/upstream-port.json`，本次独立归档在 `releases/v0.9.7-upstream-af1c363/`。普通升级沿用已保存的画图 Key。详见 [图片工具维护说明](myautoapi-imagines-tool/README.md)。
