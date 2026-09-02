package room_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestAWholeMatchIsReproducibleFromOneNumber pins the seed derivation, which is
// the property a match log will rest on the day the room writes one: one number
// in the room's configuration, and every battle of the series derived from it.
//
// The three values are written out because they are arithmetic on a constant and
// can therefore never move for any reason except the derivation changing — which
// is exactly what this test is here to notice. It is not a measurement frozen
// into the suite; there is nothing to measure.
func TestAWholeMatchIsReproducibleFromOneNumber(t *testing.T) {
	const seed = 11
	configuration := config(seed, 3)
	// The first eight bytes of sha256 over the framed pair, which is the framing
	// internal/seed and internal/wire already hash under.
	want := map[int]uint64{
		1: 16297534096469565966,
		2: 1531615077124048477,
		3: 8864668552657113015,
	}
	for index, expected := range want {
		if got := configuration.SeedFor(index); got != expected {
			t.Errorf("battle %d of a match seeded %d derives to %d, want %d",
				index, seed, got, expected)
		}
	}

	// ⚠️ This is the assertion that failed the first implementation and is the
	// whole reason the derivation is a hash of a pair rather than one round of
	// the mixer rng already declares. splitmix64 advances by adding a constant,
	// so `rng.New(Seed + index).Next()` is a function of the **sum** — and
	// battle two of a match seeded 6 was literally battle one of a match seeded
	// 7, two different matches sharing a fight. It is not a near miss to be
	// bounded; it is exact, and it holds for every adjacent pair of seeds.
	for seed := uint64(1); seed <= 64; seed++ {
		if config(seed, 3).SeedFor(2) == config(seed+1, 3).SeedFor(1) {
			t.Errorf("battle 2 of a match seeded %d is battle 1 of a match seeded %d: the pair is not hashed",
				seed, seed+1)
		}
	}
	// And no battle of one series repeats another, which the sum would also have
	// got right and is worth asserting for the same reason a control arm is.
	seen := map[uint64]int{}
	for index := 1; index <= configuration.Battles; index++ {
		derived := configuration.SeedFor(index)
		if before, repeated := seen[derived]; repeated {
			t.Errorf("battles %d and %d of one series derive to the same seed %d", before, index, derived)
		}
		seen[derived] = index
	}
	// Nothing in the derivation reads a clock or draws randomness of its own, so
	// asking twice answers twice the same. Held here as well as by the import
	// ban, because a package-level counter would satisfy the ban and fail this.
	for index := 1; index <= configuration.Battles; index++ {
		if first, second := configuration.SeedFor(index), configuration.SeedFor(index); first != second {
			t.Errorf("battle %d derived to %d and then to %d", index, first, second)
		}
	}
}

// TestBattlesAlternateAndOnlyTheUncancelledOneIsDecidedByTheSeed is the side
// rule, and it is the one design question in the whole of PvP that had a
// measurement waiting for it: which side you get is worth up to **sixty points**
// on a mirror duel, because atb.Queue breaks a tie by the order units joined and
// battle.New enlists in the order of the roster slice it was handed.
//
// So a match fights both ways round. What alternation cannot cancel is a battle
// with no partner, and there is exactly one shape of those: bo1's only battle,
// and the third battle of a 1–1 bo3. Those two are the same problem, so they get
// one rule — the seed picks the side — and the room says so honestly rather than
// dressing a coin as fairness.
//
// ⚠️ The sharpest assertion here is the pair of them: a bo3 and a bo1 seeded
// **identically** disagree about battle one, because in a bo3 it is half of an
// alternating pair and in a bo1 it is the uncancelled battle. That is what "bo1
// is not a special case — it is N = 1" comes to in practice, and it is what a
// bo1 implemented as its own branch would get wrong.
func TestBattlesAlternateAndOnlyTheUncancelledOneIsDecidedByTheSeed(t *testing.T) {
	// Chosen so the two readings of battle one **disagree**, which is what the
	// last assertion in this test needs: the derived low bit for battle one is
	// set here, so a bo1 sends the guest home while a bo3's alternation sends
	// the host. A seed where they happened to agree would satisfy every other
	// line below and quietly measure nothing.
	const seed = 8
	series := config(seed, 3)
	if got := series.HomeFor(1); got != wire.SeatHost {
		t.Errorf("battle 1 of a bo3 is home to %q, want the host: the pairs alternate from the host", got)
	}
	if got := series.HomeFor(2); got != wire.SeatGuest {
		t.Errorf("battle 2 of a bo3 is home to %q, want the guest", got)
	}
	// Battle three is the uncancelled one. For this seed the derived low bit is
	// set, which is the guest — written out for the same reason the seeds above
	// are, rather than asked of the rule.
	if got := series.HomeFor(3); got != wire.SeatGuest {
		t.Errorf("battle 3 of a bo3 seeded %d is home to %q, want the guest from seed %d",
			seed, got, series.SeedFor(3))
	}

	single := config(seed, 1)
	if got := single.HomeFor(1); got != wire.SeatGuest {
		t.Errorf("the one battle of a bo1 seeded %d is home to %q, want the guest from seed %d",
			seed, got, single.SeedFor(1))
	}
	if series.HomeFor(1) == single.HomeFor(1) {
		t.Errorf("a bo3 and a bo1 on the same seed agree about battle 1 (both %q), so the "+
			"uncancelled battle is not being decided by the seed", series.HomeFor(1))
	}

	// The seed genuinely decides it rather than the rule always answering the
	// same thing: over a sweep of seeds a bo1's home has to land on both seats.
	// A rule that returned the host every time would pass every assertion above.
	landed := map[wire.Seat]int{}
	const sweep = 200
	for value := uint64(1); value <= sweep; value++ {
		landed[config(value, 1).HomeFor(1)]++
	}
	for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if landed[seat] == 0 {
			t.Errorf("over %d seeds a bo1 was never home to %q, so the seed is not picking the side",
				sweep, seat)
		}
	}
	// And an alternating series is not seed-dependent at all, which is the other
	// half: the pairs cancel *by construction* rather than on average.
	for value := uint64(1); value <= sweep; value++ {
		alternating := config(value, 3)
		if alternating.HomeFor(1) != wire.SeatHost || alternating.HomeFor(2) != wire.SeatGuest {
			t.Fatalf("a bo3 seeded %d alternates %q then %q rather than host then guest",
				value, alternating.HomeFor(1), alternating.HomeFor(2))
		}
	}
	t.Logf("over %d seeds a bo1's home landed %v", sweep, landed)
}

