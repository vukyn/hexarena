---
name: hexarena-cast-json-must-be-in-tool-form
description: hexarena — cast.json and origins.json must be byte-equal to what forge's Marshal writes, so a character cannot be hand-appended at the end of the file
metadata:
  type: project
---

`TestTheCommittedBooksAreInTheFormTheToolWrites` (in `internal/forge`) reads the
**committed** `cast.json` and `origins.json` and compares them with
`library.Characters().Marshal()` / `library.Origins().Marshal()`. Marshal sorts by
id, so appending a new character to the end of `cast.json` — the obvious way to
keep a data diff to pure additions — fails with the first differing line:

```
line 715 committed: "id": "pokemon.gastly",
line 715 tool:      "id": "pokemon.dratini",
```

**Why the test exists:** `hexforge new` rewrites the whole book on every append. If
the committed file is already the tool's bytes, adding a character is a one-block
diff a reviewer can read; if it is not, the first append reformats everything and
the character is one block inside a file-sized diff.

⚠️ **Only those two files are held to it.** `skills.json`, `archetypes.json`,
`builds.json`, `squads.json` and `statuses.json` may be hand-appended and stay in
whatever order they are in — which is what keeps *their* diffs pure additions.
This is also why the older warning against running data through the tool
([[hexarena-shipping-a-character]]'s Lapras episode, where a tool pass re-sorted
`species.json` and re-spelled seven unrelated skills) is not a reason to skip it
here: the rule is **cast/origins through Marshal, everything else by hand**.

**How to apply:** author the character wherever it is convenient, then rewrite the
one file through the tool. A throwaway test inside `internal/forge` is enough —
`Load(shippedDataDir)`, `Characters().Marshal()`, `os.WriteFile` over `castFile` —
and `git diff --stat` should still report cast.json as insertions only, because
git renders the sorted insertion as one block. Delete the throwaway before
committing.
