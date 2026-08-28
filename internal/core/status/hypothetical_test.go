package status_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
)

// TestASetWithAnApplicationLeavesTheOriginalAlone is the test the whole rating
// layer rests on, and it is written first because the version anybody writes
// first is wrong.
//
// A Set holds its entries in a slice and each entry holds its stacks in another,
// so a value copy shares both arrays — and Apply writes through them, refreshing
// every stack already there. A copy made with `copied := *set`, or with a single
// `slices.Clone(entries)`, therefore reaches back into the original: rating a
// buff would silently hand the real unit a fresh duration on every stack it
// already held, from inside the one function in the engine that promises to
// mutate nothing.
//
// Nothing else would have said so. The refresh is invisible in a log — it looks
// exactly like sustained pressure, which is a thing that happens — and no golden
// distinguishes a status that lasted three turns because it was reapplied from
// one that lasted three turns because the opponent was thinking about it.
func TestASetWithAnApplicationLeavesTheOriginalAlone(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	// Spend a turn so the stack sits below its full duration: a refresh is only
	// visible against a duration that has already been partly used.
	set.Tick()
	before := set.Remaining("poison")
	if before != poison().Duration-1 {
		t.Fatalf("the stack has %d turns left before anything is rated, want %d",
			before, poison().Duration-1)
	}

	hypothetical := set.With(poison(), 100, 1)

	if got := set.Remaining("poison"); got != before {
		t.Errorf("rating an application moved the real stack from %d turns to %d",
			before, got)
	}
	if got := set.Stacks("poison"); got != 1 {
		t.Errorf("rating an application left the real unit with %d stacks, want 1", got)
	}
	if got := hypothetical.Stacks("poison"); got != 2 {
		t.Errorf("the hypothetical set holds %d stacks, want 2", got)
	}
	// The hypothetical is what Apply would produce, refresh included — the point
	// of layering it through Apply rather than describing what Apply does.
	if got := hypothetical.Remaining("poison"); got != poison().Duration {
		t.Errorf("the hypothetical stack has %d turns left, want the refreshed %d",
			got, poison().Duration)
	}
}

// TestASetWithNothingIsACopyRatherThanTheSameSet covers the call the pricing
// layer makes most often: it builds a hypothetical holding no new status at all,
// to read a unit as it stands without handing anything a pointer into the real
// set.
func TestASetWithNothingIsACopyRatherThanTheSameSet(t *testing.T) {
	var set status.Set
	set.Apply(haste(), 0)

	copied := set.With(status.Kind{}, 0, 0)
	copied.Apply(haste(), 0)

	if got := set.Stacks("haste"); got != 1 {
		t.Errorf("the original holds %d stacks after the copy was applied to, want 1", got)
	}
	if got := copied.Stacks("haste"); got != 2 {
		t.Errorf("the copy holds %d stacks, want 2", got)
	}
}

// TestASetWithAnApplicationObeysTheCap is why the copy layers through Apply
// instead of appending: a status already at its cap is worth nothing new, and
// that is the single term that keeps a rating from buffing for ever.
func TestASetWithAnApplicationObeysTheCap(t *testing.T) {
	var set status.Set
	for range haste().MaxStacks {
		set.Apply(haste(), 0)
	}
	capped := set.With(haste(), 0, 1)
	if got := capped.Stacks("haste"); got != haste().MaxStacks {
		t.Errorf("the hypothetical holds %d stacks past the cap of %d",
			got, haste().MaxStacks)
	}
}

// TestASetWithoutStacksTakesTheOnesACleanseWouldTake is the other direction, and
// the removal goes through Cleanse for the same reason the application goes
// through Apply: a price that chose its own stacks could name a figure the skill
// it is pricing never delivers.
func TestASetWithoutStacksTakesTheOnesACleanseWouldTake(t *testing.T) {
	var set status.Set
	set.Apply(poison(), 100)
	set.Apply(poison(), 100)
	set.Apply(haste(), 0)

	cleaned := set.Without([]status.Category{status.Dot}, 2)

	if got := cleaned.Stacks("poison"); got != 0 {
		t.Errorf("the hypothetical still holds %d poison stacks", got)
	}
	if got := cleaned.Stacks("haste"); got != 1 {
		t.Errorf("the hypothetical took a buff off as well: %d haste stacks left", got)
	}
	if got := set.Stacks("poison"); got != 2 {
		t.Errorf("pricing a cleanse removed %d real stacks", 2-got)
	}
}
