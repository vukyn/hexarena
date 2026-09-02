package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// TestEveryRaiseTargetNamesAScreenInThisClient is what stops a raise from
// silently doing nothing.
//
// A screen asks for a draw.Target and this client's map turns it into one of its
// own views. A target with no entry is a keystroke that reads as broken: the
// screen did everything right, the reader pressed the key the footer names, and
// nothing happens.
//
// ⚠️ It walks screen.TargetCount rather than the map, because the failure being
// guarded against is a target somebody added over there and did not list here.
// Ranging over raiseTargets would ask the map whether it holds what it holds.
//
// ⚠️ **It is held total per client, which is the whole reason draw.Fight is
// sayable at all.** The squad catalogue is in internal/screen and raises Fight;
// the authoring tool turns that into a measurement of two squads and this client
// turns it into a battle. Neither meaning is the catalogue's to know — see
// landSquad and pairing.go.
func TestEveryRaiseTargetNamesAScreenInThisClient(t *testing.T) {
	for value := 1; value < draw.TargetCount; value++ {
		target := draw.Target(value)
		if _, known := raiseTargets[target]; !known {
			t.Errorf("draw.Target %v (%d) names no screen in this client, so a raise carrying "+
				"it does nothing", target, value)
		}
	}
	// And nothing beyond them, which is the other half of total: an entry for a
	// value the package does not declare is a target this client would answer to
	// and no screen could ask for.
	if got, want := len(raiseTargets), draw.TargetCount-1; got != want {
		t.Errorf("raiseTargets holds %d entries against the %d targets declared besides "+
			"NoTarget", got, want)
	}
	// NoTarget is what every action that is not a raise carries, so a screen for
	// it would be a screen reachable by every Quit and every Back.
	if _, known := raiseTargets[draw.NoTarget]; known {
		t.Error("draw.NoTarget names a screen, and it is what a non-raise action carries")
	}
}

// TestEverySubjectKindIsAppliedByThisClient is the other half of a total raise.
//
// A kind with no entry makes a raise hand the describer nothing, which draws an
// empty screen off a keystroke that did everything right — quieter than a
// missing target, because a screen with nothing on it reads as a screen with
// nothing on it.
//
// ⚠️ It walks screen.SubjectKindCount rather than the map, for the reason above.
func TestEverySubjectKindIsAppliedByThisClient(t *testing.T) {
	for value := 1; value < draw.SubjectKindCount; value++ {
		kind := draw.SubjectKind(value)
		if _, known := subjects[kind]; !known {
			t.Errorf("draw.SubjectKind %v (%d) is applied by nothing in this client, so a "+
				"raise carrying it draws an empty screen", kind, value)
		}
	}
	if got, want := len(subjects), draw.SubjectKindCount-1; got != want {
		t.Errorf("subjects holds %d entries against the %d kinds declared besides NoSubject",
			got, want)
	}
	if _, known := subjects[draw.NoSubject]; known {
		t.Error("draw.NoSubject is applied by an entry, and it is what a raise about nothing " +
			"carries")
	}
}

