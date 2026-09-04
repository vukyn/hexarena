package draft

import (
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/cast"
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

// Step is which kind of decision a draft is asking for, and it doubles as the
// vocabulary of what a draft's record can hold.
//
// It is a named string rather than an iota for the reason wire.Seat is: a string
// type has no declaration order to reinterpret, so an insertion cannot silently
// change what an existing value means, and it needs no names table, no
// MarshalJSON and no count to be held against. Nothing here is dispatched on by
// index, which is the case that forces an enum.
//
// The zero Step is not a step. Turn reports whether anything is due with a bool
// rather than with a zero value, so an absent step means what it says.
type Step string

const (
	// StepBan is a side taking a character out of the pool for the match. It is
	// **optional** — see SkipBan — and it comes before every pick.
	StepBan Step = "ban"
	// StepPick is a side taking a character for itself. It is the first of the
	// two decisions a pick is made of; the character leaves the pool here.
	StepPick Step = "pick"
	// StepLoadout is the second: the form, four skills and one trait the
	// character just picked will field. It belongs to whoever took that pick and
	// nothing else can be decided until it is in.
	StepLoadout Step = "loadout"
	// StepTimeout is not a decision anybody is due to make and **Turn never
	// answers it**: it is the transport reporting that an allowance ran out,
	// which per TODO.md § "Ban and pick" (c) cancels the whole draft.
	//
	// It lives in this vocabulary rather than in one of its own because it is one
	// of the things a draft's record holds, and a second type beside Step would be
	// two vocabularies for one idea — the mistake CLAUDE.md keeps a list of.
	StepTimeout Step = "timeout"
)

// Pick is one character a side took, with the loadout it took it in: a
// placement.Placement **minus its Slot and its ID**.
//
// ⚠️ **The slot is not a draft decision, and this type is that decision made.**
// TODO.md § "Ban and pick" (g): once the draft closes both sides arrange their
// own three (or five) **privately and simultaneously**, which is a different
// shape from this one — two decisions pending at once, and each side's secret
// until both are in — and it is step 2b. So there is deliberately no
// `Squads() [2]placement.Squad` here: hex.Offset's zero value is a real cell
// (the ally back corner, exactly the trap wire.Act documents for its Aim), so a
// squad handed out with Slot left at its zero would *look* authored, and what
// happens to it is worse than a visible gap — placement.Squad.Validate refuses
// the second unit standing where the first already is, so the squad is turned
// away at the moment it is fought, naming a cell nobody chose. An output that
// cannot be used is worse than an output that is honestly incomplete.
//
// **What turning these into squads still needs**, so a reader does not have to
// work it out: a Slot per pick from the arrange phase, and an ID unique within
// the side (placement.Placement.ID is per squad, and Squad.Take prefixes the
// side, so a draft has nothing to invent there). Level and Stage are already
// resolved here, so nothing else is owed.
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

	// abandoned is the seat whose allowance ran out, and empty for a draft that
	// was not cancelled. The zero wire.Seat already means "no seat", so this is
	// one statement rather than a bool beside a name that could disagree with it.
	abandoned wire.Seat

	entries []Entry
}

