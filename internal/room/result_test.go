package room_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// theBattlesOutcomes is how many ways a battle can end, restated here as a
// literal on purpose.
//
// ⚠️ Reading battle.OutcomeCount and comparing it to itself would agree with any
// number at all. Four is the number, it has been four since the outcome became a
// typed field, and the day somebody adds a fifth *for the room's sake* this line
// is the red test that asks them not to. A forfeit, a dropped socket and a
// refused join are results of the **match**; a battle.Outcome is read by
// --verify and by every renderer, and a dropped socket is not a way a battle can
// end.
const theBattlesOutcomes = 4

// TestAForfeitAndADisconnectAddNothingToTheBattlesOutcomes drives a match to
// each of the two routes a forfeit has, so that neither is the dead branch this
// repository has recorded several times — a rule whose only real case is
// unreachable — and checks that the core enum is exactly where it was.
func TestAForfeitAndADisconnectAddNothingToTheBattlesOutcomes(t *testing.T) {
	if battle.OutcomeCount != theBattlesOutcomes {
		t.Fatalf("battle.OutcomeCount is %d and was %d: if a way for a *match* to end has been "+
			"added to the enum a *battle* ends by, it belongs in room.Verdict instead",
			battle.OutcomeCount, theBattlesOutcomes)
	}

	for _, one := range []struct {
		name string
		// take is the route: it ends the match and returns what the room said.
		take func(*testing.T, *room.Room, *table, wire.Seat) []room.Outbound
		want room.Forfeit
	}{
		{
			name: "three allowances run out",
			want: room.ForfeitTimedOut,
			take: func(t *testing.T, opened *room.Room, clients *table, victim wire.Seat) []room.Outbound {
				var last []room.Outbound
				for steps := 0; !opened.Finished(); steps++ {
					if steps > 200 {
						t.Fatal("the match did not forfeit")
					}
					onTurn, out := answerFor(t, opened, clients, victim)
					if onTurn == victim {
						last = out
					}
				}
				return last
			},
		},
		{
			name: "a peer goes away",
			want: room.ForfeitLeft,
			take: func(t *testing.T, opened *room.Room, clients *table, victim wire.Seat) []room.Outbound {
				// One real turn first, so the disconnect lands in a match that
				// was genuinely under way rather than in the opening.
				answerFor(t, opened, clients, "")
				out, err := opened.Left(victim)
				if err != nil {
					t.Fatalf("leave: %v", err)
				}
				return out
			},
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			opened, clients := openMatch(t, config(11, 3))
			const victim = wire.SeatGuest
			last := one.take(t, opened, clients, victim)

			if !opened.Finished() {
				t.Fatal("the match is not over")
			}
			result := opened.Result()
			if result.Verdict != room.VerdictForfeited {
				t.Errorf("the verdict is %q, want %q", result.Verdict, room.VerdictForfeited)
			}
			if result.Forfeit != one.want {
				t.Errorf("the forfeit is %q, want %q", result.Forfeit, one.want)
			}
			if result.Loser != victim || result.Winner != wire.SeatHost {
				t.Errorf("the forfeit reads %q lost to %q", result.Loser, result.Winner)
			}
			// Nothing goes out, which is the gap rather than the design. →
			// Room.forfeit.
			if len(last) != 0 {
				t.Errorf("ending the match produced %d messages, and the protocol has none for it", len(last))
			}
			// Whatever battle it interrupted is **not** recorded as having
			// ended: a forfeit is a result of the match, so the battle simply
			// stops being asked about.
			for index, one := range opened.Played() {
				if one.Outcome == battle.Undecided && !one.Capped {
					t.Errorf("battle %d was recorded as undecided and uncapped, so a forfeit wrote a battle result",
						index+1)
				}
				if int(one.Outcome) >= battle.OutcomeCount {
					t.Errorf("battle %d was recorded with the outcome %q, which is not one of the %d",
						index+1, one.Outcome, battle.OutcomeCount)
				}
			}
			// The room is waiting on nobody, so a transport that kept reading
			// would not be handed a turn.
			if seat, waiting := opened.Awaiting(); waiting {
				t.Errorf("a forfeited match is still waiting on %q", seat)
			}
		})
	}
}

// TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas is the backstop against a
// runaway, so a stalemate nothing on the board can resolve does not hold two
// people at a table for ever.
//
// ⚠️ It needs no new battle.Outcome, and the room does **not** stamp one on the
// battle it stopped. The engine did not conclude anything about that battle, and
// a room writing an outcome the engine never produced would be a second reading
// of how a battle ends — the mistake this repository has recorded most often —
// and the eventual log would fail its own --verify. So the outcome stays
// Undecided, BattleResult.Capped says what happened, and the series counts it
// the way it counts every other draw. That is what "the outcome already carries
// the draws" buys: the standing needs to know that neither seat won, and it
// needs no new way for a battle to end to know it.
func TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas(t *testing.T) {
	// A cap far below any real battle, so the cap is certainly what stops it.
	// The shipped 3v3 runs to dozens of decisions.
	configuration := config(11, 1)
	configuration.TurnCap = 6
	opened, clients := openMatch(t, configuration)

	for steps := 0; !opened.Finished(); steps++ {
		if steps > 4*configuration.TurnCap {
			t.Fatalf("the %d-turn cap did not stop the battle in %d turns", configuration.TurnCap, steps)
		}
		answerFor(t, opened, clients, "")
	}

	played := opened.Played()
	if len(played) != 1 {
		t.Fatalf("a bo1 played %d battles", len(played))
	}
	capped := played[0]
	if !capped.Capped {
		t.Fatalf("the battle ended as %q rather than at the cap", capped.Outcome)
	}
	if !capped.Draw() {
		t.Errorf("a capped battle went to %q", capped.Winner)
	}
	if capped.Outcome != battle.Undecided {
		t.Errorf("a capped battle was recorded as %q; the engine concluded nothing about it, so "+
			"the room must not write an outcome on its behalf", capped.Outcome)
	}
	if capped.Turns <= configuration.TurnCap {
		t.Errorf("the battle opened %d turns against a cap of %d", capped.Turns, configuration.TurnCap)
	}
	// The whole series drew, which is the verdict a bo1 of one drawn battle has
	// and needs no invented tie-break to reach.
	result := opened.Result()
	if result.Verdict != room.VerdictDrawn {
		t.Errorf("a bo1 whose only battle drew has the verdict %q, want %q", result.Verdict, room.VerdictDrawn)
	}
	if result.Winner.Valid() {
		t.Errorf("a drawn match names %q as its winner", result.Winner)
	}
	if result.Wins != [2]int{0, 0} {
		t.Errorf("a drawn match records the standing %v", result.Wins)
	}
	// And the enum is where it was.
	if battle.OutcomeCount != theBattlesOutcomes {
		t.Errorf("battle.OutcomeCount is %d, want %d", battle.OutcomeCount, theBattlesOutcomes)
	}
	// The clients followed the capped battle turn for turn, which is the half of
	// this that is not obvious: the room stops asking on a boundary a mirror also
	// stops at, so no digest disagreed on the way there.
	for _, client := range []*mirror{clients.host, clients.guest} {
		if client.compared == 0 {
			t.Errorf("%s checked no digests in a capped battle", client.seat)
		}
	}
	t.Logf("capped at %d turns with the outcome %q; %d digests agreed on the way",
		capped.Turns, capped.Outcome, clients.host.compared)
}

// TestEveryVerdictAndForfeitIsNamed is the totality guard the two enums here
// share with every other enum in this repository: a walk of the count rather
// than a range over the table of names, because ranging over a table and asking
// it whether it holds what it holds is what let five screens slip into the
// authoring client unmeasured.
func TestEveryVerdictAndForfeitIsNamed(t *testing.T) {
	for value := 0; value < room.VerdictCount; value++ {
		verdict := room.Verdict(value)
		name := verdict.String()
		// A value with no entry in the names table falls through to the numeric
		// form, which is exactly what an unnamed verdict looks like.
		if name == "" || strings.HasPrefix(name, "verdict(") {
			t.Errorf("verdict %d has no name, it stringifies as %q", value, name)
		}
		if want := value != 0; verdict.Over() != want {
			t.Errorf("verdict %q reports Over as %v, want %v", name, verdict.Over(), want)
		}
	}
	if got := room.Verdict(room.VerdictCount).String(); got != "verdict(4)" {
		t.Errorf("a verdict past the count stringifies as %q", got)
	}
	for value := 0; value < room.ForfeitCount; value++ {
		if name := room.Forfeit(value).String(); name == "" || strings.HasPrefix(name, "forfeit(") {
			t.Errorf("forfeit %d has no name, it stringifies as %q", value, name)
		}
	}
	if got := room.Forfeit(room.ForfeitCount).String(); got != "forfeit(3)" {
		t.Errorf("a forfeit past the count stringifies as %q", got)
	}
	// Both routes to a forfeit exist and both are reached by
	// TestAForfeitAndADisconnectAddNothingToTheBattlesOutcomes, which is what
	// keeps ForfeitCount from counting a branch nothing takes.
	if room.ForfeitCount != 3 {
		t.Errorf("there are %d forfeit reasons; the two real ones are both measured, so a third "+
			"needs a case in TestAForfeitAndADisconnectAddNothingToTheBattlesOutcomes",
			room.ForfeitCount)
	}
}
