---
name: matching-a-wordings-opening-clause-can-measure-nothing
description: internal/i18n's house idiom — assert on the clause ahead of a wording's first %s so a reworded translation stays honest — is blind whenever the words that tell two keys apart sit AFTER a blank; the wrong-key mutation stayed green in both languages
metadata:
  node_type: memory
  type: reference
---

**`strings.SplitN(text, "%", 2)[0]` is the house way to assert on a wording in `internal/i18n`, and it can measure nothing.** It exists for a good reason — matching the clause a wording *opens* with, rather than a copied literal, keeps a test honest when a translator rewords the sentence tomorrow — and `TestAGatedTraitIsNotDescribedAsAlways` and `TestBothEndsOfAGateAreWordedInBothLanguages` both read it that way. But it only discriminates when the words that tell two keys apart come **before the first blank**.

⚠️ **Measured, ENG-006 step 2.** The trait-level gate pair is
`"In force only at or below %s health."` / `"In force only at or above %s health."` — the comparison is ahead of the only blank, so the openings differ and reading them is decisive. The per-**grant** pair puts the status first:
`"Carries %s at or below %s health."` / `"Carries %s at or above %s health."` — and both openings are the single word `Carries` (`Mang` in Vietnamese). Deleting the `AtTop()` branch in `describe.go`, so every gated grant was worded as the *bottom* of the health bar, left the new test **green in both languages**: the opening matched, the figure was still right, and the sentence still read as a sentence. Only the comparison was inverted, which is the whole of what a reader is deciding on.

**The fix is to match the fragment that discriminates**, not the one the idiom reaches for: split on `%s` and take the text *between* the two blanks (`" at or above "` / `" khi còn >="`), assert the right one is present **and the wrong one absent**, and assert first that the two fragments differ at all — otherwise the assertion itself is vacuous and nothing says so.

**How to apply.** Before reusing the opening-clause idiom, look at where the two wordings actually diverge. A pair whose blanks come first is the common case as soon as a sentence names its subject before its condition — and every new blurb pair beside an existing one is worth this check, because the neighbouring test being decisive says nothing about yours.

⚠️ The general shape: **an assertion inherited from a neighbouring test is not inherited coverage.** The two wordings differ, the two tests read the same way, and one of them is blind. It was only caught by running the wrong-key mutation; nothing about the test's shape looked wrong.

Related: [[fixture-hidden-branch]], [[hexarena-descriptions-are-derived]], [[hexarena-tui-i18n]], [[mutate-the-producer-not-just-the-logic]], [[assert-on-the-clause-not-the-whole-page]].
