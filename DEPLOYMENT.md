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
- `docs/public/install/codex-models.ps1`
- `docs/public/install/codex-models.sh`
- `docs/public/install/myautoapi-imagines-tool/install.ps1`
- `docs/public/install/myautoapi-imagines-tool/install.sh`
- `docs/public/install/myautoapi-imagines-tool/releases/v0.9.2-upstream-c28a911/`
- `docs/public/install/myautoapi-imagines-tool/current/`
- `docs/public/install/myautoapi-imagines-tool/latest.json`

### 图片工具制品来源

当前 V0.9.2 制品来自 `youdaowudao/subdocs` 固定提交 `c28a91174db4f065ff4c4121e348f20afebda85c` 的原生程序。它们通过 `myautoapi-imagines-tool/release/port_upstream.py` 等长替换品牌、固定 API 和更新域名，并重新计算 Apple Silicon 程序现有 ad-hoc 签名的分页哈希。来源校验值与输出校验值记录在 `myautoapi-imagines-tool/release/upstream-port.json`。

维护者在仓库根目录使用固定上游检出目录复现制品：

```text
python myautoapi-imagines-tool/release/port_upstream.py <上游仓库目录>
```

脚本校验四个平台的上游 SHA-256，生成独立的 `releases/v0.9.2-upstream-c28a911/` 归档、无版本号的 `current/` 制品、两份校验清单、`latest.json` 和安装脚本中的新哈希。旧 `releases/v0.9.2/` 保留原字节与 immutable 缓存，不再覆盖。不要运行旧 `build_release.py` 来发布：保留的 Go 源码不是当前正式实现；该入口默认拒绝构建，`--legacy-source` 只允许历史复现，其输出不得覆盖正式制品或入口。

初次安装或显式重新配置 Key 时，公开安装入口调用 `install --key-stdin`。普通更新使用工具自身的 `update`，保留已保存的 Key；它先比较版本，版本相同时不会因 SHA-256 变化下载。同版本制品替换需 `update --reinstall`，该原生重装保留 Key；也可重新使用稳定安装入口并显式配置 Key。下载后仍校验 SHA-256 与程序版本。保留用户当前不带版本号的安装入口与 `current/` 地址，不要求用户追加版本参数。

发布前使用隔离 `CODEX_HOME` 和合成 Key 检查 Windows、Linux 安装及更新流程、生成的 Skill/规则与固定域名，并核对所有制品的 SHA-256。两个 Mac 制品尚未实机验证；Apple Silicon 只完成 ad-hoc 签名分页哈希核验，不能据此宣称 Mac 实机可用。

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
https://docs.myautoapi.com/install/codex-models.ps1
https://docs.myautoapi.com/install/codex-models.sh
https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1
https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh
https://docs.myautoapi.com/install/myautoapi-imagines-tool/latest.json
```

安装脚本必须直接显示纯文本，`latest.json` 必须返回有效 JSON，四个平台的 `current/` 制品必须可以下载且与公开清单的大小和 SHA-256 一致。安装入口和更新清单应及时获取新内容，避免缓存旧哈希与新程序不一致；安装脚本、清单、`current/` 文件及对应校验文件必须作为同一次发布上线。之后每次推送到 `main`，Cloudflare Pages 会自动重新构建并发布。

官方操作参考：[VitePress 部署](https://developers.cloudflare.com/pages/framework-guides/deploy-a-vitepress-site/)、[自定义域名](https://developers.cloudflare.com/pages/configuration/custom-domains/)。
