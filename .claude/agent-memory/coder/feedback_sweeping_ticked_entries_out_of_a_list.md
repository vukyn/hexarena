---
name: sweeping-ticked-entries-out-of-a-list
description: Moving finished entries out of TODO.md § Not done — rebuild from the survivors (a delete-by-range breaks on double blanks), prove verbatim with per-entry sha256, and expect most § pointers to name an OPEN sub-section
metadata:
  type: feedback
---

Moving `- [x]` entries out of a long checklist is a mechanical job with three
traps I hit on the 2026-09-13 hexarena sweep (30 entries, 2,663 lines).

**Why:** the brief demanded byte-identity against `git show HEAD:TODO.md`, so
every shortcut that "tidies" is a failure, and every shortcut that looks safe has
to be proven rather than argued.

**How to apply:**

- **Rebuild the section from the survivors; do not delete line ranges.** The
  obvious rule — delete `[start-1 .. last_non_blank]` so the preceding blank goes
  with the entry — is wrong wherever two entries are separated by **two** blank
  lines (2 of 42 pairs here) or by **none** (the CLI-001/CAST-001/CAST-002/
  CAST-004/DAT-002 run was contiguous with no blank at all). Re-emitting the
  survivors with a normalised separator, *except* where two survivors were
  already neighbours (then copy their original gap verbatim), handles both.
  Check afterwards that the diff is **deletion-only** on the source
  (`git diff -U0 f | grep -c '^+[^+]'` → 0) and **addition-only** on the target.
- **Prove verbatim per entry, not per file.** Parse the entries out of *both*
  `git show HEAD:<src>` and the new target with the same splitter, then compare
  SHA-256 per code. A whole-file diff cannot tell "moved" from "moved and
  reflowed"; 30 matching hashes can. It also catches an off-by-one in the
  end-of-entry walk-back, which a visual read never will.
- ⚠️ **Most `§ *Title*` cross-references will NOT be the ones you are looking
  for.** `grep -rn 'TODO\.md' --include='*.go' --include='*.md' . | grep '§'`
  returned ~60 hits; **7** named one of the 30. The bulk named sub-sections
  *inside a still-open entry* (NET-001's "Ban and pick", "Spectators", "The
  client", "step 5a", "The draft on the wire") and were correct as they stood.
  Resolve each hit to a **code** before touching it.
- ⚠️ **Some stale pointers are the previous sweep's, not yours.** Two hits
  (`CLAUDE.md:755` → an uncoded "the ninth narrow product had no board" entry,
  `memory/hexarena-rating-gaps.md:12` → "Four mechanics `Suggest` resolved and
  did not price") pointed at entries that left `TODO.md` on **2026-09-05**.
  Report them; do not fold them into this change's blast radius. The way to tell
  is whether the referent appears in the file's *pre-existing* region or in the
  block you just appended.
- ⚠️ **A pointer can be stale in the direction of being right.** `README.md`'s
  *"§ *Not done* — twelve items, not one of them ticked"* was FALSE before the
  sweep (42 items, 30 ticked) and true after. The brief warned me not to "fix"
  it — verify the count rather than assuming a number in prose is current *or*
  that a warning means it was always correct.
- **A rule whose compliant action costs more than its violation, and whose
  violation is invisible, decays on its own.** Ticking in place costs a
  character; moving costs a 20 KB cut and paste. The fix that holds is a command
  that fails loudly (`awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/'`),
  plus showing its **positive** case — a guard nobody has watched fail is not
  evidence it works.

Related: [[a-heading-is-not-a-rule]] (the 2026-09-05 sweep's own procedure, in
the repo's `memory/`), [[feedback_a_golden_can_be_red_before_you_touch_it]].
