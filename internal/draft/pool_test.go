package draft_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// shippedCast is the authored cast, which is what a real pool is built out of.
func shippedCast(t *testing.T) []cast.Character {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("parse the embedded cast: %v", err)
	}
	all := book.All()
	if len(all) == 0 {
		t.Fatal("the embedded cast is empty, so every measurement below is of nothing")
	}
	return all
}

// TestThePoolIsTheCastMinusTheHeldBack measures the rule against the shipped
// cast rather than restating it, which is the only way this stays true as
// characters ship.
//
// ⚠️ **The figures are derived and logged, not asserted as literals.** The pool
// moves every time a character is added or hidden — it has been twelve, then
// fourteen, then fifteen, then sixteen inside the life of TODO.md's own
// arithmetic block — so an assertion of "sixteen" would be a line that reddens on
// every content change and teaches its reader to edit the number. What is
// asserted is the *relation*: the pool is exactly the not-hidden characters, and
// nothing else. TestFiveASideFitsTheShippedCastExactly is where a moved figure is
// allowed to redden, because there the number changes a design decision.
func TestThePoolIsTheCastMinusTheHeldBack(t *testing.T) {
	all := shippedCast(t)
	pool := draft.NewPool(all)

	held := 0
	for _, character := range all {
		if character.Hidden {
			held++
		}
	}
	if want := len(all) - held; pool.Len() != want {
		t.Errorf("the pool seats %d of %d characters and %d are held back, so it should seat %d",
			pool.Len(), len(all), held, want)
	}
	// The flag has to actually be in play, or this test would agree with a
	// NewPool that filtered nothing at all.
	if held == 0 {
		t.Fatal("no shipped character carries hidden: true, so the filter this test is " +
			"about is unexercised — give the fixture a held-back character or drop the test")
	}
	for _, character := range all {
		seated := pool.Has(character.ID)
		switch {
		case character.Hidden && seated:
			t.Errorf("%s is held back and a draft can still ban or pick it", character.ID)
		case !character.Hidden && !seated:
			t.Errorf("%s is offered and a draft cannot seat it", character.ID)
		}
	}
	if pool.Has("nobody.at.all") {
		t.Error("the pool claims to seat a character that does not exist")
	}
	t.Logf("the shipped cast is %d characters, %d held back, so the pool is %d",
		len(all), held, pool.Len())
}

// TestFiveASideFitsTheShippedCastExactly is the content prerequisite held
// mechanically, and it is the one place a moved cast figure is *meant* to
// redden.
//
// ⚠️ **This test used to assert the opposite** — it was
// TestFiveASideDoesNotFitTheShippedCast, and its own instruction for the day it
// reddened was to re-run the measurement, write the new figures into TODO.md
// § "Ban and pick", and decide whether the hold-back comes off rather than relax
// the test. That day was 2026-09-05: `pokemon.gible` took the pool to sixteen,
// which is exactly `2*picks + 2*bans` at 5v5, so `draft.Fits` stopped refusing.
// The decision taken there: the **draft** no longer holds 5v5 back, and the
// balance read still does, so `hexarena-host` keeps refusing `-format 5` for its
// other reason.
//
// ⚠️ **Exactly is not comfortably, and the test says which.** Slack is nought, so
// with every ban spent the final picker of a 5v5 sees exactly one candidate — a
// pick that is not a decision. That is why the assertion below pins the slack
// rather than only the fit: a pool that grew would move this figure and is worth
// noticing, and a pool that shrank would take 5v5 away again.
func TestFiveASideFitsTheShippedCastExactly(t *testing.T) {
	pool := draft.NewPool(shippedCast(t))

	if err := draft.Fits(pool.Len(), wire.Format3v3); err != nil {
		t.Errorf("3v3 has fitted the shipped cast since it was measured, and now does not: %v", err)
	}
	if err := draft.Fits(pool.Len(), wire.Format5v5); err != nil {
		t.Errorf("a pool of %d no longer seats a 5v5 draft: %v. That is a decision to take in "+
			"TODO.md § \"Ban and pick\" and at hexarena-host's format flag, not a test to relax",
			pool.Len(), err)
	}
	// Named apart from the fit above, which would also catch a pool that grew but
	// would report it as nothing at all: growing is the case where 5v5 stops being
	// a draft whose last pick is forced, and that is a decision, not an accident.
	if slack := draft.Slack(pool.Len(), wire.Format5v5); slack != 0 {
		t.Errorf("a pool of %d seats a 5v5 with %d to spare, and it seated it with nought to "+
			"spare when that was measured. If the cast grew, say so in TODO.md § \"Ban and pick\" — "+
			"the last pick of a 5v5 is a real decision from one to spare onward",
			pool.Len(), slack)
	}
	t.Logf("pool %d: 3v3 fits with %d to spare, 5v5 with %d",
		pool.Len(), draft.Slack(pool.Len(), wire.Format3v3), draft.Slack(pool.Len(), wire.Format5v5))
}

