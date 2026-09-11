// What a drain takes back: the strike it is a share of, as that strike lands.
//
// A drain used to run once, after the whole strike loop, on the volley's total
// `dealt`. It now runs inside the loop, once per strike, on that strike's own
// damage — and it runs **before** the reply that strike provokes. The second
// half of that sentence is the balance change and the first half is the
// bookkeeping that makes it expressible:
//
//   - damage → drain → reply. A caster facing thorns meets strike two's answer
//     with strike one's heal already in hand, so a volley it cannot survive
//     under reply-then-drain is one it can survive under this. It pairs with
//     the `actor.Dead` break the per-strike reply put in the same loop: a drain
//     that keeps the caster up keeps the volley going, and those extra strikes
//     drain in their turn.
//   - the base is **that strike's** damage and not the running total, or strike
//     one's damage would be paid out again on strike two.
//   - N truncations instead of one, so the total taken back is at most what a
//     single post-loop drain took and short of it by less than one point per
//     strike. That is what a per-strike drain pays; it is pinned as a bound
//     rather than as a figure.
//
// ⚠️ The fixture skills live in this file rather than in books(). Several tests
// in this package pick their kits by property — strike count, area, pattern —
// so a four strike drainer in the shared book would quietly change what they
// measure. The one skill borrowed from the shared book is `volley`, a
// three-strike conduit, because the discharge question below needs one and the
// shared book already has it.
package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// drainBooks is the shared book plus the six attacks this file needs. They vary
// on two axes and nothing else — how many times they connect, and what share
// they take back:
//
//   - `sip` and `guzzle` take back everything, once and four times. What `sip`
//     draws is the unit a `guzzle` is counted in.
//   - `sup` and `gulp` are the same pair at six tenths, which is the share that
//     does not divide the damage evenly. They are what the truncation is read
//     off; at a full share there is nothing to truncate and the two
//     implementations agree exactly.
//   - `siphon` strikes twice, which is the shortest volley in which a drain can
//     land between two replies — and two is also the only volley a wall can
//     stop WHOLE, since Rules.MaxBlockCharges is three.
//   - `flail` is `guzzle` with the drain taken off, which is what lets the
//     reply's own drain be counted on a board where the caster contributes
//     none of its own.
//   - `spatter` is `siphon` at four tenths accuracy, because a drain that a
//     miss must not pay for needs a strike that can miss. ⚠️ Dodge cannot
//     produce one: combat.Rules.Chance returns the base outright at accuracy
//     1000 before it reads dodge at all, so the miss has to be authored into
//     the skill.
func drainBooks(t *testing.T) battle.Books {
	t.Helper()
	base := books(t)
	deps := skill.Deps{Patterns: base.Patterns, Statuses: base.Statuses}
	extra, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"sip","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","drains":1000},
	  {"id":"guzzle","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":4,"accuracy":1000,"cooldown":0,"target":"enemy","drains":1000},
	  {"id":"sup","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy","drains":600},
	  {"id":"gulp","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":4,"accuracy":1000,"cooldown":0,"target":"enemy","drains":600},
	  {"id":"siphon","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":2,"accuracy":1000,"cooldown":0,"target":"enemy","drains":1000},
	  {"id":"spatter","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":2,"accuracy":400,"cooldown":0,"target":"enemy","drains":1000},
	  {"id":"flail","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":4,"accuracy":1000,"cooldown":0,"target":"enemy"}
	]}`), deps)
	if err != nil {
		t.Fatalf("the drainers this file needs: %v", err)
	}
	book, err := base.Skills.Append(deps, extra.Skills()...)
	if err != nil {
		t.Fatalf("appending them to the fixture book: %v", err)
	}
	base.Skills = book
	return base
}

// drainsBy is every heal a unit took back out of damage it dealt, in order. A
// drain is a Healed carrying a Drained share, which is the encoding that makes
// it reproducible from the log, and it is keyed on the actor because a REPLY
// may drain too — for its own holder, from turn.go's other drain site, which
// this change does not touch.
func drainsBy(events []battle.Event, actor string) []battle.Event {
	out := make([]battle.Event, 0, len(events))
	for _, event := range events {
		if event.Kind == battle.Healed && event.Drained > 0 && event.Actor == actor {
			out = append(out, event)
		}
	}
	return out
}

// totalOf adds the amounts of a set of events up.
func totalOf(events []battle.Event) int64 {
	total := int64(0)
	for _, event := range events {
		total += event.Amount
	}
	return total
}

// drainCast is one cast by a hurt caster into a target that does not answer,
// and the events of that cast alone.
//
// The caster is faster than its target, so the first turn of the battle is the
// one being measured. It is hurt on purpose: b.drain refuses a unit already at
// its maximum, so a caster at full health measures the room check rather than
// the share — which is a real difference this change makes and is measured
// apart, in TestTheRoomCheckNowAppliesPerStrike.
func drainCast(t *testing.T, casterHealth int64, attack string) (*battle.Battle, []battle.Event) {
	t.Helper()
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{attack}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "f", casterHealth)
	if !take(t, fight, attack) {
		t.Fatalf("the caster never got to use %s, so nothing here is measuring a cast", attack)
	}
	return fight, fight.Drain()
}

// TestEveryStrikeThatLandsTakesBackItsOwnShare is the change, and it is asserted
// as a count AND as a total.
//
// The count alone would pass an implementation that fired four events each
// carrying the whole volley's drain — four times what the caster is owed — and
// the total alone would pass one that fired a single event after the loop, as
// this did yesterday. So the single strike cast is run first and its heal is
// the unit: four landing strikes must take back four times exactly that, in
// four events.
//
// The two casts have the same power per strike, so a strike is worth the same
// damage in both and the arithmetic is a multiplication rather than a
// comparison of two different figures.
func TestEveryStrikeThatLandsTakesBackItsOwnShare(t *testing.T) {
	_, once := drainCast(t, 1000, "sip")
	tookOnce := drainsBy(once, "f")
	if len(tookOnce) != 1 {
		t.Fatalf("one strike drained %d times, so there is no unit to count a volley in",
			len(tookOnce))
	}
	unit := tookOnce[0].Amount
	if unit <= 0 {
		t.Fatalf("the drain on one strike took back %d", unit)
	}

	_, volley := drainCast(t, 1000, "guzzle")
	if shape := strikeShape(volley); shape != "DDDD" {
		t.Fatalf("the volley came out %q, want DDDD: every strike has to land or "+
			"this is measuring the blocked case instead", shape)
	}
	took := drainsBy(volley, "f")
	if len(took) != 4 {
		t.Errorf("a volley that landed four times drained %d times, want 4: a drain "+
			"takes back a strike, not a use of a skill", len(took))
	}
	if total, want := totalOf(took), 4*unit; total != want {
		t.Errorf("four landing strikes took back %d in total, want %d — four times "+
			"the %d one strike takes. The count and the total are asserted "+
			"separately on purpose: four events each carrying the whole volley's "+
			"drain is four times the payout wearing the right number of lines",
			total, want, unit)
	}
	for index, event := range took {
		if event.Amount != unit {
			t.Errorf("the drain on strike %d took back %d, want %d: the base is that "+
				"strike's own damage, so a running total would make each one bigger "+
				"than the last", index+1, event.Amount, unit)
		}
	}
}

// TestTheDrainLandsBeforeTheReplyItHasToSurvive is the ordering, and it is the
// reason this step exists at all.
//
// damage → drain → reply. Build the board so that both orders are reachable and
// only one of them leaves the caster standing:
//
//	a strike deals 34 and takes all of it back; the holder's answer is 171.
//	drain first:  300 → 334 → 163 → 197 →  26   alive
//	reply first:  300 → 129 → 163 →  -8        dead at strike two
//
// 300 is inside a window 34 points wide — the size of one heal — which is what
// makes this a test of the order rather than of the arithmetic: below 275 the
// caster dies either way and above 308 it lives either way. Both bounds are
// asserted as premises, so a data change that moves the damage or the reply
// reddens here with a sentence rather than silently stopping discriminating.
//
// ⚠️ The caster has a companion, so its death would not empty a side. Without
// one the reply-first arm would end the battle and every reading below would be
// the finished return rather than the rule.
func TestTheDrainLandsBeforeTheReplyItHasToSurvive(t *testing.T) {
	const casterHealth = 300
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"siphon"}},
		{ID: "g", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 5),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "f", casterHealth)
	if !take(t, fight, "siphon") {
		t.Fatal("the caster never got its turn")
	}
	events := fight.Drain()

	// The premises, in the order they have to hold. Each is a figure the window
	// is derived from, so each is named rather than left to the outcome.
	landed := strikesOf(events)
	if len(landed) != 2 {
		t.Fatalf("the volley landed %d strikes, want 2: the whole question is what "+
			"the caster meets the SECOND answer at", len(landed))
	}
	strike := landed[0].Amount
	answered := replies(events)
	if len(answered) != 2 {
		t.Fatalf("the holder answered %d times, want 2", len(answered))
	}
	answer := answered[0].Amount
	took := drainsBy(events, "f")
	// At least one, because the count is a CLAIM rather than a premise here: a
	// caster killed by the second answer never reaches the second drain, so
	// asserting two up front would report the order fault as a missing heal
	// instead of as the death it caused. The count is checked below, after the
	// thing it is a consequence of.
	if len(took) == 0 {
		t.Fatal("the caster drained nothing at all, so this measures no drain")
	}
	heal := took[0].Amount
	if heal != strike {
		t.Fatalf("a strike dealt %d and took back %d: this skill drains the whole of "+
			"what it deals, and the window below is one heal wide", strike, heal)
	}
	if low, high := 2*answer-2*heal, 2*answer-heal; casterHealth <= low || casterHealth > high {
		t.Fatalf("a caster at %d is outside the window (%d, %d] in which the two "+
			"orders disagree: at or below %d it dies whichever way round they go, "+
			"and above %d it lives either way. The strike is %d, the answer %d",
			casterHealth, low, high, low, high, strike, answer)
	}

	// And the claim: it lived, with the second answer taken at the health the
	// second heal left it.
	caster := unitByID(t, fight, "f")
	if caster.Dead {
		t.Fatal("the caster died to the second answer, which is what happens when " +
			"the reply resolves before the drain: it meets each answer at the " +
			"health the last one left it, with the heal it earned still owing")
	}
	if want := int64(casterHealth) + 2*heal - 2*answer; caster.HP != want {
		t.Errorf("the caster is left at %d, want %d", caster.HP, want)
	}
	if len(took) != 2 {
		t.Errorf("the caster drained %d times, want 2: it survived to throw both "+
			"strikes, so both are paid", len(took))
	}
	// The log says so too: each heal sits between the strike that earned it and
	// the answer to that strike.
	order := make([]string, 0, len(events))
	for _, event := range events {
		switch {
		case event.Kind == battle.Damaged && event.Passive != "":
			order = append(order, "reply")
		case event.Kind == battle.Damaged && event.Passive == "":
			order = append(order, "strike")
		case event.Kind == battle.Healed && event.Drained > 0:
			order = append(order, "drain")
		}
	}
	want := []string{"strike", "drain", "reply", "strike", "drain", "reply"}
	if len(order) != len(want) {
		t.Fatalf("the cast read %v, want %v", order, want)
	}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("the cast read %v, want %v: damage, then drain, then the answer "+
				"to that damage", order, want)
		}
	}
}

// TestASingleStrikeDrainsExactlyWhatItDrainedBefore is the control, and the one
// test in this file that must be as green after the change as before it.
//
// One strike is one drain of one strike's damage under both rules, because the
// volley total and the strike total are the same number and one truncation is
// one truncation. Both shares are run: at a full share there is nothing to
// truncate, and six tenths of 34 is 20.4, which is where a second truncation
// would show if there were one.
//
// The figures are written down rather than derived because that is the claim —
// these are what the engine produced before a drain knew what a strike was.
func TestASingleStrikeDrainsExactlyWhatItDrainedBefore(t *testing.T) {
	for _, row := range []struct {
		attack string
		share  int
		want   int64
	}{
		{"sip", 1000, 34},
		{"sup", 600, 20},
	} {
		t.Run(row.attack, func(t *testing.T) {
			_, events := drainCast(t, 1000, row.attack)
			landed := strikesOf(events)
			if len(landed) != 1 || landed[0].Amount != 34 {
				t.Fatalf("the cast landed %d strikes for %v, want one for 34",
					len(landed), landed)
			}
			took := drainsBy(events, "f")
			if len(took) != 1 {
				t.Fatalf("one strike drained %d times, want once", len(took))
			}
			if took[0].Drained != row.share {
				t.Errorf("the heal reports a share of %d, want %d: a reader who "+
					"cannot see it cannot reproduce the figure", took[0].Drained, row.share)
			}
			if took[0].Amount != row.want {
				t.Errorf("a single strike dealing 34 at a share of %d took back %d, "+
					"want %d: the one-strike case is the case this change is not "+
					"allowed to move", row.share, took[0].Amount, row.want)
			}
		})
	}
}

// TestThePerStrikeTotalIsShortOfTheWholeVolleyFigureByLessThanOnePointAStrike is
// the rounding, stated as the bound it is rather than as a figure.
//
// b.drain divides by the base once per call, so N calls truncate N times where
// the old single call truncated once. The total therefore cannot exceed what one
// call on the volley's damage would have paid, and cannot fall short of it by a
// whole point per strike. Both halves are asserted: the first is the
// conservation rule (health taken back never exceeds damage dealt, and now never
// exceeds what the old rule allowed either), the second is what stops a genuine
// under-payment — a base that was wrong, or a share read once too few times —
// hiding behind "it is only rounding".
//
// Nothing here is a literal. A share of six tenths on this fixture happens to
// lose exactly one point across four strikes, and that is a fact about 34 and
// 600 that a data change is free to move.
func TestThePerStrikeTotalIsShortOfTheWholeVolleyFigureByLessThanOnePointAStrike(t *testing.T) {
	const share = 600
	_, events := drainCast(t, 1000, "gulp")
	landed := strikesOf(events)
	if len(landed) != 4 {
		t.Fatalf("the volley landed %d strikes, want 4", len(landed))
	}
	took := drainsBy(events, "f")
	if len(took) != 4 {
		t.Fatalf("the volley drained %d times, want 4", len(took))
	}
	perStrike := totalOf(took)
	wholeVolley := totalOf(landed) * share / 1000
	if perStrike > wholeVolley {
		t.Errorf("four per-strike drains took back %d where one drain on the whole "+
			"volley would take %d: more truncations cannot pay more, so this is a "+
			"base that is too big", perStrike, wholeVolley)
	}
	if lost := wholeVolley - perStrike; lost >= int64(len(landed)) {
		t.Errorf("four per-strike drains took back %d against the whole volley's "+
			"%d, short by %d: each strike may lose the remainder of one division "+
			"and no more, so a shortfall of %d or more is a missing payout rather "+
			"than rounding", perStrike, wholeVolley, lost, len(landed))
	}
}

// TestAStrikeThatDrewNoBloodTakesBackNothing is the conservation rule at its new
// resolution: a drain is a share of damage, so no damage is no drain.
//
// The two ways a strike draws no blood fail differently and both are rowed. A
// blocked strike ARRIVED and was stopped, which is the case a rider can still
// ride on; a missed one never touched the target at all. Neither pays, and the
// rows where the whole volley draws nothing are what catch an implementation
// that drains per strike THROWN rather than per strike landed.
//
// ⚠️ The wall's depth is set by the caster's speed rather than by counting
// braces, exactly as castIntoAWall in reply_perstrike_test.go does it: a wait is
// 1_000_000/speed, so a slower caster leaves the holder more turns in front of
// it. The miss rows are seeds, because accuracy is the only thing that can
// produce a miss against this fixture and the shape it produces is the premise.
func TestAStrikeThatDrewNoBloodTakesBackNothing(t *testing.T) {
	t.Run("a wall in front of the volley", func(t *testing.T) {
		for _, row := range []struct {
			name  string
			speed int64
			wall  int
			shape string
			heals int
		}{
			{"a volley stopped whole", 100, 2, "BB", 0},
			{"one strike of two through the wall", 150, 1, "BD", 1},
		} {
			t.Run(row.name, func(t *testing.T) {
				wall, before, after, events := walledDrain(t, row.speed, "siphon")
				if wall != row.wall {
					t.Fatalf("the holder went into the volley behind %d charges, want "+
						"%d: this row is chosen for that wall and measures nothing "+
						"without it", wall, row.wall)
				}
				if shape := strikeShape(events); shape != row.shape {
					t.Fatalf("the volley came out %q, want %q", shape, row.shape)
				}
				took := drainsBy(events, "f")
				if len(took) != row.heals {
					t.Errorf("%q drained %d times, want %d: a strike a charge "+
						"cancelled reached nobody and takes nothing back",
						row.shape, len(took), row.heals)
				}
				if row.heals == 0 && after() != before {
					t.Errorf("a volley nothing got through left its caster at %d of "+
						"the %d it went in on", after(), before)
				}
			})
		}
	})

	t.Run("a volley that can miss", func(t *testing.T) {
		for _, row := range []struct {
			seed  uint64
			shape string
			heals int
		}{
			{6, "MM", 0},
			{4, "MD", 1},
			{2, "DD", 2},
		} {
			t.Run(row.shape, func(t *testing.T) {
				fight := mustBattle(t, drainBooks(t), row.seed, []battle.Roster{
					{ID: "a", Side: hex.SideAlly, Slot: ownCell,
						Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
						Skills: []string{"jab"}},
					{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
						Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
						Skills: []string{"spatter"}},
				})
				fight.Begin()
				fight.Drain()
				atHealth(t, fight, "f", 1000)
				if !take(t, fight, "spatter") {
					t.Fatal("the caster never got its turn")
				}
				events := fight.Drain()
				if shape := strikeShape(events); shape != row.shape {
					t.Fatalf("seed %d came out %q, want %q: the row is chosen for "+
						"that arrangement and measures nothing without it",
						row.seed, shape, row.shape)
				}
				took := drainsBy(events, "f")
				if len(took) != row.heals {
					t.Errorf("%q drained %d times, want %d: a missed strike touched "+
						"nobody, so there is no damage for a share of",
						row.shape, len(took), row.heals)
				}
				if row.heals == 0 && unitByID(t, fight, "f").HP != 1000 {
					t.Errorf("a volley that missed everything left its caster at %d "+
						"of the 1000 it started on", unitByID(t, fight, "f").HP)
				}
			})
		}
	})
}

// walledDrain is one cast into a holder that has spent the turns before it
// raising a shield, so some of the volley is stopped and the rest gets through.
// It reports the wall, the caster's health going in, a reading of its health
// afterwards, and the events of the cast.
//
// Bracing on every turn rather than stopping at a count is what keeps the wall
// fresh — a charge lasts two turns, and Set.Apply refreshes the stacks it
// already holds before it decides whether there is room for another.
func walledDrain(t *testing.T, casterSpeed int64, attack string) (
	wall int, before int64, after func() int64, events []battle.Event) {
	t.Helper()
	fight := mustBattle(t, drainBooks(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"brace"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, casterSpeed),
			Skills: []string{attack}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "f", 1000)
	for turn := 0; ; turn++ {
		if turn > 80 {
			t.Fatal("the caster never got a turn, so nothing here is measuring a cast")
		}
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit == "f" {
			if prompt.Skipped {
				t.Fatal("the caster's turn was taken from it, so nothing here is " +
					"measuring a cast")
			}
			break
		}
		if prompt.Skipped {
			continue
		}
		if err := fight.Act("brace", ownCell); err != nil {
			t.Fatalf("the holder's brace: %v", err)
		}
		fight.Drain()
	}
	wall = unitByID(t, fight, "a").Statuses.Stacks("block")
	before = unitByID(t, fight, "f").HP
	if err := fight.Act(attack, ownCell); err != nil {
		t.Fatalf("the caster's %s: %v", attack, err)
	}
	return wall, before, func() int64 { return unitByID(t, fight, "f").HP }, fight.Drain()
}

// TestADeadCasterTakesNoHealForTheStrikesItNeverThrew is the other half of the
// ordering, and it is what the `actor.Dead` break buys.
//
// A drain runs before the answer to the strike that earned it, so a caster is
// alive at every drain it is paid — the guard inside b.drain is unreachable from
// this loop by construction, and the construction is what this measures. What a
// dead caster misses is the strikes it never got to throw: the volley stops, and
// with it the payout.
//
// 300 health against a 34 point strike answered for 171 dies on the third
// answer: 300 → 163 → 26 → −111. So three of four strikes are thrown, three
// drains are paid, and the fourth strike and its drain do not happen.
func TestADeadCasterTakesNoHealForTheStrikesItNeverThrew(t *testing.T) {
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"guzzle"}},
		// A companion, so the caster's death does not empty a side and turn
		// every stop below into the finished return.
		{ID: "g", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 5),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "f", 300)
	if !take(t, fight, "guzzle") {
		t.Fatal("the caster never got its turn")
	}
	events := fight.Drain()

	caster := unitByID(t, fight, "f")
	if !caster.Dead {
		t.Fatalf("the caster survived its own volley at %d, so this measures nothing",
			caster.HP)
	}
	if fight.Finished() {
		t.Fatal("the battle ended with the caster, so every stop below would be the " +
			"finished return rather than the caster's death")
	}
	landed := strikesOf(events)
	if len(landed) != 3 {
		t.Fatalf("the volley landed %d strikes, want 3: the answer to the third one "+
			"killed the caster and the fourth is not thrown", len(landed))
	}
	took := drainsBy(events, "f")
	if len(took) != 3 {
		t.Errorf("the caster drained %d times, want 3: one for each strike it threw "+
			"and none for the one it did not", len(took))
	}
	died := -1
	for index, event := range events {
		if event.Kind == battle.Died && event.Actor == "f" {
			died = index
		}
	}
	if died < 0 {
		t.Fatal("the caster is dead and the log never said so")
	}
	for _, event := range events[died+1:] {
		if event.Kind == battle.Healed && event.Actor == "f" {
			t.Errorf("the caster was healed for %d after its died line", event.Amount)
		}
	}
}

// TestTheRoomCheckNowAppliesPerStrike is the one outcome this change moves that
// is NOT rounding, written down because it is a finding rather than a detail.
//
// b.drain refuses a caster already at its maximum. Under the old rule the single
// drain ran after the whole volley, by which time every answer the volley
// provoked had already opened room; under this one strike one's drain is offered
// the room that exists when strike one lands, which on a caster at full health
// is none. So a full-health caster into a replier takes back one strike's worth
// less than it used to — the price of being paid as you go.
//
// The measurement: 4 strikes at 34, answered for 171 each. Old rule, one drain
// of 136 into 684 of room. New rule, 0 + 34 + 34 + 34 = 102.
//
// The claim asserted is the mechanism rather than the figure: the first strike
// pays nothing and every later one pays in full, so the volley is exactly one
// heal short of what its damage would buy.
func TestTheRoomCheckNowAppliesPerStrike(t *testing.T) {
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"guzzle"}},
	})
	fight.Begin()
	fight.Drain()
	if unit := unitByID(t, fight, "f"); unit.HP != 4800 {
		t.Fatalf("the caster starts at %d of 4800: this test is about a caster with "+
			"no room to heal into, and measures nothing without one", unit.HP)
	}
	if !take(t, fight, "guzzle") {
		t.Fatal("the caster never got its turn")
	}
	events := fight.Drain()
	landed := strikesOf(events)
	if len(landed) != 4 {
		t.Fatalf("the volley landed %d strikes, want 4", len(landed))
	}
	took := drainsBy(events, "f")
	if len(took) != 3 {
		t.Fatalf("the caster drained %d times, want 3: four strikes landed and the "+
			"first found a caster at full health, with nowhere to put a heal",
			len(took))
	}
	if total, want := totalOf(took), 3*landed[0].Amount; total != want {
		t.Errorf("the volley took back %d, want %d: the first strike pays nothing "+
			"and the other three pay in full", total, want)
	}
	if whole := totalOf(landed); totalOf(took) != whole-landed[0].Amount {
		t.Errorf("the volley dealt %d and took back %d: exactly one strike's worth "+
			"short is the price of paying as you go, and anything else is a "+
			"different rule", whole, totalOf(took))
	}
}

// TestAConduitDrainsTheChargeItSetOffAsWellAsTheBlow is the decision about
// discharge, held down so it is not quietly reversed.
//
// b.discharge is once per STRIKE by its own contract — the chain is re-walked
// and the stacks re-spent every time round the loop — and its damage has always
// been added to the same `dealt` a drain took its share of. So it has a well
// defined per-strike figure and it stays in the base. Leaving it out would stop
// a conduit's payload being drained from at all, which is a repricing rather
// than a restructuring, and the pairing is shipped: pokemon.pichu carries
// blood_thirst alongside `spark` and `electro_ball`, both two-strike conduits.
//
// The guard is the base, not the count: each heal must be a share of that
// strike's blow PLUS that strike's arc. A drain keyed on the blow alone still
// produces three heals in three strikes and is wrong in every one of them.
func TestAConduitDrainsTheChargeItSetOffAsWellAsTheBlow(t *testing.T) {
	const share = 250 // `thirst`
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"volley"}, Passives: []string{"thirst"}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "f", 1000)
	carrying(t, fight, "a", "mark", 3)
	if !take(t, fight, "volley") {
		t.Fatal("the caster never got its turn")
	}
	events := fight.Drain()

	// The two damage lines a strike of a conduit produces: the blow, named by
	// the skill alone, and the arc, named by the status it spent.
	type strike struct{ blow, arc int64 }
	var strikes []strike
	for _, event := range events {
		switch {
		case event.Kind == battle.Damaged && event.Passive == "" && event.Status == "":
			strikes = append(strikes, strike{blow: event.Amount})
		case event.Kind == battle.Damaged && event.Status != "" && len(strikes) > 0:
			strikes[len(strikes)-1].arc += event.Amount
		}
	}
	if len(strikes) != 3 {
		t.Fatalf("the conduit threw %d strikes, want 3", len(strikes))
	}
	fired := 0
	for _, each := range strikes {
		if each.arc > 0 {
			fired++
		}
	}
	if fired == 0 {
		t.Fatal("no strike set off a charge, so this measures an ordinary attack " +
			"and says nothing about a discharge")
	}
	took := drainsBy(events, "f")
	if len(took) != len(strikes) {
		t.Fatalf("the conduit drained %d times over %d strikes", len(took), len(strikes))
	}
	for index, each := range strikes {
		want := (each.blow + each.arc) * share / 1000
		if took[index].Amount != want {
			t.Errorf("strike %d dealt %d with the blow and %d with the arc and took "+
				"back %d, want %d: the charge's damage is this strike's damage, and "+
				"a base of the blow alone is a conduit whose payload heals nobody",
				index+1, each.blow, each.arc, took[index].Amount, want)
		}
	}
}

// TestTheReplysOwnDrainStillFiresOncePerStrike is the neighbour this change is
// not allowed to touch.
//
// turn.go has a second drain — a replying holder that also drains, paid out of
// the answer's own damage. It began firing per strike as a consequence of the
// per-strike reply, one step ago, and moving the skill's drain into the same
// loop must not move it a second time. `barbed` is both jobs on one trait, which
// is what lets the two be counted apart on one board: the holder's drains are a
// share of its own replies and the caster's are a share of its strikes.
func TestTheReplysOwnDrainStillFiresOncePerStrike(t *testing.T) {
	const share = 250 // `barbed`
	fight := mustBattle(t, drainBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"barbed"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"flail"}},
	})
	fight.Begin()
	fight.Drain()
	atHealth(t, fight, "a", 2000)
	atHealth(t, fight, "f", 4000)
	if !take(t, fight, "flail") {
		t.Fatal("the caster never got its turn")
	}
	events := fight.Drain()

	answered := replies(events)
	if len(answered) != 4 {
		t.Fatalf("the holder answered %d times, want 4", len(answered))
	}
	byHolder := drainsBy(events, "a")
	if len(byHolder) != len(answered) {
		t.Errorf("the holder answered %d times and drained %d: a reply drains for "+
			"its holder, once each, and this change is not allowed to move that",
			len(answered), len(byHolder))
	}
	for index, event := range byHolder {
		if want := answered[index].Amount * share / 1000; event.Amount != want {
			t.Errorf("the holder's drain on answer %d took back %d, want %d — a "+
				"share of that ANSWER and not of the volley", index+1, event.Amount, want)
		}
	}
	if byCaster := drainsBy(events, "f"); len(byCaster) != 0 {
		t.Errorf("the caster drained %d times off a skill that drains nothing and "+
			"a kit with no draining trait", len(byCaster))
	}
}
