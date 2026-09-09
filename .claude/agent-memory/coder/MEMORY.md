# Memory Index

## How to work with this user
- [Vietnamese UI copy style](feedback_vietnamese_ui_copy.md) — plain spoken wording in both languages; improve on the agreed glossary and report departures
- [Terminal draws some Vietnamese glyphs double-width](feedback_terminal_vietnamese_glyph_width.md) — apparent column drift is their font, not the code; never work around it

## Tooling
- [Go 1.27 takes promoted fields in a literal](feedback_go127_promoted_fields_in_literals.md) — embedding got cheap; ⚠️ the skillFile "compile error" claim is FALSE (keyed literal)
- [An identifier regex hits field KEYS](feedback_identifier_regex_hits_field_keys.md) — and the declaration, and doc-comment grammar; snapshot the PACKAGE not the plan
- [Graph rename/coverage blind spots](feedback_graph_rename_blind_spots.md) — no iota consts; test_gaps disagrees with callers_of; an untracked file or whole new package is never indexed
- [i18n key names churn the gofmt diff](feedback_i18n_key_gofmt_churn.md) — a long key or a mid-literal blank line reflows ~150 lines; appending at the end is free; width is unmeasured until drawn
- [Proving a TUI refactor is neutral](feedback_screen_neutrality_capture.md) — screens.golden holds the bytes now; a capture needs a RELATIVE data dir and a data-free commit pair
- [A fixture golden cannot see its producer](feedback_fixture_golden_cannot_see_producer.md) — messages.golden pins the FORMAT; the room dropping a field left internal/wire green
- [Three screens.golden, not one](feedback_two_screen_goldens.md) — screen=the drawing, each client=its framing; blind spots are the cast under the cursor AND the palette (all run NO_COLOR)
- [A stripped name hides a clip](feedback_name_strip_hides_a_clip.md) — the width sweep measures the line WITHOUT its names; a row carrying a value overflows green — read the golden for `…`
- [The fixture decides what a suite can see](feedback_fixture_decides_what_is_visible.md) — hand-set flag, mirror fixture, unobservable capacity, sweep at the fork-less row, shipped data already sorted
- [Measure WHICH guard masks](feedback_measure_which_guard_masks.md) — the blamed clamp was innocent; saturation kills monotonicity-in-power past a subtraction
- [A two-point reading names no distance](feedback_a_two_point_reading_names_no_distance.md) — "30×" was /4 and /64; every rung says /5–/6, and past a CLIFF a correct half-sized term reads as no term
- [A scripted revert can restore the wrong line](feedback_scripted_revert_wrong_occurrence.md) — green tests do not prove an undo; read git diff on the file
- [A fixture-cast edit costs goldens](feedback_fixture_cast_edit_costs_goldens.md) — one skill into a fixture kit moved 656 golden lines; build the carrier in the test (twinOf/forkedTwin)
- [A refusal can be right for the wrong reason](feedback_a_refusal_can_be_right_for_the_wrong_reason.md) — separate the verdict from its evidence; re-run a TODO's own example first, it may not reproduce
- [Rebasing onto a moved origin/main](feedback_rebase_onto_a_moved_main.md) — patch-not-stash, leave the golden out; main moves MID-task too, so pin HEAD either side of a golden reading
- [Measure the term before optimising it](feedback_measure_the_term_before_optimising_it.md) — SCR-011 blamed a 17 MB copy; 23ms vs 22ms on APFS, and 301 copies not 155 call sites
- [Measure the thing a bound bounds](feedback_measure_the_thing_a_bound_bounds.md) — my 1 MiB read limit bounded 2.9 KB; hold both ends. -race's first catch is a TEST's teardown assumption
- [A fixture the code can reorder](feedback_a_fixture_the_code_can_reorder.md) — an in-place sort also sorts the EXPECTATION; 0 tests failed
- [Re-entrant RLock in a Read callback](feedback_reentrant_rlock_inside_a_read_callback.md) — 1 run in 10; -race clean; hoist the seat out of the lock
- [A state the reading can never hold](feedback_a_state_the_reading_can_never_hold.md) — my own send moves NO reading; derive reachability from the producer, and a display memo is not a gate
- [A mutation upstream already neutralised](feedback_a_mutation_upstream_already_neutralised.md) — the room's cursor was ALREADY at the end; Deliver's 2nd guard refused anyway; count the roles a table sweeps
- [A well-formed measurement can measure nothing](feedback_a_well_formed_measurement_can_measure_nothing.md) — an RWMutex deadlock test needs a WRITER; a digest can be stable and always unequal
- [pty smoke test for hexarena-tui](feedback_pty_smoke_test_for_hexarena_tui.md) — it refuses a pipe; pty.fork + TIOCSWINSZ, two of them plus the host play a whole match
- [bubbles paste + nil commands](feedback_bubbles_paste_and_nil_commands.md) — textinput.Paste's msg is UNEXPORTED; a nil cmd is not "refused"; pinning a sanitiser ≠ pinning its caller
- [Unstable values in a record](feedback_unstable_values_in_a_record.md) — buildString() is "devel" under go test so hardcoding it is unmeasurable; the digest stays real; renormalize BEFORE editing
- [Measuring an installed binary](feedback_measuring_an_installed_binary.md) — a file GOPROXY prices the go-install path with no network; the module cache SHADOWS GOPROXY and re-installs the old code
- [Normalised upstream cannot discriminate](feedback_normalised_upstream_cannot_discriminate.md) — Build.Stage is resolved by the parser (28/28) and every form is reached at LevelCap, so both guards count everything
- [A two-element map range is a coin flip](feedback_two_element_map_range_is_a_coin_flip.md) — 17/20 not 20/20; read one state 64× · and label a release nothing can observe (0/10) as such
- [A mutation must hit the arm you claim](feedback_a_mutation_must_hit_the_arm_you_claim.md) — a blunt one died on table row 1; a full room hid a stolen seat; decoy a method to test a walk
- [A read after the exchange misses the last body](feedback_a_read_after_the_exchange_misses_the_last_body.md) — the room retires with it (62/63); the answer to the input must carry it
- [Three guards a neighbour already covered](feedback_three_guards_a_neighbour_already_covered.md) — one bound stated twice, a branch room.Left tolerates, a prompt action a departure repeats
- [The fixture that makes it measurable has its own price](feedback_the_fixture_that_makes_it_measurable_has_its_own_price.md) — s06 bought 0‰→2‰ of area and cost its own side 253‰; balance.md had already priced that board at 213‰
- [A null needs TWO controls](feedback_a_null_needs_two_controls.md) — an exact-null arm (arc_up, battle-for-battle) AND a payload calibration arm (fury ×2, −27‰); without the second, "no effect" = "blind"
- [A green-expected mutation proves nothing alone](feedback_a_green_expected_mutation_proves_nothing_by_itself.md) — M3 moved no test and no golden; probe the parsed field or "green" is indistinguishable from "never landed"
- [A hand-built arm misses the REPORTING branch](feedback_a_hand_built_arm_can_miss_the_reporting_branch.md) — it exercises the helper it calls; all 10 log lines came from :264 until a data mutation reached :256
- [One mutation cannot score a whole suite](feedback_one_mutation_cannot_score_a_whole_suite.md) — one PER decision; 8 of 13 red, the other 5 guarded the quantifier/validation; mutate the ONE shared home
- [The fixture reserves the id](feedback_the_fixture_reserves_the_id.md) — testfixture APPENDS 5 archetypes; ⚠️ their gloss already existing reads as "step done"
- [Measure in the slot it competes for](feedback_measure_in_the_slot_it_competes_for.md) — flex slot priced the composition (381‰) not the unit (491‰); a rate pinned at 0 both arms measures nothing
- [A mono-element mirror cannot resolve](feedback_a_mono_element_mirror_cannot_resolve.md) — all-Endless reads 0‰ not "red"; kit the probe OFF the element it is counted for (affinity is what counts)
- [A pre-reg floor is the point when it FAILS](feedback_a_prereg_floor_is_the_point_when_it_fails.md) — +48‰ vs +50‰: ship the null, do not add a board; the vacuous-rung mutation is what makes a null honest

