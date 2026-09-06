package screen

import (
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The draft screen, tested without a protocol anywhere near it
//
// Every fixture below builds a `DraftLive` **by hand**, which is the point of
// the type rather than a convenience: the screen cannot see `internal/socket`,
// so a test here cannot drive it through a mirror and does not want to. What
// holds the mapping between a reading and one of these is
// `cmd/hexarena-tui/draft_test.go`, in the package that owns both vocabularies.
//
// The split is the one the extraction settled: a test asserting *what a screen
// draws* or *how its cursor moves* lives with the screen, and a test asserting
// where a keystroke lands or what a client does with an Action lives in the
// client.

// TestNoScreenImportsTheProtocol is the import rule the whole where-it-lives
// decision rests on, and until this test it was held by **prose alone**.
//
// ⚠️ `internal/screen`'s package comment, `socket.DraftSight`'s comment and
// `lobby.go`'s header each state it from a different side — a screen two clients
// share may not know a socket exists — and nothing anywhere checked. The clock
// allowlist over in `internal/socket` bans `time` here and says in as many words
// that this is the package the countdown could most easily have broken; this is
// the same shape for the protocol, which is the thing a draft screen is under
// constant pressure to reach for.
//
// It scans **every** `.go` file including the tests, deliberately. A test file
// naming a `wire.DraftStep` to prove a mapping is a perfectly reasonable thing
// to want and it is not wanted *here*: the walk that holds the two vocabularies
// together belongs in the client, which is the one thing that knows both, and a
// test-only import would be the first step back towards a screen that takes a
// `DraftSight`.
func TestNoScreenImportsTheProtocol(t *testing.T) {
	banned := map[string]string{
		"github.com/vukyn/hexarena/internal/wire": "the protocol may not follow a screen " +
			"into the package two clients share: a wire.Code or a wire.Seat on a field here " +
			"is why i18n.Lang.Refusal and i18n.Lang.Seat take a NAME",
		"github.com/vukyn/hexarena/internal/draft": "internal/draft imports internal/wire, " +
			"so this would drag the protocol in behind it — which is the whole reason " +
			"DraftLive is declared here and mapped by the client",
		"github.com/vukyn/hexarena/internal/socket": "a screen may not know a socket exists; " +
			"a reading crosses as a value the client built inside Mirror.Read",
		"github.com/vukyn/hexarena/internal/room": "a room is a state machine on the other " +
			"side of the transport, and a screen is two layers out from it",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("%s: unquote %s: %v", name, imported.Path.Value, err)
			}
			if because, refused := banned[path]; refused {
				t.Errorf("%s imports %s: %s", name, path, because)
			}
		}
	}
	// A walk that read no files would pass whatever this package imported, which
	// is the failure the wording walker's own counter records.
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	t.Logf("scanned %d source files for a protocol import", scanned)
}

// TestEveryDraftStepIsNamedAndCounted is the count the client's two walks are
// held against, checked from this side so that a value added to the enum without
// a name is caught here rather than as a diagnostic reading `draftstep(6)`.
func TestEveryDraftStepIsNamedAndCounted(t *testing.T) {
	if got, want := len(draftStepNames), DraftStepCount; got != want {
		t.Fatalf("the names table holds %d entries against a count of %d", got, want)
	}
	seen := map[string]DraftStep{}
	for value := range DraftStepCount {
		step := DraftStep(value)
		name := step.String()
		if name == "" {
			t.Errorf("DraftStep %d has no name", value)
			continue
		}
		if first, twice := seen[name]; twice {
			t.Errorf("DraftStep %d and %d are both named %q", first, value, name)
		}
		seen[name] = step
	}
	if got := DraftStep(DraftStepCount).String(); !strings.Contains(got, "draftstep(") {
		t.Errorf("a step past the count reads %q, want the fallback that names the number", got)
	}
}

// TestADraftWithNothingRecordedSaysTheBanMayBeThrownAway is the state 5a
// measured and the one this screen exists as much for as for the pool.
//
// ⚠️ **A drafting room tells the host nothing when it fills.** The room refuses a
// decision until both seats are taken and sends no message when the second one
// is, so the host's first ban goes out, is refused, and is thrown away — and a
// player who cannot see that has no way to know it. The notice is therefore not
// decoration, and this is the test that keeps it drawn.
func TestADraftWithNothingRecordedSaysTheBanMayBeThrownAway(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	for _, lang := range i18n.Langs() {
		sized := c
		sized.Lang = lang
		screen := aDraftDue(t, lib, DraftStepBan, true)
		if !screen.Live.Waiting() {
			t.Fatal("the fixture draft has decisions recorded, so it is not the waiting state")
		}
		drawn, _ := screen.View(sized)
		if want := sized.Text(i18n.DraftNotBegun); !strings.Contains(drawn, firstLineOf(want)) {
			t.Errorf("a draft with nothing recorded says nothing about it in %s:\n%s", lang, drawn)
		}
		// And the discrimination: a draft that HAS a decision recorded must not
		// draw it, or the line is a permanent decoration rather than a state.
		begun := screen
		begun.Live.Recorded = 1
		if begun.Live.Waiting() {
			t.Fatal("a draft with a decision recorded still reads as waiting")
		}
		drawnBegun, _ := begun.View(sized)
		if strings.Contains(drawnBegun, firstLineOf(sized.Text(i18n.DraftNotBegun))) {
			t.Errorf("a draft with a decision recorded still warns about the empty room in %s",
				lang)
		}
	}
}

