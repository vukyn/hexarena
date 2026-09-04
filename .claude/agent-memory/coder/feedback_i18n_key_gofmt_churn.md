---
name: i18n-key-gofmt-churn
description: Adding an i18n key to hexarena's english.go/vietnamese.go reflows ~150 unrelated lines unless the key name fits the run's column — and a new key's WIDTH is measured by nothing until a screen draws it
metadata:
  type: feedback
---

In `hexarena/internal/i18n/{english,vietnamese}.go` the catalogs are one huge
aligned composite literal, so **gofmt decides the value column from the widest
key in the contiguous run**. Two ways to churn the diff, both of which I hit:

- **A blank line around the new entry** splits the run in two and gofmt
  realigns both halves — ~35 unrelated lines per file.
- **A key name longer than the run's current widest** pushes the whole column
  out — ~150 unrelated lines per file.

**Why:** the review cost is real (the wording change is 1 line; the diff was
300), and it is invisible until `gofmt -w` runs, so it arrives *after* the edit
looks right. Measure before naming: `SquadCharacterHeldBack` blew the column
that `SquadFieldCharacter` (20 chars) set; `SquadHeldBack` fitted and the diff
went to `+1`.

**How to apply:** before adding a key, look at the existing indentation in the
block you are inserting into and count — if `YourKey:` is wider than the column
already there, shorten the name rather than accepting the reflow. Put the entry
**inline in the run**, never in its own paragraph; the explanatory comment goes
on the `Key` constant in `keys.go`, which is a plain list and has no alignment
to disturb. If a long name is genuinely unavoidable, say in the report that the
reflow is mechanical so a reviewer does not go looking for wording changes.
See [[graph-rename-blind-spots]] for the other place a scripted, counted edit
beats the obvious tool here.

**⚠️ The blank-line rule has one exemption, measured 2026-09-03: a new block
appended at the very END of the literal costs nothing.** Thirteen wire-wording
keys went in behind a blank line and a comment paragraph and the diff was
`+20 / -0` in each catalog — because the split leaves the *earlier* run's widest
key untouched and the new run aligns only against itself. The churn arrives when
the blank line lands in the MIDDLE. So: append at the end, and only then is the
"own paragraph" shape free.

**⚠️ A new key's WIDTH is measured by nothing until a screen draws it.**
`TestEveryWordingFitsTheMinimumWidth` renders **screens** in both languages, not
the catalog, so a key no screen calls is outside its subject entirely; the only
catalog-wide sweep is `TestEveryWordingIsOneCellPerLetter`, which checks
cells==runes and bans combining marks but has no maximum. Do not conclude from
that a long wording is wrong: **five shipped entries already exceed the 119-cell
floor on one line** (measured — keys 297/299/407/408/411, e.g. `absorb`'s blurb
at 156 cells in en), because prose is *wrapped* at the floor when drawn and only
data cells are held to a single line. Write the sentence the wording needs; hard
`\n` breaks are for the too-small screen, which is drawn in a window already
known to be narrower than the floor.
