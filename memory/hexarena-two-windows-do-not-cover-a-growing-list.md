---
name: hexarena-two-windows-do-not-cover-a-growing-list
description: hexarena — a TUI list test that renders top and bottom covers the whole book only while the book is shorter than two windows, which nothing states and nothing checks
metadata:
  type: feedback
---

`TestTheSkillListNamesSkillsInVietnamese` has now broken **three times** for the
same reason, and each fix bought less than it looked like:

1. It read **one** render. The listing is a window around the cursor, so the
   named skills fell off the day another character's kit shipped.
2. Widened to **top and bottom**. Held for a while.
3. Broke again the day the bench grew by **one** skill — because two windows only
   cover the whole book while the book is shorter than two windows, and nothing
   states that condition or checks it.

The third fix walks **every cursor position** and joins the renders. There is no
implicit bound left to outgrow; it is O(book) renders of a listing that is cheap
to draw, and the test stops reddening whenever somebody adds a skill.

⚠️ **The general shape: a fixture that samples a windowed view is bounded by
something the test does not say.** "Top and bottom" reads like completeness and is
really an assumption about length. Either walk the whole thing, or assert on
something the window cannot move — but do not add a third sample.

⚠️ **The same defect lives in the cast browser and is NOT the same fix.** There
the listing draws every row and the *detail pane* is what shrinks — see TODO.md
§ "The cast listing draws every row". A test cannot walk its way out of that one;
the screen has to bound the listing the way `screen.skillsRoom` already bounds
the skill one.
