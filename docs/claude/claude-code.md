---
sidebarTitle: Claude Code
---

# Claude Code 接入

Claude Code 是 Anthropic 的终端编程助手。用 CC Switch 把 MyAutoAPI 导入 Claude Code 后，Claude Code 的请求走 MyAutoAPI。还没装 CC Switch 时，先按 [CC Switch 安装](/codex-cc-switch#_1-安装-cc-switch) 装好。

## 安装 Claude Code

已经能运行 `claude` 的跳过这一节。

::: code-group

```powershell [Windows]
irm https://claude.ai/install.ps1 | iex
```

```bash [Mac / Linux / WSL]
curl -fsSL https://claude.ai/install.sh | bash
```

:::

装完关闭并重新打开 PowerShell 或终端，运行 `claude --version`，能看到版本号就装好了。

<a id="ccswitch-claude-code-import"></a>
<a id="claude-code-cli-ccswitch"></a>

## 1. 从后台一键导入

1. 在管理后台 **API 密钥** 页面，找到 Claude 分组的 Key，点击 **导入到 CCS**。
2. 浏览器询问是否打开 CC Switch 时，点击 **打开**。
3. 确认应用类型是 **Claude Code**，点击 **导入**。
4. 导入后，CC Switch 里 `MyAutoAPI` 右侧显示 **使用中**。

后台没有弹出导入，或要自己填字段，看下面的 [手动添加](#ccswitch-claude-code-manual)。

<a id="在-claude-code-中测试"></a>

## 2. 重新打开 Claude Code 测试

彻底退出 Claude Code，重新打开终端运行 `claude`，在新对话里发送 `测试`。能正常回复就接入完成。

切换模型在 Claude Code 里输入 `/model`，不用新开对话；CC Switch 里没有模型字段。测试成功后可以关闭 CC Switch。不要开启 CC Switch 的 **Proxy**、本地代理或本地路由。

<a id="ccswitch-claude-code-manual"></a>

## 手动添加：用 CC Switch

1. 打开 CC Switch，点击顶部的 **Claude Code** 图标（中间列表出现 `Claude Official` 和 `default`），再点击右上角橙色 `+`。选错图标，配置会写到别的工具。
2. 保持顶部选中 **Claude 供应商**，点击 **自定义配置**，再点击右下角 **添加**。不要选其它站点的预设或 **统一供应商**。
3. 在 **编辑供应商** 里只填写四项，其它保持默认，点击 **保存**：

| 字段 | 填写 |
| --- | --- |
| 供应商名称 | `MyAutoAPI.com` |
| 官网链接 | `https://myautoapi.com` |
| API Key | MyAutoAPI API Key |
| 请求地址 | `https://api.myautoapi.com` |

**完整 URL** 开关保持关闭。两个地址都不加 `/v1`，末尾不加 `/`，不用填模型名。

4. 在 Claude Code 的供应商列表里，点击 `MyAutoAPI.com` 右侧的 **启用**，然后按第 2 步重新打开测试。

<details>
<summary>查看手动添加截图</summary>

<a class="doc-image-link" href="/images/ccswitch/ccswitch1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch1.jpg" alt="在 CC Switch 选择 Claude Code 并点击右上角加号">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch2.jpg" alt="在 Claude 供应商中选择自定义配置并点击添加">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch3.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch3.jpg" alt="填写 MyAutoAPI 供应商名称、官网链接、API Key 和请求地址">
</a>

<a class="doc-image-link" href="/images/ccswitch/ccswitch4.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch4.jpg" alt="在 CC Switch 供应商列表启用 MyAutoAPI.com">
</a>

</details>

<a id="claude-code-cli-manual"></a>

## 不用 CC Switch：设置环境变量

在终端里设置后，从同一个终端启动 `claude`。只对当前终端有效，关掉终端后下次要重新设置。

::: code-group

```powershell [Windows]
$env:ANTHROPIC_BASE_URL="https://api.myautoapi.com"
$env:ANTHROPIC_AUTH_TOKEN="你的 MyAutoAPI API Key"
claude
```

```bash [Mac / Linux / WSL]
export ANTHROPIC_BASE_URL="https://api.myautoapi.com"
export ANTHROPIC_AUTH_TOKEN="你的 MyAutoAPI API Key"
claude
```

:::

不要设置 `ANTHROPIC_MODEL` 或 `ANTHROPIC_DEFAULT_*_MODEL`，也不要填 GPT 模型名；进入后用 `/model` 选 Claude 模型。

## 排查

| 遇到的问题 | 处理方法 |
| --- | --- |
| 导入窗口里的应用类型不是 Claude Code | 取消导入，换一个 Claude 分组的 Key 重新导入 |
| 保存后 Claude Code 没用新配置 | 确认 CC Switch 顶部选的是 Claude Code，`MyAutoAPI.com` 显示为使用中 |
| 设置环境变量后仍在用原来的服务 | 在执行设置的同一个终端里启动 `claude` |
| `401 Unauthorized` | 重新复制 API Key，填进 CC Switch 的 **API Key** 字段或 `ANTHROPIC_AUTH_TOKEN`，保存后重新启用 |
| 请求地址错误 | 地址只填 `https://api.myautoapi.com`，删掉 `/v1` 和末尾 `/` |
| `403 Forbidden` 或切换模型后没变化 | 用 `/model` 重新选择，确认 Key 所属分组支持这个模型 |

其他报错看 [报错与踩坑](/errors/)。

## 切回默认服务

- CC Switch：在 Claude Code 的供应商列表里启用 `Claude Official`。
- 环境变量：关闭当前终端，或执行 `unset ANTHROPIC_BASE_URL ANTHROPIC_AUTH_TOKEN`（Windows 执行 `Remove-Item Env:ANTHROPIC_BASE_URL, Env:ANTHROPIC_AUTH_TOKEN`）。
