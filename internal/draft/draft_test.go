package draft_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// shippedPool is the pool a real draft runs on: the authored cast minus the
// characters held back.
func shippedPool(t *testing.T) draft.Pool {
	t.Helper()
	return draft.NewPool(shippedCast(t))
}

// fixtureCast is n characters that are nothing but distinct ids, for the cases
// where the pool's *size* or its *contents* are the whole subject and who is in
// it is not. Only ID is set, which is the whole of what NewPool reads.
func fixtureCast(n int) []cast.Character {
	all := make([]cast.Character, 0, n)
	for at := range n {
		all = append(all, cast.Character{ID: fmt.Sprintf("fixture.%02d", at)})
	}
	return all
}

// poolOf is a pool of exactly n characters.
func poolOf(n int) draft.Pool { return draft.NewPool(fixtureCast(n)) }

// characterNamed is one character out of a cast by id, and fails rather than
// answering a zero value: every id a test looks up here came out of a pool built
// from the same list.
func characterNamed(t *testing.T, all []cast.Character, id string) cast.Character {
	t.Helper()
	at := slices.IndexFunc(all, func(character cast.Character) bool { return character.ID == id })
	if at < 0 {
		t.Fatalf("no character called %q is in this cast", id)
	}
	return all[at]
}

// seatIndex is a seat's position in Picks' array, which is the order a room
// hands seats out: the host, then the guest.
func seatIndex(t *testing.T, seat wire.Seat) int {
	t.Helper()
	switch seat {
	case wire.SeatHost:
		return 0
	case wire.SeatGuest:
		return 1
	}
	t.Fatalf("%q is not a seat", seat)
	return -1
}

// legalLoadout is *a* legal loadout for a character at the cap: a form named
// explicitly, the first four skills that form knows and the first trait.
//
// ⚠️ **It is legal and it is not a design decision**, and the difference is
// CLAUDE.md's own: "the first four declared" is the order the file happens to
// list rather than anybody's choice, which is the whole reason builds.json
// exists. It is used here because these tests are about the state machine and
// not about the kit — and a *designed* loadout going through the same call is
// measured separately, over every shipped build, in
// TestEveryShippedBuildIsALegalDraftLoadout.
func legalLoadout(t *testing.T, character cast.Character) (string, []string, []string) {
	t.Helper()
	arms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("the forms %s reaches at level %d: %v", character.ID, progression.LevelCap, err)
	}
	form := arms[0].Name
	skills := character.SkillsAt(progression.LevelCap, form)
	if len(skills) == 0 {
		t.Fatalf("%s knows no skills at level %d as %s, so no loadout of it is legal and this "+
			"helper cannot build one", character.ID, progression.LevelCap, form)
	}
	passives := character.PassivesAt(progression.LevelCap, form)
	return form,
		skills[:min(cast.SkillSlots, len(skills))],
		passives[:min(cast.TraitSlots, len(passives))]
}

// playOut drives a whole draft with the decisions a player would make, and
// reports the sequence of decisions it was asked for as "seat step" rows.
//
// bans is read in the order the ban decisions come — a true spends the slot on
// the first candidate, a false skips it — and every pick takes the first
// candidate, because which character is taken cannot change the sequence.
//
// ⚠️ It asserts that a decision naming a character always has one to name, which
// is step 1's exhaustion proof held from the state machine's side rather than
// restated: a draft New allowed cannot run dry, so an empty candidate list here
// is that claim failing.
func playOut(t *testing.T, drafting *draft.Draft, all []cast.Character, bans []bool) []string {
	t.Helper()
	sequence := []string{}
	banAt := 0
	for {
		seat, step, due := drafting.Turn()
		if !due {
			return sequence
		}
		sequence = append(sequence, fmt.Sprintf("%s %s", seat, step))
		if len(sequence) > 4*(draft.PicksPerSide(wire.Format5v5)+draft.BansPerSide(wire.Format5v5)) {
			t.Fatalf("the draft asked for %d decisions and no format has that many, so it is "+
				"not advancing: last was %q", len(sequence), sequence[len(sequence)-1])
		}
		switch step {
		case draft.StepBan:
			spend := banAt < len(bans) && bans[banAt]
			banAt++
			if !spend {
				if err := drafting.SkipBan(seat); err != nil {
					t.Fatalf("%s skips ban %d: %v", seat, banAt, err)
				}
				continue
			}
			if err := drafting.Ban(seat, firstCandidate(t, drafting).ID); err != nil {
				t.Fatalf("%s spends ban %d: %v", seat, banAt, err)
			}
		case draft.StepPick:
			if err := drafting.Pick(seat, firstCandidate(t, drafting).ID); err != nil {
				t.Fatalf("%s picks: %v", seat, err)
			}
		case draft.StepLoadout:
			side := drafting.Picks()[seatIndex(t, seat)]
			open := side[len(side)-1]
			form, skills, passives := legalLoadout(t, characterNamed(t, all, open.Character))
			if err := drafting.Loadout(seat, form, skills, passives); err != nil {
				t.Fatalf("%s's loadout for %s as %s: %v", seat, open.Character, form, err)
			}
		default:
			t.Fatalf("the draft asked for a %q, which is not a decision a seat can make", step)
		}
	}
}

// firstCandidate is what a decision naming a character takes, and the assertion
// that there was one to take.
func firstCandidate(t *testing.T, drafting *draft.Draft) cast.Character {
	t.Helper()
	candidates := drafting.Candidates()
	if len(candidates) == 0 {
		seat, step, _ := drafting.Turn()
		t.Fatalf("%s is due to %s and the pool offers nothing: a draft New allowed cannot run "+
			"out of characters (see the package comment's proof), so this is that claim failing",
			seat, step)
	}
	return candidates[0]
}

// spendEvery is a ban pattern where every slot is spent, which is the worst case
// the arithmetic is measured against and the only one where the last pick of a
// 5v5 is forced.
func spendEvery(format wire.Format) []bool {
	out := make([]bool, 2*draft.BansPerSide(format))
	for at := range out {
		out[at] = true
	}
	return out
}

