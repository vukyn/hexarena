---
name: gitleaks-silence-is-not-safety
description: "gitleaks' default rules miss Postgres/Neon URIs and single-line PEM env values — \"no leaks found\" only means no rule matched"
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-08-14T06:45:42.847Z
---

A gitleaks run returning **0 findings proves no RULE MATCHED, not that no secret is
present**. Measured on gardener 2026-08-14: `gitleaks dir .env` reported clean while
that file held a live Neon DSN, an RSA signing key and the owner password. Its default
ruleset does not match a `postgres://user:pass@host/db` URI, and an RSA key stored as a
single-line `\n`-escaped env value is not a PEM block, so neither rule fires.

**Why:** a clean gitleaks report is the evidence usually quoted for "no secret leaked",
and here it would have been quoted for a file full of them.

**How to apply:** when judging whether a secret escaped, use git itself —
`git ls-files --error-unmatch .env`, `git check-ignore`, and above all
`git log --all -- .env` (0 commits = never committed = no rotation needed). Quote those,
not the scanner. Report gitleaks as "no rule matched", never as "no secrets".

Across the platform root every `.env` is untracked with 0 commits touching it, so no
rotation is outstanding; all are `chmod 600` as of 2026-08-14. That is local-disk
hygiene only and does not survive a fresh clone — see [[no-artifact-in-pet-platform]]
for the sibling rule about what belongs in this workspace.
