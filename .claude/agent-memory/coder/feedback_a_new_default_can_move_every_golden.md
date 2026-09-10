---
name: a-new-default-can-move-every-golden
description: A new cursor that feeds an existing render has an initial value, and the obvious one (nought) is a silent change to every golden downstream — pick the value that reproduces today's answer and the diff stays a pure insertion
metadata:
  type: feedback
---

When a new field starts feeding a render something that used to be **derived**,
its initial value is a behaviour change wearing the clothes of a new feature.
Choose the one that reproduces the old answer.

**Why:** in `cmd/hexarena-tui` the away side of a hot-seat battle was
`saved[(home+1)%len]` — derived — and became `saved[against]`, a cursor. Nought
is the obvious start for a cursor and would have made every unchosen pairing a
side against a **copy of itself**, which is what all eight battle entries in
`testdata/screens.golden` are drawn from: two rosters, two boards, two order
lines, at two sizes in two languages. That diff would have been dozens of moved
lines to read past, none of them the change under review, and the brief's rule
here is *if an existing golden line moves, stop and say which and why*.
`awayOpensOn = 1` — the second row, clamped where it is read — is the same
answer the derivation gave for a reader on the first row, so **328 lines added,
0 removed** and the only banners in the diff are the new screen's.

**How to apply:** before writing the initial value, ask what the code computed
in its place yesterday and evaluate it for the fixture the goldens are drawn
over. If the two differ, either match them or say in the report which entries
move and why — do not discover it in the `make golden` diff. The same question
answers a second one for free: what a reader who has chosen nothing sees.

⚠️ The corollary is that the new default is then **unmeasured by any golden** —
it is byte-identical to what was there. What measures it is a hand-written test
naming the row (`TestTheOpponentIsTheRowChosenRatherThanTheNextOneOnTheFile`),
built on a catalogue of **three**, because two rows make the old rule and the
new one agree.

Related: [[two-screen-goldens]], [[a-golden-can-be-red-before-you-touch-it]],
[[pve-pairing-chooser]].