// TestADecisionWithOneCandidateSaysThereIsNoChoice is the arithmetic finding
// drawn.
//
// ⚠️ **Nothing here writes the candidate count down and nothing may.** With every
// ban spent the final pick sees `draft.Slack + 1` candidates, and that expression
// has answered 1, 2 and 4 within two days as the pool moved from sixteen
// characters to nineteen. So the rule is the list in hand — one entry is not a
// choice — and the *derivation* is measured in `cmd/hexarena-tui`, which is the
// package that may name `draft.Slack` at all.
func TestADecisionWithOneCandidateSaysThereIsNoChoice(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepPick, true)
	if len(screen.Live.Candidates) < 2 {
		t.Fatalf("the fixture pick has %d candidates, so it cannot show the difference",
			len(screen.Live.Candidates))
	}
	drawn, _ := screen.View(c)
	only := screen.Live.Candidates[0]
	if strings.Contains(drawn, c.Text(i18n.DraftOnlyOne, only)) {
		t.Error("a pick with a real choice says there is nothing to choose")
	}
	// ⚠️ **Two candidates is the neighbour that matters, and the whole pool is
	// not.** Measured under mutation: writing the shipped figure down as
	// `len(Candidates) > 4` leaves this test green, because its "a real choice"
	// fixture is the *whole* pool — twenty-odd rows, nowhere near the boundary.
	// Any `<= n` form is caught by asking about the row next to one.
	pair := screen
	pair.Live.Candidates = screen.Live.Candidates[:2:2]
	if _, one := pair.Live.OnlyOne(); one {
		t.Error("a decision between two characters is reported as no choice at all")
	}
	pinched := screen
	pinched.Live.Candidates = []string{only}
	if got, one := pinched.Live.OnlyOne(); !one || got != only {
		t.Fatalf("a one-candidate decision answers (%q, %v)", got, one)
	}
	drawnPinched, _ := pinched.View(c)
	if !strings.Contains(drawnPinched, c.Text(i18n.DraftOnlyOne, only)) {
		t.Errorf("a pick with one candidate presents a list of one as a choice:\n%s", drawnPinched)
	}
}

// TestThePoolIsDrawnInTheOrderItWasHandedIn is step 1's decision held one layer
// out.
//
// ⚠️ **The shipped cast happens to be in id order**, so a sorted screen and an
// unsorted one draw the same thing over it and a test taken against the shipped
// books would measure nothing — which is the trap `draft.NewPool`'s own test
// names. So this hands in a pool whose order is **not** its id order and reads
// the rows back.
func TestThePoolIsDrawnInTheOrderItWasHandedIn(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepBan, true)
	pool := screen.Live.Pool
	if len(pool) < 3 {
		t.Fatalf("the fixture pool holds %d characters", len(pool))
	}
	// Reversed, which is the one order a sort cannot agree with unless the book
	// was already descending.
	slices.Reverse(pool)
	if slices.IsSortedFunc(pool, func(a, b cast.Character) int {
		return strings.Compare(a.ID, b.ID)
	}) {
		t.Fatal("the reversed pool is still in id order, so this measures nothing")
	}
	screen.Live.Pool = pool
	// ⚠️ **The expected order is COPIED before the screen is drawn, and that is
	// the whole net.** Measured: a `slices.SortFunc` planted inside the drawing
	// sorts the slice **in place** — the same backing array this test handed
	// in — so a version that read `pool` afterwards compared the sorted rows
	// against its own sorted expectation and passed. A fixture that the code
	// under test can reorder is a fixture that agrees with whatever it did.
	wanted := make([]string, 0, len(pool))
	for _, character := range pool {
		wanted = append(wanted, character.ID)
	}
	sized := c
	sized.Height = 60 // tall enough that the whole pool is drawn rather than a window of it
	drawn, _ := screen.View(sized)
	at := -1
	for _, id := range wanted {
		found := strings.Index(drawn, id)
		if found < 0 {
			t.Fatalf("the pool row for %s is not on the screen:\n%s", id, drawn)
		}
		if found < at {
			t.Fatalf("%s is drawn above the character handed in before it, so the pool was "+
				"sorted rather than drawn in the order it arrived", id)
		}
		at = found
	}
}

