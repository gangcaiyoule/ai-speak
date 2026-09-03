#!/usr/bin/env bash
# Codespaces 上安装 Flutter（只为 flutter analyze / flutter test，不装 Android SDK）。
set -euo pipefail

FLUTTER_ROOT="/opt/flutter"

echo "[setup-flutter] install system deps..."
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  curl git unzip xz-utils zip \
  clang cmake ninja-build pkg-config \
  libgtk-3-dev liblzma-dev

if [ ! -d "${FLUTTER_ROOT}/bin" ]; then
  echo "[setup-flutter] clone flutter stable (shallow)..."
  sudo git clone https://github.com/flutter/flutter.git -b stable --depth 1 "${FLUTTER_ROOT}"
  sudo chown -R "$(whoami)":"$(whoami)" "${FLUTTER_ROOT}"
else
  echo "[setup-flutter] flutter already exists, skip clone."
  sudo chown -R "$(whoami)":"$(whoami)" "${FLUTTER_ROOT}"
fi

export PATH="${FLUTTER_ROOT}/bin:${PATH}"

echo "[setup-flutter] disable analytics..."
flutter config --no-analytics

echo "[setup-flutter] flutter version:"
flutter --version

echo "[setup-flutter] precache desktop/universal artifacts (skip android/ios)..."
# test/analyze 只需要 Dart SDK；precache 失败不阻塞建环境。
flutter precache --universal || true

echo "[setup-flutter] done. Run in Codespace:"
echo "  cd mobile && flutter pub get && flutter analyze && flutter test"
