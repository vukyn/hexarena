package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// A status may now be named on its own declaration, the way a skill and a trait
// already were. The field is worth nothing on its own: a name lands in a sentence
// only if the status book reaches the describer, and the describer had no book —
// it named every status out of the compiled table in gloss.go, off the id alone.
//
// So the tests here are about the THREADING and not about the field. The field is
// four lines and cannot fail interestingly; the parameter runs through eleven
// functions and three packages, and a call site left on the old path shows up as
// a bare id in one sentence out of forty, which is exactly the kind of miss that
// ships.

// namedStatusBook is a status book whose one kind carries an authored name and
// has no entry in statusGloss, so anything naming it correctly can only have got
// the name out of the data.
func namedStatusBook(t *testing.T) *status.Book {
	t.Helper()
	kinds, err := status.ParseBook([]byte(`{"max_stacks":3,"max_duration":6,"kinds":[
	  {"id":"scorching","name":"cháy sém","category":"dot","max_stacks":3,
	   "duration":3,"tick_power":200}
	]}`))
	if err != nil {
		t.Fatalf("parse the named status book: %v", err)
	}
	return kinds
}

// applier is a skill that puts the named status on an enemy, parsed against the
// book above so the id it names is the one that carries the name.
func applier(t *testing.T, kinds *status.Book, shapes *pattern.Book) skill.Skill {
	t.Helper()
	book, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"sear","element":"fire","range":2,"pattern":"single",
	   "power":80,"strikes":1,"accuracy":1000,"cooldown":2,"target":"enemy",
	   "applies":[{"status":"scorching","stacks":2,"chance":1000}]}
	]}`), skill.Deps{Patterns: shapes, Statuses: kinds})
	if err != nil {
		t.Fatalf("parse the applying skill: %v", err)
	}
	declared, err := book.Lookup("sear")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	return declared
}

// TestAnAuthoredStatusNameReachesTheSentences is the threading, end to end.
//
// It asserts the name is there AND that the bare id is not, because those are two
// different failures: a describer that never asked the book prints the id, and one
// that asked and got nothing prints the id as well. Only the second is a bug in
// the lookup, and neither is visible from the name alone.
func TestAnAuthoredStatusNameReachesTheSentences(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	kinds := namedStatusBook(t)
	declared := applier(t, kinds, shapes)

	if i18n.Vi.Gloss("scorching") != "" {
		t.Fatal("the compiled table names `scorching`, so this test could pass without the book")
	}
	described := i18n.Vi.Describe(declared, shapes, kinds)
	if !strings.Contains(described, "cháy sém") {
		t.Errorf("the description does not carry the authored status name:\n%s", described)
	}
	if strings.Contains(described, "scorching") {
		t.Errorf("the description still shows the bare id beside the name:\n%s", described)
	}

	summary := i18n.Vi.SummariseSkill(declared, shapes, kinds)
	if !strings.Contains(summary, "cháy sém") {
		t.Errorf("the one-line summary does not carry the authored status name: %q", summary)
	}
}

// TestANilStatusBookFallsBackToTheTable holds the other half of the contract.
//
// The parameter reaches callers that have no library — a replay rendered without
// books, a test describing a hand-built skill — and the answer for those has to be
// the answer they had before the field existed. If nil went to a bare id instead,
// adding the parameter would have been a regression on data nobody touched.
func TestANilStatusBookFallsBackToTheTable(t *testing.T) {
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load the shipped patterns: %v", err)
	}
	kinds, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	var checked int
	for _, declared := range skills.Skills() {
		if len(i18n.StatusesInSkill(declared)) == 0 {
			continue
		}
		checked++
		withBook := i18n.Vi.Describe(declared, shapes, kinds)
		withNone := i18n.Vi.Describe(declared, shapes, nil)
		if withBook != withNone {
			t.Errorf("%s reads differently with and without the status book, and no shipped "+
				"status carries an authored name:\nwith:\n%s\nwithout:\n%s",
				declared.ID, withBook, withNone)
		}
	}
	if checked == 0 {
		t.Fatal("no shipped skill touches a status, so this compares nothing")
	}
}
