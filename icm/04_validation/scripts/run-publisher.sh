#!/usr/bin/env bash
# Start the publisher with an interactive allowlist FIFO.
# Usage: run-publisher.sh ["<gst pipeline>"] ["<initial allowed users>"]
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
STATE=/tmp/gst-pub-test
mkdir -p "$STATE"
[ -f "$HERE/tokens.env" ] || { echo "run mk-tokens.sh first" >&2; exit 1; }
# shellcheck disable=SC1091
source "$HERE/tokens.env"

REPO_ROOT="$(cd "$HERE/../../.." && pwd)"
BIN="${BIN:-$REPO_ROOT/gstreamer-publisher}"
[ -x "$BIN" ] || { echo "publisher binary not found at $BIN (run: go build ./...)" >&2; exit 1; }
URL="${LIVEKIT_URL:-ws://localhost:7880}"
# vp8enc (gst-plugins-good) instead of x264enc (gst-plugins-ugly — not in the dev shell)
PIPELINE="${1:-videotestsrc is-live=true ! video/x-raw,width=640,height=480 ! videoconvert ! clockoverlay ! vp8enc deadline=1 keyframe-max-dist=60 target-bitrate=1000000}"
ALLOWED="${2:-}"

rm -f "$STATE/fifo"; mkfifo "$STATE/fifo"
exec 3<>"$STATE/fifo"   # hold fifo open read-write: reader never blocks, never EOFs

"$BIN" --url "$URL" --token "$PUB_TOKEN" --allowed-users "$ALLOWED" -- $PIPELINE \
  < "$STATE/fifo" &
PID=$!
echo "$PID" > "$STATE/publisher.pid"
echo "publisher pid: $PID"
echo "update allowlist: $HERE/update-allowlist.sh alice bob"
wait "$PID"
