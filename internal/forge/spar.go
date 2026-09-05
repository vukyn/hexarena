package forge

import (
	"fmt"
	"sort"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// Books is what a battle is fought under, assembled from the directory.
//
// internal/seed has a function of this name and this shape, and the difference
// between them is the whole reason this one exists: that one reads the copy
// go:embed baked into the binary, and this one reads the files an author is
// editing. A spar run against the embedded copy would keep reporting on the
// character as it was at the last build, which is the one answer an authoring
// tool must never give.
func (l *Library) Books() battle.Books {
	return battle.Books{
		Rules: l.rules, Chart: l.chart, Bounds: l.bounds, Limits: l.limits,
		Patterns: l.patterns, Statuses: l.statuses,
		Skills: l.skills, Passives: l.passives, Bonuses: l.bonuses,
	}
}

// duelSlot is where both duellists stand: the front column, middle row.
//
// ⚠️ The column used to be load-bearing and no longer is. Reach was distance
// from the caster once, so the front column was the only one that asked nothing
// of a kit and any other would have refused a melee character before it had
// swung. Reach is depth into the enemy's half now, and a duel puts exactly one
// unit on each side, so the single occupied rank is at depth one from anywhere:
// every column asks the same of a kit, which is the shortest range a skill can
// declare.
//
// It stays the front column because a duel should look like the board's ordinary
// case, and the middle row for a reason that is still live: it is the row the
// odd-q push does not tilt away from its opposite number, which is what area
// shapes are measured through.
var duelSlot = hex.Offset{Col: hex.FormationCols - 1, Row: hex.Rows / 2}

// sparTurnLimit is when a duel is abandoned. It is the number cmd/hexarena stops
// at, deliberately: a battle this tool calls endless and a battle the game calls
// endless have to be the same battle.
const sparTurnLimit = 4000

// Duellist is one side of a spar — who it is, and what it brought.
//
// The kit is on it rather than looked up again because it is the part a reader
// has to be told. A spar chooses a loadout where a roster insists somebody
// choose one (see seedKit), so the report carries what was fielded and nobody
// ends up reading a win rate for four skills they cannot see.
type Duellist struct {
	ID    string
	Name  string
	Level int
	// Stage is the form fielded, always the furthest the level reaches. A spar
	// does not field an early form: doing that is a trade an author makes
	// deliberately, and measuring one nobody asked for answers a question nobody
	// asked.
	Stage    string
	Affinity element.Affinity
	Stats    progression.Values
	Skills   []string
	Passives []string
}

// Result is how one duel ended, from the challenger's side.
type Result uint8

const (
	// Won, Lost and Drawn are the three ways a duel that finished can read.
	// Drawn is both a stalemate and a mutual kill: they are different events in
	// the engine and the same answer to "which of these two was better".
	Won Result = iota
	Lost
	Drawn
	// Endless is the turn limit reached with both still standing and still
	// acting — two units buffing themselves at each other for ever. It is kept
	// apart from a draw because a draw is an answer and this is the absence of
	// one.
	Endless
)

// Tally is what a set of duels came to, from the challenger's side.
type Tally struct {
	Wins    int
	Losses  int
	Draws   int
	Endless int
}

// Battles is how many duels this tally counts.
func (t Tally) Battles() int { return t.Wins + t.Losses + t.Draws + t.Endless }

// Rate is the challenger's share in parts per thousand, a draw counting half to
// each side. An endless duel is left out of both halves of the fraction: it is
// not a result, and scoring it as one would let a pair that never resolves drag
// a rate towards whichever number the code picked.
//
// Parts per thousand rather than a float, for the reason every other ratio in
// this repository is: the engine has no floats in it, and a tool reporting in
// them would be the only thing here that could disagree with itself about what a
// number is. A tally with nothing decided is zero, which is what its counts say.
func (t Tally) Rate() int {
	fought := t.Wins + t.Losses + t.Draws
	if fought == 0 {
		return 0
	}
	return (2000*t.Wins + 1000*t.Draws) / (2 * fought)
}

// add folds one tally into another.
func (t Tally) add(other Tally) Tally {
	return Tally{
		Wins: t.Wins + other.Wins, Losses: t.Losses + other.Losses,
		Draws: t.Draws + other.Draws, Endless: t.Endless + other.Endless,
	}
}

// Strikes is what one unit's attacking came to over a set of duels.
//
// It is here rather than derived from a rate because a rate cannot say whether
// anything happened. A skill that was never cast, or cast and never landed, has
// a perfectly ordinary-looking win rate beside it — and a figure that prices a
// mechanism which never fired is worth less than no figure at all, because it
// reads as "rated at nought" when it means "not rated". Cast, Landed and
// Critical are what let a report refuse such a row instead of printing it.
//
// Cast counts one per use, Landed counts one per strike that connected, so a
// two-strike skill lands twice per cast at most and the two columns are not
// comparable to each other. Damage is the total dealt, which is the size of what
// the other three are counting.
type Strikes struct {
	Cast     int
	Landed   int
	Critical int
	Damage   int64
}

// add folds one set of strikes into another.
func (s Strikes) add(other Strikes) Strikes {
	return Strikes{
		Cast: s.Cast + other.Cast, Landed: s.Landed + other.Landed,
		Critical: s.Critical + other.Critical, Damage: s.Damage + other.Damage,
	}
}

// Effects is what one unit's skill did over a set of duels other than strike:
// what took hold, what health came back, what was cleared off, what was called
// in.
//
// It is here for the reason Strikes is, and it is the half Strikes could not
// hold. A rate cannot say whether anything happened — but "did anything happen"
// was asked as "did a damaging strike land", and that is the right evidence only
// for a skill whose mechanism *is* damage. A skill of nought power lands no
// strike however well it works, so every buff, debuff, heal, cleanse and summon
// in the book read as a mechanism that never fired. These four counts are the
// same question asked of the rest of the event log.
//
// Applied, Stripped and Summoned count one per event, so a use that put two
// stacks on counts once. Restored counts one per heal that landed, which is not
// one per cast: a restore aimed at a unit already at full health emits nothing
// at all, and that is exactly a row a weighing has to be able to refuse.
type Effects struct {
	Applied  int
	Restored int
	Stripped int
	Summoned int
}

// add folds one set of effects into another.
func (e Effects) add(other Effects) Effects {
	return Effects{
		Applied: e.Applied + other.Applied, Restored: e.Restored + other.Restored,
		Stripped: e.Stripped + other.Stripped, Summoned: e.Summoned + other.Summoned,
	}
}

// Any reports whether anything at all was observed.
func (e Effects) Any() bool { return e != Effects{} }

// fold counts one battle's events onto the challenger, reading only the events
// one unit produced with its own skills.
//
// ⚠️ The unit is named by roster id, and the roster id of the challenger is not
// the same in both halves of a matchup — it is enlisted first in one and second
// in the other. A Damaged event carries Actor and no Side at all, so reading the
// wrong id here reports the *opponent's* strikes under the challenger's name and
// nothing on screen would look wrong. The caller passes the id that follows the
// kit swap; see duel.
//
// A reply is left out. It carries Passive rather than Skill and is what a unit
// *is* rather than what its skill did, so counting it would price a trait under
// a skill's name. A summoned unit acts under its own roster id and is left out
// by the same test.
//
// ⚠️ **A restore is the one effect that cannot be attributed by reading its own
// event**, so it is attributed to the cast in progress instead. The Healed a
// skill's `restores` produces carries neither the skill nor the caster: Actor on
// it is the unit whose health went up, which for an ally-aimed heal is somebody
// else entirely. What the log does say is when each cast began, and a cast
// resolves whole before the next one starts — so the SkillUsed most recently
// walked past is the skill that healed, exactly rather than probably.
func (m *Matchup) fold(events []battle.Event, actor, counting string) {
	// Whose cast is resolving, and which skill it is. Updated on every
	// SkillUsed rather than only on the challenger's, because a restore is
	// credited by comparing against this pair and a stale one would credit the
	// challenger with a heal the opponent cast.
	casting, cast := "", ""
	for _, event := range events {
		if event.Kind == battle.SkillUsed {
			casting, cast = event.Actor, event.Skill
		}
		if restored(event) {
			if casting == actor && (counting == "" || cast == counting) {
				m.Effects.Restored++
			}
			continue
		}
		if event.Actor != actor || event.Passive != "" {
			continue
		}
		if counting != "" && event.Skill != counting {
			continue
		}
		switch event.Kind {
		case battle.SkillUsed:
			m.Strikes.Cast++
		case battle.Damaged:
			m.Strikes.Landed++
			m.Strikes.Damage += event.Amount
			if event.Critical {
				m.Strikes.Critical++
			}
		case battle.StatusApplied:
			m.Effects.Applied++
		case battle.StatusStripped:
			m.Effects.Stripped++
		case battle.Summoned:
			m.Effects.Summoned++
		}
	}
}

// restored reports whether an event is a skill's own `restores` landing.
//
// Three different things raise a unit's health and all three are a Healed: a
// status regenerating on its holder's turn, a drain giving back a share of
// damage dealt, and a skill restoring health outright. Only the third leaves
// both Status and Drained empty — a tick names the status that ticked, and a
// drain always carries the share it took, because a share of nought heals
// nothing and emits no event at all. So the two absences together are an exact
// reading of the log rather than a guess at one.
func restored(event battle.Event) bool {
	return event.Kind == battle.Healed && event.Status == "" && event.Drained == 0
}

// Matchup is what happened over every seed against one opponent, both ways
// round.
//
// Both ways is not thoroughness, it is the measurement. The turn queue breaks a
// tie by enlistment order, so of two units with the same speed the one placed
// first acts first for the whole battle — and that is worth so much that a
// character duelling an identical copy of itself wins about seven times in ten
// from the first slot. A one-way rate would be that advantage plus the
// character, with no way to tell which was which. Swapping the slots and adding
// the halves cancels it exactly, and leaves the two halves on the record so the
// size of what was cancelled is still readable.
type Matchup struct {
	Against Duellist
	// First and Second are the challenger placed before and after its opponent.
	// The slot and the order are one thing here — the ally slot is enlisted
	// first — so the two are swapped together and the pair of them is what
	// balances.
	First  Tally
	Second Tally
	// Turns is the median length of the duels that finished, and zero when none
	// did. A pairing whose win rate is even can still be badly matched — forty
	// turns of nothing is not the same fight as four — and a rate cannot say so.
	Turns int
	// Strikes is **the challenger's**, across both halves, and counts only the
	// skill the caller asked to count.
	//
	// Whose it is has to be written down because nothing in the type says so and
	// nothing on a screen would: the challenger stands in the ally slot for one
	// half and the enemy slot for the other, so a reader — and a fold that took
	// the wrong id — would get the opponent's attacks with no sign of it.
	Strikes Strikes
	// Effects is the same tally for everything that skill did which was not a
	// strike, and is whose and counts what Strikes does. It is a second field
	// rather than four more on Strikes because the two answer to different
	// questions — "how hard did it hit" and "did it work at all" — and a skill of
	// nought power has the second and not the first.
	Effects Effects
	// Mirror marks the row where a character meets itself. It is the control,
	// and it reads as one only because the halves are kept apart: the combined
	// rate of a mirror is exactly even by construction and therefore says
	// nothing, while First.Rate is what moving first is worth to this character
	// and is the number every other row on the screen has to be read against.
	Mirror bool
}

// Total is both halves together, which is the row's actual answer.
func (m Matchup) Total() Tally { return m.First.add(m.Second) }

// Rate is the challenger's balanced share of this matchup, in parts per
// thousand.
func (m Matchup) Rate() int { return m.Total().Rate() }

// Edge is what the first slot was worth in this pairing, in parts per thousand:
// the gap between winning from it and winning from the other one.
//
// It is worth reporting rather than merely cancelling because a large edge means
// the pairing is decided before either unit acts, which is a fact about the two
// kits — a race between two units that both kill in three turns has an enormous
// edge, and two units that both survive twenty have almost none.
func (m Matchup) Edge() int { return m.First.Rate() - m.Second.Rate() }

// SparReport is a character measured against the cast it belongs to.
type SparReport struct {
	Challenger Duellist
	// Seeds is how many duels each half of each row was fought over, so a row
	// counts twice this many.
	Seeds    int
	Matchups []Matchup
}

// Rate is the challenger's share across every opponent, the mirror excluded.
//
// The mirror is left out because it is a control rather than a matchup: a
// balanced mirror is exactly even whatever the character is, so counting it
// drags every answer towards the middle and hides exactly what the report is
// for.
func (r SparReport) Rate() int {
	total := Tally{}
	for _, matchup := range r.Matchups {
		if matchup.Mirror {
			continue
		}
		total = total.add(matchup.Total())
	}
	return total.Rate()
}

// Opponents is how many rows the headline rate was taken from.
func (r SparReport) Opponents() int {
	counted := 0
	for _, matchup := range r.Matchups {
		if !matchup.Mirror {
			counted++
		}
	}
	return counted
}

// Spar measures one character against every character in the book, itself
// included, over the given number of seeds each way.
//
// One duel is a coin toss — the same two units at two seeds can end either way —
// so a screen showing the result of a single battle would be showing noise and
// calling it a finding. Every row here is a rate over seeds 1..n taken twice,
// and n is the caller's, because the honest number depends on how long the
// caller is willing to wait.
//
// The opponent is not chosen either, and that is the design rather than an
// omission. What an author wants to know about a character they have just
// written is whether it belongs beside the ones already written, and the answer
// to that is the whole cast rather than whichever member they thought to compare
// it against.
func (l *Library) Spar(id string, level, seeds int, want string) (SparReport, error) {
	if seeds < 1 {
		return SparReport{}, fmt.Errorf("a spar over %d battles measures nothing", seeds)
	}
	if level < 1 || level > progression.LevelCap {
		return SparReport{}, fmt.Errorf("level %d is outside 1..%d", level, progression.LevelCap)
	}
	character, known := l.characters.Get(id)
	if !known {
		return SparReport{}, fmt.Errorf("no character is called %q", id)
	}
	// A forking SUBJECT is refused rather than picked, in the words StageAt
	// already uses: a spar reports what one character is worth and two arms are
	// two answers. The caller's fix is to name one, which is what want is for.
	challenger, err := l.duellist(character, level, want)
	if err != nil {
		return SparReport{}, err
	}
	report := SparReport{Challenger: challenger, Seeds: seeds}
	books := l.Books()
	for _, other := range l.characters.All() {
		// A forking OPPONENT is fielded once per arm rather than refused. Two
		// arms are two different fights, and a table that dropped one would be a
		// table missing an opponent — the same answer Inspect gives the budget,
		// one row an arm.
		arms, err := other.FurthestAt(level)
		if err != nil {
			return SparReport{}, fmt.Errorf("%s cannot be fielded at level %d: %w",
				other.ID, level, err)
		}
		for _, arm := range arms {
			opponent, err := l.duellist(other, level, arm.Name)
			if err != nil {
				return SparReport{}, fmt.Errorf("%s cannot be fielded at level %d: %w",
					other.ID, level, err)
			}
			// Nothing is counted: a spar reports which character is better, and a
			// strike tally would be four skills added together with no way to tell
			// which of them moved.
			mirror := other.ID == character.ID && opponent.Stage == challenger.Stage
			fought, err := duel(books, challenger, opponent, seeds, mirror, "")
			if err != nil {
				return SparReport{}, fmt.Errorf("%s cannot be measured against %s: %w",
					challenger.ID, opponent.ID, err)
			}
			report.Matchups = append(report.Matchups, fought)
		}
	}
	return report, nil
}

// duellist works out what a character brings to a spar.
func (l *Library) duellist(character cast.Character, level int, want string) (Duellist, error) {
	stats, stage, err := character.Resolve(level, want)
	if err != nil {
		return Duellist{}, err
	}
	skills, passives := seedKit(character, level, stage.Name)
	return Duellist{
		ID: character.ID, Name: character.Name, Level: level, Stage: stage.Name,
		Affinity: character.Element, Stats: stats,
		Skills: skills, Passives: passives,
	}, nil
}

// seedKit is what a character brings when nobody has chosen for it: the first
// cast.SkillSlots skills and the first cast.TraitSlots traits its learnset
// declares that this level and this form allow.
//
// A roster refuses to do this — internal/seed insists a placement name its four,
// because a file choosing four of nine on an author's behalf would never say
// which. That refusal and this choice are not in conflict, and the difference is
// worth being exact about: a roster *is* the conditions of a battle, so it has
// nobody to state them to, while a spar is a measurement, and a measurement
// states its conditions. The Duellist carries the kit for exactly that reason.
//
// Reading declaration order makes that order mean something it did not mean
// before: a learnset is the character's own preference now, first choice first.
// That is deliberate. Every other rule — longest range, highest power, cheapest
// cooldown — is this package inventing an opinion about what a character is for,
// and the author already has somewhere to put theirs.
func seedKit(character cast.Character, level int, stage string) (skills, passives []string) {
	return firstOf(character.SkillsAt(level, stage), cast.SkillSlots),
		firstOf(character.PassivesAt(level, stage), cast.TraitSlots)
}

func firstOf(available []string, slots int) []string {
	if len(available) > slots {
		return available[:slots]
	}
	return available
}

// The two roster ids of a duel. They are fixed rather than built from the
// character ids because a mirror would otherwise place two units with the same
// id, which a battle refuses — and refusing the control would mean the one row
// that tells the first slot's worth from the character's is the row nobody gets.
const (
	firstID  = "first"
	secondID = "second"
)

// duel fights one pairing over seeds 1..n from each slot and counts the endings.
//
// A refusal comes back as an error rather than as a row that could not be
// fought, and that is a claim about the board rather than a shortcut. Both
// duellists stand in the front column, where the nearest opposing slot is one
// cell away; a skill aimed at anybody but its caster must declare a range of at
// least one, and a skill aimed at its caster is aimed at a cell that is always
// occupied. So no legal kit can be unaimable from here, and battle.New cannot
// refuse one pairing while accepting another — what is left it can refuse is a
// fault in the books, which is the same fault in every row.
// TestTheDuelSlotAsksTheLeastOfAKit is what holds that true.
//
// counting names the one skill whose casts, landings and criticals are tallied
// onto the challenger, or is empty to count every skill it used. It is a
// parameter rather than always-everything because the thing a price is taken on
// is one skill, and a total over four of them would move for reasons that have
// nothing to do with the one being moved.
func duel(books battle.Books, challenger, opponent Duellist, seeds int, mirror bool, counting string) (Matchup, error) {
	matchup := Matchup{Against: opponent, Mirror: mirror}
	lengths := make([]int, 0, 2*seeds)

	// The challenger placed first, then placed second. Both halves run the same
	// seeds: a half fought over different seeds from the other would not cancel
	// anything, it would be two measurements of two different things.
	for _, arrangement := range []struct {
		roster []battle.Roster
		// mine is the side the challenger is standing on in this arrangement,
		// because a Result is always read from the challenger.
		mine hex.Side
		// acting is the challenger's roster id in this arrangement. It follows
		// the kit swap rather than the side, because that is what an event's
		// Actor carries — the two halves put the challenger under two different
		// ids and an event says nothing about which side produced it.
		acting string
		into   *Tally
	}{
		{[]battle.Roster{
			place(firstID, challenger, hex.SideAlly),
			place(secondID, opponent, hex.SideEnemy),
		}, hex.SideAlly, firstID, &matchup.First},
		{[]battle.Roster{
			place(firstID, opponent, hex.SideAlly),
			place(secondID, challenger, hex.SideEnemy),
		}, hex.SideEnemy, secondID, &matchup.Second},
	} {
		for seed := 1; seed <= seeds; seed++ {
			result, turns, events, err := fight(books, arrangement.roster, arrangement.mine, uint64(seed))
			if err != nil {
				// The first refusal ends the row. Every seed would refuse for the
				// same reason — a roster is checked before a die is rolled — so
				// fighting the rest would be a slower way to be told once.
				return Matchup{}, err
			}
			matchup.fold(events, arrangement.acting, counting)
			arrangement.into.count(result)
			if result != Endless {
				lengths = append(lengths, turns)
			}
		}
	}
	matchup.Turns = median(lengths)
	return matchup, nil
}

// count files one result.
func (t *Tally) count(result Result) {
	switch result {
	case Won:
		t.Wins++
	case Lost:
		t.Losses++
	case Drawn:
		t.Draws++
	default:
		t.Endless++
	}
}

// place is one duellist as the engine takes it.
//
// Stage rather than Name, the way a placement and a roster entry both resolve
// it: what the engine is handed is the form's stat line, so the name beside it
// in a log has to be the form's too. A Duellist keeps both because a spar's own
// report has a column for each — that is the report's business, not the
// battle's.
func place(id string, who Duellist, side hex.Side) battle.Roster {
	return battle.Roster{
		ID: id, Name: who.Stage, Side: side, Slot: duelSlot,
		Affinity: who.Affinity, Stats: who.Stats,
		Skills: who.Skills, Passives: who.Passives,
	}
}

// fight runs one duel and reports how it ended for the given side, how long it
// took, and everything the battle recorded.
//
// The events come back rather than a count taken here, because the event log is
// the only contract a reader of a battle has and this package is a reader like
// any other: a fight that summed its own strikes would be a second place where
// what a battle did is decided. They are drained once, after the battle has
// finished, so a caller that ignores them pays only for the slice.
func fight(books battle.Books, roster []battle.Roster, mine hex.Side, seed uint64) (Result, int, []battle.Event, error) {
	fought, err := battle.New(books, seed, roster)
	if err != nil {
		return Drawn, 0, nil, err
	}
	fought.Begin()
	turns, err := fought.RunToEnd(sparTurnLimit)
	if err != nil {
		return Drawn, turns, fought.Drain(), err
	}
	events := fought.Drain()
	return outcome(fought, mine), turns, events, nil
}

// median is the middle length, or the lower of the two middles on an even count.
// Zero for nothing, which is what a row with no finished duel reports.
func median(lengths []int) int {
	if len(lengths) == 0 {
		return 0
	}
	sorted := make([]int, len(lengths))
	copy(sorted, lengths)
	sort.Ints(sorted)
	return sorted[(len(sorted)-1)/2]
}
