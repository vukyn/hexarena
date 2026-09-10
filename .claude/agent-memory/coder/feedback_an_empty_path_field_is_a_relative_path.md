---
name: an-empty-path-field-is-a-relative-path
description: filepath.Join("", "cast.json") is "cast.json", so an empty dir field silently reads/writes the CWD — wrap the field in a type so the join cannot be written by hand, and let an AST bijection own the case list
metadata:
  type: feedback
---

An empty string in a path field is **not** a refusal — it is a relative path.
`filepath.Join("", "cast.json") == "cast.json"`, measured. So a struct that
grows a "no directory" case by leaving its `dir string` empty turns every join
site into a read from, or a write into, whatever directory the process was
started in.

**Why:** `forge.Library` gained an embedded-data constructor with no directory.
Sixteen call sites joined `l.dir`. Leaving `dir` empty would have been *worse*
than the old behaviour, which was to refuse: a refusal names the problem, a
relative path finds a stranger's file. The eight string-returning accessors
handed out `cast.json`, `skills.json`, `assets`, `battles`… all silently valid.

**How to apply:**

- Make the wrong use **not compile**, not merely tested. Change the field's
  *type* (`dir string` → `home dataHome{dir string}`) so `filepath.Join(l.home,
  x)` is a type error. Then one unexported method is the only expression in the
  package that can build a path, and it is the only place the emptiness is
  checked. A consumer added later cannot skip the check — there is nothing to
  write the join out of.
- Two answer shapes, and they are not interchangeable: an accessor whose
  signature is a bare `string` answers `""` (the one string that cannot be
  opened, written or walked by accident); anything that *can* return an error
  refuses with a sentinel. Funnel the writes through one function so the loud
  half covers all of them at once.
- ⚠️ **A table asserting "these accessors refuse" is exactly the shape that
  passes while a different accessor leaks.** Own the case list from the source:
  an AST walk over the package collecting every `FuncDecl` whose body holds a
  **SelectorExpr** naming the field, held *bijective* with the decision table —
  unknown reader = red, stale row = red. Match the selector, never the
  identifier: the field name also appears as a composite-literal key and as the
  struct declaration, and an identifier match counts both, giving rows that have
  no decision to make and a walk that cannot be made to fail. Keep a second,
  smaller allowlist for whatever is allowed past the seam to the raw string.
- Assert the leaked value by name (`got "cast.json"`, not `want ""`) — the
  failure then says *which defect it is*.
- Verified by mutation: deleting the guard reddened 8 of 8 accessors; adding a
  seventeenth accessor with no row reddened both halves of the walk.

Related: [[a-guard-on-the-wrong-side-of-the-gate]],
[[feedback_graph_rename_blind_spots]] (the new test file was untracked, so the
graph reported every function it covers as a test gap).
