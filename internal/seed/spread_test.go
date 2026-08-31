package seed_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestASpreadConsumerIsBoundedByTheShapeItBuys is the other half of the detonate
// rule, for the other thing a consume can be paid in.
//
// A detonate trades a status's remaining ticks for power, and
// TestADetonateIsWorthLessThanItsBreakEven bounds it at twice what leaving the
// status alone was worth. A spread trades a stack for *cells*, and the currency
// bounds itself: a splash lands at the pattern book's own splash share, and
// max_targets caps any shape at three cells, so the widest spread that can be
// authored is worth a plain attack plus two half ones — twice a plain attack, the
// same ceiling read from the other side.
//
// So what is held here is not a number this test invents. It is that the two
// numbers the pattern book already declares are the only thing standing between
// a spread and an unbounded skill, that they still meet the detonate ceiling, and
// that every shipped spread is inside them.
func TestASpreadConsumerIsBoundedByTheShapeItBuys(t *testing.T) {
	book, statuses := mustSkills(t), mustStatuses(t)
	shapes, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load shapes: %v", err)
	}

	// The ceiling, derived rather than written: the whole cast at the splash
	// share, over one plain attack. It is the figure the detonate rule stops at,
	// and if the pattern book ever moved past it this is the line that says so.
	widest := scale.Base + (shapes.MaxTargets-1)*shapes.SplashPower
	if widest > 2*scale.Base {
		t.Errorf("the widest shape the book can express is worth %d per mille of a plain attack, over the %d a detonate may not pass: "+
			"the two rules for the two currencies no longer meet, so a consume can be paid more in shape than it could ever be paid in power",
			widest, 2*scale.Base)
	}

	found := 0
	for _, current := range book.Skills() {
		if !current.Requires.SpreadsOn() {
			continue
		}
		found++
		if current.Requires.BonusPower != 0 {
			t.Errorf("skill %q is paid both in %d bonus power and in a shape: a consume may take one payment or the other, and taking both is the ceiling counted once and charged twice",
				current.ID, current.Requires.BonusPower)
		}
		base, err := shapes.Lookup(current.Pattern)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		spread, err := shapes.Lookup(current.Requires.Spreads)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		if spread.MaxTargets() <= base.MaxTargets() {
			t.Errorf("skill %q spreads from %q over %d cells to %q over %d: a spread that reaches no further is a consume paid nothing at all",
				current.ID, base.Name, base.MaxTargets(), spread.Name, spread.MaxTargets())
		}
		// Every one of them spends a counted number of stacks rather than the
		// pile. Taking the lot for one cast is what a detonate does, and it is
		// the one thing that would make accumulating pointless — a magazine
		// emptied by its first shot is a magazine of one.
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		if kind.Category == status.Charge && current.Requires.ConsumeStacks == 0 {
			t.Errorf("skill %q spends the whole pile of %q for one cast, so nothing is ever worth accumulating",
				current.ID, kind.ID)
		}
	}
	if found == 0 {
		t.Skip("no shipped skill spreads, so the bound above is the only thing this measured")
	}
	t.Logf("%d shipped skills spread; the widest shape is worth %d per mille of a plain attack", found, widest)
}

// TestACounterCarriesNoEffectOfItsOwn holds the argument its own stack cap rests
// on, over the shipped book rather than at the parser.
//
// status.ParseBook refuses a charge carrying a modifier, and this is the same
// claim asked of the data: the reason a counter may stack to nine hundred and
// ninety nine while everything else stops at five is that nine hundred and
// ninety nine of it does nothing. A charge that ticked, cut healing or moved a
// stat would be an effect with no bound at all, and the cap would have gone from
// a decision to an oversight.
func TestACounterCarriesNoEffectOfItsOwn(t *testing.T) {
	statuses := mustStatuses(t)
	counted := 0
	for _, kind := range statuses.Kinds() {
		if kind.Category != status.Charge {
			continue
		}
		counted++
		if kind.MaxStacks <= statuses.MaxStacks {
			t.Errorf("status %q caps at %d stacks, which the ordinary limit of %d already allows: it needs no category of its own",
				kind.ID, kind.MaxStacks, statuses.MaxStacks)
		}
		if len(kind.Modifiers) != 0 || kind.TickPower != 0 || kind.HealShare != 0 {
			t.Errorf("status %q is a counter that also does something: %d modifier(s), %d tick power, %d heal share",
				kind.ID, len(kind.Modifiers), kind.TickPower, kind.HealShare)
		}
		if kind.Permanent {
			// Remove refuses a permanent status, which is what keeps a trait from
			// being dispelled — and it is Remove that a consume goes through. A
			// permanent counter would therefore be a counter nothing could ever
			// spend, accumulating forever and paying out never.
			t.Errorf("status %q is a permanent counter, so nothing can ever spend it", kind.ID)
		}
	}
	if counted == 0 {
		t.Skip("no counter is shipped, so there is nothing here to hold")
	}
}
