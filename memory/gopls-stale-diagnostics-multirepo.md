---
name: gopls-stale-diagnostics-multirepo
description: "new-diagnostics (gopls) lag behind disk in the multi-repo platform root — false BrokenImport/undefined/IncompatibleAssign after edits or cross-repo merges; trust `go build`/`go vet` exit code, not gopls"
metadata: 
  node_type: memory
  type: feedback
---

The `<new-diagnostics>` block (gopls/compiler) frequently reports FALSE errors in the pet-platform root because the platform root is NOT a go workspace (no `go.work`; each service is a separate module under `../`). gopls operates on a stale/partial snapshot and can't resolve cross-module or just-added symbols.

Seen 3× in one session (2026-06-17):
1. After isme migration split: gopls showed `IncompatibleAssign` for `sqliteHistory.Migrations` even though the import swap was done + `go build` passed.
2. medioa2 cleanup: gopls `BrokenImport` "current file is not included in a workspace module" for every internal import + `undefined: defineConfig` — all noise; build+vet+161 tests passed.
3. isme db-backup: gopls `BrokenImport github.com/vukyn/isme/internal/domains/migration/models` (a package DELETED + merged in PR#60) on a history file that actually imports `kuery/bun/migrate`, plus `undefined: SETTINGS_ENDPOINT_DATABASE_BACKUP` (a const that WAS added). All stale.

**Why:** gopls caches pre-edit state and lacks a go.work spanning the repos, so freshly-added consts, post-merge import swaps, and cross-module refs surface as phantom errors.

**How to apply:** when `<new-diagnostics>` flags BrokenImport / undefined / IncompatibleAssign right after an edit or a cross-repo merge, do NOT panic or "fix" it. Verify against the AUTHORITATIVE source — run `cd <service> && go build ./...` (exit 0) + `go vet ./...` + `grep` the symbol on disk. Only act if the real compiler disagrees. The committer/coder verifying `go build` Success is the truth; gopls diagnostics are advisory and often wrong here. (Still verify with go build — don't blindly trust subagent reports either; both can be wrong, but in opposite directions.)
