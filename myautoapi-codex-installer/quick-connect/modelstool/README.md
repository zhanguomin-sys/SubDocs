# 内置国产模型导入

`build.py` 打包前把 `../install/codex-models.ps1` 复制到本目录（先核对它与父仓库发布的 `docs/public/install/codex-models.ps1` 逐字节一致），`go:embed` 把它编进助手。助手在后台用 Windows PowerShell 运行它，不弹控制台窗口。

`codex-models.ps1` 是构建产物，不进 Git。脚本源码只在 `codex-domestic-models/` 维护，按那边的流程同步到 `install/` 和发布目录后，重新运行 `build.py` 即可。
