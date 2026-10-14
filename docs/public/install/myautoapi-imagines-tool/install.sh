#!/usr/bin/env bash

set -euo pipefail

release_revision="V0.8.1"
release_base="https://docs.myautoapi.com/install/myautoapi-imagines-tool/releases/v0.8.1"

if ! command -v curl >/dev/null 2>&1; then
  echo "安装失败：未找到 curl，无法下载安装程序。" >&2
  exit 1
fi

system_name="$(uname -s)"
machine_name="$(uname -m)"
translated="0"
if [[ "$system_name/$machine_name" == "Darwin/x86_64" ]] && command -v sysctl >/dev/null 2>&1; then
  translated="$(sysctl -in sysctl.proc_translated 2>/dev/null || printf '0')"
fi

case "$system_name/$machine_name/$translated" in
  Darwin/arm64/*|Darwin/x86_64/1)
    artifact="myautoapi-imagines-tool-v0.8.1-darwin-arm64"
    expected_sha256="de9b5a9db388eabe575097844e7a07bf1d1112cb568b05bf441921122af533eb"
    expected_binary_format="macho-arm64"
    ;;
  Darwin/x86_64/*)
    artifact="myautoapi-imagines-tool-v0.8.1-darwin-amd64"
    expected_sha256="094a13e752a11106b263edb29e685f9277fecf5c6575732dd8ac4f6ed07833ef"
    expected_binary_format="macho-x86_64"
    ;;
  Linux/x86_64/*|Linux/amd64/*)
    artifact="myautoapi-imagines-tool-v0.8.1-linux-amd64"
    expected_sha256="05a8d7ccc7844063091920d020e414561b7ac3eb4fd9621873c9233f2eaf09da"
    expected_binary_format="elf-x86_64"
    ;;
  *)
    echo "安装失败：V0.8.1 仅支持 64 位 x64 Linux、Apple Silicon Mac 或 Intel 64 位 Mac，当前为 $system_name/$machine_name。" >&2
    exit 1
    ;;
esac

binary_matches_architecture() {
  local candidate="$1"
  local description
  if ! command -v file >/dev/null 2>&1; then
    return 1
  fi
  description="$(file -b "$candidate" 2>/dev/null || true)"
  case "$expected_binary_format" in
    macho-arm64)
      [[ "$description" == *"Mach-O"* && "$description" == *"64-bit"* && "$description" == *"arm64"* && "$description" == *"executable"* ]]
      ;;
    macho-x86_64)
      [[ "$description" == *"Mach-O"* && "$description" == *"64-bit"* && "$description" == *"x86_64"* && "$description" == *"executable"* ]]
      ;;
    elf-x86_64)
      [[ "$description" == *"ELF 64-bit"* && "$description" == *"x86-64"* && "$description" == *"executable"* ]]
      ;;
    *)
      return 1
      ;;
  esac
}

sha256_for_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$1" | awk '{print $NF}'
  else
    echo "安装失败：未找到 SHA-256 校验工具，已停止安装。" >&2
    return 1
  fi
}

codex_home="${CODEX_HOME:-$HOME/.codex}"
installed_release_path="$codex_home/tools/myautoapi-imagines-tool/RELEASE"
installed_binary_path="$codex_home/tools/myautoapi-imagines-tool/myautoapi-imagines-tool"
if [[ -f "$installed_release_path" ]] && [[ -f "$installed_binary_path" ]] && [[ -x "$installed_binary_path" ]] && [[ "$(tr -d '\r\n' < "$installed_release_path")" == "$release_revision" ]] && binary_matches_architecture "$installed_binary_path" && [[ "$(sha256_for_file "$installed_binary_path")" == "$expected_sha256" ]]; then
  echo "中转站生图工具 $release_revision 已是当前系统架构的最新版，无需下载，继续配置画图 Key。"
  binary_path="$installed_binary_path"
else
  temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/myautoapi-imagines-tool.XXXXXX")"
  binary_path="$temporary_root/$artifact"
  cleanup() {
    find "$temporary_root" -depth -delete
  }
  trap cleanup EXIT
  trap 'exit 129' HUP
  trap 'exit 130' INT
  trap 'exit 143' TERM

  echo "正在下载中转站生图工具 V0.8.1……"
  curl --fail --location --progress-bar --show-error --proto '=https' --proto-redir '=https' --tlsv1.2 \
    "$release_base/$artifact" --output "$binary_path"

  actual_sha256="$(sha256_for_file "$binary_path")"
  if [[ "$actual_sha256" != "$expected_sha256" ]]; then
    echo "安装失败：安装程序 SHA-256 校验不通过，已停止安装。" >&2
    exit 1
  fi
  if ! binary_matches_architecture "$binary_path"; then
    echo "安装失败：下载的安装程序与当前系统架构不匹配，已停止安装。" >&2
    exit 1
  fi
  chmod +x "$binary_path"
fi

if ! { true </dev/tty; } 2>/dev/null; then
  echo "安装失败：未检测到可输入的终端。请在终端中直接运行安装命令并粘贴画图 Key。" >&2
  exit 1
fi
"$binary_path" install "$@" </dev/tty
