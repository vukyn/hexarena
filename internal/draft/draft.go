package draft

import (
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/wire"
)

// seatCount is how many seats a draft has, and it is two for the reason
// internal/room's own seatCount says it is two: a third client is a full room
// today and a spectator is a different kind of citizen.
//
// ⚠️ **The seats are an array indexed by this rather than a map keyed by
// wire.Seat, and here the reason is the record.** internal/room states it as
// "the order two seats are visited in reaches the roster, and the roster's order
// decides which side wins a speed tie"; one layer earlier the outputs are this
// draft's **record** and its **remaining pool**, and a map range would randomise
// both — so two peers replaying the same decisions would compute different
// states and the mirror this package exists for would stop working.
//
// It is a copy of three unexported lines in internal/room rather than a shared
// helper, deliberately: a draft is not a room, room's copies are unexported, and
// what is worth sharing is the *rule* rather than the three lines — which is why
// the rule is written out again here instead of being cited.
const seatCount = 2

// seats is every seat a draft hands a decision to, in the order a room hands
// them out. Which of the two goes first is Config.First and is not this list.
var seats = [seatCount]wire.Seat{wire.SeatHost, wire.SeatGuest}

// indexOf is a seat's position, and reports false for anything that is not one
// of the two — including the zero Seat, which means "no seat" and must not
// quietly mean the host.
func indexOf(seat wire.Seat) (int, bool) {
	for index, candidate := range seats {
		if candidate == seat {
			return index, true
		}
	}
	return 0, false
}

// The step vocabulary — which kind of decision a draft is asking for, and what
// a draft's record can hold — is **wire.DraftStep**, and it is not declared here.
//
// ⚠️ **It was, until the record went on the wire, and the move is a consequence
// of this package importing internal/wire rather than a preference.** A draft's
// record is what a mirror replays, so it is what travels; internal/wire may not
// import this package (it would be a cycle, since Config.Format and every seat
// here are wire's), so the shape has to be declared over there and named from
// here. That is the same relationship Format and Seat already have with this
// package, which is why there is no local alias for it either: one spelling, and
// it is the protocol's. → wire.DraftStep, and TODO.md § *The draft on the wire*
// for the shape decision and what it cost.
//
// ⚠️ **The vocabulary is wider than what Turn answers, and two of the five are
// deliberately outside it.** Turn answers a ban, a pick and a loadout — one seat
// and one step — because every refusal in this package is written on there being
// exactly one open decision. wire.StepTimeout is a thing that happened to a
// draft rather than a decision anybody is due, and wire.StepArrange is **two**
// decisions pending at once, which is the reason the arrange phase has accessors
// of its own. → Turn, Arranging, AwaitingArrangement.

