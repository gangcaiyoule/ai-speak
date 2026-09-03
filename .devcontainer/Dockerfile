FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

ENV FLUTTER_HOME=/opt/flutter
ENV PATH="${FLUTTER_HOME}/bin:${PATH}"
ENV FLUTTER_ROOT=/opt/flutter

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        unzip \
        xz-utils \
        zip \
    && rm -rf /var/lib/apt/lists/*

# 镜像构建阶段安装 SDK，避免 Codespace 创建时在 onCreate 里 git clone。
# 只装 analyze / test 需要的 universal 产物，不装 Android / iOS SDK。
RUN git clone --depth 1 -b stable https://github.com/flutter/flutter.git "${FLUTTER_HOME}" \
    && git config --system --add safe.directory "${FLUTTER_HOME}" \
    && "${FLUTTER_HOME}/bin/flutter" config --no-analytics \
    && "${FLUTTER_HOME}/bin/flutter" precache --universal \
    && chown -R vscode:vscode "${FLUTTER_HOME}"
