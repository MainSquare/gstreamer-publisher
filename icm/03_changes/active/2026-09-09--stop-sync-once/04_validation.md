# Validation — 2026-09-09--stop-sync-once

Run in worktree
`/mnt/work/teleo/.worktrees/gstreamer-publisher/icm-2026-09-09--stop-sync-once`
(branch `icm/2026-09-09--stop-sync-once--gstreamer-publisher`, commit `1062b73`).

Environment: go1.26.4 linux/amd64, GStreamer 1.0 dev headers **present**
(`pkg-config gstreamer-1.0` OK) → full CGO build was available, not skipped.

| Check | Command | Result |
|---|---|---|
| Formatting | `gofmt -l .` | clean (no files listed) |
| Static analysis | `go vet ./...` | exit 0, no findings |
| Build (CGO + GStreamer) | `go build ./...` | exit 0 |

- No unit tests exist in this repo (no `*_test.go` files); no runtime shutdown
  race reproduction harness was added — fix is the owner-approved minimal
  `sync.Once` wrap, behavior verified by successful build/vet.
- Build artifact `gstreamer-publisher` (gitignored) removed from worktree
  after build; `git status` clean post-commit, nothing staged.
