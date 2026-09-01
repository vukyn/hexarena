package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
)

// The behaviour half of the picker's destinations, and the half a totality walk
// cannot give.
//
// TestEveryPickDestinationLandsSomewhere in navigate_test.go says every
// destination is handled; these say each one is handled *right*. That is the
// #207 division and the #214 one — a dispatch entry that exists and writes the
// wrong field passes a count completely — and it has teeth here in a way it did
// not for the guard, because five of the ten destinations differ from each other
// in nothing but which list field they name.
//
// So each of these asserts two things: the field named by the destination holds
// the answer, and its siblings do not. The second half is the whole point.
// Measured before it was written: pointing the element allowlist at the role
// allowlist reddened nothing in this package.

// TestEachAllowlistPickLandsInItsOwnField drives all five of the skill form's
// allowlists through the keys an author presses.
//
// One table rather than five tests, because what is being held is a property of
// the set: five destinations, five fields, and a bijection between them. A test
// per field would assert five arrows and nothing about the map.
func TestEachAllowlistPickLandsInItsOwnField(t *testing.T) {
	lists := []struct {
		name  string
		field int
		read  func(skillsScreen) []string
	}{
		{"elements", skillFieldKeptForElements, func(s skillsScreen) []string { return s.keptElements }},
		{"roles", skillFieldKeptForRoles, func(s skillsScreen) []string { return s.keptRoles }},
		{"characters", skillFieldKeptForCharacters, func(s skillsScreen) []string { return s.keptWho }},
		{"species", skillFieldKeptForSpecies, func(s skillsScreen) []string { return s.keptKinds }},
		{"origins", skillFieldKeptForOrigins, func(s skillsScreen) []string { return s.keptWorlds }},
	}
	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			m, _, _ := start(t, i18n.Vi)
			m = m.enter(screenSkills)
			m = typeText(t, m, "a")
			// A fresh form has answered none of the five, which is what makes
			// "no sibling moved" a claim about this pick rather than about
			// whatever the form arrived holding.
			for _, other := range lists {
				if got := other.read(m.skills); len(got) != 0 {
					t.Fatalf("the fresh form already keeps %v for %s", got, other.name)
				}
			}
			m = skillFormTo(t, m, list.field)
			m = key(t, m, "space")
			if m.picker == nil {
				t.Fatalf("space on the %s allowlist opened no list", list.name)
			}
			rows := m.picker.visible()
			if len(rows) == 0 {
				t.Fatalf("the %s allowlist offers no rows", list.name)
			}
			want := rows[0].id
			m = pickTo(t, m, want)
			m = key(t, m, "space")
			m = key(t, m, "enter")
			if m.picker != nil {
				t.Fatal("enter did not close the list")
			}
			if got := list.read(m.skills); !slices.Equal(got, []string{want}) {
				t.Errorf("choosing %q on the %s allowlist left that field %v",
					want, list.name, got)
			}
			// And the answer went nowhere else. This is what a destination
			// pointed at the wrong field looks like, and nothing else in the
			// package can see it.
			for _, other := range lists {
				if other.field == list.field {
					continue
				}
				if got := other.read(m.skills); len(got) != 0 {
					t.Errorf("choosing on the %s allowlist also wrote %v into the %s one",
						list.name, got, other.name)
				}
			}
			if !m.skills.touched {
				t.Errorf("choosing on the %s allowlist left the form clean, so escaping it "+
					"would throw the answer away without asking", list.name)
			}
		})
	}
}

