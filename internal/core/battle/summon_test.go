package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// calling is a duel where the ally may summon and the foe is slow enough to stay
// out of the way of whatever is being measured.
func calling(t *testing.T, allySkills []string) *battle.Battle {
	t.Helper()
	return callingAgainst(t, allySkills, 20)
}

// callingAgainst is calling with the foe's speed named, for the tests that need
// the foe to get in first and finish the caster off.
func callingAgainst(t *testing.T, allySkills []string, foeSpeed int64) *battle.Battle {
	t.Helper()
	return mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: allySkills},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, foeSpeed),
			Skills: []string{"jab"}},
	})
}

// casts gives the ally its turn with a named skill and returns what it logged.
func casts(t *testing.T, fight *battle.Battle, using string) []battle.Event {
	t.Helper()
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	if err := fight.Act(using, unitByID(t, fight, "a").Cell); err != nil {
		t.Fatalf("act %s: %v", using, err)
	}
	return fight.Drain()
}

// arrivals is every unit a cast put on the board.
func arrivals(events []battle.Event) []battle.Event {
	return find(events, battle.Summoned)
}

// TestASummonPutsAUnitOnTheCastersSide is the feature at its plainest.
func TestASummonPutsAUnitOnTheCastersSide(t *testing.T) {
	fight := calling(t, []string{"copy", "jab"})
	came := arrivals(casts(t, fight, "copy"))
	if len(came) != 1 {
		t.Fatalf("the cast summoned %d units, want one", len(came))
	}
	if came[0].Actor != "a" {
		t.Errorf("the arrival names %q as the caster", came[0].Actor)
	}
	if came[0].Side != hex.SideAlly {
		t.Errorf("the copy stands on the %s side, want its caster's", came[0].Side)
	}
	copied, ok := fight.Unit(came[0].Target)
	if !ok {
		t.Fatalf("the log announced %q and the battle has no such unit", came[0].Target)
	}
	if copied.Summoner != "a" {
		t.Errorf("the copy was summoned by %q, want a", copied.Summoner)
	}
	// The front column, because that is where a range of one can reach anybody:
	// the caster is at 2,1 so the first free slot walking forward is 2,0.
	if want := (hex.Offset{Col: 2, Row: 0}); copied.Cell != want {
		t.Errorf("the copy stands at %s, want the first free front slot %s", copied.Cell, want)
	}
	// And it is a combatant rather than scenery: it takes turns.
	acted := false
	for range 6 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if prompt.Unit == copied.ID && !prompt.Skipped {
			acted = true
			break
		}
		if !prompt.Skipped {
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		fight.Drain()
	}
	if !acted {
		t.Error("the copy never got a turn, so it is on the board and not in the fight")
	}
}

// TestASummonsShareIsFrozenAtTheCast is the same freeze every other share in
// this engine takes.
//
// A copy is a copy of what was there. Reading the caster live would make a clone
// shrink when its caster is debuffed hours after it was made, which is a rule
// about somebody else's misfortune rather than about the copy.
func TestASummonsShareIsFrozenAtTheCast(t *testing.T) {
	fight := calling(t, []string{"copy", "jab"})
	came := arrivals(casts(t, fight, "copy"))
	if len(came) != 1 {
		t.Fatalf("the cast summoned %d units, want one", len(came))
	}
	copied := unitByID(t, fight, came[0].Target)
	if want := int64(400); copied.Base[progression.Attack] != want {
		t.Errorf("the copy has %d attack, want %d — half of the caster's 800",
			copied.Base[progression.Attack], want)
	}
	if want := int64(1500); copied.MaxHP() != want {
		t.Errorf("the copy has %d health, want %d", copied.MaxHP(), want)
	}

	weaken, err := fight.Books().Statuses.Lookup("weaken")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	unitByID(t, fight, "a").Statuses.Apply(weaken, 0)
	if copied.Base[progression.Attack] != 400 {
		t.Errorf("debuffing the caster moved the copy to %d attack",
			copied.Base[progression.Attack])
	}
}

