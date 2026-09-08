package main

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/room"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The two draft vocabularies, held together in the one package that knows both
//
// `internal/screen` declares `DraftLive` and `DraftDecision`; `internal/wire`
// declares `DraftStep` and `DraftDecision`; and neither package may see the
// other. The whole cost of that arrangement is that a step added to the protocol
// could reach no screen arm and nothing would say so — a keyed composite literal
// gives no compile error, which `CLAUDE.md` measured on `skillFile` and says in
// as many words is not a precedent this repository has.
//
// So the cost is paid **here**, by walks, and the walks are the reason option (i)
// was affordable at all. → `draft.go`.

// TestEveryDraftStepTheProtocolDeclaresReachesAScreenArm is the forward walk.
//
// ⚠️ It walks `wire.DraftSteps()` rather than the map, for the reason every other
// totality walk in these clients walks a count rather than a list: ranging over
// the map would ask it whether it holds what it holds.
func TestEveryDraftStepTheProtocolDeclaresReachesAScreenArm(t *testing.T) {
	for _, step := range wire.DraftSteps() {
		drawn, known := draftSteps[step]
		if !known {
			t.Errorf("the protocol's %s step reaches no arm of the draft screen, so a draft "+
				"in that state would draw nothing about it", step)
			continue
		}
		if drawn == draw.DraftStepNone {
			t.Errorf("the protocol's %s step maps to DraftStepNone, which is *nothing due* — "+
				"a decision that reads as no decision is the quietest failure there is", step)
		}
		if int(drawn) >= draw.DraftStepCount {
			t.Errorf("the protocol's %s step maps to %d, past the %d the screen declares",
				step, drawn, draw.DraftStepCount)
		}
	}
	if got, want := len(draftSteps), len(wire.DraftSteps()); got != want {
		t.Errorf("the map holds %d entries against the %d steps the protocol declares: an "+
			"entry for a step nothing sends is an arm nothing can reach", got, want)
	}
}

// TestEveryDraftStepThisScreenDrawsComesFromTheProtocol is the reverse walk, and
// it is the half a one-way check cannot make.
//
// A screen arm nothing on the wire produces is an arm that can never be drawn —
// the mirror image of a `draw.Target` with no entry in `raiseTargets`, which this
// repository has recorded five times in the other direction.
//
// ⚠️ **DraftStepNone is excused by name and by one**, because it is the reading a
// closed `Turn` falls into: nothing is due, no protocol step says so, and the
// empty `wire.DraftStep` that arrives with it lands there by construction.
func TestEveryDraftStepThisScreenDrawsComesFromTheProtocol(t *testing.T) {
	produced := map[draw.DraftStep]wire.DraftStep{}
	for _, step := range wire.DraftSteps() {
		produced[draftSteps[step]] = step
	}
	for value := range draw.DraftStepCount {
		drawn := draw.DraftStep(value)
		if drawn == draw.DraftStepNone {
			if from, reachable := produced[drawn]; reachable {
				t.Errorf("the protocol's %s step maps to *nothing due*", from)
			}
			continue
		}
		if _, reachable := produced[drawn]; !reachable {
			t.Errorf("the screen declares %s and nothing on the wire produces it, so that arm "+
				"can never be drawn", drawn)
		}
		if _, back := wireSteps[drawn]; !back {
			t.Errorf("the screen's %s step maps to no protocol step, so a decision taken in "+
				"that state could not be sent", drawn)
		}
	}
	if got, want := len(wireSteps), draw.DraftStepCount-1; got != want {
		t.Errorf("the outward map holds %d entries against the %d steps the screen declares "+
			"besides *nothing due*", got, want)
	}
}

// TestADecisionSurvivesTheRoundTripThroughBothVocabularies is the walks' effect
// half: a count proves a step is mapped, not that it is mapped **right**.
//
// ⚠️ **The empty Character on a ban is the payload here, not a gap in the
// fixture.** A skip names nobody and that absence *is* the decision, so a
// round trip that helpfully filled it in would turn a spent ban slot into a ban
// on whoever the field defaulted to.
func TestADecisionSurvivesTheRoundTripThroughBothVocabularies(t *testing.T) {
	for _, taken := range []draw.DraftDecision{
		{Step: draw.DraftStepBan, Character: "pokemon.mew"},
		{Step: draw.DraftStepBan},
		{Step: draw.DraftStepPick, Character: "pokemon.gastly"},
		{Step: draw.DraftStepLoadout, Stage: "Gengar",
			Skills: []string{"a", "b"}, Passives: []string{"c"}},
		{Step: draw.DraftStepLoadout},
	} {
		sent, known := draftDecisionOf(taken)
		if !known {
			t.Fatalf("%+v could not be sent at all", taken)
		}
		if got := draftSteps[sent.Step]; got != taken.Step {
			t.Errorf("%+v came back as the %s step", taken, got)
		}
		if sent.Character != taken.Character || sent.Stage != taken.Stage {
			t.Errorf("%+v travelled as character %q stage %q",
				taken, sent.Character, sent.Stage)
		}
		if !slices.Equal(sent.Skills, taken.Skills) ||
			!slices.Equal(sent.Passives, taken.Passives) {
			t.Errorf("%+v travelled with %v / %v", taken, sent.Skills, sent.Passives)
		}
	}
	// And a step the protocol does not have is refused rather than sent as
	// something else, which is what makes the walk's failure mode a decline.
	if _, known := draftDecisionOf(draw.DraftDecision{Step: draw.DraftStepNone}); known {
		t.Error("a decision with no step was sendable")
	}
}

// TestEveryDraftPickDestinationLandsSomewhere is the loadout's two lists held
// total, in the shape `TestEveryPickDestinationLandsSomewhere` already has one
// client over.
//
// A destination with no entry swallows a finished pick in silence: the list comes
// down, the reader believes they chose, and the field is unchanged.
func TestEveryDraftPickDestinationLandsSomewhere(t *testing.T) {
	for value := range int(draw.DraftPickIntoCount) {
		into := draw.DraftPickInto(value)
		_, landed := draftPickedInto[into]
		if into == draw.DraftPickNothing {
			// The zero is the destination for which swallowing an answer is the
			// definition rather than the defect.
			if landed {
				t.Error("the no-destination value has a landing, so a picker built by hand " +
					"would write somewhere")
			}
			continue
		}
		if !landed {
			t.Errorf("the draft pick destination %d lands nowhere, so a finished pick into "+
				"it is swallowed", value)
		}
	}
	if got, want := len(draftPickedInto), int(draw.DraftPickIntoCount)-1; got != want {
		t.Errorf("the dispatch holds %d entries against the %d destinations declared besides "+
			"the zero", got, want)
	}
}

