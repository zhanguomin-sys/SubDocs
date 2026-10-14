# Codex 日志写盘

本页只处理 Windows 上 Codex 日志库高频写盘的问题。目标是阻止 `TRACE / DEBUG / INFO` 日志继续增长，不处理聊天记录。

## 懒人方法

把下面这段直接复制给 AI：

```text
帮我检查 Windows 默认 Codex 日志库：

%USERPROFILE%\.codex\logs_2.sqlite

先备份 logs_2.sqlite、logs_2.sqlite-wal 和 logs_2.sqlite-shm。
判断是否有 TRACE / DEBUG / INFO 日志持续高频写盘。
发现高频写盘后，用 SQLite trigger 拦截 logs 表里 TRACE / DEBUG / INFO 级别的 insert。
处理后确认 MAX(id) 和 WAL 下 trace/debug/info 级日志不再增长。
只处理本地日志库，不要删除聊天记录。
```

## 这是什么问题

Codex 会把运行日志写到 `%USERPROFILE%\.codex\logs_2.sqlite`。部分版本会持续写入大量 `TRACE / DEBUG / INFO` 日志，表现为 `logs_2.sqlite-wal` 频繁刷新，`logs` 表的 `MAX(id)` 不断增加。

这个问题和聊天记录不是一回事，主要是运行日志高频落盘。确认是这个现象后，可以先用 SQLite trigger 做本地拦截。

## 处理范围

这个方法只阻止后续 `TRACE / DEBUG / INFO` 继续写入，不会自动缩小已经变大的 `logs_2.sqlite`。数据库已经有几个 GB 时，后续还要单独清理或压缩。
