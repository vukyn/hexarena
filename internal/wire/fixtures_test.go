// The tests in this package are in package wire rather than wire_test, because
// the two totality guards walk the unexported tables — kindNames, bodyForKind
// and codeNames — and a walk is the whole point of them: ranging over a table
// and asking it whether it holds what it holds is what let five screens slip
// into the authoring client unmeasured.
package wire

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// fixturePassword is named rather than written into a struct literal, so a
// secret scanner reading this file finds a constant with an obvious purpose
// instead of a bare string beside a field called Password.
const fixturePassword = "the-cat-sat-on-the-mat"

// fixtureDigest is a hand-made data digest: the byte i at index i, which no real
// data will ever produce.
//
// ⚠️ It is **not** seed.DataDigest(), and that is the single most important
// thing about these fixtures. A golden whose hello carried the real digest would
// move on every balance commit in the repository while measuring nothing about
// the protocol, which makes it a merge-conflict generator rather than a record —
// this has happened twice to other goldens in this tree. The same rule is why
// the rosters below are written out rather than resolved from cast.json.
func fixtureDigest() Digest {
	var raw seed.Digest
	for index := range raw {
		raw[index] = byte(index)
	}
	return Digest{Digest: raw}
}

// fixtureSquad is a squad by reference — two placements, no stat line anywhere,
// which is the property that lets a client be trusted with one.
func fixtureSquad() placement.Squad {
	return placement.Squad{
		ID:   "fixture.squad",
		Name: "Fixture",
		Units: []placement.Placement{
			{
				ID:        "fixture.front",
				Character: "fixture.character.one",
				Level:     60,
				Stage:     "fixture.stage.two",
				Slot:      hex.Offset{Col: 1, Row: 1},
				Skills:    []string{"fixture.strike", "fixture.guard", "fixture.mend", "fixture.hex"},
				Passives:  []string{"fixture.trait"},
			},
			{
				ID:        "fixture.back",
				Character: "fixture.character.two",
				Level:     60,
				Slot:      hex.Offset{Col: 0, Row: 2},
				Skills:    []string{"fixture.strike", "fixture.guard", "fixture.mend", "fixture.hex"},
			},
		},
	}
}

// fixtureRoster is two resolved units, one a side, in the order a battle would
// enlist them — which is the order that decides a speed tie and therefore is not
// an arbitrary choice even in a fixture.
func fixtureRoster(t *testing.T) []battle.Roster {
	t.Helper()
	water, err := element.Single(element.Water)
	if err != nil {
		t.Fatalf("build the fixture affinity: %v", err)
	}
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("build the fixture affinity: %v", err)
	}
	stats := progression.Values{
		progression.HP:       9000,
		progression.Attack:   500,
		progression.Defense:  400,
		progression.Speed:    300,
		progression.Accuracy: 900,
		progression.Dodge:    100,
	}
	return []battle.Roster{
		{
			ID:       "fixture.ally",
			Name:     "Ally",
			Side:     hex.SideAlly,
			Slot:     hex.Offset{Col: 1, Row: 1},
			Affinity: water,
			Stats:    stats,
			Skills:   []string{"fixture.strike", "fixture.guard"},
			Passives: []string{"fixture.trait"},
		},
		{
			ID:       "fixture.enemy",
			Name:     "Enemy",
			Side:     hex.SideEnemy,
			Slot:     hex.Offset{Col: 1, Row: 1},
			Affinity: fire,
			Stats:    stats,
			Skills:   []string{"fixture.strike", "fixture.hex"},
		},
	}
}

// fixtureEvents is the run of events one turn produced, hand-written. The Turn
// fixture's digest is computed from these rather than written out as a constant,
// which is deliberate: it means the golden pins DigestEvents' **framing** as
// well as the message's shape, and a change to either shows up in the same diff.
// Balance changes cannot move it, because no shipped datum is read here.
func fixtureEvents() []battle.Event {
	return []battle.Event{
		{Kind: battle.SkillUsed, At: 1_000_000, Turn: 4, Actor: "fixture.ally", Skill: "fixture.strike", Cell: hex.At(hex.Offset{Col: 4, Row: 1})},
		{Kind: battle.Damaged, At: 1_000_000, Turn: 4, Actor: "fixture.ally", Target: "fixture.enemy", Skill: "fixture.strike", Amount: 1200, Before: 9000, Remaining: 7800, Strike: 1},
	}
}

// messageFixtures is one hand-written message per kind, keyed by kind so a walk
// over KindCount can say which kind has none. Every table in this suite that
// needs a message reads it from here: the round-trip, the golden and the
// password sweep, so a kind added to the protocol has one fixture to write and
// three tests that immediately measure it.
func messageFixtures(t *testing.T) map[Kind]Body {
	t.Helper()
	digest, err := DigestEvents(fixtureEvents())
	if err != nil {
		t.Fatalf("digest the fixture events: %v", err)
	}
	return map[Kind]Body{
		KindHello: &Hello{
			Version: Version{Protocol: Protocol, Build: "fixture-build", Data: fixtureDigest()},
			Squad:   fixtureSquad(),
			Name:    "Fixture Player",
			// A room with a password, because that is the case with something to
			// get wrong: the golden shows the password crossing in the clear,
			// which is what the design record says it does.
			Password: fixturePassword,
		},
		KindAct:  &Act{Skill: "fixture.strike", Aim: hex.At(hex.Offset{Col: 4, Row: 1})},
		KindPass: &Pass{},
		KindWelcome: &Welcome{
			Format:    Format3v3,
			Battles:   3,
			Allowance: 90,
			// A cap that is nobody's default, so the golden's two numbers cannot
			// be read for each other and neither is a copy of a constant this
			// package does not own.
			TurnCap: 200,
			Seat:    SeatGuest,
		},
		KindRefused: &Refused{Code: CodeDataMismatch},
		KindStart: &Start{
			Seed:   11,
			Roster: fixtureRoster(t),
			Side:   hex.SideEnemy,
			Battle: 2,
		},
		KindTurn: &Turn{
			Decision: battle.Decision{
				Unit:  "fixture.ally",
				Turn:  4,
				Skill: "fixture.strike",
				Aim:   hex.At(hex.Offset{Col: 4, Row: 1}),
			},
			Events: digest,
		},
		// The one closure there is. A Closed carrying ClosureNone would be a
		// room declining to say why the match ended, so the fixture is the
		// value a room may actually send.
		KindClosed: &Closed{Reason: ClosureLeft},
	}
}
