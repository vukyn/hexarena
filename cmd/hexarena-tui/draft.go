package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/draft"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The one place two draft vocabularies meet
//
// `internal/screen` cannot see `internal/wire`, `internal/draft` or
// `internal/socket` — a screen there is shared with `cmd/hexforge-tui` and the
// protocol may not follow it in — so `draw.DraftScreen` declares its own
// `DraftLive` and `DraftDecision`, and this file is the mapping. It is `liveOf`'s
// arrangement exactly, one domain wider: that function turns a `socket.Sight`
// into a `draw.PlayLive` and carries the refusal as a **name** for the same
// reason.
//
// ⚠️ **The cost of two vocabularies is paid by a WALK rather than by the
// compiler**, and this is the file the walk is over. There is no compiler trick
// available: a keyed composite literal drops a field silently — `CLAUDE.md`
// records measuring that on `skillFile`, and says in as many words that "two
// structs the compiler keeps in step" is not a precedent this repository has. So
// `draftSteps` and `wireSteps` are held **total in both directions** against
// `wire.DraftSteps()` and `draw.DraftStepCount`:
// `TestEveryDraftStepTheProtocolDeclaresReachesAScreenArm` and
// `TestEveryDraftStepThisScreenDrawsComesFromTheProtocol`. A sixth step added to
// the protocol is a red test rather than a screen quietly drawing nothing about
// it, and a seventh arm added to the screen is a red test rather than an arm
// nothing can reach.
//
// ⚠️ **The two maps are written out rather than one being derived from the
// other**, which looks like exactly the duplication the walk is about. It is the
// walk's own premise: a single map plus an inversion would make the two
// directions agree **by construction**, so the reverse walk would measure nothing
// — and the two directions are used by different code for different things (a
// reading is mapped in, a decision is mapped out), so one of them going wrong
// alone is the failure to catch.

// draftSteps is what a protocol step means to the screen.
var draftSteps = map[wire.DraftStep]draw.DraftStep{
	wire.StepBan:     draw.DraftStepBan,
	wire.StepPick:    draw.DraftStepPick,
	wire.StepLoadout: draw.DraftStepLoadout,
	wire.StepArrange: draw.DraftStepArrange,
	wire.StepTimeout: draw.DraftStepTimeout,
}

// wireSteps is the same reading the other way, for a decision on its way out.
var wireSteps = map[draw.DraftStep]wire.DraftStep{
	draw.DraftStepBan:     wire.StepBan,
	draw.DraftStepPick:    wire.StepPick,
	draw.DraftStepLoadout: wire.StepLoadout,
	draw.DraftStepArrange: wire.StepArrange,
	draw.DraftStepTimeout: wire.StepTimeout,
}

// draftSeats is the order `socket.DraftSight` indexes both sides' decisions by,
// which is the order a room hands its seats out.
//
// ⚠️ **It is handed to the screen rather than re-derived there**, and that is
// why the screen takes an array of names: `internal/draft`, `internal/room` and
// `internal/socket` each keep an unexported copy of these three lines for their
// own stated reasons, and a fourth in a package that cannot even name a
// `wire.Seat` would be the worst of the four. What the screen does with it is a
// lookup in the array it was given, so the order is this file's statement and
// not a second declaration of it.
func draftSeats() [2]string {
	return [2]string{string(wire.SeatHost), string(wire.SeatGuest)}
}

