---
name: a-doc-about-a-scanner-trigger-triggers-it
description: Writing up a secret-scanner false positive re-creates it — quoting the offending literal in a second file gave gitleaks a brand-new finding; name the identifier, never reproduce the value
metadata:
  type: feedback
---

When documenting **why a scanner finding was accepted**, name the identifier and
describe the value — do not paste the value. Every copy is a fresh finding.

**Why:** measured on hexarena (PR #431). The repo's accepted finding was a
documentation example inside a note *about* credential literals. The
`docs/decisions.md` entry explaining that acceptance quoted the same assignment in
full — and `gitleaks detect` went from **clean to one leak the moment that commit
landed**, at a new file and line. The reflex is to add a second fingerprint to
`.gitleaksignore`; that just institutionalises the duplication. One copy of an
example is an example; two is a pattern the scanner is right about.

Two related traps from the same change:

- **`.gitleaksignore` and `.gitleaks.toml` are ONE change.** The ignore file's
  entries are 40-character commit fingerprints, which the default
  `generic-api-key` rule reads as high-entropy secrets — so adding the ignore file
  alone trades a known false positive for a new one. The config must allowlist
  `^\.gitleaksignore$` and keep `useDefault = true`.
- **`gitleaks detect` scans reachable history, including remote-tracking refs.**
  After amending a pushed commit away, the finding persisted until
  `git push --force-with-lease` moved `origin/<branch>`; the orphaned commit was
  still reachable locally through `refs/remotes/`. Re-verify after the push, not
  after the amend.

**How to apply:** after any commit that touches scanner config, accepted findings,
or prose about either, **run the scanner again** rather than reasoning about it.
The check is two seconds and the failure mode is a public repository that is dirty
on every future scan. Related: [[gitleaks-silence-is-not-safety]],
[[test-credential-literals-rule]].
