---
title: 给 Codex 导入国产模型
sidebarTitle: 导入国产模型
---

# 给 Codex 导入国产模型

这个脚本把 10 个国产模型加入 Codex 的模型选择器，适合已经接入 MyAutoAPI、想在 GPT 和国产模型之间切换的用户。脚本往本机 `.codex/config.toml` 和模型列表里添加国产模型，原有 GPT 模型、接口和 Key 保留。

## 1. 一键导入

导入成功后会自动关闭 ChatGPT 和 Codex，运行前先结束正在进行的任务。

Windows 按 `Win + X`，点击 **终端** 或 **Windows PowerShell**；Mac 和 Linux 打开 **终端**。粘贴对应命令运行：

::: code-group

```powershell [Windows]
irm https://docs.myautoapi.com/install/codex-models.ps1 | iex
```

```sh [Mac / Linux]
curl -fsSL https://docs.myautoapi.com/install/codex-models.sh | sh
```

:::

Linux 需要已安装 Node.js 16 以上。

<details>
<summary>不想用 PowerShell？</summary>

加入 <span style="color: #d93025;"><strong>QQ 群：685861462</strong></span>，在群文件下载 `中转站快速导入工具.zip`，解压后双击 `3.Codex国产模型导入`（Windows 在 `Windows-x64` 文件夹，Mac 在 `Mac-Apple芯片` 文件夹）。

</details>

## 2. 切换分组并选择模型

在管理后台 **API 密钥** 页面，点击 Codex 所用 Key 旁的 **选择分组**，按下表切到要用的分组。价格看 [模型价格](/models)。

| 分组 | 模型 ID |
| --- | --- |
| DeepSeek 分组 | `deepseek-flash`（DeepSeek V4.1 Flash） |
| 国产之光 | `glm-5.3`、`glm-5.3-flash` |
| Qwen 千问分组 | `qwen3.8-max`、`qwen3.8-flash` |
| Kimi 分组 | `kimi-k3` |
| MiniMax 分组 | `minimax-m3` |
| 腾讯混元分组 | `hy4-preview` |
| 小米 MiMo 分组 | `mimo-v2.6-pro`、`mimo-v2.6-flash` |

切到国产分组后，这个 Key 暂时不能用 GPT 模型；要用 GPT 时把分组切回来。想同时用多个国产分组、不来回切，用 [Qwen Code](/domestic/qwen-code)。

重新打开 Codex，新建任务，在模型选择器里选同一分组的模型，发送 `测试`。能正常回复就导入完成。

## 更新或卸载

更新模型列表：重新运行第 1 步的命令。双击版从群文件下载最新安装包重新运行。

本次已加入 `gpt-6.1-terra`。模型列表保存在本机；后续官方新增 GPT 时，使用更新后的脚本或最新双击版重新导入。

卸载只移除本工具导入的国产模型，GPT 和其它配置保留。卸载后会自动关闭 ChatGPT 和 Codex，重新打开、新建任务即可。

::: code-group

```powershell [Windows]
& ([scriptblock]::Create((irm https://docs.myautoapi.com/install/codex-models.ps1))) -Uninstall
```

```sh [Mac / Linux]
curl -fsSL https://docs.myautoapi.com/install/codex-models.sh | sh -s -- --uninstall
```

:::
