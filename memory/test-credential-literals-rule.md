---
name: test-credential-literals-rule
description: "Never inline a credential-shaped literal — name it a fixture constant per package; GitGuardian/gitleaks fire on repeated password-shaped strings, on *KEY* identifiers, and on .gitleaksignore itself"
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-08-05T16:45:54.902Z
---

**Rule: a credential-shaped string is a named constant, never an inline literal.** One declaration per package, value deliberately self-describing:

```go
const (
	fixturePassword      = "fixture-value-not-a-credential"
	fixtureWrongPassword = "fixture-value-not-a-credential-mismatch"
	fixtureOldPassword   = "fixture-value-not-a-credential-old"
	fixtureNewPassword   = "fixture-value-not-a-credential-rotated"
	fixtureRefreshSecret = "fixture-refresh-signing-value-not-a-credential"
	fixtureProviderToken = "fixture-value-not-a-credential"
)
```

**Why:** GitGuardian flags password-shaped literals **appearing in a diff**, so a file that repeats one makes *every future edit to that file* trip the scanner — it fired on gardener PR #82 for a test literal that had been there for months, because a new subtest reused it. Second reason, independent of scanners: a value asserted against a hash built from it (`cryp.HashArgon2id(...)`) must agree across every site, which only holds with one declaration instead of a dozen retypings.

**Applies to any credential-shaped field**, not just passwords: `Token:`, `Secret`, `ApiKey`, `Password`. Not to a self-evidently fake low-entropy argument (`postApiKey(t, "mk_dead_key")` was left alone — no field assignment, no entropy).

## Scanner triggers that are NOT the value

⚠️ **A `*KEY*` identifier assigned any string literal** matches gitleaks' `generic-api-key`. medioa2 had `const STORAGE_KEY_PREFIX = "medioa2.recent_buckets"` — a localStorage namespace that **ships in the browser bundle**, about as public as a string gets. Renaming to `STORAGE_NAMESPACE` fixed it and was more accurate anyway (it is not a key; `keyFor()` appends the user id). Prefer `NAMESPACE`/`PREFIX` over `KEY` for non-secrets.

⚠️ **High-entropy placeholders in `demo/` mocks count too.** isme's `demo/aurora-invite-design.html` had a realistic random invite token (entropy 4.74). Use an obviously-fake placeholder (`token=INVITE-TOKEN-PLACEHOLDER-shown-once`) — the mock keeps its meaning. Do not delete the mock (see [[demo-mock-source-of-truth]]).

## `.gitleaksignore` discipline

**History only.** A live literal gets renamed/extracted, never listed. Each entry carries a reason and, when the value could not be *proven* fabricated, says so instead of asserting it was fine.

⚠️ **Committing `.gitleaksignore` makes gitleaks flag the file itself** — its entries are 40-char commit fingerprints, read as high-entropy secrets. Every entry added to silence a finding becomes a finding. Fix with `.gitleaks.toml`:

```toml
[extend]
useDefault = true   # extend, never replace — a hand-written rule list silently
                    # loses every detector gitleaks adds later
[allowlist]
paths = ['''^\.gitleaksignore$''']
```

## Reading a gitleaks run

`gitleaks detect` scans **git history** (a working-tree fix does not clear a historical finding). `--no-git` scans the filesystem including gitignored files, so `.env` and `certs/*.pem` show up — expected, they are where secrets belong. **Verify with `git ls-files --error-unmatch <file>` before calling anything a leak**; only a TRACKED file is a repo leak.

## Status

Done: **gardener #83** (~20 literals), **isme #87** (36 literals + the mock token), **medioa2 #90** (3 + the `STORAGE_KEY_PREFIX` false positive). isme and medioa2 carry `.gitleaks.toml` + `.gitleaksignore`; gardener needs neither (no historical findings). All three: `gitleaks detect` → no leaks found.

Not audited for this: rainy, memz, tomatime, kuery, and the gobuild `platform-service` template (which ships no tests today, so a new service starts clean — but the template is where the convention should land if tests are ever added).