// TestTheLastPickChoosesFromSlackPlusOne is the arithmetic behind the screen's
// one-candidate line, **derived** and never written down.
//
// ⚠️ **The figure has moved three times in two days.** `draft.Slack` is the
// expression and the final pick sees `slack + 1` candidates: that was **one**
// character when the pool held sixteen, and it is what it is now. So this test
// derives both ends — the pool through `draft.NewPool`, the count through
// `draft.Slack` — and asserts what the *screen* does with each answer rather than
// asserting the answer. A test naming 1, or 2, or 4 would have been wrong within
// a day of being written each time.
//
// What it holds is the pair: the shipped pool's last pick **is** a choice and the
// screen must not say otherwise, and a pool whose slack is nought leaves one
// candidate and the screen must say so.
func TestTheLastPickChoosesFromSlackPlusOne(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	pool := draft.NewPool(characters.All())
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		slack := draft.Slack(pool.Len(), format)
		if slack < 0 {
			t.Fatalf("the shipped pool of %d cannot seat a %s at all", pool.Len(), format)
		}
		candidates := slack + 1
		t.Logf("a %s out of %d characters leaves %d slack, so the last pick chooses from %d",
			format, pool.Len(), slack, candidates)
		screen := draw.NewDraftScreen().Attach(draw.Context{}, aDraftReading(pool, format, candidates))
		_, one := screen.Live.OnlyOne()
		if one != (candidates == 1) {
			t.Errorf("a %s's last pick has %d candidates and the screen calls it %v",
				format, candidates, map[bool]string{true: "no choice", false: "a choice"}[one])
		}
	}
	// And the other end of the pair, which the shipped pool does not reach: a pool
	// exactly big enough leaves nought slack, so the last pick is not a decision.
	tight := draft.Slack(2*draft.PicksPerSide(wire.Format3v3)+
		2*draft.BansPerSide(wire.Format3v3), wire.Format3v3)
	if tight != 0 {
		t.Fatalf("a pool sized to the format exactly leaves %d slack, so this derivation is "+
			"wrong rather than the screen", tight)
	}
	pinched := draw.NewDraftScreen().Attach(draw.Context{},
		aDraftReading(pool, wire.Format3v3, tight+1))
	if _, one := pinched.Live.OnlyOne(); !one {
		t.Error("a last pick with one candidate is not reported as no choice")
	}
}

// aDraftReading is a reading with a named number of candidates left, which is
// what the arithmetic above hands over.
func aDraftReading(pool draft.Pool, format wire.Format, candidates int) draw.DraftLive {
	all := pool.All()
	ids := make([]string, 0, candidates)
	for _, character := range all[:min(candidates, len(all))] {
		ids = append(ids, character.ID)
	}
	return draw.DraftLive{
		Pool:       all,
		Candidates: ids,
		Seats:      draftSeats(),
		You:        string(wire.SeatHost),
		OnTurn:     string(wire.SeatHost),
		Step:       draw.DraftStepPick,
		Yours:      true,
		Recorded:   4,
		Units:      draft.PicksPerSide(format),
		BanSlots:   draft.BansPerSide(format),
	}
}

// TestTheClientDrawsAForcedPickAsNoChoiceInBothLanguages is SCR-012's client
// half, and it is a pair rather than one claim.
//
// `internal/screen` already holds this pair over the **drawing**
// (TestADecisionWithOneCandidateSaysThereIsNoChoice). It cannot stand in for this
// one: the two draw through different paths — this client fills a `draw.DraftLive`
// through `draftLiveOf` and renders inside `frame`, where that package is handed a
// reading and answers with a body and a footer — so a line lost to this client's
// framing, or a mapping that stopped carrying the candidate list, leaves that test
// green. Before this one existed the wording had **nought hits** in this client's
// golden, in both languages.
//
// Both halves, because either alone is worth little: a screen that never draws
// the line and a screen that always draws it are both wrong, and the second reads
// as working.
//
// ⚠️ **The neighbour of one is what the negative half is drawn at.** Two
// candidates, not the whole pool — a fixture whose "real choice" is twelve rows is
// nowhere near the boundary, and `internal/screen` measured a `> 4` form of the
// rule surviving exactly that.
func TestTheClientDrawsAForcedPickAsNoChoiceInBothLanguages(t *testing.T) {
	picks := draft.PicksPerSide(wire.Format3v3)
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		forced := aForcedPickScreen(t, m)
		only := forced.draft.Live.Candidates[0]
		said := firstLine(forced.text(i18n.DraftOnlyOne, only))
		if !strings.Contains(drawnBody(forced), said) {
			t.Errorf("in %s the client draws a forced pick as an ordinary list to choose "+
				"from, saying nothing of %q:\n%s", lang, said, drawnBody(forced))
		}
		// And the other half: one decision earlier the same draft offers two, which
		// is a choice, and the line must not be there.
		pair := aPinchedPickScreen(t, m, 2*picks-2)
		if got := len(pair.draft.Live.Candidates); got != 2 {
			t.Fatalf("the neighbouring reading offers %d candidates rather than 2, so this "+
				"half is not measured at the boundary at all", got)
		}
		assertNamesNobodyAsTheOnlyOne(t, pair)
	}
}