// TestAShareOfBaseIgnoresWhatAShareWouldCopy is the whole reason there are two
// spellings, and the only test that can tell them apart.
//
// Move the caster's attack with a timed effect and cast: share copies the line
// as it stands, share_of_base does not. A debuff rather than a buff because the
// fixture book has one and the distinction is the same either way round — what
// an author picks between is a copy that can be set up beforehand and one that
// cannot.
func TestAShareOfBaseIgnoresWhatAShareWouldCopy(t *testing.T) {
	attackOf := func(using string) int64 {
		t.Helper()
		fight := calling(t, []string{using, "jab"})
		fight.Begin()
		fight.Drain()
		// Applied rather than cast, because what is being measured is the stat
		// line at the moment of the summon and not the turn order that would
		// deliver a buff.
		weaken, err := fight.Books().Statuses.Lookup("weaken")
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		caster := unitByID(t, fight, "a")
		caster.Statuses.Apply(weaken, 0)
		if fight.Stats(caster)[progression.Attack] >= caster.Base[progression.Attack] {
			t.Fatalf("the debuff left attack at %d, so this test measures nothing",
				fight.Stats(caster)[progression.Attack])
		}
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt.Unit != "a" {
			t.Fatalf("the first turn went to %s", prompt.Unit)
		}
		if err := fight.Act(using, caster.Cell); err != nil {
			t.Fatalf("act %s: %v", using, err)
		}
		came := arrivals(fight.Drain())
		if len(came) != 1 {
			t.Fatalf("%s summoned %d units, want one", using, len(came))
		}
		return unitByID(t, fight, came[0].Target).Base[progression.Attack]
	}

	live, base := attackOf("copy"), attackOf("copy_base")
	if live >= base {
		t.Errorf("a share of the debuffed line gave %d and a share of the base gave %d; "+
			"the first has to be the smaller or the two spellings are one", live, base)
	}
	if want := int64(400); base != want {
		t.Errorf("the share of base gave %d, want %d — half of the undebuffed 800", base, want)
	}
}

// TestAFixedStatLineIgnoresItsCallerEntirely is the other spelling: a creature
// that was called up is its own animal and does not get bigger because whoever
// called it did.
func TestAFixedStatLineIgnoresItsCallerEntirely(t *testing.T) {
	fight := calling(t, []string{"call_toad", "jab"})
	came := arrivals(casts(t, fight, "call_toad"))
	if len(came) != 1 {
		t.Fatalf("the call summoned %d units, want one", len(came))
	}
	toad := unitByID(t, fight, came[0].Target)
	if toad.MaxHP() != 900 || toad.Base[progression.Attack] != 300 {
		t.Errorf("the toad stands at %d health and %d attack, want the declared 900 and 300",
			toad.MaxHP(), toad.Base[progression.Attack])
	}
	// And its own element, which is what lets a summon be a different creature
	// rather than a smaller copy.
	if got := toad.Affinity.String(); got != "water" {
		t.Errorf("the toad is %s, want the water its skill declares", got)
	}
}

// TestASummonTakesTheFrontSlotsAndStopsWhenTheyRunOut is the board's own answer,
// and the reason a count is a request rather than a promise.
func TestASummonTakesTheFrontSlotsAndStopsWhenTheyRunOut(t *testing.T) {
	// Four allies already standing, so the fifth slot is the last one and a
	// swarm of three can only put down one.
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"swarm", "jab"}},
		{ID: "a2", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10), Skills: []string{"lob"}},
		{ID: "a3", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10), Skills: []string{"lob"}},
		{ID: "a4", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10), Skills: []string{"lob"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 5), Skills: []string{"jab"}},
	})
	came := arrivals(casts(t, fight, "swarm"))
	if len(came) != 1 {
		t.Fatalf("a swarm of three on a side with one slot left put down %d, want one", len(came))
	}
	if perSide := livingOn(fight, hex.SideAlly); perSide != 5 {
		t.Errorf("the ally side holds %d units, want the maximum of 5", perSide)
	}
}

func livingOn(fight *battle.Battle, side hex.Side) int {
	count := 0
	for _, unit := range fight.Units() {
		if unit.Side == side && !unit.Dead {
			count++
		}
	}
	return count
}