// draftLiveOf is a mirror's reading of a draft turned into what the draft screen
// needs of it.
//
// ⚠️ **The pool is a parameter and is NOT on the reading**, which is the one
// thing here that is a decision rather than a conversion. `socket.DraftSight`
// carries what is left to choose from and both sides' decisions; it does not
// carry the pool, because a mirror computes its own from `ClientOptions.Characters`
// and has no reason to hand it back on every reading. The screen draws the whole
// pool with the gone rows marked, so it needs the list — and it has to be the
// **same** list the mirror drafts from, which is why both come off `session.pool`
// rather than off `Context.Lib`. → `draw.DraftLive.Pool`.
//
// ⚠️ **The countdown is a parameter for `liveOf`'s reason**: this turns a reading
// into a drawing, and what a clock says is not on the reading. → clock.go.
//
// ⚠️ **The refusal is carried as a NAME**, which is what keeps `internal/wire`
// out of `internal/screen`. The **latest** one, because a refusal does not end a
// connection — and on a draft that is not a detail: a decision taken before the
// second seat is taken is refused and left open, so the refusal a player needs to
// read is the one that just happened. → `draw.DraftLive.Waiting`.
func draftLiveOf(sight socket.Sight, seat wire.Seat, pool []cast.Character,
	clock draw.PlayClock) draw.DraftLive {
	view := sight.Draft
	live := draw.DraftLive{
		Pool:       pool,
		Candidates: draftCandidateIDs(view.Candidates),
		Picks:      [2][]draw.DraftPick{draftPicksOf(view.Picks[0]), draftPicksOf(view.Picks[1])},
		Bans:       view.Bans,
		Seats:      draftSeats(),
		You:        string(seat),
		Yours:      view.Asked,
		Recorded:   view.Replayed,
		Units:      draft.PicksPerSide(sight.Welcome.Format),
		BanSlots:   draft.BansPerSide(sight.Welcome.Format),
		Arranging:  view.Arranging,
		Cancelled:  view.Cancelled,
		Clock:      clock,
	}
	// ⚠️ **Open is what says the step means anything**, and it is false during the
	// arrange phase as well as when nothing is due: `draft.Draft.Turn` answers one
	// seat and that phase has two pending at once. So the step is read off the
	// reading where one is open and off `Arranging` where it is not — which is the
	// same division `socket.Mirror.draftAsking` makes one layer down.
	if view.Open {
		live.OnTurn = string(view.OnTurn)
		// A step the map does not hold leaves DraftStepNone, which is "nothing
		// due" — the safe half. The empty wire.DraftStep a closed Turn answers
		// with lands there by construction, and that is the only value that
		// should: the walk over wire.DraftSteps() is what says so for the five
		// real ones.
		live.Step = draftSteps[view.Step]
	}
	if view.Arranging {
		live.Step = draw.DraftStepArrange
	}
	if view.Asked {
		live.Subject = view.Asking.Due.Character
	}
	if len(sight.Refusals) > 0 {
		live.Refusal = sight.Refusals[len(sight.Refusals)-1].String()
	}
	return live
}

// draftCandidateIDs is the open decision's candidates as ids, which is all the
// screen draws of them: the pool row is where a character's element and its name
// come from, so a second copy of the character here would be a second answer to
// what the row says.
func draftCandidateIDs(candidates []cast.Character) []string {
	out := make([]string, 0, len(candidates))
	for _, character := range candidates {
		out = append(out, character.ID)
	}
	return out
}

// draftPicksOf is one side's picks as the screen draws them.
//
// The level is dropped on purpose: every drafted unit fights at the cap, so a
// number on the screen would be a number a reader wonders about. →
// `draw.DraftPick`.
func draftPicksOf(picks []draft.Pick) []draw.DraftPick {
	out := make([]draw.DraftPick, 0, len(picks))
	for _, pick := range picks {
		out = append(out, draw.DraftPick{
			Character: pick.Character,
			Stage:     pick.Stage,
			Skills:    pick.Skills,
			Passives:  pick.Passives,
		})
	}
	return out
}

