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
func TestRemoveTakesTheHeaviestStacksFirst(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 50)
	set.Apply(poison(), 300)
	set.Apply(poison(), 120)
	removed, damage := set.Remove("poison", 1)
	if removed != 1 || damage != 300 {
		t.Errorf("removing one stack took %d stacks worth %d, want 1 worth 300", removed, damage)
	}
	if got, want := set.TickAmount("poison"), int64(170); got != want {
		t.Errorf("the remaining tick is %d, want %d", got, want)
	}
	removed, damage = set.Remove("poison", 5)
	if removed != 2 || damage != 170 {
		t.Errorf("removing the rest took %d stacks worth %d, want 2 worth 170", removed, damage)
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
	stacks, damage := set.Consume("poison")
	if stacks != 3 || damage != 480 {
		t.Errorf("consuming took %d stacks worth %d, want 3 worth 480", stacks, damage)
	}
	if set.Has("poison") {
		t.Error("the poison survived being consumed")
	}
	if stacks, damage := set.Consume("burn"); stacks != 0 || damage != 0 {
		t.Errorf("consuming a status that is not there took %d worth %d", stacks, damage)
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

func TestCategoryNames(t *testing.T) {
	want := []string{"dot", "stat_debuff", "control", "buff", "shield", "regen"}
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
	for _, category := range []status.Category{status.Dot, status.StatDebuff, status.Control} {
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
		{"stacks over the limit", `{"max_stacks":3,"max_duration":6,"kinds":[{"id":"x","category":"dot","max_stacks":4,"duration":1,"tick_power":500}]}`, "over the limit of 3"},
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
