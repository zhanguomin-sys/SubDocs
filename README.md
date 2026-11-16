# MyAutoAPI 文档站

MyAutoAPI 中转站的接入教程与工具下载站，使用 VitePress 构建。

2026-11-16 同步上游文档 0.1.18（`2f835e2`）、Codex 配置/恢复脚本及国产模型导入脚本 1.0.2。遵循当前正式发布的回退状态，生图工具继续为 V0.9.2；保留 MyAutoAPI 自有价格、分组和 DeepSeek 1017 模型 ID。本次未发布上游的图形化接入助手或 Mac App 合集，快速开始继续使用现有命令入口。详见 [同步记录](UPSTREAM-SYNC.md)。

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
| 图片生成工具 V0.9.2 | `myautoapi-imagines-tool/` | `docs/public/install/myautoapi-imagines-tool/` |

图片工具发布文件包含 Windows x64、macOS Intel、macOS Apple Silicon 和 Linux x64 原生程序、`SHA256SUMS` 和 `latest.json`。用户的安装入口和 `current/` 下载地址不带版本号。

当前图片工具以对方仓库固定提交的 V0.9.2 原生程序为来源，通过 `myautoapi-imagines-tool/release/port_upstream.py` 等长替换品牌、固定 API 和更新域名；来源与哈希记录在 `myautoapi-imagines-tool/release/upstream-port.json`，本次归档为 `releases/v0.9.2-upstream-c28a911/`，旧 `releases/v0.9.2/` 保留原字节。保留的 Go 源码用于历史参考，不是当前正式制品的构建来源，不能用旧构建脚本覆盖当前下载文件。安装入口显式配置画图 Key；工具原生 `update` 沿用已保存的 Key，但仅在版本变化时下载，同版本制品替换使用 `update --reinstall` 或稳定安装入口。详见 [图片工具维护说明](myautoapi-imagines-tool/README.md)。
