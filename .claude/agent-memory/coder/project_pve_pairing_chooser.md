---
name: pve-pairing-chooser
description: hexarena-tui's PvE pairing — step 1 (the chooser, screenPairing in pairing.go) is done; the two premises the brief carried that measured FALSE (m.taking is not on the wire, and the shipped s01..s06 are ALREADY offered), and why `f` still opens the battle directly
metadata:
  type: project
---

**Step 1, done** (worktree `hexarena-pve`, branch `feat/pve-picks-both-sides`,
based on `eb8ef09`): `cmd/hexarena-tui` grew `screenPairing` — two cursors on
the model, `taking` (↑/↓) and `against` (←/→), `updatePairing`/`viewPairing` in
`pairing.go` beside the `pairing()` they are read into. The menu's PvE entry
opens the chooser; `enter` on it opens the battle. The away side used to be
*the next row on the file, wrapping*.

⚠️ **`m.taking` is NOT on the network path, and `pairing.go` said it was.** Its
closing section claimed `landSquad`/`m.taking` "is now also what fills
`wire.Hello.Squad` when a room is joined". `model.dialling` reads
`joinScreen.Chosen()` — the join screen's own chooser, which has to exist
because of its **bring none** position for a drafting room. The comment is
corrected in the file, and `TestThePairingChooserDoesNotReachThePvPSquad` holds
it two ways: by behaviour, and by an AST walk saying `taking`/`against` are read
in `pairing.go` and `subject.go` only.

⚠️ **The six shipped sides are ALREADY offered, so "step 2" may be a null.**
The brief said `.Squads()` is called nowhere in this client. It is —
`draw.SquadsScreen.Refresh` is `forge.SquadsOffered(c.Lib.Squads(), c.Player)`
— and `forge.LoadEmbedded().Squads()` answers **6: s01…s06**. Measured with a
throwaway `_test.go` in the package (deleted after). The whole suite is blind to
it because `scratchDataFrom` **deletes `squads.json`** from every scratch data
directory, so no fixture anywhere has a shipped side on it.
→ [[fixture-decides-what-is-visible]].

**Why the chooser is not shared with `cmd/hexforge-tui`'s `fightScreen`:** that
screen *measures* a pairing (seeds, rate, record, a control), this one settles
who turns up. What was copied is its one structural decision — the cursors are
**indices into the catalogue, not ids**.

⚠️ **The catalogue's `f` still opens the battle directly, and that is the way
back's doing rather than taste.** Routing `draw.Fight` at the chooser makes the
chain catalogue → chooser → battle → description: **three** pushes against
`raisedFrom`/`raisedOver`'s two, so the catalogue is silently lost. The client's
own comment already predicts the day a real `[]screen` is wanted; the test sites
are cheap (3 field reads, the rest of `raisedFrom` hits are a same-named test
helper), so it is a small follow-up rather than a rewrite. Until then `against`
is a **standing answer**: `f` names home and respects the away row the reader
last chose.

**Left:** step 2 (the shipped sides — re-scope it against the finding above) and
step 3 (the wording: `GameMenuBattleDetail` still says *"with the first side on
the list"* / *"với đội đầu danh sách"*, which is now false in two ways — the
entry opens a chooser rather than a battle, and both sides are choices).

See [[a-new-default-can-move-every-golden]] for the one trick that kept this a
pure golden insertion.