// TestTheForcedPickEntryIsNotMovedByWhatTheCastShips is the fixture measured
// rather than argued for.
//
// ⚠️ **#328 is what this is written against.** A fixture that picked characters by
// **position** in the cast file moved 1,292 golden lines the day the cast was
// sorted, with no screen having changed — and a golden's whole value is that a
// line moving means a drawing moved. The forced-pick entry needs a pool of a
// particular *size*, which is the shape most likely to be reached for by slicing
// whatever is shipped, so the immunity is measured: the same entry is drawn twice,
// once over the shipped cast and once over a cast one character wider, and the two
// drawings have to be the same bytes.
//
// ⚠️ **The extra character goes in FRONT.** `cast.json` is held in id order, so a
// character shipping can land anywhere in the list — and a fixture slicing the
// front of the pool is only caught by an arrival that sorts ahead of what it was
// taking. Appending would leave that mutation green.
func TestTheForcedPickEntryIsNotMovedByWhatTheCastShips(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	shipped := shippedCast(t)
	if len(shipped) == 0 {
		t.Fatal("the embedded cast is empty, so nothing here is measured")
	}
	// A character the cast does not have, built by taking one it does and renaming
	// it: what the pool does with an arrival is all this measures, so the arrival
	// need only be a distinct id the draft would offer.
	arrival := shipped[0]
	arrival.ID = "arrival.before.everybody"
	arrival.Hidden = false
	if slices.ContainsFunc(shipped, func(character cast.Character) bool {
		return character.ID == arrival.ID
	}) {
		t.Fatalf("%s is already in the shipped cast, so widening it adds nothing", arrival.ID)
	}
	widened := append([]cast.Character{arrival}, shipped...)
	drawnOver := func(all []cast.Character) string {
		t.Helper()
		pool := pinchedDraftPoolFrom(t, all)
		if slices.ContainsFunc(pool, func(character cast.Character) bool {
			return character.ID == arrival.ID
		}) {
			t.Fatalf("the pinched pool took %s, so it is cut from whatever the cast happens "+
				"to hold rather than from the named list", arrival.ID)
		}
		return drawnBody(aDraftScreen(t, m, pool, func(live *draw.DraftLive) {
			live.Step, live.Yours, live.OnTurn = draw.DraftStepPick, true, live.Seats[1]
			live.You = live.OnTurn
			live.Candidates = []string{pool[len(pool)-1].ID}
		}))
	}
	before := drawnOver(shipped)
	after := drawnOver(widened)
	if before != after {
		t.Errorf("the forced-pick entry is drawn differently over a cast one character wider, "+
			"so its golden moves on a content commit:\n%s", firstDifference(before, after))
	}
	// And the count this walk owes: the pool it is drawn from is the size the
	// format spends, out of a cast that is bigger than that.
	pool := pinchedDraftPool(t)
	tight := 2*draft.PicksPerSide(wire.Format3v3) + 2*draft.BansPerSide(wire.Format3v3)
	if len(pool) != tight {
		t.Errorf("the pinched pool holds %d characters against the %d a 3v3 spends", len(pool),
			tight)
	}
	if len(shipped) <= tight {
		t.Fatalf("the shipped cast holds %d characters against the %d this pool needs, so a "+
			"wider cast is not what this test is telling apart", len(shipped), tight)
	}
	t.Logf("the pinched pool is %d of %d shipped characters, and %d of them are named in this "+
		"package", len(pool), len(shipped), len(namedDraftPoolIDs))
}

// TestAWatcherOfADraftingRoomIsHandedADraftThatNeverAdvances is the claim
// SCR-012 was written on, measured.
//
// The item said *"a spectator draws the same draft"*, and half of that is true: a
// watching client is welcomed into a drafting room with `Welcome.Drafts` set, so
// it builds its own `*draft.Draft`, `Sight.Draft.Mirrored` is true and the model
// really does put the draft screen in front of it. What it can never draw is a
// draft anybody has *decided* anything in. `internal/room/draft.go` writes neither
// `wire.Drafted` nor the pick clock's `wire.Closed` to `Room.watched` — its own
// comment says so, and calls watching a draft step 7 — so a watcher's record is
// empty until the draft closes and then holds the whole battle from its
// `wire.Start`.
//
// So the state this item is about is **unreachable for a spectator today**: the
// candidate list a watcher is handed is the opening one, whatever the two players
// have taken out of the pool, and it is nineteen-odd rows rather than one. That is
// what makes this item two goldens and not three.
//
// ⚠️ **The positive control is the battle.** A test asserting only that nothing
// arrived passes on a watcher that joined nothing at all, so this waits for the
// `wire.Start` the two players' finished draft produces — the watcher demonstrably
// received the match — and only then reads what its draft was told.
func TestAWatcherOfADraftingRoomIsHandedADraftThatNeverAdvances(t *testing.T) {
	held, _ := openADraftingWatchableRoom(t)
	watcher := aWatchingDrafter(t, held)
	theDraftingOpponent(t, held)
	theDraftingOpponent(t, held)
	deadline := time.Now().Add(theWholeMatch)
	for watcher.Mirror().Battle() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if watcher.Mirror().Battle() == nil {
		t.Fatalf("the two players never drafted a battle inside %v, so this measures nothing "+
			"about what a watcher was told", theWholeMatch)
	}
	if !watcher.Mirror().Watching() {
		t.Fatal("this client took a seat rather than watching, so it is a third player and " +
			"not a spectator")
	}
	view := watcher.Mirror().Draft()
	if !view.Mirrored {
		t.Fatal("a watcher of a drafting room holds no draft at all, so it cannot even reach " +
			"the draft screen — which is a different finding from the one this test is about")
	}
	if view.Replayed != 0 {
		t.Errorf("the watcher's draft took %d recorded decisions, so a spectator now sees the "+
			"ban and pick and this test is the stale half rather than TODO.md", view.Replayed)
	}
	// And what that costs the item: the reading a watcher hands the screen is the
	// opening one, so the forced last pick cannot be drawn from a watcher's seat
	// however the draft went.
	pool := draft.NewPool(held.deps.Characters.All()).All()
	live := draftLiveOf(socket.Sight{Draft: view}, "", pool, draw.PlayClock{})
	if _, one := live.OnlyOne(); one {
		t.Error("a watcher's draft reading is read as a decision with one candidate, which " +
			"nothing today can put it in")
	}
	t.Logf("the watcher was handed %d candidates out of a pool of %d with %d decisions "+
		"replayed, after the two players drafted a whole match",
		len(live.Candidates), len(pool), view.Replayed)
}

// TestTheScreenIsHandedThePoolTheRoomDrafts is the half `internal/screen` cannot
// state about itself.
//
// The screen draws whatever pool it is handed — the filter is not its business,
// and its own fixtures hand over a plain cast book and say so — so what has to be
// held here is that the list this client actually hands over is
// `draft.NewPool`'s: the cast **minus every character held back**. A client that
// handed the whole book over would offer a row a player could ban and the room
// would refuse, which is the hot loop step 5a measured.
func TestTheScreenIsHandedThePoolTheRoomDrafts(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	held := 0
	for _, character := range characters.All() {
		if character.Hidden {
			held++
		}
	}
	if held == 0 {
		t.Skip("no shipped character is held back, so nothing here can tell the two lists apart")
	}
	sess := newSession()
	sess.drafting(characters)
	pool := sess.pool()
	if got, want := len(pool), len(characters.All())-held; got != want {
		t.Errorf("the pool handed to the screen holds %d characters over a cast of %d with %d "+
			"held back, want %d", got, len(characters.All()), held, want)
	}
	for _, character := range pool {
		if character.Hidden {
			t.Errorf("%s is held back and is in the pool the screen draws", character.ID)
		}
	}
	// And the order is the book's own, which is step 1's decision: five other
	// listings in this repository are in declaration order, so a draft screen
	// laying the cast out differently is the one thing an order can get wrong.
	at := 0
	for _, character := range characters.All() {
		if character.Hidden {
			continue
		}
		if pool[at].ID != character.ID {
			t.Fatalf("the pool's %dth row is %s where the book declares %s next",
				at, pool[at].ID, character.ID)
		}
		at++
	}
}

