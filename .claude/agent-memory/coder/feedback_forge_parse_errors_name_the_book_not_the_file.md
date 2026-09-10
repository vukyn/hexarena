---
name: forge-parse-errors-name-the-book-not-the-file
description: forge.Load wraps a READ with the full path but returns a PARSE error untouched — a malformed skills.json says "decode skill book" and names no file, so a test asserting the file name is asserting something the code does not do
metadata:
  type: feedback
---

`internal/forge.loadBooks` is fifteen pairs of `read(x)` then `Parse(raw)`. The
reader wraps: `read %s: %w` with `filepath.Join(dir, name)`, so a **missing**
book refuses with `read /path/combat.json: open ...: no such file or directory`.
The parser's error is returned **unwrapped**, so a **malformed** book refuses
with `decode skill book: invalid character ',' ...` — the book, not the file, and
no path at all.

Measured (2026-09-10) by breaking `skills.json`, `combat.json` and `cast.json`
in turn: all three give `decode <x> book: …`, none names its file.

**Why it matters:** a spec or a test comment that says "the error still names the
file" is true for one of those two cases and false for the other. Writing the
assertion from the spec instead of from a probe gives a test that is red for a
reason that has nothing to do with the change under review — and the tempting
"fix" is to edit `internal/forge`, which a caller-side step is usually not
allowed to do.

**How to apply:** before asserting on a `forge.Load` refusal, break the fixture
the way the test means to and print the error once. Assert `"combat.json"` for an
absent book and `"skill book"` for a syntax error. If a step really wants file
names on parse errors, that is a change to `loadBooks` and its own item — every
one of the fifteen would need the same wrap, and the golden-free packages that
match on those strings would move with it. Related:
[[feedback_a_refusal_can_be_right_for_the_wrong_reason]].
