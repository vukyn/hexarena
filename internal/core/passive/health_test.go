package passive_test

import (
	"strings"
	"testing"
)

// TestAGatedTraitMayNotGrantHealth is the third leg of the health rule, and it is
// the only one this book can see.
//
// status.ParseBook lets a health term sit on a permanent status because a
// permanent status is the one thing nothing takes back: Remove refuses one, so a
// dispel cannot turn a trait off. A gate is the exception it cannot see from
// there — Battle.reconsider calls Release when the gate closes — so a gated trait
// carrying a health term would raise its holder's maximum when the gate opened
// and drop it again when it shut, leaving a holder that had been healed into the
// new room standing above its own maximum.
func TestAGatedTraitMayNotGrantHealth(t *testing.T) {
	_, err := parse(t,
		`[{"id":"overgrow","while":{"below_health":333},"grants":[{"status":"swollen","stacks":1}]}]`)
	if err == nil {
		t.Fatal("a gated trait granting a health status was accepted")
	}
	if !strings.Contains(err.Error(), "raises health") {
		t.Errorf("it was refused with %q, want it to mention the health it raises", err)
	}
}

// TestAnUngatedTraitMayGrantHealth is the other side, and it is what keeps the
// refusal above from being a ban on the term.
//
// An ungated trait raises the maximum once, at enlistment, and never moves it
// again — the same shape a composition bonus has. Refusing both would have left
// the whole health term unreachable and the design unbuilt.
func TestAnUngatedTraitMayGrantHealth(t *testing.T) {
	book, err := parse(t, `[{"id":"broad","grants":[{"status":"swollen","stacks":2}]}]`)
	if err != nil {
		t.Fatalf("an ungated trait granting a health status was refused: %v", err)
	}
	held, err := book.Lookup("broad")
	if err != nil {
		t.Fatalf("look up the trait: %v", err)
	}
	if len(held.Grants) != 1 || held.Grants[0].Status != "swollen" || held.Grants[0].Stacks != 2 {
		t.Errorf("the trait came back holding %+v, want two stacks of swollen", held.Grants)
	}
}
