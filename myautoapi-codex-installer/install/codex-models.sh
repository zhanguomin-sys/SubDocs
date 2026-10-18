#!/bin/sh
set -eu

say() { printf '%s\n' "$*"; }

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

target_codex_home="${CODEX_HOME:-$HOME/.codex}"
app_backend=""
model_action="import"
script_platform="$(uname -s)"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --uninstall) model_action="uninstall"; shift ;;
    --codex-home|--codex-path)
      [ "$#" -ge 2 ] || { say "参数缺少路径。" >&2; exit 1; }
      if [ "$1" = --codex-home ]; then target_codex_home="$2"; else app_backend="$2"; fi
      shift 2 ;;
    --help)
      say '用法：sh codex-models.sh [--uninstall] [--codex-home 配置目录] [--codex-path Codex后端路径]'
      exit 0 ;;
    *) say "未知参数。" >&2; exit 1 ;;
  esac
done
case "$script_platform" in
  Darwin) run_model_import() { /usr/bin/osascript -l JavaScript - "$@"; } ;;
  Linux)
    js_runtime=""
    for runtime_name in node nodejs; do
      runtime_candidate="$(command -v "$runtime_name" || true)"
      if [ -n "$runtime_candidate" ] && "$runtime_candidate" --input-type=commonjs -e 'process.exit(Number(process.versions.node.split(".")[0]) >= 16 ? 0 : 1)' >/dev/null 2>&1; then
        js_runtime="$runtime_candidate"
        break
      fi
    done
    [ -n "$js_runtime" ] || {
      say 'Linux 需要本机已有 Node.js 16 或以上版本；未修改配置，也不会自动安装依赖。' >&2
      exit 1
    }
    run_model_import() { "$js_runtime" --input-type=commonjs - "$@"; }
    ;;
  *) say '这个脚本用于 Linux/macOS；Windows 请运行 codex-models.ps1。' >&2; exit 1 ;;
esac
if [ "$model_action" = uninstall ]; then
  say '移除当前启用目录中的国产模型后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
else
  say '成功导入后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
fi
if [ "$model_action" = import ]; then
if [ -z "$app_backend" ] && [ "$script_platform" = Darwin ]; then
  for app_root in /Applications/Codex.app /Applications/ChatGPT.app "$HOME/Applications/Codex.app" "$HOME/Applications/ChatGPT.app"; do
    if [ -f "$app_root/Contents/Info.plist" ]; then
      bundle_id="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$app_root/Contents/Info.plist" 2>/dev/null || true)"
      [ "$bundle_id" = com.openai.codex ] || continue
      for candidate in "$app_root/Contents/Resources/codex" "$app_root/Contents/Resources/bin/codex"; do
        if [ -f "$candidate" ] && [ -x "$candidate" ]; then app_backend="$candidate"; break; fi
      done
    fi
    [ -z "$app_backend" ] || break
  done
