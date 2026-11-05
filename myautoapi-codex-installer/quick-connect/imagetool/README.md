# 内置生图工具

`build.py` 打包前把父仓库已发布的生图程序放进本目录，`go:embed` 把它编进助手：

- `program`：`docs/public/install/myautoapi-imagines-tool/releases/v<版本>/` 里对应系统的程序；
- `VERSION`：版本号，如 `0.8.2`；
- `SHA256`：该程序在发布目录 `SHA256SUMS` 里的校验值，助手运行前再核对一次。

这三个文件是构建产物，不进 Git。生图工具的源码只在 `myautoapi-imagines-tool/` 维护；它发布新版后，改 `build.py` 里的 `IMAGE_TOOL_RELEASE` 重新打包即可。
