package status_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/status"
)

func poison() status.Kind {
	return status.Kind{ID: "poison", Category: status.Dot, MaxStacks: 3, Duration: 3, TickPower: 500}
}

func burn() status.Kind {
	return status.Kind{ID: "burn", Category: status.Dot, MaxStacks: 2, Duration: 2, TickPower: 800}
}

func weaken() status.Kind {
	return status.Kind{ID: "weaken", Category: status.StatDebuff, MaxStacks: 3, Duration: 3}
}

func haste() status.Kind {
	return status.Kind{ID: "haste", Category: status.Buff, MaxStacks: 2, Duration: 3}
}

func block() status.Kind {
	return status.Kind{ID: "block", Category: status.Shield, MaxStacks: 3, Duration: 2}
}

func TestApplyStacksUpToTheCapThenReportsWaste(t *testing.T) {
	var set status.Set
	for i := 1; i <= 3; i++ {
		added, wasted := set.Apply(poison(), 100)
		if !added || wasted {
			t.Errorf("application %d gave added %v and wasted %v, want a stack added", i, added, wasted)
		}
		if got := set.Stacks("poison"); got != i {
			t.Errorf("after application %d there are %d stacks, want %d", i, got, i)
		}
	}
	added, wasted := set.Apply(poison(), 100)
	if added || !wasted {
		t.Errorf("the fourth application gave added %v and wasted %v, want it wasted", added, wasted)
	}
	if got := set.Stacks("poison"); got != 3 {
		t.Errorf("there are %d stacks, want the cap of 3", got)
	}
}

// TestEachStackKeepsItsOwnSnapshot is why stacks are held separately: two
// different attackers stacking the same poison must each contribute what their
// own attack was worth.
func TestEachStackKeepsItsOwnSnapshot(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 50)
	set.Apply(poison(), 200)
	set.Apply(poison(), 90)
	if got, want := set.TickAmount("poison"), int64(340); got != want {
		t.Errorf("the tick is %d, want %d", got, want)
	}
	damage, _, _ := set.Tick()
	if damage != 340 {
		t.Errorf("the first tick dealt %d, want 340", damage)
	}
}

// TestReapplyRefreshesEveryStack is what makes sustained pressure worth more
// than a single application: the stack going on also keeps the older ones alive.
func TestReapplyRefreshesEveryStack(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	set.Tick()
	set.Tick()
	if got, want := set.Remaining("poison"), 1; got != want {
		t.Fatalf("after two ticks there is %d turn left, want %d", got, want)
	}
	set.Apply(poison(), 100)
	if got, want := set.Remaining("poison"), 3; got != want {
		t.Errorf("after reapplying there are %d turns left, want the full %d", got, want)
	}
	if got, want := set.Stacks("poison"), 2; got != want {
		t.Errorf("there are %d stacks, want %d", got, want)
	}
}

// TestAStackTicksExactlyItsDuration pins the order of a turn: the damage is
// totalled before the duration is spent, so a status with one turn left still
// gets its final tick.
func TestAStackTicksExactlyItsDuration(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	total := int64(0)
	for turn := 1; turn <= 5; turn++ {
		damage, _, expired := set.Tick()
		total += damage
		switch {
		case turn < 3 && damage != 100:
			t.Errorf("turn %d dealt %d, want 100", turn, damage)
		case turn == 3 && damage != 100:
			t.Errorf("the final turn dealt %d, want 100", damage)
		case turn > 3 && damage != 0:
			t.Errorf("turn %d dealt %d after the status ran out", turn, damage)
		}
		if turn == 3 {
			if len(expired) != 1 || expired[0] != "poison" {
				t.Errorf("turn 3 reported %v expired, want poison", expired)
			}
		} else if len(expired) != 0 {
			t.Errorf("turn %d reported %v expired", turn, expired)
		}
	}
	if total != 300 {
		t.Errorf("a three turn poison dealt %d in total, want 300", total)
	}
	if set.Has("poison") {
		t.Error("the poison is still there after it ran out")
	}
}

// TestTotalDamageDoesNotDependOnSpeed is the point of counting duration in the
// holder's own turns. A hasted victim takes its ticks sooner, not more often, so
// speeding a poisoned unit up is not a hidden self-inflicted wound.
func TestTotalDamageDoesNotDependOnSpeed(t *testing.T) {
	for _, turnsToSimulate := range []int{3, 10, 40} {
		var set status.Set
		set.Apply(poison(), 160)
		total := int64(0)
		for turn := 0; turn < turnsToSimulate; turn++ {
			damage, _, _ := set.Tick()
			total += damage
		}
		if total != 480 {
			t.Errorf("over %d turns the poison dealt %d, want 480", turnsToSimulate, total)
		}
	}
}