fi
if [ -z "$app_backend" ] && [ "$script_platform" = Linux ]; then
  # Prefer the running App's backend when it is available, then the existing codex command.
  while read -r client_pid client_kind client_action; do
    [ "$client_kind" = codex ] || continue
    candidate="$(readlink "/proc/$client_pid/exe" 2>/dev/null || true)"
    case "$candidate" in
      */resources/*|*/Resources/*)
        if [ -f "$candidate" ] && [ -x "$candidate" ]; then app_backend="$candidate"; break; fi
        ;;
    esac
  done <<EOF
$(list_running_clients || true)
EOF
fi
if [ -z "$app_backend" ]; then app_backend="$(command -v codex || true)"; fi
[ -n "$app_backend" ] && [ -f "$app_backend" ] && [ -x "$app_backend" ] || {
  say '未找到 Codex 后端；请先安装 Codex，或用 --codex-path 指定现有的 Codex 后端。' >&2
  exit 1
}
fi

if import_message="$(run_model_import "$target_codex_home" "$app_backend" "$model_action" <<'MYAUTOAPI_JAVASCRIPT'
var DOMESTIC_CATALOG = {"models":[{"slug":"deepseek-flash","display_name":"DeepSeek V4.1 Flash","description":"DeepSeek Flash","context_window":1048576,"input_modalities":["text","image"],"default_reasoning_level":"high","apply_patch_tool_type":"freeform","supports_parallel_tool_calls":true,"support_verbosity":true,"default_verbosity":"low","supports_image_detail_original":true,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"high","description":"增强推理"},{"effort":"max","description":"最大推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":100,"availability_nux":null,"upgrade":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"max_context_window":1048576,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"deepseek-v4.1-flash-1017","display_name":"deepseek-v4.1-flash-1017","description":"DeepSeek V4.1 Flash 1017 多模态模型","context_window":1048576,"input_modalities":["text","image"],"default_reasoning_level":"high","apply_patch_tool_type":"freeform","supports_parallel_tool_calls":true,"support_verbosity":true,"default_verbosity":"low","supports_image_detail_original":true,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"high","description":"增强推理"},{"effort":"max","description":"最大推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":101,"availability_nux":null,"upgrade":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"max_context_window":1048576,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"glm-5.3","display_name":"GLM-5.3","description":"智谱文本模型","context_window":1000000,"input_modalities":["text"],"default_reasoning_level":"max","apply_patch_tool_type":null,"supports_parallel_tool_calls":false,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"high","description":"增强推理"},{"effort":"max","description":"最大推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":102,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"glm-5.3-flash","display_name":"GLM-5.3 Flash","description":"智谱多模态 Flash","context_window":1000000,"input_modalities":["text","image"],"default_reasoning_level":"max","apply_patch_tool_type":null,"supports_parallel_tool_calls":false,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"high","description":"增强推理"},{"effort":"max","description":"最大推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":103,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"qwen3.8-max","display_name":"Qwen3.8 Max","description":"千问 Max","context_window":1000000,"input_modalities":["text","image"],"default_reasoning_level":"xhigh","apply_patch_tool_type":null,"supports_parallel_tool_calls":true,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"medium","description":"中度推理"},{"effort":"xhigh","description":"深度推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":104,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"qwen3.8-flash","display_name":"Qwen3.8 Flash","description":"千问 Flash","context_window":1000000,"input_modalities":["text","image"],"default_reasoning_level":"xhigh","apply_patch_tool_type":null,"supports_parallel_tool_calls":true,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"medium","description":"中度推理"},{"effort":"xhigh","description":"深度推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":105,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"kimi-k3","display_name":"Kimi K3","description":"月之暗面多模态模型","context_window":1000000,"input_modalities":["text","image"],"default_reasoning_level":"max","apply_patch_tool_type":null,"supports_parallel_tool_calls":false,"supported_reasoning_levels":[{"effort":"low","description":"轻度推理"},{"effort":"high","description":"增强推理"},{"effort":"max","description":"最大推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":106,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"minimax-m3","display_name":"MiniMax M3","description":"MiniMax 多模态模型","context_window":1000000,"input_modalities":["text","image"],"default_reasoning_level":null,"apply_patch_tool_type":null,"supports_parallel_tool_calls":false,"supported_reasoning_levels":[],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":107,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":false,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"hy4-preview","display_name":"Hy4 preview","description":"腾讯混元","context_window":1000000,"input_modalities":["text"],"default_reasoning_level":"high","apply_patch_tool_type":null,"supports_parallel_tool_calls":false,"supported_reasoning_levels":[{"effort":"none","description":"关闭推理"},{"effort":"high","description":"增强推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":108,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"supports_image_detail_original":false,"max_context_window":1000000,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."},"use_responses_lite":false},{"slug":"mimo-v2.6-pro","display_name":"MiMo-V2.6-Pro","description":"小米全模态旗舰模型","context_window":1048576,"input_modalities":["text","image"],"default_reasoning_level":"low","apply_patch_tool_type":"freeform","supports_parallel_tool_calls":false,"supports_image_detail_original":true,"use_responses_lite":true,"supported_reasoning_levels":[{"effort":"none","description":"关闭推理"},{"effort":"low","description":"轻度推理"},{"effort":"medium","description":"中度推理"},{"effort":"high","description":"增强推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":109,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"max_context_window":1048576,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."}},{"slug":"mimo-v2.6-flash","display_name":"MiMo-V2.6-Flash","description":"小米低成本全模态模型","context_window":1048576,"input_modalities":["text","image"],"default_reasoning_level":"low","apply_patch_tool_type":"freeform","supports_parallel_tool_calls":false,"supports_image_detail_original":true,"use_responses_lite":true,"supported_reasoning_levels":[{"effort":"none","description":"关闭推理"},{"effort":"low","description":"轻度推理"},{"effort":"medium","description":"中度推理"},{"effort":"high","description":"增强推理"}],"shell_type":"shell_command","visibility":"list","supported_in_api":true,"priority":110,"availability_nux":null,"upgrade":null,"support_verbosity":false,"default_verbosity":null,"supports_reasoning_summaries":true,"supports_reasoning_summary_parameter":false,"default_reasoning_summary":"none","prefer_websockets":false,"web_search_tool_type":"text","supports_search_tool":false,"truncation_policy":{"mode":"tokens","limit":10000},"max_context_window":1048576,"auto_compact_token_limit":null,"effective_context_window_percent":95,"experimental_supported_tools":[],"base_instructions":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence.","model_messages":{"instructions_template":"You are a coding assistant running in Codex. Follow the user's instructions and the repository's AGENTS.md. Inspect relevant files before editing, use the available tools to complete the task, preserve unrelated changes, and verify changes with appropriate checks. Give concise, accurate progress updates and a final report. Do not claim to have run checks or completed actions without evidence."}}]};
/* Pure ES5: used by macOS JXA and the isolated test runner. */
var DomesticModels = (function () {
    'use strict';
    var effortOrder = ['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'ultra', 'max'];
    function fail(message) { var error = new Error(message); error.safeMessage = message; throw error; }
    function atStep(name, target, action) {
        try { return action(); }
        catch (error) {
            if (!error || typeof error !== 'object') { error = new Error('Unexpected failure'); }
            if (!error.importStep) { error.importStep = name; }
            if (!error.importTarget && target) { error.importTarget = target; }
            throw error;
        }
    }
    function backendFailure(stderr, detail) {
        detail = detail || {}; stderr = String(stderr || '');
        var reason = 'Codex 后端命令执行失败。';
        var toml = /TOML parse error at line (\d+), column (\d+)/i.exec(stderr) || /config\.toml:(\d+):(\d+):/i.exec(stderr);
        var missing = /missing field [`']([a-z_][a-z0-9_]{0,80})[`']/i.exec(stderr);
        var osError = /os error (\d+)/i.exec(stderr);
        if (detail.code === 'ETIMEDOUT') { reason = 'Codex 后端命令执行超过 30 秒。'; }
        else if (detail.code === 'ENOENT') { reason = '无法启动 Codex 后端：找不到可执行文件或启动器解释器。'; }
        else if (detail.code === 'EACCES') { reason = '无法启动 Codex 后端：没有执行权限。'; }
        else if (toml) { reason = 'Codex 报告 TOML 配置错误：第 ' + toml[1] + ' 行，第 ' + toml[2] + ' 列。'; }
        else if (missing) {
            var knownFields = ['slug', 'display_name', 'description', 'supported_reasoning_levels', 'shell_type', 'visibility', 'supported_in_api', 'priority', 'support_verbosity', 'default_verbosity', 'apply_patch_tool_type', 'truncation_policy', 'experimental_supported_tools', 'base_instructions', 'model_messages', 'instructions_template'];
            reason = knownFields.indexOf(missing[1]) >= 0 ? 'Codex 报告模型目录缺少字段：' + missing[1] + '。' : 'Codex 报告模型目录缺少必需字段。';
        }
        else if (/Model provider [`'][^\r\n]*[`'] not found/i.test(stderr)) { reason = 'model_provider 指向的服务名称未在 model_providers 中定义，请核对名称和大小写。'; }
        else if (/unrecognized subcommand|unknown subcommand|unexpected argument|unknown option/i.test(stderr)) { reason = 'Codex 后端未识别模型目录导出命令。'; }
        else if (/unknown variant|invalid value.*expected/i.test(stderr)) { reason = '模型目录含有当前 Codex 无法识别的字段值。'; }
        else if (/failed to read model[_ ]instructions[_ ]file/i.test(stderr)) { reason = 'Codex 无法读取配置中的 model_instructions_file。'; }
        else if (/failed to (?:read|parse) model_catalog_json/i.test(stderr)) { reason = 'Codex 无法读取或解析 model_catalog_json。'; }
        else if (/\b401\b|unauthorized|invalid api key/i.test(stderr)) { reason = 'Codex 后端请求被拒绝：鉴权失败。'; }
        else if (/connection refused|failed to connect|error sending request/i.test(stderr)) { reason = 'Codex 后端连接接口失败。'; }
        else if (osError) { reason = 'Codex 后端报告系统错误，os error ' + osError[1] + '。'; }
        var error = new Error(reason); error.safeMessage = reason;
        error.code = detail.code; error.exitStatus = detail.status; error.signal = detail.signal;
        error.configSyntax = !!toml && /TOML parse error|duplicate key|invalid table header|unterminated|expected a newline|expected newline/i.test(stderr);
        return error;
    }
    function formatError(error, action) {
        if (!error || typeof error !== 'object') { error = new Error('Unexpected failure'); }
        var messages = {
            EACCES: '权限不足，无法执行该文件操作。', EPERM: '系统拒绝该操作。',
            ENOENT: '文件或目录不存在。', ENOTDIR: '路径中的某一项不是目录。',
            EISDIR: '需要普通文件，实际路径是目录。', ENOSPC: '磁盘空间不足。',
            EROFS: '目标文件系统只读。', EEXIST: '目标路径已经存在。',
            EBUSY: '目标文件被占用，或是不能直接替换的挂载文件。',
            ELOOP: '路径包含循环符号链接。', ENAMETOOLONG: '文件路径过长。',
            ERR_FS_FILE_TOO_LARGE: '文件过大，无法完整读取。',
            EIO: '文件系统读写失败。', EMFILE: '打开的文件过多。',
            ERR_ENCODING_INVALID_ENCODED_DATA: '文件包含无效的 UTF-8 字节。',
            ERR_ACCESS_DENIED: 'Node.js 的运行权限限制拒绝了该操作。'
        };
        var code = error && typeof error.code === 'string' && /^[A-Z][A-Z0-9_]{1,60}$/.test(error.code) ? error.code : '';
        var reason = error.safeMessage || messages[code];
        if (!reason) {
            var kind = ['TypeError', 'ReferenceError', 'RangeError', 'SyntaxError', 'Error'].indexOf(error.name) >= 0 ? error.name : 'Error';
            reason = '脚本内部异常（' + kind + '）。';
        }
        var lines = [action === 'uninstall' ? '卸载失败' : '导入失败', '失败步骤：' + (error.importStep || (action === 'uninstall' ? '初始化模型卸载' : '初始化模型导入'))];
        if (error.importTarget) { lines.push('文件或命令：' + error.importTarget); }
        lines.push('原因：' + reason + (code ? ' [' + code + ']' : ''));
        if (typeof error.exitStatus === 'number') { lines.push('后端退出码：' + error.exitStatus); }
        if (typeof error.systemCode === 'number') { lines.push('系统错误码：' + error.systemCode); }
        if (error.signal && /^SIG[A-Z0-9]+$/.test(error.signal)) { lines.push('后端信号：' + error.signal); }
        if (!error.safeMessage && !messages[code] && typeof error.stack === 'string') {
            var location = /(?:\[stdin\]|core\.js|linux-host\.js):(\d+):(\d+)/.exec(error.stack);
            if (location) { lines.push('脚本位置：' + location[0]); }
        }
        return lines.join('\n');
    }
    function own(obj, key) { return Object.prototype.hasOwnProperty.call(obj, key); }
    function dict() { return Object.create(null); }
    function canonical(value) {
        if (Array.isArray(value)) { return value.map(canonical); }
        if (value && typeof value === 'object') {
            var result = dict(); Object.keys(value).sort().forEach(function (key) { result[key] = canonical(value[key]); }); return result;
        }
        return value;
    }
    function stringValue(raw) {
        raw = raw.trim();
        if (/^'[^'\r\n]*'$/.test(raw)) { return raw.slice(1, -1); }
        if (/^"(?:[^"\\\r\n]|\\.)*"$/.test(raw)) {
            try {
                return JSON.parse(raw.replace(/\\(?:U[0-9a-fA-F]{8}|e|[\s\S])/g, function (token) {
                    if (token === '\\e') { return '\\u001b'; }
                    if (token.charAt(1) !== 'U' || token.length !== 10) { return token; }
                    var n = parseInt(token.slice(2), 16);
                    if (n > 0x10ffff || (n >= 0xd800 && n <= 0xdfff)) { fail('TOML 路径转义无效。'); }
                    var value = n <= 0xffff ? String.fromCharCode(n) : String.fromCharCode(0xd800 + ((n - 0x10000) >> 10), 0xdc00 + ((n - 0x10000) & 1023));
                    return JSON.stringify(value).slice(1, -1);
                }));
            } catch (e) { fail('TOML 字符串格式无效。'); }
        }
        fail('模型目录、默认模型等目标字段须使用单行 TOML 字符串。');
    }
    function keyParts(raw) {
        var parts = [], pos = 0, m, re = /\s*("(?:[^"\\]|\\.)*"|'[^']*'|[A-Za-z0-9_-]+)\s*/g;
        while (pos < raw.length) {
            re.lastIndex = pos; m = re.exec(raw);
            if (!m || m.index !== pos) { fail('无法识别 TOML 字段或配置段。'); }
            parts.push(/^["']/.test(m[1]) ? stringValue(m[1]) : m[1]);
            pos = re.lastIndex;
            if (pos === raw.length) { break; }
            if (raw.charAt(pos++) !== '.') { fail('TOML 字段路径无效。'); }
        }
        if (!parts.length) { fail('空 TOML 字段。'); }
        return parts;
    }
    function statements(text) {
        var records = [], start = 0, depth = [], quote = '', triple = false, escaped = false, comment = false, clean = '', i, ch, run;
        for (i = 0; i < text.length; i++) {
            ch = text.charAt(i);
            if (comment) {
                if (ch !== '\n') { clean += ' '; continue; }
                comment = false;
            }
            if (quote) {
                clean += ch;
                if (escaped) { escaped = false; continue; }
                if (quote === '"' && ch === '\\') { escaped = true; continue; }
                if (ch === quote) {
                    if (!triple) { quote = ''; }
                    else if (text.substr(i, 3) === quote + quote + quote) {
                        run = 3;
                        while (run < 5 && text.charAt(i + run) === quote) { run++; }
                        clean += text.substr(i + 1, run - 1); i += run - 1; quote = ''; triple = false;
                    }
                } else if (ch === '\n' && !triple) { fail('TOML 字符串未闭合。'); }
                continue;
            }
            if (ch === '#') { comment = true; clean += ' '; continue; }
            if (ch === '"' || ch === "'") {
                quote = ch; triple = text.substr(i, 3) === ch + ch + ch;
                clean += ch;
                if (triple) { clean += ch + ch; i += 2; }
                continue;
            }
            if (ch === '[' || ch === '{') { depth.push(ch); }
            if (ch === ']' || ch === '}') {
                if (depth.pop() !== (ch === ']' ? '[' : '{')) { fail('TOML 括号不匹配。'); }
            }
            clean += ch;
            if (ch === '\n' && !depth.length) { records.push({start: start, end: i + 1, code: clean}); start = i + 1; clean = ''; }
        }
        if (quote || depth.length) { fail('TOML 字符串或括号未闭合。'); }
        if (start < text.length) { records.push({start: start, end: text.length, code: clean}); }
        return records;
    }
    function configInfo(text, readOnly) {
        var result = {values: dict(), edits: dict(), firstHeader: text.length, desktop: null, desktopInline: false, desktopDotted: false, profileCatalog: false}, scope = [], records = statements(text);
        records.forEach(function (rec) {
            var code = rec.code.trim(), header, eq, parts, path, key, vstart, vend;
            if (!code) { return; }
            if (code.charAt(0) === '[') {
                header = /^(\[\[?)([\s\S]*?)(\]\]?)$/.exec(code);
                if (!header || header[1].length !== header[3].length) { fail('TOML 配置段无效。'); }
                scope = keyParts(header[2].trim());
                result.firstHeader = Math.min(result.firstHeader, rec.start);
                if (scope.join('.') === 'desktop' && scope.length === 1) {
                    if (header[1] === '[[' || result.desktop) { fail('desktop 配置段重复或格式无效。'); }
                    result.desktop = rec;
                }
                return;
            }
            /* '=' inside a quoted key is allowed. */
            var q = '', esc = false;
            for (eq = 0; eq < rec.code.length; eq++) {
                var c = rec.code.charAt(eq);
                if (q) { if (esc) { esc = false; } else if (c === '\\' && q === '"') { esc = true; } else if (c === q) { q = ''; } }
                else if (c === '"' || c === "'") { q = c; }
                else if (c === '=') { break; }
            }
            if (eq === rec.code.length) { fail('TOML 配置行无效。'); }
            parts = keyParts(rec.code.slice(0, eq).trim()); path = scope.concat(parts); key = path.join('.');
            if (!scope.length && parts[0] === 'desktop') {
                result.desktopInline = parts.length === 1;
                result.desktopDotted = parts.length > 1 || result.desktopDotted;
            }
            if (path.length > 1 && path[0] === 'profiles' && path[path.length - 1] === 'model_catalog_json') {
                result.profileCatalog = true;
                if (!readOnly) { fail('profile 中另有 model_catalog_json；请先统一目录配置再导入。'); }
            }
            if ((path.length === 1 && ['model_catalog_json', 'model', 'review_model', 'profile', 'model_provider'].indexOf(key) >= 0) || (path.length === 2 && key === 'desktop.enabled-reasoning-efforts')) {
                if (own(result.edits, key)) { fail('目标 TOML 字段重复。'); }
                vstart = eq + 1;
                while (/\s/.test(rec.code.charAt(vstart)) && vstart < rec.code.length) { vstart++; }
                vend = rec.code.replace(/\s+$/, '').length;
                result.edits[key] = {start: rec.start + vstart, end: rec.start + vend};
                result.values[key] = rec.code.slice(vstart, vend);
            }
        });
        if (result.desktopInline && !readOnly) { fail('desktop 使用内联表；请改为 [desktop] 配置段后导入。'); }
        return result;
    }
    function stringArray(raw) {
        var body = raw.trim(), result = [], parts;
        if (body.charAt(0) !== '[' || body.slice(-1) !== ']') { fail('桌面推理档位必须是字符串数组。'); }
        body = body.slice(1, -1);
        parts = statements('x=[' + body + ']')[0]; // check balanced quotes before splitting
        var re = /\s*("(?:[^"\\]|\\.)*"|'[^']*')\s*(,|$)/g, pos = 0, m;
        while (pos < body.length && body.slice(pos).trim()) {
            re.lastIndex = pos; m = re.exec(body);
            if (!m || m.index !== pos) { fail('桌面推理档位必须是字符串数组。'); }
            result.push(stringValue(m[1])); pos = re.lastIndex;
        }
        return result;
    }
    function patchConfig(text, info, catalogPath, efforts) {
        var bom = text.charAt(0) === '\uFEFF' ? '\uFEFF' : '';
        if (bom) { text = text.slice(1); info = configInfo(text); }
        var newline = text.indexOf('\r\n') >= 0 ? '\r\n' : '\n', changes = [], rootLines = [];
        function set(key, value) {
            if (own(info.edits, key)) { changes.push({start: info.edits[key].start, end: info.edits[key].end, value: value}); return true; }
            return false;
        }
        if (!set('model_catalog_json', JSON.stringify(catalogPath))) { rootLines.push('model_catalog_json = ' + JSON.stringify(catalogPath)); }
        if (!set('desktop.enabled-reasoning-efforts', JSON.stringify(efforts))) {
            var line = 'enabled-reasoning-efforts = ' + JSON.stringify(efforts) + newline;
            if (info.desktop) { changes.push({start: info.desktop.end, end: info.desktop.end, value: (text.charAt(info.desktop.end - 1) === '\n' ? '' : newline) + line}); }
            else if (info.desktopDotted) { rootLines.push('desktop.' + line.trim()); }
            else { changes.push({start: text.length, end: text.length, value: (text && text.slice(-1) !== '\n' ? newline : '') + newline + '[desktop]' + newline + line}); }
        }
        if (rootLines.length) {
            var at = info.firstHeader;
            changes.push({start: at, end: at, value: (at && text.charAt(at - 1) !== '\n' ? newline : '') + rootLines.join(newline) + newline});
        }
        changes.forEach(function (change, index) { change.order = index; });
        changes.sort(function (a, b) { return b.start - a.start || b.end - a.end || a.order - b.order; });
        changes.forEach(function (change) { text = text.slice(0, change.start) + change.value + text.slice(change.end); });
        configInfo(text);
        return bom + text;
    }
    function validateCatalog(catalog) {
        if (!catalog || !Array.isArray(catalog.models) || !catalog.models.length) { fail('模型目录必须包含非空 models 数组。'); }
        var seen = dict();
        catalog.models.forEach(function (model, modelIndex) {
            var location = 'models[' + modelIndex + ']';
            if (!model || typeof model.slug !== 'string' || !model.slug || own(seen, model.slug)) { fail('模型目录 ' + location + '.slug 为空或重复。'); }
            seen[model.slug] = true;
            var invalid = [];
            [['display_name', typeof model.display_name === 'string'], ['supported_reasoning_levels', Array.isArray(model.supported_reasoning_levels)], ['shell_type', Boolean(model.shell_type)], ['visibility', Boolean(model.visibility)], ['supported_in_api', typeof model.supported_in_api === 'boolean'], ['priority', typeof model.priority === 'number'], ['support_verbosity', typeof model.support_verbosity === 'boolean'], ['truncation_policy', Boolean(model.truncation_policy)], ['experimental_supported_tools', Array.isArray(model.experimental_supported_tools)]].forEach(function (field) { if (!field[1]) { invalid.push(field[0]); } });
            if (invalid.length) { fail('模型目录 ' + location + ' 缺少或包含无效字段：' + invalid.join('、') + '。'); }
            if (!(model.model_messages && typeof model.model_messages.instructions_template === 'string') && typeof model.base_instructions !== 'string') { fail('模型目录 ' + location + ' 缺少 model_messages.instructions_template 或 base_instructions。'); }
            model.supported_reasoning_levels.forEach(function (level, index) {
                if (!level || typeof level !== 'object' || typeof level.effort !== 'string' || !level.effort) { fail('模型目录 ' + location + '.supported_reasoning_levels[' + index + '] 缺少有效的 effort。'); }
            });
        });
        return catalog;
    }
    function merge(existing, app, domestic, managed, info) {
        validateCatalog(app); validateCatalog(domestic);
        if (existing) { validateCatalog(existing); }
        if (!Array.isArray(managed) || managed.some(function (x) { return typeof x !== 'string' || !x; })) { fail('已管理模型记录格式无效。'); }
        if (managed.some(function (x) { return /^gpt-/i.test(x); })) { fail('管理记录包含 GPT，停止导入以保留 GPT。'); }
        var byId = dict(), order = [], owned = dict(), result, levels = [], gotGPT = false;
        managed.concat(domestic.models.map(function (x) { return x.slug; })).forEach(function (id) { owned[id] = true; });
        function add(models) { models.forEach(function (model) { if (!own(byId, model.slug) && !own(owned, model.slug)) { byId[model.slug] = model; order.push(model.slug); } }); }
        add(existing ? existing.models : []); add(app.models);
        domestic.models.forEach(function (model) { byId[model.slug] = model; order.push(model.slug); });
        result = JSON.parse(JSON.stringify(existing || app)); result.models = order.map(function (id) { return byId[id]; });
        result.models.forEach(function (model) { if (/^gpt-/i.test(model.slug)) { gotGPT = true; } });
        if (!gotGPT) { fail('未取得 GPT 完整目录，停止导入。'); }
        if (own(info.values, 'desktop.enabled-reasoning-efforts')) { levels = stringArray(info.values['desktop.enabled-reasoning-efforts']); }
        else { result.models.forEach(function (m) { if (/^gpt-/i.test(m.slug)) { m.supported_reasoning_levels.forEach(function (r) { levels.push(r.effort); }); } }); }
        domestic.models.forEach(function (m) { m.supported_reasoning_levels.forEach(function (r) { levels.push(r.effort); }); });
        var unique = dict(); levels.forEach(function (level) { if (effortOrder.indexOf(level) < 0) { fail('桌面档位存在未知值，停止导入。'); } unique[level] = true; });
        return {catalog: validateCatalog(result), efforts: effortOrder.filter(function (level) { return own(unique, level); }), managed: domestic.models.map(function (m) { return m.slug; })};
    }
    function transaction(io, files, backend, home, original, readSet) {
        var stamp = io.stamp(), stage = io.join(home, '.myautoapi-models-' + stamp), snapshots = [], committed = [], written = [];
        function checkInputs() {
            (readSet || []).forEach(function (input) {
                atStep('核对合并输入是否被修改', input.path, function () {
                    var now = io.exists(input.path) ? io.read(input.path) : null;
                    if (now !== input.text) { fail('文件在读取后被其他程序修改，已停止导入以保留新内容。'); }
                });
            });
        }
        checkInputs();
        files.forEach(function (file) {
            snapshots.push(atStep('检查并读取写入前的原文件', file.path, function () {
                if (io.exists(file.path) && !io.isFile(file.path)) { fail('目标文件不是普通文件，停止导入。'); }
                return io.exists(file.path) ? io.read(file.path) : null;
            }));
        });
        if (snapshots[2] !== original) { fail('配置在导入过程中被修改，请重新运行。'); }
        atStep('创建私有暂存目录', stage, function () { io.mkdir(stage); });
        try {
            files.forEach(function (file, index) { if (file.keep) { return; } var target = io.join(stage, 'file-' + index); atStep('写入暂存文件', target, function () { io.write(target, file.text); }); });
            var stagedCatalog = io.join(stage, 'file-0'), info = backend ? configInfo(files[2].text || '') : {values: dict()};
            var levels = own(info.values, 'desktop.enabled-reasoning-efforts') ? stringArray(info.values['desktop.enabled-reasoning-efforts']) : [];
            var expected = JSON.stringify(parseJSON(files[0].text).models.map(function (m) { return m.slug; }).sort());
            function verifyCatalog(target, step) {
                if (!backend) { return atStep(step, target, function () { return JSON.stringify(canonical(validateCatalog(parseJSON(io.read(target))).models)); }); }
                atStep('写入校验配置', io.join(stage, 'config.toml'), function () { io.write(io.join(stage, 'config.toml'), 'model_catalog_json = ' + JSON.stringify(target) + '\n[desktop]\nenabled-reasoning-efforts = ' + JSON.stringify(levels) + '\n'); });
                var validated = atStep(step, backend + ' debug models', function () { return validateCatalog(parseJSON(io.exportAt(backend, ['debug', 'models'], stage))); });
                if (JSON.stringify(validated.models.map(function (m) { return m.slug; }).sort()) !== expected) { fail('App 未读取新模型目录，停止导入。'); }
                return JSON.stringify(canonical(validated.models));
            }
            var normalized = verifyCatalog(stagedCatalog, '校验暂存模型目录');
            checkInputs();
            files.forEach(function (file, index) {
                atStep('核对导入期间的文件变化', file.path, function () {
                    var now = io.exists(file.path) ? io.read(file.path) : null;
                    if (now !== snapshots[index]) { fail('配置在导入过程中被修改，请重新运行。'); }
                });
            });
            files.forEach(function (file, index) { if (!file.keep && snapshots[index] !== null) { atStep('备份原文件', file.path, function () { io.backup(file.path, file.path + '.bak.' + stamp); }); } });
            files.forEach(function (file, index) { if (file.keep) { return; } committed.push(index); atStep('提交导入文件', file.path, function () { io.write(file.path, file.text); }); written[index] = true; });
            /* Validate the committed catalog without loading or changing the user's provider/auth. */
            if (verifyCatalog(files[0].path, '校验已写入的模型目录') !== normalized) { fail('已写入的模型元数据与本次导入不一致，停止导入。'); }
            files.forEach(function (file) {
                atStep('核对写入后的文件', file.path, function () {
                    var now = io.exists(file.path) ? io.read(file.path) : null;
                    if (now !== file.text) { fail('写入后的文件与本次导入不一致，停止导入。'); }
                });
            });
        } catch (error) {
            var conflict = null;
            if (committed.length) {
                files.some(function (file, index) {
                    try {
                        var now = io.exists(file.path) ? io.read(file.path) : null;
                        if (now === file.text || (!written[index] && now === snapshots[index])) { return false; }
                    } catch (ignored) { /* An unreadable or replaced path is also an external conflict. */ }
                    conflict = file.path; return true;
                });
            }
            if (conflict) { atStep('检查恢复是否会覆盖外部修改', conflict, function () { fail('导入过程中出现外部修改，已保留当前内容和备份，软件未关闭。'); }); }
            var restoreFailed = false;
            committed.reverse().forEach(function (index) { try { if (snapshots[index] === null) { io.remove(files[index].path); } else { io.write(files[index].path, snapshots[index]); } } catch (ignored) { restoreFailed = true; } });
            if (restoreFailed) { fail('写入失败且恢复未完成，请使用同目录的 .bak 备份恢复；软件未关闭。'); }
            throw error;
        } finally {
            try { io.remove(stage); } catch (ignored) { /* A leftover private stage does not invalidate a successful import. */ }
        }
    }
    function parseJSON(text) {
        try { return JSON.parse(text.replace(/^\uFEFF/, '')); }
        catch (error) {
            var message = !text.trim() ? '内容为空，不是有效的 JSON。' : '内容不是有效的 JSON。';
            var position = /(?:position|offset)\s+(\d+)/i.exec(error.message || '');
            var line = /line\s+(\d+)\s+column\s+(\d+)/i.exec(error.message || '');
            if (position) { message += ' 字符位置：' + position[1] + '。'; }
            if (line) { message += ' 第 ' + line[1] + ' 行，第 ' + line[2] + ' 列。'; }
            error.safeMessage = message; error.code = 'EJSON_PARSE'; throw error;
        }
    }
    function importModels(io, home, executable, domestic) {
        var configPath = io.join(home, 'config.toml'), catalogPath = io.join(home, 'myautoapi-models.json'), managedPath = io.join(home, 'myautoapi-models.managed.json');
        atStep('创建 Codex 配置目录', home, function () { if (!io.exists(home)) { io.mkdir(home, true); } });
        var original = atStep('读取 Codex 配置', configPath, function () {
            return io.exists(configPath) ? io.read(configPath) : null;
        });
        var info = atStep('解析 Codex 配置', configPath, function () { return configInfo(original || ''); }), existing = null, source, managed = [];
        var readSet = [{path: configPath, text: original}];
        function readInput(file) {
            for (var index = 0; index < readSet.length; index++) { if (readSet[index].path === file) { return readSet[index].text; } }
            var text = io.exists(file) ? io.read(file) : null;
            readSet.push({path: file, text: text}); return text;
        }
        atStep('读取目标模型目录快照', catalogPath, function () { readInput(catalogPath); });
        if (own(info.values, 'model_catalog_json')) {
            source = io.resolve(home, stringValue(info.values.model_catalog_json));
            existing = atStep('读取并解析原模型目录', source, function () { var text = readInput(source); return text === null ? null : validateCatalog(parseJSON(text)); });
        }
        managed = atStep('读取并解析已管理模型记录', managedPath, function () { var text = readInput(managedPath); return text === null ? [] : parseJSON(text); });
        var app = atStep('导出并解析 Codex 内置模型目录', executable + ' debug models --bundled', function () { return validateCatalog(parseJSON(io.exportCatalog(executable, true))); });
        if (!existing) {
            try { existing = validateCatalog(parseJSON(io.exportCatalog(executable, false))); }
            catch (error) {
                if (error.configSyntax) { atStep('解析已有 TOML 配置', configPath, function () { throw error; }); }
                /* Provider/auth failures do not prevent a local catalog import. */
            }
        }
        var merged = atStep('校验并合并模型元数据', source || executable + ' debug models --bundled', function () { return merge(existing, app, domestic, managed, info); });
        var config = atStep('合并模型目录指向和桌面档位', configPath, function () { return patchConfig(original || '', info, catalogPath, merged.efforts); });
        var files = [{path: catalogPath, text: JSON.stringify(merged.catalog, null, 2) + '\n'}, {path: managedPath, text: JSON.stringify(merged.managed, null, 2) + '\n'}, {path: configPath, text: config}];
        /* Host stages all files, validates with the selected App backend, then commits with rollback. */
        atStep('写入并校验导入结果', catalogPath, function () { io.transaction(files, executable, catalogPath, original, readSet); });
        return merged.managed.length;
    }
    function uninstallModels(io, home) {
        var configPath = io.join(home, 'config.toml'), catalogPath = io.join(home, 'myautoapi-models.json'), managedPath = io.join(home, 'myautoapi-models.managed.json');
        var managedText = atStep('读取已管理模型记录', managedPath, function () { return io.exists(managedPath) ? io.read(managedPath) : null; });
        var managed = atStep('校验已管理模型记录', managedPath, function () {
            var ids = managedText === null ? [] : parseJSON(managedText);
            if (!Array.isArray(ids) || ids.some(function (id) { return typeof id !== 'string' || !id || /^gpt-/i.test(id); })) { fail('已管理模型记录无效或包含 GPT，未移除任何模型。'); }
            return ids;
        });
        if (!managed.length) { return {changed: false, active: false, count: 0}; }
        var original = atStep('读取 Codex 配置', configPath, function () { return io.exists(configPath) ? io.read(configPath) : null; });
        var info = atStep('解析 Codex 配置', configPath, function () { return configInfo(original || '', true); });
        var catalogText = atStep('读取本工具的模型目录', catalogPath, function () { return io.read(catalogPath); });
        var catalog = atStep('校验本工具的模型目录', catalogPath, function () { return validateCatalog(parseJSON(catalogText)); });
        var owned = dict(), before = catalog.models.length;
        managed.forEach(function (id) { owned[id] = true; });
        catalog.models = catalog.models.filter(function (model) { return !own(owned, model.slug); });
        atStep('校验移除后的模型目录', catalogPath, function () { validateCatalog(catalog); });
        var count = before - catalog.models.length;
        var active = atStep('核对当前模型目录指向', configPath, function () {
            return !info.profileCatalog && own(info.values, 'model_catalog_json') && io.resolve(home, stringValue(info.values.model_catalog_json)) === catalogPath;
        });
        var files = [{path: catalogPath, text: JSON.stringify(catalog, null, 2) + '\n'}, {path: managedPath, text: '[]\n'}, {path: configPath, text: original, keep: true}];
        var readSet = [{path: configPath, text: original}, {path: catalogPath, text: catalogText}, {path: managedPath, text: managedText}];
        atStep('移除并校验国产模型', catalogPath, function () { io.transaction(files, '', catalogPath, original, readSet); });
        return {changed: true, active: active && count > 0, count: count};
    }
    function uninstallMessage(result) {
        if (result.active) { return '已移除本工具导入的国产模型，原 GPT 和其他模型已保留。'; }
        if (result.changed) { return '已清理原导入目录；软件未关闭。'; }
        return '没有需要移除的国产模型，软件未关闭。';
    }
    return {statements: statements, configInfo: configInfo, stringValue: stringValue, stringArray: stringArray, patchConfig: patchConfig, validateCatalog: validateCatalog, merge: merge, transaction: transaction, importModels: importModels, uninstallModels: uninstallModels, uninstallMessage: uninstallMessage, formatError: formatError, backendFailure: backendFailure};
}());
if (typeof module !== 'undefined') { module.exports = DomesticModels; }

/* macOS built-in JavaScript for Automation; no Node installation required. */
if (typeof ObjC !== 'undefined') { ObjC.import('Foundation'); }
function run(argv) {
    'use strict';
    var fm = $.NSFileManager.defaultManager, utf8 = $.NSUTF8StringEncoding;
    function nativeError(message, path, native) {
        var error = new Error(message); error.safeMessage = message;
        if (path) { error.importTarget = path; }
        try { if (native && native[0]) { var code = Number(native[0].code); if (isFinite(code)) { error.systemCode = code; } } } catch (ignored) {}
        throw error;
    }
    function attrs(path) { var error = Ref(); return fm.attributesOfItemAtPathError($(path), error); }
    function exists(path) { return !attrs(path).isNil(); }
    function isFile(path) { var value = attrs(path); return !value.isNil() && ObjC.unwrap(value.objectForKey($.NSFileType)) === ObjC.unwrap($.NSFileTypeRegular); }
    function read(path) {
        if (!isFile(path)) { nativeError('文件不存在或不是普通文件。', path); }
        var error = Ref(), value = $.NSString.stringWithContentsOfFileEncodingError($(path), utf8, error);
        if (value.isNil()) { nativeError('无法按 UTF-8 读取文件。', path, error); }
        return ObjC.unwrap(value);
    }
    function permissions(path) {
        var error = Ref();
        if (!fm.setAttributesOfItemAtPathError($.NSDictionary.dictionaryWithObjectForKey($(384), $.NSFilePosixPermissions), $(path), error)) { nativeError('无法设置文件权限。', path, error); }
    }
    function write(path, text) {
        var error = Ref();
        if (!$(text).writeToFileAtomicallyEncodingError($(path), true, utf8, error)) { nativeError('无法写入文件。', path, error); }
        permissions(path);
    }
    function remove(path) {
        if (exists(path) && !fm.removeItemAtPathError($(path), Ref())) { nativeError('无法清理暂存文件。'); }
    }
    function execute(executable, args, env) {
        var task = $.NSTask.alloc.init, outputPath = join(env.CODEX_HOME, '.myautoapi-export-' + ObjC.unwrap($.NSUUID.UUID.UUIDString));
        var errorPath = outputPath + '.stderr', output = null, errors = null;
        try {
            write(outputPath, ''); write(errorPath, '');
            output = $.NSFileHandle.fileHandleForWritingAtPath($(outputPath));
            errors = $.NSFileHandle.fileHandleForWritingAtPath($(errorPath));
            if (output.isNil() || errors.isNil()) { nativeError('无法打开后端校验输出文件。', outputPath); }
            task.executableURL = $.NSURL.fileURLWithPath($(executable)); task.arguments = $(args);
            task.standardOutput = output; task.standardError = errors;
            task.currentDirectoryURL = $.NSURL.fileURLWithPath($(env.CODEX_HOME));
            var environment = $.NSProcessInfo.processInfo.environment.mutableCopy;
            environment.setObjectForKey($(env.CODEX_HOME), $('CODEX_HOME'));
            var searchPath = ObjC.unwrap(environment.objectForKey($('PATH'))) || '';
            environment.setObjectForKey($(searchPath.split(':').map(function (entry) { return resolve(ObjC.unwrap(fm.currentDirectoryPath), entry || '.'); }).join(':')), $('PATH'));
            task.environment = environment;
            var launchError = Ref();
            if (!task.launchAndReturnError(launchError)) { nativeError('无法启动 Codex 后端。', executable, launchError); }
            var deadline = Date.now() + 30000;
            while (task.running && Date.now() < deadline) { $.NSThread.sleepForTimeInterval(0.05); }
            if (task.running) {
                task.terminate;
                $.NSThread.sleepForTimeInterval(0.2);
                if (task.running) {
                    var killer = $.NSTask.alloc.init;
                    killer.executableURL = $.NSURL.fileURLWithPath($('/bin/kill'));
                    killer.arguments = $(['-KILL', String(task.processIdentifier)]);
                    if (!killer.launchAndReturnError(Ref())) { nativeError('无法结束超时的校验后端。'); }
                    killer.waitUntilExit;
                }
                task.waitUntilExit;
                throw DomesticModels.backendFailure('', {code: 'ETIMEDOUT'});
            }
            task.waitUntilExit;
            if (task.terminationStatus !== 0) { throw DomesticModels.backendFailure(read(errorPath), {status: Number(task.terminationStatus)}); }
            return read(outputPath);
        } finally {
            if (output && !output.isNil()) { output.closeFile; }
            if (errors && !errors.isNil()) { errors.closeFile; }
            try { remove(outputPath); remove(errorPath); } catch (ignored) {}
        }
    }
    var homeValue = $(argv[0]).stringByExpandingTildeInPath.stringByStandardizingPath, executable = argv[1];
    if (!homeValue.isAbsolutePath) { homeValue = fm.currentDirectoryPath.stringByAppendingPathComponent(homeValue).stringByStandardizingPath; }
    var home = ObjC.unwrap(homeValue), action = argv[2] || 'import';
    function resolve(base, path) {
        var value = $(path).stringByExpandingTildeInPath;
        if (!value.isAbsolutePath) { value = $(base).stringByAppendingPathComponent(value); }
        return ObjC.unwrap(value.stringByStandardizingPath);
    }
    function join(a, b) { return ObjC.unwrap($(a).stringByAppendingPathComponent($(b))); }
    var io = {
        join: join, resolve: resolve, read: read, write: write, isFile: isFile, remove: remove,
        exists: exists,
        stamp: function () { return new Date().toISOString().replace(/[^0-9]/g, '') + '.' + ObjC.unwrap($.NSUUID.UUID.UUIDString); },
        mkdir: function (path, recursive) {
            if (!fm.createDirectoryAtPathWithIntermediateDirectoriesAttributesError($(path), !!recursive, $.NSDictionary.dictionaryWithObjectForKey($(448), $.NSFilePosixPermissions), Ref())) { nativeError('无法创建配置或暂存目录。'); }
        },
        backup: function (from, to) {
            if (!fm.copyItemAtPathToPathError($(from), $(to), Ref())) { nativeError('备份失败，停止导入。'); }
            permissions(to);
        },
        exportAt: function (backend, args, target) { return execute(backend, args, {CODEX_HOME: target}); },
        exportCatalog: function (backend, bundled) { return execute(backend, bundled ? ['debug', 'models', '--bundled'] : ['debug', 'models'], {CODEX_HOME: home}); },
        transaction: function (files, backend, catalogPath, original, readSet) {
            DomesticModels.transaction(io, files, backend, home, original, readSet);
        }
    };
    try {
        if (action === 'uninstall') { return DomesticModels.uninstallMessage(DomesticModels.uninstallModels(io, home)); }
        var count = DomesticModels.importModels(io, home, executable, DOMESTIC_CATALOG);
        return '已导入 ' + count + ' 个国产模型，原 GPT 和其他模型已保留。';
    } catch (error) {
        /* Never echo external process stderr, original TOML, or JSON (may contain credentials). */
        throw new Error(DomesticModels.formatError(error, action) + '\n软件未关闭。');
    }
}

/* Linux adapter: existing Node.js only; the model/config core is shared with macOS. */
function createLinuxIO(home, core) {
    'use strict';
    var fs = require('fs'), path = require('path'), os = require('os');
    var crypto = require('crypto'), child = require('child_process');
    function failure(message) { var error = new Error(message); error.safeMessage = message; return error; }
    function exists(file) {
        try { fs.lstatSync(file); return true; }
        catch (error) { if (error.code === 'ENOENT') { return false; } throw error; }
    }
    function isFile(file) { return fs.lstatSync(file).isFile(); }
    function read(file) {
        if (!isFile(file)) { throw failure('文件不存在或不是普通文件。'); }
        return new (require('util').TextDecoder)('utf-8', {fatal: true, ignoreBOM: true}).decode(fs.readFileSync(file));
    }
    function write(file, text) {
        if (exists(file) && !isFile(file)) { throw failure('目标文件不是普通文件，停止导入。'); }
        var temp = path.join(path.dirname(file), '.myautoapi-write-' + crypto.randomUUID());
        try {
            fs.writeFileSync(temp, text, {encoding: 'utf8', flag: 'wx', mode: 384});
            fs.renameSync(temp, file);
        } finally { if (exists(temp)) { fs.unlinkSync(temp); } }
    }
    function exportAt(executable, args, target) {
        try {
            return child.execFileSync(executable, args, {
                cwd: target, env: Object.assign({}, process.env, {CODEX_HOME: target, PATH: (process.env.PATH || '').split(path.delimiter).map(function (entry) { return path.resolve(entry || '.'); }).join(path.delimiter)}),
                timeout: 30000, killSignal: 'SIGKILL', maxBuffer: 64 * 1024 * 1024,
                encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe']
            });
        } catch (error) {
            throw core.backendFailure(error.stderr, error);
        }
    }
    var io = {
        join: path.join, exists: exists, isFile: isFile, read: read, write: write,
        resolve: function (base, file) {
            if (file === '~' || file.slice(0, 2) === '~/') { file = path.join(os.homedir(), file.slice(1)); }
            return path.resolve(base, file);
        },
        remove: function (file) { fs.rmSync(file, {force: true, recursive: true}); },
        mkdir: function (directory, recursive) { fs.mkdirSync(directory, {mode: 448, recursive: !!recursive}); },
        stamp: function () { return new Date().toISOString().replace(/[^0-9]/g, '') + '.' + crypto.randomUUID(); },
        backup: function (from, to) { fs.copyFileSync(from, to, fs.constants.COPYFILE_EXCL); fs.chmodSync(to, 384); },
        exportAt: exportAt,
        exportCatalog: function (backend, bundled) { return exportAt(backend, bundled ? ['debug', 'models', '--bundled'] : ['debug', 'models'], home); },
        transaction: function (files, backend, unused, original, readSet) { core.transaction(io, files, backend, home, original, readSet); }
    };
    return io;
}

function runLinuxInstaller(argv, core, domestic) {
    'use strict';
    var action = argv[2] || 'import';
    try {
        var path = require('path'), homeArg = argv[0];
        if (homeArg === '~' || homeArg.slice(0, 2) === '~/') { homeArg = path.join(require('os').homedir(), homeArg.slice(1)); }
        var home = path.resolve(homeArg), backend = argv[1] ? path.resolve(argv[1]) : '';
        if (action === 'uninstall') {
            process.stdout.write(core.uninstallMessage(core.uninstallModels(createLinuxIO(home, core), home)) + '\n');
            return;
        }
        var count = core.importModels(createLinuxIO(home, core), home, backend, domestic);
        process.stdout.write('已导入 ' + count + ' 个国产模型，原 GPT 和其他模型已保留。\n');
    } catch (error) {
        process.stderr.write(core.formatError(error, action) + '\n软件未关闭。\n');
        process.exitCode = 1;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {createLinuxIO: createLinuxIO, runLinuxInstaller: runLinuxInstaller};
    if (typeof DOMESTIC_CATALOG !== 'undefined') { runLinuxInstaller(process.argv.slice(2), DomesticModels, DOMESTIC_CATALOG); }
}

MYAUTOAPI_JAVASCRIPT
)"
then
  if [ "$model_action" = uninstall ]; then
    case "$import_message" in
      '没有需要移除的国产模型，软件未关闭。'|'已清理原导入目录；软件未关闭。') say "$import_message"; exit 0 ;;
      '已移除本工具导入的国产模型，原 GPT 和其他模型已保留。') ;;
      *) say '卸载未确认：运行器没有返回完整卸载结果。软件未关闭。' >&2; exit 1 ;;
    esac
  elif [ "$import_message" != '已导入 11 个国产模型，原 GPT 和其他模型已保留。' ]; then
    say '导入未确认：运行器没有返回完整导入结果。软件未关闭。' >&2
    exit 1
  fi
  say "$import_message"
  stop_running_clients
  report_running_clients
  if [ "$model_action" = uninstall ]; then
    say '重新打开 Codex，新建任务后选择模型。'
  else
    say '重新打开 Codex，新建任务后选择模型；API Key 分组由你在中转站切换。'
  fi
else
  if [ "$model_action" = uninstall ]; then say '卸载未完成，软件未关闭。' >&2; else say '导入未完成，软件未关闭。' >&2; fi
  exit 1
fi
