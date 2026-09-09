#!/usr/bin/env bash
# Update the running publisher's allowlist: write a line to the FIFO, then SIGUSR1.
# Usage: update-allowlist.sh <identity> [identity ...]
set -euo pipefail
STATE=/tmp/gst-pub-test
[ -p "$STATE/fifo" ]     || { echo "no fifo — is run-publisher.sh running?" >&2; exit 1; }
[ -f "$STATE/publisher.pid" ] || { echo "no publisher pid — is run-publisher.sh running?" >&2; exit 1; }

USERS="$*"
echo "$USERS" > "$STATE/fifo"
kill -USR1 "$(cat "$STATE/publisher.pid")"
echo "allowlist -> [$USERS]; SIGUSR1 sent (watch publisher log: 'subscription permission applied')"
