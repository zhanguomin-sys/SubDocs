#!/usr/bin/env bash
set -euo pipefail

CODEX_HOME="$HOME/.codex"
CONFIG_FILE="$CODEX_HOME/config.toml"
AUTH_FILE="$CODEX_HOME/auth.json"

say() {
  printf '%s\n' "$*"
}

backup_if_exists() {
  local path="$1"
  local private_backup="${2:-false}"
  if [ ! -f "$path" ]; then
    return
  fi

  local stamp backup suffix
  stamp="$(date +%Y%m%d%H%M%S)"
  backup="$path.bak.$stamp"
  suffix=1
  while [ -e "$backup" ]; do
    backup="$path.bak.$stamp.$suffix"
    suffix=$((suffix + 1))
  done

  cp "$path" "$backup"
  if [ "$private_backup" = "true" ]; then
    chmod 600 "$backup" 2>/dev/null || true
  fi
}

clean_config() {
  if [ ! -f "$CONFIG_FILE" ]; then
    return
  fi

  local config_temp=""
  backup_if_exists "$CONFIG_FILE"
  config_temp="$(mktemp "$CODEX_HOME/.config.toml.restore.XXXXXX")"
  trap 'rm -f "${config_temp:-}"' RETURN

  awk '
    function emit(line) {
      if (removed_root && !emitted && line ~ /^[[:space:]]*$/) {
        return
      }
      print line
      emitted = 1
    }

    BEGIN {
      in_root = 1
      skip_provider = 0
      removed_root = 0
      emitted = 0
    }

    {
      match_line = $0
      sub(/\r$/, "", match_line)

      is_table = match_line ~ /^[[:space:]]*\[\[?[^]]+\]\]?[[:space:]]*(#.*)?$/
      is_provider = match_line ~ /^[[:space:]]*\[model_providers\.[^]]+\][[:space:]]*(#.*)?$/

      if (is_table) {
        in_root = 0
        if (is_provider) {
          skip_provider = 1
          next
        }
        skip_provider = 0
      }

      if (skip_provider) {
        next
      }

      if (in_root && match_line ~ /^[[:space:]]*(model_provider|model|review_model)[[:space:]]*=/) {
        removed_root = 1
        next
      }

      emit($0)
    }
  ' "$CONFIG_FILE" > "$config_temp"

  mv -f "$config_temp" "$CONFIG_FILE"
  trap - RETURN
}

clear_auth() {
  if [ ! -f "$AUTH_FILE" ]; then
    return
  fi

  backup_if_exists "$AUTH_FILE" true
  : > "$AUTH_FILE"
  chmod 600 "$AUTH_FILE" 2>/dev/null || true
}

main() {
  if [ ! -d "$CODEX_HOME" ]; then
    say "未找到 Codex 配置，无需还原。"
    return
  fi

  umask 077
  clean_config
  clear_auth

  say "Codex 自定义模型配置和登录凭据已清除。"
  say "请彻底退出 Codex，重新打开后使用官方账号登录。"
}

main "$@"
