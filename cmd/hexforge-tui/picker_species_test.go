package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The species picker as *this client* reaches it, which is the half that could
// not move with the screen.
//
// ⚠️ What the cell draws — the authored name in Vietnamese, the bare id in
// English, and the whole column dropped when no row has one — is the picker's
// own claim and is asserted in internal/screen/picker_test.go against a picker
// built there. What is left here is the one thing that package cannot see: that
// a keystroke on the character form reaches this screen at all, and that the
// sweeps every other screen is measured by walk it.

// The species picker is one of the screens the width and translation sweeps
// walk. It was not, which is why the leak above lived: those sweeps iterate
// everyScreen, and everyScreen registered no species picker at all.
//
// ⚠️ Membership by name would prove nothing — an entry called "species picker"
// holding some other model would satisfy it while the sweeps still never
// rendered this screen. So the entry is found by what its model *is* (a picker
// over the species book) and then held to what it *draws*: byte for byte the
// screen an independently opened species picker renders.
func TestTheScreenSweepWalksTheSpeciesPicker(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := start(t, lang)
		want := m.enter(screenNew).openSpecies()
		want.width, want.height = 200, 60

		found := ""
		for name, screen := range everyScreen(t, m) {
			if screen.picker == nil || screen.picker.Kind != pickSpecies {
				continue
			}
			screen.width, screen.height = want.width, want.height
			if screen.screenContent() != want.screenContent() {
				t.Errorf("the %s entry in %s holds a species picker that draws a different screen:\n%s",
					name, lang, screen.screenContent())
				continue
			}
			found = name
		}
		if found == "" {
			t.Errorf("no entry of everyScreen renders the species picker in %s, "+
				"so no width or translation test measures it", lang)
		}
	}
}