// TestADepartedSummonsSlotIsFreeAgain is what makes a summoning skill something
// a unit can do twice.
//
// A summon was never part of the formation the roster wrote down — it borrowed a
// slot the formation left empty — so when it is gone the slot is empty again.
// Counting it would make a repeatable skill die quietly: the shipped formations
// leave two free slots a side, so the third cast of a battle would put nothing
// down and say nothing about it.
//
// ⚠️ The id is NOT reused with the cell. A cell is a place and an id is what a
// decision in the log names, so two units standing in the same place at
// different times are still two units.
func TestADepartedSummonsSlotIsFreeAgain(t *testing.T) {
	fight := calling(t, []string{"brief_copy", "jab"})
	came := arrivals(casts(t, fight, "brief_copy"))
	if len(came) != 1 {
		t.Fatalf("the cast summoned %d units, want one", len(came))
	}
	first := unitByID(t, fight, came[0].Target)
	firstCell := first.Cell

	left := false
	for range 40 {
		prompt, err := fight.Advance()
		if err != nil || prompt == nil {
			break
		}
		if !prompt.Skipped {
			if prompt.Unit == "a" && left {
				if err := fight.Act("brief_copy", unitByID(t, fight, "a").Cell); err != nil {
					t.Fatalf("brief_copy: %v", err)
				}
				again := arrivals(fight.Drain())
				if len(again) != 1 {
					t.Fatalf("the second cast summoned %d units, want one", len(again))
				}
				second := unitByID(t, fight, again[0].Target)
				if second.Cell != firstCell {
					t.Errorf("the second copy stands at %s and the first one left %s empty",
						second.Cell, firstCell)
				}
				if second.ID == first.ID {
					t.Errorf("both copies are called %q; a cell may be reused and an id may not",
						second.ID)
				}
				return
			}
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Left && event.Actor == first.ID {
				left = true
			}
		}
	}
	t.Fatalf("the first copy left=%v and the caster never cast again", left)
}