func TestTickOrderIsStable(t *testing.T) {
	build := func() *status.Set {
		set := &status.Set{}
		set.Apply(burn(), 70)
		set.Apply(poison(), 40)
		set.Apply(weaken(), 0)
		return set
	}
	first, second := build(), build()
	for turn := 0; turn < 4; turn++ {
		damageA, _, expiredA := first.Tick()
		damageB, _, expiredB := second.Tick()
		if damageA != damageB {
			t.Fatalf("turn %d dealt %d and %d", turn, damageA, damageB)
		}
		if strings.Join(expiredA, ",") != strings.Join(expiredB, ",") {
			t.Fatalf("turn %d expired %v and %v", turn, expiredA, expiredB)
		}
	}
	set := build()
	if got, want := strings.Join(set.Active(), ","), "burn,poison,weaken"; got != want {
		t.Errorf("the active statuses are %q, want %q in application order", got, want)
	}
}

// TestRemoveTakesTheHeaviestStacksFirst is what makes a partial cleanse worth
// casting. Taking the weakest would leave the player worse off than not
// cleansing in the case they care about.
//
// The figures are the ticks the stacks still owed rather than one tick of them,
// so a three-turn poison ticking for 300 is reported as the 900 it was going to
// deal. TestRemoveReportsTheTicksLeftRatherThanOne holds the duration itself.
func TestRemoveTakesTheHeaviestStacksFirst(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 50)
	set.Apply(poison(), 300)
	set.Apply(poison(), 120)
	removed, damage := set.Remove("poison", 1)
	if removed != 1 || damage != 900 {
		t.Errorf("removing one stack took %d stacks worth %d, want 1 worth 900", removed, damage)
	}
	if got, want := set.TickAmount("poison"), int64(170); got != want {
		t.Errorf("the remaining tick is %d, want %d", got, want)
	}
	removed, damage = set.Remove("poison", 5)
	if removed != 2 || damage != 510 {
		t.Errorf("removing the rest took %d stacks worth %d, want 2 worth 510", removed, damage)
	}
	if set.Has("poison") {
		t.Error("the poison is still listed after every stack went")
	}
	if removed, damage := set.Remove("poison", 1); removed != 0 || damage != 0 {
		t.Errorf("removing from nothing took %d worth %d", removed, damage)
	}
	if removed, _ := set.Remove("poison", -3); removed != 0 {
		t.Error("a negative count removed something")
	}
}

// TestConsumeIsWhatADetonateCallsFor gives a burst skill the figure to price
// itself against: the damage over time it just threw away.
func TestConsumeIsWhatADetonateCallsFor(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 160)
	set.Apply(poison(), 160)
	set.Apply(poison(), 160)
	// Three stacks of 160 with three turns each: 1440, not the 480 one tick of
	// them comes to.
	stacks, damage := set.Consume("poison")
	if stacks != 3 || damage != 1440 {
		t.Errorf("consuming took %d stacks worth %d, want 3 worth 1440", stacks, damage)
	}
	if set.Has("poison") {
		t.Error("the poison survived being consumed")
	}
	if stacks, damage := set.Consume("burn"); stacks != 0 || damage != 0 {
		t.Errorf("consuming a status that is not there took %d worth %d", stacks, damage)
	}
}

// TestRemoveReportsTheTicksLeftRatherThanOne is the whole of the change, and it
// is a separate test because every other fixture here consumes a status the turn
// it was applied — where a full duration is still on it and a mistake that
// reported a single tick, or that charged every stack the full duration
// regardless, produces figures a fresh set cannot tell apart.
//
// A status is worth less the longer it has been running, and a detonate is worth
// *more* the less the status had left. Reading a stack that has already ticked
// is the only way to see either.
func TestRemoveReportsTheTicksLeftRatherThanOne(t *testing.T) {
	kind := poison()
	for spent := 0; spent < kind.Duration; spent++ {
		var set status.Set
		set.Apply(kind, 100)
		for range spent {
			set.Tick()
		}
		left := kind.Duration - spent
		_, damage := set.Consume("poison")
		if want := int64(100 * left); damage != want {
			t.Errorf("a poison %d turns in was worth %d, want %d for the %d turns left",
				spent, damage, want, left)
		}
	}
	// A stack on its last turn is worth exactly the one tick it has still to
	// take, which is the case the old figure happened to get right.
	var last status.Set
	last.Apply(kind, 100)
	for range kind.Duration - 1 {
		last.Tick()
	}
	if _, damage := last.Consume("poison"); damage != 100 {
		t.Errorf("a poison on its last turn was worth %d, want its final tick of 100", damage)
	}
}

