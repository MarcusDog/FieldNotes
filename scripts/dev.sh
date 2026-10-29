#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if ! command -v go >/dev/null || ! command -v npm >/dev/null; then
  echo '需要 Go 1.26+ 和 Node.js 22.12+/24 LTS（含 npm）。' >&2
  exit 1
fi
mkdir -p "$ROOT/data"
PORT="${PORT:-8090}"
if command -v lsof >/dev/null && lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "端口 $PORT 已被占用。请设置 PORT=其他空闲端口，不会停止已有程序。" >&2
  exit 1
fi
if [ ! -d "$ROOT/web/node_modules" ]; then (cd "$ROOT/web" && npm ci); fi
TASK_TMP="$(mktemp -d)"
API_PID=''
WEB_PID=''
cleanup() {
  [ -z "$WEB_PID" ] || kill "$WEB_PID" 2>/dev/null || true
  [ -z "$API_PID" ] || kill "$API_PID" 2>/dev/null || true
  wait 2>/dev/null || true
  rm -f "$TASK_TMP/fieldnotes"
  rmdir "$TASK_TMP" 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT TERM
(cd "$ROOT/server" && go build -o "$TASK_TMP/fieldnotes" ./cmd/fieldnotes)
DATA_FILE="${DATA_FILE:-$ROOT/data/issues.json}" ADDR=":$PORT" "$TASK_TMP/fieldnotes" &
API_PID=$!
(cd "$ROOT/web" && export API_PROXY_TARGET="${API_PROXY_TARGET:-http://127.0.0.1:$PORT}" && exec node node_modules/vite/bin/vite.js --strictPort) &
WEB_PID=$!
echo "管理端：http://localhost:5173；手机后端：http://电脑局域网IP:$PORT"
echo '数据文件：data/issues.json（除非用 DATA_FILE 覆盖）。Ctrl+C 停止两个服务。'
while kill -0 "$API_PID" 2>/dev/null && kill -0 "$WEB_PID" 2>/dev/null; do sleep 1; done
echo '某个服务退出；请查看上方错误（例如端口占用）。' >&2
exit 1
