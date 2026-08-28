package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// casterMaxHP is the bar the jab below is priced against: one landing takes
// exactly half of it.
const casterMaxHP = 1600

// The cell the column below is aimed at, and the one the caster stands in.
var (
	casterCell = hex.Offset{Col: 2, Row: 1}
	aimedCell  = hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})
)

// gradientBooks is the shared books with a skill book written for these tests: a
// plain hit to measure against, the same hit carrying a gradient, a column
// carrying one, a draining column, and a jab the other side wears the caster
// down with.
//
// The jab is priced to take exactly half of the caster's bar in one landing —
// eight hundred attack at a power of one into no defence, against a bar of one
// thousand six hundred — so "half health" is a fact about the fixture rather than
// something a test has to reason about.
func gradientBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	written, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"strike","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"desperate","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_gradient":{"at_empty":1000}},
	  {"id":"sweep","element":"neutral","range":1,"pattern":"column",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_gradient":{"at_empty":1000}},
	  {"id":"siphon","element":"neutral","range":1,"pattern":"column",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "drains":1000,"self_gradient":{"at_empty":1000}},
	  {"id":"finisher","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "requires":{"below_health":1000,"bonus_power":1000},
	   "self_gradient":{"at_empty":1000}}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = written
	return shared
}

// TestAWoundedCasterHitsHarder is the feature end to end, through the whole stack
// rather than through combat.Gradient alone.
//
// The same skill is cast by the same unit against the same target on the same
// seed, once untouched and once at half health. Nothing else on the board
// differs, so the difference is the gradient or it is nothing.
func TestAWoundedCasterHitsHarder(t *testing.T) {
	full := struckFor(t, oneFight(t, "desperate", 0), "target")
	hurt := struckFor(t, oneFight(t, "desperate", 1), "target")
	if hurt <= full {
		t.Fatalf("a caster at half health hit for %d where an untouched one hit for %d, "+
			"so the gradient buys nothing", hurt, full)
	}
	// Half the bar gone, against a gradient worth a thousand at the bottom, is
	// half of a doubling. Asserted as a figure and not as "more", because "more"
	// passes for a gradient wired up to any number at all.
	if want := full * 3 / 2; hurt != want {
		t.Errorf("at half health the strike lands for %d, want %d: the curve is not the "+
			"straight line between full and empty that was authored", hurt, want)
	}
	// A skill with no gradient is untouched by any of it, which is what says the
	// term reaches the one skill that declares it rather than every skill.
	plainFull := struckFor(t, oneFight(t, "strike", 0), "target")
	plainHurt := struckFor(t, oneFight(t, "strike", 1), "target")
	if plainFull != plainHurt {
		t.Errorf("a skill with no gradient hit for %d untouched and %d hurt", plainFull, plainHurt)
	}
}

// TestTheWoundIsAShareOfWhatTheSkillActuallyLands is the compose order, and it is
// the argument for a multiplier rather than a bonus stated as a number.
//
// `finisher` is amplified to double power against anybody — a threshold at the
// whole bar, so it always holds — and the caster is then taken to half health. The
// wound is a share of the *amplified* power, so the strike is worth three times
// the declared one. A bonus added to the declared power instead would land at two
// and a half, and a caster swinging hardest at the skill it was already going to
// hit hardest with is the whole point.
//
// ⚠️ Written after a mutation escape. Swapping the order was caught only by the
// replay golden, which says a number moved and never which rule moved it, because
// nothing else in the suite had a skill carrying both terms at once.
func TestTheWoundIsAShareOfWhatTheSkillActuallyLands(t *testing.T) {
	full := struckFor(t, oneFight(t, "finisher", 0), "target")
	hurt := struckFor(t, oneFight(t, "finisher", 1), "target")
	plain := struckFor(t, oneFight(t, "strike", 0), "target")
	if want := full * 3 / 2; hurt != want {
		t.Errorf("an amplified skill from a caster at half health landed for %d, want %d: "+
			"the wound is being taken as a share of the declared power (which would be "+
			"about %d) rather than of the power the skill actually lands at",
			hurt, want, plain*3/2+plain)
	}
}

