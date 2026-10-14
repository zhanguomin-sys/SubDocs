# 无 Skill 基线演练

日期：2026-08-07

## 场景

用户要求使用已安装的 MyAutoAPI V0.1 工具生成并展示一张图片。演练不读取密钥、不修改文件、不发送网络请求。

## 基线回答摘要

回答先要求用户补充画面描述；获得描述后，调用：

```bash
"$CODEX_HOME/tools/myautoapi-imagines-tool/生成图片.py" \
  --model gpt-image-2 \
  --prompt '用户确认后的图片描述'
```

它明确不读取或输出密钥；成功后检查本地文件并展示；失败时不伪造结果，也不展示密钥。

## 与 V0.1 的差距

1. 把参数名留给 `--help` 决定，没有明确默认的 `size`、`quality`、`output_format` 和结果目录。
2. 没有要求读取脚本 JSON 摘要中的 `output_file` 与 `record_dir`，因此无法稳定定位证据。
3. 没有明确“失败后不重试、不中途换模型、不改用户提示词”。
4. 没有规定必须使用图片查看工具检查成功结果。

## Skill 范本应补上的规则

- 调用固定的 V0.1 脚本，且只允许 `gpt-image-2`。
- 用户未指定时显式使用 `1024x1024`、`high`、`png`；用户指定时原样传递。
- 只根据 stdout JSON 的 `output_file` 和 `record_dir` 做后续动作。
- 成功后查看 `output_file`；失败后报告 `record_dir` 并停止，不重试。
- 永不读取、展示、复制、传参或扫描 `auth.json`。
