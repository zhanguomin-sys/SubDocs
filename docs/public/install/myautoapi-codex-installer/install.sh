#!/usr/bin/env bash
set -euo pipefail

CODEX_BASE_URL="https://api.myautoapi.com"
CODEX_MODEL="gpt-6.1-sol"
CODEX_HOME="$HOME/.codex"
CONFIG_FILE="$CODEX_HOME/config.toml"
AUTH_FILE="$CODEX_HOME/auth.json"
USAGE_URL="$CODEX_BASE_URL/v1/usage"
read_clipboard=false
key_state=unknown
key_balance=""
key_http=""

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

# 按 JSON 规则转义字符串：引号、反斜杠和控制字符。
json_escape() {
  local text="$1"
  local out=""
  local char code i
  for ((i = 0; i < ${#text}; i++)); do
    char="${text:i:1}"
    case "$char" in
      '"') out+='\"' ;;
      '\') out+='\\' ;;
      *)
        printf -v code '%d' "'$char" 2>/dev/null || code=32
        if [ "$code" -ge 0 ] && [ "$code" -lt 32 ]; then
          printf -v char '\\u%04x' "$code"
        fi
        out+="$char"
        ;;
    esac
  done
  printf '%s' "$out"
}

show_write_plan() {
  if [ -f "$CONFIG_FILE" ] || [ -f "$AUTH_FILE" ]; then
    say ""
    say "检测到已有配置，会先备份；config.toml 改写中转站接口和模型，去掉以前写入的完全访问设置和其它接口，其余设置保留。"
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

mask_key() {
  local key="$1"
  local length=${#1}
  if [ "$length" -ge 24 ]; then
    printf '%s****%s' "${key:0:12}" "${key:length-8}"
  elif [ "$length" -ge 8 ]; then
    printf '%s****%s' "${key:0:3}" "${key:length-4}"
  else
    printf 'sk-****'
  fi
}

# 只在 .app 启动时调用：剪贴板里是一整段 sk- 密钥才使用。
clipboard_api_key() {
  local text
  if ! command -v pbpaste >/dev/null 2>&1; then
    return 0
  fi
  text="$(pbpaste 2>/dev/null | tr -d '\r' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')" || return 0
  if [[ "$text" =~ ^sk-[A-Za-z0-9_-]+$ ]]; then
    printf '%s' "$text"
  fi
  return 0
}

# 从 JSON 顶层取一个标量字段的原始值，跳过嵌套对象里的同名字段。
json_top_value() {
  awk -v want="$1" '
    BEGIN { RS = "\001" }
    {
      text = $0
      n = length(text)
      depth = 0
      expect_key = 0
      key = ""
      i = 1
      while (i <= n) {
        c = substr(text, i, 1)
        if (c == "\"") {
          j = i + 1
          value = ""
          while (j <= n) {
            d = substr(text, j, 1)
            if (d == "\\") {
              value = value substr(text, j, 2)
              j += 2
              continue
            }
            if (d == "\"") {
              break
            }
            value = value d
            j++
          }
          if (depth == 1 && expect_key) {
            key = value
            expect_key = 0
          }
          i = j + 1
          continue
        }
        if (c == "{" || c == "[") {
          depth++
          if (depth == 1 && c == "{") {
            expect_key = 1
          }
        } else if (c == "}" || c == "]") {
          depth--
        } else if (c == "," && depth == 1) {
          expect_key = 1
        } else if (c == ":" && depth == 1 && key == want) {
          i++
          while (substr(text, i, 1) ~ /[[:space:]]/) {
            i++
          }
          result = ""
          if (substr(text, i, 1) == "\"") {
            # 字符串值去掉引号输出；本脚本只读取不含转义的状态和时间。
            i++
            while (i <= n && substr(text, i, 1) != "\"") {
              result = result substr(text, i, 1)
              i++
            }
            print result
            exit
          }
          while (i <= n && substr(text, i, 1) !~ /[],}[:space:]]/) {
            result = result substr(text, i, 1)
            i++
          }
          print result
          exit
        }
        i++
      }
    }
  '
}

# 过期时间已过返回 0。只认 RFC 3339 时间（sub2api 的 expires_at）；认不出时按未过期处理。
expiry_passed() {
  local expires="$1"
  local now
  [ -n "$expires" ] || return 1
  now="$(date -u +%s 2>/dev/null)" || return 1
  case "$now" in
    ''|*[!0-9]*)
      return 1
      ;;
  esac
  awk -v value="$expires" -v now="$now" '
    function days(y, m, d) {
      if (m <= 2) {
        y--
        m += 12
      }
      return 365 * y + int(y / 4) - int(y / 100) + int(y / 400) + int((153 * (m - 3) + 2) / 5) + d - 719469
    }
    BEGIN {
      if (value !~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9][Tt ][0-9][0-9]:[0-9][0-9]:[0-9][0-9]/) {
        exit 1
      }
      seconds = days(substr(value, 1, 4) + 0, substr(value, 6, 2) + 0, substr(value, 9, 2) + 0) * 86400 + substr(value, 12, 2) * 3600 + substr(value, 15, 2) * 60 + substr(value, 18, 2)
      zone = substr(value, 20)
      sub(/^\.[0-9]+/, "", zone)
      if (zone ~ /^[+-][0-9][0-9]:[0-9][0-9]$/) {
        offset = substr(zone, 2, 2) * 3600 + substr(zone, 5, 2) * 60
        seconds += (substr(zone, 1, 1) == "+") ? -offset : offset
      } else if (zone != "Z" && zone != "z") {
        exit 1
      }
      exit !(now + 0 > seconds)
    }'
}

