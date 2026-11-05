#!/bin/bash
# Terminal side of the Mac quick-connect assistant (Contents/Resources).
#   run.command                double click: install.sh --from-app reads the clipboard
#   run.command --link FIFO    import link: the launcher writes the key into FIFO,
#                              install.sh gets it as MYAUTOAPI_KEY
# Only one window runs install.sh at a time (lock: $state_dir/run.lock/pid).
set -u
resource_dir="$(cd "$(dirname "$0")" && pwd)"
state_dir="$HOME/Library/Caches/com.myautoapi.codex-config"
lock="$state_dir/run.lock"

finish() {
  printf '\n'
  if [ "$1" -ne 0 ]; then
    printf '操作未完成，退出码：%s。请保留上面的错误提示。\n' "$1"
  fi
  printf '按回车结束，随后可关闭本窗口。'
  IFS= read -r ignored </dev/tty 2>/dev/null || true
  exit "$1"
}

lock_owner_alive() {
  [ -n "$1" ] && kill -0 "$1" 2>/dev/null &&
    ps -o command= -p "$1" 2>/dev/null | grep -q 'run\.command'
}

acquire_lock() {
  local attempt owner
  mkdir -p "$state_dir" && chmod 700 "$state_dir" || return 1
  for attempt in 1 2 3; do
    if mkdir "$lock" 2>/dev/null; then
      printf '%s\n' "$$" > "$lock/pid"
      return 0
    fi
    owner="$(cat "$lock/pid" 2>/dev/null)"
    if [ -z "$owner" ]; then
      # The owner may be between mkdir and writing its pid.
      sleep 0.3
      owner="$(cat "$lock/pid" 2>/dev/null)"
    fi
    if lock_owner_alive "$owner"; then
      return 1
    fi
    # Left behind by a closed window. Renaming first means only one of two
    # waiting windows clears it.
    if mv "$lock" "$lock.stale.$$" 2>/dev/null; then
      rm -rf "$lock.stale.$$"
    fi
  done
  return 1
}

release_lock() {
  if [ "$(cat "$lock/pid" 2>/dev/null)" = "$$" ]; then
    rm -rf "$lock"
  fi
}

mode=app
fifo=""
if [ "${1:-}" = "--link" ]; then
  mode=link
  fifo="${2:-}"
fi

if ! acquire_lock; then
  # Removing the FIFO tells the launcher to stop waiting.
  if [ "$mode" = link ] && [ -p "$fifo" ]; then
    rm -f "$fifo"
  fi
  printf '另一个快速接入助手窗口正在配置 Codex。请先在那个终端窗口完成或关闭它，再重新导入。\n'
  finish 1
fi
trap release_lock EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

if [ "$mode" = link ]; then
  key=""
  shown=""
  # Opening read-write never blocks, so a launcher that has quit only costs
  # the read timeout.
  if [ -p "$fifo" ] && exec 4<>"$fifo"; then
    IFS= read -r -t 30 -u 4 key || key=""
    IFS= read -r -t 2 -u 4 shown || shown=""
    exec 4<&-
  fi
  if [ -p "$fifo" ]; then
    rm -f "$fifo"
  fi
  case "$key" in
    sk-*) ;;
    *)
      printf '没有收到网页传来的密钥，请回网站重新点「一键导入」。\n'
      finish 1
      ;;
  esac
  printf '已收到网页导入的密钥：%s\n\n' "$shown"
  MYAUTOAPI_KEY="$key" /bin/bash "$resource_dir/install.sh"
  result=$?
  key=""
else
  /bin/bash "$resource_dir/install.sh" --from-app
  result=$?
fi

release_lock
trap - EXIT
finish "$result"
