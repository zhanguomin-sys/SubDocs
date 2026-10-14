#!/usr/bin/env bash

set -euo pipefail

if ! command -v python3 >/dev/null 2>&1; then
  echo "卸载失败：未找到 Python 3。请先安装 Python 3，再重新运行本卸载脚本。" >&2
  exit 1
fi

installer_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
project_root="$(dirname -- "$installer_dir")"

exec python3 "$installer_dir/安装.py" uninstall --source-root "$project_root" "$@"
