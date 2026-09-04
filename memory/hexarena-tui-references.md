---
name: hexarena-tui-references
description: "hexarena hexforge-tui reference screens — statuses, traits, elements, species and the ring-drawn affinity chart; what each one may and may not list, and the layout rules they all share"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T08:14:17.932Z
---

The read-only lookup screens on the hexforge-tui menu (PRs #100, #103, #105, #107, #108). Every listing in the tool prints an element id or a species id somewhere, and before these the only way to find out what one meant was to open the JSON.

**The screens.** `hiệu ứng` (statuses, grouped by category) · `nội tại` (traits, with a carriers column) · `hệ` (the eleven elements, each under its gloss) · `chủng loài` (species) · and `đồ thị tương khắc`, raised with **`g`** from the elements listing rather than from the menu, because at 80×24 the window will not hold both.

**The chart is drawn as rings, not a matrix** — the form the chart is *declared* in, and the same one `internal/core/element`'s package doc uses. `element.Chart` retains the declaration (`Cycles`, `MutualPairs`, `Inert`) beside the resolved edges and hands it out as **copies**; the matrix alone cannot answer "what ring is fire in". `Inert()` reads the edges, not the file's `inert` list.

Each ring is an **ASCII loop with a return line** (the user chose this over a polygon, 2026-08-28), and a ring too wide for one row turns back on itself so the return leg carries the rest — read left to right along the top, then **right to left** along the bottom, which is the way its arrows point:

```
,--> water --> metal --> grass --> wind ----,
|                                           |
'--- electric <-- ground <-- ice <-- fire <-'
```

⚠️ **The user's standing rule: add an element → the graph must still be right.** The drawing is **generated** from the chart, so a new element cannot make it *wrong* — it redraws itself, which is the only reason a picture is allowed here at all (a hand-drawn figure would be the stale-figure failure the derived descriptions exist to avoid). What a new element *can* do is make it **not fit**, and then `ringLines` silently falls back to a plain chain and the screen stops being a loop. `TestEveryRingClosesAtTheFloor` is what holds that shut: at the floor width every declared ring must draw as a loop, every line the same width, each member exactly once. `TestTheDrawnLoopIsTheDeclaredRing` walks the drawing back and compares it to the declared chain — including the two corner edges no text-adjacency test can see. So: after adding an element, run the tui tests and *look at the screen*; do not hand-edit the art.

⚠️ **ASCII only, never arrows or box-drawing characters.** `→`, `⇄`, `┌` are East-Asian ambiguous width, and a diagram is a picture made of columns — it does not survive being one cell out the way a sentence does. See [[terminal-ambiguous-width-glyphs]].

**What a reference may not list: anything that grows with another book.** The species screen once had a "who is one" column (the twin of the trait listing's carriers column) and a "skills kept for it" line, and both were removed in #107: a species is what a character *is*, so a cast of thirty puts thirty ids across five rows. The trait carriers column is fine because a trait is carried by the handful that learn it. What survives is the half that cannot grow — a kind **nobody** is still says so in words. `TestTheSpeciesScreenListsNoCastAndNoKit` writes the decision down, because the argument for adding a column back is a good one every time it is made.

**Layout rules these screens established:**
- A table cell clips against **`m.usableWidth()`**, not `minWidth`. The floor is what the program promises to draw in, not a ceiling on what it may spend — clipping data at 80 on a 160-column terminal cut "để dành cho loài dr…". Prose still wraps at the floor.
- A pane holding authored prose must **measure** its worst case, not reserve one line. A species note wraps, the frame cuts from the bottom, and the overrun eats the derived line underneath.
- A **data name** (species name, trait name) is Vietnamese whoever asks, so an English screen drops the whole column and keeps bare ids — unlike a compiled gloss, which is empty in English by construction. A **note** has no id to fall back to, so it is printed in both languages.
- Colour is decoration and never information: the chart names every element and draws every relation in text, so `NO_COLOR` loses nothing. Per-element colours live in `elementColours`; `neutral` is faint rather than coloured.
- One figure, one spelling: rates go through `i18n.Share` (67%), not `forge.Percent` (66.7%).

**Tests that were only found by the test suite, not by review:** the width test measures every screen in `everyScreen`, so a new screen must be added there; `carriesFreeText` recognises prose by its opening, which works for *clipped* text and not for *wrapped* text (`partOfFreeText` handles the second).

Related: [[hexarena-tui-i18n]], [[hexarena-core-design]], [[hexarena-descriptions-are-derived]].
