---
sidebarTitle: 日志写盘
---

# Codex 日志写盘

部分 Codex 版本会把大量 `TRACE / DEBUG / INFO` 运行日志持续写进 `%USERPROFILE%\.codex\logs_2.sqlite`，表现为 `logs_2.sqlite-wal` 频繁刷新、`logs` 表的 `MAX(id)` 不断增加。本页用 SQLite trigger 拦截这些日志，只处理 Windows 上的日志库，不动聊天记录。

## 让 AI 处理

把下面这段发给 AI：

```text
帮我检查 Windows 默认 Codex 日志库：

%USERPROFILE%\.codex\logs_2.sqlite

先备份 logs_2.sqlite、logs_2.sqlite-wal 和 logs_2.sqlite-shm。
判断是否有 TRACE / DEBUG / INFO 日志持续高频写盘。
发现高频写盘后，用 SQLite trigger 拦截 logs 表里 TRACE / DEBUG / INFO 级别的 insert。
处理后确认 MAX(id) 和 WAL 下 trace/debug/info 级日志不再增长。
只处理本地日志库，不要删除聊天记录。
```

这个方法只阻止后续 `TRACE / DEBUG / INFO` 日志写入，不会缩小已经变大的 `logs_2.sqlite`。数据库已经有几个 GB 时，还要单独清理或压缩。