// TestCleanseTakesOnlyTheNamedCategories is what stops a cleanse from stripping
// the holder's own buffs and shields.
func TestCleanseTakesOnlyTheNamedCategories(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	set.Apply(poison(), 100)
	set.Apply(weaken(), 0)
	set.Apply(haste(), 0)
	set.Apply(block(), 0)

	if got := set.Cleanse([]status.Category{status.Dot}, 1); got != 1 {
		t.Errorf("a one stack cleanse took %d", got)
	}
	if got, want := set.Stacks("poison"), 1; got != want {
		t.Errorf("%d poison stacks left, want %d", got, want)
	}
	if got, want := set.Stacks("haste"), 1; got != want {
		t.Errorf("the cleanse touched haste, %d stacks left, want %d", got, want)
	}

	if got := set.Cleanse([]status.Category{status.Dot, status.StatDebuff}, 10); got != 2 {
		t.Errorf("a full cleanse took %d stacks, want 2", got)
	}
	for _, id := range []string{"poison", "weaken"} {
		if set.Has(id) {
			t.Errorf("%s survived the cleanse", id)
		}
	}
	for _, id := range []string{"haste", "block"} {
		if !set.Has(id) {
			t.Errorf("%s was removed by a cleanse aimed at debuffs", id)
		}
	}

	// A dispel is the same call aimed at the other side of the ledger.
	if got := set.Cleanse([]status.Category{status.Buff}, 10); got != 1 {
		t.Errorf("a dispel took %d stacks, want 1", got)
	}
	if set.Has("haste") {
		t.Error("haste survived a dispel")
	}
	if !set.Has("block") {
		t.Error("a dispel aimed at buffs removed a shield")
	}
}

func TestCleanseEdges(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	if got := set.Cleanse(nil, 5); got != 0 {
		t.Errorf("a cleanse with no categories took %d", got)
	}
	if got := set.Cleanse([]status.Category{status.Dot}, 0); got != 0 {
		t.Errorf("a cleanse of no stacks took %d", got)
	}
	if got := set.Cleanse([]status.Category{status.Category(99)}, 5); got != 0 {
		t.Errorf("a cleanse of an undeclared category took %d", got)
	}
	if got := set.Cleanse([]status.Category{status.Control}, 5); got != 0 {
		t.Errorf("a cleanse of a category the unit does not carry took %d", got)
	}
}

// TestBlockChargesAreAShieldStatus is how block gained an expiry: its stack
// count is the charge count, so it inherits duration from this package instead
// of needing a second mechanism.
func TestBlockChargesAreAShieldStatus(t *testing.T) {
	var set status.Set
	for i := 0; i < 3; i++ {
		set.Apply(block(), 0)
	}
	if got, want := set.Stacks("block"), 3; got != want {
		t.Errorf("%d charges held, want %d", got, want)
	}
	// Spending a charge is a removal of one stack, and costs no damage because
	// a shield stack carries none.
	if removed, damage := set.Remove("block", 1); removed != 1 || damage != 0 {
		t.Errorf("spending a charge took %d stacks worth %d, want 1 worth 0", removed, damage)
	}
	// Charges deal nothing on a tick but do run out.
	damage, _, _ := set.Tick()
	if damage != 0 {
		t.Errorf("a shield ticked for %d, want nothing", damage)
	}
	if got, want := set.Stacks("block"), 2; got != want {
		t.Errorf("%d charges after a tick, want %d", got, want)
	}
	damage, _, expired := set.Tick()
	if damage != 0 {
		t.Errorf("a shield ticked for %d, want nothing", damage)
	}
	if len(expired) != 1 || expired[0] != "block" {
		t.Errorf("the charges expired as %v, want block", expired)
	}
	if set.Has("block") {
		t.Error("the charges outlived their duration")
	}
}

func TestCountInAndSnapshot(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 120)
	set.Apply(poison(), 80)
	set.Apply(burn(), 200)
	set.Apply(haste(), 0)
	if got, want := set.CountIn(status.Dot), 3; got != want {
		t.Errorf("%d damaging stacks, want %d", got, want)
	}
	if got, want := set.CountIn(status.Buff), 1; got != want {
		t.Errorf("%d buff stacks, want %d", got, want)
	}
	if got, want := set.CountIn(status.Control), 0; got != want {
		t.Errorf("%d control stacks, want %d", got, want)
	}
	snapshot := set.Snapshot()
	if len(snapshot) != 3 {
		t.Fatalf("the snapshot lists %d statuses, want 3", len(snapshot))
	}
	first := snapshot[0]
	if first.ID != "poison" || first.Stacks != 2 || first.TickAmount != 200 || first.Remaining != 3 {
		t.Errorf("the first snapshot is %+v", first)
	}
	if snapshot[1].ID != "burn" || snapshot[2].ID != "haste" {
		t.Errorf("the snapshot is out of application order: %+v", snapshot)
	}
}