// TestTheRoomOffersBo1AndBo3AndRefusesABo2ByName is a design decision held as a
// refusal, and the wording matters as much as the refusal: only an even series
// cancels the side, and only an even series has to invent a rule for a 1–1.
//
// The aggregate surviving-health tie-break an earlier draft of the design record
// proposed is **dropped**, so no invented metric ships anywhere in this package
// — which is the thing this refusal protects. A third battle needs no
// justification at all.
func TestTheRoomOffersBo1AndBo3AndRefusesABo2ByName(t *testing.T) {
	for _, battles := range []int{1, 3} {
		if err := config(11, battles).Validate(); err != nil {
			t.Errorf("a bo%d is refused: %v", battles, err)
		}
	}
	for _, one := range []struct {
		battles int
		says    string
	}{
		{battles: 2, says: "1–1"},
		{battles: 4, says: "2–2"},
		{battles: 0, says: "not one the room offers"},
		{battles: 5, says: "not one the room offers"},
		{battles: -1, says: "not one the room offers"},
	} {
		err := config(11, one.battles).Validate()
		if err == nil {
			t.Errorf("a series of %d battles was accepted", one.battles)
			continue
		}
		if !strings.Contains(err.Error(), one.says) {
			t.Errorf("a series of %d battles is refused with %q, which does not say %q",
				one.battles, err, one.says)
		}
	}
}

// TestARoomWithoutAWorkableSetupRefusesToOpen is the rest of the configuration,
// checked at New so that a room which cannot run a match fails before anybody
// joins it rather than in the middle of a battle.
func TestARoomWithoutAWorkableSetupRefusesToOpen(t *testing.T) {
	dependencies := deps(t)
	for _, one := range []struct {
		name  string
		spoil func(*room.Config)
		says  string
	}{
		{
			name:  "a format nobody plays",
			spoil: func(c *room.Config) { c.Format = wire.Format(4) },
			says:  "not a format",
		},
		{
			name:  "no allowance at all",
			spoil: func(c *room.Config) { c.Allowance = 0 },
			says:  "no turn to take",
		},
		{
			name:  "a turn cap of nothing",
			spoil: func(c *room.Config) { c.TurnCap = 0 },
			says:  "before it starts",
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			configuration := config(11, 3)
			one.spoil(&configuration)
			_, err := room.New(configuration, dependencies)
			if err == nil {
				t.Fatal("the room opened")
			}
			if !strings.Contains(err.Error(), one.says) {
				t.Errorf("the refusal reads %q, want it to say %q", err, one.says)
			}
		})
	}
	// And the cast book, without which no squad resolves — which would surface
	// as a squad refused at the gate for a reason that was the host's fault.
	if _, err := room.New(config(11, 3), room.Deps{Books: dependencies.Books}); err == nil {
		t.Error("a room opened with no cast book")
	}
}
