package main

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestABracketIsTheKeystrokeItLooksLike is the premise the whole alias rests on,
// and it is asserted rather than assumed for a reason this repository has already
// paid for once.
//
// A bare space stringifies as "space" and not " ", because uv.Key.String returns
// Text only when it is not a single space — so every `case " "` in this client
// compiled fine and matched nothing. A bracket is a printable key of exactly the
// same shape, and if its String came out named (a "leftbracket" or a "[" that is
// really a keystroke spelling) then `case "pgup", "["` would be a branch nothing
// can reach: it would compile, it would read as working, and no test that does
// not press the key could tell. So the mapping is measured, at the same place the
// helper below relies on it.
// ⚠️ It reads keyPresses rather than building its own keystroke, and that is the
// second half of the guard. The table test below presses two keys and demands they
// agree, so a helper that sent the page key under BOTH names would satisfy it
// while proving nothing at all — the vacuity is in the fixture, where an assertion
// cannot see it. Measuring the entry the presses actually go through is what ties
// the claim to the thing pressed.
func TestABracketIsTheKeystrokeItLooksLike(t *testing.T) {
	for _, want := range []string{"[", "]"} {
		press, known := keyPresses[want]
		if !known {
			t.Fatalf("the helper has no entry for %q, so nothing can press it", want)
		}
		if got := press.String(); got != want {
			t.Errorf("the helper's %q sends a keypress stringifying as %q, so every case "+
				"matching %q is unreachable and the whole alias is dead", want, got, want)
		}
	}
	// And the two things that must NOT claim a bracket, because both sit in front
	// of a switch that would otherwise answer one. A bracket is not a digit, so a
	// picker's number field refuses it; a bracket is not a save key, so the battle
	// screen's isSaveKey lets it through to the log.
	for _, press := range []tea.KeyPressMsg{{Code: '[', Text: "["}, {Code: ']', Text: "]"}} {
		if numberKey(press) {
			t.Errorf("numberKey takes %q, so the picker's typed field would swallow it",
				press.String())
		}
		if isSaveKey(press) {
			t.Errorf("isSaveKey takes %q, so the battle screen would save instead of scrolling",
				press.String())
		}
	}
}

// scrollSite is one place in this client where a frame is walked with a page key.
//
// The table exists so a fourth one cannot be added without noticing: the whole
// argument for the brackets is that they are the same one idea reached by keys
// every keyboard has, and a site that scrolls with only half the vocabulary is
// exactly the second vocabulary that argument refuses.
type scrollSite struct {
	name string
	// fresh builds a brand new model standing at the site.
	//
	// ⚠️ A function rather than a stored model, and not for tidiness. m.picker is
	// a POINTER and (*pickState).read has a pointer receiver, so two presses off
	// one stored model would land on each other instead of on the same start —
	// the comparison would then be between one press and two. playScreen has the
	// same hazard for a different reason (it holds a *battle.Battle the model does
	// not copy), which is why the battle fixture builds its own battle too.
	fresh func(t *testing.T, lang i18n.Lang) model
	// state is where the frame sits, in the site's own fields.
	state func(model) string
}

// primeSteps is how far the frame is walked away from its boundary before the
// direction under test is measured.
//
// A fresh battle log follows the tail, so it is at the bottom and cannot go
// forward; a fresh description and a fresh reading pane are at the top and cannot
// go back. Rather than write down which site starts where, each direction is
// primed by pressing the page key pointing the OTHER way — which always makes
// room, and at a boundary is a no-op that would leave the comparison vacuous.
// That is what the moved check below is for.
const primeSteps = 2

func scrollSites() []scrollSite {
	return []scrollSite{{
		name: "the battle log",
		fresh: func(t *testing.T, lang i18n.Lang) model {
			t.Helper()
			return aLongLog(t, lang, 3)
		},
		state: func(m model) string {
			return fmt.Sprintf("offset %d follow %v", m.play.logOffset, m.play.logFollow)
		},
	}, {
		name: "the browse blurb",
		fresh: func(t *testing.T, lang i18n.Lang) model {
			t.Helper()
			base, _, _ := start(t, lang)
			raised := base.enter(screenBrowse)
			// The character with the most traits at the cap, because the frame is
			// only worth walking on a description that runs past the window.
			raised.browse.cursor = widestTraitRow(raised)
			raised.browse.level = progression.LevelCap
			raised.blurb.from = screenBrowse
			raised = raised.hand(raised.browse.subject())
			raised.screen = screenBlurb
			return raised
		},
		state: func(m model) string { return fmt.Sprintf("scroll %d", m.blurb.Scroll) },
	}, {
		name: "the picker's reading pane",
		fresh: func(t *testing.T, lang i18n.Lang) model {
			t.Helper()
			base, _, _ := start(t, lang)
			building := base.enter(screenSquads)
			building.squad = someSquad(t, building)
			member := building
			member.squad = member.squad.editUnit(0)
			return reading(member.openSquadSkills())
		},
		state: func(m model) string { return fmt.Sprintf("scroll %d", m.picker.scroll) },
	}}
}

// TestABracketScrollsWhereverAPageKeyDoes is the guard against the alias that
// ships dead.
//
// A key alias that never fires looks identical to one that does in every test
// that does not press it, so this presses both from the same start at every site
// and demands the same landing. The equality alone would be satisfied by two
// no-ops, which is the vacuous pass this shape invites — so the page key is
// required to have MOVED the frame first, and the failure names the site that
// went unmeasured rather than reporting a mismatch it never took.
func TestABracketScrollsWhereverAPageKeyDoes(t *testing.T) {
	directions := []struct{ name, page, bracket, away string }{
		{name: "back", page: "pgup", bracket: "[", away: "pgdown"},
		{name: "forward", page: "pgdown", bracket: "]", away: "pgup"},
	}
	for _, lang := range i18n.Langs() {
		for _, site := range scrollSites() {
			for _, direction := range directions {
				name := fmt.Sprintf("%s/%s/%s", lang, site.name, direction.name)
				// A subtest per row, because t.Skip and t.Fatalf inside a range are
				// runtime.Goexit and would abandon every site after this one.
				t.Run(name, func(t *testing.T) {
					primed := func() model {
						m := site.fresh(t, lang)
						for range primeSteps {
							m = key(t, m, direction.away)
						}
						return m
					}
					atRest := site.state(primed())
					paged := key(t, primed(), direction.page)
					if site.state(paged) == atRest {
						t.Fatalf("%s: %s did not move %s off %s, so this site is UNMEASURED — "+
							"the bracket and the page key would agree by both doing nothing",
							name, direction.page, site.name, atRest)
					}
					bracketed := key(t, primed(), direction.bracket)
					if got, want := site.state(bracketed), site.state(paged); got != want {
						t.Errorf("%s: %s landed at %s where %s landed at %s, both from %s",
							name, direction.bracket, got, direction.page, want, atRest)
					}
					// And the drawn screen with it, which is the half the fields
					// cannot claim: a frame position that agreed while the screen
					// did not would be two different readings of one number.
					if bracketed.screenContent() != paged.screenContent() {
						t.Errorf("%s: %s and %s draw different screens for %s",
							name, direction.bracket, direction.page, site.name)
					}
				})
			}
		}
	}
}
