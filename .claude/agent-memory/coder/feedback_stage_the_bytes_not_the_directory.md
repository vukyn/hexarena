---
name: stage-the-bytes-not-the-directory
description: A per-process fixture stage held as bytes in memory has no cleanup problem and no cross-run staleness; a staged DIRECTORY has both, plus it silently kills a share guard keyed on the source
metadata:
  type: feedback
---

When an expensive per-test fixture has to be built once per test binary, hold the
**result as bytes in a package-level map**, not as a directory on disk.

**Why:** `SCR-011`'s second half (hexarena, 2026-09-08). `testfixture.Inject`
reloaded the library **37 times** per scratch directory (four call sites, but one
reload before every write — 31 skills + 3 origins + 2 characters), ~180 ms of a
~200 ms `scratchData`, ~300 times in `cmd/hexforge-tui` alone. Three problems the
obvious "build one staged directory and copy from it" design has, and the
in-memory version has none of them:

1. **Cleanup has no correct owner.** `t.TempDir`/`t.Cleanup` are per test, so the
   first test to ask would delete the stage while later tests still read it. The
   honest answer for a staged directory is "the OS, eventually". Staging bytes
   removes the question: `os.MkdirTemp` + `defer os.RemoveAll` inside the builder,
   and what escapes is a `map[string][]byte`.
2. **Cross-run staleness becomes impossible**, rather than merely guarded. Nothing
   is written between runs, so the injector compiled into the binary is always the
   one that ran. Within a run, key the stage on a **SHA-256 of the inputs**
   (source files byte for byte + the fixture constants), length-prefixing each
   piece — then a test that edits the source mid-run gets a fresh stage and the
   key can be reddened by dropping either half.
3. ⚠️ **A second link in a share chain can silently disable a guard.** hexarena's
   `rememberArt` refuses a write-through by comparing the source's stamp, and it
   *skips anything under `os.TempDir()`*. Had scratch directories shared their art
   **from** a staged directory (which is under TempDir) instead of from the
   committed one, the guard on the committed art would have gone dead with every
   test green. So: share from the real source, always.

**How to apply:** any "do this once per test binary" work in this repo. Split the
expensive deterministic part (staged) from the cheap per-directory part (art, ids,
anything a test may write to — keep those private per directory). State the
determinism as a precondition in the injector's doc comment; it is what makes
staging sound. Expose a `Stagings() int` counter so the arrangement is asserted as
a count, never as a stopwatch. Related: [[feedback_graph_rename_blind_spots]],
[[project_screen_extraction]].
