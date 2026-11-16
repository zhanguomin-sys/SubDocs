#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHELL_SCRIPT="$ROOT_DIR/install/install.sh"
INSTALLER_TEST_BASH="${INSTALLER_TEST_BASH:-bash}"
if [ -n "${INSTALLER_TEST_LOCALE:-}" ] &&
   ! LC_ALL="$INSTALLER_TEST_LOCALE" locale charmap >/dev/null 2>&1; then
  printf '测试语言环境不可用：%s\n' "$INSTALLER_TEST_LOCALE" >&2
  exit 1
fi
POWERSHELL_SCRIPT="$ROOT_DIR/install/install.ps1"
README_FILE="$ROOT_DIR/README.md"
REFERENCE_SHELL_SCRIPT="$ROOT_DIR/reference/original/install.sh"
REFERENCE_POWERSHELL_SCRIPT="$ROOT_DIR/reference/original/install.ps1"

MYAUTOAPI_TEST_REAL_PS="$(command -v ps)"
export MYAUTOAPI_TEST_REAL_PS

PASS_COUNT=0
FAIL_COUNT=0
TEST_ROOT="$(mktemp -d)"
USAGE_SERVER_PID=""
trap '[ -n "$USAGE_SERVER_PID" ] && kill "$USAGE_SERVER_PID" 2>/dev/null; rm -rf "$TEST_ROOT"' EXIT

# 检查密钥的请求全部转到本地假服务器，测试不访问真实接口，也不用真实密钥。
USAGE_LOG="$TEST_ROOT/usage.log"
: > "$USAGE_LOG"
python3 "$ROOT_DIR/tests/fake_usage_server.py" "$USAGE_LOG" > "$TEST_ROOT/usage.port" &
USAGE_SERVER_PID=$!
for _ in $(seq 1 50); do
  [ -s "$TEST_ROOT/usage.port" ] && break
  sleep 0.1
done
MYAUTOAPI_TEST_SERVER="http://127.0.0.1:$(head -n 1 "$TEST_ROOT/usage.port")"
MYAUTOAPI_TEST_REAL_CURL="$(command -v curl)"
MYAUTOAPI_TEST_CURL_LOG="$TEST_ROOT/curl.log"
MYAUTOAPI_TEST_PBPASTE_LOG="$TEST_ROOT/pbpaste.log"
export MYAUTOAPI_TEST_SERVER MYAUTOAPI_TEST_REAL_CURL MYAUTOAPI_TEST_CURL_LOG MYAUTOAPI_TEST_PBPASTE_LOG
unset MYAUTOAPI_KEY MYAUTOAPI_TEST_CLIPBOARD

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf 'PASS: %s\n' "$1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf 'FAIL: %s\n' "$1"
}

assert_contains() {
  local file="$1"
  local text="$2"
  local label="$3"
  if [ -f "$file" ] && grep -Fq -- "$text" "$file"; then
    pass "$label"
  else
    fail "$label"
  fi
}

assert_not_contains() {
  local file="$1"
  local text="$2"
  local label="$3"
  if [ -f "$file" ] && ! grep -Fq -- "$text" "$file"; then
    pass "$label"
  else
    fail "$label"
  fi
}

make_test_path() {
  local bin_dir="$1"
  mkdir -p "$bin_dir"
  cat > "$bin_dir/id" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "-u" ]; then
  printf '%s\n' "${MYAUTOAPI_TEST_UID:-1000}"
else
  /usr/bin/id "$@"
fi
EOF
  chmod +x "$bin_dir/id"
  cat > "$bin_dir/ps" <<'EOF'
#!/usr/bin/env bash
case " $* " in
  *" -p "*)
    case "${MYAUTOAPI_TEST_PS_ALIVE:-}" in
      real)
        exec "$MYAUTOAPI_TEST_REAL_PS" "$@"
        ;;
      "")
        ;;
      always)
        printf 'S\n'
        ;;
      *)
        checks=0
        if [ -f "$MYAUTOAPI_TEST_KILL_LOG.checks" ]; then
          checks="$(<"$MYAUTOAPI_TEST_KILL_LOG.checks")"
        fi
        printf '%s\n' "$((checks + 1))" > "$MYAUTOAPI_TEST_KILL_LOG.checks"
        if [ "$checks" -lt "$MYAUTOAPI_TEST_PS_ALIVE" ]; then
          printf 'S\n'
        fi
        ;;
    esac
    exit 0
    ;;
esac
if [ -n "${MYAUTOAPI_TEST_PS_OUTPUT:-}" ]; then
  printf '%b' "$MYAUTOAPI_TEST_PS_OUTPUT"
fi
if [ -n "${MYAUTOAPI_TEST_PS_ANCESTOR:-}" ]; then
  pid=$$
  while [ "$pid" -gt 1 ]; do
    parent_pid="$("$MYAUTOAPI_TEST_REAL_PS" -o ppid= -p "$pid" | tr -d ' ')"
    if [ -z "$parent_pid" ]; then
      break
    fi
    if [ "$pid" = "$MYAUTOAPI_TEST_PS_ANCESTOR" ]; then
      printf '%s %s 1000 S /usr/local/bin/codex\n' "$pid" "$parent_pid"
    else
      printf '%s %s 1000 S bash\n' "$pid" "$parent_pid"
    fi
    pid="$parent_pid"
  done
fi
exit 0
EOF
  chmod +x "$bin_dir/ps"
  cat > "$bin_dir/date" <<'EOF'
#!/usr/bin/env bash
# 固定时间 2026-07-22 12:34:56 UTC：备份名用年月日时分秒，检查过期时间用秒数。
case "$*" in
  *+%s*) printf '1784723696\n' ;;
  *) printf '20260722123456\n' ;;
esac
EOF
  chmod +x "$bin_dir/date"
  cat > "$bin_dir/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$bin_dir/sleep"
  cat > "$bin_dir/curl" <<'EOF'
