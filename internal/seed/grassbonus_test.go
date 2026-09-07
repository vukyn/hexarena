package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// grassBoard is two pairs on the shipped books, identical in every number and
// different only in what they are made of: the allies are both grass and so reach
// `grass_growth`, and the enemies carry two elements that share nothing and so
// reach no element bonus at all.
//
// ⚠️ The enemy pair is light and ice on purpose. Every other element now has a
// bonus of its own, and a pair sharing one would be a control that is also being
// paid — so the control has to be built out of the two elements only one shipped
// character carries, which are exactly the two that can reach no rung.
func grassBoard(t *testing.T) *battle.Battle {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	single := func(name string) element.Affinity {
		one, err := element.Parse(name)
		if err != nil {
			t.Fatalf("parse %q: %v", name, err)
		}
		affinity, err := element.Single(one)
		if err != nil {
			t.Fatalf("build the affinity for %q: %v", name, err)
		}
		return affinity
	}
	line := progression.Values{
		progression.HP: 2000, progression.Attack: 300, progression.Defense: 200,
		progression.Speed: 100, progression.Accuracy: 120, progression.Dodge: 20,
	}
	fight, err := battle.New(books, 3, []battle.Roster{
		{ID: "g1", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: line, Skills: []string{"bite"}},
		{ID: "g2", Side: hex.SideAlly, Slot: hex.Offset{Col: 1, Row: 1},
			Affinity: single("grass"), Stats: line, Skills: []string{"bite"}},
		{ID: "m1", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("light"), Stats: line, Skills: []string{"bite"}},
		{ID: "m2", Side: hex.SideEnemy, Slot: hex.Offset{Col: 1, Row: 1},
			Affinity: single("ice"), Stats: line, Skills: []string{"bite"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

func mustUnit(t *testing.T, fight *battle.Battle, id string) *battle.Unit {
	t.Helper()
	unit, ok := fight.Unit(id)
	if !ok {
		t.Fatalf("no unit is called %q", id)
	}
	return unit
}

// TestAGrassSquadWalksInWithMoreHealthThanItWasAuthoredWith is the whole of the
// grass bonus, read off the shipped data rather than off a fixture.
//
// The two pairs are the same stat line, so every number that separates them comes
// from the bonus. What is asserted is that the raise is real, that it is grass's
// alone, and that the squad is holding it rather than starting the battle short of
// it — which is the trap the enlistment order exists to avoid.
func TestAGrassSquadWalksInWithMoreHealthThanItWasAuthoredWith(t *testing.T) {
	fight := grassBoard(t)
	for _, id := range []string{"g1", "g2"} {
		unit := mustUnit(t, fight, id)
		if !unit.Statuses.Has("heartwood") {
			t.Fatalf("%s is grass beside another grass and is not carrying the bonus", id)
		}
		authored := unit.Base[progression.HP]
		if got := fight.MaxHP(unit); got <= authored {
			t.Errorf("%s has a maximum of %d against an authored line of %d: the bonus is "+
				"on the unit and is changing no number", id, got, authored)
		}
		if got, want := unit.HP, fight.MaxHP(unit); got != want {
			t.Errorf("%s opens on %d of %d health: a squad that built for a health bonus "+
				"would be starting the battle wounded by it", id, got, want)
		}
	}
	for _, id := range []string{"m1", "m2"} {
		unit := mustUnit(t, fight, id)
		if unit.Statuses.Has("heartwood") {
			t.Errorf("%s shares its element with nobody and is carrying the grass bonus", id)
		}
		if got, want := fight.MaxHP(unit), unit.Base[progression.HP]; got != want {
			t.Errorf("%s has a maximum of %d against an authored line of %d, and nothing "+
				"was awarded to it", id, got, want)
		}
	}
	grass, plain := mustUnit(t, fight, "g1"), mustUnit(t, fight, "m1")
	if fight.MaxHP(grass) <= fight.MaxHP(plain) {
		t.Errorf("the grass pair opens on %d and the pair that shares nothing opens on %d, "+
			"off one stat line", fight.MaxHP(grass), fight.MaxHP(plain))
	}
}

// TestTheGrassBonusIsTheOnlyOneThatMovesAMaximum is the reason this element got a
// mechanism where the other seven got a row of data.
//
// Every other element bonus lands on a stat the engine already read, so shipping
// one was a line of JSON. Grass needed Battle.MaxHP to stop reading the base line,
// and a term nothing reads is precisely what that refusal existed to prevent — so
// this asserts the shipped table has exactly one health term in it, and that it is
// the one the mechanism was built for. A second would not be a bug, but it would
// mean the mechanism has grown a second owner without anybody deciding to give it
// one.
func TestTheGrassBonusIsTheOnlyOneThatMovesAMaximum(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the shipped books: %v", err)
	}
	carriers := []string{}
	for _, kind := range books.Statuses.Kinds() {
		for _, term := range kind.Modifiers {
			if term.Target == modifier.HP {
				carriers = append(carriers, kind.ID)
				break
			}
		}
	}
	if len(carriers) != 1 || carriers[0] != "heartwood" {
		t.Errorf("the shipped statuses carrying a health term are %v, want just heartwood", carriers)
	}
}
