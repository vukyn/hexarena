package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// TestEveryRaiseTargetNamesAScreenInThisClient is what stops a raise from
// silently doing nothing.
//
// A screen asks for a draw.Target and this client's map turns it into one of its
// own views. A target with no entry is a keystroke that reads as broken: the
// screen did everything right, the reader pressed the key the footer names, and
// nothing happens — the same shape as a screen slipping out of everyScreen,
// which this repository has recorded five times.
//
// ⚠️ It walks screen.TargetCount rather than the map, because the failure being
// guarded against is a target somebody added over there and did not list here.
// Ranging over raiseTargets would ask the map whether it holds what it holds.
func TestEveryRaiseTargetNamesAScreenInThisClient(t *testing.T) {
	for value := 1; value < draw.TargetCount; value++ {
		target := draw.Target(value)
		if _, known := raiseTargets[target]; !known {
			t.Errorf("draw.Target %v (%d) names no screen in this client, so a raise carrying it does nothing",
				target, value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the package does not declare is a target this client would answer to
	// and no screen could ask for.
	if got, want := len(raiseTargets), draw.TargetCount-1; got != want {
		t.Errorf("raiseTargets holds %d entries against the %d targets declared besides NoTarget",
			got, want)
	}
	// NoTarget is what every action that is not a raise carries, so a screen for
	// it would be a screen reachable by every Quit and every Back.
	if _, known := raiseTargets[draw.NoTarget]; known {
		t.Error("draw.NoTarget names a screen, and it is what a non-raise action carries")
	}
}

// TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack is the one-slot memory,
// asserted as a pair because either half alone passes for the wrong reason.
//
// Remembering without forgetting leaves a reader who arrived through a trait and
// then came back through the menu being sent to the trait; forgetting without
// remembering is the hard-coded menu this step exists to remove.
//
// It goes through navigate rather than through a screen's keys so that it says
// what it is about — the per-screen paths are driven with real keystrokes in
// chart_test.go and trait_status_test.go.
func TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)

	raised, _ := m.navigate(screenPassives, draw.Action{
		Kind: draw.Raise, Target: draw.Statuses,
		Subject: draw.Subject{Kind: draw.StatusSubject, ID: firstStatusID(t, m)},
	})
	after, ok := raised.(model)
	if !ok {
		t.Fatalf("navigate returned %T, want the model", raised)
	}
	if after.screen != screenStatuses {
		t.Fatalf("a raise of the statuses reference landed on screen %v", after.screen)
	}
	if after.raisedFrom != screenPassives {
		t.Errorf("the raise remembered screen %v as the way back, want the traits listing",
			after.raisedFrom)
	}

	back, _ := after.navigate(screenStatuses, draw.Action{Kind: draw.Back})
	returned := back.(model)
	if returned.screen != screenPassives {
		t.Fatalf("back from a raised screen went to screen %v", returned.screen)
	}
	if returned.raisedFrom != screenMenu {
		t.Errorf("the way back survived being used: it still reads screen %v", returned.raisedFrom)
	}

	// And a screen nobody raised goes to the menu, which is what the four
	// listings with no raiser above them have always done.
	plain, _ := m.navigate(screenBuilds, draw.Action{Kind: draw.Back})
	if got := plain.(model).screen; got != screenMenu {
		t.Errorf("back from a screen reached through the menu went to screen %v", got)
	}
}

// TestEverySubjectKindIsAppliedByThisClient is the other half of a total raise,
// and it is the one #203 could not write.
//
// A screen names what a raise is about with a draw.Subject and this client's
// applier puts it where it belongs. A kind with no entry is a keystroke that
// reads as broken in a quieter way than a missing target does: the screen opens,
// draws nothing, and looks like a screen with nothing on it rather than like a
// raise that went wrong.
//
// ⚠️ That is exactly what `Action.Focus` was — one undeclared case. It answered
// only the statuses reference, every other target reported not-found, and a
// declined raise is silent by design, so a subject aimed anywhere else was
// indistinguishable from a key nobody pressed. Nothing counted the cases, so
// nothing could walk them.
//
// ⚠️ It walks screen.SubjectKindCount rather than the map, because the failure
// being guarded against is a kind somebody added over there and did not list
// here. Ranging over subjects would ask the map whether it holds what it holds.
func TestEverySubjectKindIsAppliedByThisClient(t *testing.T) {
	for value := 1; value < draw.SubjectKindCount; value++ {
		kind := draw.SubjectKind(value)
		if _, known := subjects[kind]; !known {
			t.Errorf("draw.SubjectKind %v (%d) is applied by nothing in this client, so a raise "+
				"carrying it hands the describer nothing and draws an empty screen", kind, value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the package does not declare is a subject this client would apply and
	// no screen could ask for.
	if got, want := len(subjects), draw.SubjectKindCount-1; got != want {
		t.Errorf("subjects holds %d entries against the %d kinds declared besides NoSubject",
			got, want)
	}
	// NoSubject is what every raise about nothing carries, so an applier for it
	// would be one every Quit and every Back could reach.
	if _, known := subjects[draw.NoSubject]; known {
		t.Error("draw.NoSubject is applied by an entry, and it is what a raise about nothing carries")
	}
}

// TestEveryScreenThatAsksAnswersItsOwnQuestion is the third totality walk in
// this file, and it guards the quietest of the three failures.
//
// A screen raises a guard, the reader reads the question and presses y, and the
// question comes down. If the asking screen has no entry in confirmedBy, that is
// **all** that happens: the work is not discarded, the squad is not deleted, and
// nothing on screen says the answer went nowhere — a keystroke that reads as
// having worked. A missing target draws no screen and a missing subject applier
// draws an empty one; this one draws the screen the reader was already looking
// at.
//
// ⚠️ It walks guardAskers rather than the map, for the reason the two above walk
// their counts: the failure being guarded against is a screen that grew a
// question and no answer, and ranging over confirmedBy would ask the map whether
// it holds what it holds.
//
// ⚠️ **And it proves presence, not effect.** A dispatch entry that exists and
// does nothing passes this completely — the same blind spot #207 measured on an
// applier table. What holds the other half is one behaviour test per confirm:
// TestLeavingAnEditedFormAsksFirst in tui_test.go for the character form, and
// the four in guard_test.go for the other four.
func TestEveryScreenThatAsksAnswersItsOwnQuestion(t *testing.T) {
	for _, asked := range guardAskers {
		if _, known := confirmedBy[asked]; !known {
			t.Errorf("screen %v raises a guard and answers none, so a confirmed y on it "+
				"takes the question down and does nothing", asked)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// screen that never asks is an answer to a question nobody can raise.
	if got, want := len(confirmedBy), len(guardAskers); got != want {
		t.Errorf("confirmedBy holds %d entries against the %d screens that ask", got, want)
	}
}

// TestEveryPickDestinationLandsSomewhere is the fourth totality walk in this
// file, and the widest of the four: ten cases against the guard's four.
//
// A picker is closed with enter, the list comes down, and the reader believes
// they chose. If the destination has no entry in pickedInto that is **all** that
// happens — the field is unchanged and nothing on screen says the answer went
// nowhere. It is confirmedBy's failure exactly, and it is easier to make: a
// screen grows one guard and it is noticed, while the skill form alone raises
// six pickers that differ in nothing but which field they fill.
//
// ⚠️ **There are three destination vocabularies now and all three are walked.**
// Six of the ten followed the skill form into internal/screen as
// draw.SkillsPick and two followed the squad builder as draw.SquadsPick; the two
// naming a screen still in this package are pickDest. The map is keyed by `any`
// because PickState carries any of them, so a walk over one count alone would
// leave the others free to grow an entry in silence.
//
// ⚠️ It walks the counts rather than the map, for the reason the three walks
// above walk theirs — and here a count is doing more than a list would. Adding a
// destination is adding a constant, and a constant added above the count enters
// this walk without anybody remembering to enrol it, which is a failure
// guardAskers' hand-written list can still have.
//
// ⚠️ **And it proves presence, not effect.** An entry that exists and writes the
// wrong field passes this completely, which is the #207 shape and the one #214
// measured on the guard. What holds the other half is one behaviour test per
// destination, driven through the real keys: TestEachAllowlistPickLandsInItsOwnField
// and TestTheCharacterFormsTwoPicksLandInTheirOwnFields in picked_test.go for
// the two still here, screen.TestEverySkillsPickDestinationWritesItsOwnField and
// screen.TestEverySquadsPickDestinationWritesItsOwnField for the eight on the
// other side of the boundary.
func TestEveryPickDestinationLandsSomewhere(t *testing.T) {
	for value := 1; value < int(pickDestCount); value++ {
		into := pickDest(value)
		if _, known := pickedInto[into]; !known {
			t.Errorf("pickDest %d lands nowhere in this client, so a picker closed with "+
				"enter on it takes the list down and writes nothing", value)
		}
	}
	for value := 1; value < int(draw.SkillsPickCount); value++ {
		into := draw.SkillsPick(value)
		if _, known := pickedInto[into]; !known {
			t.Errorf("draw.SkillsPick %d lands nowhere in this client, so a picker closed "+
				"with enter on it takes the list down and writes nothing", value)
		}
	}
	for value := 1; value < int(draw.SquadsPickCount); value++ {
		into := draw.SquadsPick(value)
		if _, known := pickedInto[into]; !known {
			t.Errorf("draw.SquadsPick %d lands nowhere in this client, so a picker closed "+
				"with enter on it takes the list down and writes nothing", value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value none of the three enums declares is a landing no picker could reach.
	want := int(pickDestCount) - 1 + int(draw.SkillsPickCount) - 1 + int(draw.SquadsPickCount) - 1
	if got := len(pickedInto); got != want {
		t.Errorf("pickedInto holds %d entries against the %d destinations the three enums "+
			"declare besides their zeros", got, want)
	}
	// The two zero values are what a picker with no destination carries, so a
	// landing for either would be one every un-destined picker fell into.
	if _, known := pickedInto[pickNowhere]; known {
		t.Error("pickNowhere lands somewhere, and it is what a picker with no destination carries")
	}
	if _, known := pickedInto[draw.SkillsPickNothing]; known {
		t.Error("draw.SkillsPickNothing lands somewhere, and it names no field")
	}
	if _, known := pickedInto[draw.SquadsPickNothing]; known {
		t.Error("draw.SquadsPickNothing lands somewhere, and it names no field")
	}
}

// TestEveryActionKindIsAppliedByThisClient is the fifth totality walk, and it
// arrived with the two kinds that made it necessary.
//
// draw.Action.Kind carried four meanings for six screens and now carries six:
// Ask and Pick came with the skill form, which is the first moved screen with
// something to lose and the first with lists to fill in. A client that silently
// ignored one would swallow a keystroke — the question never appears, or the
// list never opens — which is the shape TODO.md records five times and #207,
// #214, #216 and #218 each measured again.
//
// ⚠️ **It is a behaviour table rather than a lookup**, because navigate is a
// switch and there is no map to ask. Each arm drives navigate with an action of
// its kind and reads what the client did about it, and the table is held total
// against draw.KindCount so a seventh kind cannot arrive unhandled.
//
// ⚠️ **Stay is excluded and declared**, exactly as pickNowhere and NoTarget are:
// it is the zero value and doing nothing is its definition rather than its
// defect.
//
// ⚠️ **And a count proves a kind is handled, never that it is handled right.**
// Every arm here is also pressed as a real key somewhere: chart_test.go and
// browse_test.go for Back and Raise, screen.TestALetterIsTextWhileTheSkillFilterHasTheKeyboard
// for Quit, guard_test.go's TestDiscardingAHalfWrittenSkillEmptiesTheFormAndStays
// for Ask, and picked_test.go for Pick.
func TestEveryActionKindIsAppliedByThisClient(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	arms := map[draw.Kind]func(t *testing.T){
		draw.Back: func(t *testing.T) {
			m := base
			m.raisedFrom = screenPassives
			m.screen = screenStatuses
			after, _ := m.navigate(screenStatuses, draw.Action{Kind: draw.Back})
			if got := after.(model).screen; got != screenPassives {
				t.Errorf("a Back landed on screen %v, want the screen that raised it", got)
			}
		},
		draw.Quit: func(t *testing.T) {
			_, command := base.navigate(screenSkills, draw.Action{Kind: draw.Quit})
			if !quits(command) {
				t.Error("a Quit did not end the program")
			}
		},
		draw.Raise: func(t *testing.T) {
			after, _ := base.navigate(screenElements,
				draw.Action{Kind: draw.Raise, Target: draw.Chart})
			if got := after.(model).screen; got != screenChart {
				t.Errorf("a Raise of the chart landed on screen %v", got)
			}
		},
		draw.Ask: func(t *testing.T) {
			after, _ := base.navigate(screenSkills,
				draw.Action{Kind: draw.Ask, Question: i18n.SkillFormDiscard})
			asked := after.(model)
			if asked.guard == nil {
				t.Fatal("an Ask raised no question")
			}
			if asked.guard.question != i18n.SkillFormDiscard {
				t.Errorf("the pending question is key %d, want the one the action carried",
					asked.guard.question)
			}
			if asked.guard.asked != screenSkills {
				t.Errorf("the question was filed under screen %v, want the screen that asked",
					asked.guard.asked)
			}
			if asked.screen != base.screen {
				t.Errorf("an Ask moved to screen %v; a question is drawn over what is in front",
					asked.screen)
			}
		},
		draw.Pick: func(t *testing.T) {
			listing := base.enter(screenSkills)
			wanted := listing.skills.OpenAllowlist(listing.ctx(), draw.SkillFieldKeptForSpecies)
			after, _ := listing.navigate(screenSkills, draw.Action{Kind: draw.Pick, Picker: wanted})
			opened := after.(model)
			if opened.picker == nil {
				t.Fatal("a Pick put no list in front")
			}
			if opened.picker != wanted {
				t.Error("the list in front is not the one the screen built, so its rows and " +
					"its destination are somebody else's")
			}
			if opened.picker.Into != draw.SkillsPickKinds {
				t.Errorf("the raised list lands at %v", opened.picker.Into)
			}
		},
	}
	// Stay is the zero and is deliberately outside the table; every other kind
	// has to be in it, and the count is what says so rather than this list.
	if got, want := len(arms), draw.KindCount-1; got != want {
		t.Fatalf("this table covers %d kinds against the %d declared besides Stay — a kind "+
			"nothing here drives is a kind this client may be swallowing", got, want)
	}
	for value := 1; value < draw.KindCount; value++ {
		kind := draw.Kind(value)
		arm, covered := arms[kind]
		if !covered {
			t.Errorf("draw.Kind %v (%d) is driven by nothing here", kind, value)
			continue
		}
		t.Run(kind.String(), arm)
	}
	// And Stay really is a no-op, which is what makes leaving it out honest
	// rather than an omission.
	still, command := base.navigate(screenSkills, draw.Action{})
	if got := still.(model); got.screen != base.screen || got.guard != nil || got.picker != nil {
		t.Error("a Stay changed something")
	}
	if command != nil {
		t.Error("a Stay asked for a command")
	}
}

// TestARaiseAboutASquadOpensTheFightOnThatSquad is the effect half of the newest
// subject applier, and the half TestEverySubjectKindIsAppliedByThisClient cannot
// state.
//
// A SquadSubject names a squad by **id** and fightScreen.home is an **index**, so
// this client is where the one becomes the other. The walk one function up proves
// the kind is applied by something; an applier that wrote whichever row sorted
// first, or that was one off, passes it completely.
//
// ⚠️ **Every row of a two-squad catalogue, rather than one.** A cursor left on the
// last row cannot tell `+1` from correct — the index is clamped — which is exactly
// what TestTheCatalogueStillFightsTheSquadUnderItsCursor in fight_test.go cannot
// see, since it deliberately reads the second of two.
func TestARaiseAboutASquadOpensTheFightOnThatSquad(t *testing.T) {
	base, _, _ := start(t, i18n.En)
	base = twoSquadsSaved(t, base)
	for row, squad := range base.squad.Saved {
		raised, _ := base.navigate(screenSquads, draw.Action{
			Kind: draw.Raise, Target: draw.Fight,
			Subject: draw.Subject{Kind: draw.SquadSubject, ID: squad.ID},
		})
		after := raised.(model)
		if after.screen != screenFight {
			t.Fatalf("a raise about %q landed on screen %v", squad.ID, after.screen)
		}
		if after.fight.home != row {
			t.Errorf("a raise about %q opened the fight on row %d, want %d",
				squad.ID, after.fight.home, row)
		}
	}

	// And a squad the catalogue does not hold declines the whole trip rather than
	// opening the fight on whichever row `home` happened to be pointing at, which
	// is what landStatus does with a status the book has lost.
	stayed, _ := base.navigate(screenSquads, draw.Action{
		Kind: draw.Raise, Target: draw.Fight,
		Subject: draw.Subject{Kind: draw.SquadSubject, ID: "no.such.squad"},
	})
	if got := stayed.(model).screen; got != base.screen {
		t.Errorf("a raise about a squad nobody holds moved to screen %v", got)
	}
}

// TestARaiseAboutNothingStillArrives is the case the loop above cannot state: a
// Raise carrying no subject is ordinary rather than a subject nobody applied.
//
// The elements listing opening the chart is a whole screen rather than a thing on
// one, so it names none — and if applySubject treated the zero kind as unknown,
// that raise would decline and the g key would silently stop working.
func TestARaiseAboutNothingStillArrives(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	after, _ := m.navigate(screenElements, draw.Action{Kind: draw.Raise, Target: draw.Chart})
	if got := after.(model).screen; got != screenChart {
		t.Errorf("a raise carrying no subject landed on screen %v, want the chart", got)
	}
}

// TestARaiseNobodyCanLandDeclinesTheWholeTrip is the staying-put the traits
// listing used to do for itself.
//
// A subject the raised screen cannot find must leave the reader where they are,
// rather than opening the reference on whatever its cursor happened to be on:
// that would answer a question nobody asked, and it would read as the jump
// working.
func TestARaiseNobodyCanLandDeclinesTheWholeTrip(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m.screen = screenPassives

	after, _ := m.navigate(screenPassives, draw.Action{
		Kind: draw.Raise, Target: draw.Statuses,
		Subject: draw.Subject{Kind: draw.StatusSubject, ID: "no_such_status"},
	})
	stayed := after.(model)
	if stayed.screen != screenPassives {
		t.Errorf("a raise carrying an unfindable id moved to screen %v", stayed.screen)
	}
	if stayed.raisedFrom != screenMenu {
		t.Errorf("a declined raise wrote screen %v as a way back", stayed.raisedFrom)
	}
}

// firstStatusID is a status the reference really holds, so a focus that fails is
// a fault rather than a fixture.
func firstStatusID(t *testing.T, m model) string {
	t.Helper()
	for _, row := range m.statuses.Rows {
		if !row.Heading {
			return row.Kind.ID
		}
	}
	t.Fatal("the statuses reference holds no status, so a focus measures nothing")
	return ""
}