// TestANewDraftThatCouldNotFinishIsRefused is "a draft that cannot finish fails
// when it is set up", which is the arrangement this whole package rests on:
// every refusal below is one a room can take before anybody has decided
// anything, and it is what buys the absence of a pool-exhaustion rule at run
// time.
//
// ⚠️ The last two cases are not tidiness. Fits counts Pool.Len(), and both a
// duplicated id and an id-less entry make the pool hold fewer characters than a
// decision can name — so the count the exhaustion proof is built on would be an
// overstatement, and a draft could run dry after all.
func TestANewDraftThatCouldNotFinishIsRefused(t *testing.T) {
	shipped := shippedPool(t)
	duplicated := draft.NewPool(append(fixtureCast(16), cast.Character{ID: "fixture.00"}))
	nameless := draft.NewPool(append(fixtureCast(16), cast.Character{}))

	for _, one := range []struct {
		what   string
		config draft.Config
		refuse string
	}{
		{"the shipped pool at 3v3",
			draft.Config{Format: wire.Format3v3, Pool: shipped, First: wire.SeatHost}, ""},
		{"the shipped pool at 5v5, which fits with nothing to spare",
			draft.Config{Format: wire.Format5v5, Pool: shipped, First: wire.SeatGuest}, ""},
		{"a pool one short of a 5v5",
			draft.Config{Format: wire.Format5v5, Pool: poolOf(15), First: wire.SeatHost},
			"a 5v5 draft takes 10 picks and 6 bans out of one shared pool, which is 16 " +
				"characters, and the pool holds 15: it is short by one character"},
		{"a format the game does not offer",
			draft.Config{Format: wire.Format(4), Pool: shipped, First: wire.SeatHost},
			"a draft of 4v4 is not a format this game offers"},
		{"no seat going first",
			draft.Config{Format: wire.Format3v3, Pool: shipped},
			"a draft cannot begin with \"\", which is not one of the two seats a room hands out"},
		{"a seat a room does not hand out going first",
			draft.Config{Format: wire.Format3v3, Pool: shipped, First: wire.Seat("spectator")},
			"a draft cannot begin with \"spectator\", which is not one of the two seats a room " +
				"hands out"},
		{"a pool holding one character twice",
			draft.Config{Format: wire.Format3v3, Pool: duplicated, First: wire.SeatHost},
			"two characters in this draft's pool are called \"fixture.00\", so taking one would " +
				"take both, and the pool is one character smaller than it counts"},
		{"a pool holding a character with no id",
			draft.Config{Format: wire.Format3v3, Pool: nameless, First: wire.SeatHost},
			"the character at position 16 of this draft's pool has no id, so no ban and no pick " +
				"can name it, and the pool is one character smaller than it counts"},
	} {
		drafting, err := draft.New(one.config)
		switch {
		case one.refuse == "" && err != nil:
			t.Errorf("%s is a draft that can finish and was refused: %v", one.what, err)
		case one.refuse == "":
			if _, _, due := drafting.Turn(); !due {
				t.Errorf("%s was set up and has no first decision to make", one.what)
			}
		case err == nil:
			t.Errorf("%s was allowed, and it should be refused with %q", one.what, one.refuse)
		case err.Error() != one.refuse:
			t.Errorf("%s refuses with\n  %q\nand the wording is\n  %q", one.what, err.Error(), one.refuse)
		}
	}
}

// TestTheSettledSequenceIsEveryBanThenEveryPick is TODO.md § "Ban and pick" (f)
// written out, and it is written as a literal on purpose: the order is a
// **decision** — not alternating ban-and-pick, not a snake — so there is nothing
// for a test to derive it from, and a sequence computed the way the code
// computes it would agree with any order at all.
//
// The loadout after each pick is the other half of decision (a): a pick is two
// decisions, the character and then the kit, and the character leaves the pool
// at the first of them.
//
// ⚠️ A failure here is a design change and belongs in TODO.md before it belongs
// in the code.
func TestTheSettledSequenceIsEveryBanThenEveryPick(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}

	want := []string{
		"host ban", "guest ban", "host ban", "guest ban",
		"host pick", "host loadout", "guest pick", "guest loadout",
		"host pick", "host loadout", "guest pick", "guest loadout",
		"host pick", "host loadout", "guest pick", "guest loadout",
	}
	got := playOut(t, drafting, all, spendEvery(wire.Format3v3))
	if !slices.Equal(got, want) {
		t.Errorf("a 3v3 draft from the host asks for\n  %v\nand the settled sequence is\n  %v",
			got, want)
	}
	if !drafting.Picked() {
		t.Error("the whole sequence was played and the picking is not over")
	}
	if drafting.Cancelled() {
		t.Error("a draft that was played out reports itself cancelled")
	}
	// ⚠️ The sequence Turn asks for ends at the picking, and the draft is **not**
	// Done there: what is open is the arrange phase, which Turn deliberately does
	// not answer for. This is the assertion that keeps the two names apart.
	if drafting.Done() {
		t.Error("the picking is over and the draft calls itself done, with neither side arranged")
	}
	if !drafting.Arranging() {
		t.Error("the picking is over and the arrange phase is not open")
	}
	if got := drafting.AwaitingArrangement(); len(got) != seatCount() {
		t.Errorf("both sides have still to arrange and the draft is waiting on %v", got)
	}
	t.Logf("%d decisions: %s", len(got), strings.Join(got, ", "))
}