// TestTheMarksSayWhichSideTookEachCharacter is the column the whole screen is
// for, and the side is the half a mirror cannot derive.
//
// ⚠️ Which characters are gone is already the set difference between the pool and
// the candidates; **which side took each one** is not on any reading of the
// state, because bans alternate and a skip moves the count without naming
// anybody. → `draft.Draft.Bans`.
func TestTheMarksSayWhichSideTookEachCharacter(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepPick, true)
	pool := screen.Live.Pool
	if len(pool) < 5 {
		t.Fatalf("the fixture pool holds %d characters", len(pool))
	}
	mine, theirs := screen.Live.mine(), screen.Live.theirs()
	screen.Live.Bans[mine] = []string{pool[0].ID}
	screen.Live.Bans[theirs] = []string{pool[1].ID}
	screen.Live.Picks[mine] = []DraftPick{{Character: pool[2].ID, Stage: "x", Skills: []string{"a"}}}
	screen.Live.Picks[theirs] = []DraftPick{{Character: pool[3].ID, Stage: "y", Skills: []string{"b"}}}
	for id, want := range map[string]i18n.Key{
		pool[0].ID: i18n.DraftStateYouBanned,
		pool[1].ID: i18n.DraftStateTheyBanned,
		pool[2].ID: i18n.DraftStateYouPicked,
		pool[3].ID: i18n.DraftStateTheyPicked,
		pool[4].ID: i18n.DraftStateOpen,
	} {
		if got := screen.stateOf(id); got != want {
			t.Errorf("%s is marked %q, want %q", id, c.Text(got), c.Text(want))
		}
	}
	// And the count of what is left is derived from the marks rather than from the
	// candidate list, which is empty during a loadout while the pool is not.
	if got, want := screen.Left(), len(pool)-4; got != want {
		t.Errorf("the pool reads %d left over %d characters with four gone, want %d",
			got, len(pool), want)
	}
	loadout := screen
	loadout.Live.Step = DraftStepLoadout
	loadout.Live.Candidates = nil
	if got, want := loadout.Left(), len(pool)-4; got != want {
		t.Errorf("during a loadout the pool reads %d left, want %d: a count taken off the "+
			"candidate list would say nought while the pool was nearly full", got, want)
	}
}

// TestAReadingWithNoSeatMarksNothingAsYours is the safe half of the You field.
//
// A forgotten seat that read as *this* reader's would tell a player they had
// banned characters they never touched; reading as the opponent's is the harmless
// direction, and the field's own comment says so.
func TestAReadingWithNoSeatMarksNothingAsYours(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepBan, true)
	pool := screen.Live.Pool
	screen.Live.Bans[0] = []string{pool[0].ID}
	screen.Live.Picks[1] = []DraftPick{{Character: pool[1].ID, Skills: []string{"a"}}}
	screen.Live.You = ""
	for _, id := range []string{pool[0].ID, pool[1].ID} {
		if got := screen.stateOf(id); got == i18n.DraftStateYouBanned ||
			got == i18n.DraftStateYouPicked {
			t.Errorf("with no seat on the reading, %s is marked %q as this reader's",
				id, c.Text(got))
		}
	}
}

// TestOnlyTheSideBeingAskedIsOfferedADecision is the reading footer's promise,
// and it is a whole match rather than a nicety: a player who believes they have
// banned somebody finds out one allowance later.
func TestOnlyTheSideBeingAskedIsOfferedADecision(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	for _, step := range []DraftStep{DraftStepBan, DraftStepPick} {
		theirs := aDraftDue(t, lib, step, false)
		if _, footer := theirs.View(c); footer != c.Text(i18n.DraftReadingFooter) {
			t.Errorf("a %s on the other side draws the footer %q, which names a key this "+
				"reading ignores", step, footer)
		}
		for _, name := range []string{"enter", "s"} {
			_, result := theirs.Update(c, press(t, name))
			if result.Decided {
				t.Errorf("%q took a %s that is not this reader's to take", name, step)
			}
		}
		// And the cursor still walks, which is the other half of that footer.
		moved, _ := theirs.Update(c, press(t, "down"))
		if moved.Cursor == theirs.Cursor {
			t.Errorf("the cursor does not move while the other side is deciding a %s", step)
		}
	}
}

