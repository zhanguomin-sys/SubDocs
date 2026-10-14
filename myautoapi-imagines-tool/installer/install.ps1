$ErrorActionPreference = "Stop"

$InstallerPath = Join-Path $PSScriptRoot "安装.py"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

if (Get-Command py -ErrorAction SilentlyContinue) {
    & py -3 $InstallerPath install --source-root $ProjectRoot @args
    if ($LASTEXITCODE -ne 0) {
        throw "安装核心返回退出码 $LASTEXITCODE。"
    }
    return
}

if (Get-Command python -ErrorAction SilentlyContinue) {
    & python $InstallerPath install --source-root $ProjectRoot @args
    if ($LASTEXITCODE -ne 0) {
        throw "安装核心返回退出码 $LASTEXITCODE。"
    }
    return
}

throw "安装失败：未找到 Python 3。请先安装 Python 3，再重新运行本安装脚本。"
