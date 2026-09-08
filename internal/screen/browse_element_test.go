package screen

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheElementRowFollowsTheFormItResolvedTo is the per-stage element as a
// reader meets it, and it is the art row's twin: both are facts about the FORM
// rather than about the character, both sit under the level, and walking the
// arrow keys is what shows either of them change.
//
// ⚠️ **The form's own element is set here rather than in the fixture cast**, for
// the reason the seed test's line is built in its own file: the bench is
// injected into every scratch data directory five packages build, so a character
// gaining a field there moves three screen goldens. Detail takes the character as
// a parameter precisely so a test can hand it one, which is what makes the branch
// reachable without any of that.
//
// ⚠️ It is the DETAIL pane and not the list row above it. The list keeps the
// character's element deliberately: it has no level, so it has no form to ask
// about, and its glossed affinity column has seven cells of slack against a
// worst case already measured at 72 of 79 — `ground` growing into `ground/metal`
// does not fit. This pane has a level, so this is where the question can be
// asked at all.
func TestTheElementRowFollowsTheFormItResolvedTo(t *testing.T) {
	c, lib := start(t, i18n.Vi)
	b := NewBrowseScreen(lib)

	var subject cast.Character
	for _, candidate := range lib.Characters().All() {
		grown := candidate.Stages[len(candidate.Stages)-1]
		if len(candidate.Stages) > 1 && grown.MinLevel > 1 {
			subject = candidate
			break
		}
	}
	if subject.ID == "" {
		t.Fatal("no character on the bench has a line that grows, so this tests nothing")
	}
	grown := &subject.Stages[len(subject.Stages)-1]

	// An affinity the character has none of, so that neither answer can be
	// mistaken for the other.
	own, err := element.Single(element.Metal)
	if err != nil {
		t.Fatalf("metal: %v", err)
	}
	if subject.Element.Has(element.Metal) {
		t.Fatalf("%s is already %s, so a metal form would not be a change",
			subject.ID, subject.Element)
	}
	grown.Element = &own

	label := c.Text(i18n.LabelElement)
	for _, test := range []struct {
		level int
		want  string
	}{
		{grown.MinLevel - 1, subject.Element.String()},
		{grown.MinLevel, own.String()},
		{progression.LevelCap, own.String()},
	} {
		b.Level = test.level
		row := labelledRow(t, b.Detail(c, subject), label)
		if !strings.Contains(row, test.want) {
			t.Errorf("at level %d the element row is %q, want it to name %s",
				test.level, row, test.want)
		}
	}
}

// labelledRow is the first row of a pane whose label column holds the given
// name, so an assertion is about that row rather than about the whole page —
// an element name is a short word and would match a gloss somewhere else on it.
func labelledRow(t *testing.T, pane, label string) string {
	t.Helper()
	for _, line := range strings.Split(pane, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), label+" ") {
			return line
		}
	}
	t.Fatalf("no row is labelled %q in:\n%s", label, pane)
	return ""
}
