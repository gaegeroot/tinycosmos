
FROM node:22-bookworm-slim

WORKDIR /workspace

# System dependencies
RUN apt-get update && apt-get install -y \
    git \
    bash \
    ca-certificates \
    curl \
    nano \
    procps \
    unzip \
    zip \
    python3 \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code

# Install OpenCode
RUN npm install -g opencode-ai

# Install ngrok
RUN curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
    | tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
    && echo "deb https://ngrok-agent.s3.amazonaws.com buster main" \
    | tee /etc/apt/sources.list.d/ngrok.list \
    && apt-get update \
    && apt-get install -y ngrok \
    && rm -rf /var/lib/apt/lists/*

# Create unprivileged agent user matching host UID/GID
ARG USER_ID=1000
ARG GROUP_ID=1000

RUN set -eux; \
    if getent group "${GROUP_ID}" >/dev/null; then \
        AGENT_GROUP="$(getent group "${GROUP_ID}" | cut -d: -f1)"; \
    else \
        groupadd --gid "${GROUP_ID}" agent; \
        AGENT_GROUP=agent; \
    fi; \
    useradd --create-home --uid "${USER_ID}" \
        --gid "${AGENT_GROUP}" \
        --shell /bin/bash agent; \
    mkdir -p /home/agent/.claude /home/agent/.config/opencode /claude-auth; \
    chown -R agent:"${AGENT_GROUP}" /home/agent /claude-auth

# Install Claude Code configuration
COPY --chown=agent:agent settings.json /home/agent/.claude/settings.json
COPY --chown=agent:agent statusline.sh /home/agent/.claude/statusline.sh

RUN chmod +x /home/agent/.claude/statusline.sh

# Copy entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN sed -i 's/\r$//' /usr/local/bin/docker-entrypoint.sh \
    && chmod +x /usr/local/bin/docker-entrypoint.sh

# Environment
ENV HOME="/home/agent"

# Switch to non-root user
USER agent

# Shell configuration
RUN echo 'export PS1="checkin:\w$ "' >> /home/agent/.bashrc && \
    echo 'alias ll="ls -alF"' >> /home/agent/.bashrc && \
    echo 'alias la="ls -A"' >> /home/agent/.bashrc && \
    echo 'alias l="ls -CF"' >> /home/agent/.bashrc

WORKDIR /workspace

ENTRYPOINT ["/bin/bash", "/usr/local/bin/docker-entrypoint.sh"]
CMD ["/bin/bash", "-i"]
