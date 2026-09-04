---
name: a-well-formed-measurement-can-measure-nothing
description: Two ways a check looked right and checked nothing (an RWMutex deadlock test with no writer; a digest whose framed names were shifted), and the pty recipe that caught the second
metadata:
  type: feedback
---

When a guard is added because "otherwise X deadlocks/diverges", **build the X and
watch the guard's absence produce it** before believing the test. Twice in one
session a check was well formed, stable, and blind.

**Why:** both failures were invisible to the whole suite and to `-race`, and both
were caught by something outside it — a mutation, and running the binary.

**How to apply:**

- **An RWMutex is not a Mutex, and a "lock held too long" test needs a WRITER.**
  The plan said holding the read lock across a blocking callback "deadlocks the
  client against itself on the very first turn". It does not: `sync.RWMutex`
  admits several readers, and if the only writer is the goroutine already blocked
  in the callback, nobody ever queues. Inverting the release **passed** the first
  test. What is actually fatal is a third party writing while the callback waits —
  Go queues a waiting writer ahead of new readers, so the renderer then blocks
  behind the writer, and the renderer is what the callback is waiting for. The
  test has to be three goroutines (blocked reader, writer, reader) with a bounded
  wait on each.
- **A digest can be well formed, stable, and always unequal.** `seed.digest`
  frames each file with its **name**, and the embed's names are `data/x.json`
  while a `--data` directory is rooted *at* `data`. Stripping the prefix from the
  hashed name as well as from the read path made every unedited directory read as
  edited — both digests fine, both stable, nothing red. Fix is a wrapper `fs.FS`
  that trims only on `Open`, so one name list serves reading and framing. Any new
  digest-of-a-directory needs `DigestOf(realDir) == DataDigest()` as a test.
- **Both were caught outside `go test`.** The digest one only surfaced by running
  the binary; see [[pty-smoke-test-for-hexarena-tui]] for the recipe.
- **A derived set is what makes a "which screen / which producer" table able to
  fail.** Two screens that both word any id cannot tell a swapped table from a
  right one. Deriving *both* sets from a real producer and asserting they are
  disjoint and total gives the walk a `default` arm, and that arm is what fires.

Related: [[measure-which-guard-masks]], [[fixture-decides-what-is-visible]].
