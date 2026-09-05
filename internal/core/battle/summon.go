package battle

import (
	"strconv"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// summon puts a skill's units on the board, on the caster's side.
//
// # Why it goes through enlist rather than building a Unit
//
// enlist is every rule about what may stand on this board: the slot is a real
// formation slot, the cell is free, the side is not over strength, the stat line
// is inside the progression limits, the skills exist and the unit's element may
// carry them, and the traits are granted before the first wait is computed. A
// summon that built its own Unit would be a second answer to all of that, and
// the first thing to diverge would be the one nobody tests — a clone standing at
// a stat line the limits refuse.
//
// # Why a replay does not need any of this in the log
//
// It is derived. The caster is known, the skill is known, the board is known,
// and the numbering below is a counter on the caster — so re-running the same
// decisions from the same seed puts the same units in the same cells with the
// same ids. That is the whole reason the id is built here and not passed in: an
// id chosen by a caller is a fact a log would have to carry.
func (b *Battle) summon(caster *Unit, known skill.Skill, turn atb.Turn) {
	if !known.Summons.Summons() {
		return
	}
	stats, err := b.summonStats(caster, known.Summons)
	if err != nil {
		return
	}
	affinity, err := b.summonAffinity(caster, known.Summons)
	if err != nil {
		return
	}
	name := known.Summons.Name
	if name == "" {
		name = caster.Name
	}

	perSide, occupied := b.census()
	for _, slot := range b.summonPlaces(caster, known.Summons, perSide, occupied) {
		// The counter never resets, so two copies are never called the same
		// thing even when the second stands exactly where the first did. A cell
		// is reusable and an id is not: the id is what a decision in the log
		// names, and one reused would make two different units the same row.
		caster.Summoned++
		entry := Roster{
			ID:       caster.ID + "#" + strconv.Itoa(caster.Summoned),
			Name:     name + " " + strconv.Itoa(caster.Summoned),
			Side:     caster.Side,
			Slot:     slot,
			Affinity: affinity,
			Stats:    stats,
			Skills:   known.Summons.Skills,
		}
		unit, err := b.enlist(entry, perSide, occupied)
		if err != nil {
			// Refused by the same rules a roster is refused by, and there is
			// nothing sensible to do about it mid-battle: the skill is data and
			// the data is wrong. It is silent here because it is loud where it
			// belongs — TestASummonedUnitObeysTheRosterRules is what catches a
			// skill whose summon could never stand.
			return
		}
		unit.Summoner = caster.ID
		unit.Bound = known.Summons.Bound
		// The caster is a smaller creature now, and this is the only place in the
		// engine that writes a unit's maximum health.
		//
		// ⚠️ **Taken AFTER the copy is enlisted, on purpose.** enlist can refuse —
		// a full side, a taken cell — and it returns above when it does. Charging
		// first would leave a caster that paid for a body the board never gave it,
		// and the refusal is silent by design, so nothing would say so.
		//
		// The current health follows the maximum down by the same amount rather
		// than being clamped to it: clamping would make a wounded caster split for
		// free, since a unit at half health has the headroom to give away and
		// would lose nothing it was using. The floor of one point is the rule
		// spendHealth already applies and for the same reason — a skill may not
		// kill its own user, because Suggest has no term for "and then I am not
		// here".
		if moved := splitHealth(caster.Base[progression.HP], known.Summons.Splits); moved > 0 {
			caster.Base[progression.HP] -= moved
			if caster.Base[progression.HP] < 1 {
				caster.Base[progression.HP] = 1
			}
			caster.HP -= moved
			if caster.HP < 1 {
				caster.HP = 1
			}
			if caster.HP > caster.Base[progression.HP] {
				caster.HP = caster.Base[progression.HP]
			}
			b.emit(Event{
				Kind: Split, At: turn.At, Turn: turn.Number,
				Actor: caster.ID, Target: unit.ID, Skill: known.ID,
				Amount: moved, Remaining: caster.Base[progression.HP],
			})
		}
		// Minus one rather than nought for a summon that stays, because nought
		// is the zero value every roster unit already has and a count that read
		// it as "no turns left" would dismiss the whole board.
		unit.Leaves = -1
		if known.Summons.Lasts > 0 {
			unit.Leaves = known.Summons.Lasts
		}
		b.units = append(b.units, unit)
		b.byID[unit.ID] = unit
		b.emit(Event{
			Kind: Summoned, At: turn.At, Turn: turn.Number,
			Actor: caster.ID, Target: unit.ID, Skill: known.ID,
			Name: unit.Name, Cell: hex.At(unit.Cell), Side: unit.Side,
			Amount: unit.HP,
		})
	}
}

// summonAffinity is the elements a summoned unit holds: the caster's, or the one
// the skill names instead.
//
// A function rather than a few lines inside summon because battle.Suggest reads
// it too, and an element decides every matchup on the board — a rating that
// guessed the caster's element for a toad the skill declares as water would
// price the cast against the wrong half of the chart, and nothing would report
// the disagreement. The same reason conditionTarget is one function.
func (b *Battle) summonAffinity(caster *Unit, declared *skill.Summon) (element.Affinity, error) {
	if declared.Affinity == "" {
		return caster.Affinity, nil
	}
	parsed, err := element.Parse(declared.Affinity)
	if err != nil {
		return element.Affinity{}, err
	}
	return element.Single(parsed)
}

// summonPlaces is the formation slots a cast would actually put copies in, in
// the order it would fill them.
//
// One function for the same reason summonAffinity is one: Suggest prices a cast
// by what it would put on the board, and this is what puts it there. Two
// readings of "how many fit" would let the rating pay for a copy the board has
// no room for, on exactly the boards where the answer matters.
//
// Three things bound it, and the last is the one a caller would forget: the
// skill's own count, the slots nobody is standing in, and the side's strength.
// A side has nine formation slots and may fill five of them, so free slots are
// not the same question as room — and the count is read against perSide as it
// grows, because each copy enlisted is one more unit on that side.
func (b *Battle) summonPlaces(
	caster *Unit, declared *skill.Summon,
	perSide map[hex.Side]int, occupied map[hex.Offset]string,
) []hex.Offset {
	free := b.freeSlots(caster.Side, occupied)
	// Out of room is a fact about the board rather than a fault in the skill, so
	// a cast is not refused and the shortfall is simply fewer units. The log says
	// how many arrived; nothing has to say how many did not, because the board
	// already does.
	room := hex.MaxTeamSize - perSide[caster.Side]
	out := make([]hex.Offset, 0, declared.Count)
	for i := range declared.Count {
		if i >= len(free) || i >= room {
			break
		}
		out = append(out, free[i])
	}
	return out
}

// summonStats is the stat line a summoned unit stands at, in whichever of the
// three spellings the skill used.
//
// A share is truncated once per stat, the way every other share in this engine
// is, and it is read here rather than stored: a copy is a copy of what was there
// when it was made, so the numbers are frozen at the cast exactly as a
// damage-over-time's tick is frozen at its application.
func (b *Battle) summonStats(caster *Unit, declared *skill.Summon) (progression.Values, error) {
	if declared.Stats != nil {
		return *declared.Stats, nil
	}
	from := b.Stats(caster)
	share := declared.Share
	if declared.ShareOfBase > 0 {
		from, share = caster.Base, declared.ShareOfBase
	}
	var out progression.Values
	for _, kind := range progression.Kinds() {
		out[kind] = from[kind] * int64(share) / int64(scale.Base)
	}
	// A summon that is split off its caster takes its health from the caster's
	// maximum instead of from the spelling, which sized every other stat. Read
	// off Base rather than off b.Stats so a buff standing on the caster cannot
	// inflate the copy — what is being divided is the creature, not its mood.
	if declared.Splits > 0 {
		out[progression.HP] = splitHealth(caster.Base[progression.HP], declared.Splits)
	}
	// A summon with no health could not be enlisted and would be killed by the
	// first thing that looked at it, so the floor is one rather than nought —
	// the same floor combat.Rules puts under a damaging hit.
	if out[progression.HP] < 1 {
		out[progression.HP] = 1
	}
	return out, nil
}

// splitHealth is the health one copy takes off a maximum, and it is a function
// so that the two sites reading it cannot drift: summonStats gives it to the
// copy and Battle.summon takes it off the caster. A share that came out of one
// arithmetic and went into another is a body that gained or lost mass on the way
// across.
func splitHealth(maximum int64, share int) int64 {
	return maximum * int64(share) / int64(scale.Base)
}

// census rebuilds what New hands enlist: how many units each side holds and
// which cells are taken.
//
// # Which of the fallen still hold their places
//
// A unit the roster placed does, and a summon does not.
//
// The formation is what a roster wrote down. A side that was authored with three
// units in three named slots is that arrangement for the whole battle, and a
// fourth appearing in a dead comrade's cell would be a placement nobody chose —
// so a corpse keeps its slot and its side has not shrunk as far as
// hex.MaxTeamSize is concerned, which is what the roster means by that number.
//
// A summon was never in that arrangement. It borrowed a slot the formation left
// empty, and when it is gone the slot is empty again, so a skill that calls one
// up every few turns can go on doing it. Counting a departed summon would make
// such a skill die quietly: the shipped formations leave two free slots a side,
// so the third cast of a battle would put nothing down and say nothing about it.
//
// ⚠️ The first version counted both, and the reason written down for it was
// wrong — that removing a corpse would change what everybody can aim at. It would
// not: Battle.occupant already skips the dead, so a corpse is not a target and
// blocks nothing but a place to stand. The reach check that *is* sensitive to
// corpses runs in New, over a roster, before any of this.
//
// Built from the ordered slice, so the maps are lookups and never a source of
// order.
func (b *Battle) census() (map[hex.Side]int, map[hex.Offset]string) {
	perSide := make(map[hex.Side]int, 2)
	occupied := make(map[hex.Offset]string, len(b.units))
	for _, unit := range b.units {
		if unit.Dead && unit.Summoner != "" {
			continue
		}
		perSide[unit.Side]++
		occupied[unit.Cell] = unit.ID
	}
	return perSide, occupied
}

// freeSlots is the formation slots on a side that nobody has ever stood in, in
// board order.
//
// Front column first, then back, and top to bottom within a column.
//
// The order is fixed rather than clever — a summon has no say in where it lands,
// and one a player can predict is one a replay cannot disagree about — but which
// fixed order it is still matters. hex.Place puts the highest formation column
// against the enemy for both sides, so walking columns forward puts a copy where
// a range of one can reach somebody. Walking them backward would drop every
// summon at the far edge, where most kits cannot aim at all, and the mechanism
// would look broken for a reason that is nowhere near it.
func (b *Battle) freeSlots(side hex.Side, occupied map[hex.Offset]string) []hex.Offset {
	free := make([]hex.Offset, 0, hex.MaxTeamSize)
	for col := hex.FormationCols - 1; col >= 0; col-- {
		for row := range hex.Rows {
			slot := hex.Offset{Col: col, Row: row}
			if _, taken := occupied[hex.Place(side, slot)]; taken {
				continue
			}
			free = append(free, slot)
		}
	}
	return free
}

// dismiss takes a summoned unit off the board without killing it.
//
// It is not a death and the log says so with a kind of its own, because the two
// are different things to everything that reads them: a death is somebody being
// beaten, and this is a copy running out of turns or losing the unit it was a
// copy of. A renderer that drew both the same would report a fight going worse
// than it is.
//
// What it shares with a death is the consequence. The unit stops taking turns,
// and a side with nothing left standing has lost whether the last unit was
// killed or simply left — a summon holds a slot and counts, which is the whole
// point of putting one there.
func (b *Battle) dismiss(unit *Unit, reason string) {
	if unit.Dead {
		return
	}
	unit.Dead = true
	unit.HP = 0
	b.queue.Remove(unit.ID)
	b.emit(Event{
		Kind: Left, Actor: unit.ID, Cell: hex.At(unit.Cell), Side: unit.Side, Note: reason,
	})
	b.checkEnd()
}

// dismissBound sends home every summon that was bound to a unit that has just
// died.
//
// In enlistment order, which is the order b.units is in: a clone is dismissed
// before the ones summoned after it, and two runs of the same seed emit the two
// events the same way round.
func (b *Battle) dismissBound(summoner *Unit) {
	for _, unit := range b.units {
		if unit.Bound && unit.Summoner == summoner.ID && !unit.Dead {
			b.dismiss(unit, "summoner died")
		}
	}
}

// expired reports whether a summoned unit has run out of turns, and spends one
// if it has not.
//
// Its own turns, so a slow summon gets as many as a fast one. The count is spent
// at the start of a turn rather than the end, which is what makes "lasts three"
// mean three turns it may act on: the third turn spends the last of them and the
// fourth finds none left.
func (b *Battle) expired(unit *Unit) bool {
	if unit.Summoner == "" || unit.Leaves < 0 {
		return false
	}
	if unit.Leaves == 0 {
		return true
	}
	unit.Leaves--
	return false
}