// TestAReadingBecomesADrawing drives `draftLiveOf` over a hand-built
// `socket.Sight`, which is the mapping this file is named for.
//
// ⚠️ **Built by hand rather than read off a mirror, deliberately.** What is being
// measured is the conversion, and a `DraftSight` taken out of a real match would
// be whatever that match happened to reach — several of these fields are only
// ever true in states a match passes through in microseconds. The socket path is
// measured whole by the end-to-end test below.
func TestAReadingBecomesADrawing(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	pool := draft.NewPool(characters.All()).All()
	sight := socket.Sight{
		Welcome:  wire.Welcome{Format: wire.Format3v3, Battles: 1, Allowance: 90, Drafts: true},
		Refusals: []wire.Code{wire.CodeIllegalAction, wire.CodeNotYourTurn},
		Draft: socket.DraftSight{
			Mirrored: true,
			OnTurn:   wire.SeatGuest,
			Step:     wire.StepLoadout,
			Open:     true,
			Asked:    true,
			Asking: socket.DraftPrompt{
				Due: socket.DraftDue{Seat: wire.SeatGuest, Step: wire.StepLoadout,
					Character: pool[3].ID, Recorded: 7},
			},
			Picks: [2][]draft.Pick{
				{{Character: pool[1].ID, Stage: "one", Skills: []string{"a"}}},
				{{Character: pool[3].ID}},
			},
			Bans:     [2][]string{{pool[0].ID}, {pool[2].ID}},
			Replayed: 7,
		},
	}
	live := draftLiveOf(sight, wire.SeatGuest, pool, draw.PlayClock{Waiting: draw.PlayClockYou})
	if live.Step != draw.DraftStepLoadout {
		t.Errorf("a loadout arrived as %s", live.Step)
	}
	if live.You != string(wire.SeatGuest) || live.OnTurn != string(wire.SeatGuest) {
		t.Errorf("the seats arrived as you=%q onturn=%q", live.You, live.OnTurn)
	}
	if !live.Yours || live.Subject != pool[3].ID {
		t.Errorf("the loadout arrived yours=%v for %q", live.Yours, live.Subject)
	}
	if live.Recorded != 7 || live.Units != 3 || live.BanSlots != 2 {
		t.Errorf("the counts arrived recorded=%d units=%d bans=%d",
			live.Recorded, live.Units, live.BanSlots)
	}
	// ⚠️ **The LATEST refusal**, because a refusal does not end a connection: on a
	// draft that is not a detail, since a decision taken too early is refused and
	// left open, so what a player needs is the one that just happened.
	if want := wire.CodeNotYourTurn.String(); live.Refusal != want {
		t.Errorf("the refusal arrived as %q, want the latest one %q", live.Refusal, want)
	}
	// And the seat indexing: [0] is the host and [1] the guest, which is
	// socket.DraftSight's own promise, so this reader's own side is the second.
	screen := draw.NewDraftScreen().Attach(draw.Context{}, live)
	if got := screen.Live.You; got != live.Seats[1] {
		t.Fatalf("this reader's seat is %q against seats %v", got, live.Seats)
	}
	// ⚠️ **The marks are read off a BAN reading and not the loadout above**, and
	// that is the screen working rather than a fixture detail: while a loadout is
	// due the editor replaces the pool listing outright, so there are no rows to
	// mark. The same decisions, one step earlier.
	banning := sight
	banning.Draft.Step, banning.Draft.Asked = wire.StepBan, false
	banning.Draft.Asking = socket.DraftPrompt{}
	banning.Draft.Candidates = []cast.Character{pool[4], pool[5]}
	marked := draw.NewDraftScreen().Attach(draw.Context{},
		draftLiveOf(banning, wire.SeatGuest, pool, draw.PlayClock{}))
	c, _, _ := start(t, i18n.Vi)
	for id, want := range map[string]i18n.Key{
		pool[0].ID: i18n.DraftStateTheyBanned,
		pool[1].ID: i18n.DraftStateTheyPicked,
		pool[2].ID: i18n.DraftStateYouBanned,
		pool[3].ID: i18n.DraftStateYouPicked,
		pool[4].ID: i18n.DraftStateOpen,
	} {
		drawn, _ := marked.View(c.ctx())
		row := poolRowFor(t, drawn, id)
		if !strings.Contains(row, c.text(want)) {
			t.Errorf("%s's pool row is %q, want the mark %q", id, row, c.text(want))
		}
	}
	// A closed turn carries no step at all, and it must land on *nothing due*
	// rather than on whichever arm sits at nought by accident.
	closed := sight
	closed.Draft.Open, closed.Draft.Asked, closed.Draft.Step = false, false, ""
	if got := draftLiveOf(closed, wire.SeatGuest, pool, draw.PlayClock{}).Step; got != draw.DraftStepNone {
		t.Errorf("a closed turn arrived as the %s step", got)
	}
	// And the arrange phase is asked about separately, because Turn answers one
	// seat and that phase has two pending at once.
	arranging := closed
	arranging.Draft.Arranging, arranging.Draft.Asked = true, true
	if got := draftLiveOf(arranging, wire.SeatGuest, pool, draw.PlayClock{}); got.Step !=
		draw.DraftStepArrange || !got.Arranging || !got.Yours {
		t.Errorf("the arrange phase arrived as %+v", got)
	}
}

// poolRowFor is the **pool listing's** row for a character, which is not simply
// the first line naming it.
//
// ⚠️ An id appears on this screen three ways: on a pool row, in a side's own
// block, and in the sentence naming whose decision is due — and a helper that
// took the first match read the turn line, which carries no mark at all and
// therefore failed every assertion for the wrong reason. A pool row is the one
// with a **two-cell** marker in front of it; a side's rows are indented four.
func poolRowFor(t *testing.T, drawn, id string) string {
	t.Helper()
	for _, line := range strings.Split(drawn, "\n") {
		if len(line) < 2 || (line[:2] != "  " && line[:2] != "> ") {
			continue
		}
		if strings.HasPrefix(line[2:], id) {
			return line
		}
	}
	t.Fatalf("no pool row of the screen names %s:\n%s", id, drawn)
	return ""
}

