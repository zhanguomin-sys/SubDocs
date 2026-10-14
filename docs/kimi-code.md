# Kimi Code 接入 Kimi K3

Kimi Code 是月之暗面推出的编程 Agent，可以读取本机项目、修改文件和执行任务，适合在浏览器界面处理代码工作。

本文只讲 Windows：打开 PowerShell，安装 Kimi Code 后运行 `kimi web`，再在网页中选择项目并接入 MyAutoAPI 的 `kimi-k3`。无需手动修改文件；Kimi Code 会把配置保存到 `C:\Users\你的用户名\.kimi-code\config.toml`。

## 1. 申请并复制 API Key

按[快速开始](/quick-start)创建 MyAutoAPI API Key，并选择正确的分组。配置到第 4 步时，再回到网站点击复制按钮。

<details>
<summary>查看图片：复制 MyAutoAPI API Key</summary>

![复制以 sk- 开头的 MyAutoAPI API Key](/images/kimi/kimi0.jpg)

</details>

## 2. 打开 PowerShell 并安装 Kimi Code

按下 `Windows` 键，搜索并打开 **PowerShell**，运行官方安装命令：

```powershell
irm https://code.kimi.com/kimi-code/install.ps1 | iex
```

安装完成后继续使用 PowerShell。

## 3. 打开 Kimi Web

在 PowerShell 运行：

```powershell
kimi web
```

浏览器会自动打开 Kimi Code。保持 PowerShell 窗口运行，项目稍后直接在网页中选择。

首次打开时，选择 **添加自定义供应商**。已经进入主界面时，打开左下角 **设置 -> 供应商 -> 添加供应商**。

<details>
<summary>查看图片：首次选择自定义供应商</summary>

![首次选择添加自定义供应商](/images/kimi/kimi1.jpg)

</details>

<details>
<summary>查看图片：从设置添加供应商</summary>

![从设置打开供应商配置](/images/kimi/kimi2.jpg)

</details>

## 4. 添加 MyAutoAPI

选择 **手动添加**。

<details>
<summary>查看图片：选择手动添加</summary>

![选择手动添加供应商](/images/kimi/kimi3.png)

</details>

按下面填写：

| 字段 | 填写 |
| --- | --- |
| 名称 | `MyAutoAPI中转站` |
| API 协议 | `OpenAI` |
| API Key | 回到第 1 步复制，以 `sk-` 开头 |
| Base URL | `https://api.myautoapi.com/v1` |
| 模型 ID | `kimi-k3` |
| 上下文 | `1048576` |
| 显示名 | 可以留空 |

点击 **保存**。Base URL 必须保留结尾的 `/v1`；必须选择第一步对应的 Kimi 分组。

<details>
<summary>查看图片：填写 Kimi K3 配置</summary>

![填写 MyAutoAPI 和 Kimi K3](/images/kimi/kimi4.jpg)

</details>

## 5. 选择项目并测试

关闭设置，在主界面点击 **选择文件夹**，选择要交给 Kimi Code 处理的项目。

新建对话，在输入框右下角选择 `kimi-k3`，发送：

```text
测试
```

界面显示 `kimi-k3` 且能正常回复，说明接入成功。不要根据模型自述判断实际模型。

<details>
<summary>查看图片：选择 Kimi K3 并测试</summary>

![选择 Kimi K3 并发送测试消息](/images/kimi/kimi5.jpg)

</details>

## 常见错误

| 现象 | 处理 |
| --- | --- |
| PowerShell 找不到 `kimi` | 关闭并重新打开 PowerShell，再运行 `kimi web` |
| `kimi web` 不支持 | 运行 `kimi upgrade`，完成后重新打开 PowerShell |
| `401 Unauthorized` | 重新复制 MyAutoAPI API Key |
| `model not found` | 确认当前 Key 所属分组支持 `kimi-k3` |
| 网页无法继续使用 | 确认运行 `kimi web` 的 PowerShell 窗口没有关闭 |
