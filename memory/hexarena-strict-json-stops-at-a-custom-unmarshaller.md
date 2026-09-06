---
name: hexarena-strict-json-stops-at-a-custom-unmarshaller
description: hexarena — DisallowUnknownFields is inherited by nested structs but NOT past a type with its own UnmarshalJSON, so a strict book is still blind inside modifier.Modifier
metadata:
  type: reference
---

`statuses.json` used to accept any field at all and keep the ones it knew: a
status authored with `"name"` and `"flavour"` beside its numbers — the shape a
skill and a trait are written in — parsed clean, both fields went nowhere, and
the id reached the battle log bare. The fix is one flag:

```go
reader := json.NewDecoder(bytes.NewReader(raw))
reader.DisallowUnknownFields()
```

**The flag is inherited by nested structs, and it stops at a custom
unmarshaller.** `modifier.Modifier` declares `UnmarshalJSON`, which calls
`json.Unmarshal` on its own `modifierFile` with a fresh, lenient decoder — so
`{"target":"attack","mode":"add","amount":100,"amout":100}` inside a status is
still silently accepted, in every book that carries a modifier. Strictness at the
outer level says nothing about the inner one, and nothing in the type system or
the test output points at the boundary.

**How to apply.** When making a book strict, list the types under it that own an
`UnmarshalJSON` — those are the holes, and each needs its own decoder changed.
And check the *other* books at the same time: refusing an unknown field on one
book while its two neighbours still drop them replaces the asymmetry that was
closed with a new one, which is what the author hit here (see [[hexarena-a-new-clause-needs-every-site-that-evaluates-it]] for the same shape one level up).

⚠️ A stricter decoder is also the one change that can refuse the data it was
written for. The shipped file has to be parsed by a test that already exists
(`TestShippedStatusBook`) before the flag is trusted, because the failure would
otherwise be at startup rather than in a unit test of the parser.
