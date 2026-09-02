package main

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// # A shared screen with authoring in it, drawn by a client that does not author
//
// Three of the seven catalogues this client offers are screens that **write a
// file** in cmd/hexforge-tui: the skill listing opens a form over `skills.json`
// with `a` and `e`, the works catalogue opens one over `origins.json` with `a`,
// and the squad catalogue's `n`, `enter` and `d` reach the two depths that write
// `squads.json` and the deletion that removes a side outright. A game client
// offers none of that.
//
// The answer is one capability on the value every screen already reads —
// `screen.Context.Authoring` — consulted beside the keystroke it turns off, so
// the keys and the footer are decided in one place and cannot disagree. What
// this file is, is the measurement, and it is four tests rather than one because
// the claim has four independent halves:
//
//   - **The keys do nothing** (TestNoAuthoringKeyDoesAnythingHere), driven
//     through the real model on the real screens.
//   - **No footer names one** (TestNoFooterHereNamesAKeyThisClientIgnores),
//     over every footer the client draws. This is the one the picker's own
//     comment is about: *"A key announced on a screen that ignores it is worse
//     than one nobody was told about."* A screen that swallows a keystroke
//     leaves a reader wondering whether they pressed it; a footer naming that
//     keystroke is the program promising something it does not do.
//   - **The list is the real one** (TestTheKeysThisClientIgnoresAreTheOnesListed),
//     which is the half neither of the two above can state: both of them are
//     written against `authoringKeys`, so a key missing from that list is a key
//     both of them skip. It is derived rather than trusted — every keystroke
//     this suite can send, pressed under both readings, and any key the
//     authoring reading answers and this one does not has to be on the list.
//   - **The two footers differ** (TestTheAuthoringFooterNamesWhatThisOneDrops),
//     because a read-only footer that happened to be byte-identical to the
//     authoring one would pass the second test by naming nothing at all.

// authoringKeys is every keystroke that reaches a change to the data files, per
// screen.
//
// ⚠️ **It is a design statement and not a derivation**, which is why it is
// written out: `a` adds, `e` edits, `n` starts a side, `enter` opens one for
// editing and `d` deletes one. What keeps it honest is
// TestTheKeysThisClientIgnoresAreTheOnesListed, which derives the same set from
// the two readings of every key this suite can send and holds the two equal in
// **both** directions — so a key added to a screen over in internal/screen
// fails here rather than quietly joining the ones nothing measures.
//
// ⚠️ **`enter` is on this list for one screen and is an ordinary key on three
// others**: it opens a menu entry, takes a turn in a battle and closes a filter.
// That is why the list is per screen and not a set — a union of it would flag
// the menu's own footer for naming the key that works.
var authoringKeys = map[screen][]string{
	screenSkills: {"a", "e"},
	screenWorks:  {"a"},
	screenSquads: {"n", "enter", "d"},
}

// TestNoAuthoringKeyDoesAnythingHere presses every one of them on the client
// that cannot author.
//
// ⚠️ **Driven through the real model rather than through a Context**, because
// what is being measured is the client: a screen handed a read-only Context is
// half the claim, and the other half is that this client hands one down. It
// compares the whole framed screen, so a mode that opened without drawing
// anything different would still be caught by the footer changing with it.
//
// Both languages, because a footer is wording and the two spellings are two
// entries in the catalog.
func TestNoAuthoringKeyDoesAnythingHere(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		for view, keys := range authoringKeys {
			at := base.enter(view)
			before := at.screenContent()
			for _, name := range keys {
				after := key(t, at, name)
				if after.screen != view {
					t.Errorf("%q on screen %v in %s moved to screen %v",
						name, view, lang, after.screen)
				}
				if got := after.screenContent(); got != before {
					t.Errorf("%q on screen %v in %s changed the screen:\nbefore\n%s\nafter\n%s",
						name, view, lang, before, got)
				}
			}
		}
	}
}

