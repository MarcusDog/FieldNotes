#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EDITOR="${UNITY_EDITOR:-/Applications/Unity/Hub/Editor/6000.3.27f1/Unity.app/Contents/MacOS/Unity}"
if [ ! -x "$EDITOR" ]; then echo '未找到 Unity。安装 6000.3.27f1 和 iOS Build Support，或设置 UNITY_EDITOR 为编辑器可执行文件路径。' >&2; exit 2; fi
mkdir -p "$ROOT/artifacts"
export FIELDNOTES_IOS_OUTPUT="${FIELDNOTES_IOS_OUTPUT:-$ROOT/unity/Builds/iOS}"
case "${1:-test}" in
  setup) exec "$EDITOR" -batchmode -nographics -quit -projectPath "$ROOT/unity" -buildTarget iOS -executeMethod FieldNotes.Editor.ProjectSetup.Configure -logFile "$ROOT/artifacts/unity-setup.log" ;;
  test) exec "$EDITOR" -batchmode -nographics -projectPath "$ROOT/unity" -runTests -testPlatform EditMode -testResults "$ROOT/artifacts/unity-tests.xml" -logFile "$ROOT/artifacts/unity-tests.log" ;;
  ios)
    "$EDITOR" -batchmode -nographics -quit -projectPath "$ROOT/unity" -buildTarget iOS -executeMethod FieldNotes.Editor.ProjectSetup.Configure -logFile "$ROOT/artifacts/unity-setup.log"
    exec "$EDITOR" -batchmode -nographics -quit -projectPath "$ROOT/unity" -buildTarget iOS -executeMethod FieldNotes.Editor.ProjectSetup.ExportIOS -logFile "$ROOT/artifacts/unity-ios.log"
    ;;
  *) echo '用法：scripts/unity.sh setup|test|ios' >&2; exit 2 ;;
esac
