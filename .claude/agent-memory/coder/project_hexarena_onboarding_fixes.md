---
name: hexarena-onboarding-fixes
description: hexarena post-onboarding fixes landed 2026-08-24 (LICENSE, .gitignore .claude/, unverified-replay notice, go 1.27.0, Makefile); plus the govulncheck-toolchain gotcha that recurs on every Go bump
metadata:
  type: project
---

hexarena's `/pet-onboard` follow-ups were implemented 2026-08-24: MIT LICENSE
copied byte-identical from `sgo`/`speedtest`, `.claude/` added to `.gitignore`,
an `unverified:` notice printed when `--replay` runs without `--verify`,
`go.mod` bumped to `go 1.27.0`, and a thin `Makefile` added whose `golden`
target hides the `-update` footgun. Root platform `CLAUDE.md` gained the two
missing repo entries (`hexarena`, `worldbit`) — the root is not a git repo, so
that edit needs no commit.

**Why:** the repo was newly onboarded and drifted from its standalone siblings
(`sgo`/`gobuild`/`speedtest`/`worldbit`), which all carry a LICENSE and a
Makefile; the platform repo list had silently fallen two repos behind.

**How to apply:**
- **Recurring Go-bump gotcha:** after raising a `go` directive past the version
  the installed `govulncheck` was built with, that binary fails with
  `package requires newer Go version goX.Y (application built with goX.Z)` —
  it is not a code problem. Rebuild the scanner against the new toolchain, e.g.
  `GOTOOLCHAIN=go1.27.0 go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.
  Applies to every repo bump, not just hexarena. See [[go-toolchain-bumps]].
- go1.27.0 cleared all 8 stdlib advisories hexarena reported under go1.26.5
  (`GO-2026-6218/-6091/-6090/-6089/-6088/-5972/-5942/-5026`) — a plain
  directive bump is the whole fix when a repo has no third-party deps.
- Sibling Makefiles open with `include ./.env` (they keep `VERSION` there).
  hexarena has no `.env`, so that line would break every target — omit it
  rather than copying it, and guard/skip a `tag` target that has no version
  source.
- `cmd/hexarena` has **no tests at all** (package `main`, prints straight to
  stdout); the graph flags `replay` as a coverage gap. Verify changes there by
  running the binary, and expect the gap to keep being reported.
