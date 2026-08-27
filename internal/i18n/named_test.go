package i18n_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestATraitNamesEveryStatusItsDescriptionNames is the anti-drift test the whole
// function exists behind.
//
// StatusesNamed is a second reading of DescribePassive's own rules, kept apart
// from it so a screen can act on the names without parsing prose. Two readings
// of one thing drift, and the failure is silent in both directions: a name the
// list has and the sentences do not gets marked where nothing is printed, and a
// name the sentences have and the list does not is the one word on the screen
// that ? cannot open.
func TestATraitNamesEveryStatusItsDescriptionNames(t *testing.T) {
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped traits: %v", err)
	}
	checked := 0
	for _, held := range passives.All() {
		named := i18n.StatusesNamed(held)
		if len(named) == 0 {
			continue
		}
		checked++
		described := i18n.Vi.DescribePassive(held)
		for _, id := range named {
			name := i18n.Vi.Gloss(id)
			if name == "" {
				name = id
			}
			if !strings.Contains(described, name) {
				t.Errorf("%s names %q and its description never says %q:\n%s",
					held.ID, id, name, described)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no shipped trait names a status, so nothing here was measured")
	}
}

// TestNoTraitNamesTheSameStatusTwice is what makes the list usable as a list.
//
// venom_blood is immune to poison and answers with poison, so it holds the id
// twice and names it once. A caller marking names would style the word twice
// and a caller offering to look one up would offer the same status two ways.
func TestNoTraitNamesTheSameStatusTwice(t *testing.T) {
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped traits: %v", err)
	}
	repeated := false
	for _, held := range passives.All() {
		seen := map[string]bool{}
		for _, id := range i18n.StatusesNamed(held) {
			if seen[id] {
				t.Errorf("%s names %q twice", held.ID, id)
			}
			seen[id] = true
		}
		// The trait that would repeat if nothing stopped it, so that this test
		// is measuring something on the shipped book rather than agreeing with
		// an empty one.
		if held.ID == "venom_blood" {
			repeated = len(held.Resists) > 0 && len(held.Replies.Applies) > 0
		}
	}
	if !repeated {
		t.Fatal("venom_blood no longer both resists and answers with a status, " +
			"so nothing in the shipped book can repeat and this test is vacuous")
	}
}

// TestAReplyNamesItsFirstStatusOnly is the rule that cannot be read off the
// shipped data, because nothing in it answers with two statuses.
//
// DescribePassive gives a reply one sentence and that sentence has room for one
// status, so a trait answering with two holds two and names one. A list that
// reported both would offer to look up a status whose name is nowhere on screen.
func TestAReplyNamesItsFirstStatusOnly(t *testing.T) {
	book, err := passive.ParseBook([]byte(`{"passives":[
	  {"id":"twofold","replies":{"applies":[
	    {"status":"poison","chance":1000},
	    {"status":"burn","chance":1000}
	  ]}}
	]}`), passive.Deps{Statuses: mustStatuses(t)})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	twofold, err := book.Lookup("twofold")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(twofold.Replies.Applies) != 2 {
		t.Fatalf("the fixture answers with %d statuses, want two",
			len(twofold.Replies.Applies))
	}
	named := i18n.StatusesNamed(twofold)
	if len(named) != 1 || named[0] != "poison" {
		t.Errorf("a reply with two applications named %v, want only the first", named)
	}
	if strings.Contains(i18n.Vi.DescribePassive(twofold), i18n.Vi.Gloss("burn")) {
		t.Error("the description names the second application after all, " +
			"so the list is now the one that is wrong")
	}
}

func mustStatuses(t *testing.T) *status.Book {
	t.Helper()
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	return statuses
}