# 检查结果写入 key_state：usable、no_balance、rejected、disabled，
# 检查不了时为 timeout（10 秒内没有回应）、offline（连不上）、
# server_error（服务器出错或返回异常，key_http 为状态码）或 unknown（没有 curl）。
check_api_key() {
  local key="$1"
  local response status body valid remaining key_status curl_exit
  key_state=unknown
  key_balance=""
  key_http=""
  if ! command -v curl >/dev/null 2>&1; then
    return 0
  fi
  # 请求头从标准输入传给 curl，密钥不出现在进程参数里。
  response="$(printf 'Authorization: Bearer %s\n' "$key" |
    curl -sS --max-time 10 -H @- -H 'Accept: application/json' -w '\n%{http_code}' "$USAGE_URL" 2>/dev/null)" && curl_exit=0 || curl_exit=$?
  if [ "$curl_exit" -ne 0 ]; then
    # curl 退出码 28 是超时；其余（解析不了域名、连接被拒、TLS 失败等）是连不上。
    if [ "$curl_exit" -eq 28 ]; then
      key_state=timeout
    else
      key_state=offline
    fi
    return 0
  fi
  status="${response##*$'\n'}"
  body="${response%$'\n'*}"
  case "$status" in
    401|403)
      key_state=rejected
      return 0
      ;;
    200)
      ;;
    *)
      key_state=server_error
      key_http="$status"
      return 0
      ;;
  esac
  valid="$(printf '%s' "$body" | json_top_value isValid)"
  case "$valid" in
    false)
      key_state=disabled
      return 0
      ;;
    true)
      ;;
    *)
      key_state=server_error
      return 0
      ;;
  esac
  # /v1/usage 不做计费检查：过期或额度用完的密钥仍返回 isValid true；到期后 status 仍是 active。
  key_status="$(printf '%s' "$body" | json_top_value status)"
  case "$key_status" in
    expired|quota_exhausted)
      key_state=disabled
      return 0
      ;;
  esac
  if expiry_passed "$(printf '%s' "$body" | json_top_value expires_at)"; then
    key_state=disabled
    return 0
  fi
  key_state=usable
  remaining="$(printf '%s' "$body" | json_top_value remaining)"
  if ! printf '%s' "$remaining" | grep -Eq '^-?[0-9]+(\.[0-9]+)?([eE][-+]?[0-9]+)?$'; then
    return 0
  fi
  key_balance="$(awk -v value="$remaining" 'BEGIN { printf "%.2f", value }')"
  if awk -v value="$remaining" 'BEGIN { exit !(value + 0 <= 0) }'; then
    key_state=no_balance
  fi
}