// Pick is one character a side took, with the loadout it took it in: a
// placement.Placement **minus its Slot and its ID**.
//
// ⚠️ **The slot is not a pick's decision, and this type is that decision made.**
// TODO.md § "Ban and pick" (g): once picking closes both sides arrange their own
// three (or five) **privately and simultaneously**, which is a different shape
// from this one — two decisions pending at once, and each side's secret until
// both are in. That is the arrange phase, and Arrange is where a Slot arrives.
//
// ⚠️ **So a Pick is deliberately not a placement, and Squads answers only once
// the phase has closed.** hex.Offset's zero value is a real cell (the ally back
// corner, exactly the trap wire.Act documents for its Aim), so a squad handed
// out with Slot left at its zero would *look* authored, and what happens to it
// is worse than a visible gap — placement.Squad.Validate refuses the second unit
// standing where the first already is, so the squad is turned away at the moment
// it is fought, naming a cell nobody chose. An output that cannot be used is
// worse than an output that is honestly incomplete, which is why Squads answers
// two squads with nobody in them until Done.
//
// **What turning these into squads takes**, from this end: a Slot per pick from
// the arrange phase, and an ID unique within the side, which is the **character
// id** — a drafted side cannot double up, because the pool is exclusive.
// placement.Placement.ID is per squad and Squad.Take prefixes the side, so a
// draft has nothing to invent there. Level and Stage are already resolved here,
// so nothing else is owed.
//
// ⚠️ **A drafted squad cannot double up, and that does not contradict the
// design record — it scopes it.** CLAUDE.md records "one squad may field the
// same character twice" as decided *yes*, and that is about a **saved** squad.
// The pool is exclusive, so a drafted side gets six or ten different characters
// by construction: both hold, and the scope is what has to be legible.
type Pick struct {
	// Character names an entry in the cast book, and is the id the pick was
	// taken with.
	Character string
	// Level is progression.LevelCap, always.
	//
	// ⚠️ **Every drafted unit fights at the cap on its chosen form, and this is
	// where a reader will look for that.** It is not a knob: cast.ParseBuilds
	// validates every entry of builds.json at the cap on the furthest form, so a
	// draft at any other level would make every shipped build an illegal loadout
	// — which is decision (a), "a pick takes a build already in builds.json or
	// one made on the spot", broken by arithmetic. It is carried as a field
	// rather than left to be assumed so that a Pick becomes a placement by
	// copying fields rather than by a reader remembering a number.
	Level int
	// Stage is the form this pick will field, **as resolved** rather than as
	// named: on a line that does not fork the loadout may leave it out and this
	// is the one form there is, so nothing downstream has to know which sort of
	// line it came off. It is the same normalisation cast.resolveBuild does, for
	// the same reason.
	Stage string
	// Skills and Passives are the loadout, already checked against what this
	// character knows at the cap as this form — cast.ChooseLoadout's answer, and
	// its order.
	//
	// Both are empty while the pick's loadout is still open, which is a state
	// Turn reports rather than a state to be detected: a pick with no skills is
	// a character taken out of the pool whose second decision has not been made.
	Skills   []string
	Passives []string
}

// clone hands out a copy, so a caller holding a pick cannot edit the draft's.
func (p Pick) clone() Pick {
	out := p
	out.Skills = slices.Clone(p.Skills)
	out.Passives = slices.Clone(p.Passives)
	return out
}

// Config is everything a draft is set up with, and every one of these is settled
// before the first decision is asked for.
//
// ⚠️ There is no clock in it and there is no clock anywhere in this package —
// TestTheDraftReadsNoClock holds that by import. The allowance a draft runs
// under is counted down by whoever owns the transport, exactly as a room's is,
// and it arrives here as TimedOut.
type Config struct {
	// Format is how many units a side, which is how many picks a side and — via
	// BansPerSide — how many bans.
	Format wire.Format
	// Pool is the characters this draft may seat, fixed for the whole of it.
	Pool Pool
	// First is the seat that decides first, in **both** stages. The host is what
	// a room will pass; it is a parameter because who goes first is a fact about
	// the room rather than about the draft, and because a bo3 draft (its own
	// TODO.md item) may well want the loser of the previous battle.
	First wire.Seat
}

