package draft_test

import (
	"fmt"
	"testing"

	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestNoDraftThatFitsCanRunOutOfCharacters is the finding this package was
// written around, and it replaces a runtime rule with a refused configuration.
//
// # What TODO.md used to say, and why it was wrong
//
// The design record carried this: *"Bans being optional is what makes a
// shortfall a runtime failure rather than a refused configuration. A room that
// is legal when it opens can still run out of characters partway through a
// draft, so whatever the counts are, the draft owes a rule for the moment the
// pool would no longer seat both sides: refuse the ban, and grey the slot."*
//
// It has the optionality backwards. Every ban and every pick takes **exactly
// one** character out of the pool, and a ban that is skipped takes out none — so
// optionality can only ever leave the pool *fuller*, never emptier. The most a
// whole draft can remove is `2*PicksPerSide + 2*BansPerSide`, with every ban
// spent, and that is the number Fits already requires the pool to hold. So
// before the k-th of at most that many decisions at most k-1 characters are
// gone, at least one is left for the decision about to be taken, and the pool
// never falls below the picks the two sides still owe.
//
// The consequence is that there is no runtime rule to write: no ban to refuse,
// no slot to grey, in this package or in the state machine, the room or the
// screen above it. A room that opened legally finishes its draft.
//
// # What this walk drives, and why it is exhaustive rather than sampled
//
// A claim of the form "this can never happen" is not something a random sample
// can support, and randomness is the one thing this package's own import ban
// refuses anyway. So: both formats, every pool size from nought to forty,
// **every** legal ban/skip sequence — each side's ban slots independently spent
// or skipped, which is 2^B per side — and three different orders of the whole
// sequence, because the argument does not depend on the order and driving only
// the settled one would leave that untested.
//
// The three orders matter for step 2 rather than for step 1. TODO.md § "Ban and
// pick" settles all bans first and then all picks; its own prose ("the two sides
// take turns banning a character and picking one") reads like alternation and is
// the other candidate; and "every removal first" is the adversarial order that
// empties the pool as early as anything can. If the invariant holds under the
// third it holds under any order at all, which is what makes this a proof of the
// arithmetic and not of one sequence.
//
// *Sees:* a ban count raised past what the pool can seat; Fits made permissive;
// the arithmetic and the walk disagreeing about how much a draft removes; a
// draft that removes more than one character per decision.
// *Cannot see:* anything about the *state machine* — that a draft asks the right
// side at the right time is step 2's, and this only says the pool is never the
// thing that stops it.
func TestNoDraftThatFitsCanRunOutOfCharacters(t *testing.T) {
	const largestPool = 40
	cases, allowedCases, decisions := 0, 0, 0

	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		picks, bans := draft.PicksPerSide(format), draft.BansPerSide(format)
		if picks <= 0 || bans <= 0 {
			t.Fatalf("%s draws %d picks and %d bans a side, and a walk over nothing "+
				"measures nothing", format, picks, bans)
		}
		for poolSize := 0; poolSize <= largestPool; poolSize++ {
			refusal := draft.Fits(poolSize, format)
			allowed := refusal == nil
			slack := draft.Slack(poolSize, format)

			for host := 0; host < 1<<bans; host++ {
				for guest := 0; guest < 1<<bans; guest++ {
					spend := [2][]bool{spentSlots(host, bans), spentSlots(guest, bans)}
					spent := countSpent(spend)

					for _, order := range draftOrders {
						cases++
						walked := walkDraft(order.of(picks, bans, spend), poolSize, picks)
						decisions += walked.decisions
						if !allowed {
							continue
						}
						allowedCases++
						where := fmt.Sprintf("%s, pool %d, %s, %d of %d bans spent",
							format, poolSize, order.name, spent, 2*bans)

						if walked.dry != "" {
							t.Errorf("%s: the pool ran out at %s. Fits allowed this "+
								"configuration, so either the arithmetic is wrong or a "+
								"decision removes more than one character", where, walked.dry)
							continue
						}
						if walked.unseatable != "" {
							t.Errorf("%s: after %s the pool held %d and the two sides still "+
								"owed picks it could not seat", where, walked.unseatable,
								walked.remaining)
							continue
						}
						if want := 2*picks + 2*bans; walked.decisions != want {
							t.Errorf("%s: the draft took %d decisions and a full one is %d",
								where, walked.decisions, want)
						}
						// The closed form, checked by the walk rather than restated
						// by it: a draft removes one character per pick and one per
						// spent ban, and nothing else.
						if want := 2*picks + spent; walked.removed != want {
							t.Errorf("%s: the draft removed %d characters and the arithmetic "+
								"says %d", where, walked.removed, want)
						}
						if want := poolSize - walked.removed; walked.remaining != want {
							t.Errorf("%s: %d left after removing %d from %d",
								where, walked.remaining, walked.removed, poolSize)
						}
						if want := slack + (2*bans - spent); walked.remaining != want {
							t.Errorf("%s: %d left, and slack %d plus %d unspent bans says %d",
								where, walked.remaining, slack, 2*bans-spent, want)
						}
						if walked.remaining < slack {
							t.Errorf("%s: %d left, which is under the slack of %d — skipping "+
								"a ban has somehow cost the pool a character",
								where, walked.remaining, slack)
						}
						// The last pick always has somebody to take, which is the
						// claim at its tightest. When every ban is spent it has
						// exactly `slack + 1`, which is why TODO.md's *other* slack
						// note is still real: at slack nought the final pick is a
						// list of one and is not a decision at all.
						if walked.candidatesAtLastPick < 1 {
							t.Errorf("%s: the last pick had %d characters to choose from",
								where, walked.candidatesAtLastPick)
						}
						if spent == 2*bans && walked.candidatesAtLastPick != slack+1 {
							t.Errorf("%s: the last pick saw %d candidates and slack %d says it "+
								"should see %d", where, walked.candidatesAtLastPick, slack, slack+1)
						}
					}
				}
			}
		}
	}

	// A walk that ran no iterations agrees with every claim there is, and a walk
	// in which nothing ever *fitted* would agree with a Fits that refused
	// everything. Both counts are asserted by value so a loop bound that moved
	// says so.
	if want := 41*16*len(draftOrders) + 41*64*len(draftOrders); cases != want {
		t.Errorf("the walk drove %d cases and the bounds say %d; a case count that moved "+
			"without the bounds moving means the walk is not covering what it claims",
			cases, want)
	}
	if allowedCases == 0 {
		t.Fatal("no configuration in the whole walk fitted, so the invariant was never " +
			"asked about anything: it measures nothing")
	}
	if decisions == 0 {
		t.Fatal("the walk took no decisions at all")
	}
	t.Logf("drove %d configurations (%d of them allowed by Fits) over %d decisions; "+
		"no draft that Fits allowed ran out of characters", cases, allowedCases, decisions)
}