// TestBothStagesAlternateFromWhoeverGoesFirst holds the properties of the
// sequence over every format and either seat first, which is the half a single
// literal cannot cover.
//
// Four claims, and each is one clause of decision (f): every ban is taken before
// any pick; both stages alternate from Config.First; every pick is immediately
// followed by its own seat's loadout; and each side ends with exactly the picks
// its format fields.
func TestBothStagesAlternateFromWhoeverGoesFirst(t *testing.T) {
	all := shippedCast(t)
	for _, format := range []wire.Format{wire.Format3v3, wire.Format5v5} {
		for _, first := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
			drafting, err := draft.New(draft.Config{
				Format: format, Pool: draft.NewPool(all), First: first,
			})
			if err != nil {
				t.Fatalf("set up a %s draft from the %s: %v", format, first, err)
			}
			sequence := playOut(t, drafting, all, spendEvery(format))

			bans, picks, loadouts, lastBan, firstPick := 0, 0, 0, -1, -1
			for at, row := range sequence {
				seat, step, found := strings.Cut(row, " ")
				if !found {
					t.Fatalf("%q is not a decision", row)
				}
				switch draft.Step(step) {
				case draft.StepBan:
					if want := alternating(first, bans); wire.Seat(seat) != want {
						t.Errorf("%s from the %s: ban %d is %s's and alternation from the %s "+
							"makes it %s's", format, first, bans+1, seat, first, want)
					}
					bans++
					lastBan = at
				case draft.StepPick:
					if want := alternating(first, picks); wire.Seat(seat) != want {
						t.Errorf("%s from the %s: pick %d is %s's and alternation from the %s "+
							"makes it %s's", format, first, picks+1, seat, first, want)
					}
					if firstPick < 0 {
						firstPick = at
					}
					picks++
					// The loadout belongs to whoever just picked and comes next,
					// with nothing in between: the character has left the pool
					// and the pick is not finished.
					if at+1 >= len(sequence) || sequence[at+1] != seat+" loadout" {
						t.Errorf("%s from the %s: %s's pick at %d is not followed by its own "+
							"loadout", format, first, seat, at)
					}
				case draft.StepLoadout:
					loadouts++
				}
			}
			if firstPick >= 0 && lastBan > firstPick {
				t.Errorf("%s from the %s: a ban at %d comes after the first pick at %d, and "+
					"every ban is taken before any pick", format, first, lastBan, firstPick)
			}
			if want := 2 * draft.BansPerSide(format); bans != want {
				t.Errorf("%s asked for %d ban decisions and two sides of %d bans is %d",
					format, bans, draft.BansPerSide(format), want)
			}
			if want := 2 * draft.PicksPerSide(format); picks != want || loadouts != want {
				t.Errorf("%s asked for %d picks and %d loadouts, and two sides of %d picks is %d",
					format, picks, loadouts, draft.PicksPerSide(format), want)
			}
			for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
				side := drafting.Picks()[seatIndex(t, seat)]
				if len(side) != draft.PicksPerSide(format) {
					t.Errorf("%s from the %s: the %s ends with %d picks and a %s fields %d",
						format, first, seat, len(side), format, draft.PicksPerSide(format))
				}
			}
			t.Logf("%s from the %s: %d decisions", format, first, len(sequence))
		}
	}
}

// alternating is whose n-th decision of a stage it is, counting from the seat
// that goes first. It is the claim rather than a copy of the code: two seats
// taking turns is what "alternating from the host" means.
func alternating(first wire.Seat, n int) wire.Seat {
	if n%2 == 0 {
		return first
	}
	if first == wire.SeatHost {
		return wire.SeatGuest
	}
	return wire.SeatHost
}