// Draft is the ban-and-pick in progress: whose decision is due, what the pool
// has left, what each side has taken, and the record of every decision so far.
//
// It holds state and nothing else does — the same division internal/room keeps.
// Nothing in it is a clock, a source of entropy or a map that reaches an output,
// so a draft is a pure function of the decisions taken and a client replaying
// those decisions computes exactly this.
type Draft struct {
	format wire.Format
	pool   Pool
	// first is Config.First as a position in seats, so the alternation is
	// arithmetic rather than a comparison chain.
	first int

	// spent is every character out of the pool and who took it how. A slice
	// rather than a map keyed by id, for the reason Pool.Has scans: at this size
	// the scan is free, and a map beside it would be a second copy of the pool
	// that a later range could turn into an ordering.
	spent []spending
	// taken is each side's picks, in the order that side took them, indexed by
	// seats.
	taken [seatCount][]Pick

	bans  int
	picks int
	// awaiting says a loadout is due, and pending is the pool position of the
	// character it is due for.
	//
	// ⚠️ Absence is **declared** rather than encoded in pending. A sentinel index
	// would read as working, because it is a legal index the moment it is not
	// used — which is the mistake CLAUDE.md records against the battle log's
	// follow-the-tail offset, and the one the abandoned queue tie-break paid for.
	awaiting bool
	pending  int

	// arranged is each side's arrangement, indexed by seats: the cell for each of
	// that side's picks, in pick order.
	//
	// ⚠️ **It is held here rather than recorded as it arrives, and that buffer is
	// the whole mechanism the arrange phase's secrecy costs.** An entry is public
	// the moment it is appended and a mirror replaying the record computes the
	// state, so appending the first arrangement when it arrives *is* showing it
	// to the other player. Both go in together, in seats order. → Arrange.
	//
	// ⚠️ Absence is the **empty slice** here and needs no flag beside it, which
	// is the opposite call from awaiting/pending above and for a stated reason:
	// Arrange refuses a slice whose length is not that side's pick count, and no
	// format fields nought units, so a stored arrangement always holds at least
	// three cells and an empty one cannot be one. A flag would be the mistake
	// wire.DraftDecision.Character refuses — two fields that could disagree about
	// one fact.
	arranged [seatCount][]hex.Offset

	// abandoned is the seat whose allowance ran out, and empty for a draft that
	// was not cancelled. The zero wire.Seat already means "no seat", so this is
	// one statement rather than a bool beside a name that could disagree with it.
	abandoned wire.Seat

	entries []wire.DraftEntry
}

// spending is one character out of the pool: which, by whom, and as what.
type spending struct {
	character string
	seat      wire.Seat
	step      wire.DraftStep
}

// New sets a draft up, or says why it could never finish.
//
// ⚠️ **A draft that cannot be completed is refused here and never discovered
// halfway through**, which is the whole arrangement this package rests on: Fits
// measures the pool against every ban being spent, and the package comment
// carries the proof that a draft Fits allowed cannot run dry whatever order the
// decisions come in and whichever bans are skipped. So there is no pool
// exhaustion rule anywhere below this line, and adding one would be re-adding
// the branch #304 deleted.
func New(config Config) (*Draft, error) {
	// Fits refuses a format the game does not offer as well as a pool too small,
	// so it is the whole of the format gate and BansPerSide is safe after it.
	if err := Fits(config.Pool.Len(), config.Format); err != nil {
		return nil, err
	}
	if !config.First.Valid() {
		return nil, fmt.Errorf("a draft cannot begin with %q, which is not one of the two "+
			"seats a room hands out", config.First)
	}
	// The two checks below are what make Fits' figure a count of characters a
	// decision can actually name. Fits counted Pool.Len(), and both of these
	// would leave the pool holding fewer usable characters than it holds
	// entries — which is the arithmetic the exhaustion proof is built on
	// quietly failing rather than a cosmetic complaint.
	seated := config.Pool.All()
	for at, character := range seated {
		if character.ID == "" {
			return nil, fmt.Errorf("the character at position %d of this draft's pool has no "+
				"id, so no ban and no pick can name it, and the pool is one character smaller "+
				"than it counts", at)
		}
		for _, earlier := range seated[:at] {
			if earlier.ID == character.ID {
				return nil, fmt.Errorf("two characters in this draft's pool are called %q, so "+
					"taking one would take both, and the pool is one character smaller than it "+
					"counts", character.ID)
			}
		}
	}
	first, _ := indexOf(config.First)
	return &Draft{
		format: config.Format,
		pool:   config.Pool,
		first:  first,
	}, nil
}