// TestADraftThatDoesNotFitCanRunOutOfCharacters is the other half of the walk
// above, and without it that one measures nothing.
//
// ⚠️ **A simulator that never runs dry agrees with "it never runs dry".** So
// this drives the pool that is short by exactly one with every ban spent — the
// configuration Fits refuses most narrowly — and holds that the draft *does*
// run out. The failure the walk says cannot happen is therefore a failure it can
// see.
//
// It also pins the thing that makes Fits deliberately **stricter** than the
// arithmetic strictly needs: at that same pool size, a draft in which both sides
// skip every ban finishes comfortably. Fits refuses it all the same, because a
// side's bans are not knowable when the room opens and a refused configuration
// beats a draft that fails halfway through one.
func TestADraftThatDoesNotFitCanRunOutOfCharacters(t *testing.T) {
	observed := 0
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		picks, bans := draft.PicksPerSide(format), draft.BansPerSide(format)
		shortByOne := 2*picks + 2*bans - 1
		if draft.Fits(shortByOne, format) == nil {
			t.Fatalf("%s: a pool of %d is one short of what the draft removes and Fits "+
				"allowed it", format, shortByOne)
		}

		allSpent := [2][]bool{spentSlots(1<<bans-1, bans), spentSlots(1<<bans-1, bans)}
		for _, order := range draftOrders {
			walked := walkDraft(order.of(picks, bans, allSpent), shortByOne, picks)
			if walked.dry == "" {
				t.Errorf("%s, pool %d, %s, every ban spent: the draft finished with %d "+
					"characters left. This is the case the exhaustive walk's claim is "+
					"measured against, so a walk that cannot fail here cannot pass either",
					format, shortByOne, order.name, walked.remaining)
				continue
			}
			observed++
		}

		noneSpent := [2][]bool{spentSlots(0, bans), spentSlots(0, bans)}
		walked := walkDraft(draftOrders[0].of(picks, bans, noneSpent), shortByOne, picks)
		if walked.dry != "" {
			t.Errorf("%s, pool %d, no ban spent: the draft still ran out at %s, and %d picks "+
				"a side out of %d cannot", format, shortByOne, walked.dry, picks, shortByOne)
		}
		t.Logf("%s: a pool of %d runs out with every ban spent and finishes with %d left "+
			"when none is — Fits refuses both, on purpose",
			format, shortByOne, walked.remaining)
	}
	if observed == 0 {
		t.Fatal("no configuration ran out of characters, so the exhaustive walk's claim is " +
			"unfalsifiable as written")
	}
}