func TestNegativeTickDamageIsTreatedAsNone(t *testing.T) {
	var set status.Set
	set.Apply(poison(), -500)
	if got := set.TickAmount("poison"); got != 0 {
		t.Errorf("a negative snapshot became %d, want 0", got)
	}
}

func TestZeroSetIsUsable(t *testing.T) {
	var set status.Set
	damage, _, expired := set.Tick()
	if damage != 0 || len(expired) != 0 {
		t.Errorf("a clean unit ticked for %d and expired %v", damage, expired)
	}
	if set.Has("poison") || set.Stacks("poison") != 0 || set.Remaining("poison") != 0 {
		t.Error("a clean unit reports a status")
	}
	if len(set.Active()) != 0 || len(set.Snapshot()) != 0 {
		t.Error("a clean unit lists statuses")
	}
}

// TestATauntIsSomethingItsHolderWantsGone.
//
// Harmful is the split that says which categories are an attack, and it gates
// what a trait may resist: a trait refusing a buff would be refusing its own
// side's help. A taunt on the wrong side of that line would make "cannot be
// provoked" unwritable, and nothing else in the engine would notice -- the
// mutation that moved it passed every other test in the repository.
func TestATauntIsSomethingItsHolderWantsGone(t *testing.T) {
	if !status.Taunt.Harmful() {
		t.Error("a taunt is not counted as an attack, so no trait can be written to refuse one")
	}
	// And the split still holds for the three it must never cover.
	for _, kind := range []status.Category{status.Buff, status.Shield, status.Regen} {
		if kind.Harmful() {
			t.Errorf("%s is counted as an attack, so a trait could refuse its own side's help", kind)
		}
	}
}

// TestOnlyATickOutlastsAShield is the whole of OutlastsAShield asserted over
// every category there is, rather than over the one it answers yes to.
//
// A one-case switch is exactly what a later reader completes into something
// tidier, and the two candidates are both wrong in a way nothing else here would
// notice. Harmful is the near miss: it covers Dot, StatDebuff, Control and Taunt,
// so reusing it would let a stat debuff through a shield -- which was measured
// and rejected, because with mire unstoppable a squirtle stops being able to
// finish a duel against itself and TestABothWaysMirrorIsExactlyEven, a fairness
// invariant, breaks. Reading the split off the enum's own order is the other:
// Dot is at zero, so "the first category" happens to be the right answer today
// and stops being one the moment anything is declared above it.
func TestOnlyATickOutlastsAShield(t *testing.T) {
	for _, category := range status.Categories() {
		want := category == status.Dot
		if got := category.OutlastsAShield(); got != want {
			t.Errorf("%s outlasts a shield: %v, want %v", category, got, want)
		}
	}
	// And the two splits are stated apart, because the whole risk is that one is
	// mistaken for the other.
	for _, category := range []status.Category{
		status.StatDebuff, status.Control, status.Taunt, status.HealCut,
	} {
		if !category.Harmful() {
			t.Errorf("%s stopped being harmful, so this test no longer says the two splits differ", category)
		}
		if category.OutlastsAShield() {
			t.Errorf("%s reaches a target through a shield, which is Harmful's answer rather than this one", category)
		}
	}
}

func TestCategoryNames(t *testing.T) {
	// Declaration order, and it is asserted as a LIST rather than as a set on
	// purpose: CategoryCount and every table built from this order — the grouped
	// reference's print order among them — move when a category is slotted in
	// rather than appended, which is the rule both HealCut and Taunt are declared
	// under. A new category belongs on the END of this line.
	want := []string{"dot", "stat_debuff", "control", "buff", "shield", "regen", "taunt", "heal_cut", "charge"}
	categories := status.Categories()
	if len(categories) != len(want) {
		t.Fatalf("there are %d categories, want %d", len(categories), len(want))
	}
	for i, category := range categories {
		if category.String() != want[i] {
			t.Errorf("category %d is %q, want %q", i, category, want[i])
		}
		parsed, err := status.ParseCategory(category.String())
		if err != nil || parsed != category {
			t.Errorf("ParseCategory(%q) gave %v, %v", category, parsed, err)
		}
	}
	if _, err := status.ParseCategory("curse"); err == nil {
		t.Error("an unknown category name was accepted")
	}
	if got := status.Category(99).String(); !strings.Contains(got, "99") {
		t.Errorf("an undeclared category renders as %q", got)
	}
	for _, category := range []status.Category{
		status.Dot, status.StatDebuff, status.Control, status.HealCut,
	} {
		if !category.Harmful() {
			t.Errorf("%s should be harmful", category)
		}
	}
	for _, category := range []status.Category{status.Buff, status.Shield} {
		if category.Harmful() {
			t.Errorf("%s should not be harmful", category)
		}
	}
}