// Turn is whose decision is due and which kind, and false when nothing is: a
// draft whose picking is over, and one that was cancelled.
//
// ⚠️ **Turn answers one seat and one step, and it never widened for the arrange
// phase.** Every refusal in this package is written on there being exactly one
// open decision — the loadout refusal ("nothing else can be decided until the
// form, the skills and the trait are chosen") is that assumption spoken aloud —
// and two arrangements are pending at once. Widening this to answer a *set*
// would change every one of those refusals for the sake of one phase, so the
// phase is asked about through Arranging and AwaitingArrangement and this keeps
// answering ban / pick / loadout and nothing else. → TODO.md § "The arrange
// phase".
//
// ⚠️ **The false does not tell three states apart and is not meant to** — Picked,
// Arranging, Done and Cancelled are what a caller asks, and they exist as
// separate questions because only one of them has two squads to field.
//
// A loadout's owner is derived rather than stored: it is whoever took the pick
// the draft is waiting on, and storing the seat beside the pick would be a
// second statement of one fact.
func (d *Draft) Turn() (wire.Seat, wire.DraftStep, bool) {
	switch {
	case d.Cancelled() || d.Picked():
		return "", "", false
	case d.awaiting:
		return d.seatAt(d.picks - 1), wire.StepLoadout, true
	case d.bans < 2*BansPerSide(d.format):
		return d.seatAt(d.bans), wire.StepBan, true
	default:
		return d.seatAt(d.picks), wire.StepPick, true
	}
}

// seatAt is whose turn the n-th decision of a stage is: the seat that went
// first, then the other, alternating. Both stages alternate from Config.First,
// which is TODO.md § "Ban and pick" (f).
func (d *Draft) seatAt(n int) wire.Seat {
	return seats[(d.first+n)%seatCount]
}

// Picked reports whether the ban-and-pick was played out: every ban spent or
// skipped, every pick taken and every pick's loadout chosen.
//
// ⚠️ **This is what Done used to mean, and the rename was taken rather than
// left.** Adding the arrange phase gave "is this draft over" two answers, and
// the dangerous one is the one a caller reaches for before fielding a squad — so
// Done is the whole thing now (picking *and* arrangement, the state in which
// Squads answers) and this is the narrower half it was split from. What is
// written against Picked is exactly what was written against the old Done: Turn
// closes on it, Arranging opens on it, and Picks is what it produces. The
// alternative — leaving Done alone and adding a third name for the whole — was
// less churn and left Done naming a draft nothing can field.
//
// A cancelled draft is **not** picked and not done. It has no rosters, so a
// caller that treated them as one would field a side nobody picked.
func (d *Draft) Picked() bool {
	return !d.Cancelled() && !d.awaiting &&
		d.bans == 2*BansPerSide(d.format) &&
		d.picks == 2*PicksPerSide(d.format)
}

// Done reports whether the **whole** draft is over: the ban-and-pick played out
// and both sides arranged. It is the state in which Squads answers, which is the
// question a caller asking this is about to ask next.
//
// A cancelled draft is not done, for the reason Picked gives.
func (d *Draft) Done() bool { return d.Picked() && d.arrangedBoth() }

// Cancelled reports whether the draft was abandoned, which today happens for
// exactly one reason: an allowance ran out. → TimedOut.
func (d *Draft) Cancelled() bool { return d.abandoned != "" }

// Candidates is what the open decision may choose from: the pool minus every
// character already banned or picked, in the pool's own order.
//
// ⚠️ **It is not a convenience.** With every ban spent the final pick of a 5v5
// sees exactly `slack + 1` candidates, and slack was **nought** on the shipped
// cast when this was written — a list of one, live behaviour rather than a
// hypothetical. `pokemon.pichu` took it to one on 2026-09-05, so the last pick is
// a decision again; the shape stays because the next format or the next hidden
// character puts it back, and a screen owes a player the difference between
// choosing and being told either way. → TODO.md's slack note, and
// TestTheLastPickOfAFiveASideChoosesFromSlackPlusOne.
//
// A loadout chooses a form and a kit rather than a character, so it has no
// candidates and the answer is **nil**. Nil is also the answer when nothing is
// due at all, and it is a different answer from an empty list: an empty
// candidate list would mean a decision with nothing to take, which step 1 proved
// unreachable for a draft that Fits allowed.
func (d *Draft) Candidates() []cast.Character {
	_, step, due := d.Turn()
	if !due || step == wire.StepLoadout {
		return nil
	}
	out := make([]cast.Character, 0, len(d.pool.characters))
	for _, character := range d.pool.characters {
		if _, gone := d.spentBy(character.ID); gone {
			continue
		}
		out = append(out, character)
	}
	return out
}

