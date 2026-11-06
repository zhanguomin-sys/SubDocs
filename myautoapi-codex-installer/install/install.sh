#!/usr/bin/env bash
set -euo pipefail

CODEX_BASE_URL="https://api.myautoapi.com"
CODEX_MODEL="gpt-5.6-sol"
CODEX_HOME="$HOME/.codex"
CONFIG_FILE="$CODEX_HOME/config.toml"
AUTH_FILE="$CODEX_HOME/auth.json"

if ! { exec 3</dev/tty; } 2>/dev/null; then
  exec 3<&0
fi

say() {
  printf '%s\n' "$*"
  if [ -n "$*" ]; then
    sleep 0.5
  fi
}

read_input() {
  local var_name="$1"
  local prompt="$2"
  read -r -u 3 -p "$prompt" "$var_name"
}

list_running_clients() {
  ps -axo pid=,ppid=,uid=,stat=,comm= 2>/dev/null | awk -v uid="$(id -u)" -v self="$$" '
    {
      parent[$1] = $2
      if ($3 != uid || $4 ~ /^Z/) {
        next
      }
      name = $0
      sub(/^[ \t]*[0-9]+[ \t]+[0-9]+[ \t]+[0-9]+[ \t]+[^ \t]+[ \t]+/, "", name)
      sub(/^.*\//, "", name)
      lower = tolower(name)
      if (lower == "codex" || lower == "codex.exe") {
        client[$1] = "codex"
      } else if (lower == "chatgpt" || lower == "chatgpt.exe") {
        client[$1] = "chatgpt"
      }
    }
    END {
      for (pid = self; (pid in parent) && !(pid in own); pid = parent[pid]) {
        own[pid] = 1
      }
      for (pid in client) {
        print pid, client[pid], ((pid in own) ? "skip" : "stop")
      }
    }
  '
}

client_is_alive() {
  local state
  state="$(ps -o stat= -p "$1" 2>/dev/null | tr -d '[:space:]')" || true
  [ -n "$state" ] && [ "${state#Z}" = "$state" ]
}

wait_for_clients_exit() {
  local attempts="$1"
  local pid alive
  while true; do
    alive=""
    for pid in $pending_pids; do
      if client_is_alive "$pid"; then
        alive="$alive $pid"
      fi
    done
    pending_pids="$alive"
    if [ -z "$pending_pids" ]; then
      return 0
    fi
    if [ "$attempts" -le 0 ]; then
      return 1
    fi
    sleep 0.5
    attempts=$((attempts - 1))
  done
}

stop_running_clients() {
  local pid kind action
  local pending_pids=""
  chatgpt_running=false
  codex_running=false
  clients_left=false

  while read -r pid kind action; do
    case "$kind" in
      chatgpt)
        chatgpt_running=true
        ;;
      codex)
        codex_running=true
        ;;
      *)
        continue
        ;;
    esac
    if [ "$action" = "stop" ]; then
      pending_pids="$pending_pids $pid"
    else
      clients_left=true
    fi
  done <<EOF
$(list_running_clients || true)
EOF

  if [ -z "$pending_pids" ]; then
    return
  fi

  kill -TERM $pending_pids 2>/dev/null || true
  if wait_for_clients_exit 10; then
    return
  fi
  kill -KILL $pending_pids 2>/dev/null || true
  if wait_for_clients_exit 4; then
    return
  fi
  clients_left=true
}

report_running_clients() {
  local clients
  if [ "$chatgpt_running" = true ] && [ "$codex_running" = true ]; then
    clients="ChatGPT 软件和 Codex"
  elif [ "$chatgpt_running" = true ]; then
    clients="ChatGPT 软件"
  elif [ "$codex_running" = true ]; then
    clients="Codex"
  else
    say "当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。"
    return
  fi

  if [ "$clients_left" = true ]; then
    say "未能自动关闭 ${clients}，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。"
  else
    say "已自动关闭 ${clients}，请重新打开并新建对话测试，如有问题请联系网站客服。"
  fi
}

backup_if_exists() {
  local path="$1"
  local private_backup="${2:-false}"
  if [ -f "$path" ]; then
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
  fi
}

json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

show_write_plan() {
  if [ -f "$CONFIG_FILE" ] || [ -f "$AUTH_FILE" ]; then
    say ""
    say "检测到已有旧配置，将会在备份后再覆盖。"
  fi
}

read_validated_api_key() {
  local var_name="$1"
  local entered_key
  while true; do
    read_input entered_key "请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认："
    if [ -z "$entered_key" ]; then
      say ""
      say "API Key 不能为空，请重新输入。"
      say ""
      continue
    fi
    case "$entered_key" in
      sk-*)
        printf -v "$var_name" '%s' "$entered_key"
        return
        ;;
      *)
        say ""
        say "API Key 格式错误，必须以 sk- 开头，请重新输入。"
        say ""
        ;;
    esac
  done
}

write_codex_files() {
  local api_key="$1"
  local escaped_key
  local config_temp=""
  local auth_temp=""

  escaped_key="$(json_escape "$api_key")"
  mkdir -p "$CODEX_HOME"
  umask 077
  config_temp="$(mktemp "$CODEX_HOME/.config.toml.tmp.XXXXXX")"
  auth_temp="$(mktemp "$CODEX_HOME/.auth.json.tmp.XXXXXX")"

  trap 'rm -f "${config_temp:-}" "${auth_temp:-}"' RETURN

  cat > "$config_temp" <<EOF
model_provider = "OpenAI"
model = "$CODEX_MODEL"
review_model = "$CODEX_MODEL"
model_reasoning_effort = "high"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "$CODEX_BASE_URL"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true
image_generation = true
EOF
  printf '{\n  "OPENAI_API_KEY": "%s"\n}\n' "$escaped_key" > "$auth_temp"

  backup_if_exists "$CONFIG_FILE"
  backup_if_exists "$AUTH_FILE" true
  mv -f "$config_temp" "$CONFIG_FILE"
  mv -f "$auth_temp" "$AUTH_FILE"
  chmod 600 "$AUTH_FILE" 2>/dev/null || true
  trap - RETURN
}

main() {
  if [ "$(id -u)" -eq 0 ]; then
    say "请不要用 sudo 运行。脚本只应该写入当前用户的 $CODEX_HOME。"
    exit 1
  fi

  say "欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。"
  say ""
  say "本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json 文件。"

  show_write_plan
  say ""
  say "配置完成后会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。"
  say ""

  local api_key
  read_validated_api_key api_key

  write_codex_files "$api_key"
  stop_running_clients

  say ""
  say "配置已经完成，可以关闭终端了。"
  say ""
  report_running_clients
  say ""
  say "感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。"
}

main "$@"
