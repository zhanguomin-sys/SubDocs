---
sidebarTitle: Open WebUI
---

# Open WebUI 接入

Open WebUI 是自建的网页聊天界面，给自己或小团队提供浏览器聊天入口。在管理后台添加 `MyAutoAPI` 连接后，聊天请求走 `https://api.myautoapi.com/v1`。

## 1. 启动 Open WebUI

已经有可用的 Open WebUI 时跳到第 2 步。没有时用 Docker 启动：

```bash
docker run -d -p 3000:8080 -v open-webui:/app/backend/data \
  -e WEBUI_SECRET_KEY="请换成一串随机密钥" \
  --name open-webui --restart always \
  ghcr.io/open-webui/open-webui:main
```

打开 `http://localhost:3000`，第一次进入时创建管理员账号。`WEBUI_SECRET_KEY` 固定下来，重建容器后登录状态不会失效。

## 2. 添加 MyAutoAPI 连接

进入 **Admin Settings -> Connections**，在 OpenAI / OpenAI API 区域新增连接：

| 字段 | 填写 |
| --- | --- |
| Name / 名称 | `MyAutoAPI` |
| URL / Base URL | `https://api.myautoapi.com/v1`，只填到 `/v1` |
| API Key | MyAutoAPI API Key |
| Model IDs (Filter) | 当前 Key 分组可用模型，例如 `claude-fable-5-1`、`gemini-3.8-flash`、`grok-4.7`；Claude 模型要用 **CC MAX 外接分组** 的 Key |

模型列表自动出现时可以不填 `Model IDs (Filter)`；列表为空或太多时，在这里手动添加模型，保存连接。

## 3. 测试

回到聊天页面，新建对话，在模型选择器里选 MyAutoAPI 连接下的模型，发送 `测试`。能正常回复就接入完成。

## 进阶：用环境变量预置

维护 Docker / Compose 部署脚本时，可以用环境变量预置 MyAutoAPI。这两个变量只在首次启动（数据卷为空）时写入；已经运行过的实例，在 **Admin Settings -> Connections** 里改，或在命令里加 `-e ENABLE_PERSISTENT_CONFIG=False` 让环境变量优先（此时后台改的设置重启后失效）。用了环境变量，就不要在管理后台再维护另一套连接。

```bash
docker run -d -p 3000:8080 -v open-webui:/app/backend/data \
  -e WEBUI_SECRET_KEY="请换成一串随机密钥" \
  -e OPENAI_API_BASE_URL="https://api.myautoapi.com/v1" \
  -e OPENAI_API_KEY="sk-你的MyAutoAPI_API_Key" \
  --name open-webui --restart always \
  ghcr.io/open-webui/open-webui:main
```

已经有同名容器时，先停掉并移除旧容器，再用上面的命令重新创建。数据卷 `open-webui` 保留不动，删了会清空 Open WebUI 数据。

```bash
docker stop open-webui
docker rm open-webui
```

## 排查

| 现象 | 处理 |
| --- | --- |
| 页面看不到模型 | 确认连接已启用，在 `Model IDs (Filter)` 添加模型后刷新页面 |
| 连接验证失败 | 先手动添加模型，再到聊天页测试 |
| `401 Unauthorized` | 重新复制 API Key，填进 MyAutoAPI 连接 |
| `403 Forbidden` | 换成 Key 分组支持的模型；仍失败看 [报错与踩坑](/errors/) |
| 路径错误 | Base URL 改回 `https://api.myautoapi.com/v1` |
| 改了环境变量但连接没变 | 数据库里的旧配置优先；在后台 Connections 里改，或加 `-e ENABLE_PERSISTENT_CONFIG=False` 后重建容器 |
| 要不要填 `host.docker.internal` | 不用。它只用于连接本机模型服务，连 MyAutoAPI 不需要 |