// TestNoFooterHereNamesAKeyThisClientIgnores is the half a swallowed keystroke
// cannot be seen from.
//
// ⚠️ **Over every footer the client draws**, taken off the sweep rather than off
// the three screens that author, because the failure is a *footer* rather than a
// screen: the day a fourth screen grows an authoring key, its footer is measured
// here on the same terms as the three that have one today.
//
// A footer's clauses are separated by the same middle dot in both languages and
// the key is each clause's first word, which is what makes the keys readable off
// a rendered line rather than off a second declaration of what a footer holds.
func TestNoFooterHereNamesAKeyThisClientIgnores(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		checked := 0
		for name, m := range everyScreen(t, base) {
			ignored := authoringKeys[m.screen]
			if len(ignored) == 0 {
				continue
			}
			checked++
			_, footer := m.parts()
			for _, named := range keysNamed(footer) {
				if slices.Contains(ignored, named) {
					t.Errorf("the %s screen's %s footer names %q, which this client ignores:\n%s",
						name, lang, named, footer)
				}
			}
		}
		if checked == 0 {
			t.Errorf("no footer in the %s sweep belongs to a screen that can author, so this "+
				"measures nothing", lang)
		}
	}
}

// TestTheAuthoringFooterNamesWhatThisOneDrops is what stops the test above
// passing on a footer that names no keys at all.
//
// ⚠️ **A read-only footer identical to the authoring one would satisfy every
// other test in this file** — nothing above asserts the two differ — so the
// authoring spelling is asked for the same three screens and has to name every
// key this client ignores. That is also the check that the clause-splitting
// above finds keys where they really are: a reader that came back empty would
// pass the previous test on every footer in the catalog.
func TestTheAuthoringFooterNamesWhatThisOneDrops(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		reading := base.ctx()
		authoring := reading
		authoring.Authoring = true
		for view, keys := range authoringKeys {
			probe, known := probes()[view]
			if !known {
				t.Fatalf("screen %v has authoring keys and no probe, so nothing measures its "+
					"two footers", view)
			}
			_, named := probe.at(authoring)
			_, silent := probe.at(reading)
			if named == silent {
				t.Errorf("screen %v draws the same %s footer either way, so the read-only one "+
					"names nothing it should have dropped:\n%s", view, lang, named)
			}
			advertised := keysNamed(named)
			for _, want := range keys {
				if !slices.Contains(advertised, want) {
					t.Errorf("the authoring %s footer for screen %v does not name %q, so the "+
						"clause reader is not finding keys:\n%s", lang, view, want, named)
				}
			}
		}
	}
}

// TestTheKeysThisClientIgnoresAreTheOnesListed derives the list rather than
// trusting it.
//
// ⚠️ **It is the only test here that can fail on a key nobody thought of.** The
// other three are written against `authoringKeys`, so a key added to one of
// these screens and left off the list is a key all three of them skip — the
// exact shape of the five screens TODO.md records slipping out of a sweep. What
// this walks instead is every keystroke this suite can send, pressed under both
// readings of the same screen, and the answer it collects is *what the authoring
// reading answers and this one does not*.
//
// Both directions are held. A key the derivation finds and the list lacks is a
// keystroke this client silently ignores with nothing measuring it; a key the
// list holds and the derivation does not find is either a key that no longer
// authors or one the guard has stopped covering, and either way the list has
// stopped describing the code.
//
// ⚠️ **The fingerprint carries the Action as well as the drawing**, because one
// of the five changes neither: `d` on the squad catalogue asks a question, which
// is an Action the screen hands back and a body it does not touch.
func TestTheKeysThisClientIgnoresAreTheOnesListed(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	reading := base.ctx()
	authoring := reading
	authoring.Authoring = true
	for view, probe := range probes() {
		restAuthoring := fingerprintOf(probe.at(authoring))
		restReading := fingerprintOf(probe.at(reading))
		var found []string
		for _, name := range everyKeyPressed() {
			pressed := press(t, name)
			answered := probe.after(authoring, pressed) != restAuthoring
			ignored := probe.after(reading, pressed) == restReading
			if answered && ignored {
				found = append(found, name)
			}
		}
		want := append([]string(nil), authoringKeys[view]...)
		sortStrings(want)
		sortStrings(found)
		if !slices.Equal(found, want) {
			t.Errorf("screen %v answers %v under an authoring client and ignores them here; "+
				"authoringKeys says %v", view, found, want)
		}
	}
}

