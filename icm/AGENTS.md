# gstreamer-publisher specialist ICM hub

Technical specialist context for the `gstreamer-publisher` Go CLI — the LiveKit
publisher used by the Teleo browser-sandbox room containers.

## What this repository is

Fork of LiveKit's GStreamer publisher example (`github.com/livekit/gstreamer-publisher`
lineage, forked as `github.com/MainSquare/gstreamer-publisher`). A CLI that:

1. Parses `--url`, `--token`, `--delay` and a gst-launch-style pipeline string.
2. Joins a LiveKit room with `lksdk` (`server-sdk-go/v2`), `autoSubscribe=false`.
3. Discovers encoded VP8/Opus source pads in the pipeline and attaches
   `appsink`-based publisher tracks (see `track.go`), publishing video as
   `CAMERA` and audio as `MICROPHONE`.
4. Runs until SIGINT/EOS/pipeline error.

## How it is consumed

Built inside the `teleo-browser-sandbox` room container image
(`rooms/room/.docker/Dockerfile`, stage `gstreamer-publisher-builder`, via
`go install github.com/MainSquare/gstreamer-publisher@latest`). The room
server's node process spawns `/etc/livekit-gstreamer-pipeline.sh` as a detached
child with env `LIVEKIT_URL` / `LIVEKIT_PUBLISH_TOKEN` (and, since change
`2026-09-09--sandbox-stream-subscription-allowlist`, `LIVEKIT_ALLOWED_USERS`).

Key constraint: **builds pull this repo from the fork's main branch with
`@latest`** — pushing to `main` here propagates to the next room image build.

## Route by task

| If you need to… | Read |
|---|---|
| receive a change request or record change progress | `03_changes/CONTEXT.md` |
| understand the publish path | `publish.go`, `track.go`, `main.go` |
| validate locally | `04_validation/test-runbook.md` |

## Dev-agent rules (hard)

Full policy: `/mnt/work/teleo/icm/_shared/dev-agent-policy.md`.

1. Code changes in this repository are executed by a dev agent
   (`icm-repository-specialist`, implementation mode) in a Worktrunk worktree.
2. Ponytail ON while implementing — shortest working diff.
3. Atomic commits, message `<type>(<ticket-or-change-id>): <subject>` plus
   trailer `Change-Id: <change-folder-slug>`.
4. Validation evidence (build/lint output) recorded in the change folder
   before a change can close. If CGO/GStreamer dev headers are unavailable
   locally, record that and apply best-effort static checks (`gofmt`,
   targeted review) — note it explicitly.
