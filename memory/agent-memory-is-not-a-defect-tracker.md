---
name: agent-memory-is-not-a-defect-tracker
description: A defect recorded only in an agent's memory has no code and is not in the index — file it in TODO.md or the next occurrence looks like the first
metadata:
  type: feedback
---

**A defect written down in `.claude/agent-memory/` is remembered, not filed.** It
carries no `AREA-NNN`, so it is absent from `TODO.md` § *The codes*, which is the
only list anybody greps.

`NET-002`. `internal/socket`'s `TestALateTimeoutLeavesTheLiveAllowanceArmed` failed
once in CI on a **documentation-only** commit, then passed twice on the same tree.
`TODO.md` records two intermittent LAN tests. A **third** —
`cmd/hexarena-tui`'s `TestTheCountdownReachesTheScreenOverASocket` — had been seen
on 2026-09-04 and written to `.claude/agent-memory/coder/project_third_lan_flake.md`
**only**, so the fourth one arrived looking like a second.

⚠️ **The two stores answer different questions.** Agent memory answers *"what did
I learn"*; `TODO.md` answers *"what is open"*. A flake is open work: it has a
subject area, it will recur, and the next person to meet it needs the earlier
sightings in the same place they look for everything else.

**Also worth keeping from this one:** the failing line was the test's **own
vacuity guard** — *"no allowance is armed … so this test cannot tell a clock that
survived from one that was never there"*. It refused to claim anything rather than
claim something false, which is why the failure is legible at all. ⚠️ If the cause
turns out to be the test's own race, the fix is **not** to drop that guard.

**How to apply.**

- An intermittent failure gets a code the first time it is seen, with the exact
  failing line, the run URL, and how many targeted re-runs passed
  (this one: **125/125** locally — 5 plain, 20 and 100 at `GOMAXPROCS=1`).
- A memory note may hold the reasoning; the entry holds the fact that it is open.
- ⚠️ Never conclude "flake" from a red on a commit that changed behaviour. The
  argument here rests on the commit being markdown and comment text only.

Xem thêm [[todo-items-are-addressed-by-code]], [[a-heading-held-by-prose-decays]].
