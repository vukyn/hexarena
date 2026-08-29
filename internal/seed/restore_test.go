package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestEveryShippedRestoreActuallyHeals is the engine fix meeting the shipped
// data, and it is written over the book rather than over one skill.
//
// internal/core/battle proves both aims pay the same restore; this proves the
// skills in the book that declare one are no longer inert. **Both of them are
// self-aimed**, which is exactly why the defect survived: the payout sat inside
// resolveAgainst, Act returns before that for a Target: Self skill, and no
// shipped skill took the other path. `synthesis` — whose entire body is a nine
// hundred restore on itself — was castable, described, priced by the opponent
// and did nothing at all.
//
// It walks the book instead of naming synthesis so that a restore authored onto
// a third skill tomorrow is covered the day it is written, and so a shipped
// skill losing its restores is a failure here rather than a test quietly
// agreeing with whatever the data now says.
//
// Hand-played, for the reason TestTheShippedRegenerationHeals is: Suggest does
// price a restore, so it *would* choose one — and a test that let it choose
// would be measuring the rating rather than the engine.
func TestEveryShippedRestoreActuallyHeals(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	restoring := make([]skill.Skill, 0, 2)
	for _, declared := range books.Skills.Skills() {
		if declared.Restores > 0 {
			restoring = append(restoring, declared)
		}
	}
	if len(restoring) == 0 {
		t.Fatal("no shipped skill restores anything, so this proves nothing")
	}
	for _, declared := range restoring {
		t.Run(declared.ID, func(t *testing.T) {
			healed := castRestoringSkill(t, books, declared)
			if healed <= 0 {
				t.Errorf("%q declares restores %d and healed %d",
					declared.ID, declared.Restores, healed)
			}
		})
	}
}

// castRestoringSkill fields a caster that carries the skill, hurts it, takes one
// turn with it, and returns what the log says was healed.
//
// The health is read off the `healed` events rather than off the unit, because a
// unit's health can move for reasons this test is not about — and the event is
// what a renderer reads, so a restore the log does not carry is a restore a
// player cannot see happening.
func castRestoringSkill(t *testing.T, books battle.Books, declared skill.Skill) int64 {
	t.Helper()
	fight, err := battle.New(books, 3, []battle.Roster{
		{ID: "healer", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, declared.Element.String()),
			Stats:    benchStats(3000, 800, 400, 200),
			Skills:   []string{declared.ID}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, declared.Element.String()),
			Stats:    benchStats(4800, 100, 300, 1),
			Skills:   []string{declared.ID}},
	})
	if err != nil {
		t.Fatalf("new battle carrying %q: %v", declared.ID, err)
	}
	fight.Begin()
	fight.Drain()
	healer, ok := fight.Unit("healer")
	if !ok {
		t.Fatal("no healer in the battle")
	}
	// Hurt far enough that the payout has room: heal clamps at full health, so a
	// caster near the top would report a truncated figure and a caster at the top
	// would report nothing while the engine was working.
	healer.HP = 900
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "healer" {
		t.Fatalf("the turn did not go to the healer: %+v", prompt)
	}
	if err := fight.Act(declared.ID, healer.Cell); err != nil {
		t.Fatalf("act %q: %v", declared.ID, err)
	}
	var healed int64
	for _, event := range fight.Drain() {
		if event.Kind == battle.Healed {
			healed += event.Amount
		}
	}
	return healed
}