// TestABanAndAPickAndASkipAreThreeDecisions drives the three keystrokes the pool
// answers and reads back what each one would send.
//
// ⚠️ **A skip names nobody and that absence IS the decision** — a ban slot spent
// on nobody, with no third state and no flag beside it — so the assertion is on
// the empty Character rather than on a marker.
func TestABanAndAPickAndASkipAreThreeDecisions(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	ban := aDraftDue(t, lib, DraftStepBan, true)
	chosen, have := ban.Chosen()
	if !have {
		t.Fatal("the fixture ban has nothing under the cursor")
	}
	_, taken := ban.Update(c, press(t, "enter"))
	if !taken.Decided || taken.Decision.Step != DraftStepBan ||
		taken.Decision.Character != chosen {
		t.Errorf("enter on a ban took %+v, want a ban of %q", taken.Decision, chosen)
	}
	_, skipped := ban.Update(c, press(t, "s"))
	if !skipped.Decided || skipped.Decision.Step != DraftStepBan ||
		skipped.Decision.Character != "" {
		t.Errorf("s on a ban took %+v, want a ban naming nobody", skipped.Decision)
	}
	pick := aDraftDue(t, lib, DraftStepPick, true)
	first, _ := pick.Chosen()
	_, picked := pick.Update(c, press(t, "enter"))
	if !picked.Decided || picked.Decision.Step != DraftStepPick ||
		picked.Decision.Character != first {
		t.Errorf("enter on a pick took %+v, want a pick of %q", picked.Decision, first)
	}
	// ⚠️ **A pick has no skip and must not grow one.** A side that skipped a pick
	// would have no squad to field, which is what the draft's own timeout refuses
	// to invent on anybody's behalf.
	_, notSkipped := pick.Update(c, press(t, "s"))
	if notSkipped.Decided {
		t.Errorf("s on a pick took %+v; a pick cannot be skipped", notSkipped.Decision)
	}
}

// TestTheCursorFollowsTheCharacterRatherThanTheRow is what stops a reader's
// cursor jumping every time the opponent decides anything.
//
// The candidate list shrinks under a reader — the other side bans while they are
// walking it — and every row below the one that went moves up by one, so a kept
// index lands on a neighbour.
func TestTheCursorFollowsTheCharacterRatherThanTheRow(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen := aDraftDue(t, lib, DraftStepBan, true)
	for range 3 {
		screen, _ = screen.Update(c, press(t, "down"))
	}
	on, have := screen.Chosen()
	if !have || screen.Cursor != 3 {
		t.Fatalf("the cursor is at %d on %q after three presses", screen.Cursor, on)
	}
	// The opponent takes the character above the cursor, so every row below it
	// moves up by one.
	shorter := screen.Live
	taken := shorter.Candidates[0]
	shorter.Candidates = slices.Clone(shorter.Candidates[1:])
	shorter.Bans[shorter.theirs()] = []string{taken}
	shorter.Recorded++
	after := screen.Attach(c, shorter)
	if got, _ := after.Chosen(); got != on {
		t.Errorf("the cursor moved to %q when %q left the list; it was on %q", got, taken, on)
	}
	if after.Cursor != 2 {
		t.Errorf("the cursor index is %d, want 2 — the same character one row up", after.Cursor)
	}
	// And when the character under the cursor is the one that went, the row
	// number is kept, which lands on whatever is now next.
	gone := after.Live
	next := gone.Candidates[after.Cursor+1]
	gone.Candidates = slices.Delete(slices.Clone(gone.Candidates), after.Cursor, after.Cursor+1)
	gone.Recorded++
	landed := after.Attach(c, gone)
	if got, _ := landed.Chosen(); got != next {
		t.Errorf("with the character under the cursor taken, the cursor landed on %q, want "+
			"the row that moved up into it, %q", got, next)
	}
}

// TestTheLoadoutIsThrownAwayWhenThePickChanges is the one thing Attach resets.
//
// A form and a kit belong to the character they were chosen for, so carrying them
// into the next pick would offer skills the new character has never learned — and
// cast.ChooseLoadout would refuse them one decision later, with the reader unable
// to see why.
func TestTheLoadoutIsThrownAwayWhenThePickChanges(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen, first := aDraftLoadout(t, c, lib)
	if len(screen.Skills) == 0 {
		t.Fatal("the fixture loadout chose no skills, so there is nothing to throw away")
	}
	kept := screen.Attach(c, screen.Live)
	if len(kept.Skills) != len(screen.Skills) || kept.Stage != screen.Stage {
		t.Error("the same reading threw the loadout away; Attach runs on every redraw")
	}
	next := screen.Live
	for _, character := range screen.Live.Pool {
		if character.ID != first {
			next.Subject = character.ID
			break
		}
	}
	next.Recorded++
	moved := screen.Attach(c, next)
	if len(moved.Skills) != 0 || len(moved.Passives) != 0 || moved.Stage != "" ||
		moved.Err != nil || moved.Field != 0 {
		t.Errorf("a loadout for %q survived the pick moving to %q: %+v",
			first, next.Subject, moved)
	}
}

