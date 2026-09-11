// What a reply answers: one strike, not one use of a skill.
//
// A trait that answers used to fire once per target, after the whole skill had
// resolved against every cell it covered. It now fires once per **damaging
// strike**, so a volley that connects four times is answered four times and a
// reply is worth four times what it was against that volley. That is a balance
// change rather than a change of bookkeeping, and it is the game owner's
// decision — the shape of the old loop was written to make the question
// unaskable, and the answer to it is now that the rest of the skill does not
// happen.
//
// Three rules survive the move and are measured here, because each of them is a
// line that a per-strike reply could quietly drop:
//
//   - a strike that drew no blood answers nothing, so a blocked volley costs its
//     attacker nothing at all. Otherwise a shield would be a thorns amplifier:
//     the more of a volley a target stopped, the harder it would answer.
//   - the strike that kills a holder is not answered, the same rule as the skill
//     that kills it — the holder is at nought health when the strike resolves
//     and its died line is two events away.
//   - the reply may kill the caster now, in the middle of its own volley, and
//     when it does the volley stops: no further strikes at this target, and no
//     further cells of the shape.
//
// ⚠️ The fixture skills live in this file rather than in books(). Several tests
// in this package pick their kits by property — strike count, area, pattern —
// so a four strike attack in the shared book would quietly change what they
// measure.
package battle_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// perStrikeBooks is the shared book plus the four attacks this file needs. They
// differ only in how many times they connect and in what they cover, which is
// the whole of the variable under test:
//
//   - `barb` strikes once and `barrage` four times at the same power per
//     strike, so what barb draws is the unit a barrage's answer is counted in.
//   - `double` strikes twice, which is the only volley a wall can stop WHOLE:
//     Rules.MaxBlockCharges is three, so a four strike volley always gets
//     something through.
//   - `rake` is barrage over a column, so one cast reaches two holders and can
//     be cut off partway down the shape.
func perStrikeBooks(t *testing.T) battle.Books {
	t.Helper()
	base := books(t)
	deps := skill.Deps{Patterns: base.Patterns, Statuses: base.Statuses}
	extra, err := skill.ParseBook([]byte(`{"skills":[
	  {"id":"barb","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":1,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"double","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":2,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"barrage","element":"neutral","range":1,"pattern":"single",
	   "power":100,"strikes":4,"accuracy":1000,"cooldown":0,"target":"enemy"},
	  {"id":"rake","element":"neutral","range":2,"pattern":"column",
	   "power":100,"strikes":4,"accuracy":1000,"cooldown":0,"target":"all"}
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

// strikesOf is the strikes a skill landed: a Damaged event with a skill behind it
// rather than a trait. replies is its opposite number and lives in reply_test.go.
func strikesOf(events []battle.Event) []battle.Event {
	out := make([]battle.Event, 0, len(events))
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Passive == "" {
			out = append(out, event)
		}
	}
	return out
}

// answeredFor is the total a set of replies took off the attacker, which is the
// figure this change moves. A count on its own would pass an implementation
// that fired the right number of events carrying a share each.
func answeredFor(events []battle.Event) int64 {
	total := int64(0)
	for _, event := range replies(events) {
		total += event.Amount
	}
	return total
}

// strikeShape is the volley written one letter per strike — B blocked, M missed,
// D landed — with the replies left out of it, because a reply is a Damaged event
// too and this is a statement about what the ATTACK did.
func strikeShape(events []battle.Event) string {
	var shape strings.Builder
	for _, event := range events {
		if event.Passive != "" {
			continue
		}
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

// castInto is one cast into a holder that answers, and the events of that cast
// and nothing else.
//
// The attacker is faster than the holder, so the first turn of the battle is
// the one being measured and the holder never gets to act. Its health is the
// caller's, because how much of the answer the attacker can survive is the
// variable in half the cases here.
func castInto(t *testing.T, attackerHealth int64, attack string) (*battle.Battle, []battle.Event) {
	t.Helper()
	fight := mustBattle(t, perStrikeBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(attackerHealth, 800, 400, 120),
			Skills: []string{attack}},
	})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, attack) {
		t.Fatalf("the attacker never got to use %s, so nothing here is measuring a cast", attack)
	}
	return fight, fight.Drain()
}

// castIntoAWall is one cast into a holder that has spent the turns before it raising a
// shield, so that some of the volley is stopped and the rest gets through.
//
// The wall's depth is set by the attacker's speed rather than by counting
// braces, exactly as walledVolley in blockcount_test.go does it: a wait is
// 1_000_000/speed, so a slower attacker leaves the holder more turns in front of
// it. Against the holder's 200 that is one charge at 150 and two at 100.
// Bracing on every turn rather than stopping at a count is what keeps the wall
// fresh — a charge lasts two turns, and Set.Apply refreshes the stacks it
// already holds before it decides whether there is room for another.
func castIntoAWall(t *testing.T, attackerSpeed int64, attack string) (wall int, events []battle.Event) {
	t.Helper()
	fight := mustBattle(t, perStrikeBooks(t), 5, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 200),
			Skills: []string{"brace"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, attackerSpeed),
			Skills: []string{attack}},
	})
	fight.Begin()
	fight.Drain()
	for turn := 0; ; turn++ {
		if turn > 80 {
			t.Fatal("the attacker never got a turn, so nothing here is measuring a cast")
		}
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		fight.Drain()
		if prompt.Unit == "f" {
			if prompt.Skipped {
				t.Fatal("the attacker's turn was taken from it, so nothing here is measuring a cast")
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
	if err := fight.Act(attack, ownCell); err != nil {
		t.Fatalf("the attacker's %s: %v", attack, err)
	}
	return wall, fight.Drain()
}

// TestEveryStrikeThatConnectsDrawsItsOwnReply is the change, and it is asserted
// as a count AND as a total.
//
// The count alone would pass an implementation that fired four events sharing
// one strike's worth of damage between them, which is the old rule wearing the
// new one's log. So the single strike cast is run first and its answer is the
// unit: four connecting strikes must cost four times exactly that.
//
// The two casts have the same power per strike and the same stats on both sides,
// so a reply is priced identically in both — b.replyDamage reads the holder's
// stat and the attacker's defence and rolls nothing at all.
func TestEveryStrikeThatConnectsDrawsItsOwnReply(t *testing.T) {
	_, once := castInto(t, 4800, "barb")
	answeredOnce := replies(once)
	if len(answeredOnce) != 1 {
		t.Fatalf("one strike drew %d replies, so there is no unit to count a volley in",
			len(answeredOnce))
	}
	unit := answeredOnce[0].Amount
	if unit <= 0 {
		t.Fatalf("the reply to one strike dealt %d", unit)
	}

	_, volley := castInto(t, 4800, "barrage")
	if shape := strikeShape(volley); shape != "DDDD" {
		t.Fatalf("the volley came out %q, want DDDD: every strike has to connect or "+
			"this is measuring the blocked case instead", shape)
	}
	answered := replies(volley)
	if len(answered) != 4 {
		t.Errorf("a volley that connected four times drew %d replies, want 4: a reply "+
			"answers a strike, not a use of a skill", len(answered))
	}
	if total, want := answeredFor(volley), 4*unit; total != want {
		t.Errorf("four connecting strikes were answered for %d in total, want %d — "+
			"four times the %d that one strike draws. The count and the total are "+
			"asserted separately on purpose: four events sharing one strike's worth "+
			"is the old rule wearing the new one's log", total, want, unit)
	}
	for index, event := range answered {
		if event.Amount != unit {
			t.Errorf("reply %d of the volley dealt %d, want %d: nothing between the "+
				"strikes changes what a reply is priced at", index+1, event.Amount, unit)
		}
	}
}

// TestOnlyTheStrikesThatDrewBloodAreAnswered is the rule that keeps a shield
// from being a thorns amplifier.
//
// A block cancels a strike whole: nothing arrived, so nothing is answered. If a
// blocked strike provoked a reply then every point a target guarded would come
// back as a point taken, and raising a shield in front of a spiked unit would be
// the attacker's problem rather than the target's.
//
// The rows are chosen so that the two ways of getting this wrong fail
// differently. A volley stopped WHOLE catches an implementation that answers per
// strike thrown; a volley half stopped catches one that answers per strike that
// reached the target, blocked or not.
func TestOnlyTheStrikesThatDrewBloodAreAnswered(t *testing.T) {
	for _, row := range []struct {
		name    string
		speed   int64
		attack  string
		wall    int
		shape   string
		answers int
	}{
		{"a volley stopped whole", 100, "double", 2, "BB", 0},
		{"one strike of two through the wall", 150, "double", 1, "BD", 1},
		{"two strikes of four through the wall", 100, "barrage", 2, "BBDD", 2},
	} {
		t.Run(row.name, func(t *testing.T) {
			wall, events := castIntoAWall(t, row.speed, row.attack)
			if wall != row.wall {
				t.Fatalf("the holder went into the volley behind %d charges, want %d: "+
					"this row is chosen for that wall and measures nothing without it",
					wall, row.wall)
			}
			if shape := strikeShape(events); shape != row.shape {
				t.Fatalf("the volley came out %q, want %q", shape, row.shape)
			}
			answered := replies(events)
			if len(answered) != row.answers {
				t.Errorf("%q drew %d replies, want %d: a strike a charge cancelled "+
					"arrived at nobody and costs its attacker nothing",
					row.shape, len(answered), row.answers)
			}
			if landed := len(strikesOf(events)); len(answered) != landed {
				t.Errorf("%d strikes landed and %d replies answered them: the two are "+
					"the same number by definition of the rule", landed, len(answered))
			}
			if row.answers == 0 && answeredFor(events) != 0 {
				t.Errorf("a volley nothing got through took %d off its attacker",
					answeredFor(events))
			}
		})
	}
}

// TestTheStrikeThatKillsAHolderIsNotAnswered keeps the old rule alive at its new
// resolution.
//
// A holder the skill killed never answered, the way a dead unit cannot be
// healed. Per strike that becomes: the strikes it survived are answered and the
// one that finished it is not. The guard has to be written on health rather than
// on the Dead flag, because the strike loop leaves a target at nought for the
// rest of the skill and kills it afterwards — a flag-only guard would have the
// corpse answering from two events before its died line.
func TestTheStrikeThatKillsAHolderIsNotAnswered(t *testing.T) {
	// 50 health against a 34 point strike: the first leaves it at 16 and the
	// second takes the rest, so the volley has exactly one survivable strike in
	// it and one that is not.
	fight := mustBattle(t, perStrikeBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(50, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		// A second ally, so that killing the holder does not end the battle:
		// Act returns the moment a side is emptied, and this would then be
		// measuring the early return rather than the rule.
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 10),
			Skills: []string{"jab"}},
		{ID: "f", Side: hex.SideEnemy, Slot: ownCell,
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"barrage"}},
	})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "barrage") {
		t.Fatal("the attacker did not get its turn")
	}
	events := fight.Drain()
	holder := unitByID(t, fight, "a")
	if !holder.Dead {
		t.Fatalf("the holder survived the volley at %d, so this measures the wrong case",
			holder.HP)
	}
	if landed := len(strikesOf(events)); landed != 2 {
		t.Fatalf("the volley landed %d strikes, want 2: one the holder survives and "+
			"one that finishes it", landed)
	}
	if answered := replies(events); len(answered) != 1 {
		t.Errorf("a holder that survived one strike of the volley answered %d times, "+
			"want once: the strike that killed it is not answered", len(answered))
	}
	// And the answer came before the died line rather than after it.
	died, lastReply := -1, -1
	for index, event := range events {
		switch {
		case event.Kind == battle.Died && event.Actor == "a":
			died = index
		case event.Kind == battle.Damaged && event.Passive != "":
			lastReply = index
		}
	}
	if died < 0 {
		t.Fatal("the holder is dead and the log never said so")
	}
	if lastReply > died {
		t.Error("a reply was logged after its holder's died line")
	}
}

// TestAVolleyStopsWhereTheReplyKillsItsCaster is the question the old loop was
// shaped to make unaskable, and the owner's answer to it: the caster is dead, so
// nothing further happens.
//
// Nothing further means two things and both are asserted, because they are two
// separate guards in two separate loops: the strikes left in the volley are not
// thrown, and the cells left in the shape are not reached.
//
// ⚠️ The attacker has a companion of its own, so its death does not empty a
// side. Without it the battle would end on the killing reply and every stop
// below would be the b.finished return rather than the guard this test is
// named after.
func TestAVolleyStopsWhereTheReplyKillsItsCaster(t *testing.T) {
	// 300 health against a reply of 171: the first answer leaves the caster at
	// 129 and the second finishes it, at strike two of four.
	fight := mustBattle(t, perStrikeBooks(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 80),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(300, 800, 400, 120),
			Skills: []string{"rake"}},
		{ID: "g", Side: hex.SideEnemy, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 5),
			Skills: []string{"jab"}},
	})
	fight.Begin()
	fight.Drain()
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if prompt == nil || prompt.Unit != "f" {
		t.Fatalf("the attacker did not go first: %+v", prompt)
	}
	if err := fight.Act("rake", prompt.Options[0].Aims[0]); err != nil {
		t.Fatalf("rake: %v", err)
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
	// The strikes it did not get to throw. The first cell of the shape is the
	// far holder, and it was hit twice out of four.
	landed := strikesOf(events)
	if len(landed) != 2 {
		t.Errorf("the volley landed %d strikes, want 2: the reply to the second one "+
			"killed the caster and the third and fourth are not thrown", len(landed))
	}
	for index, event := range landed {
		if event.Strike != index+1 {
			t.Errorf("strike %d of the volley is logged as number %d", index+1, event.Strike)
		}
	}
	if answered := replies(events); len(answered) != 2 {
		t.Errorf("the volley was answered %d times, want 2", len(answered))
	}
	// The cells it did not get to reach. Whichever of the two holders the shape
	// puts first, the other one is untouched — no strike, no status, nothing.
	hit := map[string]int{}
	for _, event := range landed {
		hit[event.Target]++
	}
	if len(hit) != 1 {
		t.Fatalf("the volley reached %d holders, want 1: the second cell of the shape "+
			"is not walked once the caster is dead", len(hit))
	}
	spared := "a"
	if _, first := hit["a"]; first {
		spared = "b"
	}
	for _, event := range events {
		if event.Target == spared {
			t.Errorf("%s reached %s, which is a cell the dead caster never got to",
				event.Kind, spared)
		}
	}
	// And nothing at all reaches the caster after its died line, reply or status.
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
		if event.Target == "f" {
			t.Errorf("%s reached the caster after its died line", event.Kind)
		}
	}
}

// TestAnAreaSkillIsAnsweredByEachHolderForEachStrikeThatConnected is the two
// dimensions multiplied: a shape that covers two holders, thrown four times.
//
// The second arm is the guard that a holder is not the caster. An area skill
// with target "all" can be aimed at the caster's own cell, so the caster catches
// itself four times — and a unit is not attacking itself however its own skill
// reaches it. Its ally in the same column holds the same trait and answers all
// four, which is what says the arm reached the reply code at all.
func TestAnAreaSkillIsAnsweredByEachHolderForEachStrikeThatConnected(t *testing.T) {
	t.Run("two holders in the shape", func(t *testing.T) {
		fight := mustBattle(t, perStrikeBooks(t), 7, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
				Skills: []string{"jab"}, Passives: []string{"spiked"}},
			{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 80),
				Skills: []string{"jab"}, Passives: []string{"spiked"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
				Skills: []string{"rake"}},
		})
		fight.Begin()
		fight.Drain()
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if err := fight.Act("rake", prompt.Options[0].Aims[0]); err != nil {
			t.Fatalf("rake: %v", err)
		}
		events := fight.Drain()
		if landed := len(strikesOf(events)); landed != 8 {
			t.Fatalf("the volley landed %d strikes, want 8: two holders four times "+
				"each, or the arithmetic below is about a different cast", landed)
		}
		byHolder := map[string]int{}
		for _, event := range replies(events) {
			byHolder[event.Actor]++
		}
		if byHolder["a"] != 4 || byHolder["b"] != 4 {
			t.Errorf("the holders answered %v, want four each: per target, per "+
				"connecting strike", byHolder)
		}
	})

	t.Run("the caster catches itself", func(t *testing.T) {
		fight := mustBattle(t, perStrikeBooks(t), 7, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
				Skills: []string{"jab"}, Passives: []string{"spiked"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
				Skills: []string{"rake"}, Passives: []string{"spiked"}},
			{ID: "g", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 0},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 5),
				Skills: []string{"jab"}, Passives: []string{"spiked"}},
		})
		fight.Begin()
		fight.Drain()
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil || prompt.Unit != "f" {
			t.Fatalf("the caster did not go first: %+v", prompt)
		}
		// Its own cell, which for a target "all" shape is a legal aim: the last
		// one offered is the column it is standing in.
		aims := prompt.Options[0].Aims
		if err := fight.Act("rake", aims[len(aims)-1]); err != nil {
			t.Fatalf("rake: %v", err)
		}
		events := fight.Drain()
		selfHit := 0
		for _, event := range strikesOf(events) {
			if event.Target == "f" {
				selfHit++
			}
		}
		if selfHit != 4 {
			t.Fatalf("the caster hit itself %d times, want 4: this arm measures nothing "+
				"unless the shape really caught it", selfHit)
		}
		byHolder := map[string]int{}
		for _, event := range replies(events) {
			byHolder[event.Actor]++
		}
		if byHolder["f"] != 0 {
			t.Errorf("the caster answered its own skill %d times: a unit its own shape "+
				"caught is still not somebody attacking it", byHolder["f"])
		}
		if byHolder["g"] != 4 {
			t.Errorf("the caster's neighbour answered %d times, want 4 — without which "+
				"the nought above is a reply path that was never reached", byHolder["g"])
		}
	})
}

// TestASingleStrikeIsAnsweredExactlyAsItWasBeforeTheChange is the control, and
// the one test in this file that must be as green after the change as before it.
//
// One strike drew one reply under the old rule and draws one under the new one,
// for the same amount, in the same place in the log. The figures are written
// down rather than derived because that is the claim: these are the numbers the
// engine produced before a reply knew what a strike was.
func TestASingleStrikeIsAnsweredExactlyAsItWasBeforeTheChange(t *testing.T) {
	fight, events := castInto(t, 4800, "barb")
	answered := replies(events)
	if len(answered) != 1 {
		t.Fatalf("one strike drew %d replies, want exactly one", len(answered))
	}
	if answered[0].Amount != 171 {
		t.Errorf("the reply to a single strike dealt %d, want 171: the one-hit case "+
			"is the case this change is not allowed to move", answered[0].Amount)
	}
	attacker := unitByID(t, fight, "f")
	if attacker.HP != 4800-171 {
		t.Errorf("the attacker is left at %d of 4800, want %d", attacker.HP, 4800-171)
	}
	// And it is logged after the strike it answers rather than before it, which
	// is the reading a renderer plays back.
	landed := strikesOf(events)
	if len(landed) != 1 {
		t.Fatalf("the cast landed %d strikes, want one", len(landed))
	}
	strikeAt, replyAt := -1, -1
	for index, event := range events {
		switch {
		case event.Kind == battle.Damaged && event.Passive == "":
			strikeAt = index
		case event.Kind == battle.Damaged && event.Passive != "":
			replyAt = index
		}
	}
	if strikeAt < 0 || replyAt < 0 || replyAt < strikeAt {
		t.Errorf("the strike is at %d in the log and its answer at %d", strikeAt, replyAt)
	}
}
