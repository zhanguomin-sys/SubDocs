#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHELL_SCRIPT="$ROOT_DIR/install/install.sh"
POWERSHELL_SCRIPT="$ROOT_DIR/install/install.ps1"
README_FILE="$ROOT_DIR/README.md"
REFERENCE_SHELL_SCRIPT="$ROOT_DIR/reference/original/codex.sh"
REFERENCE_POWERSHELL_SCRIPT="$ROOT_DIR/reference/original/codex.ps1"

MYAUTOAPI_TEST_REAL_PS="$(command -v ps)"
export MYAUTOAPI_TEST_REAL_PS

PASS_COUNT=0
FAIL_COUNT=0
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT

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
  printf '1000\n'
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
printf '20260722123456\n'
EOF
  chmod +x "$bin_dir/date"
  cat > "$bin_dir/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$bin_dir/sleep"
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
    printf '%b' "$input" | env HOME="$home_dir" PATH="$bin_dir:$PATH" MYAUTOAPI_TEST_KILL_LOG="$home_dir/kill.log" bash "$SHELL_SCRIPT" > "$output_file" 2>&1
  )
}

assert_contains "$SHELL_SCRIPT" 'CODEX_HOME="$HOME/.codex"' "Shell 固定使用普通客户的 ~/.codex"
assert_not_contains "$SHELL_SCRIPT" 'CODEX_HOME="${CODEX_HOME:-' "Shell 不处理自定义 CODEX_HOME"
assert_contains "$SHELL_SCRIPT" 'model_provider = "OpenAI"' "Shell provider 与网站配置一致"
assert_contains "$SHELL_SCRIPT" '[model_providers.OpenAI]' "Shell provider 表名与选择值一致"
assert_contains "$POWERSHELL_SCRIPT" 'model_provider = "OpenAI"' "PowerShell provider 与网站配置一致"
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
   [ "$powershell_write_line" -lt "$powershell_stop_line" ] &&
   [ "$powershell_stop_line" -lt "$powershell_finish_line" ]; then
  pass "PowerShell 先写入配置，再关闭进程，最后显示完成提示"
else
  fail "PowerShell 先写入配置，再关闭进程，最后显示完成提示"
fi
assert_not_contains "$POWERSHELL_SCRIPT" '-AsSecureString' "PowerShell API Key 使用明文输入"
assert_contains "$POWERSHELL_SCRIPT" 'while (Test-Path -LiteralPath $backup)' "PowerShell 不覆盖同名历史备份"
assert_not_contains "$SHELL_SCRIPT" 'confirm_existing_files' "Shell 不再要求二次确认"
assert_not_contains "$POWERSHELL_SCRIPT" 'Confirm-ExistingFiles' "PowerShell 不再要求二次确认"
assert_not_contains "$SHELL_SCRIPT" 'read_input ignored' "Shell 不因运行中客户端等待"
assert_not_contains "$POWERSHELL_SCRIPT" 'Read-Host "关闭后按 Enter 继续"' "PowerShell 不因运行中客户端等待"
assert_contains "$SHELL_SCRIPT" 'report_running_clients' "Shell 写入后检查客户端运行状态"
assert_contains "$POWERSHELL_SCRIPT" 'Report-RunningClients' "PowerShell 写入后检查客户端运行状态"
assert_contains "$SHELL_SCRIPT" 'sleep 0.5' "Shell 每条非空提示间隔 0.5 秒"
assert_contains "$POWERSHELL_SCRIPT" 'Start-Sleep -Milliseconds 500' "PowerShell 每条非空提示间隔 0.5 秒"
assert_contains "$SHELL_SCRIPT" '欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。' "Shell 使用指定欢迎语"
assert_contains "$POWERSHELL_SCRIPT" '欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。' "PowerShell 使用指定欢迎语"
assert_contains "$SHELL_SCRIPT" '本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json 文件。' "Shell 使用指定脚本说明"
assert_contains "$POWERSHELL_SCRIPT" '本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json 文件。' "PowerShell 使用指定脚本说明"
assert_contains "$SHELL_SCRIPT" '请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认' "Shell 使用指定 API Key 提示"
assert_contains "$POWERSHELL_SCRIPT" '请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认' "PowerShell 使用指定 API Key 提示"
assert_contains "$SHELL_SCRIPT" '配置已经完成，可以关闭终端了。' "Shell 使用指定完成提示"
assert_contains "$POWERSHELL_SCRIPT" '配置已经完成，可以关闭 PowerShell 了。' "PowerShell 使用指定完成提示"
assert_contains "$SHELL_SCRIPT" '感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。' "Shell 显示感谢与客服提示"
assert_contains "$POWERSHELL_SCRIPT" '感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。' "PowerShell 显示感谢与客服提示"
assert_contains "$SHELL_SCRIPT" '配置完成后会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。' "Shell 开场告知会自动关闭客户端"
assert_contains "$POWERSHELL_SCRIPT" '配置完成后会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。' "PowerShell 开场告知会自动关闭客户端"
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

