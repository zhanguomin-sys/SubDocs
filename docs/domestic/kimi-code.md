---
sidebarTitle: Kimi Code
---

# Kimi Code 接入 Kimi K3

Kimi Code 是月之暗面推出的编程 Agent，能读取本机项目、修改文件和执行任务，在浏览器界面里处理代码工作。本文在 Windows 上安装 Kimi Code，运行 `kimi web` 后在网页里接入 MyAutoAPI 的 `kimi-k3`；不用手动改文件，配置自动保存到 `C:\Users\你的用户名\.kimi-code\config.toml`。

需要一个 **Kimi 分组** 的 MyAutoAPI API Key，创建方法看 [快速开始](/quick-start)。

## 1. 安装 Kimi Code

按下 `Windows` 键，搜索并打开 **PowerShell**，运行官方安装命令：

```powershell
irm https://code.kimi.com/kimi-code/install.ps1 | iex
```

## 2. 打开 Kimi Web

在同一个 PowerShell 窗口运行：

```powershell
kimi web
```

浏览器会自动打开 Kimi Code。使用期间保持这个 PowerShell 窗口开着。

首次打开时选择 **添加自定义供应商**；已经进入主界面时，打开左下角 **设置 -> 供应商 -> 添加供应商**。

<details>
<summary>查看图片：添加供应商入口</summary>

![首次选择添加自定义供应商](/images/kimi/kimi1.jpg)

![从设置打开供应商配置](/images/kimi/kimi2.jpg)

</details>

<a id="_4-添加-myautoapi"></a>

## 3. 添加 MyAutoAPI

选择 **手动添加**，按下表填写后点击 **保存**：

| 字段 | 填写 |
| --- | --- |
| 名称 | `MyAutoAPI中转站` |
| API 协议 | `OpenAI` |
| API Key | Kimi 分组的 MyAutoAPI API Key，以 `sk-` 开头 |
| Base URL | `https://api.myautoapi.com/v1`，保留结尾的 `/v1` |
| 模型 ID | `kimi-k3` |
| 上下文 | `1048576` |
| 显示名 | 留空 |

<details>
<summary>查看图片：复制 Key、手动添加、填写配置</summary>

![复制以 sk- 开头的 MyAutoAPI API Key](/images/kimi/kimi0.jpg)

![选择手动添加供应商](/images/kimi/kimi3.png)

![填写 MyAutoAPI 和 Kimi K3](/images/kimi/kimi4.jpg)

</details>

## 4. 选择项目并测试

关闭设置，在主界面点击 **选择文件夹**，选择要交给 Kimi Code 处理的项目。

新建对话，在输入框右下角选择 `kimi-k3`，发送 `测试`。界面显示 `kimi-k3` 且能正常回复，就接入完成；不要凭模型自述判断实际模型。

<details>
<summary>查看图片：选择 Kimi K3 并测试</summary>

![选择 Kimi K3 并发送测试消息](/images/kimi/kimi5.jpg)

</details>

## 常见错误

| 现象 | 处理 |
| --- | --- |
| PowerShell 找不到 `kimi` | 关闭并重新打开 PowerShell，再运行 `kimi web` |
| 提示不支持 `kimi web` | 运行 `kimi upgrade`，完成后重新打开 PowerShell |
| `401 Unauthorized` | 重新复制 MyAutoAPI API Key |
| `model not found` | 换成 Kimi 分组的 Key，模型 ID 填 `kimi-k3` |
| 网页无法继续使用 | 运行 `kimi web` 的 PowerShell 窗口被关了，重新运行 `kimi web` |