// TestTheLoadoutRefusesAKitTheRuleRefuses is `cast.ChooseLoadout` asked as the
// kit is chosen, which is what makes the refusal drawn here the answer the room
// would give.
//
// ⚠️ **And it blocks the send**, which is not politeness: a refused draft decision
// leaves the decision open, so a client that sent one would re-send it — and the
// retry-on-refusal loop is an unbounded hot loop for a decision that is genuinely
// illegal rather than merely early.
func TestTheLoadoutRefusesAKitTheRuleRefuses(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen, subject := aDraftLoadout(t, c, lib)
	if screen.Err != nil {
		t.Fatalf("the fixture loadout is already refused: %v", screen.Err)
	}
	screen.Field = DraftSend
	sent, result := screen.Update(c, press(t, "enter"))
	if !result.Decided || result.Decision.Step != DraftStepLoadout {
		t.Fatalf("a legal loadout was not sent: %+v (%v)", result, sent.Err)
	}
	if len(result.Decision.Skills) != len(screen.Skills) {
		t.Errorf("the sent loadout carries %d skills against the %d chosen",
			len(result.Decision.Skills), len(screen.Skills))
	}
	// A skill the subject has not learned, which is exactly what the rule refuses.
	illegal := screen
	illegal.Skills = append(slices.Clone(screen.Skills[:1]), "nothing.at.all")
	refused, _ := illegal.Picked(c, DraftPickKit, PickAnswer{Chosen: illegal.Skills})
	if refused.Err == nil {
		t.Fatalf("a kit naming a skill %s has never learned was accepted", subject)
	}
	refused.Field = DraftSend
	blocked, result := refused.Update(c, press(t, "enter"))
	if result.Decided {
		t.Errorf("a refused loadout was sent anyway: %+v", result.Decision)
	}
	if blocked.Err == nil {
		t.Error("the send cleared the refusal it was blocked by")
	}
	drawn, _ := blocked.View(c)
	if !strings.Contains(drawn, firstLineOf(c.Lang.Error(blocked.Err))) {
		t.Errorf("the refusal is not drawn:\n%s", drawn)
	}
}

// TestTheLoadoutsTwoListsAreTheMultiSelect is the reuse the step turns on:
// PickState is already "several out of a list, keeping the order", and the squad
// builder's two are the prior art.
func TestTheLoadoutsTwoListsAreTheMultiSelect(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen, subject := aDraftLoadout(t, c, lib)
	for _, one := range []struct {
		field DraftField
		into  DraftPickInto
		slots int
	}{
		{DraftKit, DraftPickKit, cast.SkillSlots},
		{DraftTrait, DraftPickTrait, cast.TraitSlots},
	} {
		open := screen
		open.Field = one.field
		next, result := open.Update(c, press(t, "enter"))
		if result.Action.Kind != Pick || result.Action.Picker == nil {
			t.Fatalf("enter on field %d asked for %v rather than a picker",
				one.field, result.Action.Kind)
		}
		picker := result.Action.Picker
		if picker.Into != one.into {
			t.Errorf("the picker for field %d lands in %v, want %v",
				one.field, picker.Into, one.into)
		}
		if picker.Slots != one.slots {
			t.Errorf("the picker for field %d holds %d slots, want %d",
				one.field, picker.Slots, one.slots)
		}
		if len(picker.Options) == 0 {
			t.Errorf("the picker for field %d offers nothing for %s", one.field, subject)
		}
		// Every row is something the subject has learned, so no row can carry a
		// refusal — which is why each list names its own hint.
		for _, option := range picker.Options {
			if option.Refusal != nil {
				t.Errorf("the learnset row %q carries a refusal: %v", option.ID, option.Refusal)
			}
		}
		if next.Field != one.field {
			t.Errorf("raising the list moved the cursor to field %d", next.Field)
		}
	}
}