func TestParseBookRejects(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"malformed json", "{", "decode status book"},
		{"no stack limit", `{"max_stacks":0,"max_duration":6,"kinds":[]}`, "max_stacks"},
		{"no duration limit", `{"max_stacks":5,"max_duration":0,"kinds":[]}`, "max_duration"},
		{"no kinds", `{"max_stacks":5,"max_duration":6,"kinds":[]}`, "empty"},
		{"a kind with no id", `{"max_stacks":5,"max_duration":6,"kinds":[{"category":"dot","max_stacks":1,"duration":1,"tick_power":500}]}`, "needs an id"},
		{"an unknown category", `{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"curse","max_stacks":1,"duration":1}]}`, "unknown status category"},
		{"stacks over the limit", `{"max_stacks":3,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":4,"duration":1,"tick_power":500}]}`, "over the max_stacks limit of 3"},
		{"no stacks", `{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":0,"duration":1,"tick_power":500}]}`, "at least 1"},
		{"duration over the limit", `{"max_stacks":5,"max_duration":2,"kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":3,"tick_power":500}]}`, "over the limit of 2"},
		{"no duration", `{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":0,"tick_power":500}]}`, "lasts 0 turns"},
		{"a duplicate id", `{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":1,"tick_power":500},{"id":"x","category":"buff","max_stacks":1,"duration":1}]}`, "declared twice"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := status.ParseBook([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

func TestBookLookup(t *testing.T) {
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5,
	  "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "block", "category": "shield", "max_stacks": 3, "duration": 2}
	  ]
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	kind, err := book.Lookup("poison")
	if err != nil || kind.MaxStacks != 3 || kind.Category != status.Dot {
		t.Errorf("Lookup(poison) gave %+v, %v", kind, err)
	}
	if _, err := book.Lookup("curse"); err == nil {
		t.Error("an unknown id was accepted")
	}
	if got := len(book.Kinds()); got != 2 {
		t.Errorf("the book holds %d kinds, want 2", got)
	}
	// Kinds returns a copy, so a caller cannot rewrite the book.
	book.Kinds()[0] = status.Kind{ID: "tampered"}
	if got := book.Kinds()[0].ID; got != "poison" {
		t.Errorf("the book was modified through its own accessor, first kind is now %q", got)
	}
}

// TestTickPowerBelongsOnlyToADot keeps the one field that describes damage from
// drifting onto statuses that never deal any, where it would be dead data nobody
// notices is wrong.
func TestTickPowerBelongsOnlyToADot(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"a dot with no tick power",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":1}]}`,
			"no tick_power"},
		{"a buff with tick power",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"buff","max_stacks":1,"duration":1,"tick_power":300}]}`,
			"only a ticking status uses"},
		{"a shield with tick power",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"shield","max_stacks":1,"duration":1,"tick_power":300}]}`,
			"only a ticking status uses"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := status.ParseBook([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

// TestHealShareBelongsOnlyToAHealCut is TestTickPowerBelongsOnlyToADot's twin,
// and it exists for the same two reasons pointed opposite ways.
//
// A heal_cut is a category whose one job is a number, so one without it would
// parse, apply, appear in the log and change nothing anybody can see — the same
// dead-data shape a regeneration that froze nought had. And a heal_share on
// anything else is a figure the engine never reads, since only HealCut
// contributes to Set.HealShare.
//
// The bound is on both sides. A positive share would raise the healing its holder
// receives, which the category's own name says it does not do — the wording on
// screen is "cuts healing received", so accepting one ships a description that
// lies — and one stack promising more than total negation is a promise the floor in
// the engine refuses to keep.
func TestHealShareBelongsOnlyToAHealCut(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"a heal cut with no share",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"heal_cut","max_stacks":1,"duration":1}]}`,
			"no heal_share"},
		{"a heal cut that raises healing",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"heal_cut","max_stacks":1,"duration":1,"heal_share":400}]}`,
			"it must lower the healing"},
		{"a heal cut past total negation",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"heal_cut","max_stacks":1,"duration":1,"heal_share":-1001}]}`,
			"want between -1000 and -1"},
		{"a buff with a heal share",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"buff","max_stacks":1,"duration":1,"heal_share":-400}]}`,
			"which only a heal_cut uses"},
		{"a dot with a heal share",
			`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":1,"duration":1,"tick_power":500,"heal_share":-400}]}`,
			"which only a heal_cut uses"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := status.ParseBook([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
	// The control: the shape all five refusals are a deviation from parses, so an
	// error above is the rule rather than a typo in the fixture.
	book, err := status.ParseBook([]byte(
		`{"max_stacks":5,"max_duration":6,"kinds":[{"id":"x","category":"heal_cut","max_stacks":2,"duration":2,"heal_share":-400}]}`))
	if err != nil {
		t.Fatalf("a well-formed heal cut was refused: %v", err)
	}
	kind, err := book.Lookup("x")
	if err != nil || kind.HealShare != -400 {
		t.Errorf("the parsed heal cut carries %+v, %v — the share has to survive the parse", kind, err)
	}
}

// TestHealShareAccumulatesPerStack is Modifiers' rule for the one field that is
// not a modifier: a stack contributes its share once, so stacking is worth doing.
//
// It bounds nothing on purpose, which is the half worth asserting: the total is a
// share of an amount rather than of a stat, so the floor lives where the amount is
// and a second bound here would be a second answer. Three stacks of -600 therefore
// come back as -1800 and the engine is what refuses to pay a negative heal.
func TestHealShareAccumulatesPerStack(t *testing.T) {
	festerKind := status.Kind{
		ID: "fester", Category: status.HealCut, MaxStacks: 3, Duration: 3, HealShare: -600,
	}
	var set status.Set
	if got := set.HealShare(); got != 0 {
		t.Errorf("a clean unit cuts healing by %d, want nothing", got)
	}
	for stacks := 1; stacks <= 3; stacks++ {
		set.Apply(festerKind, 0)
		if got, want := set.HealShare(), -600*stacks; got != want {
			t.Errorf("%d stacks summed to %d, want %d", stacks, got, want)
		}
	}
	// Past the cap the total stands still, because Apply refuses the stack.
	set.Apply(festerKind, 0)
	if got := set.HealShare(); got != -1800 {
		t.Errorf("a fourth stack over a cap of three summed to %d, want -1800", got)
	}
	// A status of another category contributes nothing, so the sum reads the
	// category rather than every share it can see.
	set.Apply(status.Kind{
		ID: "weaken", Category: status.StatDebuff, MaxStacks: 1, Duration: 1,
		Modifiers: []modifier.Modifier{
			{Target: modifier.Attack, Mode: modifier.Percent, Amount: -300},
		},
	}, 0)
	if got := set.HealShare(); got != -1800 {
		t.Errorf("a stat debuff moved the heal share to %d", got)
	}
}

// TestModifiersAccumulatePerStack is the wiring between a status and the buff
// layer: three stacks of a debuff contribute their term three times, and the
// modifier package is what bounds the total.
func TestModifiersAccumulatePerStack(t *testing.T) {
	weakenKind := status.Kind{
		ID: "weaken", Category: status.StatDebuff, MaxStacks: 3, Duration: 3,
		Modifiers: []modifier.Modifier{
			{Target: modifier.Attack, Mode: modifier.Percent, Amount: -300},
		},
	}
	var set status.Set
	for stacks := 1; stacks <= 3; stacks++ {
		set.Apply(weakenKind, 0)
		terms := set.Modifiers()
		if got, want := terms.Percent(modifier.Attack), int64(-300*stacks); got != want {
			t.Errorf("%d stacks summed to %d, want %d", stacks, got, want)
		}
	}
	// Removing a stack takes its term with it.
	set.Remove("weaken", 1)
	if got, want := set.Modifiers().Percent(modifier.Attack), int64(-600); got != want {
		t.Errorf("after removing a stack the term is %d, want %d", got, want)
	}
	// A status with no terms contributes nothing.
	var clean status.Set
	clean.Apply(poison(), 100)
	if got := clean.Modifiers().Percent(modifier.Attack); got != 0 {
		t.Errorf("a poison contributed an attack term of %d", got)
	}
}

func TestBookRejectsAnAffinityModifierOnAStatus(t *testing.T) {
	_, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [{"id":"x","category":"buff","max_stacks":1,"duration":1,
	    "modifiers":[{"target":"affinity","mode":"percent","amount":300}]}]
	}`))
	if err == nil {
		t.Fatal("an affinity term on a status was accepted")
	}
	if !strings.Contains(err.Error(), "cannot carry") {
		t.Errorf("error %q does not explain why", err)
	}
}

func TestBookRejectsAnInvalidModifier(t *testing.T) {
	_, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [{"id":"x","category":"buff","max_stacks":1,"duration":1,
	    "modifiers":[{"target":"attack","mode":"percent","amount":0}]}]
	}`))
	if err == nil {
		t.Fatal("a modifier with no amount was accepted")
	}
}