// Ban takes a character out of the pool for the whole match.
//
// A ban lasts the match and the first cut is bo1 only — TODO.md § "Ban and pick"
// (d) — so nothing here has a series in it.
func (d *Draft) Ban(seat wire.Seat, characterID string) error {
	if err := d.due(seat, wire.StepBan); err != nil {
		return err
	}
	if _, err := d.offered(characterID, wire.StepBan); err != nil {
		return err
	}
	d.spent = append(d.spent,
		spending{character: characterID, seat: seat, step: wire.StepBan})
	d.bans++
	d.record(wire.DraftEntry{Seat: seat, Step: wire.StepBan, Character: characterID})
	return nil
}

// SkipBan spends a ban slot without banning anybody.
//
// ⚠️ **A skipped ban leaves the character in the pool because it names none** —
// bans are optional, per TODO.md § "Ban and pick" (b), and a skip takes **nought**
// characters out. That is the direction that matters: optionality can only ever
// leave the pool fuller than Fits measured, which is why it cannot turn a legal
// room into a draft that runs dry.
func (d *Draft) SkipBan(seat wire.Seat) error {
	if err := d.due(seat, wire.StepBan); err != nil {
		return err
	}
	d.bans++
	// No character, which is what a skip is. The record says the slot was spent
	// and names nobody, and a replay reads the absence as the decision.
	d.record(wire.DraftEntry{Seat: seat, Step: wire.StepBan})
	return nil
}

// Pick takes a character for this side. It is the **first** of the two decisions
// a pick is made of: the character leaves the pool now and the loadout follows,
// which is why Turn asks for a loadout immediately afterwards and refuses
// everything else until it arrives.
//
// The split is forced rather than chosen. cast.Character.SkillsAt cannot be
// asked without a form, so the form belongs to the loadout — and the on-the-spot
// path in decision (a) is a whole screen's worth of choosing, which cannot be a
// single call.
func (d *Draft) Pick(seat wire.Seat, characterID string) error {
	if err := d.due(seat, wire.StepPick); err != nil {
		return err
	}
	at, err := d.offered(characterID, wire.StepPick)
	if err != nil {
		return err
	}
	index, _ := indexOf(seat)
	d.spent = append(d.spent,
		spending{character: characterID, seat: seat, step: wire.StepPick})
	d.taken[index] = append(d.taken[index], Pick{
		Character: characterID,
		Level:     progression.LevelCap,
	})
	d.picks++
	d.awaiting = true
	d.pending = at
	d.record(wire.DraftEntry{Seat: seat, Step: wire.StepPick, Character: characterID})
	return nil
}

