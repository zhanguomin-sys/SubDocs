---
sidebarTitle: 保留 ChatGPT 登录
---

# 保留 ChatGPT 登录

在 CC Switch 里打开“保留官方登录”开关后再导入 MyAutoAPI：Codex 请求走 MyAutoAPI，ChatGPT 的登录状态、插件入口和手机连接都保留。不需要改文件。

::: warning
打开后，原 ChatGPT 账号的 Codex 对话不再显示（本地文件还在）。打开 **统一 Codex 会话历史** 能看到列表，但基本无法继续旧对话；需要保留的记录先备份。
:::

## 1. 打开保留登录开关

还没装 CC Switch 时，先按 [CC Switch 接入](/codex-cc-switch) 第 1 步安装。

1. 打开 CC Switch，点击左上角的配置图标。
2. 进入 **通用**，打开 **非接管切换时保留官方登录**。
3. 要看原账号的 Codex 会话列表，再打开 **统一 Codex 会话历史**。

<details>
<summary>查看设置截图</summary>

<a class="doc-image-link" href="/images/ccswitch/cc保持登陆1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/cc保持登陆1.jpg" alt="在 CC Switch 左上角点击配置图标">
</a>

<a class="doc-image-link" href="/images/ccswitch/cc保持登陆2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/cc保持登陆2.jpg" alt="在 CC Switch 通用设置中打开保留官方登录开关">
</a>

</details>

## 2. 导入 MyAutoAPI

回到 [CC Switch 接入](/codex-cc-switch#_2-从管理后台导入配置) 第 2 步，从后台导入配置。

不用 CC Switch、手动改配置文件时，Key 的写法看 [Key 写在哪个文件](/codex/manual-config#codex-auth-modes)。

## 手机连接

保留 ChatGPT 登录后，手机端可以在 ChatGPT 的 **Remote** 标签查看电脑上的 Codex 任务。在电脑上：

1. 左侧点击 **连接**。
2. 在 **控制此电脑** 页面点击 **添加**。
3. 手机登录 ChatGPT 后扫描二维码。
4. 长期使用时，打开 **使此电脑保持唤醒**。
5. 不再使用某台设备时，在设备列表点击 **撤销访问权限**。

手机连不上时，回到 **连接** 页面重新点 **添加** 并扫码。

<details>
<summary>查看连接设置截图</summary>

<a class="doc-image-link" href="/images/连接/手机连接.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/手机连接.jpg" alt="ChatGPT 桌面应用中 Codex 的连接设置和手机扫码入口">
</a>

</details>
