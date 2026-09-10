---
name: a-search-that-returns-zero-needs-a-known-positive
description: grep reported zero matches for a Vietnamese string that was really in the binary — a zero from a search is a fact about the SEARCH until a known-positive control says otherwise
metadata:
  type: feedback
---

Verifying `v0.4.0` meant proving the shipped binary carried a new sentence and
had dropped the old one. `grep -a` and `strings | grep` both answered **0** for
every Vietnamese string — including one in the *previous* tag's binary that was
certainly there, because that release drew it on screen.

The zero was about the tool, not the binary. Searching the bytes explicitly
answered at once:

```
v0.4.0: new_vi=1  old_vi=0
v0.3.1: new_vi=0  old_vi=1
```

```python
b = pathlib.Path(binary).read_bytes()
print(b.count("máy cầm đội kia".encode()))
```

**Why:** a search returning nothing has two explanations that look identical —
the thing is absent, or the search cannot see it. Encoding, locale, a tool's
binary handling, a minimum-run-length, a scope that excludes the file: each
turns a present string into a clean, confident zero. The English half of the
same check worked perfectly, which is exactly what made the Vietnamese zero
believable.

**How to apply:** before believing a zero, run the same search against something
you already know is there — the previous release, the file you just wrote, the
line you are looking at. Here the control was one command and it inverted the
conclusion. If no control is available, say the result is "the search found
nothing", never "it is not there". Same discipline as
[[a-guard-on-the-wrong-side-of-the-gate]]: ask where the check would fire before
trusting that it fired.

⚠️ This also caught a scoping version of the same error twice in one session:
`.Squads()` "appears nowhere under `cmd/hexarena-tui`" was true and the
conclusion drawn from it was false — the call is in the shared screen that
client draws. And a survey of `forge.Load` call sites missed `runCheck`, whose
load is spelled `forge.Inspect`. **A grep's scope is not the world's scope.**
