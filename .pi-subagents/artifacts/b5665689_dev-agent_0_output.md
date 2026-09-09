Implemented the approved `sync.Once` fix for `Publisher.Stop()`.

Changed files: `publish.go` (worktree only — added `stopOnce sync.Once` field and wrapped the entire `Stop()` teardown body, including the `logger.Infow("stopping publisher..")` line, in `p.stopOnce.Do(...)`; no other changes). Change-folder docs (gitignored `icm/` hub, untracked by design): `icm/03_changes/active/2026-09-09--stop-sync-once/README.md` + `04_validation.md`.

Commits: `1062b73` `fix(2026-09-09--stop-sync-once): make Publisher.Stop idempotent under concurrent invocation` + trailer `Change-Id: 2026-09-09--stop-sync-once`.

Validation: `gofmt -l .` clean; `go vet ./...` exit 0; `go build ./...` exit 0 — GStreamer 1.0 dev headers were present, so the full CGO build ran (not skipped). No `*_test.go` files exist in this repo, so no test suite to run; no runtime race harness added (out of approved scope).

Branch status: change is on worktree branch `icm/2026-09-09--stop-sync-once--gstreamer-publisher` (at `/mnt/work/teleo/.worktrees/gstreamer-publisher/icm-2026-09-09--stop-sync-once`, based on `7369965`, current head of `joao/eng-2178-multi-browser-sandbox-support-for-group-sessions-private`). **Not merged to main** — per repo `icm/AGENTS.md`, pushing `main` propagates to the next room image build via `@latest`, so merge is an owner decision.

Isolation note: the initial cwd was the repo's primary checkout; I did not edit tracked code there — I created a dedicated worktree first. Only gitignored `icm/` change-folder files were written in the primary checkout.