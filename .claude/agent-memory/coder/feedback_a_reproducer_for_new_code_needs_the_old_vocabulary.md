---
name: a-reproducer-for-new-code-needs-the-old-vocabulary
description: A test that calls a function this step introduces cannot fail on the base commit — it fails to COMPILE, which proves nothing; re-express it in the base commit's own call chain, or run the base BINARY
metadata:
  type: feedback
---

When a step is asked for "a test that fails on `<base sha>`", and the test calls
something the step invents (`loadLibrary`, a new struct field), copying it into a
`git archive <base>` tree gives `undefined: loadLibrary` — a build failure, not a
red test. A build failure is indistinguishable from a typo and is not evidence
the defect existed.

**Why:** measured on the hexarena-tui `--data` fallback (step 2 of 3, base
`009b200`). The new `data_test.go` gave eight `undefined:` lines there. Two
things did produce real evidence, and both took minutes:

1. **The base commit's own vocabulary.** At `009b200` `run` was literally
   `forge.Load(parseOptions(...).dir)`, so a throwaway test in the archived tree
   spelling exactly that, with the *same test name*, went red with the shipped
   symptom: `start with no data directory anywhere: read
   internal/seed/data/combat.json: open ...: no such file or directory`.
2. **The base binary under a pty.** `git archive <base> | tar -x`, `go build`,
   run from an empty directory — see [[feedback_pty_smoke_test_for_hexarena_tui]],
   this client refuses a pipe before it ever loads books, so a plain
   subprocess only ever prints "not a terminal" and measures the wrong guard.
   The two binaries side by side is the strongest artefact: old dies on the
   relative path, new draws the menu.

**How to apply:** before writing the "fails on base" claim, ask which symbols the
test touches that the base does not have. If any, plan for two artefacts — the
old-vocabulary throwaway (delete it afterwards; it lives in the archive tree, not
the worktree) and, where the change is a startup path, the two binaries. Say in
the report which of the two the claim rests on. See also
[[feedback_a_green_expected_mutation_proves_nothing_by_itself]] and
[[feedback_pin_comparison_base_to_worktree_head]] for the base-pinning half.
