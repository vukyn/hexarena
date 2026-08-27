package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// TestThePassiveListingNamesTheStatAReplyIsPricedOff.
//
// The listing said "attack" as a literal while every reply was priced off
// attack, and a mutation that put the literal back passed the whole suite — the
// column is not in any golden and nothing else reads it. That is the shape of
// failure worth a test of its own: an author tuning a thorns trait would read
// "8% attack" off a listing whose engine was multiplying defence, and the number
// they wrote would be a third out.
func TestThePassiveListingNamesTheStatAReplyIsPricedOff(t *testing.T) {
	lib, err := forge.Load(scratchData(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var drawn strings.Builder
	renderPassives(&drawn, lib)
	listing := drawn.String()

	answering := 0
	for _, held := range lib.Passives().All() {
		if held.Replies == nil || held.Replies.Power == 0 {
			continue
		}
		answering++
		stat := held.Replies.Scaling.Stat.String()
		if !rowFor(listing, held.ID, stat) {
			t.Errorf("%q answers off %s and its row does not say so:\n%s", held.ID, stat, listing)
		}
	}
	if answering == 0 {
		t.Skip("no shipped trait answers with damage, so this asserts nothing")
	}
}

// rowFor reports whether the trait's own row mentions the word.
func rowFor(listing, id, word string) bool {
	for _, line := range strings.Split(listing, "\n") {
		if strings.HasPrefix(line, id+" ") || line == id {
			return strings.Contains(line, word)
		}
	}
	return false
}