// spending is one character out of the pool: which, by whom, and as what.
type spending struct {
	character string
	seat      wire.Seat
	step      Step
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
// draft that has finished, and one that was cancelled.
//
// ⚠️ **The false does not tell those two apart and is not meant to** — Done and
// Cancelled are what a caller asks, and they exist as two questions because a
// finished draft has two rosters to field and an abandoned one has none.
//
// A loadout's owner is derived rather than stored: it is whoever took the pick
// the draft is waiting on, and storing the seat beside the pick would be a
// second statement of one fact.
func (d *Draft) Turn() (wire.Seat, Step, bool) {
	switch {
	case d.Cancelled() || d.Done():
		return "", "", false
	case d.awaiting:
		return d.seatAt(d.picks - 1), StepLoadout, true
	case d.bans < 2*BansPerSide(d.format):
		return d.seatAt(d.bans), StepBan, true
	default:
		return d.seatAt(d.picks), StepPick, true
	}
}

// seatAt is whose turn the n-th decision of a stage is: the seat that went
// first, then the other, alternating. Both stages alternate from Config.First,
// which is TODO.md § "Ban and pick" (f).
func (d *Draft) seatAt(n int) wire.Seat {
	return seats[(d.first+n)%seatCount]
}

// Done reports whether the draft was played out: every ban spent or skipped,
// every pick taken and every pick's loadout chosen.
//
// A cancelled draft is **not** done. It has no rosters, so a caller that treated
// the two as one would field a side nobody picked.
func (d *Draft) Done() bool {
	return !d.Cancelled() && !d.awaiting &&
		d.bans == 2*BansPerSide(d.format) &&
		d.picks == 2*PicksPerSide(d.format)
}

// Cancelled reports whether the draft was abandoned, which today happens for
// exactly one reason: an allowance ran out. → TimedOut.
func (d *Draft) Cancelled() bool { return d.abandoned != "" }

// Candidates is what the open decision may choose from: the pool minus every
// character already banned or picked, in the pool's own order.
//
// ⚠️ **It is not a convenience.** Slack is nought at 5v5 on the shipped cast, so
// with every ban spent the final pick of a 5v5 sees exactly **one** candidate —
// live behaviour rather than a hypothetical — and a screen owes a player the
// difference between choosing and being told. → TODO.md's slack note, and
// TestTheLastPickOfAFiveASideChoosesFromOne.
//
// A loadout chooses a form and a kit rather than a character, so it has no
// candidates and the answer is **nil**. Nil is also the answer when nothing is
// due at all, and it is a different answer from an empty list: an empty
// candidate list would mean a decision with nothing to take, which step 1 proved
// unreachable for a draft that Fits allowed.
func (d *Draft) Candidates() []cast.Character {
	_, step, due := d.Turn()
	if !due || step == StepLoadout {
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
	if err := d.due(seat, StepBan); err != nil {
		return err
	}
	if _, err := d.offered(characterID, StepBan); err != nil {
		return err
	}
	d.spent = append(d.spent, spending{character: characterID, seat: seat, step: StepBan})
	d.bans++
	d.record(Entry{Seat: seat, Step: StepBan, Character: characterID})
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
	if err := d.due(seat, StepBan); err != nil {
		return err
	}
	d.bans++
	// No character, which is what a skip is. The record says the slot was spent
	// and names nobody, and a replay reads the absence as the decision.
	d.record(Entry{Seat: seat, Step: StepBan})
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
	if err := d.due(seat, StepPick); err != nil {
		return err
	}
	at, err := d.offered(characterID, StepPick)
	if err != nil {
		return err
	}
	index, _ := indexOf(seat)
	d.spent = append(d.spent, spending{character: characterID, seat: seat, step: StepPick})
	d.taken[index] = append(d.taken[index], Pick{
		Character: characterID,
		Level:     progression.LevelCap,
	})
	d.picks++
	d.awaiting = true
	d.pending = at
	d.record(Entry{Seat: seat, Step: StepPick, Character: characterID})
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
	if err := d.due(seat, StepLoadout); err != nil {
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
	// could disagree. → Entry.
	d.record(Entry{
		Seat:     seat,
		Step:     StepLoadout,
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
func (d *Draft) TimedOut(seat wire.Seat) error {
	onTurn, open, due := d.Turn()
	switch {
	case d.Cancelled():
		return fmt.Errorf("this draft was already cancelled when %s ran out of time, so a "+
			"second timeout has nothing to end", d.abandoned)
	case !due:
		return fmt.Errorf("this draft is finished, so there is no open decision for an " +
			"allowance to run out on")
	case seat != onTurn:
		return fmt.Errorf("the draft is waiting on %s to %s and not on %q, so that timeout "+
			"cancels nothing", onTurn, open, seat)
	}
	d.abandoned = seat
	d.record(Entry{Seat: seat, Step: StepTimeout})
	return nil
}

// Picks is what a finished draft produces: each side's characters and the
// loadouts they were taken in.
//
// Indexed by seat in the order a room hands seats out — **[0] is wire.SeatHost
// and [1] is wire.SeatGuest**, whichever of the two decided first. It is an
// array rather than a map for the reason seatCount is an array; the type is
// `[2][]Pick` because seatCount is two.
//
// ⚠️ **This is deliberately not a pair of squads**, and Pick's own comment says
// why and what the missing half is: the slot is step 2b's decision.
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
func (d *Draft) due(seat wire.Seat, step Step) error {
	onTurn, open, anyDue := d.Turn()
	switch {
	case d.Cancelled():
		return fmt.Errorf("this draft was cancelled when %s ran out of time, so %q cannot %s: "+
			"a draft that runs out of time is not resumed, it is played again from a new room "+
			"code", d.abandoned, seat, step)
	case !anyDue:
		return fmt.Errorf("this draft is finished — every ban is spent or skipped and every "+
			"pick has its loadout — so %q cannot %s", seat, step)
	case !seat.Valid():
		return fmt.Errorf("%q is not one of the two seats a room hands out, so it has no "+
			"decision to make in this draft", seat)
	// A second loadout for one pick, told apart from a plain out-of-turn call
	// because the two are different mistakes: this one is a caller answering a
	// question it has already answered, and the advice is that a loadout is
	// chosen once.
	case step == StepLoadout && d.settled(seat):
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
func (d *Draft) wrongStep(seat wire.Seat, open, wanted Step) error {
	switch {
	case open == StepBan && wanted == StepPick:
		return fmt.Errorf("%s is due to ban and not to pick: every ban is taken before any "+
			"pick, so that a ban can still deny one (TODO.md § \"Ban and pick\" (f)), and %d of "+
			"the %d ban slots are still open",
			seat, 2*BansPerSide(d.format)-d.bans, 2*BansPerSide(d.format))
	case open == StepPick && wanted == StepBan:
		return fmt.Errorf("the banning stage is over, so %s cannot ban: a ban is only worth "+
			"spending while it can still deny a pick, and picking has begun", seat)
	case open == StepLoadout:
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
func (d *Draft) offered(characterID string, step Step) (int, error) {
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
