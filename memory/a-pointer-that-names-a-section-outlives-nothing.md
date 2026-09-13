---
name: a-pointer-that-names-a-section-outlives-nothing
description: A cross-reference naming a heading breaks silently when the entry moves — name the CODE; the index is what carries a reader to wherever it now lives
metadata:
  type: feedback
---

**A pointer that names a SECTION is a bet that the section keeps its contents.
That bet loses, and nothing reports it.**

`DOC-001`. Moving thirty finished entries out of `TODO.md` § *Not done* broke
**seven** pointers that named one of them by title — `README.md` (`ENG-006`),
`docs/balance.md` and `internal/seed/guardboard_test.go` (`RAT-004`),
`internal/seed/mender_test.go` (`RAT-003`), `internal/seed/diglett_test.go` and
`internal/seed/battle_test.go` (`RAT-005`), and one inside `docs/decisions.md`
itself. Every pointer that named a **code** survived untouched, which is the whole
argument for the codes.

⚠️ **Two more had already been broken for eight days and nobody noticed** —
`CLAUDE.md`'s wall-of-charges pointer and `memory/hexarena-rating-gaps.md`'s own
heading, both naming `TODO.md` § *Not done* for entries that left the file on
2026-09-05. A stale pointer is invisible: it names a heading that still exists and
still has content, so a reader follows it, finds something plausible, and does not
know they are in the wrong place.

**A count in prose is the same defect at one remove.** `README.md` said *"§ *Not
done* — twelve items, not one of them ticked"*. It was **false** before this sweep
(forty-two items, thirty ticked) and true after it — the prose had drifted into
being right by accident, and then needed changing twice more the same day as two
items were filed. A count that is not derived goes stale the moment anything is
added.

**How to apply.**

- Point by code: `` → `docs/decisions.md` § `RAT-004` ``, never by heading text.
- ⚠️ After moving anything between documents, run
  `grep -rn 'TODO\.md' --include='*.go' --include='*.md' . | grep '§'` and check
  each hit against the moved set. A pointer naming a still-open entry's sub-section
  is fine; one naming a moved entry is not.
- Do not write a count of a list into prose beside it. If it must be written, say
  where to re-derive it.

Xem thêm [[todo-items-are-addressed-by-code]], [[a-heading-held-by-prose-decays]].
