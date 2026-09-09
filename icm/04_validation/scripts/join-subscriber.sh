#!/usr/bin/env bash
# Join the test room as a subscribe-only participant and log track events.
# Usage: join-subscriber.sh <identity>   (alice | bob | any other identity)
set -euo pipefail
command -v lk >/dev/null 2>&1 || {
  echo "lk CLI not found. Install: curl -sSL https://get.livekit.io/cli | bash" >&2; exit 1; }
IDENTITY="${1:?usage: join-subscriber.sh <identity>}"
HERE="$(cd "$(dirname "$0")" && pwd)"
[ -f "$HERE/tokens.env" ] || { echo "run mk-tokens.sh first" >&2; exit 1; }
# shellcheck disable=SC1091
source "$HERE/tokens.env"

case "$IDENTITY" in
  alice) TOKEN=$ALICE_TOKEN ;;
  bob)   TOKEN=$BOB_TOKEN ;;
  *)     TOKEN=$(lk token create --api-key devkey --api-secret secret \
                 --identity "$IDENTITY" --room "$LIVEKIT_ROOM" --join --valid-for 24h \
                 --grant '{"canPublish": false, "canPublishData": false}' \
                 | awk '/Access token:/{print $NF}') ;;
esac

lk room join --url "${LIVEKIT_URL:-ws://localhost:7880}" \
  --api-key devkey --api-secret secret --identity "$IDENTITY" --auto-subscribe \
  "$LIVEKIT_ROOM"
