#!/usr/bin/env bash
# Start a local LiveKit server in dev mode (accepts tokens signed devkey/secret).
set -euo pipefail
if docker ps --format '{{.Names}}' | grep -q livekit-dev; then
  echo "livekit-dev already running"; exit 0
fi
# --network host: livekit binds 127.0.0.1 inside the container, so port mapping
# (-p 7880:7880) yields 'connection reset by peer' from the host. host networking avoids it.
exec docker run --rm --name livekit-dev --network host livekit/livekit-server --dev