// TestEveryRefusalThisStateMachineOwes pins each refusal by value, which is the
// same terms internal/draft's arithmetic refusal and the command line's are held
// under: a refusal somebody reads is a contract, and a message that says "no"
// without saying what cannot happen is one nobody can act on.
//
// Each case sets a draft up to the state where the refusal is the right answer,
// makes the wrong decision, and compares the whole sentence.
func TestEveryRefusalThisStateMachineOwes(t *testing.T) {
	all := shippedCast(t)
	// A 3v3 from the host, so "the host bans first" is the state every case
	// starts from and each one walks as far as it needs.
	fresh := func(t *testing.T) *draft.Draft {
		t.Helper()
		drafting, err := draft.New(draft.Config{
			Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
		})
		if err != nil {
			t.Fatalf("set up a 3v3 draft: %v", err)
		}
		return drafting
	}
	// intoThePickingStage skips every ban, which is the cheapest way to the
	// picking stage and exercises the skip on the way.
	intoThePickingStage := func(t *testing.T, drafting *draft.Draft) {
		t.Helper()
		for range 2 * draft.BansPerSide(wire.Format3v3) {
			seat, _, _ := drafting.Turn()
			if err := drafting.SkipBan(seat); err != nil {
				t.Fatalf("%s skips a ban: %v", seat, err)
			}
		}
	}
	first := func(t *testing.T, drafting *draft.Draft) string {
		t.Helper()
		return firstCandidate(t, drafting).ID
	}

	for _, one := range []struct {
		what   string
		refuse func(t *testing.T, drafting *draft.Draft) error
		want   string
	}{
		{"the seat that is not on turn bans",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Ban(wire.SeatGuest, first(t, drafting))
			},
			"it is host's turn to ban, so \"guest\" cannot act out of turn"},
		{"a seat a room does not hand out bans",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Ban(wire.Seat("spectator"), first(t, drafting))
			},
			"\"spectator\" is not one of the two seats a room hands out, so it has no decision " +
				"to make in this draft"},
		{"the seat on turn picks during the banning stage",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Pick(wire.SeatHost, first(t, drafting))
			},
			"host is due to ban and not to pick: every ban is taken before any pick, so that a " +
				"ban can still deny one (TODO.md § \"Ban and pick\" (f)), and 4 of the 4 ban " +
				"slots are still open"},
		{"a seat bans once the picking has started",
			func(t *testing.T, drafting *draft.Draft) error {
				intoThePickingStage(t, drafting)
				return drafting.Ban(wire.SeatHost, first(t, drafting))
			},
			"the banning stage is over, so host cannot ban: a ban is only worth spending while " +
				"it can still deny a pick, and picking has begun"},
		{"a character that is in no pool is banned",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Ban(wire.SeatHost, "nobody.at.all")
			},
			"\"nobody.at.all\" is not in this draft's pool, so a ban cannot name it: the pool " +
				"is the cast minus every character held back, and it is fixed for the whole of " +
				"a draft"},
		{"a character held back from every draft is picked",
			func(t *testing.T, drafting *draft.Draft) error {
				intoThePickingStage(t, drafting)
				return drafting.Pick(wire.SeatHost, heldBack(t, all))
			},
			"\"naruto.naruto\" is not in this draft's pool, so a pick cannot name it: the pool " +
				"is the cast minus every character held back, and it is fixed for the whole of " +
				"a draft"},
		{"a banned character is picked",
			func(t *testing.T, drafting *draft.Draft) error {
				banned := first(t, drafting)
				if err := drafting.Ban(wire.SeatHost, banned); err != nil {
					t.Fatalf("the host bans %s: %v", banned, err)
				}
				for range 2*draft.BansPerSide(wire.Format3v3) - 1 {
					seat, _, _ := drafting.Turn()
					if err := drafting.SkipBan(seat); err != nil {
						t.Fatalf("%s skips a ban: %v", seat, err)
					}
				}
				return drafting.Pick(wire.SeatHost, banned)
			},
			"\"pokemon.bulbasaur\" is out of this draft already: host took it with a ban"},
		{"a character the other side has picked is picked",
			func(t *testing.T, drafting *draft.Draft) error {
				intoThePickingStage(t, drafting)
				taken := first(t, drafting)
				if err := drafting.Pick(wire.SeatHost, taken); err != nil {
					t.Fatalf("the host picks %s: %v", taken, err)
				}
				form, skills, passives := legalLoadout(t, characterNamed(t, all, taken))
				if err := drafting.Loadout(wire.SeatHost, form, skills, passives); err != nil {
					t.Fatalf("the host's loadout for %s: %v", taken, err)
				}
				return drafting.Pick(wire.SeatGuest, taken)
			},
			"\"pokemon.bulbasaur\" is out of this draft already: host took it with a pick"},
		{"a loadout arrives with no pick open",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Loadout(wire.SeatHost, "", []string{"strike"}, nil)
			},
			"no pick of host's is waiting for a loadout: host is due to ban"},
		{"a second loadout for one pick",
			func(t *testing.T, drafting *draft.Draft) error {
				intoThePickingStage(t, drafting)
				taken := first(t, drafting)
				if err := drafting.Pick(wire.SeatHost, taken); err != nil {
					t.Fatalf("the host picks %s: %v", taken, err)
				}
				form, skills, passives := legalLoadout(t, characterNamed(t, all, taken))
				if err := drafting.Loadout(wire.SeatHost, form, skills, passives); err != nil {
					t.Fatalf("the host's loadout for %s: %v", taken, err)
				}
				return drafting.Loadout(wire.SeatHost, form, skills, passives)
			},
			"host's pick of pokemon.bulbasaur already has its loadout, and a pick's loadout is " +
				"chosen once: it is guest's turn to pick now"},
		{"anything at all is decided while a loadout is owed",
			func(t *testing.T, drafting *draft.Draft) error {
				intoThePickingStage(t, drafting)
				taken := first(t, drafting)
				if err := drafting.Pick(wire.SeatHost, taken); err != nil {
					t.Fatalf("the host picks %s: %v", taken, err)
				}
				return drafting.Pick(wire.SeatHost, "pokemon.mew")
			},
			"host has just picked pokemon.bulbasaur and owes it a loadout, so nothing else can " +
				"be decided until the form, the skills and the trait are chosen"},
		// ⚠️ These two used to be one case. Playing the picking out no longer
		// finishes a draft — the arrange phase is open at that point — so the
		// state the old sentence described is now reached only by arranging both
		// sides, and the state the old case *drove* has a sentence of its own.
		{"a decision arrives once the picking is over and the arranging is open",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				return drafting.Pick(wire.SeatHost, "pokemon.mew")
			},
			"the picking is over — every ban is spent or skipped and every pick has its loadout " +
				"— and what is open now is the arrangement, so \"host\" cannot pick: the two " +
				"sides arrange at once, which Arrange takes and Turn does not answer for"},
		{"a decision arrives after the whole draft is done",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				arrangeBothSides(t, drafting)
				return drafting.Pick(wire.SeatHost, "pokemon.mew")
			},
			"this draft is finished — every ban is spent or skipped, every pick has its loadout " +
				"and both sides have arranged — so \"host\" cannot pick"},
		{"an arrangement arrives before the picking is over",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.Arrange(wire.SeatHost, formationCells(3))
			},
			"the draft is waiting on host to ban, so \"host\" cannot arrange yet: both sides " +
				"arrange once every ban is spent and every pick has its loadout, and not before"},
		{"a seat arranges twice",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				arrangeSide(t, drafting, wire.SeatHost)
				return drafting.Arrange(wire.SeatHost, formationCells(3))
			},
			"host has already arranged, and an arrangement is made once: this draft is waiting " +
				"on guest"},
		{"an arrangement one cell short of the side's picks",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				return drafting.Arrange(wire.SeatHost, formationCells(2))
			},
			"host drafted 3 units and this arrangement names 2 cells: a side arranges its whole " +
				"squad in one call, and slots[i] is the cell for its i-th pick"},
		{"a seat a room does not hand out arranges",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				return drafting.Arrange(wire.Seat("spectator"), formationCells(3))
			},
			"\"spectator\" is not one of the two seats a room hands out, so it has no " +
				"arrangement to make in this draft"},
		{"a third arrangement once both are in",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				arrangeBothSides(t, drafting)
				return drafting.Arrange(wire.SeatHost, formationCells(3))
			},
			"both sides have arranged and this draft is finished, so \"host\" cannot arrange " +
				"again"},
		{"an arrangement arrives after the draft was cancelled",
			func(t *testing.T, drafting *draft.Draft) error {
				if err := drafting.TimedOut(wire.SeatHost); err != nil {
					t.Fatalf("the host's allowance runs out: %v", err)
				}
				return drafting.Arrange(wire.SeatHost, formationCells(3))
			},
			"this draft was cancelled when host ran out of time, so \"host\" cannot arrange: a " +
				"draft that runs out of time is not resumed, it is played again from a new " +
				"room code"},
		{"a timeout is reported for a seat that has already arranged",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				arrangeSide(t, drafting, wire.SeatHost)
				return drafting.TimedOut(wire.SeatHost)
			},
			"host has already arranged, so it has no allowance left to run out: this draft is " +
				"waiting on guest"},
		{"a timeout is reported for a seat a room does not hand out while the arranging is open",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				return drafting.TimedOut(wire.Seat("spectator"))
			},
			"\"spectator\" is not one of the two seats a room hands out, so it has no " +
				"arrangement to be waited on and that timeout cancels nothing"},
		{"a decision arrives after the draft was cancelled",
			func(t *testing.T, drafting *draft.Draft) error {
				if err := drafting.TimedOut(wire.SeatHost); err != nil {
					t.Fatalf("the host's allowance runs out: %v", err)
				}
				return drafting.Ban(wire.SeatGuest, "pokemon.mew")
			},
			"this draft was cancelled when host ran out of time, so \"guest\" cannot ban: a " +
				"draft that runs out of time is not resumed, it is played again from a new " +
				"room code"},
		{"a timeout is reported for a seat nobody is waiting on",
			func(t *testing.T, drafting *draft.Draft) error {
				return drafting.TimedOut(wire.SeatGuest)
			},
			"the draft is waiting on host to ban and not on \"guest\", so that timeout " +
				"cancels nothing"},
		{"a second timeout on an already cancelled draft",
			func(t *testing.T, drafting *draft.Draft) error {
				if err := drafting.TimedOut(wire.SeatHost); err != nil {
					t.Fatalf("the host's allowance runs out: %v", err)
				}
				return drafting.TimedOut(wire.SeatHost)
			},
			"this draft was already cancelled when host ran out of time, so a second timeout " +
				"has nothing to end"},
		{"a timeout on a draft that is finished",
			func(t *testing.T, drafting *draft.Draft) error {
				playOut(t, drafting, all, spendEvery(wire.Format3v3))
				arrangeBothSides(t, drafting)
				return drafting.TimedOut(wire.SeatHost)
			},
			"this draft is finished, so there is no open decision for an allowance to run out on"},
	} {
		err := one.refuse(t, fresh(t))
		switch {
		case err == nil:
			t.Errorf("%s was allowed, and it should be refused with\n  %q", one.what, one.want)
		case err.Error() != one.want:
			t.Errorf("%s refuses with\n  %q\nand the wording is\n  %q", one.what, err.Error(), one.want)
		}
	}
}