// TestThePoolKeepsTheBooksDeclarationOrder holds the order the draft screen will
// draw, and it is written against a book of its own on purpose.
//
// ⚠️ **The shipped cast is in id order today, so this test taken against it
// would measure nothing** — `naruto.naruto` then every `pokemon.*` alphabetically
// is exactly what a sort would produce, and a NewPool that sorted would pass. So
// the fixture's declaration order is deliberately not its id order, and the
// first assertion below is that this is still so: a fixture that drifted into
// sorted order would quietly take the test with it.
//
// *Sees:* a sort added to NewPool, an accessor that orders on the way out, the
// filter reversing the survivors.
// *Cannot see:* what the screen does with the list afterwards, which is step 5.
func TestThePoolKeepsTheBooksDeclarationOrder(t *testing.T) {
	declared := unsortedFixture()

	ids := make([]string, 0, len(declared))
	for _, character := range declared {
		ids = append(ids, character.ID)
	}
	if slices.IsSorted(ids) {
		t.Fatal("the fixture's declaration order is also its id order, so this test cannot " +
			"tell declaration order from a sort: it measures nothing")
	}

	want := []string{}
	for _, character := range declared {
		if !character.Hidden {
			want = append(want, character.ID)
		}
	}
	got := []string{}
	for _, character := range draft.NewPool(declared).All() {
		got = append(got, character.ID)
	}
	if !slices.Equal(got, want) {
		t.Errorf("the pool is %v and the book declares %v. The order is the book's own and "+
			"is not sorted by id — see NewPool's comment for why, because sorting is the "+
			"obvious thing to try here", got, want)
	}
	t.Logf("%d declared, %d seated, in declaration order %v", len(declared), len(got), got)
}

// TestAllHandsBackACopy is cast.Book.All()'s promise kept one layer up: the pool
// is fixed for the whole of a draft, and a caller that sorted or truncated what
// it was handed would otherwise be reordering the pool itself.
func TestAllHandsBackACopy(t *testing.T) {
	pool := draft.NewPool(unsortedFixture())
	before := pool.All()
	if len(before) < 2 {
		t.Fatalf("the fixture seats %d characters, and a reordering test needs at least two",
			len(before))
	}

	handed := pool.All()
	slices.Reverse(handed)
	handed[0] = cast.Character{ID: "written.over"}

	after := pool.All()
	for at := range before {
		if after[at].ID != before[at].ID {
			t.Fatalf("changing the slice All() handed back changed the pool: row %d was %q "+
				"and is now %q", at, before[at].ID, after[at].ID)
		}
	}
	if pool.Has("written.over") {
		t.Error("a character written into the slice All() handed back is now in the pool")
	}
}

// TestTheZeroPoolSeatsNobody is the empty case answered rather than left to be
// discovered: a Pool nobody built is a draft with nothing in it, not a panic.
func TestTheZeroPoolSeatsNobody(t *testing.T) {
	var pool draft.Pool
	if pool.Len() != 0 {
		t.Errorf("the zero pool seats %d characters", pool.Len())
	}
	if pool.Has("pokemon.mew") {
		t.Error("the zero pool claims to seat somebody")
	}
	if all := pool.All(); all == nil || len(all) != 0 {
		t.Errorf("the zero pool's All() is %v, and an empty list is not the same answer as no list", all)
	}
	if got := draft.NewPool(nil).Len(); got != 0 {
		t.Errorf("a pool of no characters seats %d", got)
	}
}

// unsortedFixture is a cast whose declaration order is deliberately not its id
// order, with one character held back in the middle of it so the filter and the
// order are exercised by the same list.
//
// Only ID and Hidden are set, which is the whole of what NewPool reads. A fuller
// fixture would be a second declaration of a character, and internal/testfixture
// is where one belongs if a later step needs one.
func unsortedFixture() []cast.Character {
	return []cast.Character{
		{ID: "zzz.last"},
		{ID: "aaa.first"},
		{ID: "mmm.middle", Hidden: true},
		{ID: "bbb.second"},
		{ID: "nnn.after"},
	}
}