// step is one decision of a draft and whether it takes a character out of the
// pool. A pick always does; a ban does only when it is spent, and that
// difference is the whole hinge of the claim being tested.
type step struct {
	// what names the decision, so a failure says which one found an empty pool.
	what  string
	pick  bool
	takes bool
}

// draftOrders is the sequence a draft's decisions come in. There are three
// because the invariant does not depend on the order and a walk over one order
// would not have shown that.
var draftOrders = []struct {
	name string
	of   func(picks, bans int, spend [2][]bool) []step
}{
	// The order settled in TODO.md § "Ban and pick": a banning stage, then a
	// picking stage.
	{"bans then picks", bansThenPicks},
	// The other reading of that section's own prose, "the two sides take turns
	// banning a character and picking one", kept because the sentence will be
	// re-read as alternation by somebody.
	{"a ban and a pick each turn", banAndPickEachTurn},
	// The adversarial order: every character that will be removed, removed as
	// early as possible. No order can empty the pool sooner, so an invariant that
	// holds here holds under every order there is.
	{"every removal first", removalsFirst},
}

func bansThenPicks(picks, bans int, spend [2][]bool) []step {
	out := make([]step, 0, 2*picks+2*bans)
	for slot := range bans {
		for side := range 2 {
			out = append(out, ban(side, slot, spend[side][slot]))
		}
	}
	for round := range picks {
		for side := range 2 {
			out = append(out, pick(side, round))
		}
	}
	return out
}

func banAndPickEachTurn(picks, bans int, spend [2][]bool) []step {
	out := make([]step, 0, 2*picks+2*bans)
	for round := range max(picks, bans) {
		for side := range 2 {
			if round < bans {
				out = append(out, ban(side, round, spend[side][round]))
			}
			if round < picks {
				out = append(out, pick(side, round))
			}
		}
	}
	return out
}

func removalsFirst(picks, bans int, spend [2][]bool) []step {
	settled := bansThenPicks(picks, bans, spend)
	out := make([]step, 0, len(settled))
	for _, one := range settled {
		if one.takes {
			out = append(out, one)
		}
	}
	for _, one := range settled {
		if !one.takes {
			out = append(out, one)
		}
	}
	return out
}

func ban(side, slot int, spent bool) step {
	spelling := "skips ban"
	if spent {
		spelling = "spends ban"
	}
	return step{what: fmt.Sprintf("%s %s %d", sideName(side), spelling, slot+1), takes: spent}
}

func pick(side, round int) step {
	return step{
		what:  fmt.Sprintf("%s takes pick %d", sideName(side), round+1),
		pick:  true,
		takes: true,
	}
}

func sideName(side int) string {
	if side == 0 {
		return "the host"
	}
	return "the guest"
}

// spentSlots reads a bit pattern as which of a side's ban slots it spent, which
// is how the walk enumerates every legal ban/skip sequence for a side rather
// than only the counts.
func spentSlots(pattern, bans int) []bool {
	out := make([]bool, bans)
	for slot := range bans {
		out[slot] = pattern&(1<<slot) != 0
	}
	return out
}

func countSpent(spend [2][]bool) int {
	total := 0
	for _, side := range spend {
		for _, slot := range side {
			if slot {
				total++
			}
		}
	}
	return total
}

// walked is what one run of a draft did to its pool.
type walked struct {
	decisions int
	removed   int
	remaining int
	// dry names the decision that found an empty pool, and is empty when none
	// did. The walk stops there, because a draft that cannot seat a decision does
	// not go on to take the next one.
	dry string
	// unseatable names the decision after which the pool held fewer characters
	// than the picks the two sides still owed — the stronger reading of "run out
	// of characters", and the one TODO.md's refused rule was written about.
	unseatable string
	// candidatesAtLastPick is how many characters the final pick chose from,
	// which is what TODO.md's surviving slack note is about.
	candidatesAtLastPick int
}

// walkDraft runs a sequence of decisions against a pool of poolSize and reports
// what happened to it. It counts characters rather than holding them: which
// character a side bans cannot change how many are left, and a draft.Pool of
// named characters here would make the walk a test of slice bookkeeping instead
// of the arithmetic.
func walkDraft(steps []step, poolSize, picks int) walked {
	out := walked{remaining: poolSize}
	owed := 2 * picks
	lastPick := -1
	for at, one := range steps {
		if one.pick {
			lastPick = at
		}
	}
	for at, one := range steps {
		out.decisions++
		if at == lastPick {
			out.candidatesAtLastPick = out.remaining
		}
		if one.takes {
			if out.remaining == 0 {
				out.dry = one.what
				return out
			}
			out.remaining--
			out.removed++
		}
		if one.pick {
			owed--
		}
		if out.remaining < owed && out.unseatable == "" {
			out.unseatable = one.what
		}
	}
	return out
}
