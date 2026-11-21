FROM node:22-bookworm-slim

ARG USER_ID=1000
ARG GROUP_ID=1000
ARG GO_VERSION=1.25.1
ARG DART_VERSION=3.13.7
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    GOPATH=/home/agent/go \
    GOCACHE=/home/agent/.cache/go-build \
    PATH=/home/agent/.dart-sdk/bin:/usr/local/go/bin:/home/agent/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    build-essential \
    ca-certificates \
    curl \
    git \
    jq \
    less \
    nano \
    openssh-client \
    procps \
    python3 \
    ripgrep \
    sqlite3 \
    unzip \
    zip \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH:-amd64}.tar.gz" \
    | tar -C /usr/local -xz

RUN DART_ARCH=x64; \
    if [ "${TARGETARCH:-amd64}" = "arm64" ]; then DART_ARCH=arm64; fi; \
    mkdir -p /home/agent /tmp/dart-sdk \
    && curl -fsSL "https://storage.googleapis.com/dart-archive/channels/stable/release/${DART_VERSION}/sdk/dartsdk-linux-${DART_ARCH}-release.zip" \
    -o /tmp/dart-sdk/dart.zip \
    && unzip -q /tmp/dart-sdk/dart.zip -d /tmp/dart-sdk \
    && mv /tmp/dart-sdk/dart-sdk /home/agent/.dart-sdk \
    && rm -rf /tmp/dart-sdk

RUN npm install --global @anthropic-ai/claude-code

RUN set -eux; \
    if getent group "${GROUP_ID}" >/dev/null; then \
        AGENT_GROUP="$(getent group "${GROUP_ID}" | cut -d: -f1)"; \
    else \
        groupadd --gid "${GROUP_ID}" agent; \
        AGENT_GROUP=agent; \
    fi; \
    if getent passwd "${USER_ID}" >/dev/null; then \
        AGENT_USER="$(getent passwd "${USER_ID}" | cut -d: -f1)"; \
        usermod -d /home/agent -s /bin/bash -g "${AGENT_GROUP}" "${AGENT_USER}"; \
    else \
        useradd --create-home --uid "${USER_ID}" --gid "${AGENT_GROUP}" \
        --shell /bin/bash agent; \
    fi; \
    mkdir -p /home/agent/.claude /home/agent/.config /home/agent/go \
        /home/agent/.cache /claude-auth /workspace; \
    chown -R "${USER_ID}:${GROUP_ID}" /home/agent /claude-auth /workspace

COPY --chown=1000:1000 infrastructure/docker/settings.json /home/agent/.claude/settings.json
COPY --chown=1000:1000 infrastructure/docker/statusline.sh /home/agent/.claude/statusline.sh
COPY infrastructure/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh \
    /home/agent/.claude/statusline.sh \
    && chown -R "${USER_ID}:${GROUP_ID}" /home/agent/.claude

RUN printf '%s\n' \
    'export PATH=/home/agent/.dart-sdk/bin:/usr/local/go/bin:/home/agent/go/bin:$PATH' \
    'export GOPATH=/home/agent/go' \
    'export GOCACHE=/home/agent/.cache/go-build' \
    > /etc/profile.d/tiny-cosmos.sh

WORKDIR /workspace

USER ${USER_ID}:${GROUP_ID}

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/bin/bash"]
