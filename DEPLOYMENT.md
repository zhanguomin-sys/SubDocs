# 部署到 GitHub 和 Cloudflare Pages

本站使用 VitePress 构建。GitHub 保存源码，Cloudflare Pages 连接仓库并在每次推送到 `main` 后自动发布。

## 1. 本地检查

在仓库根目录运行：

```powershell
npm ci
npm run docs:build
node --test tests/*.test.mjs
```

构建产物位于 `docs/.vitepress/dist`。发布前还要确认以下地址对应的源文件存在：

- `docs/public/install/myautoapi-codex-installer/install.ps1`
- `docs/public/install/myautoapi-codex-installer/install.sh`
- `docs/public/install/myautoapi-imagines-tool/install.ps1`
- `docs/public/install/myautoapi-imagines-tool/install.sh`
- `docs/public/install/myautoapi-imagines-tool/releases/v0.8.1/`

## 2. 上传 GitHub

使用仓库：<https://github.com/zhanguomin-sys/SubDocs>。首次上传使用清理后的发布副本，保留原项目和备份，避免把旧提交历史中的过期安装包、旧品牌截图一起上传。

后续在发布副本目录更新并推送：

```powershell
Set-Location D:\Downloads\subdocs-publish
git add -A
git commit -m "更新 MyAutoAPI 文档"
git push origin main
```

在另一台电脑编辑时，先克隆仓库：

```powershell
git clone https://github.com/zhanguomin-sys/SubDocs.git
Set-Location SubDocs
npm ci
npm run docs:build
```

## 3. 创建 Cloudflare Pages 项目

1. 登录 Cloudflare，进入 **Workers & Pages**。
2. 点击 **Create application**，选择 **Pages** 和 **Import an existing Git repository**。
3. 连接 GitHub，选择 `zhanguomin-sys/SubDocs` 仓库。
4. 生产分支填写 `main`。
5. Framework preset 选择 `VitePress`；没有该选项时选择 `None`。
6. Build command 填写 `npm run docs:build`。
7. Build output directory 填写 `docs/.vitepress/dist`。
8. Root directory 留空。
9. 添加环境变量 `NODE_VERSION=22`。
10. 点击 **Save and Deploy**，等待首次构建完成。

## 4. 绑定文档域名

1. 打开 Pages 项目的 **Custom domains**。
2. 点击 **Set up a custom domain**。
3. 输入 `docs.myautoapi.com` 并确认。
4. 当前 `docs` 已有指向服务器的 A 记录。删除这条 `docs` A 记录，再让 Pages 自动创建并接管 `docs.myautoapi.com` 的 DNS 记录。
5. 等待证书状态变为 Active，再打开 `https://docs.myautoapi.com`。

`canvas.myautoapi.com` 是独立的无限画布服务，继续保留当前指向服务器的 A 记录。本次 Pages 项目只绑定 `docs.myautoapi.com`，不要删除或修改 `canvas` 记录。

## 5. 上线后验证

依次检查：

```text
https://docs.myautoapi.com/
https://docs.myautoapi.com/install/myautoapi-codex-installer/install.ps1
https://docs.myautoapi.com/install/myautoapi-codex-installer/install.sh
https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1
https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh
```

安装脚本必须直接显示纯文本，V0.8.1 的四个平台制品必须可以下载。之后每次推送到 `main`，Cloudflare Pages 会自动重新构建并发布。

官方操作参考：[VitePress 部署](https://developers.cloudflare.com/pages/framework-guides/deploy-a-vitepress-site/)、[自定义域名](https://developers.cloudflare.com/pages/configuration/custom-domains/)。
