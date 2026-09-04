package draft_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTheCountsAreTheOnesTheDesignRecordSettled is the one place in this package
// the ban numbers are written down twice on purpose.
//
// They are a **decision** rather than a derivation — TODO.md § "Ban and pick"
// (b), two a side at 3v3 and three at 5v5, mirrored — so there is nothing for a
// test to compute them from, and a test that read them off BansPerSide would
// agree with any number at all. Picks are the format's own unit count and are
// asserted against `Units()` rather than against a literal, because that
// genuinely is a derivation.
//
// ⚠️ A failure here is a design change and belongs in TODO.md before it belongs
// in the code: the counts decide how much cast a draft needs, and the exhaustive
// walk's whole claim is measured against them.
func TestTheCountsAreTheOnesTheDesignRecordSettled(t *testing.T) {
	for _, one := range []struct {
		format wire.Format
		bans   int
	}{
		{wire.Format3v3, 2},
		{wire.Format5v5, 3},
	} {
		if got := draft.BansPerSide(one.format); got != one.bans {
			t.Errorf("%s draws %d bans a side and TODO.md § \"Ban and pick\" (b) settled %d",
				one.format, got, one.bans)
		}
		if got := draft.PicksPerSide(one.format); got != one.format.Units() {
			t.Errorf("%s picks %d a side and fields %d units", one.format, got, one.format.Units())
		}
		// Slack is the arithmetic and nothing else, so it is checked against the
		// arithmetic spelled out rather than against a table of figures that would
		// have to move with the counts.
		for _, poolSize := range []int{0, 1, 10, 15, 16, 40} {
			want := poolSize - 2*one.format.Units() - 2*one.bans
			if got := draft.Slack(poolSize, one.format); got != want {
				t.Errorf("%s slack over a pool of %d is %d and the arithmetic says %d",
					one.format, poolSize, got, want)
			}
		}
	}
}

// TestAShortfallIsNamedInCharacters pins the refusal's wording by value, which
// is the same terms `TestARefusalKeepsTheWordingTheCommandLinePrints` holds the
// command line's refusals under: a refusal a player reads is a contract, and
// "it does not fit" with no figures is a message nobody can act on.
//
// ⚠️ **The singular is the case that ships, and it is the one a plural rule
// gets wrong.** internal/i18n has already had to fix "1 turns of cooldown left"
// once; today's shortfall at 5v5 is exactly one character, so "1 characters" is
// the wording a reader would actually meet. Both counts are asserted, because a
// test of the plural alone goes green on the bug.
func TestAShortfallIsNamedInCharacters(t *testing.T) {
	for _, one := range []struct {
		poolSize int
		format   wire.Format
		want     string
	}{
		// The measured case: the shipped pool is fifteen and a 5v5 needs sixteen.
		{15, wire.Format5v5, "a 5v5 draft takes 10 picks and 6 bans out of one shared pool, " +
			"which is 16 characters, and the pool holds 15: it is short by one character"},
		{13, wire.Format5v5, "a 5v5 draft takes 10 picks and 6 bans out of one shared pool, " +
			"which is 16 characters, and the pool holds 13: it is short by 3 characters"},
		{0, wire.Format3v3, "a 3v3 draft takes 6 picks and 4 bans out of one shared pool, " +
			"which is 10 characters, and the pool holds 0: it is short by 10 characters"},
		{9, wire.Format3v3, "a 3v3 draft takes 6 picks and 4 bans out of one shared pool, " +
			"which is 10 characters, and the pool holds 9: it is short by one character"},
	} {
		err := draft.Fits(one.poolSize, one.format)
		if err == nil {
			t.Errorf("a pool of %d seats a %s draft, and the arithmetic says it is short by %d",
				one.poolSize, one.format, -draft.Slack(one.poolSize, one.format))
			continue
		}
		if err.Error() != one.want {
			t.Errorf("a pool of %d at %s refuses with\n  %q\nand the wording is\n  %q",
				one.poolSize, one.format, err.Error(), one.want)
		}
	}
}

// TestFitsAllowsAPoolWithNothingToSpare crosses the boundary rather than
// approaching it: slack nought is the *last* pool that fits, and an off-by-one
// in either direction is the only thing this function can get wrong.
//
// The pool with nothing to spare is also the state TODO.md's surviving slack
// note is about — the final pick is a list of one — so it fits and is worth
// drawing, which are two different statements about the same number.
func TestFitsAllowsAPoolWithNothingToSpare(t *testing.T) {
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		exactly := 2*draft.PicksPerSide(format) + 2*draft.BansPerSide(format)
		if got := draft.Slack(exactly, format); got != 0 {
			t.Errorf("%s: a pool of %d should have nothing to spare and has %d",
				format, exactly, got)
		}
		if err := draft.Fits(exactly, format); err != nil {
			t.Errorf("%s: a pool of exactly %d does not fit: %v", format, exactly, err)
		}
		if err := draft.Fits(exactly-1, format); err == nil {
			t.Errorf("%s: a pool of %d is one short and fits", format, exactly-1)
		}
		if err := draft.Fits(exactly+1, format); err != nil {
			t.Errorf("%s: a pool of %d has one to spare and does not fit: %v",
				format, exactly+1, err)
		}
	}
}

// TestFitsRefusesAFormatTheGameDoesNotOffer is the gate BansPerSide leans on,
// and it has to be a test of its own because nothing else can see it.
//
// ⚠️ **Deleting the gate reddened nothing in the whole package**, measured, and
// the reason is that an unknown format answers nought bans: `Fits(15,
// Format(4))` then computes `15 - 8 - 0` and returns nil, cheerfully allowing a
// draft of a format the game does not have. wire.Format's own comment is the
// standing rule here — "a Format of 4 is refused rather than misread" — and this
// is where this package obeys it.
//
// A zero of bans is *no answer* rather than "a format with no bans", which is
// why the second half asserts that too: a reader who found BansPerSide returning
// nought and concluded a Format(4) draft simply has no bans would be reading a
// gap as a decision.
func TestFitsRefusesAFormatTheGameDoesNotOffer(t *testing.T) {
	for _, format := range []wire.Format{0, 1, 2, 4, 6, -3} {
		if format.Valid() {
			t.Fatalf("%s is a format the game offers, so it does not belong in this table", format)
		}
		err := draft.Fits(10_000, format)
		if err == nil {
			t.Errorf("a %s draft out of a pool of ten thousand is allowed, and %s is not a "+
				"format the game offers — the pool being large is not the question", format, format)
			continue
		}
		if want := "a draft of " + format.String() + " is not a format this game offers"; err.Error() != want {
			t.Errorf("%s refuses with %q and the wording is %q", format, err.Error(), want)
		}
		if got := draft.BansPerSide(format); got != 0 {
			t.Errorf("%s draws %d bans a side, and an unknown format has no answer to give",
				format, got)
		}
	}
	// The two real formats must not be caught by that gate, or the refusal above
	// would be refusing everything and saying nothing.
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		if err := draft.Fits(2*draft.PicksPerSide(format)+2*draft.BansPerSide(format), format); err != nil {
			t.Errorf("%s is a format the game offers and the gate refused it: %v", format, err)
		}
	}
}
