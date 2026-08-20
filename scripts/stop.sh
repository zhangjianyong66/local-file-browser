#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
BINARY="${FILE_BROWSER_BINARY:-$PROJECT_DIR/local-file-browser}"
PID_FILE="${FILE_BROWSER_PID_FILE:-$PROJECT_DIR/local-file-browser.pid}"

if [[ ! -f "$PID_FILE" ]]; then
  printf '服务未运行（PID 文件不存在）。\n'
  exit 0
fi
pid="$(<"$PID_FILE")"
if ! [[ "$pid" =~ ^[0-9]+$ ]]; then
  printf 'PID 文件无效: %s\n' "$PID_FILE" >&2
  exit 1
fi
if [[ -e "/proc/$pid/exe" ]]; then
  actual="$(readlink -f -- "/proc/$pid/exe" 2>/dev/null || true)"
  expected="$(readlink -f -- "$BINARY" 2>/dev/null || true)"
  if [[ -z "$actual" || -z "$expected" || "$actual" != "$expected" ]]; then
    printf 'PID=%s 不是目标程序，未执行停止操作。\n' "$pid" >&2
    exit 1
  fi
fi
if ! kill -0 "$pid" 2>/dev/null; then
  rm -f -- "$PID_FILE"
  printf '服务已停止。\n'
  exit 0
fi
kill -TERM "$pid"
for _ in {1..50}; do
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f -- "$PID_FILE"
    printf '服务已优雅停止。\n'
    exit 0
  fi
  sleep 0.1
done
printf '服务未在 5 秒内退出，请检查进程 PID=%s。\n' "$pid" >&2
exit 1