// TestTheStatusPickMarksTheFormAndNotOnlyTheField is the half of the inflicts
// destination that the three tests already covering it cannot see.
//
// ⚠️ **Measured, and it is a property of the field rather than of this step.**
// skillsScreen.inputs is a *slice*, so the landing writes the inflicts entry
// through backing storage the model already shares — which means dropping the
// client's `m.skills = skills` and keeping only the action leaves that field
// filled in and every existing status test green, while the flags beside it
// (touched, the cleared refusal, the cleared last-write note) are thrown away.
// It is the one destination of the ten whose answer survives an applier that
// applies nothing, so it is the one that needs a flag asserted rather than a
// value.
func TestTheStatusPickMarksTheFormAndNotOnlyTheField(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	m = skillFormTo(t, m, skillFieldInflicts)
	if m.skills.touched {
		t.Fatal("the fresh skill form is already dirty, so a pick that dirties it says nothing")
	}
	m = key(t, m, "space")
	if m.picker == nil {
		t.Fatal("space on the inflicts field opened no list")
	}
	want := m.picker.visible()[0].id
	m = pickTo(t, m, want)
	m = key(t, m, "space")
	m = key(t, m, "enter")
	if got := m.skills.inputs[skillFieldInflicts].Value(); !strings.Contains(got, want) {
		t.Errorf("choosing %q left the inflicts field %q", want, got)
	}
	if !m.skills.touched {
		t.Error("choosing a status left the form clean, so escaping it would throw the " +
			"entry away without asking")
	}
}

// TestTheCharacterFormsTwoPicksLandInTheirOwnFields is the same claim for the
// kit and the species, which are the other pair a destination has to tell apart.
//
// It also holds the one flag that is not shared between them: choosing a kit is
// setting it by hand, so it stops following the preset, and choosing a species
// must not. A flag written in the shared tail rather than in the kit's own arm
// would pass every other assertion here.
func TestTheCharacterFormsTwoPicksLandInTheirOwnFields(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	// The kit arrives filled from the preset and the species arrives empty, so
	// both halves of the trade below are stated against what the form really
	// starts from rather than against nothing.
	preset := append([]string(nil), m.form.kit...)
	if len(preset) == 0 {
		t.Fatal("the fresh form brings no kit, so a kit that did not move says nothing")
	}
	if len(m.form.species) != 0 {
		t.Fatalf("the fresh form already is %v", m.form.species)
	}
	if !m.form.kitFollowsPreset {
		t.Fatal("the fresh form's kit is not following the preset")
	}

	// The species first, because it is the one that must leave the kit alone.
	species := m
	species = formCursorTo(t, species, fieldSpecies)
	species = key(t, species, "space")
	if species.picker == nil {
		t.Fatal("space on the species row opened no list")
	}
	rows := species.picker.visible()
	if len(rows) == 0 {
		t.Fatal("the species list offers no rows")
	}
	wanted := rows[0].id
	species = pickTo(t, species, wanted)
	species = key(t, species, "space")
	species = key(t, species, "enter")
	if got := species.form.species; !slices.Equal(got, []string{wanted}) {
		t.Errorf("choosing %q left the species %v", wanted, got)
	}
	if got := species.form.kit; !slices.Equal(got, preset) {
		t.Errorf("choosing a species rewrote the kit as %v, want the preset's %v", got, preset)
	}
	if !species.form.kitFollowsPreset {
		t.Error("choosing a species took the kit off the preset, so the next preset " +
			"change would leave a kit nobody chose")
	}

	// And the kit, which must leave the species alone and does stop following.
	kit := m
	kit = formCursorTo(t, kit, fieldKit)
	kit = key(t, kit, "space")
	if kit.picker == nil {
		t.Fatal("space on the kit row opened no list")
	}
	kit = clearKit(t, kit)
	// The first row this character may actually take: the kit list offers every
	// skill in the book, marked with who it is for, and space declines a marked
	// one — so a test that took row nought would be measuring the refusal.
	chosen := ""
	for _, row := range kit.picker.visible() {
		if row.refusal == nil {
			chosen = row.id
			break
		}
	}
	if chosen == "" {
		t.Fatal("this character may carry nothing in the book, so no kit can be chosen")
	}
	kit = pickTo(t, kit, chosen)
	kit = key(t, kit, "space")
	kit = key(t, kit, "enter")
	if got := kit.form.kit; !slices.Equal(got, []string{chosen}) {
		t.Errorf("choosing %q left the kit %v", chosen, got)
	}
	if got := kit.form.species; len(got) != 0 {
		t.Errorf("choosing a kit wrote %v into the species", got)
	}
	if kit.form.kitFollowsPreset {
		t.Error("a kit chosen by hand is still following the preset, so changing the " +
			"preset would throw it away")
	}
}
