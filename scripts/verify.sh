#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
(cd "$ROOT/server" && go test -race -count=1 ./... && go vet ./...)
(cd "$ROOT/web" && npm ci && npm test -- --run && npm run build)
"$ROOT/scripts/smoke.sh"
echo 'Go、React 和真实服务重启闭环验证完成。Unity 编译/真机 AR 不包含在此命令中。'