// TestNoScreenInThisClientAsksOrPicks is what navigate's Ask-and-Pick arm rests
// on.
//
// Both kinds are the authoring half of the vocabulary — a question is what a
// form puts to a reader before throwing a draft away, and a picker is the list a
// form fills a field from — and every screen that raises one does it from a mode
// this client cannot enter. That is a claim about behaviour rather than about
// types, so it is pressed: every keystroke this suite can send, on every screen
// that has ever raised either, under this client's own reading.
//
// ⚠️ **If it ever fails, the arm in navigate is the thing to change and not this
// test.** A client that swallowed a real Ask would take the question down before
// it was drawn, which is the quietest of the failures TODO.md keeps a list of.
func TestNoScreenInThisClientAsksOrPicks(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	reading := base.ctx()
	for view, probe := range probes() {
		for _, name := range everyKeyPressed() {
			if kind := probe.kind(reading, press(t, name)); kind == draw.Ask || kind == draw.Pick {
				t.Errorf("%q on screen %v asks for a %v, which this client draws nowhere",
					name, view, kind)
			}
		}
	}
}

// probe is one screen that can author, built fresh on demand: `at` is the screen
// as a reader finds it, `after` is the same screen one keystroke later, and
// `kind` is what that keystroke asked the client for.
//
// ⚠️ **Fresh every call, which is what makes a sweep over every key sound.** A
// probe that kept the screen between keystrokes would be measuring a walk
// through a hundred of them rather than a hundred single presses, and the first
// key that opened a mode would decide what every later one did.
type probe struct {
	at    func(draw.Context) (string, string)
	after func(draw.Context, tea.KeyPressMsg) string
	kind  func(draw.Context, tea.KeyPressMsg) draw.Kind
}

// probes is one per screen that can author.
//
// They call the screens' own constructors rather than going through the model,
// deliberately: what varies between the two readings is the Context, so a probe
// that took a model would have to build two models to ask one question.
func probes() map[screen]probe {
	return map[screen]probe{
		screenSkills: {
			at: func(c draw.Context) (string, string) {
				return draw.NewSkillsScreen(c).View(c)
			},
			after: func(c draw.Context, k tea.KeyPressMsg) string {
				next, action, _ := draw.NewSkillsScreen(c).Update(c, k)
				return fingerprint(next.View(c))(action)
			},
			kind: func(c draw.Context, k tea.KeyPressMsg) draw.Kind {
				_, action, _ := draw.NewSkillsScreen(c).Update(c, k)
				return action.Kind
			},
		},
		screenWorks: {
			at: func(c draw.Context) (string, string) {
				return draw.NewOriginsScreen(c).View(c)
			},
			after: func(c draw.Context, k tea.KeyPressMsg) string {
				next, action, _ := draw.NewOriginsScreen(c).Update(c, k)
				return fingerprint(next.View(c))(action)
			},
			kind: func(c draw.Context, k tea.KeyPressMsg) draw.Kind {
				_, action, _ := draw.NewOriginsScreen(c).Update(c, k)
				return action.Kind
			},
		},
		screenSquads: {
			at: func(c draw.Context) (string, string) {
				return draw.NewSquadsScreen(c).View(c)
			},
			after: func(c draw.Context, k tea.KeyPressMsg) string {
				next, action, _ := draw.NewSquadsScreen(c).Update(c, k)
				return fingerprint(next.View(c))(action)
			},
			kind: func(c draw.Context, k tea.KeyPressMsg) draw.Kind {
				_, action, _ := draw.NewSquadsScreen(c).Update(c, k)
				return action.Kind
			},
		},
	}
}

// fingerprint is a drawing and what the screen asked for, as one comparable
// string. fingerprintOf is the same reading for a screen nobody pressed a key
// on, which is a draw.Stay by definition.
func fingerprint(body, footer string) func(draw.Action) string {
	return func(action draw.Action) string {
		return action.Kind.String() + "\x00" + body + "\x00" + footer
	}
}

func fingerprintOf(body, footer string) string {
	return fingerprint(body, footer)(draw.Action{})
}

// keysNamed is the keystrokes a footer advertises.
//
// A footer is clauses separated by a middle dot, and the key is each clause's
// first word — the shape every footer in the catalog is written in, in both
// languages. Read off the rendered line rather than off a second declaration of
// what a footer holds, which is the only way this can disagree with the wording
// an author actually typed.
func keysNamed(footer string) []string {
	var out []string
	for _, clause := range strings.Split(footer, "·") {
		if fields := strings.Fields(clause); len(fields) > 0 {
			out = append(out, fields[0])
		}
	}
	return out
}
