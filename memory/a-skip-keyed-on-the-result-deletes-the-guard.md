---
name: a-skip-keyed-on-the-result-deletes-the-guard
description: SCR-013 — a test that skips where the machine cannot do the thing must PROBE the machine, never read the result it was going to assert on; "the art was copied" is true both on a box that cannot link and on a mechanism that stopped linking, so a skip keyed on it is off on every machine forever
metadata:
  node_type: memory
  type: reference
---

**A guard that has to tolerate a machine must probe the machine, and the probe
may never be the assertion's own reading.** `SCR-011` left five packages (six
tests) asserting that a scratch data directory *shared* the shipped art.
`testfixture.CopyData` shares with a hard link, falls back to a symbolic link,
then to a byte copy — and on Windows with the repository on `H:` and `TMP` under
`C:\…\Temp` **both links are refused**:

| call | refusal |
|---|---|
| hard link | `The system cannot move the file to a different disk drive.` |
| symbolic link | `Administrator privilege required for this operation.` |

So the copy is that box's normal, correct, designed path, and the guard was
unconditionally red there while saying nothing about why.

⚠️ **The obvious repair is the trap.** "Skip when the art was copied" reads like
the fix and is the guard deleting itself: *the art was copied* is equally true on
a machine that cannot link **and** on a mechanism that stopped linking — which is
the only failure the guard exists for. It would be green everywhere, forever, on
every machine, and nobody would ever see it fire.

What discriminates is a **probe**: attempt the two links yourself, in the very
directory the shares would have landed in, from a real source on the other
volume, through the *same seams* the mechanism uses. Refused both ways → skip,
quoting the refusals. Accepted and the art was copied anyway → fail. Those two
states are indistinguishable to any reading of the copy's own result.

**How to apply:** any `t.Skip` whose condition could also be produced by the bug
is not a skip, it is a deletion. Write the condition as an *independent
observation of the environment*, and owe it two tests — one where the
environment refuses and the guard skips, one where the environment allows and the
guard still fails. `internal/testfixture/guard_test.go` holds both.

Three more things that cost time here:

- **A cheaper invariant does not fix a platform problem.** Counting bytes
  written instead of comparing inodes is a better assertion — it sees a copy the
  inode comparison cannot price — but a copied picture is 17 MB however it is
  read, so `…StillSharesRatherThanCopies` was red on that box for exactly the
  reason the inode one was (`17300838 bytes written, want under 1048576`,
  measured under the simulated refusals). Two readings of one claim share one
  skip decision; they do not each invent one.
- **The seam is what makes the other platform testable here.** `linkFile` and
  `symlinkFile` in `internal/testfixture/data.go` already existed with the note
  *"a fallback nothing has ever run is not a fallback, it is code with an opinion
  about a platform nobody tested"*. Both arms — the skip and the failure — were
  demonstrated on macOS through them, using the real Windows error strings inside
  a real `*os.LinkError`, so the sentence a Windows reader will meet is the
  sentence that was measured.
- **The decision goes in one place.** It was six copies of one assertion, and
  `SCR-011` had already consolidated five identical `copyTree`s for that reason.
  Six copies of a *skip rule* is worse than six copies of an assertion: an
  assertion that drifts fails somewhere, a skip that drifts is a guard silently
  off in the one package nobody looked at. → `testfixture.RequireSharedArt`.

Same family as [[goldens-cannot-be-green-on-two-platforms]] — a test that pins
the machine of whoever ran it. That one pins the path separator, this one pinned
the filesystem's link support, and neither said so when it failed.