#!/usr/bin/env bash
# 记录参数，并把检查密钥的地址换成本地假服务器；没有假服务器时按网络不通处理。
printf '%s\n' "$*" >> "$MYAUTOAPI_TEST_CURL_LOG"
if [ -z "${MYAUTOAPI_TEST_SERVER:-}" ]; then
  exit 7
fi
args=()
previous=""
for argument in "$@"; do
  # MYAUTOAPI_TEST_SHORT_TIMEOUT 把 10 秒超时缩成 1 秒，测试超时不用真等 10 秒。
  if [ "$previous" = "--max-time" ] && [ -n "${MYAUTOAPI_TEST_SHORT_TIMEOUT:-}" ]; then
    argument=1
  fi
  args+=("${argument//https:\/\/api.myautoapi.com/$MYAUTOAPI_TEST_SERVER}")
  previous="$argument"
done
case " ${args[*]} " in
  *api.myautoapi.com*)
    exit 7
    ;;
esac
exec "$MYAUTOAPI_TEST_REAL_CURL" "${args[@]}"
EOF
  chmod +x "$bin_dir/curl"
  cat > "$bin_dir/pbpaste" <<'EOF'
#!/usr/bin/env bash
printf 'called\n' >> "$MYAUTOAPI_TEST_PBPASTE_LOG"
printf '%b' "${MYAUTOAPI_TEST_CLIPBOARD:-}"
EOF
  chmod +x "$bin_dir/pbpaste"
}

run_shell_script() {
  local home_dir="$1"
  local input="$2"
  local output_file="$3"
  local bin_dir="$home_dir/test-bin"
  make_test_path "$bin_dir"
  : > "$home_dir/kill.log"
  (
    # kill 是 bash 内建命令，PATH 打桩拦不住：这里只记录调用，只对指定的测试进程真正发信号。
    kill() {
      local signal="$1"
      local pid config_state=missing
      shift
      if [ -f "$HOME/.codex/config.toml" ] && [ -f "$HOME/.codex/auth.json" ]; then
        config_state=written
      fi
      for pid in "$@"; do
        printf '%s %s %s\n' "$signal" "$pid" "$config_state" >> "$MYAUTOAPI_TEST_KILL_LOG"
        if [ "$pid" = "${MYAUTOAPI_TEST_REAL_KILL_PID:-}" ]; then
          builtin kill "$signal" "$pid"
        fi
      done
    }
    export -f kill
    printf '%b' "$input" | env HOME="$home_dir" PATH="$bin_dir:$PATH" LC_ALL="${INSTALLER_TEST_LOCALE:-${LC_ALL:-}}" MYAUTOAPI_TEST_KILL_LOG="$home_dir/kill.log" "$INSTALLER_TEST_BASH" "$SHELL_SCRIPT" "${@:4}" > "$output_file" 2>&1
  )
}

assert_contains "$SHELL_SCRIPT" 'CODEX_HOME="$HOME/.codex"' "Shell 固定使用普通客户的 ~/.codex"
assert_not_contains "$SHELL_SCRIPT" 'CODEX_HOME="${CODEX_HOME:-' "Shell 不处理自定义 CODEX_HOME"
assert_contains "$SHELL_SCRIPT" 'model_provider = \"OpenAI\"' "Shell provider 与网站配置一致"
assert_contains "$SHELL_SCRIPT" '[model_providers.OpenAI]' "Shell provider 表名与选择值一致"
assert_contains "$POWERSHELL_SCRIPT" 'model_provider = `"OpenAI`"' "PowerShell provider 与网站配置一致"
assert_contains "$POWERSHELL_SCRIPT" '[model_providers.OpenAI]' "PowerShell provider 表名与选择值一致"
assert_contains "$SHELL_SCRIPT" 'requires_openai_auth = true' "Shell 使用 auth.json 中的 API Key 鉴权"
assert_contains "$POWERSHELL_SCRIPT" 'requires_openai_auth = true' "PowerShell 使用 auth.json 中的 API Key 鉴权"
assert_not_contains "$SHELL_SCRIPT" 'x-openai-actor-authorization' "Shell 不写入生图请求头"
assert_not_contains "$POWERSHELL_SCRIPT" 'x-openai-actor-authorization' "PowerShell 不写入生图请求头"

assert_not_contains "$SHELL_SCRIPT" 'detect_codex_app' "Shell 不再检测 App 安装位置"
assert_not_contains "$POWERSHELL_SCRIPT" 'Find-CodexApp' "PowerShell 不再检测 App 安装位置"
assert_contains "$SHELL_SCRIPT" 'ps -axo pid=,ppid=,uid=,stat=,comm=' "Shell 一次查询取得进程号、父进程、用户、状态和名称"
assert_contains "$POWERSHELL_SCRIPT" 'Get-Process -Name "codex", "chatgpt"' "PowerShell 按名称精确查找 codex/chatgpt"
assert_contains "$POWERSHELL_SCRIPT" '$_.SessionId -eq $sessionId' "PowerShell 只处理当前登录会话的进程"
assert_contains "$SHELL_SCRIPT" 'kill -TERM $pending_pids' "Shell 先正常结束客户端进程"
assert_contains "$SHELL_SCRIPT" 'kill -KILL $pending_pids' "Shell 对未退出的客户端进程强制结束"
assert_contains "$POWERSHELL_SCRIPT" '$targets | Stop-Process -Force' "PowerShell 强制结束客户端进程"
assert_contains "$POWERSHELL_SCRIPT" 'Get-CimInstance Win32_Process -Filter "ProcessId = $current" -ErrorAction Stop' "PowerShell 父进程查询失败时不结束进程"
assert_contains "$POWERSHELL_SCRIPT" '-not (Test-ProcessExited $_)' "PowerShell 只确认最初记录的进程是否退出"
assert_contains "$POWERSHELL_SCRIPT" '} elseif ($Result.Left) {' "PowerShell 查询出错时仍提示重启电脑"
assert_not_contains "$SHELL_SCRIPT" 'open ' "Shell 只关闭不重新打开客户端"
assert_not_contains "$POWERSHELL_SCRIPT" 'Start-Process' "PowerShell 只关闭不重新打开客户端"

