# tests

这里放不需要真实 API Key 的测试：Images 与 Responses 文生图 JSON、参考图与修图输入、多图补齐、八模型白名单、两个 GPT Image 2.5 模型的默认 2K 与质量边界、其它模型专属参数、尺寸与质量校验、Grok 和 Nano Banana 比例与分辨率、嵌套图片响应解析、图片签名、扁平输出、安全文件名、多模型并发与逐张展示契约、Linux/macOS Code Mode 结果转发和目录恢复模拟、错误处理、V0.8.1 临时目录安装/卸载、专门画图分组 Key 交互，以及当前两个在线安装入口和四个原生构建产物契约。

运行 V0.1 测试：

```bash
python3 -m unittest tests/test_generate_image.py -v
```

在子项目根目录运行全部聚焦测试：

```bash
python3 -m unittest discover -s tests -p 'test_*.py' -v
```

运行原生工具测试：

```bash
go test ./... -v
```

测试只使用假的传输函数、1x1 PNG 夹具和临时 Codex Home。需要真实网络和 API Key 的验证单独记录，不把密钥写入测试文件、请求快照或日志。
