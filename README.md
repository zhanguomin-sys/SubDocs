# MyAutoAPI 文档站

MyAutoAPI 中转站的接入教程与工具下载站，使用 VitePress 构建。

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

| 子项目 | 源码目录 | 随文档站发布的文件 |
| --- | --- | --- |
| Codex 一键配置 | `myautoapi-codex-installer/` | `docs/public/install/myautoapi-codex-installer/` |
| 图片生成工具 V0.8.1 | `myautoapi-imagines-tool/` | `docs/public/install/myautoapi-imagines-tool/` |

图片工具发布文件包含 Windows x64、macOS Intel、macOS Apple Silicon 和 Linux x64 原生程序，以及 `SHA256SUMS` 校验文件。
