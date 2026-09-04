---
name: test-only-what-changed
description: Run the packages a change touched, not `go test ./...`; the full suite is minutes of CPU and a starved machine reports "signal: killed" as a test failure
metadata:
  node_type: memory
  type: feedback
  modified: 2026-09-04T16:20:00.000Z
---

**Run the packages a change actually touched.** `go test ./...` here is minutes of CPU per invocation, and running it after every edit is most of a session's machine time.

**Why:** the user asked for it after a session ran the whole suite a dozen times over one data change. Measured on this repo: `internal/forge` **143s**, `cmd/hexforge-tui` **161s**, `internal/screen` ~20s, `internal/seed` ~200s with the golden set — against **0.67s** for the test the change was actually about. The expensive packages are the ones that fight battles, and they are expensive whether or not the change could reach them.

**How to apply:**

- Name the packages: `go test ./internal/seed/ ./internal/screen/ ./internal/i18n/ -count=1`. Add `cmd/…-tui` only when a golden or a client screen moved.
- `make golden` already runs the eight golden-owning packages; do not follow it with a full suite that runs them again.
- Keep the whole-suite run for the last check before pushing, once — not per edit.
- ⚠️ **A hung or killed package is not always a failing test.** `signal: killed` / `signal: terminated` from `go test` is the OS reaping the process, and on a loaded machine it lands on whichever package happens to be running. Measured: `internal/forge` "failed" twice with `signal: killed`, then passed in 143s with the machine idle. Before believing such a failure, re-run that package alone.
- ⚠️ **Check what is eating the machine before blaming the suite.** In the session that produced this note, 48 orphaned `while :; do :; done` shells from **another** session's flaky-test repro had been spinning for six hours at ~18% CPU each; its `jobs -p | xargs kill` cleanup never ran. `ps -Ao pid,pcpu,etime,command | sort -k2 -rn | head` finds them, `pkill -f "while :; do :; done"` clears them. A deliberate CPU-load loop must be killed by something that survives the command failing.
- Related: [[kill-test-ports-after-smoke]] — the same shape one layer down, a child process outliving the parent that was supposed to clean it up.
