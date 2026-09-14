---
name: assertion-may-belong-to-someone-else
description: An assertion can be satisfied by the stdlib or the encoder rather than by your code — os.Open's *fs.PathError names the file for you, and encoding/json rewrites ESC as a six-character escape; both made a security test vacuous
metadata:
  type: feedback
---

Before trusting a test, ask **who guarantees the thing it asserts**. If the answer
is the standard library, the encoder, or a framework, the test holds whatever your
code does and the mutation that should redden it stays green.

**Why:** two separate instances in one hexarena session (PRs #430/#431), both in
security tests, both caught only by mutation:

- `TestAMissingLogStillSaysWhichFile` asserted a wrapped error "still names the
  file". Deleting the path from the wrapper — `fmt.Errorf("read the log: %w", err)`
  — left it **GREEN**: `os.Open` returns a `*fs.PathError` whose own `Error()`
  already prints the path. The assertion was the standard library's guarantee
  wearing the function's clothes. Rewritten as a claim the code actually owns
  (`errors.Is(err, fs.ErrNotExist)` still resolves — a `%v` would flatten it), and
  the "names the file" assertion moved to a message built from **nothing**
  underneath it.
- A precondition asserted an encoded JSON message still carried an escape, by
  searching for the raw `0x1b` byte. It **failed immediately** — `encoding/json`
  writes a control character as a six-character JSON escape (backslash-u-0-0-1-b). Had it been written
  the other way round (assert absence), it would have passed while proving
  nothing. On the wire the six characters *are* the payload; the decoder turns
  them back into ESC.

**How to apply:** when an assertion is about a *string containing* something,
find the shortest path by which that something could get there without your code.
Prefer asserting on a value your code constructs from scratch (no wrapped error,
no encoder in between) over one it merely passes through. The mutation to cut is
"delete the contribution", not "break the feature" — and if it survives, the test
was never about you. Related: [[mutate-the-wiring-not-just-the-resolver]],
[[silent-write-check-needs-a-producer]], [[feedback-prove-regression-tests]].
