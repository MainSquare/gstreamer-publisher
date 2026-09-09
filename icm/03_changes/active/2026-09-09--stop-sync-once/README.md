# 2026-09-09 -- stop-sync-once

## Intake

**Repo:** gstreamer-publisher · **Type:** bug fix · **Status:** implemented (worktree branch, not merged)

**Problem:** `Publisher.Stop()` is called from two goroutines — the signal
handler goroutine started in `Start()` (SIGINT/SIGTERM/SIGQUIT) and the
GStreamer bus-watch callback `messageWatch` (EOS / pipeline error). When both
fire concurrently, the check-then-act nil guards race:

- `p.stopStdin` can be double-`close`d → panic
- `p.pipeline` / `p.room` / `p.loop` can become nil between check and use → nil deref

**Approved fix (owner-selected):** wrap the entire `Stop()` teardown body in a
`sync.Once`. No other refactors, no extra mutexes, no behavior changes.

## Iterations

### 2026-09-09 — implementation

- Added `stopOnce sync.Once` field to `Publisher` (next to existing `mu sync.Mutex`).
- Moved whole `Stop()` body into `p.stopOnce.Do(func() { ... })`; the
  `logger.Infow("stopping publisher..")` line stays inside the Once body.
- Diff: `publish.go` only, +8/-12 lines (indentation-driven).

## Validation

See `04_validation.md` in this folder.

## Decision log

- Fix shape (`sync.Once` vs mutex/channel-close rework) was owner-approved
  before dispatch; no deviations made during implementation.
- Change is on worktree branch
  `icm/2026-09-09--stop-sync-once--gstreamer-publisher`, based on
  `7369965` (current `joao/eng-2178-multi-browser-sandbox-support-for-group-sessions-private`
  head, ahead of `main`). **Not merged** — note: repo `icm/AGENTS.md` warns
  that `main` is consumed by room image builds via `@latest`.