// TestWalkingTheFormEmptiesTheKit is the squad builder's own answer, and the
// reason is the same: a skill belongs to the form it was learned as.
func TestWalkingTheFormEmptiesTheKit(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	screen, subject := aDraftForkedLoadout(t, c, lib)
	if len(screen.FormChoices()) < 3 {
		t.Fatalf("%s offers %d forms, so nothing here walks a fork",
			subject, len(screen.FormChoices()))
	}
	// ⚠️ **An unnamed form on a forking line has no answer, and the screen says so
	// BEFORE anything has gone wrong.** Both lists show only what every arm
	// learns, so a reader looking at a shortened learnset with nothing saying why
	// is the defect the squad builder shipped once; and the send is refused, which
	// is progression's own refusal passed through.
	drawn, _ := screen.View(c)
	if !strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
		t.Errorf("the form row on the forking line %s does not say no arm is named:\n%s",
			subject, drawn)
	}
	if !strings.Contains(drawn, firstLineOf(c.Text(i18n.DraftForkArms, ""))[:20]) {
		t.Errorf("the forking line %s names no arms to choose between:\n%s", subject, drawn)
	}
	unsent := screen
	unsent.Field = DraftSend
	blocked, result := unsent.Update(c, press(t, "enter"))
	if result.Decided {
		t.Errorf("a loadout with no arm named was sent: %+v", result.Decision)
	}
	if blocked.Err == nil {
		t.Errorf("the send on an unnamed fork was blocked by nothing")
	}
	named, _ := screen.Update(c, press(t, "right"))
	if named.Stage == "" {
		t.Fatal("right did not name a form")
	}
	if named.Err != nil {
		t.Errorf("a named arm of %s is still refused: %v", subject, named.Err)
	}
	picker := named.OpenSkills()
	if picker == nil || len(picker.Options) == 0 {
		t.Fatalf("the named arm of %s offers no skills", subject)
	}
	kitted, _ := named.Picked(c, DraftPickKit,
		PickAnswer{Chosen: []string{picker.Options[0].ID}})
	if len(kitted.Skills) != 1 {
		t.Fatalf("the arm's kit did not land: %+v", kitted.Skills)
	}
	// And walking on empties it, because the next arm's learnset is a different
	// list and a name carried over would be refused a decision later.
	walked, _ := kitted.Update(c, press(t, "right"))
	if walked.Stage == kitted.Stage {
		t.Fatal("right did not move off the arm, so nothing here walks the fork")
	}
	if len(walked.Skills) != 0 {
		t.Errorf("the kit chosen as %q survived the form moving to %q: %v",
			kitted.Stage, walked.Stage, walked.Skills)
	}
}

// TestADraftScreenWithNoReadingDrawsNothing is the zero value, which is what a
// client holds before it has joined anything and after it has left.
func TestADraftScreenWithNoReadingDrawsNothing(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	body, footer := NewDraftScreen().View(c)
	if body != "" {
		t.Errorf("a draft screen nothing has attached to drew a body:\n%s", body)
	}
	if footer != c.Text(i18n.DraftReadingFooter) {
		t.Errorf("the empty screen's footer is %q", footer)
	}
}

// TestTheDraftFitsTheFloorInEveryStateItDraws is the vertical half of the width
// rule, taken at the floor where the room helper bites.
//
// ⚠️ **The pool shrinks as the sides fill and that is the budget working**, not a
// clip: the two side blocks grow a row per pick and the listing takes what is
// left, with a floor of three. What must never happen is the footer leaving the
// screen, which is what the frame's own cut would do — so what is measured is
// that the body stays inside the rows a frame gives it.
func TestTheDraftFitsTheFloorInEveryStateItDraws(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	floor := atTheFloor(c)
	for name, screen := range everyDraftState(t, floor, lib) {
		body, _ := screen.View(floor)
		if got, room := len(drawnLines(body)), floor.Height-4; got > room {
			t.Errorf("the %s state draws %d body lines against the %d a frame leaves:\n%s",
				name, got, room, body)
		}
	}
}

// aDraftDue is a reading with one decision open, this reader's or the other
// side's, over the fixture cast.
//
// ⚠️ **The pool is the whole book and the held-back filter is NOT applied here.**
// The pool is a parameter of the draft — `draft.NewPool` is the single
// declaration of "the cast minus every character held back" and it lives in a
// package this one may not import — so what a fixture here can honestly hand over
// is a list, and what holds the *real* pool is `cmd/hexarena-tui`'s own test
// against that function. A mirror of the filter written here would be a second
// declaration that agreed with the first by having been copied from it.
func aDraftDue(t *testing.T, lib *forge.Library, step DraftStep, yours bool) DraftScreen {
	t.Helper()
	pool := lib.Characters().All()
	if len(pool) == 0 {
		t.Fatal("the fixture cast is empty, so there is no pool to draft from")
	}
	candidates := make([]string, 0, len(pool))
	for _, character := range pool {
		candidates = append(candidates, character.ID)
	}
	live := DraftLive{
		Pool:       pool,
		Candidates: candidates,
		Seats:      [2]string{fixtureHost, fixtureGuest},
		You:        fixtureHost,
		OnTurn:     fixtureHost,
		Step:       step,
		Yours:      yours,
		Units:      3,
		BanSlots:   2,
		Clock:      aCountdown(PlayClockYou),
	}
	if !yours {
		live.OnTurn = fixtureGuest
		live.Clock = aCountdown(PlayClockThem)
	}
	return NewDraftScreen().Attach(Context{}, live)
}

