---
title: Codex CC Switch 接入
sidebarTitle: CC Switch 接入
---

# Codex CC Switch 接入

CC Switch 是图形化配置切换工具：一键导入 MyAutoAPI 配置，也能一键切回官方账号。

## 1. 安装 CC Switch

从 [GitHub Releases](https://github.com/farion1231/cc-switch/releases) 或 [国内下载](https://gitcode.com/YujinDawnlight/cc-switch-download) 下载安装包：Windows 选 `.msi`，Mac 选 `.dmg`。安装后打开 CC Switch。

<details>
<summary>查看下载文件选择截图</summary>

<a class="doc-image-link" href="/images/ccswitch/ccswitchdownload.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitchdownload.jpg" alt="CC Switch 下载文件选择说明，macOS 选择 dmg 文件，Windows 选择 msi 文件">
</a>

</details>

## 2. 从管理后台导入配置

需要保留 ChatGPT 账号登录的，先按 [保留 ChatGPT 登录](/codex/chatgpt-login) 打开开关，再回来导入。

打开管理后台 **API 密钥**，找到 GPT 分组的 Key，点击 **导入到 CCS**。浏览器询问是否打开 CC Switch 时点击 **打开**，确认应用类型是 `Codex`，点击 **导入**。

导入后 `MyAutoAPI` 右侧显示 **使用中**。

<details>
<summary>查看导入截图</summary>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入1.jpg" alt="在 API 密钥页面点击导入到 CCS 并允许浏览器打开 CC Switch">
</a>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入2.jpg" alt="核对 Codex 的 MyAutoAPI 供应商配置并点击导入">
</a>

<a class="doc-image-link" href="/images/ccswitch/CCSWITCH导入3.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/CCSWITCH导入3.jpg" alt="确认 MyAutoAPI 地址正确并显示使用中">
</a>

</details>

## 3. 重新打开 Codex 测试

彻底退出 Codex，重新打开，新建任务，发送 `测试`。能正常回复就接入完成，之后可以关闭 CC Switch。

切回官方账号：在 CC Switch 的官方供应商右侧点击 **启用**，再彻底退出并重新打开 Codex。

## 接入失败

| 遇到的问题 | 处理方法 |
| --- | --- |
| 点击 **导入到 CCS** 没有反应 | 先打开 CC Switch，再点一次，浏览器询问时允许打开外部应用 |
| 应用类型不是 `Codex` | 取消导入，换一个 GPT 分组的 Key 重新导入 |
| 没有显示 **使用中** | 在 `MyAutoAPI` 右侧点击 **启用** |
| `401 Unauthorized` | 在后台确认 Key 未删除、未禁用，重新导入 |
| `403 Forbidden` | 在 Codex 里换成当前 Key 分组支持的模型，见 [模型价格](/models) |

其他报错看 [报错与踩坑](/errors/)。