// TestAPermanentStatusNeitherExpiresNorCanBeTakenOff is what "permanent" has to
// mean, stated as the four things that end a status and do not end this one.
//
// A passive grants one, and a passive is granted once when its holder is
// enlisted. So anything that took a stack off would turn the trait off for the
// rest of the battle with no way back, which is a far larger effect than
// stripping a buff somebody cast a moment ago.
func TestAPermanentStatusNeitherExpiresNorCanBeTakenOff(t *testing.T) {
	book := permanentBook(t)
	kind, err := book.Lookup("toughened")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	var set status.Set
	set.Apply(kind, 0)
	set.Apply(kind, 0)
	if got := set.Stacks("toughened"); got != 2 {
		t.Fatalf("two applications gave %d stacks", got)
	}

	// Turns going by, which is what expires anything timed. Far more than any
	// duration the book allows.
	for turn := range 50 {
		damage, healing, expired := set.Tick()
		if damage != 0 || healing != 0 {
			t.Fatalf("turn %d: a permanent buff ticked %d damage and %d healing", turn, damage, healing)
		}
		if len(expired) != 0 {
			t.Fatalf("turn %d: a permanent status expired: %v", turn, expired)
		}
	}
	if got := set.Stacks("toughened"); got != 2 {
		t.Errorf("after 50 turns the status is down to %d stacks, want 2", got)
	}

	// A dispel, a cleanse and a detonate all reach Remove, so all three are
	// refused by one guard — and all three are checked, because a guard in the
	// wrong place would stop only the one it was written for.
	if removed, damage := set.Remove("toughened", 2); removed != 0 || damage != 0 {
		t.Errorf("a dispel took %d stacks and %d tick damage off a permanent status", removed, damage)
	}
	if got := set.Cleanse([]status.Category{status.Buff}, 5); got != 0 {
		t.Errorf("a cleanse took %d stacks off a permanent status", got)
	}
	if stacks, _ := set.Consume("toughened"); stacks != 0 {
		t.Errorf("consuming took %d stacks off a permanent status", stacks)
	}
	if got := set.Stacks("toughened"); got != 2 {
		t.Errorf("the status is down to %d stacks, want the 2 it started with", got)
	}

	// A timed status in the same set still expires, so the guard is on the kind
	// rather than on the set.
	timed, err := book.Lookup("haste")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	set.Apply(timed, 0)
	for range timed.Duration {
		set.Tick()
	}
	if set.Has("haste") {
		t.Error("a timed status in a set holding a permanent one did not expire")
	}
	if got := set.Stacks("toughened"); got != 2 {
		t.Errorf("expiring the timed status also cost the permanent one; %d stacks left", got)
	}
}