powershell_write_line="$(grep -n -F -- 'Write-CodexFiles -ApiKey $ApiKey' "$POWERSHELL_SCRIPT" | cut -d: -f1)"
powershell_stop_line="$(grep -n -F -- '$StopResult = Stop-RunningClients' "$POWERSHELL_SCRIPT" | cut -d: -f1)"
powershell_finish_line="$(grep -n -F -- 'Say "配置已经完成，可以关闭 PowerShell 了。"' "$POWERSHELL_SCRIPT" | cut -d: -f1)"
if [ -n "$powershell_write_line" ] && [ -n "$powershell_stop_line" ] && [ -n "$powershell_finish_line" ] &&
   [ "$powershell_stop_line" -lt "$powershell_write_line" ] &&
   [ "$powershell_write_line" -lt "$powershell_finish_line" ]; then
  pass "PowerShell 先关闭进程，再写入配置，最后显示完成提示"
else
  fail "PowerShell 先关闭进程，再写入配置，最后显示完成提示"
fi
assert_not_contains "$POWERSHELL_SCRIPT" '-AsSecureString' "PowerShell API Key 使用明文输入"
assert_contains "$POWERSHELL_SCRIPT" 'while (Test-Path -LiteralPath $backup)' "PowerShell 不覆盖同名历史备份"
assert_not_contains "$SHELL_SCRIPT" 'confirm_existing_files' "Shell 不再要求二次确认"
assert_not_contains "$POWERSHELL_SCRIPT" 'Confirm-ExistingFiles' "PowerShell 不再要求二次确认"
assert_not_contains "$SHELL_SCRIPT" 'read_input ignored' "Shell 不因运行中客户端等待"
assert_not_contains "$POWERSHELL_SCRIPT" 'Read-Host "关闭后按 Enter 继续"' "PowerShell 不因运行中客户端等待"
assert_contains "$SHELL_SCRIPT" 'report_running_clients' "Shell 报告客户端关闭结果"
assert_contains "$POWERSHELL_SCRIPT" 'Report-RunningClients' "PowerShell 报告客户端关闭结果"
assert_contains "$SHELL_SCRIPT" 'sleep 0.5' "Shell 每条非空提示间隔 0.5 秒"
assert_contains "$POWERSHELL_SCRIPT" 'Start-Sleep -Milliseconds 500' "PowerShell 每条非空提示间隔 0.5 秒"
assert_contains "$SHELL_SCRIPT" '欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。' "Shell 使用指定欢迎语"
assert_contains "$POWERSHELL_SCRIPT" '欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。' "PowerShell 使用指定欢迎语"
assert_contains "$SHELL_SCRIPT" '本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。' "Shell 使用指定脚本说明"
assert_contains "$POWERSHELL_SCRIPT" '本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。' "PowerShell 使用指定脚本说明"
assert_contains "$SHELL_SCRIPT" '请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认' "Shell 使用指定 API Key 提示"
assert_contains "$POWERSHELL_SCRIPT" '请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认' "PowerShell 使用指定 API Key 提示"
assert_contains "$SHELL_SCRIPT" '配置已经完成，可以关闭终端了。' "Shell 使用指定完成提示"
assert_contains "$POWERSHELL_SCRIPT" '配置已经完成，可以关闭 PowerShell 了。' "PowerShell 使用指定完成提示"
assert_contains "$SHELL_SCRIPT" '感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。' "Shell 显示感谢与客服提示"
assert_contains "$POWERSHELL_SCRIPT" '感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。' "PowerShell 显示感谢与客服提示"
assert_contains "$SHELL_SCRIPT" '写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。' "Shell 开场告知会自动关闭客户端"
assert_contains "$POWERSHELL_SCRIPT" '写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。' "PowerShell 开场告知会自动关闭客户端"
assert_contains "$SHELL_SCRIPT" '已自动关闭 ${clients}，请重新打开并新建对话测试，如有问题请联系网站客服。' "Shell 报告已自动关闭客户端"
assert_contains "$POWERSHELL_SCRIPT" '已自动关闭 ${clients}，请重新打开并新建对话测试，如有问题请联系网站客服。' "PowerShell 报告已自动关闭客户端"
assert_contains "$SHELL_SCRIPT" '未能自动关闭 ${clients}，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。' "Shell 关闭失败时提示重启电脑"
assert_contains "$POWERSHELL_SCRIPT" '未能自动关闭 ${clients}，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。' "PowerShell 关闭失败时提示重启电脑"
assert_contains "$SHELL_SCRIPT" '当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。' "Shell 报告未检测到运行中客户端"
assert_contains "$POWERSHELL_SCRIPT" '当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。' "PowerShell 报告未检测到运行中客户端"
assert_not_contains "$SHELL_SCRIPT" '接口地址：' "Shell 不再显示接口地址"
assert_not_contains "$POWERSHELL_SCRIPT" '接口地址：' "PowerShell 不再显示接口地址"
assert_not_contains "$SHELL_SCRIPT" '默认模型：' "Shell 不再显示默认模型"
assert_not_contains "$POWERSHELL_SCRIPT" '默认模型：' "PowerShell 不再显示默认模型"
assert_contains "$README_FILE" 'irm https://docs.myautoapi.com/install/myautoapi-codex-installer/install.ps1 | iex' "README 写明 Windows 完整命令"
assert_contains "$README_FILE" 'curl -fsSL https://docs.myautoapi.com/install/myautoapi-codex-installer/install.sh | bash' "README 写明 macOS/Linux 完整命令"

