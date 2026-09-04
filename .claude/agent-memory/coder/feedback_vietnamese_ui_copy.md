---
name: vietnamese-ui-copy-style
description: How the user wants Vietnamese (and English) UI copy written in this repo — plain, spoken, not literal translation
metadata:
  type: feedback
---

Vietnamese UI wording must be `dễ hiểu, tránh máy móc, tối nghĩa, viết tắt quá
nhiều` — plain and comfortable to read, not literal-translation stiff, not
cryptic, not full of abbreviations. The same applies to the English beside it:
short enough for the layout, but never abbreviated where the whole word fits
("biography", not "bio").

**Why:** the user stated this as the acceptance criterion for the hexarena
authoring client's localisation, not as a nicety — "a sentence that is
technically correct but reads like machine output is a defect here". They also
supplied a vocabulary table but explicitly expected it to be improved on where a
term reads badly in context, with the departures reported.

**How to apply:** write what a Vietnamese-speaking player would actually say and
reorder the sentence if that reads better; do not translate the English string
word for word. Keep ids, flag names and short stat labels untranslated — an
author has to type them. When departing from an agreed term, say so and why in
the report. Developer loanwords that people here actually use ("build lại") beat
stiff calques ("biên dịch lại").