// The two seats, spelled as the protocol spells them and handed in like every
// other fixture value here. ⚠️ They are **strings** rather than a type, which is
// the whole of what this package knows about a seat: it words one through
// i18n.Lang.Seat and never branches on one.
const (
	fixtureHost  = "host"
	fixtureGuest = "guest"
)

// aDraftLoadout is the loadout due for a pick already taken, with a legal kit
// chosen through the real picker.
//
// It answers the subject as well as the screen, because every assertion about a
// loadout is about a particular character's learnset.
func aDraftLoadout(t *testing.T, c Context, lib *forge.Library) (DraftScreen, string) {
	t.Helper()
	screen := aDraftDue(t, lib, DraftStepLoadout, true)
	// ⚠️ **Both halves of a loadout, found rather than named.** A character with a
	// learnset and no traits is a perfectly ordinary character and it makes the
	// trait picker open on an empty list — which every assertion about that list
	// then passes on for the wrong reason. A helper that settled for the first
	// character with a skill picked exactly such a one out of the fixture cast.
	subject := ""
	for _, character := range screen.Live.Pool {
		skills := character.SkillsAt(progression.LevelCap, progression.Furthest)
		traits := character.PassivesAt(progression.LevelCap, progression.Furthest)
		if len(skills) >= cast.SkillSlots && len(traits) > 0 {
			subject = character.ID
			break
		}
	}
	if subject == "" {
		t.Fatalf("no character in the fixture cast knows %d skills and a trait at the cap, "+
			"so nothing here measures a whole loadout", cast.SkillSlots)
	}
	live := screen.Live
	live.Subject = subject
	live.Candidates = nil
	live.Picks[live.mine()] = []DraftPick{{Character: subject}}
	live.Recorded = 1
	screen = screen.Attach(c, live)
	if !screen.Choosing() {
		t.Fatal("the loadout fixture does not open the editor")
	}
	picker := screen.OpenSkills()
	if picker == nil || len(picker.Options) == 0 {
		t.Fatalf("%s offers no skills at the cap", subject)
	}
	chosen := make([]string, 0, cast.SkillSlots)
	for _, option := range picker.Options {
		if len(chosen) == cast.SkillSlots {
			break
		}
		chosen = append(chosen, option.ID)
	}
	screen, _ = screen.Picked(c, DraftPickKit, PickAnswer{Chosen: chosen})
	return screen, subject
}

// aDraftForkedLoadout is the loadout due for the one fixture character whose
// evolution line forks, which is the case every other entry here walks past: an
// unnamed form has no answer on such a line and progression refuses to pick one.
//
// ⚠️ **Found rather than named, and fatal when there is none.** A helper that
// quietly settled for a linear character would turn "the fixture changed" into "a
// test measures nothing" without a word.
func aDraftForkedLoadout(t *testing.T, c Context, lib *forge.Library) (DraftScreen, string) {
	t.Helper()
	screen := aDraftDue(t, lib, DraftStepLoadout, true)
	for _, character := range screen.Live.Pool {
		arms, err := character.FurthestAt(progression.LevelCap)
		if err != nil || len(arms) < 2 {
			continue
		}
		live := screen.Live
		live.Subject = character.ID
		live.Candidates = nil
		live.Picks[live.mine()] = []DraftPick{{Character: character.ID}}
		live.Recorded = 1
		return screen.Attach(c, live), character.ID
	}
	t.Fatal("no character in the fixture cast forks at the cap, so nothing here measures the " +
		"state progression refuses to resolve")
	return screen, ""
}

// firstLineOf is enough of a wrapped sentence to recognise it by after the screen
// has broken it at the floor.
func firstLineOf(text string) string { return WrapWords(text, MinWidth-3)[0] }

