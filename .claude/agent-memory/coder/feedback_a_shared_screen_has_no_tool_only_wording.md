---
name: a-shared-screen-has-no-tool-only-wording
description: Since the internal/screen extraction there is no such thing as an "authoring tool's screen" — grep the i18n KEY to find every site of a wording, not the screens a spec names; artLine was a fourth site the spec called the tool's own
metadata:
  type: feedback
---

**In hexarena, "that wording belongs to the authoring tool" is no longer a fact
anybody can state from the file it lives in.** `internal/screen` holds thirteen
screens and **both** clients draw them, so a sentence written for an author is
drawn at a player unless something in the render asks.

**Why:** the art step's spec named three sites and warned that `i18n.ArtMissing`
"is shared with the authoring tool's browser and check screens, where MISSING is
the right word". The browser is not the authoring tool's — `internal/screen/browse.go`
is one screen and `cmd/hexarena-tui` draws it — so `artLine` was drawing
`assets/x.svg  MISSING` in the bad style at a player, on the detail pane they
reach **one keystroke before** the preview the spec was about. The check screen
half of the warning was correct; the browser half was not, and the difference is
invisible from the key's own doc comment.

Found by grepping the **key**, not by reading the spec's list:

```
grep -rn "i18n.ArtMissing" --include="*.go" internal cmd | grep -v _test
```

Two production sites, `preview.go:92` and `browse.go:471`, and the second was
the one nobody had counted. `cmd/hexforge*/check.go` also uses it and really is
tool-only.

**How to apply:** before changing what a wording means in one state, list its
call sites from the constant and ask of **each** whether both clients can reach
it. `cmd/hexforge-tui`-only is now a small set: the two clients' own `main.go`,
`model.go`, `check.go` and the lobby. Anything under `internal/screen` is
shared by default, and a "this is the author's screen" claim there needs
`Context.Authoring` behind it or it is not a claim at all.

⚠️ The corollary for tests: the fixture in every golden and every sweep of both
clients loads a **data directory**, so a wording that only differs when there is
none is outside all three `screens.golden` files. Measured — see
[[two-screen-goldens]].

Related: [[project_screen_extraction]], [[project_client_runs_without_a_checkout]].