// Loadout is the second half of a pick: the form the character fields, four of
// the skills it knows at the cap as that form, and one trait.
//
// form is progression.Furthest — the empty string — for "the furthest the cap
// reaches", which is what a line that does not fork means and what every
// placement meant before one could choose. ⚠️ **On a line that forks there is no
// such thing**, and pokemon.poliwag is in the shipped pool: it reaches both
// Poliwrath and Politoed at the cap, so an unnamed form has no answer, the
// refusal is progression's own, and the resulting placement would not be
// fieldable at all.
//
// ⚠️ **The legality of the kit is cast.ChooseLoadout's answer and its error is
// passed through unreworded.** That function is this repository's single
// declaration of "may this unit bring that skill" — its own comment records it
// having existed three times before it was one — and a state machine phrasing
// its own refusal here would be the fourth. → CLAUDE.md § "One rule, one
// declaration". The form's resolution gets a lead-in naming the pick and keeps
// progression's own words behind it, which is exactly what cast.resolveBuild
// does with the same call.
func (d *Draft) Loadout(seat wire.Seat, form string, skills, passives []string) error {
	if err := d.due(seat, wire.StepLoadout); err != nil {
		return err
	}
	// The pool position rather than a second lookup by id, so there is no "the
	// character has left the pool" branch to write: the pool is fixed for the
	// whole of a draft and this index was taken out of it by the pick this
	// loadout belongs to.
	character := d.pool.characters[d.pending]
	_, stage, err := character.Resolve(progression.LevelCap, form)
	if err != nil {
		return fmt.Errorf("%s's pick of %s at level %d: %w",
			seat, character.ID, progression.LevelCap, err)
	}
	chosenSkills, chosenPassives, err := cast.ChooseLoadout(
		fmt.Sprintf("%s's pick of %s", seat, character.ID),
		skills, passives, character, progression.LevelCap, stage.Name)
	if err != nil {
		return err
	}
	index, _ := indexOf(seat)
	open := &d.taken[index][len(d.taken[index])-1]
	open.Stage = stage.Name
	open.Skills = chosenSkills
	open.Passives = chosenPassives
	d.awaiting = false
	// ⚠️ The record keeps what was **named**, not what it resolved to: `form` and
	// not stage.Name. An entry holding the resolved form would be a second
	// statement of something the replay computes, and the one place two peers
	// could disagree. → wire.DraftDecision.Stage.
	d.record(wire.DraftEntry{
		Seat:     seat,
		Step:     wire.StepLoadout,
		Stage:    form,
		Skills:   slices.Clone(skills),
		Passives: slices.Clone(passives),
	})
	return nil
}

// TimedOut tells the draft that the allowance for the open decision ran out, and
// **cancels the whole draft**.
//
// ⚠️ It is an **input**, exactly as internal/room's is: this package does not
// know how long anything took and does not ask, so nothing here reads a clock
// and a replay cannot tell a timed-out draft from any other by anything except
// the entry that says so.
//
// ⚠️ **There is no auto-pick and no default**, which is TODO.md § "Ban and pick"
// (c) and the one place the design does not follow "a timeout announces and
// passes": a side that never picked has no squad to fight with, so the match
// starts over from a new code. A defaulted pick would hand somebody a squad they
// did not choose and call it theirs.
//
// A timeout for a seat the draft is not asking is **refused**, and what the
// refusal protects is the decision: a transport reporting a spurious timeout
// must not cancel a draft on behalf of a seat nobody is waiting on.
//
// ⚠️ **It covers the arrange phase too, where the seat it may name is any seat
// that has not arranged rather than the one on turn** — both are being asked at
// once, so both have an allowance running. A seat that has already arranged is
// refused for the reason above: it has answered, so there is nothing of its own
// left to run out.
func (d *Draft) TimedOut(seat wire.Seat) error {
	if d.Cancelled() {
		return fmt.Errorf("this draft was already cancelled when %s ran out of time, so a "+
			"second timeout has nothing to end", d.abandoned)
	}
	if d.Arranging() {
		index, seated := indexOf(seat)
		switch {
		case !seated:
			return fmt.Errorf("%q is not one of the two seats a room hands out, so it has no "+
				"arrangement to be waited on and that timeout cancels nothing", seat)
		case len(d.arranged[index]) > 0:
			return fmt.Errorf("%s has already arranged, so it has no allowance left to run out: "+
				"this draft is waiting on %s", seat, wordSeats(d.AwaitingArrangement()))
		}
		d.abandon(seat)
		return nil
	}
	onTurn, open, due := d.Turn()
	switch {
	case !due:
		return fmt.Errorf("this draft is finished, so there is no open decision for an " +
			"allowance to run out on")
	case seat != onTurn:
		return fmt.Errorf("the draft is waiting on %s to %s and not on %q, so that timeout "+
			"cancels nothing", onTurn, open, seat)
	}
	d.abandon(seat)
	return nil
}