# reference/ 是不进 Git 的本机对照材料；其它机器上没有时跳过这两项。
if [ -f "$REFERENCE_POWERSHELL_SCRIPT" ] || [ -f "$REFERENCE_SHELL_SCRIPT" ]; then
  if [ "$(sha256sum "$REFERENCE_POWERSHELL_SCRIPT" | awk '{print $1}')" = "1dd391654588bba861cb908486af94d2e2f03bd6b9ace8b586c1f924781a5f4c" ]; then
    pass "PowerShell 原始参考文件保持不变"
  else
    fail "PowerShell 原始参考文件保持不变"
  fi
  if [ "$(sha256sum "$REFERENCE_SHELL_SCRIPT" | awk '{print $1}')" = "fa787f8dd096673a6698701342a1f006dc31aed751f6801423f9ca8418f8aedd" ]; then
    pass "Shell 原始参考文件保持不变"
  else
    fail "Shell 原始参考文件保持不变"
  fi
else
  printf 'SKIP: 本机没有 reference/original，跳过原始参考文件检查\n'
fi

sudo_home="$TEST_ROOT/sudo-home"
mkdir -p "$sudo_home"
MYAUTOAPI_TEST_UID=0 run_shell_script "$sudo_home" '' "$sudo_home/output.txt"
sudo_status=$?
if [ "$sudo_status" -eq 1 ] &&
   grep -Fq -- "请不要用 sudo 运行。脚本只应该写入当前用户的 $sudo_home/.codex。" "$sudo_home/output.txt" &&
   ! grep -Fq 'unbound variable' "$sudo_home/output.txt" &&
   [ ! -e "$sudo_home/.codex" ] && [ ! -s "$sudo_home/kill.log" ]; then
  pass "sudo 运行时显示完整路径并退出，不写配置或关闭客户端"
else
  fail "sudo 运行时显示完整路径并退出，不写配置或关闭客户端"
  cat "$sudo_home/output.txt"
fi

new_home="$TEST_ROOT/new-home"
mkdir -p "$new_home"
if run_shell_script "$new_home" 'sk-new-test\n' "$new_home/output.txt"; then
  new_output="$(<"$new_home/output.txt")"
  new_intro_block=$'欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。\n\n本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。\n\n写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。'
  new_finish_block=$'配置已经完成，可以关闭终端了。\n\n当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。\n\n感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。'
  if [ -f "$new_home/.codex/config.toml" ] &&
     [ -f "$new_home/.codex/auth.json" ] &&
     [ ! -s "$new_home/kill.log" ] &&
     [[ "$new_output" == *"$new_intro_block"* ]] &&
     [[ "$new_output" == *"$new_finish_block"* ]]; then
    pass "没有旧文件时直接创建 config.toml 和 auth.json"
  else
    fail "没有旧文件时直接创建 config.toml 和 auth.json"
  fi
else
  fail "没有安装 App 时 Shell 脚本仍可配置 CLI/App 共用文件"
fi

if [ -f "$new_home/.codex/config.toml" ]; then
  # 与快速接入助手、install.ps1 比对的是同一份期望文件：只有中转站这几行。
  if cmp -s "$new_home/.codex/config.toml" "$ROOT_DIR/tests/fixtures/expected-config.toml"; then
    pass "没有旧文件时 config.toml 只写中转站接口和默认模型 gpt-6.1-sol，与助手逐字节一致"
  else
    fail "没有旧文件时 config.toml 只写中转站接口和默认模型 gpt-6.1-sol，与助手逐字节一致"
    cat "$new_home/.codex/config.toml"
  fi

  if command -v codex >/dev/null 2>&1; then
    if env HOME="$new_home" codex features list > "$new_home/codex-parse.txt" 2>&1; then
      pass "当前本机 Codex 可以解析生成的完整配置"
    else
      fail "当前本机 Codex 可以解析生成的完整配置"
      sed -n '1,20p' "$new_home/codex-parse.txt"
    fi
  fi
fi

accept_home="$TEST_ROOT/accept-home"
mkdir -p "$accept_home/.codex"
printf 'old config\n' > "$accept_home/.codex/config.toml"
printf 'old auth\n' > "$accept_home/.codex/auth.json"
if run_shell_script "$accept_home" 'sk-accept-test\n' "$accept_home/output.txt"; then
  accept_output="$(<"$accept_home/output.txt")"
  existing_block=$'本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。\n\n检测到已有配置，会先备份；config.toml 改写中转站接口和模型，去掉以前写入的完全访问设置和其它接口，其余设置保留。\n\n写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。'
  config_backups="$(find "$accept_home/.codex" -name 'config.toml.bak.*' | wc -l)"
  auth_backups="$(find "$accept_home/.codex" -name 'auth.json.bak.*' | wc -l)"
  if [ "$config_backups" -eq 1 ] && [ "$auth_backups" -eq 1 ] &&
     grep -Fq 'sk-accept-test' "$accept_home/.codex/auth.json" &&
     [[ "$accept_output" == *"$existing_block"* ]]; then
    pass "已有配置时自动备份两个旧文件再覆盖"
  else
    fail "已有配置时自动备份两个旧文件再覆盖"
  fi
else
  fail "已有配置时 Shell 脚本自动完成"
fi

if run_shell_script "$accept_home" 'sk-second-test\n' "$accept_home/second-output.txt"; then
  config_backups="$(find "$accept_home/.codex" -name 'config.toml.bak.*' | wc -l)"
  auth_backups="$(find "$accept_home/.codex" -name 'auth.json.bak.*' | wc -l)"
  if [ "$config_backups" -eq 2 ] && [ "$auth_backups" -eq 2 ]; then
    pass "同一时间戳连续运行不会覆盖历史备份"
  else
    fail "同一时间戳连续运行不会覆盖历史备份"
  fi
