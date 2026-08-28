package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// TestTheSkillFormReadsACriticalChanceBackOut is the half of the field that no
// other test can see while every shipped skill declares nought.
//
// The form both writes the file and is opened over it, so a field the form can
// write and cannot read back is worse than one it cannot write at all: every
// balance edit through this screen would silently zero a skill's critical chance,
// with the file still loading and the whole suite green. That is the exact
// mechanism `TestEveryShippedSkillTakesABalanceEdit` was widened for after
// `species` and then `flavour` were each lost to it.
//
// The empty case is the other half. An unauthored chance has to come back as an
// empty answer rather than as a "0", because accepting the form as it stands must
// write the file that was read — and a nought would add a `"crit": 0` key to
// every skill it touched.
func TestTheSkillFormReadsACriticalChanceBackOut(t *testing.T) {
	m, lib, _ := start(t, i18n.Vi)
	current, err := lib.Skills().Lookup("venom_fang")
	if err != nil {
		t.Fatalf("look up a shipped skill: %v", err)
	}
	if current.Crit != 0 {
		t.Fatalf("%s already crits, so the empty case below measures nothing", current.ID)
	}

	form := m.enter(screenSkills)
	form.skills = form.skills.prefill(form.lib, current)
	if got := form.skills.inputs[skillFieldCrit].Value(); got != "" {
		t.Errorf("a skill that cannot crit prefilled the field with %q, want an empty answer:"+
			" accepting the form as it stands would add a crit key to the file", got)
	}
	if drafted := form.skills.draft(form); drafted != forge.SkillAnswers(current) {
		t.Errorf("the form opened with\n%+v\nwant\n%+v", drafted, forge.SkillAnswers(current))
	}

	critting := current
	critting.Crit = 200
	form.skills = form.skills.prefill(form.lib, critting)
	if got := form.skills.inputs[skillFieldCrit].Value(); got != "200" {
		t.Errorf("a skill that crits prefilled the field with %q, want %q", got, "200")
	}
	if drafted := form.skills.draft(form); drafted.Crit != "200" {
		t.Errorf("the draft the form would write carries a critical chance of %q, want %q",
			drafted.Crit, "200")
	}
	// And the whole answer set round-trips, which is the assertion that survives
	// the next field somebody adds.
	if drafted := form.skills.draft(form); drafted != forge.SkillAnswers(critting) {
		t.Errorf("the form opened with\n%+v\nwant\n%+v", drafted, forge.SkillAnswers(critting))
	}
}

// TestTheCriticalFieldIsLabelledAndHelpedInBothLanguages is the rule that no
// user-visible wording lives in this package, checked at the one field this
// change adds. A field missing from either table draws an empty label, which is a
// form with a blank row rather than a failure.
func TestTheCriticalFieldIsLabelledAndHelpedInBothLanguages(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.Vi, i18n.En} {
		m, _, _ := start(t, lang)
		form := m.enter(screenSkills)
		if got, want := skillFieldLabel(form, skillFieldCrit), lang.Text(i18n.SkillFieldCrit); got != want {
			t.Errorf("%s labels the critical field %q, want %q", lang, got, want)
		}
		if got, want := skillFieldHelp(form, skillFieldCrit), lang.Text(i18n.SkillHelpCrit); got != want {
			t.Errorf("%s helps the critical field with %q, want %q", lang, got, want)
		}
	}
	// Beside pierce and before the allowlists, so the form reads in the order the
	// file is written in.
	if skillFieldCrit != skillFieldPierce+1 {
		t.Errorf("the critical field is at %d and pierce at %d, want them adjacent",
			skillFieldCrit, skillFieldPierce)
	}
	if skillFieldCrit >= skillFieldCount {
		t.Errorf("the critical field is at %d, past the %d fields the form counts",
			skillFieldCrit, skillFieldCount)
	}
}