// TestTimedIgnoresWhatNeverRunsOut is the question a battle asks before deciding
// it can no longer change: is anything on this unit still counting down.
//
// A permanent status must not answer yes. A passive is what a unit is rather
// than something happening to it, so a board where the only statuses left are
// traits is a board that will never move on its own — and counting one would
// keep a deadlocked battle open for ever, which is the failure the whole
// question exists to end.
func TestTimedIgnoresWhatNeverRunsOut(t *testing.T) {
	book := permanentBook(t)
	permanent, err := book.Lookup("toughened")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	timed, err := book.Lookup("haste")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	var set status.Set
	if set.Timed() {
		t.Error("a clean unit is holding something timed")
	}
	set.Apply(permanent, 0)
	if set.Timed() {
		t.Error("a permanent status counts as something still counting down")
	}
	set.Apply(timed, 0)
	if !set.Timed() {
		t.Fatal("a buff with a duration on it does not count")
	}
	// The timed one runs out and the permanent one does not, so the answer has
	// to go back to no rather than staying yes for the rest of the battle.
	for range timed.Duration {
		set.Tick()
	}
	if set.Timed() {
		t.Error("the answer is still yes after the only timed status expired")
	}
	if !set.Has("toughened") {
		t.Error("the permanent status went with it")
	}
}

