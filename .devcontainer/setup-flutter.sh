#!/usr/bin/env bash
# 备用脚本：镜像里的 Flutter 不可用时再手动装。
# 正常 Codespace 应直接使用 Dockerfile 打好的 /opt/flutter。
set -euo pipefail

FLUTTER_ROOT="/opt/flutter"

if command -v flutter >/dev/null 2>&1; then
  echo "[setup-flutter] flutter already on PATH:"
  flutter --version
  exit 0
fi

echo "[setup-flutter] install system deps..."
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  ca-certificates curl git unzip xz-utils zip

if [ ! -d "${FLUTTER_ROOT}/bin" ]; then
  echo "[setup-flutter] clone flutter stable (shallow)..."
  sudo git clone https://github.com/flutter/flutter.git -b stable --depth 1 "${FLUTTER_ROOT}"
fi

sudo chown -R "$(whoami)":"$(whoami)" "${FLUTTER_ROOT}"
sudo git config --system --add safe.directory "${FLUTTER_ROOT}" || true
export PATH="${FLUTTER_ROOT}/bin:${PATH}"

flutter config --no-analytics
flutter --version
flutter precache --universal || true

echo "[setup-flutter] done. Next:"
echo "  cd mobile && flutter pub get && flutter analyze && flutter test"
