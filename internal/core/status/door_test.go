package status_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
)

// doorBook is the smallest book that has both sorts of status in it, because
// every question here is about which of the two a call will touch.
func doorBook(t *testing.T) *status.Book {
	t.Helper()
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "poison", "category": "dot", "max_stacks": 3, "duration": 3, "tick_power": 500},
	    {"id": "toughened", "category": "buff", "max_stacks": 2, "duration": 0, "permanent": true,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 200}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	return book
}

func doorKind(t *testing.T, book *status.Book, id string) status.Kind {
	t.Helper()
	kind, err := book.Lookup(id)
	if err != nil {
		t.Fatalf("lookup %s: %v", id, err)
	}
	return kind
}

// TestHoldAndReleaseAreTheDoorAPassiveOwns is what the pair is for.
//
// Remove refuses a permanent status so that nothing in the game can dispel a
// trait, which is right and which left a gated trait with no way to take its own
// grant back. Hold and Release are that way, and they are a separate pair rather
// than an exception inside Remove so that the rule Remove states stays true as
// written.
func TestHoldAndReleaseAreTheDoorAPassiveOwns(t *testing.T) {
	book := doorBook(t)
	toughened := doorKind(t, book, "toughened")
	var set status.Set

	if held := set.Hold(toughened, 0, 2); held != 2 {
		t.Fatalf("Hold put on %d stacks, want 2", held)
	}
	if got := set.Stacks("toughened"); got != 2 {
		t.Fatalf("the unit carries %d stacks, want 2", got)
	}
	// The rule that made the door necessary, still true.
	if removed, _ := set.Remove("toughened", 2); removed != 0 {
		t.Errorf("Remove took %d stacks off a permanent status", removed)
	}
	if released := set.Release("toughened"); released != 2 {
		t.Errorf("Release gave back %d stacks, want the 2 that were on", released)
	}
	if set.Has("toughened") {
		t.Error("a released status is still on the unit")
	}
	if released := set.Release("toughened"); released != 0 {
		t.Errorf("releasing what is not there gave back %d", released)
	}
}

// TestTheDoorRefusesTimedStatusesBothWays is what keeps it from becoming a
// second Apply and Remove.
//
// A trait may only grant a permanent status, so Hold has nothing to do with a
// timed one — and Release reaching a timed status would be a cleanse that
// ignored every resistance and reported none of the damage it stopped, which is
// exactly what Remove exists to do properly.
func TestTheDoorRefusesTimedStatusesBothWays(t *testing.T) {
	book := doorBook(t)
	poison := doorKind(t, book, "poison")
	var set status.Set

	if held := set.Hold(poison, 0, 2); held != 0 {
		t.Errorf("Hold put %d stacks of a timed status on", held)
	}
	if set.Has("poison") {
		t.Error("Hold put a timed status on the unit")
	}
	set.Apply(poison, 100)
	set.Apply(poison, 100)
	if released := set.Release("poison"); released != 0 {
		t.Errorf("Release took %d stacks off a timed status", released)
	}
	if got := set.Stacks("poison"); got != 2 {
		t.Errorf("the timed status has %d stacks left, want the 2 it had", got)
	}
}

// TestHoldStopsAtTheDeclaredCap is the one number a grant cannot argue with.
//
// A stack count comes from the data and the cap comes from the same book, so a
// trait asking for more than the status allows gets what the status allows —
// silently, because there is nobody to tell: the parse layer already refuses a
// grant over the cap, and this is the second line of that same rule rather than
// a new decision.
func TestHoldStopsAtTheDeclaredCap(t *testing.T) {
	book := doorBook(t)
	toughened := doorKind(t, book, "toughened")
	var set status.Set
	if held := set.Hold(toughened, 0, 5); held != 2 {
		t.Errorf("Hold put on %d stacks against a cap of 2", held)
	}
	if got := set.Stacks("toughened"); got != 2 {
		t.Errorf("the unit carries %d stacks, want the cap of 2", got)
	}
	if held := set.Hold(toughened, 0, 0); held != 0 {
		t.Errorf("holding nothing put on %d stacks", held)
	}
}

// TestAHeldStatusIsNotTimed is the property a stalemate is decided on.
//
// Timed asks whether anything on the unit will change on its own, and a trait is
// not something that will: a gated one changes when its holder's health moves,
// which is an action somebody has to take. A held grant counting as timed would
// make every battle with a trait in it un-drawable.
func TestAHeldStatusIsNotTimed(t *testing.T) {
	book := doorBook(t)
	var set status.Set
	set.Hold(doorKind(t, book, "toughened"), 0, 1)
	if set.Timed() {
		t.Error("a held grant counts as timed, so a board holding one can never be a draw")
	}
}
