---
sidebarTitle: Claude Desktop
---

# Claude Desktop 接入

Claude Desktop 是 Anthropic 的桌面应用，包含 Chat、Cowork 和 Code 等功能。Windows 用 MyAutoAPI 接入助手一键连接；Mac 在 CC Switch 里手动添加一次（后台的 **导入到 CCS** 不能直接导入 Desktop）；两样都不用时，在应用内配置 Gateway。

<a id="claude-desktop-helper"></a>

## Windows 推荐：用接入助手一键连接

适用于从 [Claude 下载页](https://claude.com/download) 安装的 Claude Desktop。

1. [下载 Windows 接入助手](/download/myautoapi-quick-connect-windows-x64.exe)，打开后点右上角 **Claude**。还没装 Claude Desktop 时，点卡片里的 **官网下载**；打不开官网就进 QQ 群在群文件下载。
2. 后台 **API 密钥** 页复制一把 Claude 分组的密钥，粘贴到助手输入框，点 **一键连接 Claude**。助手会先关闭 Claude，再写入这把密钥能用的全部 Claude 模型（Fable、Opus、Sonnet、Haiku）。
3. 重新打开 Claude Desktop，新建对话，发送 `测试`。能回复就接入完成了。

电脑装了 Claude Code CLI 时，助手会问要不要一起接入，点 **是** 即可。余额为 0 时助手会提示先充值。要改回官方账号，在助手 Claude 页点右下角 **恢复官方登录**。

<a id="claude-desktop-ccswitch"></a>

## Mac 推荐：用 CC Switch 配置

还没装 CC Switch 时，先按 [CC Switch 安装](/codex-cc-switch#_1-安装-cc-switch) 装好。

1. 打开 CC Switch，点击顶部的 **Claude Desktop** 图标，再点击右上角橙色 `+`。必须选 **Claude Desktop**，选成 Claude Code 配置不会写进 Desktop。
2. 保持顶部选中 **Claude 供应商**，点击 **自定义配置**，再点击右下角 **添加**。
3. 在 **编辑供应商** 里填写四项，其它保持默认，点击 **保存**：

| 字段 | 填写 |
| --- | --- |
| 供应商名称 | `MyAutoAPI.com` |
| 官网链接 | `https://myautoapi.com` |
| API Key | MyAutoAPI API Key |
| 请求地址 | `https://api.myautoapi.com` |

**完整 URL** 开关保持关闭。两个地址都不加 `/v1`，末尾不加 `/`，不用填模型名。

4. 在 Claude Desktop 的供应商列表里，点击 `MyAutoAPI.com` 右侧的 **启用**。
5. 彻底退出 Claude Desktop，重新打开，新建对话，发送 `测试`。能正常回复后，在模型选择器里选 Claude 模型。

<details>
<summary>查看 CC Switch 截图</summary>

<a class="doc-image-link" href="/images/ccswitch/ccswitch1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch1.jpg" alt="在 CC Switch 选择 Claude Desktop 并点击右上角加号">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch2.jpg" alt="在 Claude Desktop 的 Claude 供应商中选择自定义配置并点击添加">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch3.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch3.jpg" alt="在 Claude Desktop 的 CC Switch 供应商配置中填写四个字段">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch4.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch4.jpg" alt="在 Claude Desktop 的 CC Switch 供应商列表启用 MyAutoAPI.com">
</a>

</details>

<a id="claude-desktop-gateway"></a>

## 备用：在应用内配置 Gateway

### 1. 打开第三方推理设置

从 [Claude 下载页](https://claude.com/download) 安装 Claude Desktop。首次打开先不要登录或创建 Anthropic 账号。

Mac 从系统菜单栏、Windows 从登录页左上角菜单进入：

1. 点击 **Help → Troubleshooting → Enable Developer Mode**。
2. 点击 **Developer → Configure third-party inference**。

<a id="_2-填写-myautoapi-gateway"></a>

### 2. 填写连接信息

在 **Connection** 中填写：

| 字段 | 填写 |
| --- | --- |
| Inference provider | `Gateway` |
| Gateway base URL | `https://api.myautoapi.com`，不加 `/v1` |
| Credential kind | `Static API key` |
| Gateway API key | MyAutoAPI API Key |
| Gateway auth scheme | `Bearer` |

### 3. 填写模型列表

向下找到 **Models**（有左侧导航时直接点 **Models**），关闭 **Model discovery**，在 **Model list** 里手动添加模型。以 `claude-opus-5-5` 为例：

| 字段 | 填写 |
| --- | --- |
| Model ID | `claude-opus-5-5`；换成当前 Key 分组可用的完整模型 ID |
| Display name | `Claude Sonnet 4.6`，只用于显示，留空会自动生成 |
| Tier alias | `sonnet` |
| Default for tier | 开启 |
| Offer 1M-context variant | 保持关闭 |

**Model ID 必须和 MyAutoAPI 的模型 ID 完全一致**，不能只填 `sonnet` 或显示名称；模型 ID 看 [模型价格](/models)。

加更多模型时每个单独添加一条，Tier alias 按系列选：Sonnet 填 `sonnet`，Opus 填 `opus`，Haiku 填 `haiku`。同一系列只把一个模型设为 **Default for tier**。列表第一条是默认模型，最常用的放第一条。

### 4. 测试连接并应用

1. 回到 **Connection → Gateway credentials**，点击 **Test connection**，看到绿色成功提示。
2. 点击 **Apply Changes**（旧版叫 **Apply locally**），按提示重启 Claude Desktop，在登录页选择使用第三方配置进入。
3. 新建对话，在模型选择器里选刚配置的模型，发送 `测试`。

## 排查

| 遇到的问题 | 处理方法 |
| --- | --- |
| 找不到第三方推理入口 | 回到登录页，按 **Help → Troubleshooting → Enable Developer Mode** 开启后再进 **Developer** 菜单 |
| 应用配置后仍显示普通登录页 | 彻底退出并重新打开，选择第三方配置入口 |
| Test connection 提示模型不存在或不可用 | 回到 **Models → Model list**，核对 Model ID 完整拼写，换成 Key 分组支持的 Claude 模型后重新测试 |
| 模型选择器为空 | 在 **Model list** 添加完整模型 ID，应用配置后重新打开 |
| `401 Unauthorized` | 重新复制 API Key；Gateway 方式确认 auth scheme 是 `Bearer` |
| 请求地址错误 | 地址只填 `https://api.myautoapi.com`，不加 `/v1` |

其他报错看 [报错与踩坑](/errors/)。

## 切回默认服务

- CC Switch：在 Claude Desktop 的供应商列表里启用原来的官方供应商。
- Gateway：回到登录页，选择 Anthropic 登录入口。
