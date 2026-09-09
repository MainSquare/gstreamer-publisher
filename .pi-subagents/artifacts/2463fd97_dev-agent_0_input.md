# Task for dev-agent

Finish work package 1 of change 2026-09-09--sandbox-stream-allowlist-dynamic-updates in THIS repository (owner-approved to complete the in-flight work). A previous dev agent left uncommitted changes: `git diff` shows ~61 insertions / 16 deletions in publish.go implementing the stdin reader goroutine + SIGUSR1 handling. The plan is at /mnt/work/teleo/icm/03_changes/active/2026-09-09--sandbox-stream-allowlist-dynamic-updates/03_plan/plan.md (work package 1); context in 01_intake.md.\n\nDo exactly:\n1. Read the plan (WP1) and the uncommitted diff (git diff). Verify it matches the plan: stdin reader goroutine (bufio.Scanner) storing latest complete line as comma-split allowlist in a mutex-protected field; SIGUSR1 in signal.Notify applies the latest list via SetSubscriptionPermission (TrackPermissions with ParticipantIdentity + AllTracks: true, same shape as the startup flag; log COUNT only, never identities); blank line / no line yet → no-op; SIGINT/SIGTERM/SIGQUIT and the startup --allowed-users flag unchanged; goroutine terminates cleanly in Stop(). Fix any deviation, change nothing else.\n2. Check main.go was not required to change (plan expects no flag changes) — if the diff touches main.go unnecessarily, flag it.\n3. Validation: gofmt -l .; attempt go build ./... and go vet ./... (CGO/GStreamer headers may be unavailable — if so record that explicitly and compensate by verifying lksdk/protocol signatures against module sources in /home/jpporta/go/pkg/mod/github.com/livekit + careful diff re-read, same approach as the previous change's evidence).\n4. Commit as ONE atomic commit: message feat(2026-09-09--sandbox-stream-allowlist-dynamic-updates): apply stdin allowlist updates on SIGUSR1, trailer 'Change-Id: 2026-09-09--sandbox-stream-allowlist-dynamic-updates'.\n5. Append evidence to /mnt/work/teleo/icm/03_changes/active/2026-09-09--sandbox-stream-allowlist-dynamic-updates/04_implementation/iterations/wp1-gstreamer-publisher.md (create it): files, commit hash, validation results, open risks.\n6. Report: files changed, commit hash, validation commands with results, open risks. Do NOT push anywhere.

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