// heldBack is a character the shipped cast holds back, which is what a draft
// must refuse to seat. It is looked up rather than named so this stays true as
// the cast moves, and it fails when nothing is held back — because then the
// refusal it feeds is unexercised.
func heldBack(t *testing.T, all []cast.Character) string {
	t.Helper()
	for _, character := range all {
		if character.Hidden {
			return character.ID
		}
	}
	t.Fatal("no shipped character is held back, so the refusal this feeds measures nothing")
	return ""
}

// TestASkippedBanLeavesEveryCharacterInThePool is the optionality of a ban held
// as the thing that actually matters about it: a skip takes **nobody** out.
//
// ⚠️ That direction is the whole reason a shortfall is a refused configuration
// and never a runtime failure — optionality can only leave the pool fuller than
// Fits measured — so a skip that quietly spent a character would break step 1's
// proof rather than merely miscount.
func TestASkippedBanLeavesEveryCharacterInThePool(t *testing.T) {
	all := shippedCast(t)
	pool := draft.NewPool(all)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: pool, First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	before := candidateIDs(drafting)
	if len(before) != pool.Len() {
		t.Fatalf("the first ban chooses from %d of a pool of %d, so this test's own premise "+
			"is wrong", len(before), pool.Len())
	}

	for slot := range 2 * draft.BansPerSide(wire.Format3v3) {
		seat, _, _ := drafting.Turn()
		if err := drafting.SkipBan(seat); err != nil {
			t.Fatalf("%s skips ban %d: %v", seat, slot+1, err)
		}
		after := candidateIDs(drafting)
		if !slices.Equal(after, before) {
			t.Fatalf("after %d skipped bans the pool offers %v and every character was still "+
				"in it: a skipped ban names nobody, so it takes nobody out", slot+1, after)
		}
	}
	// The pool the draft was built from is untouched as well, which is the other
	// half of "fixed for the whole of a draft".
	if pool.Len() != len(before) {
		t.Errorf("the pool itself now seats %d of the %d it was built with", pool.Len(), len(before))
	}
	t.Logf("%d skipped bans left all %d characters in the pool",
		2*draft.BansPerSide(wire.Format3v3), len(before))
}

// candidateIDs is the open decision's candidates by id, which is the shape a
// comparison wants.
func candidateIDs(drafting *draft.Draft) []string {
	out := []string{}
	for _, character := range drafting.Candidates() {
		out = append(out, character.ID)
	}
	return out
}

// TestCandidatesAreThePoolMinusWhatEitherSideTook is what Candidates is for, and
// it walks a whole draft asserting the count falls by exactly one per character
// spent — a ban, a pick, and nothing for a loadout.
//
// *Sees:* a Candidates that ignores what has been taken, one that drops a
// character a skip did not take, one that answers the same list twice.
// *Cannot see:* what a screen does with the list, which is step 5.
func TestCandidatesAreThePoolMinusWhatEitherSideTook(t *testing.T) {
	all := shippedCast(t)
	pool := draft.NewPool(all)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: pool, First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}

	gone := []string{}
	for {
		seat, step, due := drafting.Turn()
		if !due {
			break
		}
		if step == draft.StepLoadout {
			// A loadout chooses a form and a kit rather than a character, so it
			// has no candidates at all — a different answer from an empty list.
			if candidates := drafting.Candidates(); candidates != nil {
				t.Errorf("a loadout offers %d characters to choose from", len(candidates))
			}
			side := drafting.Picks()[seatIndex(t, seat)]
			open := side[len(side)-1]
			form, skills, passives := legalLoadout(t, characterNamed(t, all, open.Character))
			if err := drafting.Loadout(seat, form, skills, passives); err != nil {
				t.Fatalf("%s's loadout for %s: %v", seat, open.Character, err)
			}
			continue
		}
		offered := candidateIDs(drafting)
		if want := pool.Len() - len(gone); len(offered) != want {
			t.Fatalf("%s is due to %s with %d characters spent, and is offered %d of a pool "+
				"of %d — it should be %d", seat, step, len(gone), len(offered), pool.Len(), want)
		}
		for _, spent := range gone {
			if slices.Contains(offered, spent) {
				t.Errorf("%s is offered %s, which is already out of the draft", seat, spent)
			}
		}
		taking := offered[len(offered)-1]
		if step == draft.StepBan {
			if err := drafting.Ban(seat, taking); err != nil {
				t.Fatalf("%s bans %s: %v", seat, taking, err)
			}
		} else if err := drafting.Pick(seat, taking); err != nil {
			t.Fatalf("%s picks %s: %v", seat, taking, err)
		}
		gone = append(gone, taking)
	}
	if want := 2 * (draft.BansPerSide(wire.Format3v3) + draft.PicksPerSide(wire.Format3v3)); len(gone) != want {
		t.Errorf("a 3v3 with every ban spent takes %d characters out of the pool and this one "+
			"took %d", want, len(gone))
	}
}