// abandon cancels the draft and records the timeout that did it, which is the
// one way a draft ends without being played out.
//
// ⚠️ **The behavioural claim is that the record gets the timeout and nothing
// else** — a buffered arrangement is not appended on the way out, because doing
// so would leak exactly what the buffer exists to hide, to a draft that is being
// thrown away anyway. That claim is held by this function only ever recording
// one entry, and TestATimeoutInTheArrangePhaseDiscardsWhatItHeld reads the
// record for it.
//
// ⚠️ **Clearing the buffer is a release rather than a guard, and it is measured
// as unobservable** — deleting the loop below leaves the whole suite green, and
// nothing here is relying on it: every accessor that could answer from the
// buffer gates on Cancelled or Picked first, so a cancelled draft reports
// nothing to arrange, nothing awaited and no squads whether the cells are still
// held or not. It stays because "the arrangement is discarded" should be true of
// the state and not only of the record — an accessor added later must not be able
// to answer out of a draft that was thrown away — and it is written down as
// unobservable so the next reader does not go looking for the test that holds it.
func (d *Draft) abandon(seat wire.Seat) {
	d.abandoned = seat
	for index := range d.arranged {
		d.arranged[index] = nil
	}
	d.record(wire.DraftEntry{Seat: seat, Step: wire.StepTimeout})
}

// Picks is what the ban-and-pick produces: each side's characters and the
// loadouts they were taken in. → Picked, which is the state in which it is
// complete, and Squads, which is these picks with a cell each.
//
// Indexed by seat in the order a room hands seats out — **[0] is wire.SeatHost
// and [1] is wire.SeatGuest**, whichever of the two decided first. It is an
// array rather than a map for the reason seatCount is an array; the type is
// `[2][]Pick` because seatCount is two.
//
// ⚠️ **This is deliberately not a pair of squads**, and Pick's own comment says
// why and what the missing half is: the slot is the arrange phase's decision and
// Squads is where the two meet. Picks stays the shape it is because the *order*
// here is the order the picks were taken, which is what Arrange's slice is
// indexed by — `slots[i]` is the cell for `Picks()[seat][i]`.
//
// It is safe to read mid-draft and answers what has been taken so far, including
// a pick whose loadout is still open — a pick with no skills, which Turn is what
// reports. Copies out, so a caller cannot edit the draft's own picks.
func (d *Draft) Picks() [seatCount][]Pick {
	var out [seatCount][]Pick
	for index, side := range d.taken {
		out[index] = make([]Pick, 0, len(side))
		for _, one := range side {
			out[index] = append(out[index], one.clone())
		}
	}
	return out
}

// due is the whole of "may this seat make this decision now", and every refusal
// it hands back is a sentence saying what cannot happen and why.
func (d *Draft) due(seat wire.Seat, step wire.DraftStep) error {
	onTurn, open, anyDue := d.Turn()
	switch {
	case d.Cancelled():
		return fmt.Errorf("this draft was cancelled when %s ran out of time, so %q cannot %s: "+
			"a draft that runs out of time is not resumed, it is played again from a new room "+
			"code", d.abandoned, seat, step)
	// The picking being over is two different states now, and they get two
	// different sentences: one has a decision open and the other has none.
	case d.Arranging():
		return fmt.Errorf("the picking is over — every ban is spent or skipped and every pick "+
			"has its loadout — and what is open now is the arrangement, so %q cannot %s: the two "+
			"sides arrange at once, which Arrange takes and Turn does not answer for", seat, step)
	case !anyDue:
		return fmt.Errorf("this draft is finished — every ban is spent or skipped, every pick "+
			"has its loadout and both sides have arranged — so %q cannot %s", seat, step)
	case !seat.Valid():
		return fmt.Errorf("%q is not one of the two seats a room hands out, so it has no "+
			"decision to make in this draft", seat)
	// A second loadout for one pick, told apart from a plain out-of-turn call
	// because the two are different mistakes: this one is a caller answering a
	// question it has already answered, and the advice is that a loadout is
	// chosen once.
	case step == wire.StepLoadout && d.settled(seat):
		return fmt.Errorf("%s's pick of %s already has its loadout, and a pick's loadout is "+
			"chosen once: it is %s's turn to %s now", seat, d.lastOf(seat).Character, onTurn, open)
	case seat != onTurn:
		return fmt.Errorf("it is %s's turn to %s, so %q cannot act out of turn", onTurn, open, seat)
	case open != step:
		return d.wrongStep(seat, open, step)
	}
	return nil
}

