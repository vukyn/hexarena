// What a blocked strike says is left of the wall it hit.
//
// A block charge cancels one strike, so a volley of several eats several of
// them and the log has to count down as it goes: the first blocked strike of a
// pair against a wall of three leaves two, the second leaves one. The whole
// volley's spend is taken off the status set in one call BEFORE the strikes are
// walked, so a line reading the set again inside the walk reports the figure
// after all of it and every blocked strike of a multi-strike skill claims the
// same remainder — the log telling a reader the barrier fell all at once.
//
// ⚠️ **The wall has to be DEEPER than the volley spends for that to be visible
// at all.** Against one charge, and against a wall exactly as deep as the volley
// is long, the buggy reading and the correct one agree on the last figure the
// reader sees, which is the figure a test naturally asserts. So the decisive
// fixture here is three charges against two connecting strikes, and what is
// asserted is the SEQUENCE.
package battle_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// walledBooks is the shared fixture book with two attacks appended for this file
// alone. They are appended here rather than declared in books() because a skill
// in the shared book is a skill every other test in the package can see, and
// several of those pick their kits by property rather than by name — a two
// strike attack and a four strike one would quietly change what they are
// measuring.
//
//   - `pair` connects on every strike, so a wall deeper than it is long is spent
//     down by exactly two and the last figure it reports is a real remainder
//     rather than nought.
//   - `spray` misses about half the time over four strikes, which is the only way
//     to get a miss standing BETWEEN two blocks: combat.Roll checks accuracy
//     before it offers a charge, so a missed strike spends nothing, and nothing
//     in the shared book both strikes more than once and can miss (Rules.Chance
//     returns outright at an accuracy of a thousand, so no amount of dodge on the
//     target will do it).
func walledBooks(t *testing.T) battle.Books {
	t.Helper()
	base := books(t)
	deps := skill.Deps{Patterns: base.Patterns, Statuses: base.Statuses}
	extra, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"pair","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":2,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"spray","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":4,"accuracy":500,"cooldown":0,"target":"enemy"}
	]}`), deps)
	if err != nil {
		t.Fatalf("the volleys this file needs: %v", err)
	}
	book, err := base.Skills.Append(deps, extra.Skills()...)
	if err != nil {
		t.Fatalf("appending them to the fixture book: %v", err)
	}
	base.Skills = book
	return base
}

// walledVolley drives one duel in which the foe spends every turn in front of
// the ally raising its shield, then has the ally cast once into the wall that
// built up. It hands back what the wall held before the cast, what it holds
// after, and the events of that cast and nothing else.
//
// The wall's DEPTH is set by the ally's speed rather than by counting braces.
// A wait is 1_000_000/speed, so a slower ally gives the foe more turns in front
// of it: 150 against the foe's 200 buys one charge, 100 buys two and anything at
// or under 60 buys the cap of three. Bracing on every turn rather than stopping
// at a count is what keeps the wall fresh — a charge lasts two turns and the foe
// takes many, so a wall raised early and then left alone would have expired by
// the time the ally moved. Set.Apply refreshes every stack it already holds
// before it decides whether there is room for another, so a brace at the cap
// still buys the durations.
//
// ⚠️ Battle.Drain EMPTIES the buffer, so it is called after Begin, after every
// Advance and after every Act — the batch returned at the end then holds one
// cast, which is the only thing anything here reads.
func walledVolley(t *testing.T, seed uint64, allySpeed int64, allySkill string) (wall, left int, events []battle.Event) {
	t.Helper()
	fight := mustBattle(t, walledBooks(t), seed, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, allySpeed),
			Skills: []string{allySkill}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 200),
			Skills: []string{"brace"}},
	})
	fight.Begin()
	fight.Drain()
	for turn := 0; ; turn++ {
		if turn > 80 {
			t.Fatal("the ally never got a turn, so nothing here is measuring a cast")
		}
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit == "a" {
			if prompt.Skipped {
				t.Fatal("the ally's turn was taken from it, so nothing here is measuring a cast")
			}
			break
		}
		if prompt.Skipped {
			continue
		}
		if err := fight.Act("brace", acrossTheBoard); err != nil {
			t.Fatalf("the foe's brace: %v", err)
		}
		fight.Drain()
	}
	foe, found := fight.Unit("f")
	if !found {
		t.Fatal("the foe is not on the board")
	}
	wall = foe.Statuses.Stacks("block")
	if err := fight.Act(allySkill, acrossTheBoard); err != nil {
		t.Fatalf("the ally's %s: %v", allySkill, err)
	}
	events = fight.Drain()
	return wall, foe.Statuses.Stacks("block"), events
}

// blockedCounts is the "N charges left" figure of every blocked strike, in the
// order the log names them — which is the order a reader reads them in, and the
// whole of what this file is about.
func blockedCounts(events []battle.Event) []int64 {
	out := make([]int64, 0, 4)
	for _, event := range find(events, battle.Blocked) {
		out = append(out, event.Remaining)
	}
	return out
}

// volleyShape is the volley written as one strike per letter — B blocked, M
// missed, D landed. It is asserted as a premise before any charge figure is
// read: a seed that stops producing the arrangement a row was chosen for would
// otherwise leave that row passing while measuring something else entirely.
func volleyShape(events []battle.Event) string {
	var shape strings.Builder
	for _, event := range events {
		switch event.Kind {
		case battle.Blocked:
			shape.WriteByte('B')
		case battle.Missed:
			shape.WriteByte('M')
		case battle.Damaged:
			shape.WriteByte('D')
		}
	}
	return shape.String()
}

func sameCounts(got []int64, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestAMultiStrikeVolleyCountsItsBlockChargesDownStrikeByStrike is the guard the
// defect was reported against, and the fixture is chosen so that a correct
// implementation and the one that read the set inside the loop cannot agree.
//
// Three charges, two connecting strikes: the volley spends two and leaves one,
// so the reader must be told two and then one. Reading the set again inside the
// loop reports one for BOTH — the barrier falling all at once — and no shallower
// wall can tell the two apart, because at one charge both say nought and at a
// wall exactly as deep as the volley both end at nought.
//
// The last figure is asserted against what the set is actually left holding
// rather than against a written-down number, because that is the claim worth
// making: the line a reader sees at the end of a volley is the wall that is
// really there for the next attacker.
func TestAMultiStrikeVolleyCountsItsBlockChargesDownStrikeByStrike(t *testing.T) {
	wall, left, events := walledVolley(t, 5, 10, "pair")
	if wall != 3 {
		t.Fatalf("the foe went into the volley holding %d charges, want 3: a wall no "+
			"deeper than the volley cannot tell a countdown from a single figure", wall)
	}
	if shape := volleyShape(events); shape != "BB" {
		t.Fatalf("the volley came out %q, want BB: both strikes have to be blocked "+
			"or there is no sequence here to read", shape)
	}
	counts := blockedCounts(events)
	if want := []int64{2, 1}; !sameCounts(counts, want) {
		// ⚠️ The message says what was read and stops there. It used to name
		// the flat-volley cause outright, which is only one of the two ways
		// this goes wrong: a reading taken before the charge is spent answers
		// [3 2] here, and a message blaming the whole-volley spend for that
		// sends the next reader at the wrong line.
		t.Errorf("two strikes blocked by a wall of three reported %v charges left, want %v: "+
			"the charges have to come down one per blocked strike. %v is the whole volley's "+
			"spend reported against every strike; %v is the wall read before the charge is spent",
			counts, want, []int64{1, 1}, []int64{3, 2})
	}
	if left != 1 {
		t.Fatalf("the wall is left holding %d charges, want 1", left)
	}
	if len(counts) > 0 && counts[len(counts)-1] != int64(left) {
		t.Errorf("the last blocked strike reported %d charges left while the wall is "+
			"really holding %d: the figure a reader is given at the end of a volley "+
			"has to be the wall the next attacker will meet",
			counts[len(counts)-1], left)
	}
}

// TestASingleStrikeBlockStillReportsTheChargesLeftAfterIt is the other half of
// the same claim, and it is here to say the fix moved nothing it should not
// have. One strike spends one charge, so the figure after the volley and the
// figure after that strike are the same number — the reading that was wrong for
// a volley was right for a single blow, at every depth of wall.
func TestASingleStrikeBlockStillReportsTheChargesLeftAfterIt(t *testing.T) {
	for _, row := range []struct {
		name  string
		speed int64
		wall  int
		want  int64
	}{
		{"one charge", 150, 1, 0},
		{"two charges", 100, 2, 1},
		{"three charges", 50, 3, 2},
	} {
		t.Run(row.name, func(t *testing.T) {
			wall, left, events := walledVolley(t, 5, row.speed, "strike")
			if wall != row.wall {
				t.Fatalf("the foe went into the strike holding %d charges, want %d",
					wall, row.wall)
			}
			if shape := volleyShape(events); shape != "B" {
				t.Fatalf("the cast came out %q, want B", shape)
			}
			if counts := blockedCounts(events); !sameCounts(counts, []int64{row.want}) {
				t.Errorf("one strike blocked by a wall of %d reported %v charges left, "+
					"want [%d]", row.wall, counts, row.want)
			}
			if left != int(row.want) {
				t.Fatalf("the wall is left holding %d charges, want %d", left, row.want)
			}
		})
	}
}

// TestOnlyTheBlockedStrikesOfAVolleyCountDown says the countdown follows the
// charges rather than the strikes. A missed strike never reaches a charge at all
// — combat.Roll checks accuracy before it offers one — and a strike that landed
// got past the wall because there was nothing left of it, so neither may move
// the figure.
//
// Two arrangements, because they fail differently. A miss standing BETWEEN two
// blocks catches a countdown driven off the strike index, which would skip a
// number over the miss. A landed strike after the blocks catches the same thing
// from the other end.
func TestOnlyTheBlockedStrikesOfAVolleyCountDown(t *testing.T) {
	for _, row := range []struct {
		name  string
		seed  uint64
		skill string
		shape string
		want  []int64
		left  int
	}{
		// Two misses standing between the two blocked strikes, and a charge left
		// over at the end: the strongest single reading in this file, because the
		// countdown has to step over the misses AND stop short of nought.
		{"misses between the blocks", 1, "spray", "BMMB", []int64{2, 1}, 1},
		// The wall is spent to nothing by the third strike, so the fourth is the
		// first thing this fixture lets through.
		{"a landed strike after the blocks", 18, "spray", "BBBD", []int64{2, 1, 0}, 0},
	} {
		t.Run(row.name, func(t *testing.T) {
			wall, left, events := walledVolley(t, row.seed, 10, row.skill)
			if wall != 3 {
				t.Fatalf("the foe went into the volley holding %d charges, want 3", wall)
			}
			if shape := volleyShape(events); shape != row.shape {
				t.Fatalf("the volley came out %q, want %q: this row is chosen for that "+
					"arrangement and measures nothing without it", shape, row.shape)
			}
			if counts := blockedCounts(events); !sameCounts(counts, row.want) {
				t.Errorf("the blocked strikes of %s reported %v charges left, want %v: "+
					"only a strike a charge cancelled may move the figure",
					row.shape, counts, row.want)
			}
			if left != row.left {
				t.Fatalf("the wall is left holding %d charges, want %d", left, row.left)
			}
		})
	}
}

// TestAVolleySpendsTheSameChargesHoweverItIsReported is the control on all of
// the above. This is a REPORTING fix: what a blocked strike says is left changed,
// what a volley actually takes off the status set did not, and combat.Roll — which
// decides that spend — was not touched. Without this, a countdown that also
// started removing a charge per line would pass every assertion in this file.
//
// The claim is arithmetic and holds for every arrangement: one charge is spent
// per blocked strike and by nothing else, so what the wall is left holding is
// what it held less the number of blocked strikes.
func TestAVolleySpendsTheSameChargesHoweverItIsReported(t *testing.T) {
	for _, row := range []struct {
		name  string
		seed  uint64
		speed int64
		skill string
	}{
		{"two strikes into a wall of three", 5, 10, "pair"},
		{"one strike into a wall of three", 5, 50, "strike"},
		{"one strike into a wall of one", 5, 150, "strike"},
		{"a four strike volley with misses in it", 1, 10, "spray"},
		{"a four strike volley that empties the wall", 18, 10, "spray"},
	} {
		t.Run(row.name, func(t *testing.T) {
			wall, left, events := walledVolley(t, row.seed, row.speed, row.skill)
			blocked := len(find(events, battle.Blocked))
			if blocked == 0 {
				t.Fatalf("no strike of %q was blocked, so this row measures no spend at all",
					volleyShape(events))
			}
			if want := wall - blocked; left != want {
				t.Errorf("a wall of %d met %q and is left holding %d, want %d: a volley "+
					"spends one charge per blocked strike and the way the log reports "+
					"them may not change that", wall, volleyShape(events), left, want)
			}
		})
	}
}