else
  fail "已有同名备份时仍可安全完成"
fi

invalid_key_home="$TEST_ROOT/invalid-key-home"
mkdir -p "$invalid_key_home/.codex"
printf 'old config\n' > "$invalid_key_home/.codex/config.toml"
printf 'old auth\n' > "$invalid_key_home/.codex/auth.json"
if run_shell_script "$invalid_key_home" '\nnot-a-key\nsk-valid-test\n' "$invalid_key_home/output.txt"; then
  config_backups="$(find "$invalid_key_home/.codex" -name 'config.toml.bak.*' | wc -l)"
  auth_backups="$(find "$invalid_key_home/.codex" -name 'auth.json.bak.*' | wc -l)"
  if grep -Fq 'API Key 不能为空，请重新输入。' "$invalid_key_home/output.txt" &&
     grep -Fq 'API Key 格式错误，必须以 sk- 开头，请重新输入。' "$invalid_key_home/output.txt" &&
     grep -Fq 'sk-valid-test' "$invalid_key_home/.codex/auth.json" &&
     [ "$config_backups" -eq 1 ] && [ "$auth_backups" -eq 1 ]; then
    pass "空值和错误格式会持续提示，直到输入 sk- 开头的 Key"
  else
    fail "空值和错误格式会持续提示，直到输入 sk- 开头的 Key"
  fi
else
  fail "输入有效 Key 后脚本应继续完成"
fi

running_home="$TEST_ROOT/running-home"
mkdir -p "$running_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999901 1 1000 S /Applications/ChatGPT.app/Contents/MacOS/ChatGPT\n99999902 99999901 1000 S /Applications/ChatGPT.app/Contents/Frameworks/ChatGPT Helper (Renderer).app/Contents/MacOS/ChatGPT Helper (Renderer)\n99999903 1 1000 Ss+ /usr/local/bin/codex\n99999904 99999903 1000 S codex-code-mode\n' \
   run_shell_script "$running_home" 'sk-running-test\n' "$running_home/output.txt"; then
  running_finish_block=$'配置已经完成，可以关闭终端了。\n\n已自动关闭 ChatGPT 软件和 Codex，请重新打开并新建对话测试，如有问题请联系网站客服。\n\n感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。'
  if [[ "$(<"$running_home/output.txt")" == *"$running_finish_block"* ]] &&
     ! grep -Fq '当前未检测到 ChatGPT/Codex CLI 运行' "$running_home/output.txt" &&
     ! grep -Fq '按 Enter 继续' "$running_home/output.txt" &&
     [ "$(sort "$running_home/kill.log")" = $'-TERM 99999901 missing\n-TERM 99999903 missing' ]; then
    pass "写入配置前只结束名称为 chatgpt/codex 的进程并报告已关闭"
  else
    fail "写入配置前只结束名称为 chatgpt/codex 的进程并报告已关闭"
  fi
else
  fail "关闭运行中客户端后脚本仍正常结束"
fi

chatgpt_home="$TEST_ROOT/chatgpt-home"
mkdir -p "$chatgpt_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999911 1 1000 S /Applications/ChatGPT.app/Contents/MacOS/ChatGPT\n' \
   run_shell_script "$chatgpt_home" 'sk-chatgpt-test\n' "$chatgpt_home/output.txt" &&
   grep -Fq '已自动关闭 ChatGPT 软件，请重新打开并新建对话测试，如有问题请联系网站客服。' "$chatgpt_home/output.txt"; then
  pass "只检测到 ChatGPT 时使用对应提示"
else
  fail "只检测到 ChatGPT 时使用对应提示"
fi

codex_home="$TEST_ROOT/codex-home"
mkdir -p "$codex_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999921 1 1000 S /usr/local/bin/codex\n' \
   run_shell_script "$codex_home" 'sk-codex-test\n' "$codex_home/output.txt" &&
   grep -Fq '已自动关闭 Codex，请重新打开并新建对话测试，如有问题请联系网站客服。' "$codex_home/output.txt"; then
  pass "只检测到 Codex 时使用对应提示"
else
  fail "只检测到 Codex 时使用对应提示"
fi

stubborn_home="$TEST_ROOT/stubborn-home"
mkdir -p "$stubborn_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999931 1 1000 S /usr/local/bin/codex\n' MYAUTOAPI_TEST_PS_ALIVE=always \
   run_shell_script "$stubborn_home" 'sk-stubborn-test\n' "$stubborn_home/output.txt" &&
   [ "$(<"$stubborn_home/kill.log")" = $'-TERM 99999931 missing\n-KILL 99999931 missing' ] &&
   grep -Fq '未能自动关闭 Codex，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。' "$stubborn_home/output.txt" &&
   grep -Fq '配置已经完成，可以关闭终端了。' "$stubborn_home/output.txt"; then
  pass "进程未退出时先 TERM 再 KILL，仍未退出则提示重启电脑"
else
  fail "进程未退出时先 TERM 再 KILL，仍未退出则提示重启电脑"
fi

late_home="$TEST_ROOT/late-home"
mkdir -p "$late_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999961 1 1000 S /usr/local/bin/codex\n' MYAUTOAPI_TEST_PS_ALIVE=10 \
   run_shell_script "$late_home" 'sk-late-test\n' "$late_home/output.txt" &&
   [ "$(<"$late_home/kill.log")" = '-TERM 99999961 missing' ] &&
   grep -Fq '已自动关闭 Codex，请重新打开并新建对话测试，如有问题请联系网站客服。' "$late_home/output.txt"; then
  pass "进程在最后一次等待期间退出时不再 KILL，也不误报失败"
else
  fail "进程在最后一次等待期间退出时不再 KILL，也不误报失败"
fi

