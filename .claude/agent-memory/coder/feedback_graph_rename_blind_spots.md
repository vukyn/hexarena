---
name: graph-rename-blind-spots
description: refactor_tool cannot rename a Go const in an iota block; test_gaps and callers_of read different edges and disagree; and an UNTRACKED file — or a whole new package, full_rebuild included — is not in the graph at all
metadata:
  type: feedback
---

The platform rule is "cross-file renames go through `refactor_tool(mode="rename")`,
never a manual multi-file rename". Two places the graph cannot hold up its end, so
reach for the fallback without burning a round trip discovering it again:

- **A Go constant declared inside a `const (...)` iota block has no graph node.**
  `refactor_tool` answers `status: not_found` for it. `internal/i18n/keys.go` is
  entirely that shape, so every `i18n.Key` rename is in this category. Fallback:
  grep every site first, then a scripted exact replace that **asserts a count of
  1 per file** before writing — that assertion is what makes it as safe as the
  graph would have been.
- **`detect_changes_tool`'s "Untested: …" list is a `tests_for`-edge heuristic
  keyed on naming, not on call edges.** Every *unexported* Go function reads as
  untested however hard the suite drives it, and even an exported one reads as
  untested when the test calls it as a method on a value. Confirm with
  `query_graph_tool(callers_of, <qualified_name>)` and look for a `kind: "Test"`
  caller before treating a flagged symbol as a real gap.

- **The graph indexes git-TRACKED files only, so a brand-new test file is
  invisible to it.** Measured: `build_or_update_graph_tool(base="HEAD")` then
  `detect_changes_tool(base="HEAD")` on a change that added three exported
  methods and a new `*_test.go` covering all three reported *4 test gaps* naming
  every one of them — and `query_graph_tool(children_of, <the new test file>)`
  came back `not_found`, because the file was untracked. This is the sharpest
  form of the bullet above: the coverage is not merely unlinked, the tests do
  not exist as far as the graph is concerned. It cannot be fixed before the
  committer runs, so the honest move is to say so in the report.
  - ⚠️ **A whole new untracked PACKAGE is likewise invisible, and
    `full_rebuild=true` does NOT help.** Measured on `internal/room` (11 files,
    ~3,100 lines): the full rebuild parsed **308 files both before and after**
    adding it, and `semantic_search_nodes_tool` for a symbol in it answered
    `search_mode: "none"`, 0 results. So for a PR whose whole subject is a new
    package there is no graph self-review available at all — do the callers/tests
    pass by hand and say in the report that the graph could not see it.
  - ⚠️ **`callers_of` on a method whose name collides with a stdlib interface
    method is UNUSABLE, and it answers `status: ok` while being wrong.** Measured
    on `internal/draft`: `callers_of` for the fully qualified
    `internal/draft/draft.go::Draft.Done` reported **15 callers** and named
    `cmd/hexarena-tui/session.go::choose` and a `sender_test.go` among the first
    five — neither of which imports `internal/draft` at all; they call
    `context.Context.Done` and `sync.WaitGroup.Done`. `Draft.Picked` likewise
    named three `cmd/hexforge-tui/model.go` functions. Grep found the real answer:
    6 call sites, all inside the package. So for `Done`, `Close`, `String`,
    `Len`, `Error`, `Slots` and friends, **the qualified name does not narrow the
    query** — go straight to `grep -rn "\.Name()" --include="*.go"` plus a check
    of who imports the package, and treat a nonzero result count as noise until
    each file is confirmed to import it.
  - ⚠️ **`test_gaps` and `callers_of` read DIFFERENT edges, so they disagree.**
    Measured in the same run: `detect_changes_tool` listed two new exported
    methods as test gaps while `query_graph_tool(callers_of, …)` returned a
    `kind: "Test"` caller for each. `callers_of` is the one to believe; the gap
    list is a `TESTS`-edge heuristic and a call from a Test does not create one.

**Why:** none of the three is a finding — the first reads as "the
symbol does not exist", the second as "you forgot a test", and acting on either
wastes a cycle or adds a redundant test.

**How to apply:** when a rename target is a const/enum member, skip
`refactor_tool` and go straight to the counted scripted rename. When
`detect_changes_tool` flags unexported symbols as untested, verify with
`callers_of` and, if the coverage is really behaviour-level, say so in the report
instead of chasing it — mutation-testing the behaviour is the honest substitute.
When the flagged symbols are new API, check whether the covering test file is
untracked before reading the gap count as anything at all.