// TestADraftedRoomIsDrawnAndABanBeforeThePeerArrivesIsThrownAway is the whole
// vertical for a draft, and it is the finding of step 5a as a player meets it.
//
// A real registry, a real server, a real listener, a room that **drafts**; this
// client takes the host's seat and presses a ban into a room with one player in
// it. ⚠️ **The room refuses it and says nothing when the second seat is later
// taken**, so nothing recorded plus a refusal is the whole of what a player has
// to go on — which is the pair this test asserts is drawn, and then it presses
// again and the ban lands.
//
// *Sees:* the chooser, the one-slot channel, the mapping both ways, Attach, the
// footers, and `Mirror.DecideDraft`'s own staleness check — over a real socket.
// *Cannot see:* the arrange phase, which is step 5c; the draft stops there by
// design and this test says so rather than working around it.
func TestADraftedRoomIsDrawnAndABanBeforeThePeerArrivesIsThrownAway(t *testing.T) {
	held, library := openADraftingRoom(t)
	m, fake := joinedADraft(t, held, library)
	if seat := m.session.seat(); seat != wire.SeatHost {
		t.Fatalf("this client took the %s seat; the ban-before-the-peer state is the "+
			"host's", seat)
	}
	if !m.draft.Live.Waiting() {
		t.Fatalf("a draft with %d decisions recorded is not the waiting state",
			m.draft.Live.Recorded)
	}
	if !strings.Contains(drawnBody(m), firstLine(m.text(i18n.DraftNotBegun))) {
		t.Fatalf("the draft says nothing about the room maybe still being empty:\n%s",
			drawnBody(m))
	}
	// The ban a player makes before anybody else is in the room.
	early, have := m.draft.Chosen()
	if !have || !m.draft.Deciding() {
		t.Fatal("the draft offers no ban to make")
	}
	m = key(t, m, "enter")
	// ⚠️ **Waited for by its EFFECT and not by "a message arrived"**, because a
	// message has already arrived: the chooser sends matchAskingMsg before it
	// blocks — which is 5a's pre-read answer — so it is sitting in the queue
	// before this keystroke is pressed. A drain that took the next message took
	// that one and looked at a screen the refusal had not reached.
	m = settled(t, m, fake, func(m model) bool { return m.draft.Live.Refusal != "" })
	if !strings.Contains(drawnBody(m), m.text(i18n.DraftRefused)) {
		t.Fatalf("the ban %q went out into an empty room and the screen says nothing about "+
			"it having been refused:\n%s", early, drawnBody(m))
	}
	if m.draft.Live.Recorded != 0 {
		t.Errorf("the room recorded %d decisions with one player in it", m.draft.Live.Recorded)
	}
	// The refused ban took nothing out of the pool, which is the state the room's
	// own record says it is in: nothing was recorded, so nothing left.
	if !slices.ContainsFunc(m.draft.Live.Pool, func(one cast.Character) bool {
		return one.ID == early
	}) {
		t.Fatalf("the pool lost %q, which a refused ban never took out", early)
	}
	// ⚠️ **And the row is still marked open**, which is the half a refusal line
	// cannot say on its own: a player reading "you banned" beside a character the
	// room never took has been told the opposite of what happened.
	if row := poolRowFor(t, drawnBody(m), early); !strings.Contains(row,
		m.text(i18n.DraftStateOpen)) {
		t.Errorf("the refused ban left %q drawn as %q, so a thrown-away ban reads as one "+
			"that landed", early, row)
	}
	// Now the second player arrives, and the same keystroke lands.
	//
	// ⚠️ **A second press is what gets it through and there is no memo stopping
	// one**, which is the guard this client may not add: the retry-on-refusal
	// loop is the only mechanism that ever gets a host's first decision through,
	// so a screen that remembered having answered this decision would stall the
	// match for good. → session.decide.
	_, failed := theDraftingOpponent(t, held)
	m = key(t, m, "enter")
	m = settled(t, m, fake, func(m model) bool { return m.draft.Live.Recorded > 0 })
	if m.draft.Live.Recorded == 0 {
		t.Fatalf("no decision was ever recorded, so the retry never got through:\n%s",
			drawnBody(m))
	}
	if m.draft.Live.Waiting() {
		t.Error("the draft still warns about an empty room after a decision was recorded")
	}
	select {
	case err := <-failed:
		t.Fatalf("the opponent's loop ended during the draft: %v", err)
	default:
	}
}

