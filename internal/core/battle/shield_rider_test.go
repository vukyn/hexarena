package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// The two absolute cells this fixture uses. Aims are board cells rather than
// offsets from the caster, and hex.Place mirrors the enemy formation, so the
// ally's slot 2,1 is 2,1 and the foe's is acrossTheBoard (declared in
// rider_test.go).
var ownCell = hex.Offset{Col: 2, Row: 1}

// shieldedCast drives one duel and hands back the events of the ALLY's single
// cast and nothing else. The foe is faster, so it takes the turns in front and
// either raises a shield with them or spends them jabbing.
//
// ⚠️ Battle.Drain EMPTIES the buffer, so it is called after Begin, after every
// Advance and after every Act — the batch returned at the end then holds one
// cast. Draining twice around the cast would show the second call nothing, and
// not draining on the turns nobody is measuring lets the foe's events pile into
// the batch that is read.
func shieldedCast(t *testing.T, allySkill string, traits []string, shield bool) []battle.Event {
	t.Helper()
	fight := mustBattle(t, books(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 70),
			Skills: []string{allySkill}, Passives: traits},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"brace", "jab"}},
	})
	fight.Begin()
	fight.Drain()
	raised := false
	for turn := 0; ; turn++ {
		if turn > 20 {
			t.Fatal("the ally never got a turn")
		}
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit == "a" {
			if prompt.Skipped {
				t.Fatal("the ally's turn was taken from it, so nothing here is measuring a cast")
			}
			break
		}
		if prompt.Skipped {
			continue
		}
		// Brace once and then jab: a second charge would eat a second strike, and
		// every skill measured here strikes once.
		foeSkill, foeAim := "jab", ownCell
		if shield && !raised {
			foeSkill, foeAim, raised = "brace", acrossTheBoard, true
		}
		if err := fight.Act(foeSkill, foeAim); err != nil {
			t.Fatalf("the foe's %s: %v", foeSkill, err)
		}
		fight.Drain()
	}
	if shield && !raised {
		t.Fatal("the foe never raised its shield, so the blocked arm is not blocked")
	}
	if err := fight.Act(allySkill, acrossTheBoard); err != nil {
		t.Fatalf("the ally's %s: %v", allySkill, err)
	}
	return fight.Drain()
}

// wasBlocked fails unless the cast was eaten whole by the shield: a Blocked
// strike and no Damaged one. Without it every "nothing landed" assertion below
// would pass on a cast that never happened.
func wasBlocked(t *testing.T, what string, events []battle.Event) {
	t.Helper()
	if blocked := find(events, battle.Blocked); len(blocked) != 1 {
		t.Fatalf("%s: %d strikes were blocked, want 1: %+v", what, len(blocked), events)
	}
	if struck := find(events, battle.Damaged); len(struck) != 0 {
		t.Fatalf("%s: %d strikes landed, so the shield did not eat the cast", what, len(struck))
	}
}

// wasStruck is the control's counterpart: the same cast with no shield in front
// of it has to land, or an absence in the blocked arm says nothing.
func wasStruck(t *testing.T, what string, events []battle.Event) {
	t.Helper()
	if struck := find(events, battle.Damaged); len(struck) != 1 {
		t.Fatalf("%s: %d strikes landed, want 1: %+v", what, len(struck), events)
	}
	if blocked := find(events, battle.Blocked); len(blocked) != 0 {
		t.Fatalf("%s: %d strikes were blocked, so this arm is not the control", what, len(blocked))
	}
}

// appliedStatuses is which statuses landed, in the order the log names them.
func appliedStatuses(events []battle.Event) []string {
	out := make([]string, 0, 2)
	for _, event := range find(events, battle.StatusApplied) {
		out = append(out, event.Status)
	}
	return out
}

// TestAShieldStopsTheBlowAndTheWearButNotTheContamination is the rule, with both
// arms and a control, because any one of the three alone proves nothing.
//
// A shield stops the blow and the wear, but not the contamination: fire still
// burns through it and poison still gets on you, while a stat the blow never bent
// and a turn it never took are stopped with the strike.
//
// Three things are asserted per skill and all three are needed. The blocked arm
// says what survives. The unblocked arm is the control, so an absence upstairs is
// not "nothing ever lands from this skill". And a cancelled rider must produce no
// StatusResisted either — the roll must not happen at all, rather than happen and
// fail, since a rider that rolled would move the rng stream and would land on the
// day somebody widens the chance.
func TestAShieldStopsTheBlowAndTheWearButNotTheContamination(t *testing.T) {
	for _, test := range []struct {
		name     string
		skill    string
		status   string
		category string
		survives bool
	}{
		{"a poison rides through", "envenom", "poison", "dot", true},
		{"a burn rides through", "scorch", "burn", "dot", true},
		{"a stat debuff does not", "sap", "weaken", "stat_debuff", false},
		{"a control does not", "daze", "stun", "control", false},
		{"a taunt does not", "provoke", "taunting", "taunt", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			blocked := shieldedCast(t, test.skill, nil, true)
			wasBlocked(t, test.skill+" blocked", blocked)

			got := appliedStatuses(blocked)
			switch {
			case test.survives && (len(got) != 1 || got[0] != test.status):
				t.Errorf("a blocked %s applied %v, want just %s: a %s outlasts a shield",
					test.skill, got, test.status, test.category)
			case !test.survives && len(got) != 0:
				t.Errorf("a blocked %s applied %v, want nothing: a %s is stopped with the blow",
					test.skill, got, test.category)
			}
			if !test.survives {
				if refused := find(blocked, battle.StatusResisted); len(refused) != 0 {
					t.Errorf("a blocked %s rolled for its %s and lost: %+v — the roll must not happen at all",
						test.skill, test.category, refused)
				}
			}

			// The control. Same skill, same fixture, nothing in front of it.
			landed := shieldedCast(t, test.skill, nil, false)
			wasStruck(t, test.skill+" unblocked", landed)
			if got := appliedStatuses(landed); len(got) != 1 || got[0] != test.status {
				t.Fatalf("an unblocked %s applied %v rather than %s, so nothing above measured the shield",
					test.skill, got, test.status)
			}
		})
	}
}