// everyDraftState is the states of this screen that draw a line no other state
// draws, which is what the golden registers and what the floor is measured over.
//
// ⚠️ **Every one of them asserts it drew the line it exists for**, which is the
// rule this package's golden is written under: a registered state that renders
// nothing passes every sweep over it, and this repository has shipped that
// fixture more than once.
func everyDraftState(t *testing.T, c Context, lib *forge.Library) map[string]DraftScreen {
	t.Helper()
	// The base ban, with a decision recorded so the waiting notice is a state of
	// its own rather than something every entry carries.
	begun := func(screen DraftScreen) DraftScreen {
		live := screen.Live
		live.Recorded = 2
		live.Bans[live.mine()] = []string{live.Pool[0].ID}
		live.Bans[live.theirs()] = []string{live.Pool[1].ID}
		live.Candidates = slices.Clone(live.Candidates[2:])
		return screen.Attach(c, live)
	}
	ban := begun(aDraftDue(t, lib, DraftStepBan, true))
	pick := begun(aDraftDue(t, lib, DraftStepPick, true))
	// A side with picks on it, which is the block the budget shrinks the listing
	// for. Their loadouts are in, so the row draws a form and a count.
	withPicks := pick
	{
		live := withPicks.Live
		mine, theirs := live.mine(), live.theirs()
		live.Picks[mine] = []DraftPick{
			{Character: live.Pool[2].ID, Stage: "one", Skills: []string{"a", "b"},
				Passives: []string{"c"}},
			{Character: live.Pool[3].ID},
		}
		live.Picks[theirs] = []DraftPick{
			{Character: live.Pool[4].ID, Stage: "two", Skills: []string{"d"}},
		}
		live.Recorded = 6
		withPicks = withPicks.Attach(c, live)
	}
	loadout, _ := aDraftLoadout(t, c, lib)
	forked, _ := aDraftForkedLoadout(t, c, lib)
	refused := loadout
	refused.Skills = append(slices.Clone(loadout.Skills), "nothing.at.all")
	refused, _ = refused.Picked(c, DraftPickKit, PickAnswer{Chosen: refused.Skills})
	if refused.Err == nil {
		t.Fatal("the refused-loadout state is not refused, so it records an ordinary editor")
	}
	only := pick
	only.Live.Candidates = []string{only.Live.Pool[len(only.Live.Pool)-1].ID}
	only.Cursor = 0
	// The two endings, neither of which draws a listing at all.
	arranging := ban
	{
		live := arranging.Live
		live.Step, live.Arranging, live.Yours, live.OnTurn = DraftStepArrange, true, true, ""
		live.Candidates = nil
		live.Recorded = 10
		arranging = arranging.Attach(c, live)
	}
	cancelled := ban
	{
		live := cancelled.Live
		live.Step, live.Cancelled, live.Yours, live.OnTurn = DraftStepNone, true, false, ""
		live.Candidates = nil
		live.Clock = PlayClock{}
		cancelled = cancelled.Attach(c, live)
	}
	waiting := aDraftDue(t, lib, DraftStepBan, true)
	// A refusal the room sent, which is the other half of the waiting state: a
	// ban that was thrown away leaves this and nothing else behind.
	throwaway := waiting
	throwaway.Live.Refusal = fixtureRefusal
	states := map[string]DraftScreen{
		"a draft":                    ban,
		"a draft waiting for a peer": waiting,
		"a draft ban thrown away":    throwaway,
		"a draft on the other side":  begun(aDraftDue(t, lib, DraftStepBan, false)),
		"a draft pick":               pick,
		"a draft part way through":   withPicks,
		"a draft with one candidate": only,
		"a draft loadout":            loadout,
		"a forked draft loadout":     forked,
		"a refused draft loadout":    refused,
		"a draft arranging":          arranging,
		"a cancelled draft":          cancelled,
	}
	// Each entry's own discrimination, so a state that has stopped reaching its
	// branch fails loudly instead of recording an ordinary draft twice.
	assert := func(name string, want string) {
		t.Helper()
		drawn, _ := states[name].View(c)
		if !strings.Contains(drawn, want) {
			t.Fatalf("the %q state draws no line saying so:\n%s", name, drawn)
		}
	}
	assert("a draft", c.Text(i18n.DraftYourBan))
	assert("a draft waiting for a peer", firstLineOf(c.Text(i18n.DraftNotBegun)))
	assert("a draft ban thrown away", c.Text(i18n.DraftRefused))
	assert("a draft on the other side", c.Text(i18n.DraftTheirBan, c.Lang.Seat(fixtureGuest)))
	assert("a draft pick", c.Text(i18n.DraftYourPick))
	assert("a draft part way through", c.Text(i18n.DraftLoadoutOpen))
	assert("a draft with one candidate",
		c.Text(i18n.DraftOnlyOne, only.Live.Candidates[0]))
	assert("a draft loadout", c.Text(i18n.DraftSendReady))
	assert("a forked draft loadout", c.Text(i18n.SquadForkUnnamed))
	assert("a refused draft loadout", firstLineOf(c.Lang.Error(refused.Err)))
	assert("a draft arranging", firstLineOf(c.Text(i18n.DraftArrangingNow)))
	assert("a cancelled draft", firstLineOf(c.Text(i18n.ClosedDraftExpired)))
	return states
}

// fixtureRefusal is the name of the code a decision taken too early comes back
// with, spelled as wire.Code spells it — a **name**, which is the whole of what
// this package knows about a refusal. → i18n.Lang.Refusal.
const fixtureRefusal = "not_your_turn"