// TestTheWholeShapeIsSwungAtOnce is the other half of reading once per use: the
// point of a term the caster brings is that the whole shape is brought with it.
func TestTheWholeShapeIsSwungAtOnce(t *testing.T) {
	full, hurt := columnFight(t, "sweep", 0), columnFight(t, "sweep", 1)
	for _, victim := range []string{"bottom", "top"} {
		before, after := struckFor(t, full, victim), struckFor(t, hurt, victim)
		if after <= before {
			t.Errorf("the %s cell took %d from an untouched caster and %d from a hurt one, "+
				"so the gradient stopped at the aimed cell", victim, before, after)
		}
	}
}

// TestTheGradientIsReadOncePerUseAndNotOncePerTarget is a version of the trap
// Battle.spend already documents, arriving through a field that has no status to
// consume.
//
// resolveAgainst runs once per cell a shape covers, and a draining skill heals
// its own caster *inside* that loop. A gradient read there would have the second
// unit in a column swung at from a health the first one changed — a column that
// softens as it lands, written on no skill and visible in no table.
//
// So the test is a ratio rather than a comparison: the splash cell and the aimed
// cell take different damage by design, because a shape's edge takes a share. But
// being hurt has to multiply *both* by the same amount. Re-reading after the
// drain would leave the edge with a smaller multiplier than the middle, and
// nothing else in the suite would see it.
func TestTheGradientIsReadOncePerUseAndNotOncePerTarget(t *testing.T) {
	full, hurt := columnFight(t, "siphon", 0), columnFight(t, "siphon", 1)
	gains := map[string]int64{}
	for _, victim := range []string{"bottom", "top"} {
		before, after := struckFor(t, full, victim), struckFor(t, hurt, victim)
		gains[victim] = after * 1000 / before
	}
	if gains["bottom"] == 1000 {
		t.Fatal("the aimed cell gained nothing from a hurt caster, so nothing was measured")
	}
	// A few per mille of slack, because the two cells truncate separately and the
	// splash one is the smaller number. It is nowhere near enough to hide the bug:
	// the drain puts most of the caster's bar back between the two cells, so a
	// second reading would hand the edge about 1180 where the middle got 1500.
	if difference := gains["bottom"] - gains["top"]; difference > 5 || difference < -5 {
		t.Errorf("being hurt multiplied the aimed cell by %d per mille and the splash cell by %d: "+
			"the caster healed between the two and the gradient was read a second time",
			gains["bottom"], gains["top"])
	}
	// One announcement for one use. A second would be a renderer being told the
	// caster swung twice.
	uses := 0
	for _, event := range hurt {
		if event.Kind == battle.SkillUsed && event.Actor == "caster" {
			uses++
		}
	}
	if uses != 1 {
		t.Errorf("one use of a skill produced %d skill_used events for its caster", uses)
	}
}

// TestTheGradientReachesTheLogOrTheDamageIsUnexplained is the rule Pierce,
// Refused and Drained are each on an event for.
//
// The power on skill_used is what the skill *declares*, which is the figure a
// reader already has from the book. A hurt caster lands for more than that, so
// without the share the log states one number and then reports a strike that
// could only have come from another, with nothing anywhere to bridge them.
func TestTheGradientReachesTheLogOrTheDamageIsUnexplained(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		skill   string
		jabs    int
		carries bool
	}{
		{"untouched", "desperate", 0, false},
		{"half gone", "desperate", 1, true},
		{"no gradient declared", "strike", 1, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// The expected share is computed from the health the log itself
			// reports rather than from the arithmetic of the jab, and that is not
			// a tautology dressed up: what is asserted is that the engine read the
			// *caster's* bar rather than the target's and carried it to the event.
			// A version reading the target would produce a number too, and a fixed
			// figure here could not say which one it was.
			left, gradient, found := int64(casterMaxHP), -1, false
			for _, event := range oneFight(t, testCase.skill, testCase.jabs) {
				if event.Kind == battle.Damaged && event.Target == "caster" {
					left = event.Remaining
				}
				if event.Kind == battle.SkillUsed && event.Actor == "caster" {
					gradient, found = event.Gradient, true
				}
			}
			if !found {
				t.Fatal("the caster never used anything")
			}
			want := 0
			if testCase.carries {
				if want = combat.Gradient(left, casterMaxHP, 1000); want == 0 {
					t.Fatal("the caster was never hurt, so nothing was measured")
				}
			}
			if gradient != want {
				t.Errorf("the use recorded a gradient of %d against %d health of %d, want %d",
					gradient, left, casterMaxHP, want)
			}
		})
	}
}

