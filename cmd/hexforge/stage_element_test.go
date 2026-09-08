package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// TestShowNamesTheElementOfAFormThatDeclaresOne is the per-stage element on the
// one-shot print, and it is the art block's twin in every respect: a row per
// form that declares its own and nothing at all for the ordinary line, because a
// row on every character in a cast where no form differs says nothing on every
// listing it is on.
//
// The element row above it stays the character's, which is what the line STARTS
// as — the neighbouring `stages` row already carries the shape of the line, and
// the table `hexforge cast` prints has no level in it at all.
//
// ⚠️ **The form's element is set on a character taken from the library rather
// than written into the shipped data.** renderCharacter takes the character as a
// parameter, so the branch is reachable without a data edit; a fixture character
// would move goldens in three other packages for a print this command owns.
func TestShowNamesTheElementOfAFormThatDeclaresOne(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	var subject cast.Character
	for _, candidate := range lib.Characters().All() {
		if len(candidate.Stages) > 1 && !candidate.Element.Has(element.Metal) {
			subject = candidate
			break
		}
	}
	if subject.ID == "" {
		t.Fatal("no shipped character has a line that grows, so this measures nothing")
	}

	// Before: no form declares one, so the block prints nothing and the page is
	// exactly what it has always been. Asserting this half is what stops the
	// row becoming noise on every character in the cast.
	var plain bytes.Buffer
	renderCharacter(&plain, lib, subject, progression.LevelCap)
	if strings.Contains(plain.String(), " as "+subject.Stages[1].Name) {
		t.Errorf("a line whose forms declare no element of their own still gets a row per form:\n%s",
			plain.String())
	}

	own, err := element.Single(element.Metal)
	if err != nil {
		t.Fatalf("metal: %v", err)
	}
	grown := &subject.Stages[len(subject.Stages)-1]
	grown.Element = &own

	var page bytes.Buffer
	renderCharacter(&page, lib, subject, progression.LevelCap)
	drawn := page.String()
	if want := own.String() + " as " + grown.Name; !strings.Contains(drawn, want) {
		t.Errorf("the page does not say %q:\n%s", want, drawn)
	}
	if row := labelledRow(t, drawn, "element"); !strings.Contains(row, subject.Element.String()) {
		t.Errorf("the element row is %q, and the line starts as %s:\n%s",
			row, subject.Element, drawn)
	}
}

// labelledRow is the first row of a page whose label column holds the given
// name, so an assertion is about that row rather than about the whole page: an
// element name is a short word and a page carries a kit, a bio and a stage list.
func labelledRow(t *testing.T, page, label string) string {
	t.Helper()
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), label+" ") {
			return line
		}
	}
	t.Fatalf("no row is labelled %q in:\n%s", label, page)
	return ""
}
