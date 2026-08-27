package skill

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// Summon is a unit a skill puts on the board.
//
// It is the first thing a skill does that is neither damage, healing nor a
// status: everything else a skill can do happens *to* somebody already standing
// there, and this adds somebody. The engine half of that is in battle.summon;
// what is here is the declaration and every rule that can be checked without a
// battle in front of it.
//
// # Why the stat line has three spellings and not one
//
// A clone and a summoned creature are different things wearing the same
// mechanism. A clone is a copy, so it is written as a share of whoever made it
// and moves when that unit is retuned. A creature is its own animal — a toad
// does not get bigger because the ninja levelled — so it is written as a stat
// line of its own. Forcing one spelling would mean either a creature that
// scales off somebody it has nothing to do with, or a clone that has to be
// re-authored every time its caster's curve moves.
//
// The two shares differ in *when* they read. Share is the caster's stats as they
// stand, so a caster that buffed itself first makes a better copy — which is the
// same freeze every damage-over-time in this engine takes, and reads as the
// clone being a copy of what was there. ShareOfBase ignores every timed effect,
// which is what an author reaches for when a copy that can be pre-buffed is an
// exploit rather than a play.
type Summon struct {
	// Count is how many units the skill puts down. Fewer arrive when the side
	// has no room, which is a fact about the board rather than about the skill.
	Count int
	// Name is what the summoned units are called on screen. They are numbered
	// from it, so a name is worth having: three units all reading "clone" are
	// three rows a player cannot tell apart.
	Name string
	// Share and ShareOfBase are parts per thousand of the caster's stats, and
	// exactly one of the three stat spellings is set.
	Share       int
	ShareOfBase int
	// Stats is a fixed line, for a summon that is its own creature.
	Stats *progression.Values
	// Affinity is the summoned unit's element by name, or empty to take the
	// caster's. A clone shares its caster's element; a toad need not.
	Affinity string
	// Skills is what the summoned unit knows. It may not include a skill that
	// summons: see WhySummonCannotCarry.
	Skills []string
	// Lasts is how many of the summoned unit's **own** turns it stays for, or
	// nought for one that stays until something kills it.
	//
	// Its own turns rather than the battle's, for the reason a cooldown counts
	// the caster's: a slow summon and a fast one given "three turns" should each
	// get three, and counting the battle's would hand the fast one more of them.
	Lasts int
	// Bound says the summoned unit leaves when whoever summoned it dies.
	//
	// A per-skill flag rather than a rule, because the two things this mechanism
	// carries answer it differently: a clone is an extension of its caster and
	// goes when the caster goes, while a creature that was called up is its own
	// and stays to finish the fight.
	Bound bool
}

// Summons reports whether the skill puts anything on the board. A nil receiver
// answers false, so a caller never has to check for one.
func (s *Summon) Summons() bool { return s != nil && s.Count > 0 }

// MaxSummonCount bounds what one cast may put down.
//
// It is the formation's own size rather than a number of its own: a side cannot
// hold more than a full team however many a skill asks for, so a larger figure
// could only ever be an author asking for something the board will refuse
// silently. Refusing it here says so while the file is still in front of them.
const MaxSummonCount = 5

// resolveSummon checks a declared summon and returns it.
//
// The skill ids it names are checked against the book one layer up, in the same
// place a character's kit is: this package parses a skill without the finished
// book existing yet, so a name here can only be held and looked at later.
func resolveSummon(declared *summonFile, fail func(string, ...any) error) (*Summon, error) {
	if declared == nil {
		return nil, nil
	}
	spellings := 0
	for _, given := range []bool{declared.Share > 0, declared.ShareOfBase > 0, declared.Stats != nil} {
		if given {
			spellings++
		}
	}
	switch {
	case spellings == 0:
		return nil, fail("summons without saying what the summoned unit is made of; " +
			"give it a share, a share_of_base or a stats line")
	case spellings > 1:
		return nil, fail("summons with more than one stat line; " +
			"share, share_of_base and stats are three spellings of one thing")
	}
	for _, share := range []struct {
		name   string
		amount int
	}{{"share", declared.Share}, {"share_of_base", declared.ShareOfBase}} {
		if share.amount == 0 {
			continue
		}
		if share.amount < 0 || share.amount > scale.Base {
			return nil, fail("summons at a %s of %d, want 1 to %d",
				share.name, share.amount, scale.Base)
		}
	}
	count := declared.Count
	if count == 0 {
		count = 1
	}
	if count < 1 || count > MaxSummonCount {
		return nil, fail("summons %d units, want 1 to %d", count, MaxSummonCount)
	}
	if len(declared.Skills) == 0 {
		return nil, fail("summons a unit that knows no skill, so it would stand there")
	}
	seen := make(map[string]bool, len(declared.Skills))
	for _, id := range declared.Skills {
		if seen[id] {
			return nil, fail("summons a unit that knows %q twice", id)
		}
		seen[id] = true
	}
	if declared.Lasts < 0 {
		return nil, fail("summons a unit for %d turns, want nought or more", declared.Lasts)
	}
	return &Summon{
		Count: count, Name: declared.Name,
		Share: declared.Share, ShareOfBase: declared.ShareOfBase, Stats: declared.Stats,
		Affinity: declared.Affinity,
		Skills:   append([]string(nil), declared.Skills...),
		Lasts:    declared.Lasts, Bound: declared.Bound,
	}, nil
}

// WhySummonsCannotRecurse is the rule a book can only check once every skill in
// it is parsed: a summoned unit may not know a skill that summons.
//
// Without it a single cast is an unbounded one. The board would stop it in
// practice — a side holds five and no more — but "it runs out of room" is not a
// rule anybody can read off the file, and a summon whose skills are refused at
// the moment the room appears is worse than one refused while it is written.
func WhySummonsCannotRecurse(book *Book, declared Skill) error {
	if !declared.Summons.Summons() {
		return nil
	}
	for _, id := range declared.Summons.Skills {
		known, err := book.Lookup(id)
		if err != nil {
			return fmt.Errorf("skill %q summons a unit knowing %q: %w", declared.ID, id, err)
		}
		if known.Summons.Summons() {
			return fmt.Errorf(
				"skill %q summons a unit knowing %q, which summons as well; "+
					"a summon that summons has no end but the board running out of room",
				declared.ID, known.ID)
		}
	}
	return nil
}
