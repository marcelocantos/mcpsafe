#!/bin/bash
# Copyright 2026 Marcelo Cantos
# SPDX-License-Identifier: Apache-2.0
#
# Wrapper that starts mcpsafe, then runs fogbugz-mcp through it.
# Usage: claude mcp add fogbugz -- /path/to/scripts/fogbugz-mcp.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
MCPSAFE="${REPO_DIR}/mcpsafe"
CONFIG="${REPO_DIR}/examples/fogbugz.star"
BACKEND="https://hmssupport.fogbugz.com"

# Build if needed.
if [[ ! -x "$MCPSAFE" ]]; then
    (cd "$REPO_DIR" && go build -o mcpsafe .) >&2
fi

# Start mcpsafe and capture the listen address from stderr.
ADDR_FILE=$(mktemp)
trap 'kill "$PROXY_PID" 2>/dev/null; rm -f "$ADDR_FILE"' EXIT

"$MCPSAFE" -config "$CONFIG" -backend "$BACKEND" 2>"$ADDR_FILE" &
PROXY_PID=$!

# Wait for mcpsafe to print its listen address.
for i in $(seq 1 20); do
    if grep -q 'listening on' "$ADDR_FILE" 2>/dev/null; then
        break
    fi
    sleep 0.1
done

PROXY_URL=$(grep -o 'http://[^ ]*' "$ADDR_FILE")
if [[ -z "$PROXY_URL" ]]; then
    echo "mcpsafe failed to start" >&2
    cat "$ADDR_FILE" >&2
    exit 1
fi

# fogbugz-mcp sends the token in requests; mcpsafe overwrites it.
export FOGBUGZ_URL="$PROXY_URL"
export FOGBUGZ_TOKEN="proxied"

exec npx fogbugz-mcp