new_home="$TEST_ROOT/new-home"
mkdir -p "$new_home"
if run_shell_script "$new_home" 'sk-new-test\n' "$new_home/output.txt"; then
  new_output="$(<"$new_home/output.txt")"
  new_intro_block=$'欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。\n\n本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json 文件。\n\n配置完成后会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。'
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
  assert_contains "$new_home/.codex/config.toml" 'model = "gpt-5.6-sol"' "默认模型为 gpt-5.6-sol"
  assert_contains "$new_home/.codex/config.toml" 'review_model = "gpt-5.6-sol"' "review_model 保持完整"
  assert_contains "$new_home/.codex/config.toml" 'model_reasoning_effort = "high"' "默认推理强度为 high"
  assert_contains "$new_home/.codex/config.toml" 'network_access = "enabled"' "保留用户指定的网络配置"
  assert_contains "$new_home/.codex/config.toml" '[model_providers.OpenAI]' "生成配置保留完整自定义 provider 段"
  assert_contains "$new_home/.codex/config.toml" 'requires_openai_auth = true' "生成配置使用 auth.json 中的 API Key 鉴权"
  assert_not_contains "$new_home/.codex/config.toml" 'x-openai-actor-authorization' "生成配置不写入生图请求头"
  assert_contains "$new_home/.codex/config.toml" 'goals = true' "生成配置开启 goals"
  assert_contains "$new_home/.codex/config.toml" 'image_generation = true' "生成配置开启图片生成"
  assert_not_contains "$new_home/.codex/config.toml" 'web_search' "不额外写入 web_search"
  assert_not_contains "$new_home/.codex/config.toml" 'sandbox_mode' "不写入 sandbox_mode"
  assert_not_contains "$new_home/.codex/config.toml" 'approval_policy' "不写入 approval_policy"

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
  existing_block=$'本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json 文件。\n\n检测到已有旧配置，将会在备份后再覆盖。\n\n配置完成后会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。'
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
     [ "$(sort "$running_home/kill.log")" = $'-TERM 99999901 written\n-TERM 99999903 written' ]; then
    pass "写入配置后只结束名称为 chatgpt/codex 的进程并报告已关闭"
  else
    fail "写入配置后只结束名称为 chatgpt/codex 的进程并报告已关闭"
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
   [ "$(<"$stubborn_home/kill.log")" = $'-TERM 99999931 written\n-KILL 99999931 written' ] &&
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
   [ "$(<"$late_home/kill.log")" = '-TERM 99999961 written' ] &&
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
   [ "$(<"$ancestor_home/kill.log")" = '-TERM 99999951 written' ] &&
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

printf '\n%d passed, %d failed\n' "$PASS_COUNT" "$FAIL_COUNT"
if [ "$FAIL_COUNT" -ne 0 ]; then
  exit 1
fi