// TestADraftedMatchIsPlayedFromTheFirstBanToTheLastBlowOverALoopbackListener is
// the test this step exists for, and until it landed **a drafting match could not
// reach a board at all**.
//
// ⚠️ **That is measured rather than described.** Picking closed into a phase with
// two decisions open and no screen to take either, so both clients' choosers were
// asked for a StepArrange, answered nothing, and Play returned — *"a draft has no
// pass, so there is nothing to send in its place"*. This test asserted that ending
// **by name** until step 5c; what it asserts now is the whole vertical: a real
// registry, a real server, a real listener, eighteen decisions taken through this
// client's own screens, a formation placed with the arrow keys, and a battle
// fought to a finish.
//
// *Sees:* both draft screens, the chooser, the one-slot channel, the mapping both
// ways, the arrangement in pick order, the room opening the battle on the drafted
// squads, and the result — over a real socket.
// *Cannot see:* that a real *tea.Program delivered the messages. → the note at the
// head of match_test.go.
func TestADraftedMatchIsPlayedFromTheFirstBanToTheLastBlowOverALoopbackListener(t *testing.T) {
	held, library := openADraftingRoom(t)
	// ⚠️ **Both peers are dialled before either is driven**, which is the same
	// arrangement socket's own headline draft test takes and for the same measured
	// reason: the room announces nothing when it fills, so a host driven first bans
	// into an empty room. That state has its own test above.
	_, failed := theDraftingOpponent(t, held)
	m, fake := joinedADraft(t, held, library)
	deadline := time.Now().Add(theWholeMatch)
	for !m.draft.Live.Arranging && time.Now().Before(deadline) {
		if m.draft.Live.Cancelled {
			t.Fatalf("the draft was cancelled part way through:\n%s", drawnBody(m))
		}
		m = aDecisionTaken(t, m, fake)
	}
	if !m.draft.Live.Arranging {
		t.Fatalf("the draft did not reach the arrange phase inside %v:\n%s",
			theWholeMatch, drawnBody(m))
	}
	// Both sides picked, and every pick has its loadout: a pick with no skills is
	// one whose second decision is still open, so this is the whole of the picking.
	units := m.draft.Live.Units
	for side, picks := range m.draft.Live.Picks {
		if len(picks) != units {
			t.Errorf("side %d holds %d picks against %d a side", side, len(picks), units)
		}
		for _, pick := range picks {
			if !pick.Settled() {
				t.Errorf("side %d's pick of %s has no loadout", side, pick.Character)
			}
		}
	}
	// The pool is exclusive, so the six picks are six different characters.
	taken := map[string]bool{}
	for _, picks := range m.draft.Live.Picks {
		for _, pick := range picks {
			if taken[pick.Character] {
				t.Errorf("%s was picked twice out of a pool that is exclusive", pick.Character)
			}
			taken[pick.Character] = true
		}
	}
	// ⚠️ **Derived, and the shape of the derivation is the thing to get right:**
	// a ban slot is one decision and both sides spend all of theirs, while a
	// **pick is two** — the character, then the loadout. So it is
	// `2*bans + 2*2*picks` and never `2*2*bans`, which is the arithmetic slip a
	// literal 16 would have hidden.
	if got, want := m.draft.Live.Recorded, 2*draft.BansPerSide(wire.Format3v3)+
		2*2*draft.PicksPerSide(wire.Format3v3); got != want {
		t.Errorf("the record holds %d decisions, want %d", got, want)
	}
	// ⚠️ **The phase opening is what moves the reader**, and nothing else announces
	// it: the room records the first arrangement nowhere, so there is no second
	// message behind this one. A client that did not move here would sit on a
	// finished ban and pick until the allowance cancelled the draft.
	if m.screen != screenArrange {
		t.Fatalf("the arrange phase opened and this client is on screen %v:\n%s",
			m.screen, drawnBody(m))
	}
	mine := m.arrange.Picks()
	if len(mine) != units {
		t.Fatalf("the arrange screen holds %d of this side's %d picks", len(mine), units)
	}
	// The formation, placed with the keys a player presses: the cursor opens on the
	// front rank, each pick is walked a rank further round than the last, and enter
	// puts it down.
	//
	// ⚠️ **The cells this produces are deliberately not in board order** — measured
	// on the shipped pool it comes out `2,0 · 0,1 · 2,1` where reading the board
	// would give `2,0 · 2,1 · 0,1` — so the check below is a real one: an
	// arrangement that travelled sorted, or in the order the cells were chosen on
	// the board, would put two of these three units in the wrong place.
	for at := range mine {
		for range at {
			m = key(t, m, "right")
		}
		m = key(t, m, "enter")
		if got := len(m.arrange.Slots); got != at+1 {
			t.Fatalf("placing pick %d left %d cells taken", at, got)
		}
	}
	if !m.arrange.Whole() || m.arrange.Err != nil {
		t.Fatalf("the formation is not whole or is refused: %v\n%s", m.arrange.Err, drawnBody(m))
	}
	arrangement := slices.Clone(m.arrange.Slots)
	if len(arrangement) != len(slices.Compact(slices.Clone(arrangement))) {
		t.Fatalf("two picks were put on one cell: %v", arrangement)
	}
	m = key(t, m, "enter")
	if !m.arrange.Sent {
		t.Fatal("enter on a whole formation sent nothing")
	}
	// And the room took it: the phase closes only when **both** arrangements are
	// in, and the battle that follows is the room's answer to this one.
	m = settled(t, m, fake, func(m model) bool { return m.screen == screenBattle })
	if m.draft.Live.Refusal != "" {
		t.Errorf("the room refused a decision on the way through: %q", m.draft.Live.Refusal)
	}

	// ⚠️ **The battle was opened on the drafted squads, by value.** This is the
	// claim the whole step is for: what is standing on the board is what was picked,
	// each unit on the cell this client chose for it, in the order the picks were
	// taken.
	stood := map[string]hex.Offset{}
	var side hex.Side
	m.session.read(func(sight socket.Sight) {
		if sight.Fight == nil {
			return
		}
		side = sight.Side
		for _, unit := range sight.Fight.Units() {
			if unit.Side == sight.Side {
				// The id a drafted unit carries is `<side>.<character>` — the
				// character is the unit id in a drafted squad, because the pool is
				// exclusive, and placement.Squad.Take puts the side in front of it.
				stood[strings.TrimPrefix(unit.ID, sight.Side.String()+".")] = unit.Cell
			}
		}
	})
	if len(stood) != len(mine) {
		t.Fatalf("this client's half of the board holds %d units against %d picks: %v",
			len(stood), len(mine), stood)
	}
	for at, pick := range mine {
		want := placedAt(side, arrangement[at])
		got, standing := stood[pick.Character]
		if !standing {
			t.Errorf("%s was drafted and is not on the board", pick.Character)
			continue
		}
		if got != want {
			t.Errorf("%s stands at %s and was arranged onto %s (%s): Slots[i] is the cell for "+
				"the i-th pick and nothing may reorder it", pick.Character, got, arrangement[at], want)
		}
	}

	// The battle, played out with the keys a player presses. It is match_test.go's
	// own loop, and the only difference is where the board came from.
	answers := 0
	deadline = time.Now().Add(theWholeMatch)
	for m.screen != screenResult && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
		// Two presses a turn, the same driver match_test.go carries and with the
		// same caveat written there: the break agrees with any number of them, so
		// this reaches the result screen and holds no rule about the count.
		for range 2 {
			if m.screen != screenBattle || !m.battle.Live ||
				m.battle.Pending == nil || m.battle.Answered {
				break
			}
			m = key(t, m, "enter")
		}
		if m.battle.Answered {
			answers++
		}
	}
	if m.screen != screenResult {
		t.Fatalf("the drafted match did not reach the result inside %s; the client is on "+
			"screen %v", theWholeMatch, m.screen)
	}
	if err := <-failed; err != nil {
		t.Fatalf("the opponent's loop: %v", err)
	}
	// ⚠️ **The vacuity guards, and they are what make "it finished" mean
	// anything.** A capped battle is a **hang detector firing** rather than an
	// ending — the engine concluded nothing about it and the room records it as
	// undecided — and a run in which this client answered no turn would have proved
	// the chooser and the channel were never exercised.
	var fought []socket.Fought
	m.session.read(func(sight socket.Sight) { fought = append(fought, sight.Fought...) })
	if len(fought) != 1 {
		t.Fatalf("a drafting room is a bo1 and this one settled %d battles", len(fought))
	}
	if fought[0].Capped {
		t.Errorf("the battle stopped at the turn cap after %d turns, which is a hang detector "+
			"firing rather than an ending", fought[0].Turns)
	}
	if !fought[0].Decided {
		t.Errorf("the battle ended undecided (%s) after %d turns",
			fought[0].Outcome, fought[0].Turns)
	}
	if answers == 0 {
		t.Error("this client answered no turn at all, so the chooser and the channel were " +
			"never exercised")
	}
	if m.result.Err != nil {
		t.Errorf("the match ended with %v", m.result.Err)
	}
	standingMine, standingTheirs := m.result.standing()
	t.Logf("drafted %v against %v; arranged %v on the %s side; %s won after %d turns "+
		"(%s), standing %d-%d, %d turns answered here",
		pickedNames(m.draft.Live.Picks[0]), pickedNames(m.draft.Live.Picks[1]),
		arrangement, side, fought[0].Winner, fought[0].Turns, fought[0].Outcome,
		standingMine, standingTheirs, answers)
}

