#!/bin/bash
set -eu

AUTH_DIR=/claude-auth
CLAUDE_DIR=/home/agent/.claude
CLAUDE_CONFIG=/home/agent/.claude.json

mkdir -p "$AUTH_DIR" "$CLAUDE_DIR"

if [ -f "$AUTH_DIR/.credentials.json" ]; then
    cp "$AUTH_DIR/.credentials.json" "$CLAUDE_DIR/.credentials.json"
    chmod 600 "$CLAUDE_DIR/.credentials.json"

    cat > "$CLAUDE_CONFIG" <<'EOF'
{"hasCompletedOnboarding":true,"projects":{"/workspace":{"hasTrustDialogAccepted":true}}}
EOF
fi

(
    set +e

    while true; do
        if [ -f "$CLAUDE_DIR/.credentials.json" ] && \
           { [ ! -f "$AUTH_DIR/.credentials.json" ] || \
             [ "$CLAUDE_DIR/.credentials.json" -nt "$AUTH_DIR/.credentials.json" ]; }; then

            tmp=$(mktemp "$AUTH_DIR/.credentials.json.tmp.XXXXXX")
            cp "$CLAUDE_DIR/.credentials.json" "$tmp"
            chmod 600 "$tmp"
            mv -f "$tmp" "$AUTH_DIR/.credentials.json"
        fi

        sleep 5
    done
) &

exec "$@"
