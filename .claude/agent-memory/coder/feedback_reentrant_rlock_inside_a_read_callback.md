---
name: reentrant-rlock-inside-a-read-callback
description: Calling an accessor that takes the same RWMutex from inside a Read callback is a self-deadlock whenever a writer is queued — read every such fact BEFORE entering the callback; neither the suite nor -race saw it
metadata:
  type: feedback
---

Inside a `Mirror.Read`-style callback, do not call anything that takes **the same
lock again** — not even a read lock. Read every fact you need before entering the
callback and close over the values.

**Why:** Go's `sync.RWMutex` queues a waiting **writer** ahead of new readers, so
a second `RLock` on the same goroutine while a writer is blocked is a
self-deadlock. Measured in hexarena step 5b: `model.stepped` called
`session.seat()` inside `session.read`'s callback, and `Client.Seat()` →
`Mirror.Seat()` takes that mirror's read lock — which `Mirror.Read` was already
holding. ⚠️ **The whole suite passed and `-race` was clean**, because whether it
hangs depends on `Receive` happening to be waiting to write at that instant: about
one run in ten, which is the same shape as the `session.out` sighting the file's
own comment records. It was found by *reading the lock discipline by hand* during
the graph self-review, not by any test.

**How to apply:**

- The rule to hold is the one already written down for these files: **the mirror's
  lock, then the session's, and never nested the other way** — and the way to keep
  it is that every accessor releases its own mutex *before* reaching for another
  object's. Check each accessor you are about to call from inside a callback for
  what lock it takes.
- Facts about *the match* (which seat, which pool, which allowance) are not facts
  about *this reading*, so they do not belong inside the reading's lock at all.
  Hoisting them is the fix and it is also the clearer code.
- A lock-order claim is not testable by the detector. Write it as a comment beside
  the hoist naming the accessor and the lock, so the next reader who moves the
  line back knows what they are moving.

See [[hexarena-pvp-lobby]] (an RWMutex held across a callback deadlocks against
the next writer, and two readers sitting happily beside each other is why a
"two readers cannot both be in" test measures nothing) and
[[measure-the-thing-a-bound-bounds]].
