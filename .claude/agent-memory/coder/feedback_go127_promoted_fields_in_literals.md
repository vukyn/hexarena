---
name: go127-promoted-fields-in-literals
description: Go 1.27 ACCEPTS promoted fields in a composite literal, which changes what embedding costs — and it also removes the compile error a keyed parse-shape pair was claimed to give
metadata:
  type: feedback
---

**Go 1.27 accepts a promoted (embedded) field as a key in a composite literal.**
`Outer{S: "x", A: "y"}` compiles where `A` belongs to an embedded `Inner`.
Verified against the toolchain (`go1.27.1`) with a throwaway program rather than
assumed, because it was false in every earlier Go.

**Why:** it decides whether "declare the shared fields once and embed" is cheap
or expensive. In hexarena's `internal/wire` the draft's two message bodies —
`Decide` (no seat) and `DraftEntry` (seat) — both embed one `DraftDecision`, so
the six decision fields are declared **once**; every existing
`Entry{Seat: …, Step: …, Character: …}` literal in `internal/draft` kept
compiling untouched. On an older toolchain each of those literals would have
needed the embedded field spelled out, and the churn would have made the
duplicate-struct-plus-test shape look cheaper than it is.

**How to apply:** when a shared field list wants declaring once, reach for an
anonymous embed — `encoding/json` inlines it too, so the JSON stays flat and a
golden shows one struct. Do check the toolchain (`go.mod` directive + `go
version`) before relying on the literal form; hexarena is on 1.27, other
platform repos are on 1.25/1.26 and there the flat literal will not compile.

⚠️ **Related and measured: the "compile error" claim about a parse-shape pair is
FALSE.** hexarena's `CLAUDE.md` says of `Skill.MarshalJSON`/`skillFile` that *"a
field added to one struct is a compile error in the other until it is added there
too, which is the point"*. `skill.Skill.file()` builds a **keyed** literal, so a
field added to either side is a compile error **nowhere**; what actually catches
it is `TestTheShippedSkillBookSurvivesBeingWritten`, a round trip over the real
data. An *unkeyed* literal would give the compile error — that is the shape the
claim describes and the code does not use. Do not cite that paragraph as
precedent for "two structs are safe because the compiler holds them in step".
See [[feedback_fixture_golden_cannot_see_producer]].