// wrongStep refuses a decision of the wrong kind from the seat whose turn it
// genuinely is, and each arm carries the reason the sequence is what it is.
func (d *Draft) wrongStep(seat wire.Seat, open, wanted wire.DraftStep) error {
	switch {
	case open == wire.StepBan && wanted == wire.StepPick:
		return fmt.Errorf("%s is due to ban and not to pick: every ban is taken before any "+
			"pick, so that a ban can still deny one (TODO.md § \"Ban and pick\" (f)), and %d of "+
			"the %d ban slots are still open",
			seat, 2*BansPerSide(d.format)-d.bans, 2*BansPerSide(d.format))
	case open == wire.StepPick && wanted == wire.StepBan:
		return fmt.Errorf("the banning stage is over, so %s cannot ban: a ban is only worth "+
			"spending while it can still deny a pick, and picking has begun", seat)
	case open == wire.StepLoadout:
		return fmt.Errorf("%s has just picked %s and owes it a loadout, so nothing else can "+
			"be decided until the form, the skills and the trait are chosen",
			seat, d.lastOf(seat).Character)
	default:
		return fmt.Errorf("no pick of %s's is waiting for a loadout: %s is due to %s",
			seat, seat, open)
	}
}

// offered is "may a ban or a pick name this character", and the pool position of
// it when they may.
//
// ⚠️ **This is where the pool is spent, and it is why a drafted squad cannot
// double up**: a character that has been banned or picked by either side is out
// of one shared pool, so a drafted side is six or ten *different* characters by
// construction. CLAUDE.md's "one squad may field the same character twice" is
// about a **saved** squad and both statements hold — see Pick.
func (d *Draft) offered(characterID string, step wire.DraftStep) (int, error) {
	at := slices.IndexFunc(d.pool.characters, func(character cast.Character) bool {
		return character.ID == characterID
	})
	if at < 0 {
		return 0, fmt.Errorf("%q is not in this draft's pool, so a %s cannot name it: the pool "+
			"is the cast minus every character held back, and it is fixed for the whole of a "+
			"draft", characterID, step)
	}
	if gone, taken := d.spentBy(characterID); taken {
		return 0, fmt.Errorf("%q is out of this draft already: %s took it with a %s",
			characterID, gone.seat, gone.step)
	}
	return at, nil
}

// spentBy is how a character left the pool, and whether it has.
func (d *Draft) spentBy(characterID string) (spending, bool) {
	for _, one := range d.spent {
		if one.character == characterID {
			return one, true
		}
	}
	return spending{}, false
}

// settled reports whether this seat's most recent pick already has its loadout,
// which is what makes a second loadout for it a second answer to one question.
func (d *Draft) settled(seat wire.Seat) bool {
	index, seated := indexOf(seat)
	if !seated || len(d.taken[index]) == 0 {
		return false
	}
	return len(d.taken[index][len(d.taken[index])-1].Skills) > 0
}

// lastOf is this seat's most recent pick, and the zero Pick for a seat that has
// taken none — which is a state only a refusal reads, and it names nobody
// rather than nobody in particular.
func (d *Draft) lastOf(seat wire.Seat) Pick {
	index, seated := indexOf(seat)
	if !seated || len(d.taken[index]) == 0 {
		return Pick{}
	}
	return d.taken[index][len(d.taken[index])-1]
}