ignored_home="$TEST_ROOT/ignored-home"
mkdir -p "$ignored_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999941 1 0 S /usr/local/bin/codex\n99999942 1 1000 Zs codex\n99999943 1 1000 S codexbar\n' \
   run_shell_script "$ignored_home" 'sk-ignored-test\n' "$ignored_home/output.txt" &&
   [ ! -s "$ignored_home/kill.log" ] &&
   grep -Fq '当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。' "$ignored_home/output.txt"; then
  pass "不处理其他用户的进程、僵尸进程和名称相近的进程"
else
  fail "不处理其他用户的进程、僵尸进程和名称相近的进程"
fi

ancestor_home="$TEST_ROOT/ancestor-home"
mkdir -p "$ancestor_home"
if MYAUTOAPI_TEST_PS_OUTPUT='99999951 1 1000 S /usr/local/bin/codex\n' MYAUTOAPI_TEST_PS_ANCESTOR="$$" \
   run_shell_script "$ancestor_home" 'sk-ancestor-test\n' "$ancestor_home/output.txt" &&
   [ "$(<"$ancestor_home/kill.log")" = '-TERM 99999951 missing' ] &&
   grep -Fq '未能自动关闭 Codex，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。' "$ancestor_home/output.txt"; then
  pass "脚本运行在 Codex 内时跳过自己所在的进程并提示重启电脑"
else
  fail "脚本运行在 Codex 内时跳过自己所在的进程并提示重启电脑"
fi

real_home="$TEST_ROOT/real-home"
mkdir -p "$real_home"
sleep 30 &
real_pid=$!
if MYAUTOAPI_TEST_PS_OUTPUT="$real_pid 1 1000 S /usr/local/bin/codex\n" MYAUTOAPI_TEST_PS_ALIVE=real MYAUTOAPI_TEST_REAL_KILL_PID="$real_pid" \
   run_shell_script "$real_home" 'sk-real-test\n' "$real_home/output.txt"; then
  wait "$real_pid"
  real_status=$?
  if [ "$real_status" -gt 128 ] &&
     grep -Fq '已自动关闭 Codex，请重新打开并新建对话测试，如有问题请联系网站客服。' "$real_home/output.txt"; then
    pass "真实进程被信号结束后报告已关闭"
  else
    fail "真实进程被信号结束后报告已关闭"
  fi
else
  kill "$real_pid" 2>/dev/null
  fail "结束真实进程后脚本仍正常结束"
fi

# ---- 检查密钥、MYAUTOAPI_KEY 和 Mac 剪贴板（本地假服务器，合成密钥） ----

POWERSHELL_BIN=""
for candidate in "${PWSH:-}" "$ROOT_DIR/codex-domestic-models/.dev-tools/pwsh/pwsh" "$(command -v pwsh || true)"; do
  if [ -n "$candidate" ] && [ -x "$candidate" ]; then
    POWERSHELL_BIN="$candidate"
    break
  fi
done

# PowerShell 版同样只把检查密钥的地址换成本地假服务器；Get-Process 返回空，不碰真实进程。
POWERSHELL_PRELUDE='function Start-Sleep { }
function Get-Process { }
function Stop-Process { }
function Invoke-WebRequest {
  [CmdletBinding()] param([string]$Uri, [hashtable]$Headers, [int]$TimeoutSec, [switch]$UseBasicParsing)
  $Uri = $Uri.Replace("https://api.myautoapi.com", $env:MYAUTOAPI_TEST_SERVER)
  if (-not $env:MYAUTOAPI_TEST_SERVER -or $Uri.Contains("api.myautoapi.com")) { throw "network down" }
  if ($env:MYAUTOAPI_TEST_SHORT_TIMEOUT) { $TimeoutSec = 1 }
  Microsoft.PowerShell.Utility\Invoke-WebRequest -Uri $Uri -Headers $Headers -TimeoutSec $TimeoutSec -UseBasicParsing
}
. $env:MYAUTOAPI_TEST_SCRIPT'

run_powershell_script() {
  local home_dir="$1"
  local input="$2"
  local output_file="$3"
  printf '%b' "$input" | env HOME="$home_dir" USERPROFILE="$home_dir" XDG_CACHE_HOME="$home_dir/.cache" \
    XDG_DATA_HOME="$home_dir/.local/share" XDG_CONFIG_HOME="$home_dir/.config" \
    POWERSHELL_TELEMETRY_OPTOUT=1 POWERSHELL_UPDATECHECK=Off MYAUTOAPI_TEST_SCRIPT="$POWERSHELL_SCRIPT" \
    "$POWERSHELL_BIN" -NoLogo -NoProfile -Command "$POWERSHELL_PRELUDE" > "$output_file" 2>&1
}

KEY_CASE=0
check_key_case() {
  local runner="$1"
  local label="$2"
  local env_key="$3"
  local input="$4"
  local expected="$5"
  local written_key="$6"
  local server="${7:-$MYAUTOAPI_TEST_SERVER}"
  local home
  KEY_CASE=$((KEY_CASE + 1))
  home="$TEST_ROOT/key-case-$KEY_CASE"
  mkdir -p "$home"
  if MYAUTOAPI_KEY="$env_key" MYAUTOAPI_TEST_SERVER="$server" "$runner" "$home" "$input" "$home/output.txt" &&
     grep -Fq -- "$expected" "$home/output.txt" &&
     grep -Fq -- "\"OPENAI_API_KEY\": \"$written_key\"" "$home/.codex/auth.json" &&
     [ "$(find "$home/.codex" -name '*.bak.*' | wc -l)" -eq 0 ]; then
    pass "$label"
  else
    fail "$label"
    sed -n '1,40p' "$home/output.txt"
  fi
}