obtain_api_key() {
  local var_name="$1"
  local candidate=""
  local clipboard_key=""
  local entered_key
  if [ -n "${MYAUTOAPI_KEY:-}" ]; then
    case "$MYAUTOAPI_KEY" in
      sk-*)
        candidate="$MYAUTOAPI_KEY"
        ;;
      *)
        say "MYAUTOAPI_KEY 格式错误，必须以 sk- 开头，请手动输入。"
        say ""
        ;;
    esac
  fi
  unset MYAUTOAPI_KEY
  if [ -z "$candidate" ] && [ "$read_clipboard" = true ]; then
    clipboard_key="$(clipboard_api_key)"
  fi

  while true; do
    if [ -z "$candidate" ] && [ -n "$clipboard_key" ]; then
      printf '检测到剪贴板里的密钥：%s，直接回车使用；要换别的就粘贴后回车：' "$(mask_key "$clipboard_key")"
      read_input entered_key ""
      case "$entered_key" in
        "")
          candidate="$clipboard_key"
          ;;
        sk-*)
          candidate="$entered_key"
          ;;
        *)
          say ""
          say "API Key 格式错误，必须以 sk- 开头，请重新输入。"
          say ""
          ;;
      esac
      clipboard_key=""
    fi
    if [ -z "$candidate" ]; then
      read_validated_api_key candidate
    fi

    check_api_key "$candidate"
    case "$key_state" in
      rejected)
        say ""
        say "密钥无效，请回网站重新复制完整的密钥。"
        say ""
        candidate=""
        continue
        ;;
      disabled)
        say ""
        say "这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。"
        say ""
        candidate=""
        continue
        ;;
    esac
    say ""
    case "$key_state" in
      no_balance)
        say "密钥可用，但余额为 \$0。充值后即可使用；新人进群领 2 元。"
        ;;
      timeout)
        say "网络超时：MyAutoAPI 服务器 10 秒内没有回应，暂时无法检查密钥，继续写入。"
        ;;
      offline)
        say "连不上 MyAutoAPI 服务器（网络不通，或被代理、防火墙拦截），暂时无法检查密钥，继续写入。"
        ;;
      server_error)
        if [ -n "$key_http" ]; then
          say "MyAutoAPI 服务器暂时出错（HTTP ${key_http}），暂时无法检查密钥，继续写入。"
        else
          say "MyAutoAPI 服务器暂时出错（返回内容异常），暂时无法检查密钥，继续写入。"
        fi
        ;;
      unknown)
        say "暂时无法检查密钥，继续写入。"
        ;;
      *)
        if [ -n "$key_balance" ]; then
          say "密钥检查通过，余额 \$${key_balance}。"
        else
          say "密钥检查通过。"
        fi
        ;;
    esac
    printf -v "$var_name" '%s' "$candidate"
    return
  done
}

# 导入时 config.toml 只写中转站这几行：开头的 model_provider、model，以及作为第一个段的
# [model_providers.OpenAI]。删掉的内容：开头的 review_model、以前版本写的完全访问和沙箱
# 设置、推理强度、国产模型列表 model_catalog_json（重新接入后国产模型照旧要再导入）、
# profile（会盖过 model 的设置）、会让 Key 不生效的登录设置 forced_login_method 和
# cli_auth_credentials_store；所有接口段 [model_providers.*]，包括写成
# model_providers.xx = 或 {…} 形式的。其它内容原样保留（包括 Codex App 设置沙箱时写的
# [windows]），旧文件另有备份。取舍见 quick-connect/取舍记录.md。
# 与快速接入助手 connect/codexfiles.go 的 ConfigFor、install/codex.ps1 逐字节一致。
# 按字节处理（LC_ALL=C），各系统的 awk 结果相同。
codex_config() {
  LC_ALL=C awk -v model="$CODEX_MODEL" -v base_url="$CODEX_BASE_URL" '
    # 跟踪 TOML 多行字符串和数组、行内表的括号，字符串里的内容不会被当成表或键。
    # multiline 是这一行结束时还没关上的 """ 或 \047\047\047，depth 加上这一行打开减去
    # 关上的 [ { 个数；注释和单行字符串整段跳过，里面的引号和括号不算。
    # 与 connect/codexfiles.go 的 scanLine 相同（codex-restore.sh 仍用旧的写法）。
    function update_multiline(line,    n, i, c, extra) {
      n = length(line)
      i = 1
      while (i <= n) {
        c = substr(line, i, 1)
        if (multiline != "") {
          if (multiline == "\"\"\"" && c == "\\") {
            i += 2
          } else if (substr(line, i, 3) == multiline) {
            # 后面最多再两个引号也属于字符串。
            i += 3
            extra = 0
            while (extra < 2 && i <= n && substr(line, i, 1) == substr(multiline, 1, 1)) {
              i++
              extra++
            }
            multiline = ""
          } else {
            i++
          }
        } else if (c == "#") {
          return
        } else if (c == "[" || c == "{") {
          depth++
          i++
        } else if (c == "]" || c == "}") {
          depth--
          i++
        } else if (substr(line, i, 3) == "\"\"\"" || substr(line, i, 3) == "\047\047\047") {
          multiline = substr(line, i, 3)
          i += 3
        } else if (c == "\"" || c == "\047") {
          i++
          while (i <= n && substr(line, i, 1) != c) {
            if (c == "\"" && substr(line, i, 1) == "\\") {
              i++
            }
            i++
          }
          i++
        } else {
          i++
        }
      }
    }

    BEGIN {
      # 表头按 TOML 键名写法识别：裸键、带转义的 "…"、单引号字符串，用点连接；
      # 引号里的 ] 不算表头结束。
      key = "([A-Za-z0-9_-]+|\"([^\"\\\\]|\\\\.)*\"|\047[^\047]*\047)"
      header_re = "^[ \t]*\\[\\[?[ \t]*" key "([ \t]*\\.[ \t]*" key ")*[ \t]*\\]\\]?[ \t]*(#.*)?$"
      # 要删的根级键和 model_providers 都可以写成裸键、"…" 或单引号形式。
      names = "(model_provider|model|review_model|model_reasoning_effort|disable_response_storage|approval_policy|sandbox_mode|network_access|windows_wsl_setup_acknowledged|model_catalog_json|profile|forced_login_method|cli_auth_credentials_store)"
      providers = "(model_providers|\"model_providers\"|\047model_providers\047)"
      root_re = "^[ \t]*((" names "|\"" names "\"|\047" names "\047)[ \t]*=|" providers "[ \t]*[.=])"
      provider_re = "^[ \t]*\\[[ \t]*" providers "([ \t]*\\.[ \t]*" key ")*[ \t]*\\][ \t]*(#.*)?$"
      in_root = 1
      skip_table = 0
      multiline = ""
      depth = 0
      skip_value = 0
      root_count = 0
      table_count = 0
      crlf = 0
    }

    {
      line = $0
      if (NR == 1 && substr(line, 1, 3) == "\357\273\277") {
        line = substr(line, 4)
      }
      if (sub(/\r$/, "", line)) {
        crlf = 1
      }
      lower = tolower(line)

      if (skip_value) {
        # 被删键的值还没结束：多行字符串、数组或行内表关上之前都跳过。
        update_multiline(line)
        skip_value = multiline != "" || depth > 0
        next
      }
      in_string = multiline != ""
      depth = 0
      update_multiline(line)
      if (in_string) {
        if (skip_table) {
          next
        }
      } else {
        if (lower ~ header_re) {
          in_root = 0
          skip_table = lower ~ provider_re
        }
        if (skip_table) {
          next
        }
        if (in_root && lower ~ root_re) {
          skip_value = multiline != "" || depth > 0
          next
        }
      }
      if (in_root) {
        root[++root_count] = line
      } else {
        table[++table_count] = line
      }
    }

    END {
      newline = crlf ? "\r\n" : "\n"
      # 去掉开头和结尾只有空格、Tab 的行（TOML 只认这两种空白）。
      root_start = 1
      while (root_start <= root_count && root[root_start] ~ /^[ \t]*$/) root_start++
      root_end = root_count
      while (root_end >= root_start && root[root_end] ~ /^[ \t]*$/) root_end--
      table_start = 1
      while (table_start <= table_count && table[table_start] ~ /^[ \t]*$/) table_start++
      table_end = table_count
      while (table_end >= table_start && table[table_end] ~ /^[ \t]*$/) table_end--

      out = "model_provider = \"OpenAI\"" newline "model = \"" model "\""
      for (i = root_start; i <= root_end; i++) out = out newline root[i]
      out = out newline newline "[model_providers.OpenAI]" newline "name = \"OpenAI\"" newline "base_url = \"" base_url "\"" newline "wire_api = \"responses\"" newline "requires_openai_auth = true"
      if (table_start <= table_end) {
        out = out newline newline table[table_start]
        for (i = table_start + 1; i <= table_end; i++) out = out newline table[i]
      }
      printf "%s", out
    }
  ' "$1"
}

