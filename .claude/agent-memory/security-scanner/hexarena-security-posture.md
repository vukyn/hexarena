---
name: hexarena-security-posture
description: hexarena scan baseline (re-confirmed 2026-09-02) — clean; no longer stdlib-only but govulncheck/osv read clean, so only gosec/gitleaks are informative; G115 in rng/combat and G101 in i18n are proven false positives; --verify is opt-in
metadata:
  type: project
---

hexarena's security surface is nearly empty and its scan baseline is clean: **0 critical, 0 high, 0 medium, 19 low/informational** as of 2026-09-02 (was 0/0/0/5 at the 2026-08-24 baseline; the count grew because the cast/TUI grew, not because anything got worse — every one of the 19 is a gosec finding adjudicated down).

**Why:** it is a local terminal game (engine + TUI + authoring CLI), with no network listener, no DB, no auth, no credentials, no `ui/` (so `npm audit` is N/A). ⚠️ **It is no longer stdlib-only** — the 2026-08-24 note said `go.mod` had no `require` block and that is stale: it now pulls charm.land bubbletea/bubbles/lipgloss v2, oksvg/rasterx, urfave/cli and `golang.org/x/text` (indirect). `govulncheck` and `osv-scanner` still read clean, but they are no longer vacuous — re-read them rather than assuming.

**How to apply:** only `gosec` and `gitleaks` are load-bearing. These judgments are made — confirm they still hold, don't re-litigate:

- **G115 `internal/core/rng/rng.go` (`int(draw % bound)`)**: false positive, provably. `bound := uint64(n)` where `n int` is guarded `> 0`, so `draw % bound < bound <= MaxInt` on both 32- and 64-bit.
- **G115 `internal/core/combat/combat.go` (`damage` / `divided`, the `uint64(denominator)` line in each)**: false positive. Every operand is guarded — the function returns 0 on any non-positive input and clamps `defense` at 0 — and the comment above the denominator derives its own overflow bound (largest reachable value 2,699 against a limit of 9,223,372,037).
- **G115 `internal/screen/preview.go` `ink()` and `cmd/hexforge-tui/form.go`**: not security. `ink` un-premultiplies alpha (`R*255/A`), which can only exceed 255 on a non-premultiplied `image.RGBA` the rasteriser cannot produce; the form ones convert internal field-index constants, never input.
- **G101 `internal/i18n/{vietnamese,english}.go`**: the whole translation table matched at LOW confidence. Nonsense.
- **G304 `forge.Load` / `forge.ArtImage`**: no privilege boundary. `Load` takes the operator's own data directory; `ArtImage` resolves an art path already refused by `cast.ValidateImagePath` (relative, no `..`, `.svg`/`.png` only) at parse time.
- **Everything in `internal/testfixture`** (G304/G301/G306): the package has **no non-test importer**, so it is in no shipped binary.
- **Hand-rolled splitmix64 in `internal/core/rng`** is by design: its only consumers are gameplay `Chance`/`Roll`; no output becomes an id, nonce, token or key. `crypto/rand` would *break* seed-reproducible battles.

**File modes — the earlier note was half right and is corrected.** `cmd/hexarena --log` writes **0o600** (`cmd/hexarena/main.go:344`). But `internal/forge.replaceFile`, which every authoring write and `SaveBattleLog` go through, chmods **0o644** and `MkdirAll`s **0o755** — that is what gosec's G302/G301 point at. It is authored game data (cast, skills, squads, battle logs) in the operator's own directory, so world-readable is right; do not "fix" it to 0600.

**Non-obvious, verified by experiment (re-check when the graphical client / PvP lands):** `--replay --verify` *does* detect a hand-edited log, but plain `--replay` renders a tampered log silently as authoritative — verification is **opt-in**. And because verification is determinism-based rather than a MAC/signature, a log proves a battle is *reproducible*, not that a particular player played it. Fine for single-player; a real anti-cheat gap the moment logs are submitted to a server.

**Graph state:** the code-review-graph is **now populated** (3,344 nodes / 45,574 edges at the 2026-08-31 build) — the old "0 nodes, skip forever" note is stale. But it goes stale fast: at the 2026-09-02 scan it was built at `62daafa` against a HEAD of `6fbf194` with **229 files changed**, including 4 of the 7 files carrying findings, so graph triage was skipped again and reachability was checked by hand (`git grep` for callers). security-scanner is read-only on the graph — never build it; ask planner/coder. Check freshness every run rather than trusting it.

Related: [[gitleaks-silence-is-not-safety]]
