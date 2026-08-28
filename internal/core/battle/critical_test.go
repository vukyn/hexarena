package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// critCasterMaxHP is the bar the jab below is priced against: one landing takes
// exactly half of it, so "half health" is a fact about the fixture rather than
// something a test has to reason about. Eight hundred attack at a power of one
// into no defence is eight hundred.
const critCasterMaxHP = 1600

var (
	critCasterCell = hex.Offset{Col: 2, Row: 1}
	critAimedCell  = hex.Place(hex.SideEnemy, hex.Offset{Col: 2, Row: 1})
)

// criticalBooks is the shared books with a skill book written for these tests.
//
// Every crit-carrying skill has a twin that declares none and is otherwise
// identical, because the whole shape of the mechanic is "the same skill, one
// figure different", and a test that cannot hold everything else still is a test
// measuring the fixture.
//
// ⚠️ `sure` and `sure_crit` declare 999 rather than 1000 on purpose. A power that
// divides cleanly by the critical multiplier hides the difference between folding
// the multiplier into the power and folding it into the whole expression, which
// is exactly what TestTheGradientAndACriticalCompose is about.
func criticalBooks(t *testing.T) battle.Books {
	t.Helper()
	shared := books(t)
	written, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"jab","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"plain","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"savage","element":"neutral","range":1,"pattern":"single",
	   "power":1000,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","crit":1000},
	  {"id":"flurry","element":"neutral","range":1,"pattern":"single",
	   "power":300,"strikes":4,"accuracy":1000,"cooldown":0,"target":"enemy","crit":500},
	  {"id":"venom","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "applies":[{"status":"poison","chance":1000}]},
	  {"id":"venom_crit","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","crit":1000,
	   "applies":[{"status":"poison","chance":1000}]},
	  {"id":"sure","element":"neutral","range":1,"pattern":"single",
	   "power":999,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy",
	   "self_gradient":{"at_empty":1000}},
	  {"id":"sure_crit","element":"neutral","range":1,"pattern":"single",
	   "power":999,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","crit":1000,
	   "self_gradient":{"at_empty":1000}}
	]}`), skill.Deps{Patterns: shared.Patterns, Statuses: shared.Statuses})
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	shared.Skills = written
	return shared
}

// critFight sets a caster opposite one target, lets the target wear it down by
// the given number of halvings, and has it use the named skill once. The target
// may hold a trait, which is how a reply gets into these tests.
func critFight(t *testing.T, seed uint64, id string, jabs int, targetTraits ...string) []battle.Event {
	t.Helper()
	fight, err := battle.New(criticalBooks(t), seed, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: critCasterCell,
			Affinity: single("neutral"), Stats: stats(critCasterMaxHP, 800, 400, 100),
			Skills: []string{id}},
		{ID: "target", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"jab"}, Passives: targetTraits},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	for range jabs {
		waitFor(t, fight, "target")
		if err := fight.Act("jab", hex.Place(hex.SideAlly, critCasterCell)); err != nil {
			t.Fatalf("jab: %v", err)
		}
		fight.Drain()
	}
	waitFor(t, fight, "caster")
	if err := fight.Act(id, critAimedCell); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return fight.Drain()
}

// struck is every landing on the named unit that came from a skill, in order.
func struck(events []battle.Event, victim string) []battle.Event {
	out := []battle.Event{}
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Target == victim && event.Passive == "" {
			out = append(out, event)
		}
	}
	return out
}

// TestADamageOverTimeTickNeverCrits is the one test standing between the shipped
// engine and somebody "completing" the feature by pushing a critical into
// inflict.
//
// A tick is frozen when the stack is applied and lasts the stack's whole life, so
// a critical one would be worth as many critical hits as the stack has turns —
// a different skill from the one the author wrote, and the same argument that
// keeps piercing out of a tick. The coupling that prevents it is Damage's default
// fifth argument, which is invisible at the call site in turn.go: nothing there
// says a tick cannot crit, so nothing but this would notice it starting to.
func TestADamageOverTimeTickNeverCrits(t *testing.T) {
	plain := critFight(t, 4, "venom", 0)
	critting := critFight(t, 4, "venom_crit", 0)

	// The strike itself does crit, so the fixture is doing what it says.
	plainHits, critHits := struck(plain, "target"), struck(critting, "target")
	if len(plainHits) != 1 || len(critHits) != 1 {
		t.Fatalf("the two casts landed %d and %d strikes, want one each", len(plainHits), len(critHits))
	}
	if !critHits[0].Critical {
		t.Fatal("venom_crit declares a certain critical chance and did not crit; the test measures nothing")
	}
	if critHits[0].Amount <= plainHits[0].Amount {
		t.Fatalf("the critical strike dealt %d against the ordinary %d", critHits[0].Amount, plainHits[0].Amount)
	}

	// And the tick frozen onto the stack is the same figure either way.
	plainApplied, critApplied := find(plain, battle.StatusApplied), find(critting, battle.StatusApplied)
	if len(plainApplied) != 1 || len(critApplied) != 1 {
		t.Fatalf("the two casts applied %d and %d statuses, want one each",
			len(plainApplied), len(critApplied))
	}
	if plainApplied[0].Amount == 0 {
		t.Fatal("the poison froze a tick of nothing; the test measures nothing")
	}
	if critApplied[0].Amount != plainApplied[0].Amount {
		t.Errorf("a critical strike froze a tick of %d where the ordinary one freezes %d;"+
			" a tick is computed once and lasts the stack's whole life, so a critical one is"+
			" worth a critical hit every turn it has left",
			critApplied[0].Amount, plainApplied[0].Amount)
	}
	if critApplied[0].Critical {
		t.Error("a status_applied event carries the critical flag, which belongs to a strike")
	}
}

// TestAReplyNeverCrits: a reply is what attacking a unit costs, priced off the
// unit that was attacked. It is not a strike anybody chose, so there is nothing
// for it to have landed well — and, more to the point, whether it crits cannot
// depend on the *attacker's* skill, which is the only crit chance anywhere near
// it.
func TestAReplyNeverCrits(t *testing.T) {
	answered := func(events []battle.Event) []battle.Event {
		out := []battle.Event{}
		for _, event := range events {
			if event.Kind == battle.Damaged && event.Passive != "" {
				out = append(out, event)
			}
		}
		return out
	}
	plain := answered(critFight(t, 6, "plain", 0, "spiked"))
	critting := answered(critFight(t, 6, "savage", 0, "spiked"))
	if len(plain) != 1 || len(critting) != 1 {
		t.Fatalf("the two casts drew %d and %d replies, want one each", len(plain), len(critting))
	}
	if critting[0].Critical {
		t.Error("a reply to a critical strike carries the critical flag")
	}
	if critting[0].Amount != plain[0].Amount {
		t.Errorf("attacking with a skill that always crits drew a reply of %d where an"+
			" ordinary skill draws %d; a reply is priced off the unit that was attacked",
			critting[0].Amount, plain[0].Amount)
	}
}

// TestTheDamagedEventCarriesTheCritical is the log half: a renderer reads events
// and nothing else, so a figure it cannot account for is the log failing to
// explain itself.
//
// It is checked per strike rather than per cast, on a hit where some strikes crit
// and some do not, because that is the only shape that can tell a per-strike flag
// from a per-cast one.
func TestTheDamagedEventCarriesTheCritical(t *testing.T) {
	var hits []battle.Event
	for seed := uint64(1); seed <= 200; seed++ {
		candidate := struck(critFight(t, seed, "flurry", 0), "target")
		if len(candidate) != 4 {
			continue
		}
		mixed := false
		for _, hit := range candidate[1:] {
			if hit.Critical != candidate[0].Critical {
				mixed = true
			}
		}
		if mixed {
			hits = candidate
			break
		}
	}
	if hits == nil {
		t.Fatal("no seed in two hundred produced a four-strike cast that crit on some strikes and not others")
	}
	var ordinary, critical int64
	for _, hit := range hits {
		if hit.Critical {
			critical = hit.Amount
		} else {
			ordinary = hit.Amount
		}
	}
	if critical <= ordinary {
		t.Fatalf("a critical strike is logged at %d and an ordinary one at %d", critical, ordinary)
	}
	for i, hit := range hits {
		want := ordinary
		if hit.Critical {
			want = critical
		}
		if hit.Amount != want {
			t.Errorf("strike %d carries critical=%v and %d damage, want %d",
				i+1, hit.Critical, hit.Amount, want)
		}
		// Power is the power the skill resolved at and a critical does not move
		// it: the multiplier is applied to the whole expression, not folded back
		// into what the log calls the skill's power.
		if hit.Power != hits[0].Power {
			t.Errorf("strike %d logs a power of %d where the first logs %d",
				i+1, hit.Power, hits[0].Power)
		}
	}
}

// TestTheGradientAndACriticalCompose holds the two terms apart.
//
// A gradient folds into the skill's *power*, before the damage expression; a
// critical multiplies the whole expression, after it. So a critical wrongly
// folded into the power takes its own truncation on a power the gradient has
// already truncated, and comes out a point light.
//
// ⚠️ The obvious assertion — "the ratio a critical is worth is the same at full
// health and at half" — does **not** hold and must not be written. Both figures
// are floored integers, so the ratio wobbles by a part per thousand with where
// the defence curve lands: this fixture reads 1251 at full health and 1250 at
// half, both of them correct. What is exact is the figure itself, so that is what
// is asserted, at both health levels, against the same one-truncation expression
// combat resolves through.
//
// A power of 999 is what makes this measurable: at 1000 the two readings agree to
// the point and the test would pass on either.
func TestTheGradientAndACriticalCompose(t *testing.T) {
	one := func(id string, jabs int) battle.Event {
		t.Helper()
		hits := struck(critFight(t, 9, id, jabs), "target")
		if len(hits) != 1 {
			t.Fatalf("%s at %d jabs landed %d strikes, want one", id, jabs, len(hits))
		}
		return hits[0]
	}
	fullPlain, fullCrit := one("sure", 0), one("sure_crit", 0)
	hurtPlain, hurtCrit := one("sure", 1), one("sure_crit", 1)

	if !fullCrit.Critical || !hurtCrit.Critical {
		t.Fatal("sure_crit declares a certain critical chance and did not crit; the test measures nothing")
	}
	if hurtPlain.Amount <= fullPlain.Amount {
		t.Fatalf("a caster at half health hit for %d where an untouched one hit for %d;"+
			" the gradient is not in force and the test measures nothing",
			hurtPlain.Amount, fullPlain.Amount)
	}
	if hurtPlain.Power == fullPlain.Power {
		t.Fatalf("the gradient left the power at %d either way; the test measures nothing", fullPlain.Power)
	}
	rules := criticalBooks(t).Rules
	// The hit the engine built, rebuilt from what the log says about it, so the
	// figure is checked against the expression rather than against itself.
	hitAt := func(power int) combat.Hit {
		return combat.Hit{
			Scaling: 800, Multiplier: power, Affinity: combat.PermilleBase,
			Defense: 400, SkillAccuracy: combat.PermilleBase,
		}
	}
	for _, at := range []struct {
		name        string
		plain, crit battle.Event
	}{
		{"full health", fullPlain, fullCrit},
		{"half health", hurtPlain, hurtCrit},
	} {
		if want := rules.Strike(hitAt(at.plain.Power)); at.plain.Amount != want {
			t.Errorf("at %s the ordinary strike landed for %d at a power of %d, want %d",
				at.name, at.plain.Amount, at.plain.Power, want)
		}
		if at.crit.Power != at.plain.Power {
			t.Errorf("at %s the critical cast resolved at a power of %d and the ordinary one at %d;"+
				" the gradient is the only thing allowed to move a power",
				at.name, at.crit.Power, at.plain.Power)
		}
		want := rules.CriticalStrike(hitAt(at.crit.Power))
		if at.crit.Amount != want {
			// The figure the fold-into-power mutation gives, named so a failure
			// says which mistake it is rather than only that a number moved.
			folded := rules.Strike(hitAt(at.crit.Power * rules.CriticalMultiplier / combat.PermilleBase))
			t.Errorf("at %s the critical strike landed for %d at a power of %d, want the %d one"+
				" truncation gives (%d is what folding the multiplier into the power gives)",
				at.name, at.crit.Amount, at.crit.Power, want, folded)
		}
	}
}

// TestSuggestPricesACriticalWithoutRollingOne is the rating half, in two parts:
// the opponent charges for a critical chance, and it does not roll one to find
// out what to charge.
//
// The second half is the one that breaks silently. Suggest may not touch the
// battle's source, because a draw taken while thinking moves every later roll in
// the battle — so a rated battle and an unrated one would replay differently and
// only a golden would say so.
func TestSuggestPricesACriticalWithoutRollingOne(t *testing.T) {
	// A unit carrying the same skill twice over, once critting and once not, and
	// the critting one is second so preferring it is a preference rather than kit
	// order.
	fight, err := battle.New(criticalBooks(t), 3, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: critCasterCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"plain", "savage"}},
		{ID: "target", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 100),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	prompt := waitPrompt(t, fight, "caster")
	choice, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	if choice.Skill != "savage" {
		t.Errorf("the rating chose %q over the identical skill that always crits;"+
			" a critical chance is worth something and Expected is what prices it", choice.Skill)
	}

	// And the same prompt rated many times leaves the battle's own rolls exactly
	// where they were. flurry is the action because it crits on half its strikes,
	// so a draw taken inside the rating shows up as a different pattern of flags
	// rather than as nothing at all.
	rated := critFlurryAfterRatings(t, 200)
	unrated := critFlurryAfterRatings(t, 0)
	if len(rated) != len(unrated) {
		t.Fatalf("a rated cast produced %d strikes and an unrated one %d", len(rated), len(unrated))
	}
	for i := range rated {
		if rated[i] != unrated[i] {
			t.Fatalf("strike %d differs between a rated battle and an unrated one:\nrated   %+v\nunrated %+v"+
				"\nsomething in the rating is drawing from the battle's own source", i+1, rated[i], unrated[i])
		}
	}
}

// critFlurryAfterRatings rates the caster's prompt the given number of times and
// then casts flurry, returning what landed.
func critFlurryAfterRatings(t *testing.T, ratings int) []battle.Event {
	t.Helper()
	fight, err := battle.New(criticalBooks(t), 12, []battle.Roster{
		{ID: "caster", Side: hex.SideAlly, Slot: critCasterCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"flurry"}},
		{ID: "target", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 100),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	prompt := waitPrompt(t, fight, "caster")
	for range ratings {
		if _, ok := fight.Suggest(prompt); !ok {
			t.Fatal("Suggest offered nothing at all")
		}
	}
	if err := fight.Act("flurry", critAimedCell); err != nil {
		t.Fatalf("flurry: %v", err)
	}
	return struck(fight.Drain(), "target")
}