// TestATraitsRiderGoesThroughAShieldOnTheSameRuleAsASkillsOwn is the half no
// shipped trait reaches.
//
// A trait's rider and a skill's own application are fed to one inflict on
// purpose, so they take the same roll, the same resistance and the same event; a
// rider surviving a block on a different rule would be a difference no reader
// could find on either. NOTHING SHIPPED DECLARES `applies` ON A TRAIT — every one
// of the eleven uses grants, resists, amplifies, replies or drains — so this
// branch is latent today and this test is the only thing that exercises it.
func TestATraitsRiderGoesThroughAShieldOnTheSameRuleAsASkillsOwn(t *testing.T) {
	for _, test := range []struct {
		name     string
		trait    string
		status   string
		category string
		survives bool
	}{
		{"a poison rides through", "venomous", "poison", "dot", true},
		{"a stat debuff does not", "enfeebling", "weaken", "stat_debuff", false},
		{"a control does not", "dazing", "stun", "control", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			// strike inflicts nothing of its own, so whatever is in the log is the
			// trait's and could not be anything else.
			blocked := shieldedCast(t, "strike", []string{test.trait}, true)
			wasBlocked(t, test.trait+" blocked", blocked)

			got := appliedStatuses(blocked)
			switch {
			case test.survives && (len(got) != 1 || got[0] != test.status):
				t.Errorf("a blocked strike from a %s unit applied %v, want just %s",
					test.trait, got, test.status)
			case !test.survives && len(got) != 0:
				t.Errorf("a blocked strike from a %s unit applied %v, want nothing", test.trait, got)
			}
			if !test.survives {
				if refused := find(blocked, battle.StatusResisted); len(refused) != 0 {
					t.Errorf("a blocked strike rolled the %s trait's %s and lost: %+v",
						test.trait, test.category, refused)
				}
			}

			landed := shieldedCast(t, "strike", []string{test.trait}, false)
			wasStruck(t, test.trait+" unblocked", landed)
			if got := appliedStatuses(landed); len(got) != 1 || got[0] != test.status {
				t.Fatalf("an unblocked strike from a %s unit applied %v rather than %s",
					test.trait, got, test.status)
			}
			// And the control on the trait itself: the same skill with no trait
			// inflicts nothing, so the arms above are reading the rider.
			plain := shieldedCast(t, "strike", nil, false)
			if got := appliedStatuses(plain); len(got) != 0 {
				t.Fatalf("strike applied %v on its own, so nothing here reads the trait", got)
			}
		})
	}
}

// TestAMissedStrikeDeliversNothingIncludingATick is the distinction the whole
// rule rests on.
//
// A block means the blow arrived and was stopped; a miss means nothing touched
// the target. So a missed strike delivers nothing, a damage-over-time included —
// and the two must not be collapsed, because a rule that read "did not damage"
// rather than "was stopped" would poison through thin air.
//
// feint carries an accuracy of one per mille, which is the least a damaging skill
// may declare (skill.resolve refuses nought outright: "deals damage but can never
// connect"). The assertion that the strike missed is therefore load-bearing rather
// than decorative — a seed that rolled the one in a thousand fails here instead of
// passing for the wrong reason.
func TestAMissedStrikeDeliversNothingIncludingATick(t *testing.T) {
	for _, shield := range []bool{false, true} {
		name := "with nothing in front of it"
		if shield {
			name = "with a shield in front of it"
		}
		t.Run(name, func(t *testing.T) {
			events := shieldedCast(t, "feint", nil, shield)
			if missed := find(events, battle.Missed); len(missed) != 1 {
				t.Fatalf("%d strikes missed, want 1: %+v", len(missed), events)
			}
			if struck := find(events, battle.Damaged); len(struck) != 0 {
				t.Fatalf("the strike landed, so this measures nothing about a miss")
			}
			// A miss is resolved before a charge is offered one, so a shield in
			// front of it is untouched: nothing is blocked, which is exactly why a
			// missed strike cannot reach the arm a blocked one does.
			if blocked := find(events, battle.Blocked); len(blocked) != 0 {
				t.Fatalf("a missed strike was also blocked: %+v", blocked)
			}
			if got := appliedStatuses(events); len(got) != 0 {
				t.Errorf("a missed feint applied %v, want nothing: a miss is not a block", got)
			}
			if refused := find(events, battle.StatusResisted); len(refused) != 0 {
				t.Errorf("a missed feint rolled for its poison: %+v", refused)
			}
		})
	}
	// The control: the same poison off a skill that connects does land, so the
	// absences above are the miss rather than the fixture.
	landed := shieldedCast(t, "envenom", nil, false)
	wasStruck(t, "envenom unblocked", landed)
	if got := appliedStatuses(landed); len(got) != 1 || got[0] != "poison" {
		t.Fatalf("an unblocked envenom applied %v rather than poison", got)
	}
}
