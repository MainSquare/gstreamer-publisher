# Task for dev-agent

Implement an approved bug fix in the gstreamer-publisher repository at /mnt/work/teleo/gstreamer-publisher (your scope: its local icm/ hub and this repo's code).

**Change:** Make `Publisher.Stop()` in `publish.go` safe against concurrent invocation. `Stop()` is currently called from two goroutines: the signal goroutine (SIGINT/SIGTERM/SIGQUIT in `Start()`) and the GStreamer bus-watch callback (`messageWatch` on EOS/error). Concurrent calls race on the check-then-act nil guards for `p.stopStdin` (double `close` panics) and on `p.pipeline`/`p.room`/`p.loop` (nil deref risk).

**Approved fix (owner-selected):** wrap the entire `Stop()` teardown body in a `sync.Once`. Keep the laziest working diff:
- Add `stopOnce sync.Once` to the `Publisher` struct (next to the existing `mu sync.Mutex`).
- Move the whole body of `Stop()` into a closure executed via `p.stopOnce.Do(...)`. The log line `logger.Infow("stopping publisher..")` may stay inside the Once body.
- No other refactors, no extra mutexes, no behavior changes.

**Process requirements (per repo icm/AGENTS.md):**
1. Create a change folder `icm/03_changes/active/2026-<mm>-<dd>--stop-sync-once/` (use today's date) with a minimal intake/README noting the concurrent-shutdown panic and the approved sync.Once fix.
2. Do the change in a Worktrunk worktree per the dev-agent policy, atomic commit, message format `<type>(<change-id>): <subject>` with trailer `Change-Id: <change-folder-slug>`.
3. Validation: run `gofmt` and `go vet ./...` / `go build ./...` if CGO/GStreamer headers are available; otherwise record best-effort static checks and note explicitly that CGO build was unavailable. Save validation evidence in the change folder (04_validation runbook conventions).
4. Report back: files changed, commit hash + Change-Id, validation results, and whether the change is on a worktree branch or merged to main.

## Acceptance Contract
Acceptance level: checked
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Implement the requested change without widening scope

Required evidence: changed-files, tests-added, commands-run, residual-risks, no-staged-files

Finish with a fenced JSON block tagged `acceptance-report` in this shape:
Use empty arrays when no items apply; array fields contain strings unless object entries are shown.
```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "specific proof"
    }
  ],
  "changedFiles": [
    "src/file.ts"
  ],
  "testsAddedOrUpdated": [
    "test/file.test.ts"
  ],
  "commandsRun": [
    {
      "command": "command",
      "result": "passed",
      "summary": "short result"
    }
  ],
  "validationOutput": [
    "validation output or concise summary"
  ],
  "residualRisks": [
    "none"
  ],
  "noStagedFiles": true,
  "diffSummary": "short description of the diff",
  "reviewFindings": [
    "blocker: file.ts:12 - issue found, or no blockers"
  ],
  "manualNotes": "anything else the parent should know"
}
```