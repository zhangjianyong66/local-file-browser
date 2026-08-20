#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
BINARY="${FILE_BROWSER_BINARY:-$PROJECT_DIR/local-file-browser}"
CONFIG="${FILE_BROWSER_CONFIG:-$PROJECT_DIR/config.json}"
PID_FILE="${FILE_BROWSER_PID_FILE:-$PROJECT_DIR/local-file-browser.pid}"
LOG_FILE="${FILE_BROWSER_LOG_FILE:-$PROJECT_DIR/local-file-browser.log}"

if [[ ! -x "$BINARY" ]]; then
  printf '可执行文件不存在或不可执行: %s\n' "$BINARY" >&2
  exit 1
fi
if [[ ! -f "$CONFIG" ]]; then
  printf '配置文件不存在: %s\n请复制 config.example.json 为 config.json 并设置密码。\n' "$CONFIG" >&2
  exit 1
fi
if [[ -f "$PID_FILE" ]]; then
  pid="$(<"$PID_FILE")"
  if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
    printf '服务已运行，PID=%s\n' "$pid"
    exit 0
  fi
  rm -f -- "$PID_FILE"
fi

umask 077
nohup "$BINARY" -config "$CONFIG" >>"$LOG_FILE" 2>&1 &
pid=$!
printf '%s\n' "$pid" >"$PID_FILE"
printf '服务已启动，PID=%s，日志: %s\n' "$pid" "$LOG_FILE"