// placedAt is where an authored formation cell lands on the shared board for a
// side, which is the one rotation the geometry does: hex.Place maps the enemy
// formation through 180 degrees, so a cell authored at 2,0 is a different
// battlefield cell depending on which half it is fielded as.
//
// ⚠️ It goes through the engine's own function rather than restating the
// arithmetic, which is the whole point of the assertion above: a test that worked
// the rotation out for itself would agree with a client that had worked it out
// wrong in the same way.
func placedAt(side hex.Side, authored hex.Offset) hex.Offset { return hex.Place(side, authored) }

// pickedNames is a side's picks as the ids a report reads by.
func pickedNames(picks []draw.DraftPick) []string {
	out := make([]string, 0, len(picks))
	for _, pick := range picks {
		out = append(out, pick.Character)
	}
	return out
}

// aDecisionTaken presses whatever the screen is offering and waits for the
// record to move.
//
// ⚠️ **It waits for the RECORD and not for a message**, which is the same trap
// `settled` exists for: the chooser sends matchAskingMsg before it blocks, so a
// drain that stopped at the next message would come back with the same decision
// still open — and the loop would then answer it a second time. That is not
// harmless: a kit picker re-opened over an answer already given has those rows
// **chosen**, so a fixture pressing space on them takes them back off, which is
// exactly what an empty kit and a refusal one keystroke later turned out to be.
//
// The loadout is the interesting arm: it walks to the kit row, opens the real
// picker, chooses up to the slots, closes it, does the same for the trait and
// then sends — which is a player's whole path through a pick's second decision.
func aDecisionTaken(t *testing.T, m model, fake *fakeSender) model {
	t.Helper()
	before := m.draft.Live.Recorded
	moved := func(m model) bool {
		return m.draft.Live.Recorded > before || m.draft.Live.Arranging ||
			m.draft.Live.Cancelled
	}
	if !m.draft.Live.Yours {
		// The other side is deciding, so there is nothing to press.
		return settled(t, m, fake, moved)
	}
	switch m.draft.Live.Step {
	case draw.DraftStepBan, draw.DraftStepPick:
		m = key(t, m, "enter")
	case draw.DraftStepLoadout:
		m = aLoadoutChosen(t, m)
		m = key(t, m, "enter")
	default:
		return settled(t, m, fake, moved)
	}
	return settled(t, m, fake, moved)
}

// aLoadoutChosen fills the two lists through the real pickers and leaves the
// cursor on the send row.
func aLoadoutChosen(t *testing.T, m model) model {
	t.Helper()
	for _, one := range []struct {
		field draw.DraftField
		slots int
	}{{draw.DraftKit, cast.SkillSlots}, {draw.DraftTrait, cast.TraitSlots}} {
		for m.draft.Field != one.field {
			m = key(t, m, "down")
		}
		m = key(t, m, "enter")
		if m.picker == nil {
			t.Fatalf("enter on field %d opened no picker", one.field)
		}
		// ⚠️ **Only rows that are not chosen yet**, because space is a toggle: a
		// row already on the answer comes back off, which is how a full kit
		// becomes an empty one.
		for range one.slots {
			for len(m.picker.Chosen) < one.slots &&
				slices.Contains(m.picker.Chosen, m.picker.Visible()[m.picker.Cursor].ID) {
				m = key(t, m, "down")
			}
			if len(m.picker.Chosen) >= one.slots {
				break
			}
			m = key(t, m, "space")
			m = key(t, m, "down")
		}
		m = key(t, m, "enter")
		if m.picker != nil {
			t.Fatalf("enter did not close the picker for field %d", one.field)
		}
	}
	if m.draft.Err != nil {
		t.Fatalf("the loadout this client chose is refused: %v", m.draft.Err)
	}
	for m.draft.Field != draw.DraftSend {
		m = key(t, m, "down")
	}
	return m
}

// joinedADraft is this client seated in a drafting room, on the draft screen.
//
// ⚠️ **It brings NO squad, and that is the handshake wrinkle rather than a
// fixture shortcut.** A room that drafts refuses a squad outright — the two
// sides ban and pick out of a shared pool there, so the side built at home is
// not the side that will be fielded — and a client cannot know a room drafts
// until it has been **welcomed**, because the hello carrying the squad goes
// first. So the join screen's chooser has a "bring none" position and this is a
// player cycling to it. → joinScreen.Squad.
//
// ⚠️ **And the draft screen is reached without a single message arriving.** The
// welcome is what tells a client the room drafts, and the room announces nothing
// when its second seat is taken, so model.joined enters the screen itself —
// which is what the assertion below is really about.
func joinedADraft(t *testing.T, held *aRoom, library *forge.Library) (model, *fakeSender) {
	t.Helper()
	m, fake := joining(t, held, library, i18n.Vi)
	// left from the first side wraps onto the last position, which is "none".
	m = key(t, m, "left")
	if _, have := m.join.Chosen(); have {
		t.Fatal("the chooser is still on a saved side, so this join would be refused for " +
			"bringing one")
	}
	if !strings.Contains(drawnBody(m), m.text(i18n.JoinBringNone)) {
		t.Fatalf("the join screen does not say it is bringing nothing:\n%s", drawnBody(m))
	}
	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	answer := command()
	if failure, refused := answer.(matchFailedMsg); refused {
		t.Fatalf("the dial was turned away: %v", failure.err)
	}
	m = send(t, m, answer)
	if m.screen != screenDraft {
		t.Fatalf("a joined drafting room landed on screen %v rather than the draft "+
			"(welcome %+v)", m.screen, m.waiting.Welcome)
	}
	if !m.draft.Drafting {
		t.Fatal("the draft screen has no reading, so nothing was attached")
	}
	return m, fake
}

// settled drains whatever the match sends until the model answers `until`, and
// fails rather than hanging when it never does.
//
// ⚠️ **It waits for an EFFECT rather than for a message**, and that is the
// difference this fixture had to learn: the draft chooser sends matchAskingMsg
// *before* it blocks, so on a drafting room there is already a message in the
// queue when a keystroke is pressed. A helper that drained the next message and
// stopped read a screen the answer had not reached yet.
func settled(t *testing.T, m model, fake *fakeSender, until func(model) bool) model {
	t.Helper()
	deadline := time.Now().Add(theWholeMatch)
	for !until(m) && time.Now().Before(deadline) {
		if !fake.awaits(time.Second) {
			continue
		}
		for _, message := range fake.take() {
			m = send(t, m, message)
		}
	}
	if !until(m) {
		t.Fatalf("the match never reached the state this test is about inside %v:\n%s",
			theWholeMatch, drawnBody(m))
	}
	return m
}

// firstLine is enough of a wrapped sentence to recognise it by after the screen
// has broken it at the floor.
func firstLine(text string) string { return draw.WrapWords(text, draw.MinWidth-3)[0] }