// TestTheLastPickOfAFiveASideChoosesFromSlackPlusOne is a **behaviour** test of
// what TODO.md's slack note states as arithmetic: with every ban spent the final
// picker of a 5v5 sees exactly `slack + 1` candidates.
//
// ⚠️ It is why Candidates exists at all. A screen drawing a list of one has to be
// able to say the pick is not a decision, and it cannot know that without asking.
//
// ⚠️ **It was TestTheLastPickOfAFiveASideChoosesFromOne and refused to run at any
// other slack**, which made it a test that reddened on a content change while its
// own assertion — `slack + 1`, written that way from the start — never needed to.
// `pokemon.pichu` took slack from nought to one on 2026-09-05 and the premise
// fired where nothing was wrong. The figure that *is* worth pinning is the slack
// itself, because that is what decides whether the last pick is a decision at
// all, and it is pinned once, in TestFiveASideFitsTheShippedCastWithRoomToChoose.
// Holding it in two places is what made this one churn.
func TestTheLastPickOfAFiveASideChoosesFromSlackPlusOne(t *testing.T) {
	all := shippedCast(t)
	pool := draft.NewPool(all)
	slack := draft.Slack(pool.Len(), wire.Format5v5)
	// The premise, held rather than assumed: a pool that cannot seat the format
	// never reaches a last pick, so the walk below would prove nothing.
	if slack < 0 {
		t.Fatalf("a pool of %d cannot seat a 5v5 at all (slack %d), so there is no last pick "+
			"to measure — see TODO.md § \"Ban and pick\"", pool.Len(), slack)
	}
	drafting, err := draft.New(draft.Config{
		Format: wire.Format5v5, Pool: pool, First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 5v5 draft: %v", err)
	}

	atLastPick, picks := -1, 0
	for {
		seat, step, due := drafting.Turn()
		if !due {
			break
		}
		switch step {
		case draft.StepBan:
			if err := drafting.Ban(seat, firstCandidate(t, drafting).ID); err != nil {
				t.Fatalf("%s bans: %v", seat, err)
			}
		case draft.StepPick:
			picks++
			offered := len(drafting.Candidates())
			if picks == 2*draft.PicksPerSide(wire.Format5v5) {
				atLastPick = offered
			}
			if err := drafting.Pick(seat, firstCandidate(t, drafting).ID); err != nil {
				t.Fatalf("%s picks: %v", seat, err)
			}
		case draft.StepLoadout:
			side := drafting.Picks()[seatIndex(t, seat)]
			open := side[len(side)-1]
			form, skills, passives := legalLoadout(t, characterNamed(t, all, open.Character))
			if err := drafting.Loadout(seat, form, skills, passives); err != nil {
				t.Fatalf("%s's loadout for %s: %v", seat, open.Character, err)
			}
		}
	}
	if want := slack + 1; atLastPick != want {
		t.Errorf("the last pick of a 5v5 out of a pool of %d chose from %d, and slack %d makes "+
			"it %d", pool.Len(), atLastPick, slack, want)
	}
	t.Logf("pool %d, slack %d: the last of %d picks chose from %d",
		pool.Len(), slack, picks, atLastPick)
}

// TestAForkedLineMustNameItsForm is the fork refused, and it is measured on
// **pokemon.poliwag** rather than on a synthetic character on purpose: the fork
// is live shipped data in the draftable pool, so a fixture fork would be a test
// asserting a branch the shipped cast also has — and only one of the two would
// be measured.
//
// At the cap poliwag reaches Poliwrath and Politoed both, so an unnamed form has
// no answer (progression.Line.StageAt refuses to choose between two arms) and
// the placement that came out of it would not be fieldable at all. The refusal
// is progression's own, behind a lead-in naming the pick.
func TestAForkedLineMustNameItsForm(t *testing.T) {
	all := shippedCast(t)
	const forked = "pokemon.poliwag"
	character := characterNamed(t, all, forked)
	arms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("the forms %s reaches at the cap: %v", forked, err)
	}
	// The premise. If poliwag stopped forking at the cap, this test would pass
	// against a character with one arm and measure nothing at all.
	if len(arms) < 2 {
		t.Fatalf("%s reaches %v at level %d, and this test is about a line that FORKS there",
			forked, progression.StageNames(arms), progression.LevelCap)
	}

	for _, one := range []struct {
		what string
		form string
		want string
	}{
		{"no form named at all", progression.Furthest,
			"host's pick of pokemon.poliwag at level 60: level 60 reaches [Poliwrath Politoed], " +
				"which are alternatives: name the one being fielded"},
		{"a form no stage of the line answers to", "Blastoise",
			"host's pick of pokemon.poliwag at level 60: no stage of this line is called " +
				"\"Blastoise\"; it has [Poliwag Poliwhirl Poliwrath Politoed]"},
	} {
		drafting := draftAtItsFirstPick(t, all)
		if err := drafting.Pick(wire.SeatHost, forked); err != nil {
			t.Fatalf("the host picks %s: %v", forked, err)
		}
		skills := character.SkillsAt(progression.LevelCap, arms[0].Name)
		err := drafting.Loadout(wire.SeatHost, one.form, skills[:cast.SkillSlots], nil)
		switch {
		case err == nil:
			t.Errorf("a loadout for a forked line with %s was accepted, and it should be "+
				"refused with\n  %q", one.what, one.want)
		case err.Error() != one.want:
			t.Errorf("a loadout for a forked line with %s refuses with\n  %q\nand the wording "+
				"is\n  %q", one.what, err.Error(), one.want)
		}
	}

	// The other half: each arm named is a loadout that stands, and they are
	// different loadouts. A test of the refusal alone would pass on a Loadout
	// that refused every form there is.
	for _, arm := range arms {
		drafting := draftAtItsFirstPick(t, all)
		if err := drafting.Pick(wire.SeatHost, forked); err != nil {
			t.Fatalf("the host picks %s: %v", forked, err)
		}
		skills := character.SkillsAt(progression.LevelCap, arm.Name)
		if len(skills) < cast.SkillSlots {
			t.Fatalf("%s knows %d skills as %s at the cap, and a loadout brings %d",
				forked, len(skills), arm.Name, cast.SkillSlots)
		}
		if err := drafting.Loadout(wire.SeatHost, arm.Name, skills[:cast.SkillSlots], nil); err != nil {
			t.Fatalf("a loadout for %s as %s: %v", forked, arm.Name, err)
		}
		if got := drafting.Picks()[seatIndex(t, wire.SeatHost)][0].Stage; got != arm.Name {
			t.Errorf("a pick of %s fielded as %s came back as %s", forked, arm.Name, got)
		}
	}
	t.Logf("%s reaches %v at level %d, and a loadout has to say which",
		forked, progression.StageNames(arms), progression.LevelCap)
}