## Ongoing work
- [The internal/screen extraction](project_screen_extraction.md) — DONE 1…6c: cmd/hexarena-tui is the second client; Context.Authoring gates the 3 authoring screens; pairing.go is the PvP seam
- [The ban/pick draft machine](project_draft_state_machine.md) — 2a…5b done (5c + host flag left); ⚠️ NOTHING tells the host the room is full; internal/screen declares its OWN DraftLive
- [A third LAN flake](project_third_lan_flake.md) — TestTheCountdownReachesTheScreenOverASocket raced once in make check; not in TODO.md, not reproducible, not yours

## Moved in from the platform-level store (2026-09-05)

⚠️ These were written while the session's working directory was the platform
root, so they landed in `<root>/.claude/agent-memory/` where this repo's own
agent could not see them — same knowledge, same agent, different cwd. They live
here now, and new ones belong here.

- [hexarena art picker](project_hexarena_art_picker.md) — hexforge-tui art field = chooser over forge.ArtFiles (2026-08-25); bubbles SetValue cursor trap, measure vs minWidth floor
- [hexarena cast gate](project_hexarena_cast_gate.md) — `gates` PR1 2026-09-04; 3 traps that make a test measure nothing + bench-row vs golden conflict
- [hexarena forge + TUI](project_hexarena_forge_tui.md) — 1st third-party dep (bubbletea) confined to cmd/hexforge-tui by an import-graph test; bubbletea pty startup queries hang headless tests
- [hexarena onboarding fixes](project_hexarena_onboarding_fixes.md) — LICENSE/.claude ignore/replay notice/go 1.27.0/Makefile 2026-08-24; govulncheck must be rebuilt after a Go bump
- [hexarena refusal wordings](project_hexarena_refusal_wordings.md) — Block wordings + OptionRefusal + gated opening; ⚠️ distinctness-only sweep measured nothing
- [Stage the bytes, not the directory](feedback_stage_the_bytes_not_the_directory.md) — a per-process fixture in memory has no cleanup owner and no cross-run staleness; ⚠️ a share chain kills a TempDir-skipping guard
- [Two test binaries beat stashing](feedback_two_test_binaries_beat_stashing.md) — `go test -c` both sides, interleave, `=== RUN` counts free; ⚠️ run from the package dir
- [A silent slot is what an under-price looks like](feedback_a_silent_slot_is_what_an_underprice_looks_like.md) — count CASTS to refuse a repricing; 76% control vs rapid_spin's 0.5%; ⚠️ actor id is side-prefixed
- [A golden can be red before you touch it](feedback_a_golden_can_be_red_before_you_touch_it.md) — 24 lines were the PREVIOUS commit's debt; prove it with `git archive HEAD`, never a shared stash
- [An uncommitted harness leaves only a shape](feedback_an_uncommitted_harness_leaves_only_a_shape.md) — 400/400/0 never reproduced; 3 fixtures gave 462/464/410 and the same cliff. Run the CONTROL
- [A spar golden is fought from the cast](feedback_a_spar_golden_is_fought_from_the_cast.md) — "no golden moves" reasoned over roster.json; naruto's duel moved 58→60. Prove causation by pinning the line back
