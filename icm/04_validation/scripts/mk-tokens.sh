#!/usr/bin/env bash
# Generate dev-mode tokens (devkey/secret) for publisher testing.
# Usage: mk-tokens.sh [room]   (default room: test)
# Writes tokens.env next to this script and prints the decoded grants.
set -euo pipefail
command -v lk >/dev/null 2>&1 || {
  echo "lk CLI not found. Install: curl -sSL https://get.livekit.io/cli | bash" >&2; exit 1; }

ROOM="${1:-test}"
HERE="$(cd "$(dirname "$0")" && pwd)"

tok() { # identity, extra-grant-json (optional)
  local identity="$1" grant="${2:-}"
  local args=(--api-key devkey --api-secret secret --identity "$identity"
              --room "$ROOM" --join --valid-for 24h)
  [ -n "$grant" ] && args+=(--grant "$grant")
  lk token create "${args[@]}" | awk '/Access token:/{print $NF}'
}

PUB_TOKEN=$(tok publisher)
ALICE_TOKEN=$(tok alice '{"canPublish": false, "canPublishData": false}')
BOB_TOKEN=$(tok bob '{"canPublish": false, "canPublishData": false}')

cat > "$HERE/tokens.env" <<EOF
LIVEKIT_ROOM=$ROOM
PUB_TOKEN=$PUB_TOKEN
ALICE_TOKEN=$ALICE_TOKEN
BOB_TOKEN=$BOB_TOKEN
EOF
echo "wrote $HERE/tokens.env (room=$ROOM)"
