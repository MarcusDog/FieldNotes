#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [ -n "$(git status --porcelain)" ]; then
  echo '请先检查并提交当前源码和文档，避免压缩包遗漏未提交内容。' >&2
  exit 1
fi
mkdir -p "$ROOT/artifacts"
TASK_PACKAGE="$(mktemp -d)"
trap 'rm -rf "$TASK_PACKAGE"' EXIT
mkdir -p "$TASK_PACKAGE/FieldNotes-delivery/source" "$TASK_PACKAGE/FieldNotes-delivery/videos"
git archive HEAD | tar -x -C "$TASK_PACKAGE/FieldNotes-delivery/source"
git bundle create "$TASK_PACKAGE/FieldNotes-delivery/history.bundle" --all
if [ -d "$ROOT/delivery/videos" ]; then
  find "$ROOT/delivery/videos" -maxdepth 1 -type f \( -name '*.mp4' -o -name '*.mov' \) -exec cp {} "$TASK_PACKAGE/FieldNotes-delivery/videos/" \;
fi
(cd "$TASK_PACKAGE" && zip -qr "$ROOT/artifacts/FieldNotes-delivery.zip" FieldNotes-delivery)
echo "已生成 $ROOT/artifacts/FieldNotes-delivery.zip"
if ! find "$TASK_PACKAGE/FieldNotes-delivery/videos" -type f | read -r _; then
  echo '注意：压缩包尚不包含真实演示视频；完整交付还需录制与上传分享链接。'
fi
