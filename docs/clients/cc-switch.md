# CC Switch 接入 Claude Code CLI

CC Switch 可以管理 Claude Code CLI 和 Claude Desktop 的供应商。MyAutoAPI 后台的 **导入到 CCS** 可以一键导入 Claude Code CLI；Claude Desktop 不能使用这条后台导入，但可以按 [Claude Desktop 的 CC Switch 主路径](./claude-code-desktop#claude-desktop-ccswitch) 手动配置。

还没有安装 CC Switch 时，先进入 [CC Switch 安装步骤](/codex-cc-switch#_1-安装-cc-switch)；安装完成后回到本页继续。

<a id="ccswitch-claude-code-import"></a>
## 从后台一键导入 Claude Code CLI

### 1. 从 API 密钥页面发起导入

回到 MyAutoAPI 管理后台 **API 密钥**，找到要使用的 Key，点击 **导入到 CCS**。

浏览器询问是否打开 CC Switch 时，点击 **打开**。确认应用类型是 **Claude Code**，再点击 **导入**。

### 2. 确认导入并启动 Claude Code

导入完成后，确认 CC Switch 中的 `MyAutoAPI` 右侧显示 **使用中**。彻底退出 Claude Code，重新打开并新建一个对话或任务。

完成后进入 [在 Claude Code 中测试](#在-claude-code-中测试)。模型在 Claude Code 内选择，不在 CC Switch 的供应商页面填写；可用模型以 Claude Code 模型选择器和当前 API Key 所属分组为准。

如果后台没有弹出 CC Switch 导入，或你需要手动填写字段，继续看下面的 [手动添加 Claude Code 供应商](#ccswitch-claude-code-manual)。

<a id="ccswitch-claude-code-manual"></a>
## 手动添加 MyAutoAPI 到 Claude Code CLI

### 1. 选择 Claude Code 并新建供应商

打开 CC Switch，点击顶部的 **Claude Code** 图标。中间列表出现 `Claude Official` 和 `default`，说明当前管理的是 Claude Code 配置。

点击右上角橙色 `+`，进入新增供应商页面。

<a class="doc-image-link" href="/images/ccswitch/ccswitch1.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch1.jpg" alt="在 CC Switch 选择 Claude Code 并点击右上角加号">
</a>

顶部图标代表不同的目标工具。选错图标会把配置写到其它工具，Claude Code 不会读取。

### 2. 选择 Claude 供应商的自定义配置

保持顶部选中 **Claude 供应商**，在预设供应商列表中点击 **自定义配置**，再点击右下角 **添加**。

<a class="doc-image-link" href="/images/ccswitch/ccswitch2.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch2.jpg" alt="在 Claude 供应商中选择自定义配置并点击添加">
</a>

这里不使用其它站点的预设，也不选择 **统一供应商**。MyAutoAPI 的 Claude Code 配置只需要在这个自定义配置里填写一次。

### 3. 填写四个字段并保存

进入 **编辑供应商** 后，只填写下面四项：

| 字段 | 填写内容 |
| --- | --- |
| 供应商名称 | `MyAutoAPI.com` |
| 官网链接 | `https://api.myautoapi.com` |
| API Key | MyAutoAPI API Key |
| 请求地址 | `https://api.myautoapi.com` |

<a class="doc-image-link" href="/images/ccswitch/ccswitch3.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch3.jpg" alt="填写 MyAutoAPI 供应商名称、官网链接、API Key 和请求地址">
</a>

**完整 URL** 开关保持关闭。官网链接和请求地址都不要添加 `/v1`，末尾也不要添加 `/`；其它字段保持默认，不需要填写模型名。

填写完成后，点击右下角 **保存**。

### 4. 启用 MyAutoAPI

回到 Claude Code 的供应商列表，找到刚保存的 `MyAutoAPI.com`，点击右侧 **启用**。

<a class="doc-image-link" href="/images/ccswitch/ccswitch4.jpg" target="_blank" rel="noopener">
  <img src="/images/ccswitch/ccswitch4.jpg" alt="在 CC Switch 供应商列表启用 MyAutoAPI.com">
</a>

启用后，这一行会变成当前使用中的供应商。退出当前 Claude Code，再重新打开并新建一个对话或任务。

## 在 Claude Code 中测试

在新对话中发送：

```text
测试
```

需要切换模型时，在当前对话输入 `/model`。模型在 Claude Code 内选择，不在 CC Switch 的供应商页面填写；可用模型以 Claude Code 模型选择器和当前 API Key 所属分组为准。

测试成功后可以关闭 CC Switch，不需要后台运行。不要开启 **Proxy**、本地代理或本地路由功能。

这篇页面的主路径是 Claude Code CLI。需要配置 Claude Desktop 时，直接进入 [Claude Desktop 的 CC Switch 主路径](./claude-code-desktop#claude-desktop-ccswitch)；不要把本页的后台一键导入步骤当成 Desktop 配置步骤。

## 排查

| 现象 | 检查动作 |
| --- | --- |
| 保存后 Claude Code 没有使用新配置 | 回到 CC Switch 顶部确认选中的是 Claude Code，再确认 `MyAutoAPI.com` 已显示为当前使用中的供应商 |
| `401 Unauthorized` | 重新复制 MyAutoAPI API Key，替换 **API Key** 字段后保存并再次启用 |
| 请求地址错误 | **官网链接** 和 **请求地址** 都改成 `https://api.myautoapi.com`，删除 `/v1` 和末尾 `/` |
| 模型无法切换 | 回到 Claude Code 输入 `/model`，不要在 CC Switch 供应商页面寻找模型字段 |
| `403 Forbidden` | 确认当前 API Key 所属分组支持所选模型；仍失败时看 [报错与踩坑](/errors/) |
