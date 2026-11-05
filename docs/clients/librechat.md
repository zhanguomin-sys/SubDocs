---
sidebarTitle: LibreChat
---

# LibreChat 接入

LibreChat 是自部署的网页聊天站，给自己或团队提供浏览器聊天入口。在项目里加一个 MyAutoAPI 自定义 endpoint 后，聊天请求走 `https://api.myautoapi.com/v1`。要改的是 LibreChat 项目根目录的 `.env`、`librechat.yaml` 和 `docker-compose.override.yml`。

## 1. 启动 LibreChat

已经能打开 LibreChat 时跳到第 2 步。没有时用 Docker 启动：

```bash
git clone https://github.com/danny-avila/LibreChat.git
cd LibreChat
cp .env.example .env
docker compose up -d
```

启动后访问 `http://localhost:3080`。

## 2. 写入 API Key

在项目根目录的 `.env` 里添加一行，换成你的 MyAutoAPI API Key。不要把真实 Key 提交到公开仓库。

```dotenv
MYAUTOAPI_API_KEY=替换为你的MyAutoAPI_API_Key
```

## 3. 添加 MyAutoAPI endpoint

在项目根目录创建或编辑 `librechat.yaml`：

```yaml
version: 1.3.13

endpoints:
  custom:
    - name: "MyAutoAPI"
      apiKey: "${MYAUTOAPI_API_KEY}"
      baseURL: "https://api.myautoapi.com/v1"
      models:
        default:
          - "claude-fable-5-1"
        fetch: false
      titleConvo: true
      titleModel: "claude-fable-5-1"
      modelDisplayLabel: "MyAutoAPI"
```

把两处 `claude-fable-5-1` 换成当前 Key 分组可用的模型，例如 `gpt-6.1-sol`、`claude-opus-5-5`、`gemini-3.8-flash`、`grok-4.7`。示例里的 `claude-fable-5-1` 要用 **CC MAX 外接分组** 的 Key。第一次只写一个模型，能正常回复后再在 `default` 下逐行追加。

## 4. 挂载配置文件

在项目根目录添加或编辑 `docker-compose.override.yml`。文件已存在时，只合并这段挂载，不覆盖原有内容：

```yaml
services:
  api:
    volumes:
      - type: bind
        source: ./librechat.yaml
        target: /app/librechat.yaml
```

## 5. 重启并测试

```bash
docker compose down
docker compose up -d
```

打开 LibreChat，新建对话，选择 `MyAutoAPI` 和刚写入的模型，发送 `测试`。能正常回复就接入完成。

## 排查

| 现象 | 处理 |
| --- | --- |
| 页面里看不到 `MyAutoAPI` | 确认 `librechat.yaml` 在项目根目录、已在 `docker-compose.override.yml` 挂载、服务已重启 |
| 服务启动失败 | 运行 `docker compose logs api`，检查 YAML 缩进、`version`、`apiKey`、`baseURL` |
| `401 Unauthorized` | 重新复制 API Key，核对 `.env` 变量名和 `apiKey: "${MYAUTOAPI_API_KEY}"` 一致 |
| 模型不存在 | 换成当前 Key 分组里的完整模型名 |
| Base URL 路径错误 | `baseURL` 只写 `https://api.myautoapi.com/v1` |
| 写了多个模型后出错 | 先缩回一个确认可用的模型 |
