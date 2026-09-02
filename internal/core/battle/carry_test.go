package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// What the engine and the rating do with a figure that has saturated.
//
// `overkill` is the fixture these three tests share: a column skill whose power
// is nine quintillion, which is not a skill anybody would author and is exactly
// the input the arithmetic downstream of a power has to survive. Its damage
// against a bare target comes to about three quintillion, and three quintillion
// is the range where a plain `figure * ratio / 1000` stops being arithmetic and
// starts being a coin toss — the products below wrapped to negatives, and a
// negative on this path does not read as an error. It reads as a skill that does
// nothing.
//
// ⚠️ Every one of these is measured through a CONSEQUENCE — a unit that did or
// did not take damage, a skill Suggest did or did not pick — rather than through
// a figure read out of the rating. A rating is not observable and the choice it
// leads to is, which is the only reason these can be tests at all.

// splashBoard is a caster with one enemy in front of it and one behind that,
// stacked in the column the skill's shape runs down.
func splashBoard(t *testing.T) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"overkill"}},
		{ID: "front", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"lob"}},
		{ID: "back", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"lob"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	return fight
}

// TestASplashOffAnEnormousPowerStillLands is the engine's own half, and the
// failure it guards against is the quietest one in the set.
//
// A splashed target takes a share of the power, and the share used to be taken
// in a plain narrow product. Fed a power at the edge of the type that product
// wrapped to a negative, `Rules.damage` refused the negative on its first line,
// and the unit standing in the blast took nothing at all — no damage event, no
// error, no sign that anything had gone wrong. The unit in front of it died as
// expected, so the skill looked like it worked.
func TestASplashOffAnEnormousPowerStillLands(t *testing.T) {
	fight := splashBoard(t)
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Act("overkill", hex.Offset{Col: 3, Row: 1}); err != nil {
		t.Fatalf("overkill: %v", err)
	}

	dealt := map[string]int64{}
	for _, event := range fight.Drain() {
		if event.Kind == battle.Damaged {
			dealt[event.Target] += event.Amount
		}
	}
	if dealt["front"] <= 0 {
		t.Fatalf("the aimed-at unit took %d, so this board measures nothing",
			dealt["front"])
	}
	if dealt["back"] <= 0 {
		t.Errorf("the splashed unit took %d: a share of an enormous power is "+
			"still enormous, and a share that wraps lands as nothing",
			dealt["back"])
	}
	// Half the power, and the fixture's two targets are identical, so the splash
	// is worth about half the blow. Anything wildly off that is a wrap that
	// happened to stay positive rather than the share the pattern book declares.
	if dealt["back"] > dealt["front"] {
		t.Errorf("the splashed unit took %d against the aimed-at unit's %d, and a "+
			"splash is a fraction of a blow rather than more than one",
			dealt["back"], dealt["front"])
	}
}

// TestARatingSeesAnEnormousBlow is the rating's half of the same figure, and it
// is the one that changes what the opponent does.
//
// `expected` takes what one strike comes to and multiplies it by how many
// strikes connect. Both halves are needed separately — a wall of charges cancels
// whole strikes — so the multiplication is written out rather than left to
// Rules.Expected, and written out in a narrow product it wrapped. A blow that
// one-shots the board came back rated below a jab, and Suggest took the jab.
func TestARatingSeesAnEnormousBlow(t *testing.T) {
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"overkill", "jab"}},
		{ID: "front", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"lob"}},
		{ID: "back", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"lob"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if choice := chosen(t, fight); choice.Skill != "overkill" {
		t.Errorf("Suggest picked %q over a blow that kills the board outright, "+
			"which is what a rating that wrapped looks like from outside",
			choice.Skill)
	}
}

// TestARatingAimsWhereTheSplashCatchesMost is the third product, and the aim is
// the only thing that shows it.
//
// The rating's own copy of the splash share wrapped the same way the engine's
// did, and with a *negative* share a splashed enemy is priced as a cost rather
// than a gain — so the rating preferred the aim that caught fewer of them. The
// three enemies stand in one column: the middle cell's shape covers all three
// and either end's covers two.
//
// ⚠️ The right answer is deliberately not the aim the tie order reaches first.
// With the product wrapped Suggest falls to 3,0, so a fixture that wanted 3,0
// would pass either way.
func TestARatingAimsWhereTheSplashCatchesMost(t *testing.T) {
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("grass"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"overkill", "jab"}},
	}
	for _, row := range []int{0, 1, 2} {
		roster = append(roster, battle.Roster{
			ID: []string{"top", "middle", "bottom"}[row], Side: hex.SideEnemy,
			Slot: hex.Offset{Col: 2, Row: row}, Affinity: single("neutral"),
			Stats: stats(3000, 800, 400, 10), Skills: []string{"lob"}})
	}
	fight, err := battle.New(books(t), 7, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()

	choice := chosen(t, fight)
	if choice.Skill != "overkill" {
		t.Fatalf("Suggest picked %q, so this board says nothing about the aim",
			choice.Skill)
	}
	if want := (hex.Offset{Col: 3, Row: 1}); choice.Aim != want {
		t.Errorf("Suggest aimed at %v, want %v: that is the one cell whose shape "+
			"catches all three, and a splash share priced as a cost prefers the "+
			"aim that catches fewer", choice.Aim, want)
	}
}