// TestEveryActionKindIsAppliedByThisClient is a behaviour table rather than a
// lookup, because navigate is a switch and there is no map to ask.
//
// Each arm drives navigate with an action of its kind and reads what the client
// did about it, and the table is held total against draw.KindCount so a seventh
// kind cannot arrive unhandled.
//
// ⚠️ **Stay is excluded and declared**, exactly as NoTarget and NoSubject are:
// it is the zero value and doing nothing is its definition rather than its
// defect. It is asserted at the bottom all the same, which is what makes leaving
// it out honest rather than an omission.
//
// ⚠️ **Ask and Pick share an arm and it asserts that nothing happens**, which is
// this client's honest answer rather than a gap: both are the authoring half of
// the vocabulary and no screen here can raise either.
// TestNoScreenInThisClientAsksOrPicks is what says so about behaviour; this arm
// says what the client does if one ever arrives anyway.
func TestEveryActionKindIsAppliedByThisClient(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	arms := map[draw.Kind]func(t *testing.T){
		draw.Back: func(t *testing.T) {
			m := base
			m.raisedFrom = screenTraits
			m.screen = screenStatuses
			after, _ := m.navigate(screenStatuses, draw.Action{Kind: draw.Back})
			if got := after.(model).screen; got != screenTraits {
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
			after, command := base.navigate(screenSkills,
				draw.Action{Kind: draw.Ask, Question: i18n.SkillFormDiscard})
			if got := after.(model); got.screen != base.screen {
				t.Errorf("an Ask moved to screen %v; this client draws no question", got.screen)
			}
			if command != nil {
				t.Error("an Ask asked for a command")
			}
		},
		draw.Pick: func(t *testing.T) {
			listing := base.enter(screenSkills)
			wanted := listing.skills.OpenAllowlist(listing.ctx(), draw.SkillFieldKeptForSpecies)
			after, command := listing.navigate(screenSkills,
				draw.Action{Kind: draw.Pick, Picker: wanted})
			if got := after.(model); got.screen != listing.screen {
				t.Errorf("a Pick moved to screen %v; this client draws no picker", got.screen)
			}
			if command != nil {
				t.Error("a Pick asked for a command")
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
	still, command := base.navigate(screenSkills, draw.Action{})
	if got := still.(model); got.screen != base.screen {
		t.Error("a Stay changed the screen")
	}
	if command != nil {
		t.Error("a Stay asked for a command")
	}
}

// TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack is the one-slot memory,
// asserted as a pair because either half alone passes for the wrong reason.
//
// Remembering without forgetting leaves a reader who arrived through a trait and
// then came back through the menu being sent to the trait; forgetting without
// remembering is a hard-coded way out.
func TestARaiseRemembersWhoAskedAndForgetsItOnTheWayBack(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	traits := m.enter(screenTraits)
	raised := key(t, traits, "?")
	if raised.screen != screenStatuses {
		t.Fatalf("? on the traits listing landed on screen %v", raised.screen)
	}
	if raised.raisedFrom != screenTraits {
		t.Errorf("the raise remembered screen %v as the way back, want the traits listing",
			raised.raisedFrom)
	}
	back := key(t, raised, "esc")
	if back.screen != screenTraits {
		t.Fatalf("esc from the statuses reference went to screen %v", back.screen)
	}
	if back.raisedFrom != screenMenu {
		t.Errorf("the way back survived being used: it still reads screen %v", back.raisedFrom)
	}
	// And a screen nobody raised goes to the menu, which is what every listing
	// the menu itself opens does.
	plain := key(t, m.enter(screenSpecies), "esc")
	if plain.screen != screenMenu {
		t.Errorf("esc from a screen reached through the menu went to screen %v", plain.screen)
	}
}

// TestAWayBackSurvivesTheScreenItRaised is the two-slot memory, and it is the
// test #228 had to write for the authoring tool because every other way-back
// test walks a chain one raise deep.
//
// ⚠️ **The defect needs the raise in between.** catalogue → battle → description
// → esc → esc: one slot answers the first three steps perfectly and then sends
// the reader to the menu, because the description's raise overwrote the only
// record there was. Collapsing model.raisedBy to an assignment leaves this
// client's whole suite green except this test.
func TestAWayBackSurvivesTheScreenItRaised(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	catalogue := m.enter(screenSquads)
	if len(catalogue.squads.Saved) == 0 {
		t.Fatal("the fixture catalogue is empty, so f raises nothing")
	}
	battle := key(t, catalogue, "f")
	if battle.screen != screenBattle {
		t.Fatalf("f on the catalogue landed on screen %v", battle.screen)
	}
	if battle.battle.Pending == nil || len(battle.battle.Pending.Options) == 0 {
		t.Fatalf("the battle opened with no turn for the player (%v), so ? raises nothing",
			battle.battle.Err)
	}
	described := key(t, battle, "?")
	if described.screen != screenBlurb {
		t.Fatalf("? in the battle landed on screen %v", described.screen)
	}
	returned := key(t, described, "esc")
	if returned.screen != screenBattle {
		t.Fatalf("esc from the description went to screen %v, want the battle", returned.screen)
	}
	out := key(t, returned, "esc")
	if out.screen != screenSquads {
		t.Errorf("esc from the battle went to screen %v, want the catalogue that raised it",
			out.screen)
	}
}

// TestABattleOpensOnTheSideTheRaiseNamed is the effect half of the Fight seam,
// and the half TestEverySubjectKindIsAppliedByThisClient cannot state.
//
// A SquadSubject names a side by **id** and this client keeps a **row**, so the
// applier is where the one becomes the other; and the pairing that row is turned
// into is where the two clients answer draw.Fight differently. A walk over the
// kinds proves the kind is applied by something. An applier that took whichever
// row sorted first, that was one off, or that handed the two sides to Open the
// wrong way round passes it completely.
//
// ⚠️ **Every row of a two-side catalogue rather than one**, because a cursor left
// on the last row cannot tell `+1` from correct — the row is clamped — which is
// the off-by-one #223 measured. And **both squads are named**, because the away
// side is the half a swap moves: home alone would be satisfied by a client that
// fielded the named side twice.
func TestABattleOpensOnTheSideTheRaiseNamed(t *testing.T) {
	base, _, _ := start(t, i18n.En)
	catalogue := base.enter(screenSquads)
	sides := catalogue.squads.Saved
	if len(sides) < 2 {
		t.Fatalf("the fixture catalogue holds %d sides; one row cannot tell a swap from "+
			"correct", len(sides))
	}
	for row, side := range sides {
		raised, _ := catalogue.navigate(screenSquads, draw.Action{
			Kind: draw.Raise, Target: draw.Fight,
			Subject: draw.Subject{Kind: draw.SquadSubject, ID: side.ID},
		})
		after := raised.(model)
		if after.screen != screenBattle {
			t.Fatalf("a raise about %q landed on screen %v", side.ID, after.screen)
		}
		if after.battle.Home.ID != side.ID {
			t.Errorf("a raise about %q opened the battle on %q as the home side",
				side.ID, after.battle.Home.ID)
		}
		if want := sides[(row+1)%len(sides)].ID; after.battle.Away.ID != want {
			t.Errorf("a raise about %q put %q on the other side, want %q",
				side.ID, after.battle.Away.ID, want)
		}
		// The player fights the side they named, which is what makes "home" mean
		// anything at all: a client that got the pairing right and the side wrong
		// would hand the reader the opponent's turns.
		if after.battle.Side != hex.SideAlly {
			t.Errorf("a raise about %q put the player on %v, and Take fields the home side "+
				"as the ally half", side.ID, after.battle.Side)
		}
	}

	// And a side the catalogue does not hold declines the whole trip rather than
	// opening a battle on whichever row `taking` happened to point at.
	stayed, _ := catalogue.navigate(screenSquads, draw.Action{
		Kind: draw.Raise, Target: draw.Fight,
		Subject: draw.Subject{Kind: draw.SquadSubject, ID: "no.such.side"},
	})
	if got := stayed.(model).screen; got != catalogue.screen {
		t.Errorf("a raise about a side nobody holds moved to screen %v", got)
	}
}

// TestAnEmptyCatalogueOpensNoBattle drives the seam's other end through the
// client's own path.
//
// A reader who has never run the authoring tool has no sides, and this client's
// menu offers a battle all the same — so what happens then is a decision rather
// than a crash: pairing hands Open two empty squads, Open reads that as no
// pairing, and the screen says a side has to be built. The refusal is in one
// place, which is why pairing does not branch on it.
func TestAnEmptyCatalogueOpensNoBattle(t *testing.T) {
	m := startEmpty(t, i18n.Vi)
	opened := m.enter(screenBattle)
	if opened.battle.Fight != nil {
		t.Error("an empty catalogue built a battle")
	}
	if opened.battle.Err != nil {
		t.Errorf("an empty catalogue reported %v, want no pairing rather than a refusal",
			opened.battle.Err)
	}
	if want := opened.text(i18n.SquadsEmpty); !strings.Contains(opened.screenContent(), want) {
		t.Errorf("the battle with no pairing does not say a side has to be built:\n%s",
			opened.screenContent())
	}
}

// TestTheMenuOpensEveryCatalogueItOffers is the wiring, pressed rather than
// read.
//
// Every entry is walked to with the arrow key a reader uses and opened with
// enter, because a menu whose target was wrong for one entry would draw a
// perfectly good screen — somebody else's.
func TestTheMenuOpensEveryCatalogueItOffers(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	for index, item := range menuItems {
		m := base.enter(screenMenu)
		for range index {
			m = key(t, m, "down")
		}
		if m.menu != index {
			t.Fatalf("walking down %d times left the cursor on entry %d", index, m.menu)
		}
		opened := key(t, m, "enter")
		if opened.screen != item.target {
			t.Errorf("entry %d (%q) opened screen %v, want %v",
				index, base.text(item.label), opened.screen, item.target)
		}
		if body := strings.TrimSpace(opened.screenContent()); body == "" {
			t.Errorf("entry %d drew nothing", index)
		}
	}
	// Seven catalogues and a battle, which is what this client offers. A count
	// rather than a spot check: an entry quietly dropped is a catalogue nothing
	// reaches, and the sweep would still pass because the screen is registered
	// there directly.
	if got, want := len(menuItems), 8; got != want {
		t.Errorf("the menu offers %d entries, want the %d this client was built to offer",
			got, want)
	}
}
