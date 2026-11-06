# Open WebUI 接入

Open WebUI 是自建网页聊天界面，适合给自己或小团队提供浏览器聊天入口。在管理后台添加 `MyAutoAPI` 连接后，聊天请求会通过 `https://api.myautoapi.com/v1` 发出。

## 启动 Open WebUI

已经有可用的 Open WebUI 时，直接进入管理后台配置。

Docker 最小启动方式：

```bash
docker run -d -p 3000:8080 -v open-webui:/app/backend/data \
  -e WEBUI_SECRET_KEY="请换成一串随机密钥" \
  --name open-webui --restart always \
  ghcr.io/open-webui/open-webui:main
```

启动后打开：

```text
http://localhost:3000
```

第一次进入时创建管理员账号。`WEBUI_SECRET_KEY` 固定后，容器重建时登录状态更稳定。

## 添加连接

进入：

```text
Admin Settings -> Connections
```

在 OpenAI / OpenAI API 区域新增连接：

| 字段 | 填写 |
| --- | --- |
| Name / 名称 | `MyAutoAPI` |
| URL / Base URL | `https://api.myautoapi.com/v1` |
| API Key | MyAutoAPI API Key |
| Model IDs (Filter) | 当前 Key 分组可用模型，例如 `claude-fable-5`、`gemini-3.6-flash` 或 `grok-4.5` |

Base URL 只填到 `/v1`，不要追加其它路径。

## 添加模型

Open WebUI 自动出现模型时，选择可用模型测试。

模型列表为空或太多时，在 `Model IDs (Filter)` 手动添加当前 Key 分组可用模型，保存连接。

## 测试

回到聊天页面：

1. 打开模型选择器。
2. 选择 MyAutoAPI 连接下的模型。
3. 发送：

```text
测试
```

能正常回复，说明当前聊天已使用 MyAutoAPI 连接。

## 进阶部署

维护 Docker / Compose 部署脚本时，可以用环境变量预置 MyAutoAPI。使用环境变量后，不要在管理后台再维护另一套连接。

```bash
docker run -d -p 3000:8080 -v open-webui:/app/backend/data \
  -e WEBUI_SECRET_KEY="请换成一串随机密钥" \
  -e OPENAI_API_BASE_URL="https://api.myautoapi.com/v1" \
  -e OPENAI_API_KEY="sk-你的MyAutoAPI_API_Key" \
  --name open-webui --restart always \
  ghcr.io/open-webui/open-webui:main
```

已经创建过同名容器时，先停掉旧容器，再用新环境变量重新创建。不要删除数据卷，除非要清空 Open WebUI 数据。

```bash
docker stop open-webui
docker rm open-webui
```

## 排查

| 现象 | 处理 |
| --- | --- |
| 页面看不到模型 | 确认连接已启用，在 `Model IDs (Filter)` 添加模型并刷新页面 |
| 连接验证失败 | 先手动添加模型，再到聊天页测试 |
| `401 Unauthorized` | 重新复制 API Key，确认填在 MyAutoAPI 连接里 |
| `403 Forbidden` | 确认 Key 分组支持当前模型；仍失败时看 [报错与踩坑](/errors/) |
| 路径错误 | Base URL 改回 `https://api.myautoapi.com/v1` |
| 环境变量和后台配置冲突 | 只保留一种配置方式，重建容器后再测试 |
| Docker 访问本机模型 | 本机模型服务才需要 `host.docker.internal`；连接 MyAutoAPI 不需要 |