// draftAtItsFirstPick is a 3v3 draft with every ban skipped, which is the state
// a test about a loadout wants and the shortest way to it.
func draftAtItsFirstPick(t *testing.T, all []cast.Character) *draft.Draft {
	t.Helper()
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	for range 2 * draft.BansPerSide(wire.Format3v3) {
		seat, _, _ := drafting.Turn()
		if err := drafting.SkipBan(seat); err != nil {
			t.Fatalf("%s skips a ban: %v", seat, err)
		}
	}
	if _, step, _ := drafting.Turn(); step != draft.StepPick {
		t.Fatalf("after every ban the draft is due a %s and not a pick", step)
	}
	return drafting
}

// TestAnIllegalKitKeepsChooseLoadoutsOwnWords is the loadout half of decision
// (a): the draft adds a **chooser** and not a second legality rule.
//
// ⚠️ The assertion is that the sentence is cast.ChooseLoadout's, produced by
// calling it directly with the same arguments. A state machine that phrased its
// own refusal would be the fourth spelling of "may this unit bring that skill",
// and that function's own comment records the three there were.
func TestAnIllegalKitKeepsChooseLoadoutsOwnWords(t *testing.T) {
	all := shippedCast(t)
	drafting := draftAtItsFirstPick(t, all)
	taken := firstCandidate(t, drafting)
	if err := drafting.Pick(wire.SeatHost, taken.ID); err != nil {
		t.Fatalf("the host picks %s: %v", taken.ID, err)
	}
	form, legal, _ := legalLoadout(t, taken)

	for _, one := range []struct {
		what     string
		skills   []string
		passives []string
	}{
		{"a skill the character has not learned", []string{"hydro_pump", "hydro_pump", "hydro_pump", "hydro_pump"}, nil},
		{"no skills at all", nil, nil},
		{"one skill twice", append([]string{legal[0]}, legal[0:cast.SkillSlots-1]...), nil},
		{"more skills than there are slots", taken.SkillsAt(progression.LevelCap, form), nil},
		{"a trait the character does not hold", legal, []string{"nothing.at.all"}},
	} {
		_, _, want := cast.ChooseLoadout(fmt.Sprintf("%s's pick of %s", wire.SeatHost, taken.ID),
			one.skills, one.passives, taken, progression.LevelCap, form)
		if want == nil {
			t.Fatalf("%s is a legal loadout for %s as %s, so this case measures nothing",
				one.what, taken.ID, form)
		}
		err := drafting.Loadout(wire.SeatHost, form, one.skills, one.passives)
		switch {
		case err == nil:
			t.Errorf("a loadout with %s was accepted, and cast.ChooseLoadout refuses it with\n"+
				"  %q", one.what, want.Error())
		case err.Error() != want.Error():
			t.Errorf("a loadout with %s refuses with\n  %q\nand cast.ChooseLoadout's own "+
				"wording is\n  %q", one.what, err.Error(), want.Error())
		}
	}
	// The pick is still open after every refusal, which is what makes a refusal
	// a refusal rather than a spent decision.
	if seat, step, due := drafting.Turn(); !due || seat != wire.SeatHost || step != draft.StepLoadout {
		t.Errorf("after five refused loadouts the draft is due %s %s (%v), and the host still "+
			"owes a loadout", seat, step, due)
	}
}

// TestEveryShippedBuildIsALegalDraftLoadout is decision (a)'s first path
// measured on the whole catalogue: **a pick takes a build already in
// builds.json** or one made on the spot, and both go through cast.ChooseLoadout.
//
// It walks every shipped build rather than naming one, which is the shape
// CLAUDE.md asks for wherever a claim is about the data: a build authored later
// that the draft would refuse is a red test here rather than a refusal somebody
// meets in a lobby.
//
// ⚠️ It also holds the level assumption where it would break. Builds are
// validated by cast.ParseBuilds at progression.LevelCap on the furthest form; a
// draft at any other level would refuse most of this catalogue, so this test is
// what would redden if Pick stopped fielding at the cap.
func TestEveryShippedBuildIsALegalDraftLoadout(t *testing.T) {
	all := shippedCast(t)
	pool := draft.NewPool(all)
	catalogue, err := seed.Builds()
	if err != nil {
		t.Fatalf("parse the embedded builds: %v", err)
	}
	builds := catalogue.All()
	if len(builds) == 0 {
		t.Fatal("the embedded build catalogue is empty, so this test measures nothing")
	}

	checked, forked := 0, 0
	for _, build := range builds {
		if !pool.Has(build.Character) {
			// A build for a held-back character is not a draft's business, and
			// naruto.naruto has none today — so this is a skip rather than a gap.
			continue
		}
		drafting := draftAtItsFirstPick(t, all)
		if err := drafting.Pick(wire.SeatHost, build.Character); err != nil {
			t.Fatalf("the host picks %s: %v", build.Character, err)
		}
		if err := drafting.Loadout(wire.SeatHost, build.Stage, build.Skills, build.Passives); err != nil {
			t.Errorf("the shipped build %q is not a loadout a draft accepts: %v", build.ID, err)
			continue
		}
		checked++
		got := drafting.Picks()[seatIndex(t, wire.SeatHost)][0]
		if got.Stage != build.Stage {
			t.Errorf("the build %q is for %s and the pick fielded %s", build.ID, build.Stage, got.Stage)
		}
		if !slices.Equal(got.Skills, build.Skills) {
			t.Errorf("the build %q brings %v and the pick brought %v", build.ID, build.Skills, got.Skills)
		}
		if got.Level != progression.LevelCap {
			t.Errorf("the build %q was fielded at level %d and a build is authored at the cap, %d",
				build.ID, got.Level, progression.LevelCap)
		}
		// ⚠️ **Every build names a form, so counting that would count 28 and
		// measure nothing.** cast.resolveBuild stores the *resolved* stage —
		// absent in the file means the one form there is — so what is worth
		// counting is the build whose **line forks at the cap**, which is the
		// one where naming it is the difference between a legal loadout and no
		// answer at all.
		arms, err := characterNamed(t, all, build.Character).FurthestAt(progression.LevelCap)
		if err != nil {
			t.Fatalf("the forms %s reaches at the cap: %v", build.Character, err)
		}
		if len(arms) > 1 {
			forked++
		}
	}
	if checked == 0 {
		t.Fatal("no shipped build is for a character in the draftable pool, so this walk " +
			"measured nothing")
	}
	if forked == 0 {
		t.Fatal("no shipped build is for a line that forks at the cap, so the half of this " +
			"walk where naming the form is load-bearing is unexercised — see " +
			"TestAForkedLineMustNameItsForm")
	}
	t.Logf("%d of %d shipped builds are for draftable characters and every one is a legal "+
		"loadout; %d are for a line that forks at the cap", checked, len(builds), forked)
}

