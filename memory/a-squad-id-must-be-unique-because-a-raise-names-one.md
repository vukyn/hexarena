---
name: a-squad-id-must-be-unique-because-a-raise-names-one
description: cmd/hexarena-tui's landSquad turns a screen.Subject's squad id back into a row by taking the FIRST match, so two squads under one id is a player pointing at the second and fielding the first — which is why forge.SquadsOffered merges the player's file over the game's rather than concatenating
metadata:
  node_type: memory
  type: reference
---

**A squad id has to be unique across everything offered, because a raise names
an id and the client turns it back into a row.** `internal/screen`'s squad
catalogue raises `draw.Subject{Kind: SquadSubject, ID: …}` about whatever row is
under the cursor, and `cmd/hexarena-tui/subject.go`'s `landSquad` walks
`m.squads.Saved` for the **first** entry with that id and records its index in
`m.taking`. `pairing()` then opens the battle on that index. So two rows sharing
an id is not cosmetic: a reader points at the second and fields the first, with
the name of the one they chose still on the screen.

**Why it matters:** it is the constraint that decided the id-collision rule for
a player's own squad file. The game ships `s01`–`s04` in
`internal/seed/data/squads.json`, and the obvious way to start a file of your
own is to copy that one — so collisions are the common case rather than the edge
one. `forge.SquadsOffered` therefore **lays the player's list over the game's**:
a shared id replaces the game's row in place, new ids are appended, and the
answer holds each id once. Concatenating was measured and produces exactly the
failure above — `[phe-0 phe-1 phe-0]`, the third row drawing the player's name
and the first row's units taking the field.

**How to apply:** anything that adds a second source of squads goes through
`SquadsOffered` rather than appending to the list. And the collision cannot be
resolved silently: the row that won says whose it is (`i18n.SquadMine`, drawn on
the catalogue's row and in the join screen's chooser), because a substitution a
player cannot see is the same defect as the ambiguous id wearing better clothes.