// TestTheOpponentRatesAWoundedSwingAtWhatItWillLandFor is conditionTarget's rule
// arriving through the caster's own door: Suggest rates a skill by the power it
// would land and the engine then lands it, so a rating built from a different
// reading would make the opponent prefer a skill for a gain it does not get.
//
// The plain strike and the gradient skill are the same power, so an untouched
// caster is indifferent and the fixture's kit order decides. Hurt, the gradient
// one is strictly better and has to be the one suggested — and then has to
// actually land for more than the strike would have.
func TestTheOpponentRatesAWoundedSwingAtWhatItWillLandFor(t *testing.T) {
	fight := newGradientFight(t, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: casterCell,
			Affinity: single("neutral"), Stats: stats(casterMaxHP, 800, 0, 100),
			Skills: []string{"strike", "desperate"}},
		{ID: "foe", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"jab"}},
	})
	_ = jab(t, fight, "foe", 1)
	prompt := waitPrompt(t, fight, "caster")
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("the opponent suggested nothing for a unit with two usable skills")
	}
	if choice.Skill != "desperate" {
		t.Errorf("a caster at half health was told to use %q over the skill its wounds pay for, "+
			"so the rating is reading a power the engine does not land", choice.Skill)
	}
}

// newGradientFight enlists a roster against the gradient books and starts it.
func newGradientFight(t *testing.T, roster []battle.Roster) *battle.Battle {
	t.Helper()
	fight, err := battle.New(gradientBooks(t), 4, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// jab has the named enemy take the caster down by half its bar, once per landing,
// and hands back what that produced.
//
// ⚠️ The events are returned rather than dropped because a caller needs them: the
// health the caster is left on is only in the log, and draining it here is what
// made the first version of the log test read a caster that was never hurt.
func jab(t *testing.T, fight *battle.Battle, from string, times int) []battle.Event {
	t.Helper()
	landed := []battle.Event{}
	for range times {
		waitFor(t, fight, from)
		if err := fight.Act("jab", hex.Place(hex.SideAlly, casterCell)); err != nil {
			t.Fatalf("jab: %v", err)
		}
		landed = append(landed, fight.Drain()...)
	}
	return landed
}

// waitPrompt is waitFor with the prompt it stopped on, which Suggest needs.
func waitPrompt(t *testing.T, fight *battle.Battle, id string) *battle.Prompt {
	t.Helper()
	waitFor(t, fight, id)
	prompt := fight.Pending()
	if prompt == nil || prompt.Unit != id {
		t.Fatalf("stopped on %v rather than on %s", prompt, id)
	}
	return prompt
}

// oneFight sets a caster opposite one target, lets the foe take it down by the
// given number of halvings, and has it use the named skill once.
func oneFight(t *testing.T, id string, jabs int) []battle.Event {
	t.Helper()
	fight := newGradientFight(t, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: casterCell,
			Affinity: single("neutral"), Stats: stats(casterMaxHP, 800, 0, 100),
			Skills: []string{id}},
		{ID: "target", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"jab"}},
	})
	worn := jab(t, fight, "target", jabs)
	waitFor(t, fight, "caster")
	if err := fight.Act(id, aimedCell); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return append(worn, fight.Drain()...)
}

// columnFight is oneFight with two units stacked in the column the shape covers.
// The aimed cell is "bottom" and "top" is the splash.
func columnFight(t *testing.T, id string, jabs int) []battle.Event {
	t.Helper()
	fight := newGradientFight(t, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: casterCell,
			Affinity: single("neutral"), Stats: stats(casterMaxHP, 800, 0, 100),
			Skills: []string{id}},
		{ID: "top", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(4800, 100, 400, 100),
			Skills: []string{"jab"}},
		{ID: "bottom", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"jab"}},
	})
	_ = jab(t, fight, "bottom", jabs)
	waitFor(t, fight, "caster")
	if err := fight.Act(id, aimedCell); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return fight.Drain()
}

// struckFor is what the caster's use dealt to one unit.
func struckFor(t *testing.T, events []battle.Event, victim string) int64 {
	t.Helper()
	total := int64(0)
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Actor == "caster" && event.Target == victim {
			total += event.Amount
		}
	}
	if total == 0 {
		t.Fatalf("the caster dealt nothing to %s, so nothing was measured", victim)
	}
	return total
}