// TestARosterUnitsSlotIsNotFreeWhenItFalls is the other half, and the two are
// the whole of the placement rule.
//
// The formation is what a roster wrote down. A side authored with units in named
// slots is that arrangement for the whole battle, and a summon appearing in a
// dead comrade's cell would be a placement nobody chose.
//
// ⚠️ It is driven through a real death rather than through a zeroed health bar.
// Nothing reads health looking for a corpse — kill is what makes a unit dead —
// and a test that set HP to nought would leave the board with no corpse on it at
// all, which is how the first draft of the test above passed a mutation.
func TestARosterUnitsSlotIsNotFreeWhenItFalls(t *testing.T) {
	// The caster stands at the back where nothing can reach it and the ally in
	// front is the only thing the foe may aim at, so the death below is the one
	// the test asked for rather than one the engine chose.
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 0, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 20),
			Skills: []string{"copy"}},
		{ID: "shield", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"jab"}},
		// The formation mirrors through 180 degrees, so the enemy slot facing the
		// ally at 2,0 is 2,2 rather than 2,0.
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	shield := unitByID(t, fight, "shield")
	shieldCell := shield.Cell
	shield.HP = 1

	for range 40 {
		prompt, err := fight.Advance()
		if err != nil || prompt == nil {
			break
		}
		if prompt.Skipped {
			fight.Drain()
			continue
		}
		switch {
		case prompt.Unit == "a" && shield.Dead:
			if err := fight.Act("copy", unitByID(t, fight, "a").Cell); err != nil {
				t.Fatalf("copy: %v", err)
			}
			came := arrivals(fight.Drain())
			if len(came) != 1 {
				t.Fatalf("the cast summoned %d units, want one", len(came))
			}
			if cell := unitByID(t, fight, came[0].Target).Cell; cell == shieldCell {
				t.Errorf("the copy took the fallen ally's cell at %s", shieldCell)
			}
			return
		case prompt.Unit == "f":
			if choice, ok := fight.Suggest(prompt); ok {
				if err := fight.Act(choice.Skill, choice.Aim); err != nil {
					t.Fatalf("act %s: %v", choice.Skill, err)
				}
			} else if err := fight.Pass("nothing to do"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		default:
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		fight.Drain()
	}
	t.Fatalf("the ally in front died=%v and the caster never cast", shield.Dead)
}

// TestASummonLeavesWhenItRunsOutOfTurnsAndItIsNotADeath is the expiry, and the
// distinction the second event kind exists for.
func TestASummonLeavesWhenItRunsOutOfTurnsAndItIsNotADeath(t *testing.T) {
	fight := calling(t, []string{"brief_copy", "jab"})
	came := arrivals(casts(t, fight, "brief_copy"))
	if len(came) != 1 {
		t.Fatalf("the cast summoned %d units, want one", len(came))
	}
	copied := came[0].Target

	turns, left := 0, battle.Event{}
	for range 30 {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if prompt.Unit == copied && !prompt.Skipped {
			turns++
		}
		if !prompt.Skipped {
			if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		for _, event := range fight.Drain() {
			if event.Kind == battle.Left && event.Actor == copied {
				left = event
			}
			if event.Kind == battle.Died && event.Actor == copied {
				t.Fatal("a copy that ran out of turns was logged as a death")
			}
		}
		if left.Kind == battle.Left {
			break
		}
	}
	if left.Kind != battle.Left {
		t.Fatal("the copy never left")
	}
	if turns != 2 {
		t.Errorf("a copy declared for two turns acted on %d of them", turns)
	}
	if left.Note != "out of turns" {
		t.Errorf("the copy left saying %q", left.Note)
	}
}

// TestABoundSummonGoesWithItsSummoner is the third way off the board, and it is
// a per-skill flag rather than a rule: a clone is an extension of whoever made
// it, and a creature that was called up is not.
func TestABoundSummonGoesWithItsSummoner(t *testing.T) {
	for _, bound := range []struct {
		using string
		gone  bool
	}{{"brief_copy", true}, {"copy", false}} {
		fight := callingAgainst(t, []string{bound.using, "jab"}, 200)
		came := arrivals(casts(t, fight, bound.using))
		if len(came) != 1 {
			t.Fatalf("%s summoned %d units, want one", bound.using, len(came))
		}
		copied := unitByID(t, fight, came[0].Target)

		caster := unitByID(t, fight, "a")
		caster.HP = 1
		// A turn from the foe finishes the caster off, which is what a summoner
		// dying looks like from the engine's side.
		for range 40 {
			prompt, err := fight.Advance()
			if err != nil || prompt == nil || caster.Dead {
				break
			}
			if !prompt.Skipped {
				if prompt.Unit == "f" {
					if err := fight.Act("jab", caster.Cell); err != nil {
						t.Fatalf("jab: %v", err)
					}
				} else if err := fight.Pass("waiting"); err != nil {
					t.Fatalf("pass: %v", err)
				}
			}
			fight.Drain()
		}
		if !caster.Dead {
			t.Fatalf("%s: the caster never fell, so nothing was measured", bound.using)
		}
		if copied.Dead != bound.gone {
			t.Errorf("%s: the caster died and the copy's gone is %v, want %v",
				bound.using, copied.Dead, bound.gone)
		}
	}
}

// TestASummonCountsWhenTheBattleIsDecided is the decision that makes a summon
// worth putting on a slot at all.
//
// It holds a place on the board and it is fought over like anything else, so a
// side is not empty while one is standing. The alternative — a copy that cannot
// keep its side alive — would be a unit the win condition does not see, and a
// player would watch a battle end with somebody still on the board.
func TestASummonCountsWhenTheBattleIsDecided(t *testing.T) {
	fight := callingAgainst(t, []string{"copy", "jab"}, 200)
	came := arrivals(casts(t, fight, "copy"))
	if len(came) != 1 {
		t.Fatalf("the cast summoned %d units, want one", len(came))
	}
	copied := unitByID(t, fight, came[0].Target)

	// The caster falls and the copy is unbound, so the ally side is one summon
	// and nothing else.
	caster := unitByID(t, fight, "a")
	caster.HP = 1
	for range 40 {
		prompt, err := fight.Advance()
		if err != nil || prompt == nil || caster.Dead {
			break
		}
		if !prompt.Skipped {
			if prompt.Unit == "f" {
				if err := fight.Act("jab", caster.Cell); err != nil {
					t.Fatalf("jab: %v", err)
				}
			} else if err := fight.Pass("waiting"); err != nil {
				t.Fatalf("pass: %v", err)
			}
		}
		fight.Drain()
	}
	if !caster.Dead {
		t.Fatal("the caster never fell, so nothing was measured")
	}
	if copied.Dead {
		t.Fatal("the copy went with it, so the side was never held by a summon alone")
	}
	if _, decided := fight.Winner(); decided {
		t.Error("the battle ended with a summon still standing on the losing side")
	}
}

// suggested is the skill autopilot would use on the first turn of a duel where
// the ally holds one summoning skill and one attack.
func suggested(t *testing.T, allySkills []string) string {
	t.Helper()
	fight := duel(t, allySkills, []string{"jab"}, 120, 100)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatalf("Suggest offered nothing at all for %v", allySkills)
	}
	return choice.Skill
}

// TestSuggestPricesASummonByTheTurnsItBuys is the whole of the opponent's side
// of summoning, in the three answers that separate a price from a guess.
//
// Before this, a summoning skill had no power of its own, so it reached Suggest
// only as the fallback — the option taken when nothing at all could be hurt. The
// shipped summoner therefore never called anybody up while it had a kunai in
// reach, and every figure measured of it was measured without its own mechanism
// firing.
//
// A summon is the only thing in the book that buys turns rather than spending
// one, so the turns are the price. Which makes three things checkable that a
// flat "summons are good" would not:
//
//   - a swarm of three copies beats a jab, so a cast happens at all;
//   - the same swarm for one turn does not, so the number of turns is read
//     rather than assumed;
//   - the same swarm for a hundred turns still loses to a strike, so the horizon
//     is capped rather than believed.
func TestSuggestPricesASummonByTheTurnsItBuys(t *testing.T) {
	if got := suggested(t, []string{"swarm", "jab"}); got != "swarm" {
		t.Errorf("autopilot picked %q over three copies that would each outlast the "+
			"jab twice; a summon priced at nothing is a summon never cast", got)
	}
	// lasts:1 against the same three copies at the same share: one turn each
	// instead of the capped four, and the jab wins.
	if got := suggested(t, []string{"brief_swarm", "jab"}); got != "jab" {
		t.Errorf("autopilot picked %q, want the jab: copies that leave after one turn "+
			"each are worth a quarter of ones that stay, and the price has to say so", got)
	}
	// lasts:100 is the cap's own case. A hundred turns of three copies would
	// dwarf any single attack in the book, so a rating that believed the skill
	// would take this over a strike every time and for ever.
	if got := suggested(t, []string{"long_swarm", "strike"}); got != "strike" {
		t.Errorf("autopilot picked %q over the strike: a summon that lasts a hundred "+
			"turns is priced for %d of them, not for the hundred it claims", got, 4)
	}
}

// TestASummonTheBoardHasNoRoomForIsNotPricedAtAll is the board's half of the
// price, and it is a different failure from getting the arithmetic wrong.
//
// summonPlaces is what puts copies down, and Suggest pays for exactly what it
// returns — so a side already at full strength pays for nothing, and the skill
// goes back to being the fallback it was. That is not the same as rating it at
// nought: a rating of nought is still a rating, and it would beat "no damaging
// option at all" and take the turn ahead of a shield or a cleanse.
//
// The contrast is the test. The same kit on a board with room casts the summon.
func TestASummonTheBoardHasNoRoomForIsNotPricedAtAll(t *testing.T) {
	if got := suggested(t, []string{"swarm", "jab"}); got != "swarm" {
		t.Fatalf("the control picked %q, so this measures nothing", got)
	}
	// Five on a side is hex.MaxTeamSize, so nothing else may stand there however
	// many formation slots are empty.
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: []string{"swarm", "jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: []string{"jab"}},
	}
	// ⚠️ The crowd leaves 2,2 empty on purpose, and that is what makes this
	// measure the room rather than the reach. Only two cells on this side are
	// within a jab of the duel slot — the caster's own and 2,2 — so a board whose
	// every free slot were out of range would price a copy at nothing whatever
	// the strength bound said, and a mutation dropping that bound would pass.
	for _, slot := range []hex.Offset{
		{Col: 2, Row: 0}, {Col: 1, Row: 0}, {Col: 1, Row: 1}, {Col: 1, Row: 2},
	} {
		roster = append(roster, battle.Roster{
			ID: "crowd" + slot.String(), Side: hex.SideAlly, Slot: slot,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 1),
			// lob rather than jab: New refuses a roster unit that cannot aim at
			// anybody, and the back rows are two cells from the duel slot. The
			// crowd is here to fill the side, not to fight.
			Skills: []string{"lob"},
		})
	}
	fight := mustBattle(t, books(t), 7, roster)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt.Unit != "a" {
		t.Fatalf("the first turn went to %s", prompt.Unit)
	}
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	if choice.Skill != "jab" {
		t.Errorf("autopilot picked %q on a side with no room for a copy, want the jab",
			choice.Skill)
	}

	// And the half that separates "not priced" from "priced at nought". Swap the
	// jab for a shield and there is nothing to rate at all, so the fallback
	// decides — and the fallback is the first usable skill with no power, which
	// is the shield standing ahead of the summon in this kit. A summon rated at
	// nought would beat it, because a rating of nought still sets found.
	roster[0].Skills = []string{"brace", "swarm"}
	shielding := mustBattle(t, books(t), 7, roster)
	prompt, err = shielding.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	choice, ok = shielding.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	if choice.Skill != "brace" {
		t.Errorf("autopilot picked %q, want the shield: a cast the board has no room "+
			"for is worth nothing, and nothing is not a score", choice.Skill)
	}
}
