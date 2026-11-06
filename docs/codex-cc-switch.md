---
title: Codex CC Switch 接入
sidebarTitle: CC Switch 接入
---

# Codex CC Switch 接入

CC Switch 是图形化配置切换工具，适合不想运行脚本的 Codex App 用户。导入 MyAutoAPI 配置后，彻底退出 Codex，再重新打开测试。

## 1. 安装 CC Switch

| 下载来源 | 入口 |
| --- | --- |
| 官方 GitHub | [GitHub Releases](https://github.com/farion1231/cc-switch/releases) |
| GitCode | [国内下载](https://gitcode.com/YujinDawnlight/cc-switch-download) |
| Gitee | [国内下载](https://gitee.com/iamzhihuix/cc-switch-mirror/releases) |

<details>
<summary>查看下载文件选择说明</summary>

<a class="doc-image-link" href="/images/ccswitch/ccswitchdownload.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitchdownload.jpg" alt="CC Switch 下载文件选择说明，macOS 选择 dmg 文件，Windows 选择 msi 文件">
</a>

</details>

<details>
<summary>或者采用命令行安装方式</summary>

Windows 打开 PowerShell 执行：

```powershell
winget install -e --id farion1231.CC-Switch
```

macOS 已安装 Homebrew 时执行：

```bash
brew install --cask cc-switch
```

</details>

安装完成后打开 CC Switch。

## 2. 从管理后台导入配置

正在使用 ChatGPT 桌面应用里的 Codex，并且需要保留 ChatGPT 账号登录时，先按 [保留 ChatGPT 登录](/clients/chatgpt-login-myautoapi) 打开 CC Switch 的保持登录开关，再回到本页继续导入。普通 Codex App 接入，直接继续下面步骤。

回到 MyAutoAPI 管理后台 **API 密钥**，找到要使用的 Key，点击 **导入到 CCS**。

浏览器询问是否打开 CC Switch 时，点击 **打开**。确认应用类型是 `Codex`，再点击 **导入**。

<details>
<summary>查看导入截图</summary>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入1.jpg" alt="在 API 密钥页面点击导入到 CCS 并允许浏览器打开 CC Switch">
</a>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入2.jpg" alt="核对 Codex 的 MyAutoAPI 供应商配置并点击导入">
</a>

</details>

## 3. 关闭并重新打开 Codex

导入完成后，确认 `MyAutoAPI` 右侧显示 **使用中**。

<details>
<summary>查看使用中截图</summary>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入3.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入3.jpg" alt="确认 MyAutoAPI 地址正确并显示使用中">
</a>

</details>

彻底退出 Codex，再重新打开并新建任务，发送：

```text
测试
```

能正常回复，就接入完成。测试正常后可以关闭 CC Switch。

## CC Switch 接入失败先看这里

| 遇到的问题 | 先做这个 | 还是不行 |
| --- | --- | --- |
| 点击 **导入到 CCS** 没有反应 | 确认 CC Switch 已安装并打开，再允许浏览器打开外部应用 | 重新打开浏览器和 CC Switch 后再点一次 |
| 导入窗口里的应用类型不是 `Codex` | 点击取消，回到同一个 API Key 后重新点击 **导入到 CCS** | 不要导入到其它应用类型 |
| 导入后没有显示 **使用中** | 在 CC Switch 里选中 `MyAutoAPI`，确认右侧显示 **使用中** | 回后台重新导入当前 Key |
| 导入后 Codex 仍使用旧配置 | 彻底退出 Codex，重新打开后新建任务测试 | 看 [报错与踩坑](/errors/) |
| `401 Unauthorized` | 回到后台重新导入当前有效的 API Key | 看 [报错与踩坑](/errors/) |
| `403 Forbidden` | 确认当前 Key 所属分组支持导入的模型 | 看 [报错与踩坑](/errors/) |
