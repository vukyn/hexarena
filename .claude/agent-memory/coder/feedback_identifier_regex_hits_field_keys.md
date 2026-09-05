---
name: identifier-regex-hits-field-keys
description: A regex rename over a Go identifier also rewrites struct field KEYS and the type's own declaration; and a mutation snapshot must cover every file a later step could touch, not the files the first step touched
metadata:
  type: feedback
---

**A `\bIdent\b` regex over Go source rewrites four different things, and only one
of them is what you meant:** the type/const *use*, the type's own *declaration*
(`type Step string` → `type wire.DraftStep string`, which does not compile), a
*struct field key* in a literal (`Step:` → `wire.DraftStep:`), and the *field
name in the type* (`Step Step` → both halves). Moving `draft.Step`/`draft.Entry`
into `internal/wire` hit all four on the first attempt.

**Why:** the damage is not always a compile error. The declaration and the field
key are, so those are found immediately — but prose in doc comments comes out
mangled and grammatically wrong (`An wire.DraftEntry carries two slices`,
`rather than a wire.DraftStep out of Turn`) and **nothing fails**. Two rounds of
hand review over `grep -n "^\s*//.*wire\."` were what found those.

**How to apply:**

- Exclude field keys with a negative lookahead (`(?<![\w.])Step\b(?!:)`) and a
  `.`-lookbehind for selectors, then **delete the declaration by hand** — it is
  moving, not being renamed.
- Grep the comments afterwards (`grep -n '^\s*//.*<newname>'`) and read every
  hit as English. A doc comment is the thing this repository is most careful
  about and a regex cannot write one.
- Prefer the graph's `refactor_tool(mode="rename")` when the symbol keeps its
  package. It cannot do this job — moving a symbol across packages needs a
  qualifier added, which is not a rename — so say that rather than reaching for
  sed silently. See [[feedback_graph_rename_blind_spots]].

⚠️ **The snapshot must cover every file a LATER step might touch.** The mutation
harness snapshotted the six files the mutations named; a clean-up regex two steps
later damaged `fixtures_test.go`, which was not in it, and `git checkout` was
unavailable because the file held an hour of uncommitted work. It had to be
repaired by hand from a brace-unbalanced state. Snapshot the whole *package*, not
the files in the plan. See [[feedback_scripted_revert_wrong_occurrence]] and
[[git-checkout-discards-to-head]].
