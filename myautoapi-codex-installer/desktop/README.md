# 构建 MyAutoAPI 下载工具

Windows 接入助手为 Go + Wails 窗口程序，同一窗口可接入 Codex、Claude，安装生图工具和导入国产模型。Apple 芯片 Mac 使用终端 `.app`，分别提供 Codex 接入、生图工具安装和国产模型导入。

## 当前发布方式

在仓库根目录运行，要求 Go 1.25+ 和 Python 3：

```bash
python3 myautoapi-codex-installer/desktop/publish_downloads.py --terminal-mac
```

这会从源码交叉编译 Windows x64 和 Mac arm64 程序，生成 `myautoapi-codex-installer/dist/desktop/`，再更新以下网站下载文件：

- `docs/public/download/myautoapi-quick-connect-windows-x64.exe`
- `docs/public/download/myautoapi-quick-connect-mac-arm64.zip`
- `docs/public/download/myautoapi-tools.zip`

合集内 Windows 只有 `1.MyAutoAPI中转快速接入助手.exe`；Mac 保留编号 1、2、3 三个 `.app`。根目录包含使用说明。每个下载文件不得超过 Cloudflare Pages 的 25 MiB 上限。

Windows 助手内置已发布的 MyAutoAPI 生图工具 **V0.8.3**；Mac 生图 `.app` 内置同版 arm64 程序。构建时核对 `SHA256SUMS`，不升级本机生图工具，也不调用生图 API。默认模型为 GPT Image 2.6 Flare，保留 GPT Image 2.5 的显式调用；GIF 尚未纳入本次发布。

国产模型脚本只在 `../codex-domestic-models/` 维护。修改后先构建并同步到 `../install/` 和 `docs/public/install/`，再重新打包；Windows 源文件保留 UTF-8 BOM，网站副本去除 BOM，打包时核对内容一致。

## Mac 启动器

当前使用 `--terminal-mac`，双击后在终端提示使用剪贴板中的密钥或手动粘贴，不登记 `myautoapi://`。该启动器可在 Linux 或 Windows 交叉编译，不要求 Apple Cocoa。

源码还保留了可选的 Cocoa 协议启动器，由 `.github/workflows/mac-quick-connect.yml` 在 Apple 芯片 Mac 上测试和构建。只有以后后台接入导入协议、并完成对应验收后，才应发布该版本。`build.py --mac-only` 在 Mac 上构建它；`--mac-launcher <路径>` 可复用已构建的启动器；这两种方式与 `--terminal-mac` 互斥。

## 验证

```bash
cd myautoapi-codex-installer/quick-connect && go test ./...
cd ../desktop && go test ./... && python3 -m unittest test_packaging.py
```

回到仓库根目录运行 `node --test tests/*.test.mjs` 和 `npm run docs:build`。

测试覆盖配置写入、备份和恢复、本地假服务器的密钥检查、中文路径、下载包结构、PE/Mach-O 架构、可执行权限和内置程序版本。脚本测试使用临时目录和合成密钥。

当前构建不包含发布者代码签名或 Apple 公证。Windows 和 Mac 的实际桌面交互仍需在目标系统验收。

固定下载地址更新时覆盖文件，并同步 `docs/public/_headers` 中合集附件名的发布时间。