// TestASnapshotSaysWhichStatusesHaveNoCountdown is the renderer's half: reading
// Remaining alone would draw "0 turns left" beside the one thing that never runs
// out.
func TestASnapshotSaysWhichStatusesHaveNoCountdown(t *testing.T) {
	book := permanentBook(t)
	forever, err := book.Lookup("toughened")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	timed, err := book.Lookup("haste")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	var set status.Set
	set.Apply(forever, 0)
	set.Apply(timed, 0)
	found := 0
	for _, entry := range set.Snapshot() {
		switch entry.ID {
		case "toughened":
			found++
			if !entry.Permanent {
				t.Error("the permanent status is not marked permanent in the snapshot")
			}
		case "haste":
			found++
			if entry.Permanent {
				t.Error("a timed status is marked permanent in the snapshot")
			}
			if entry.Remaining != timed.Duration {
				t.Errorf("the timed status has %d turns left, want %d", entry.Remaining, timed.Duration)
			}
		}
	}
	if found != 2 {
		t.Errorf("the snapshot covered %d of the 2 statuses applied", found)
	}
}

func TestPermanentDeclarationsAreRefused(t *testing.T) {
	cases := []struct {
		name    string
		kind    string
		wantErr string
	}{
		{
			"permanent and timed at once",
			`{"id": "odd", "category": "buff", "max_stacks": 1, "duration": 3, "permanent": true}`,
			"two different answers",
		},
		{
			"a permanent damage-over-time",
			`{"id": "rot", "category": "dot", "max_stacks": 1, "duration": 0, "permanent": true, "tick_power": 300}`,
			"whole battle",
		},
		{
			"a permanent regeneration",
			`{"id": "bloom", "category": "regen", "max_stacks": 1, "duration": 0, "permanent": true, "tick_power": 300}`,
			"whole battle",
		},
		{
			"a timed status with no duration",
			`{"id": "brief", "category": "buff", "max_stacks": 1, "duration": 0}`,
			"want at least 1",
		},
		{
			"a health modifier, which nothing reads",
			`{"id": "swell", "category": "buff", "max_stacks": 1, "duration": 2,
			  "modifiers": [{"target": "hp", "mode": "percent", "amount": 200}]}`,
			"nothing in the engine reads",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := status.ParseBook([]byte(
				`{"max_stacks": 5, "max_duration": 6, "kinds": [` + test.kind + `]}`))
			if err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("%s was refused with %q, want it to mention %q", test.name, err, test.wantErr)
			}
		})
	}
}

func permanentBook(t *testing.T) *status.Book {
	t.Helper()
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "toughened", "category": "buff", "max_stacks": 3, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 200}]},
	    {"id": "haste", "category": "buff", "max_stacks": 2, "duration": 3,
	     "modifiers": [{"target": "speed", "mode": "percent", "amount": 300}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	return book
}

// TestGroupedFilesEveryKindUnderItsCategory covers the three decisions Grouped
// makes, and they are all decisions somebody reading a listing would notice.
//
// Category order is the enum's rather than the book's, so two books declaring
// the same statuses in a different order still read the same way down a
// reference. Kind order inside a group is the book's, because that order is
// authored and a reference sorting it would be printing something nobody wrote.
// And a category nothing is declared in is left out rather than printed empty: a
// heading with no rows reads as a listing that failed to load them.
func TestGroupedFilesEveryKindUnderItsCategory(t *testing.T) {
	// Declared out of category order on purpose: the buff sits between the two
	// dots, so a Grouped that walked the book would come out in three groups
	// rather than two.
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5,
	  "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "fury", "category": "buff", "max_stacks": 3, "duration": 3,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": 300}]},
	    {"id": "burn", "category": "dot", "max_stacks": 2, "duration": 2, "tick_power": 800}
	  ]
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	groups := book.Grouped()
	if len(groups) != 2 {
		t.Fatalf("three kinds of two categories came out in %d groups: %+v", len(groups), groups)
	}
	if groups[0].Category != status.Dot || groups[1].Category != status.Buff {
		t.Errorf("the groups are %s then %s, want the enum's order",
			groups[0].Category, groups[1].Category)
	}
	// The book's order inside the group, which is not the order they were
	// declared in overall.
	if got := []string{groups[0].Kinds[0].ID, groups[0].Kinds[1].ID}; got[0] != "poison" || got[1] != "burn" {
		t.Errorf("the dots came out %v, want the order the book declares them in", got)
	}
	counted := 0
	for _, group := range groups {
		if len(group.Kinds) == 0 {
			t.Errorf("%s is a group with nothing in it", group.Category)
		}
		counted += len(group.Kinds)
	}
	if counted != len(book.Kinds()) {
		t.Errorf("the book holds %d kinds and the groups hold %d", len(book.Kinds()), counted)
	}
}
