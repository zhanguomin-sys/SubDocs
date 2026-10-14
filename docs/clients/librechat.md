# LibreChat 接入

LibreChat 是自部署网页聊天站，适合给自己或团队提供浏览器聊天入口。添加 MyAutoAPI 自定义 endpoint 后，网页聊天请求会通过 `https://api.myautoapi.com/v1` 发出。

## 启动 LibreChat

还没有 LibreChat 时，用 Docker 启动：

```bash
git clone https://github.com/danny-avila/LibreChat.git
cd LibreChat
cp .env.example .env
docker compose up -d
```

Windows 没有 `cp` 命令时，把 `.env.example` 手动复制一份，命名为 `.env`。

启动后访问：

```text
http://localhost:3080
```

已经能打开 LibreChat 时，直接继续配置 MyAutoAPI。

## 写入 API Key

在 LibreChat 项目根目录的 `.env` 里添加：

```dotenv
MYAUTOAPI_API_KEY=替换为你的MyAutoAPI_API_Key
```

把 `替换为你的MyAutoAPI_API_Key` 换成 MyAutoAPI API Key。不要把真实 Key 提交到公开仓库。

## 添加 endpoint

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
          - "claude-fable-5"
        fetch: false
      titleConvo: true
      titleModel: "claude-fable-5"
      modelDisplayLabel: "MyAutoAPI"
```

把 `claude-fable-5` 换成当前 Key 分组可用模型。也可以填 `gpt-5.5`、`claude-opus-5`、`gemini-3.6-flash` 或 `grok-4.5`。第一次只写一个模型，能正常回复后再增加其它模型。

## 挂载配置

Docker / Docker Compose 部署时，在项目根目录添加或编辑 `docker-compose.override.yml`：

```yaml
services:
  api:
    volumes:
      - type: bind
        source: ./librechat.yaml
        target: /app/librechat.yaml
```

已经有 `docker-compose.override.yml` 时，只合并这段挂载，不要覆盖原有配置。

## 重启并测试

保存后重启：

```bash
docker compose down
docker compose up -d
```

打开 LibreChat，选择 `MyAutoAPI` 和刚写入的模型，发送：

```text
测试
```

能正常回复，说明 LibreChat 已经通过 MyAutoAPI 发起请求。

## 排查

| 现象 | 处理 |
| --- | --- |
| 页面里看不到 `MyAutoAPI` | 确认 `librechat.yaml` 在项目根目录、Docker 已挂载、服务已重启 |
| 服务启动失败 | 查看 `docker compose logs api`，重点检查 YAML 缩进、`version`、`apiKey`、`baseURL` |
| `401 Unauthorized` | 重新复制 API Key，确认 `.env` 变量名和 `apiKey: "${MYAUTOAPI_API_KEY}"` 一致 |
| 模型不存在 | 换成当前 Key 分组里的完整模型名 |
| Base URL 路径错误 | `baseURL` 只写 `https://api.myautoapi.com/v1` |
| 写了多个模型后出错 | 先缩小到一个确认可用的模型 |

追加模型时，在 `default` 下继续加模型名：

```yaml
models:
  default:
    - "claude-fable-5"
    - "替换为另一个可用模型名"
  fetch: false
```
