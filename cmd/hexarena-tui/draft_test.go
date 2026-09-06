package main

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
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

// TestADraftIsPlayedOutToTheArrangePhaseOverALoopbackListener drives the whole
// ban and pick with the keys a player presses, loadouts and pickers included.
//
// ⚠️ **It stops at the arrange phase and that is the honest end of step 5b.**
// Picking closes into a phase with **two** decisions open at once and no screen
// to take either — that is 5c — so the draft cannot reach a battle yet. What this
// asserts is everything up to that line: eighteen decisions taken through this
// client's own screen, both sides' picks with their loadouts in, and the arranging
// notice drawn.
func TestADraftIsPlayedOutToTheArrangePhaseOverALoopbackListener(t *testing.T) {
	held, library := openADraftingRoom(t)
	// ⚠️ **Both peers are dialled before either is driven**, which is the same
	// arrangement socket's own headline draft test takes and for the same measured
	// reason: the room announces nothing when it fills, so a host driven first
	// bans into an empty room. That state has its own test above; this one is
	// about the eighteen decisions after it.
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
	if !strings.Contains(drawnBody(m), firstLine(m.text(i18n.DraftArrangingNow))) {
		t.Fatalf("the arrange phase draws nothing saying picking is over:\n%s", drawnBody(m))
	}
	// And no keystroke on this screen can take an arrangement, which is what
	// stops 5b sending a decision it has no screen for.
	before := m.draft
	for _, name := range []string{"enter", "s"} {
		after := key(t, m, name)
		if after.draft.Live.Recorded != before.Live.Recorded {
			t.Errorf("%q took an arrangement this step has no screen for", name)
		}
	}
	// ⚠️ **And the opponent's loop ends HERE, naming the arrangement**, which is
	// the boundary of this step written down rather than worked around: a draft
	// has no pass, so a chooser with nothing to answer an arrange with reports a
	// failure and Play returns — see socket.Client.answer, which says so in as
	// many words. That is what 5c closes, and until it does, a drafting match
	// cannot reach a board.
	select {
	case err := <-failed:
		if err == nil || !strings.Contains(err.Error(), string(wire.StepArrange)) {
			t.Fatalf("the opponent's loop ended during the draft with %v, want the arrange "+
				"phase nothing can answer yet", err)
		}
		t.Logf("both loops end at the arrange phase, which is step 5c: %v", err)
	case <-time.After(time.Second):
		t.Log("the opponent's loop is still waiting out the arrange phase's allowance")
	}
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
		default:
			return socket.DraftAnswer{}, false
		}
		return answer, true
	}
}
