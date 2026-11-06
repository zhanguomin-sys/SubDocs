# 保留 ChatGPT 登录

CC Switch 是图形化配置切换工具，这里用它打开 ChatGPT 桌面应用的 Codex 保持登录开关。已经安装 ChatGPT 桌面应用、并且有 ChatGPT 账号时，按本文操作；配置完成后，Codex 请求通过 MyAutoAPI 发出，同时保留 ChatGPT 登录状态、插件入口和手机连接。本文不需要手动修改文件。

::: warning 重要提示
配置后，原 ChatGPT 账号的 Codex 对话记录默认不会显示在 app 上（本地文件还在）。在 CC Switch 设置中打开 **统一 Codex 会话历史** 可以显示历史列表，但基本无法继续原来的旧对话。需要保留记录时，请提前备份或连接后让 AI 整理。
:::

## 用 CC Switch 保留登录

还没有安装 CC Switch 时，先按 [Codex CC Switch 接入](/codex-cc-switch) 的第 1 步安装，安装完成后回到本页继续。

1. 打开 CC Switch，点击左上角的配置图标。

<a class="doc-image-link" href="/images/ccswitch/cc保持登陆1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/cc保持登陆1.jpg" alt="在 CC Switch 左上角点击配置图标">
</a>

2. 进入 **通用** 设置，打开 **非接管切换时保留官方登录**。
3. 需要保留原 ChatGPT 账号的 Codex 会话列表时，再打开 **统一 Codex 会话历史**；这个开关基本无法继续原来的旧对话。

<a class="doc-image-link" href="/images/ccswitch/cc保持登陆2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/cc保持登陆2.jpg" alt="在 CC Switch 通用设置中打开保留官方登录开关">
</a>

4. 完成设置后，回到 [Codex CC Switch 接入](/codex-cc-switch#_2-从管理后台导入配置)，从第 2 步“从管理后台导入配置”继续导入 MyAutoAPI。

不能使用 CC Switch、需要手动修改配置文件时，API Key 的位置和 `requires_openai_auth` 的写法看 [Codex 手动接入](/clients/codex-manual-config#codex-auth-modes)。

## 手机连接

ChatGPT 桌面应用中的 Codex 保持 ChatGPT 登录后，手机端可以在 ChatGPT 的 **Remote** 标签查看受支持的桌面任务。电脑端连接设置仍按图中三个箭头操作：

<a class="doc-image-link" href="/images/连接/手机连接.jpg" target="_blank" rel="noopener">
  <img src="/images/连接/手机连接.jpg" alt="ChatGPT 桌面应用中 Codex 的连接设置和手机扫码入口">
</a>

1. 左侧点击 **连接**。
2. 在 **控制此电脑** 页面点击 **添加**。
3. 手机端登录 ChatGPT 后扫描二维码。
4. 长期使用时，打开 **使此电脑保持唤醒**。
5. 不再使用某台设备时，在设备列表里点击 **撤销访问权限**。

手机端连不上时，回到 **连接** 页面重新点击 **添加**，手机端登录 ChatGPT 后重新扫码。