// openADraftingRoom is openARoom with the ban and pick turned on.
//
// ⚠️ **Bo1 only, and that is refused rather than chosen.** `room.Config.Validate`
// declines a drafting series by name: "a ban lasts the match" is ambiguous in a
// bo3, and accepting one would silently pick one of the three games that item
// lists.
func openADraftingRoom(t *testing.T) (*aRoom, *forge.Library) {
	t.Helper()
	return openARoomConfigured(t, room.Config{
		Format: wire.Format3v3, Battles: 1, Drafts: true,
		Allowance: room.DefaultAllowance, Seed: 11, TurnCap: room.DefaultTurnCap,
	})
}

// openADraftingWatchableRoom is openADraftingRoom with somebody allowed to
// watch, which room.Config.Validate accepts: the two settings are orthogonal —
// nothing refuses a drafting room a spectator, and what such a spectator is told
// is the finding rather than a refusal.
func openADraftingWatchableRoom(t *testing.T) (*aRoom, *forge.Library) {
	t.Helper()
	return openARoomConfigured(t, room.Config{
		Format: wire.Format3v3, Battles: 1, Drafts: true, Watchable: true,
		Allowance: room.DefaultAllowance, Seed: 11, TurnCap: room.DefaultTurnCap,
	})
}

// aWatchingDrafter is a plain socket.Client watching a room that drafts.
//
// ⚠️ **It is dialled with the cast book even though it decides nothing**, and
// that is production rather than belt and braces: a welcome saying the room
// drafts is where a client builds its own pool, and socket.Mirror.openDraft
// **errors** — closing the connection — when it has no book to build one from. So
// a watcher dialled without one could not join a drafting room at all, which is
// why cmd/hexarena-tui hands Characters over on every dial and not only on the
// ones that will take a seat. → session.dial.
//
// The chooser answers nothing: a watcher is never asked, and a battle chooser
// that returned a decision would be this fixture playing the match it came to
// watch.
func aWatchingDrafter(t *testing.T, held *aRoom) *socket.Client {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	client, err := socket.Dial(context.Background(), held.code, wire.Hello{
		Version: version, Name: "Khách", Watch: true,
	}, held.books, socket.ClientOptions{Characters: held.deps.Characters})
	if err != nil {
		t.Fatalf("the watcher could not join the drafting room: %v", err)
	}
	go func() {
		_ = client.Play(context.Background(), func(*battle.Prompt) (battle.Choice, bool) {
			return battle.Choice{}, false
		})
	}()
	t.Cleanup(client.Close)
	return client
}

// theDraftingOpponent is a plain socket.Client on the other seat, answering its
// own draft decisions off its own mirror.
//
// ⚠️ **It brings NO squad**, which is the handshake wrinkle rather than an
// omission: a room that drafts refuses a squad outright (CodeSquadUnwanted),
// because the two sides ban and pick their sides in the room. A client cannot
// know that until it is welcomed, so the refusal is the discovery mechanism —
// and this fixture is the informed second attempt.
func theDraftingOpponent(t *testing.T, held *aRoom) (*socket.Client, chan error) {
	t.Helper()
	version, err := wire.Local(buildString())
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	client, err := socket.Dial(context.Background(), held.code, wire.Hello{
		Version: version, Name: "Nam",
	}, held.books, socket.ClientOptions{
		Characters: held.deps.Characters,
		Draft:      firstOffered(held.deps.Characters),
	})
	if err != nil {
		t.Fatalf("the opponent could not join the draft: %v", err)
	}
	failed := make(chan error, 1)
	go func() {
		failed <- client.Play(context.Background(), func(prompt *battle.Prompt) (battle.Choice, bool) {
			fight := client.Mirror().Battle()
			if fight == nil {
				return battle.Choice{}, false
			}
			return fight.Suggest(prompt)
		})
	}()
	t.Cleanup(client.Close)
	return client, failed
}

// firstOffered is the simplest legal draft chooser there is: the first candidate
// on the list, and the first legal loadout for whatever it just picked.
//
// ⚠️ **It asks cast.ChooseLoadout for its own kit exactly as the screen does**,
// which is what keeps it from spinning: an illegal decision is refused and left
// open, so a chooser that answered one would re-send it for ever. → the draft
// screen's own send, which blocks on the same call.
func firstOffered(characters *cast.Book) socket.DraftChooser {
	return func(prompt socket.DraftPrompt) (socket.DraftAnswer, bool) {
		answer := socket.DraftAnswer{
			For:      prompt.Due,
			Decision: wire.DraftDecision{Step: prompt.Due.Step},
		}
		switch prompt.Due.Step {
		case wire.StepBan, wire.StepPick:
			if len(prompt.Candidates) == 0 {
				return socket.DraftAnswer{}, false
			}
			answer.Decision.Character = prompt.Candidates[0].ID
		case wire.StepLoadout:
			character, known := characters.Get(prompt.Due.Character)
			if !known {
				return socket.DraftAnswer{}, false
			}
			_, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
			if err != nil {
				// A forking line has no single furthest form, so an arm has to be
				// named — which is exactly what the screen's own chooser does.
				arms, armErr := character.FurthestAt(progression.LevelCap)
				if armErr != nil || len(arms) == 0 {
					return socket.DraftAnswer{}, false
				}
				stage = arms[0]
				answer.Decision.Stage = stage.Name
			}
			skills, passives, err := cast.ChooseLoadout(character.ID,
				upToSlots(character.SkillsAt(progression.LevelCap, stage.Name), cast.SkillSlots),
				upToSlots(character.PassivesAt(progression.LevelCap, stage.Name), cast.TraitSlots),
				character, progression.LevelCap, stage.Name)
			if err != nil {
				return socket.DraftAnswer{}, false
			}
			answer.Decision.Skills, answer.Decision.Passives = skills, passives
		case wire.StepArrange:
			// ⚠️ **One cell per pick, in pick order, and the phase closes only when
			// BOTH sides have answered** — so an opponent that could not arrange is
			// an opponent whose Play loop ends the moment the picking does, which is
			// exactly what this fixture could not do before step 5c.
			//
			// The cells are the formation's own first n, which is the simplest legal
			// answer there is: draw.FormationSlots is front first, so this side is
			// packed into its front column. What a *player* does with the decision is
			// the client's screen, and that is what the test above drives.
			slots := draw.FormationSlots()
			if len(prompt.Mine) > len(slots) {
				return socket.DraftAnswer{}, false
			}
			answer.Decision.Slots = slices.Clone(slots[:len(prompt.Mine)])
		default:
			return socket.DraftAnswer{}, false
		}
		return answer, true
	}
}