: > "$MYAUTOAPI_TEST_CURL_LOG"
for runner in run_shell_script run_powershell_script; do
  if [ "$runner" = run_shell_script ]; then
    name="Shell"
  elif [ -n "$POWERSHELL_BIN" ]; then
    name="PowerShell"
  else
    printf 'SKIP: 本机没有 PowerShell，跳过 install.ps1 的密钥检查流程\n'
    continue
  fi
  check_key_case "$runner" "$name 读取 MYAUTOAPI_KEY，检查通过后直接写入" 'sk-env-usable-0123456789abcdef' '' '密钥检查通过，余额 $12.35。' 'sk-env-usable-0123456789abcdef'
  check_key_case "$runner" "$name MYAUTOAPI_KEY 格式错误时提示并回到输入" 'not-a-key' 'sk-typed-usable-key\n' 'MYAUTOAPI_KEY 格式错误，必须以 sk- 开头，请手动输入。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name 401 时提示密钥无效并重新输入，期间不写入" 'sk-env-rejected-key' 'sk-typed-usable-key\n' '密钥无效，请回网站重新复制完整的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name 403 按密钥无效处理" '' 'sk-forbidden-key\nsk-typed-usable-key\n' '密钥无效，请回网站重新复制完整的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name isValid 为 false 时提示换密钥并重新输入" '' 'sk-disabled-key\nsk-typed-usable-key\n' '这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name status 为 expired 时按停用处理并重新输入" '' 'sk-statusexpired-key\nsk-typed-usable-key\n' '这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name status 为 quota_exhausted 时按停用处理，不当作余额为 0" '' 'sk-exhausted-key\nsk-typed-usable-key\n' '这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name 过期时间已过（status 仍为 active）时按停用处理" '' 'sk-pastdate-key\nsk-typed-usable-key\n' '这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name 过期时间未到时正常通过" '' 'sk-futuredate-key\n' '密钥检查通过，余额 $4.00。' 'sk-futuredate-key'
  check_key_case "$runner" "$name 余额为 0 时提示并继续写入" '' 'sk-zero-key\n' '密钥可用，但余额为 $0。充值后即可使用；新人进群领 2 元。' 'sk-zero-key'
  check_key_case "$runner" "$name 没有 remaining 字段时按成功处理" '' 'sk-noremaining-key\n' '密钥检查通过。' 'sk-noremaining-key'
  check_key_case "$runner" "$name 服务器 5xx 时提示服务器出错和状态码并继续写入" '' 'sk-broken-key\n' 'MyAutoAPI 服务器暂时出错（HTTP 500），暂时无法检查密钥，继续写入。' 'sk-broken-key'
  check_key_case "$runner" "$name 服务器返回的不是 JSON 时提示返回内容异常并继续写入" '' 'sk-notjson-key\n' 'MyAutoAPI 服务器暂时出错（返回内容异常），暂时无法检查密钥，继续写入。' 'sk-notjson-key'
  check_key_case "$runner" "$name 连接被拒时提示连不上并继续写入" '' 'sk-network-key\n' '连不上 MyAutoAPI 服务器（网络不通，或被代理、防火墙拦截），暂时无法检查密钥，继续写入。' 'sk-network-key' 'http://127.0.0.1:9'
  MYAUTOAPI_TEST_SHORT_TIMEOUT=1 check_key_case "$runner" "$name 服务器超时没有回应时提示网络超时并继续写入" '' 'sk-slow-key\n' '网络超时：MyAutoAPI 服务器 10 秒内没有回应，暂时无法检查密钥，继续写入。' 'sk-slow-key'
  check_key_case "$runner" "$name 原有输入流程：空值继续提示" '' '\nnot-a-key\nsk-typed-usable-key\n' 'API Key 不能为空，请重新输入。' 'sk-typed-usable-key'
  check_key_case "$runner" "$name 原有输入流程：格式错误继续提示" '' '\nnot-a-key\nsk-typed-usable-key\n' 'API Key 格式错误，必须以 sk- 开头，请重新输入。' 'sk-typed-usable-key'
done

# 密钥里混入制表符、引号、反斜杠时，auth.json 仍是合法 JSON，读回的值不变。
odd_key=$'sk-escape-usable\tkey"quote\\slash'
for runner in run_shell_script run_powershell_script; do
  if [ "$runner" = run_shell_script ]; then
    name="Shell"
  elif [ -n "$POWERSHELL_BIN" ]; then
    name="PowerShell"
  else
    continue
  fi
  home="$TEST_ROOT/escape-$name"
  mkdir -p "$home"
  if MYAUTOAPI_KEY="$odd_key" "$runner" "$home" '' "$home/output.txt" &&
     python3 -c 'import json, sys; sys.exit(json.load(open(sys.argv[1], encoding="utf-8"))["OPENAI_API_KEY"] != sys.argv[2])' "$home/.codex/auth.json" "$odd_key"; then
    pass "$name 写入 auth.json 时按 JSON 规则转义控制字符、引号和反斜杠"
  else
    fail "$name 写入 auth.json 时按 JSON 规则转义控制字符、引号和反斜杠"
    sed -n '1,40p' "$home/output.txt"
  fi
done

if grep -Fq '/v1/usage Bearer sk-env-usable-0123456789abcdef' "$USAGE_LOG"; then
  pass "检查密钥请求 GET /v1/usage，并带 Authorization: Bearer 请求头"
else
  fail "检查密钥请求 GET /v1/usage，并带 Authorization: Bearer 请求头"
fi
if [ -s "$MYAUTOAPI_TEST_CURL_LOG" ] && ! grep -Fq 'sk-' "$MYAUTOAPI_TEST_CURL_LOG" &&
   grep -Fq -- '--max-time 10' "$MYAUTOAPI_TEST_CURL_LOG"; then
  pass "Shell 密钥不出现在 curl 参数里，超时 10 秒"
else
  fail "Shell 密钥不出现在 curl 参数里，超时 10 秒"
fi
if ! grep -Fq 'api.myautoapi.com' "$USAGE_LOG"; then
  pass "测试只请求本地假服务器"
else
  fail "测试只请求本地假服务器"