// draftDecisionOf is a decision the screen took, on its way to the wire.
//
// It reports false for a step the protocol does not have, which is a screen arm
// with no protocol behind it — a bug rather than a state, exactly as a
// `draw.Target` with no entry in `raiseTargets` is, and the walk above is what
// says so out loud.
func draftDecisionOf(decision draw.DraftDecision) (wire.DraftDecision, bool) {
	step, known := wireSteps[decision.Step]
	if !known {
		return wire.DraftDecision{}, false
	}
	return wire.DraftDecision{
		Step: step,
		// ⚠️ **An empty Character on a ban is the SKIP and travels as an
		// absence.** It is not a missing field to fill in: a ban that names
		// nobody is a slot spent on nobody, there is no third state, and
		// `wire.DraftDecision.Character` is `omitempty` for exactly this.
		Character: decision.Character,
		// The form as **named**, never as resolved. → `draw.DraftDecision`.
		Stage:    decision.Stage,
		Skills:   decision.Skills,
		Passives: decision.Passives,
		// ⚠️ **In pick order, and the mapping may not reorder it.** `Slots[i]` is
		// the cell for that side's i-th pick — `draft.Arrange` reads it that way
		// and has no other way to know who a cell is for — so anything that sorted
		// or regrouped this would field somebody else's formation without a word.
		// → `draw.ArrangeScreen`, which builds it, and `wire.DraftDecision.Slots`.
		Slots: decision.Slots,
	}, true
}

// draftPickedInto is where each of the loadout's two lists lands.
//
// ⚠️ It has to be **total** over `draw.DraftPickIntoCount`: a destination with no
// entry swallows a finished pick in silence — the list comes down, the reader
// believes they chose, and the field is unchanged.
// `TestEveryDraftPickDestinationLandsSomewhere` walks the count rather than this
// map, for the reason `raiseTargets`' own walk does.
//
// Two entries and one adapter, which is not a table wanting collapsing: the key
// is the **destination**, because a destination is what may go unhandled, and
// `draw.DraftScreen.Picked` is what knows which of its own fields each one is.
var draftPickedInto = map[any]func(model, any, draw.PickAnswer) (model, draw.Action){
	draw.DraftPickKit:   landOnDraft,
	draw.DraftPickTrait: landOnDraft,
}

func landOnDraft(m model, into any, answer draw.PickAnswer) (model, draw.Action) {
	next, action := m.draft.Picked(m.ctx(), into, answer)
	m.draft = next
	return m, action
}

// updateDraft hands one keystroke to the draft screen and does whatever it asks
// for.
//
// ⚠️ **A decision goes to the session and never to the mirror**, which is the
// division the whole PvP design rests on: a client steps its draft from the
// `wire.Drafted` that comes back rather than from its own input, so that both
// ends compute the state from the same call. → `socket.Mirror.DecideDraft`.
//
// ⚠️ **The record length is read AFTER the screen took the keystroke**, exactly
// as the battle's prompt is: `m.draft` is already the answered screen here, and
// the reading it holds is the one the decision was taken against. That number is
// what the chooser routes on. → `session.decide`, and `socket.DraftDue.Recorded`.
func (m model) updateDraft(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, result := m.draft.Update(m.ctx(), message)
	m.draft = next
	if result.Decided {
		decision, known := draftDecisionOf(result.Decision)
		if known {
			m.session.decide(decision, m.draft.Live.Recorded)
		}
		return m, nil
	}
	return m.navigate(screenDraft, result.Action)
}

// updateArrange is updateDraft one decision later, and it is the same three lines
// for the same reasons: the decision goes to the session rather than to the mirror,
// and the record length is read **after** the screen took the keystroke, because
// that is the reading the decision was taken against.
//
// ⚠️ **It is a second function rather than a shared one taking a screen**, because
// what the two share is a result type and not a screen: `m.draft` and `m.arrange`
// are different types with different Updates, and a wrapper over both would be a
// third vocabulary for a keystroke. What is genuinely shared — the mapping out to
// the wire and the routing key — is `draftDecisionOf` and `session.decide`, and
// both are called here unchanged.
func (m model) updateArrange(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, result := m.arrange.Update(m.ctx(), message)
	m.arrange = next
	if result.Decided {
		decision, known := draftDecisionOf(result.Decision)
		if known {
			m.session.decide(decision, m.arrange.Live.Recorded)
		}
		return m, nil
	}
	return m.navigate(screenArrange, result.Action)
}