write_codex_files() {
  local api_key="$1"
  local escaped_key
  local config_temp=""
  local auth_temp=""
  local old_config="/dev/null"

  escaped_key="$(json_escape "$api_key")"
  mkdir -p "$CODEX_HOME"
  umask 077
  config_temp="$(mktemp "$CODEX_HOME/.config.toml.tmp.XXXXXX")"
  auth_temp="$(mktemp "$CODEX_HOME/.auth.json.tmp.XXXXXX")"

  trap 'rm -f "${config_temp:-}" "${auth_temp:-}"' RETURN

  if [ -f "$CONFIG_FILE" ]; then
    old_config="$CONFIG_FILE"
  fi
  codex_config "$old_config" > "$config_temp"
  printf '{\n  "OPENAI_API_KEY": "%s"\n}\n' "$escaped_key" > "$auth_temp"

  backup_if_exists "$CONFIG_FILE"
  backup_if_exists "$AUTH_FILE" true
  mv -f "$config_temp" "$CONFIG_FILE"
  mv -f "$auth_temp" "$AUTH_FILE"
  chmod 600 "$AUTH_FILE" 2>/dev/null || true
  trap - RETURN
}

main() {
  local argument
  for argument in "$@"; do
    if [ "$argument" = "--from-app" ]; then
      read_clipboard=true
    fi
  done

  if [ "$(id -u)" -eq 0 ]; then
    say "请不要用 sudo 运行。脚本只应该写入当前用户的 ${CODEX_HOME}。"
    exit 1
  fi

  say "欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。"
  say ""
  say "本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。"

  show_write_plan
  say ""
  say "写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。"
  say ""

  local api_key
  obtain_api_key api_key

  stop_running_clients
  write_codex_files "$api_key"

  say ""
  say "配置已经完成，可以关闭终端了。"
  say ""
  report_running_clients
  say ""
  say "感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。"
}

main "$@"
