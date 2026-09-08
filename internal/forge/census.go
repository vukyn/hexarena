package forge

import (
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// SkillCasts is how often one of a build's skills was actually cast.
//
// A count rather than a share, because the question a census answers is whether
// a slot is used *at all*, and a share of nought is that same nought with a
// division in front of it.
type SkillCasts struct {
	Skill string
	Cast  int
}

// CastCensus is what a build did with its own four slots.
//
// ⚠️ **It exists because a rate cannot say whether a build played its own kit.**
// A build reading a healthy figure may be reading three skills and a dead slot,
// and that is not a hypothetical: the reading RAT-006 was raised on quoted a
// collapse from 725‰ to 110‰ and blamed a summon the rating had never cast —
// re-measured, that summon is uncast in the *winning* kit as well, and what had
// actually changed was a hide the rating took three times a battle for turns it
// should not have spent. Neither fact is visible in a rate. Both are one line of
// a census.
//
// Counted in the build's own kit order and never by ranging a map: this is a
// measurement somebody reads, and Go randomises that order.
type CastCensus struct {
	Build cast.Build
	// Against is the squad that supplied the opposing side and the subject's own
	// companions, and it is on the report because a count without its board is a
	// figure nobody can re-take.
	Against string
	// Seeds is how many battles each arrangement was fought over, so Battles is
	// twice this.
	Seeds   int
	Battles int
	Casts   []SkillCasts
}

// Silent is the skills the build never cast, in kit order.
//
// The whole point of the instrument: a slot that never fires is a slot the
// author is not getting, whether because the rating prices it at nothing, or
// because its fuel is not in the kit beside it, or because something else in the
// kit is always worth more.
func (c CastCensus) Silent() []string {
	out := []string{}
	for _, counted := range c.Casts {
		if counted.Cast == 0 {
			out = append(out, counted.Skill)
		}
	}
	return out
}

// Census counts how often a build casts each of its own skills, standing in a
// squad against that same squad.
//
// ⚠️ **The board is a squad and a duel would be the wrong instrument.** Several
// whole categories are worth nothing at all in a one on one, correctly: a taunt
// denies an aim nobody had a choice about, and a hide has no other ally for its
// `elsewhere` term to point at. Measured on the shipped `squirtle.fortress`,
// `taunt` was cast **0** times in 132 duels and **252** times in the same number
// of squad battles — so a duelling census reports a dead slot on a build that
// plays it every battle, which is the blind board this repository has drawn a
// wrong conclusion from twice.
//
// ⚠️ **Nor is the board a mirror.** The first cut put the subject's own
// character on both sides, which sounds like the cleanest control there is and
// is a third blind board: against a copy of itself a skill can be strictly
// dominated by its own kit-mate, and seven shipped builds read a silent slot
// that every one of them plays against somebody else.
//
// So the subject stands **in** an authored squad, in place of its first member,
// against that squad intact. The companions and the opponents are then the same
// six units on both sides of the reading and cancel; the only difference on the
// board is the subject's own four slots. The squad is named by the caller
// because one board is one matchup however it is chosen — a slot silent here is
// silent against *this* squad, and the rule that walks several of them is the
// caller's.
//
// Both arrangements are fought over the same seeds, for the reason everything
// else in this package is: the roster order decides a tie in the turn queue, so
// one arrangement folds that advantage into the answer.
func (l *Library) Census(build cast.Build, against string, seeds int) (CastCensus, error) {
	if seeds < 1 {
		return CastCensus{}, fmt.Errorf("a census over %d battles counts nothing", seeds)
	}
	squad, err := l.squad(against)
	if err != nil {
		return CastCensus{}, err
	}
	if len(squad.Units) == 0 {
		return CastCensus{}, fmt.Errorf("the squad %q fields nobody", against)
	}
	character, known := l.characters.Get(build.Character)
	if !known {
		return CastCensus{}, fmt.Errorf("the build %q is for %q, which is in no cast book",
			build.ID, build.Character)
	}
	_, stage, err := character.Resolve(progression.LevelCap, build.Stage)
	if err != nil {
		return CastCensus{}, fmt.Errorf("field %s for a census: %w", build.ID, err)
	}
	subject := placement.Placement{
		// Its own id rather than the member's it stands in for: the two sides
		// otherwise hold the same name on two different characters, and the log
		// is what this reading is taken off.
		ID: "subject", Character: build.Character,
		Level: progression.LevelCap, Stage: stage.Name, Slot: squad.Units[0].Slot,
		Skills: slices.Clone(build.Skills), Passives: slices.Clone(build.Passives),
	}
	standing := squad.Clone()
	standing.ID += ".census"
	standing.Units[0] = subject

	report := CastCensus{Build: build, Against: against, Seeds: seeds}
	counts := make(map[string]int, len(build.Skills))
	books := l.Books()
	for _, arrangement := range []struct {
		ally, enemy placement.Squad
		mine        hex.Side
	}{
		{standing, squad, hex.SideAlly},
		{squad, standing, hex.SideEnemy},
	} {
		roster, err := arrangement.ally.Take(hex.SideAlly, l.characters)
		if err != nil {
			return CastCensus{}, fmt.Errorf("census %s: %w", build.ID, err)
		}
		facing, err := arrangement.enemy.Take(hex.SideEnemy, l.characters)
		if err != nil {
			return CastCensus{}, fmt.Errorf("census %s: %w", build.ID, err)
		}
		roster = append(roster, facing...)
		// Take prefixes the side onto every id, which is what tells the subject
		// from the identical body standing opposite it.
		acting := roster[0].ID
		if arrangement.mine == hex.SideEnemy {
			acting = facing[0].ID
		}
		for seed := 1; seed <= seeds; seed++ {
			_, _, events, err := fight(books, roster, arrangement.mine, uint64(seed))
			if err != nil {
				return CastCensus{}, fmt.Errorf("census %s at seed %d: %w", build.ID, seed, err)
			}
			report.Battles++
			for _, event := range events {
				if event.Kind == battle.SkillUsed && event.Actor == acting {
					counts[event.Skill]++
				}
			}
		}
	}
	for _, id := range build.Skills {
		report.Casts = append(report.Casts, SkillCasts{Skill: id, Cast: counts[id]})
	}
	return report, nil
}

// CensusSeeds is how many battles a census fights each arrangement over when
// nobody says otherwise.
//
// Small, because of what a census asserts: "was this slot ever used" needs one
// cast to answer yes, not a confident rate.
//
// ⚠️ **It is not as small as it can be, and the floor was measured rather than
// guessed.** At two seeds `machop.charge` reads `wrecking_swing` silent on all
// four boards — that skill is gated on five stacks of `heft` and `brace` grants
// them a battle at a time, so its gate is only crossed in a battle that runs long
// enough. A gated slot needs board TIME rather than more boards, which is exactly
// what a seed buys and what another opponent does not.
const CensusSeeds = 6

// CensusWalk is a build measured against the authored squads until one of them
// leaves nothing silent, which is the authoring rule itself rather than a
// convenience over Census.
//
// ⚠️ **The walk is the rule and a census is one board.** A slot silent against
// one squad is a fact about that matchup — `split` is uncast against eight of the
// twenty-one characters and cast four hundred times across the rest — so what a
// catalogue has to answer is whether a slot fires *anywhere*. A reader handed a
// single board would take a silent row as a failed rule, and it is not one.
//
// It lives here rather than in the test that first needed it because two callers
// now ask the same question — the catalogue test and `hexforge census` — and a
// rule worded twice is the mistake this repository keeps a list of.
type CensusWalk struct {
	Build cast.Build
	// Taken is every board the build was measured on, in the order the squads are
	// declared, ending at the board that left nothing silent when there was one.
	Taken []CastCensus
	// Silent is the skills that stayed silent on EVERY board walked, which is
	// empty for a build that passes the rule. A slot named here is a slot the
	// author did not get.
	Silent []string
}

// Boards is how many squads the build was measured against.
func (w CensusWalk) Boards() int { return len(w.Taken) }

// Plays reports whether the build cast every skill it names, somewhere.
func (w CensusWalk) Plays() bool { return len(w.Silent) == 0 }

// CensusWalk takes a census against each authored squad in turn and stops at the
// first board that leaves nothing silent.
//
// Stopping early is what keeps the rule affordable: most builds are done after
// the first board, and only the ones with something to explain pay for the rest.
// A caller that wants one named board calls Census directly.
func (l *Library) CensusWalk(build cast.Build, seeds int) (CensusWalk, error) {
	squads := l.Squads()
	if len(squads) == 0 {
		return CensusWalk{}, fmt.Errorf("no squad is authored, so there is no board to stand %s on", build.ID)
	}
	walk := CensusWalk{Build: build}
	for _, against := range squads {
		census, err := l.Census(build, against.ID, seeds)
		if err != nil {
			return CensusWalk{}, err
		}
		walk.Taken = append(walk.Taken, census)
		if walk.Silent = census.Silent(); len(walk.Silent) == 0 {
			return walk, nil
		}
	}
	return walk, nil
}