// TestATimeoutCancelsTheWholeDraft is TODO.md § "Ban and pick" (c): no
// auto-pick, no default, and the match starts over from a new code.
//
// It is the one place the design does not follow "a timeout announces and
// passes", and the reason is that a side which never picked has no squad to
// fight with — so what is asserted is that the draft is over, is **not** done,
// and takes nothing further.
func TestATimeoutCancelsTheWholeDraft(t *testing.T) {
	all := shippedCast(t)
	drafting := draftAtItsFirstPick(t, all)
	taken := firstCandidate(t, drafting)
	if err := drafting.Pick(wire.SeatHost, taken.ID); err != nil {
		t.Fatalf("the host picks %s: %v", taken.ID, err)
	}
	form, skills, passives := legalLoadout(t, taken)
	if err := drafting.Loadout(wire.SeatHost, form, skills, passives); err != nil {
		t.Fatalf("the host's loadout for %s: %v", taken.ID, err)
	}
	// The premise: somebody is being asked something, so there is an allowance
	// to run out.
	onTurn, step, due := drafting.Turn()
	if !due || onTurn != wire.SeatGuest || step != draft.StepPick {
		t.Fatalf("the draft is due %s %s (%v) and this test wants the guest on a pick",
			onTurn, step, due)
	}

	if err := drafting.TimedOut(wire.SeatGuest); err != nil {
		t.Fatalf("the guest's allowance runs out: %v", err)
	}
	if !drafting.Cancelled() {
		t.Error("the guest ran out of time and the draft is not cancelled")
	}
	if drafting.Done() {
		t.Error("a cancelled draft reports itself done, and a done draft has two sides to " +
			"field where this one has none")
	}
	if drafting.Picked() {
		t.Error("a cancelled draft reports its picking as played out, and it stopped mid-pick")
	}
	if drafting.Arranging() {
		t.Error("a cancelled draft has the arrange phase open, and there is nothing to arrange")
	}
	if seat, step, due := drafting.Turn(); due {
		t.Errorf("a cancelled draft is still asking %s for a %s", seat, step)
	}
	if got := drafting.Candidates(); got != nil {
		t.Errorf("a cancelled draft offers %d characters to choose from", len(got))
	}
	// ⚠️ No auto-pick: the guest's side is exactly as empty as it was, and the
	// host keeps what it took. A default would be a squad somebody did not choose.
	picks := drafting.Picks()
	if got := len(picks[seatIndex(t, wire.SeatGuest)]); got != 0 {
		t.Errorf("the guest never picked and has %d picks, so something picked for it", got)
	}
	if got := len(picks[seatIndex(t, wire.SeatHost)]); got != 1 {
		t.Errorf("the host picked once and has %d picks", got)
	}
}

// TestPicksAreWhatAFinishedDraftProduces is the output of the whole step: two
// sides' worth of characters and loadouts, and nothing that pretends to be a
// squad.
func TestPicksAreWhatAFinishedDraftProduces(t *testing.T) {
	all := shippedCast(t)
	drafting, err := draft.New(draft.Config{
		Format: wire.Format3v3, Pool: draft.NewPool(all), First: wire.SeatHost,
	})
	if err != nil {
		t.Fatalf("set up a 3v3 draft: %v", err)
	}
	playOut(t, drafting, all, spendEvery(wire.Format3v3))
	if !drafting.Picked() {
		t.Fatal("the draft was played out and the picking is not over")
	}

	picks := drafting.Picks()
	fielded := []string{}
	for index, side := range picks {
		if len(side) != draft.PicksPerSide(wire.Format3v3) {
			t.Errorf("side %d has %d picks and a 3v3 fields %d",
				index, len(side), draft.PicksPerSide(wire.Format3v3))
		}
		for _, one := range side {
			if one.Level != progression.LevelCap {
				t.Errorf("%s was drafted at level %d and every drafted unit fights at the cap, %d",
					one.Character, one.Level, progression.LevelCap)
			}
			if one.Stage == "" {
				t.Errorf("%s was drafted with no form, and a placement needs one", one.Character)
			}
			if len(one.Skills) == 0 || len(one.Skills) > cast.SkillSlots {
				t.Errorf("%s brings %d skills and there are %d slots",
					one.Character, len(one.Skills), cast.SkillSlots)
			}
			if len(one.Passives) > cast.TraitSlots {
				t.Errorf("%s brings %d traits and there is %d slot",
					one.Character, len(one.Passives), cast.TraitSlots)
			}
			fielded = append(fielded, one.Character)
		}
	}
	// ⚠️ The pool is exclusive, so a drafted match fields six different
	// characters — CLAUDE.md's "one squad may field the same character twice" is
	// about a *saved* squad, and this is the scope of it.
	for at, one := range fielded {
		if slices.Contains(fielded[:at], one) {
			t.Errorf("%s was drafted twice, and one shared exclusive pool cannot do that", one)
		}
	}

	// Copies out, so a caller cannot edit the draft's own picks.
	picks[0][0].Character = "written.over"
	picks[0][0].Skills[0] = "written.over"
	again := drafting.Picks()
	if again[0][0].Character == "written.over" || again[0][0].Skills[0] == "written.over" {
		t.Errorf("editing what Picks handed back changed the draft's own pick: %+v", again[0][0])
	}
	t.Logf("a 3v3 draft fielded %v", fielded)
}