fi

clipboard_case() {
  local label="$1"
  local clipboard="$2"
  local input="$3"
  local written_key="$4"
  local prompts="$5"
  shift 5
  local home
  KEY_CASE=$((KEY_CASE + 1))
  home="$TEST_ROOT/clipboard-case-$KEY_CASE"
  mkdir -p "$home"
  : > "$MYAUTOAPI_TEST_PBPASTE_LOG"
  if MYAUTOAPI_TEST_CLIPBOARD="$clipboard" run_shell_script "$home" "$input" "$home/output.txt" "$@" &&
     grep -Fq -- "\"OPENAI_API_KEY\": \"$written_key\"" "$home/.codex/auth.json" &&
     [ "$(grep -o '检测到剪贴板里的密钥' "$home/output.txt" | wc -l)" -eq "$prompts" ]; then
    pass "$label"
  else
    fail "$label"
    sed -n '1,40p' "$home/output.txt"
  fi
}

clipboard_case "Mac .app 读到剪贴板密钥，直接回车使用" '  sk-clipboard-key-0123456789-usable\n' '\n' 'sk-clipboard-key-0123456789-usable' 1 --from-app
if grep -Fq '检测到剪贴板里的密钥：sk-clipboard****9-usable，直接回车使用；要换别的就粘贴后回车：' "$TEST_ROOT/clipboard-case-$KEY_CASE/output.txt" &&
   ! grep -Fq 'sk-clipboard-key-0123456789-usable' "$TEST_ROOT/clipboard-case-$KEY_CASE/output.txt"; then
  pass "剪贴板密钥按前 12 位、****、后 8 位显示"
else
  fail "剪贴板密钥按前 12 位、****、后 8 位显示"
fi
clipboard_case "Mac .app 粘贴别的密钥时使用新输入的密钥" 'sk-clipboard-key-0123456789-usable' 'sk-typed-usable-key\n' 'sk-typed-usable-key' 1 --from-app
clipboard_case "剪贴板密钥无效时只提示一次，再走原来的输入" 'sk-clipboard-rejected-0123456789' '\nsk-typed-usable-key\n' 'sk-typed-usable-key' 1 --from-app
clipboard_case "剪贴板不是整段 sk- 密钥时走原来的输入" 'hello sk-abc\nsk-second-line' 'sk-typed-usable-key\n' 'sk-typed-usable-key' 0 --from-app
clipboard_case "在线命令不读剪贴板" 'sk-clipboard-key-0123456789-usable' 'sk-typed-usable-key\n' 'sk-typed-usable-key' 0
if [ ! -s "$MYAUTOAPI_TEST_PBPASTE_LOG" ]; then
  pass "没有 --from-app 时不调用 pbpaste"
else
  fail "没有 --from-app 时不调用 pbpaste"
fi

# ---- 不改 Codex App 的权限：先关闭 Codex 再写配置，不碰 .codex-global-state.json ----

if awk '/^main\(\) \{/ { inside = 1 } inside && /stop_running_clients/ { stop = NR } inside && /write_codex_files "\$api_key"/ { write = NR } END { exit !(stop && write && stop < write) }' "$SHELL_SCRIPT"; then
  pass "Shell 先关闭 Codex，再写配置"
else
  fail "Shell 先关闭 Codex，再写配置"
fi
if awk '/^\$StopResult = Stop-RunningClients/ { stop = NR } /^Write-CodexFiles -ApiKey/ { write = NR } END { exit !(stop && write && stop < write) }' "$POWERSHELL_SCRIPT"; then
  pass "PowerShell 先关闭 Codex，再写配置"
else
  fail "PowerShell 先关闭 Codex，再写配置"
fi
assert_not_contains "$SHELL_SCRIPT" 'codex-global-state' "Shell 不改 Codex App 权限"
assert_not_contains "$POWERSHELL_SCRIPT" 'codex-global-state' "PowerShell 不改 Codex App 权限"

for runner in run_shell_script run_powershell_script; do
  if [ "$runner" = run_shell_script ]; then
    name="Shell"
  elif [ -n "$POWERSHELL_BIN" ]; then
    name="PowerShell"
  else
    printf 'SKIP: 本机没有 PowerShell，跳过 install.ps1 的 App 权限检查\n'
    continue
  fi

  home="$TEST_ROOT/app-new-$name"
  mkdir -p "$home"
  if MYAUTOAPI_KEY='sk-env-usable-0123456789abcdef' "$runner" "$home" '' "$home/output.txt" &&
     grep -Fq '配置已经完成' "$home/output.txt" &&
     [ ! -e "$home/.codex/.codex-global-state.json" ]; then
    pass "$name 不新建 App 状态文件"
  else
    fail "$name 不新建 App 状态文件"
    sed -n '1,40p' "$home/output.txt"
  fi

  home="$TEST_ROOT/app-old-$name"
  mkdir -p "$home/.codex"
  state='{"electron-persisted-atom-state":{"agent-mode-by-host-id":{"local":"auto"}}}'
  printf '%s' "$state" > "$home/.codex/.codex-global-state.json"
  if MYAUTOAPI_KEY='sk-env-usable-0123456789abcdef' "$runner" "$home" '' "$home/output.txt" &&
     grep -Fq '配置已经完成' "$home/output.txt" &&
     [ "$(cat "$home/.codex/.codex-global-state.json")" = "$state" ] &&
     [ "$(find "$home/.codex" -name '.codex-global-state.json*' | wc -l)" -eq 1 ]; then
    pass "$name 已有 App 状态文件原样不动"
  else
    fail "$name 已有 App 状态文件原样不动"
    sed -n '1,40p' "$home/output.txt"
  fi
done

printf '\n%d passed, %d failed\n' "$PASS_COUNT" "$FAIL_COUNT"
if [ "$FAIL_COUNT" -ne 0 ]; then
  exit 1
fi
