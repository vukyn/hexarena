package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// answering is a duel where each side may hold a trait, so a test can say which
// of the two answers and let the other one attack.
//
// The attacker is given the health to survive what it provokes, because every
// question here is about what the reply *does* rather than about who wins; the
// one test that wants a reply to kill sets its own numbers.
func answering(t *testing.T, allyTrait, foeTrait string, allySkills, foeSkills []string) *battle.Battle {
	t.Helper()
	roster := []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: allySkills},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 100),
			Skills: foeSkills},
	}
	if allyTrait != "" {
		roster[0].Passives = []string{allyTrait}
	}
	if foeTrait != "" {
		roster[1].Passives = []string{foeTrait}
	}
	fight, err := battle.New(books(t), 7, roster)
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	return fight
}

// replies is the damage a trait answered with, in the order it was dealt. A
// reply is a Damaged event carrying a trait instead of a skill, which is the
// encoding a renderer reads it by, so a test that looked for anything else would
// be checking a different contract from the one shipped.
func replies(events []battle.Event) []battle.Event {
	out := make([]battle.Event, 0, len(events))
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Passive != "" {
			out = append(out, event)
		}
	}
	return out
}

// TestATraitAnswersTheUnitThatAttackedItsHolder is the feature: a trait that
// fires on somebody else's turn, from a unit that is not acting.
func TestATraitAnswersTheUnitThatAttackedItsHolder(t *testing.T) {
	fight := answering(t, "spiked", "", []string{"strike"}, []string{"strike"})
	fight.Begin()
	fight.Drain()

	// The ally acts first and is the holder, so its own turn must draw nothing:
	// a skill the holder chose is not somebody attacking it.
	if !take(t, fight, "strike") {
		t.Fatal("the holder did not get its turn")
	}
	if answered := replies(fight.Drain()); len(answered) != 0 {
		t.Errorf("the holder answered its own attack: %+v", answered)
	}

	if !take(t, fight, "strike") {
		t.Fatal("the attacker did not get its turn")
	}
	answered := replies(fight.Drain())
	if len(answered) != 1 {
		t.Fatalf("being attacked produced %d replies, want exactly one", len(answered))
	}
	if answered[0].Actor != "a" || answered[0].Target != "f" {
		t.Errorf("the reply reads %s answering %s, want a answering f",
			answered[0].Actor, answered[0].Target)
	}
	if answered[0].Passive != "spiked" {
		t.Errorf("the reply names %q, want the trait it came from", answered[0].Passive)
	}
	if answered[0].Skill != "" {
		t.Errorf("the reply names the skill %q as well as the trait, and exactly one of the two is the source",
			answered[0].Skill)
	}
	if answered[0].Amount <= 0 {
		t.Errorf("the reply dealt %d", answered[0].Amount)
	}
}

// TestAReplyAnswersAStrikeRatherThanAUseOfASkill is one of the four rules, and
// it is the one that has been REVERSED.
//
// It used to read the other way round: a reply answered a use, so a trait's
// worth could not scale with somebody else's strike count and a three-strike
// skill cost exactly what a single one did. The game's owner decided the
// opposite — a volley that connects three times is answered three times — so
// reply damage is now multiplied by the strikes that got through, deliberately.
//
// The count is what is asserted here, beside its siblings. The damage total,
// the blocked cases, the mid-volley kill and the one-strike control are in
// reply_perstrike_test.go, which owns the rule.
func TestAReplyAnswersAStrikeRatherThanAUseOfASkill(t *testing.T) {
	fight := answering(t, "spiked", "", []string{"strike"}, []string{"triple"})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "strike") {
		t.Fatal("the holder did not get its turn")
	}
	fight.Drain()
	if !take(t, fight, "triple") {
		t.Fatal("the attacker did not get its turn")
	}
	events := fight.Drain()
	strikes := 0
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Passive == "" && event.Target == "a" {
			strikes++
		}
	}
	if strikes < 2 {
		t.Fatalf("the attacking skill landed %d strikes, so this measures nothing", strikes)
	}
	if answered := replies(events); len(answered) != strikes {
		t.Errorf("a skill that connected %d times drew %d replies, want one per strike",
			strikes, len(answered))
	}
}

// TestAReplyNeverTriggersAReply is closed by rule rather than by a depth
// counter, because a counter is a number somebody raises.
//
// Two holders facing each other settle in one exchange: the attacker's trait has
// nothing to answer, because a reply is not an attack anybody made.
func TestAReplyNeverTriggersAReply(t *testing.T) {
	fight := answering(t, "spiked", "spiked", []string{"strike"}, []string{"strike"})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "strike") {
		t.Fatal("the first unit did not get its turn")
	}
	answered := replies(fight.Drain())
	if len(answered) != 1 {
		t.Fatalf("one attack between two holders produced %d replies, want one", len(answered))
	}
	if answered[0].Actor != "f" {
		t.Errorf("the reply came from %q, want the unit that was attacked", answered[0].Actor)
	}
}

// TestAHolderKilledByTheSkillDoesNotAnswer is the counter retaliation gets
// without anybody designing one: kill it outright and there is no reply.
//
// Dead is dead, the same rule that stops a dead unit being healed. It also means
// a trait that punishes attacking rewards hitting hard rather than taxing
// everybody equally.
func TestAHolderKilledByTheSkillDoesNotAnswer(t *testing.T) {
	// A second ally, so that killing the holder does not end the battle. Without
	// it this test passes whatever the code does: Act returns the moment a side
	// is wiped out, so nothing would reach the reply either way and the check
	// below would be measuring the early return rather than the rule.
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(600, 800, 100, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		// Behind the holder rather than beside it: reach is counted in ranks now,
		// so a companion in the same rank is an equally legal aim and the attack
		// lands on whichever comes first in cell order — which is not the unit
		// this case is about.
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 0, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 10),
			Skills: []string{"lob"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 120),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "strike") {
		t.Fatal("the attacker did not get its turn")
	}
	events := fight.Drain()
	ally := unitByID(t, fight, "a")
	if !ally.Dead {
		t.Fatalf("the holder survived at %d, so this measures the wrong case", ally.HP)
	}
	if answered := replies(events); len(answered) != 0 {
		t.Errorf("a dead holder answered: %+v", answered)
	}
}

// TestAReplyMayKill is the first rule, and the one that changes what a battle
// can be: it can end on a turn nobody took.
//
// A damage-over-time tick already ends battles, so the shape exists; what is new
// is that the unit dying is the one whose turn it is.
func TestAReplyMayKill(t *testing.T) {
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"jab"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(700, 800, 100, 120),
			Skills: []string{"jab"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	fight.Drain()
	attacker := unitByID(t, fight, "f")
	for round := 0; round < 12 && !attacker.Dead; round++ {
		if !step(t, fight) {
			break
		}
	}
	if !attacker.Dead {
		t.Fatalf("the attacker survived at %d of %d, so no reply ever killed",
			attacker.HP, fight.MaxHP(attacker))
	}
	events := fight.Drain()
	// The battle is over, and it is over because of a reply rather than because
	// somebody's turn came round: the last thing to hurt the attacker names a
	// trait.
	if !fight.Finished() {
		t.Error("the last unit on a side is dead and the battle did not end")
	}
	lastHarm := battle.Event{}
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Target == "f" {
			lastHarm = event
		}
	}
	if lastHarm.Passive == "" {
		t.Errorf("the killing damage came from the skill %q rather than a reply", lastHarm.Skill)
	}
}

// TestNobodyAnswersACorpse is the second holder's turn to speak, and the point
// is that it does not get one.
//
// An area skill can bite two holders at once, and the first one's reply may kill
// the unit that cast it. It used to be able to reach the second holder anyway —
// every cell was resolved before anybody answered — so the rule that saved the
// log from carrying damage against an already-dead attacker was a guard inside
// the answering loop. It is now a guard in the cell walk instead: the second
// holder is never touched at all, so there is nothing for it to answer.
//
// The two arms are the same cast with the same aim, and only the caster's
// health differs. The surviving one is what says the shape really covers two
// holders — without it, "the second holder was not hit" would hold just as well
// for an aim that never reached it.
func TestNobodyAnswersACorpse(t *testing.T) {
	sweep := func(t *testing.T, casterHealth int64) (*battle.Battle, []battle.Event) {
		t.Helper()
		fight, err := battle.New(books(t), 7, []battle.Roster{
			{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
				Skills: []string{"lob"}, Passives: []string{"spiked"}},
			{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(4800, 800, 400, 80),
				Skills: []string{"lob"}, Passives: []string{"spiked"}},
			{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
				Affinity: single("neutral"), Stats: stats(casterHealth, 800, 100, 120),
				Skills: []string{"sweep"}},
		})
		if err != nil {
			t.Fatalf("new battle: %v", err)
		}
		fight.Begin()
		fight.Drain()

		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil || prompt.Unit != "f" {
			t.Fatalf("the attacker did not go first: %+v", prompt)
		}
		acted := false
		for _, option := range prompt.Options {
			if option.Skill != "sweep" || !option.Available() {
				continue
			}
			for _, aim := range option.Aims {
				if err := fight.Act("sweep", aim); err != nil {
					t.Fatalf("act sweep: %v", err)
				}
				acted = true
				break
			}
			break
		}
		if !acted {
			t.Fatal("the attacker could not use its area skill")
		}
		return fight, fight.Drain()
	}
	bitten := func(events []battle.Event) map[string]bool {
		out := map[string]bool{}
		for _, event := range events {
			if event.Kind == battle.Damaged && event.Passive == "" && event.Amount > 0 {
				out[event.Target] = true
			}
		}
		return out
	}

	// The premise: a caster that can take both answers reaches both holders and
	// is answered by both.
	survived, whole := sweep(t, 4800)
	if reached := bitten(whole); len(reached) != 2 {
		t.Fatalf("the area skill bit %d holders when its caster survived, want 2: "+
			"the arm below measures nothing unless the shape covers them both", len(reached))
	}
	if unitByID(t, survived, "f").Dead {
		t.Fatal("the caster with full health died anyway, so it is not the control")
	}
	if answered := replies(whole); len(answered) != 2 {
		t.Errorf("two holders bitten once each answered %d times, want twice", len(answered))
	}

	// The case: a caster the first answer kills.
	fight, events := sweep(t, 200)
	attacker := unitByID(t, fight, "f")
	if !attacker.Dead {
		t.Fatalf("the attacker survived at %d, so no reply ever killed it", attacker.HP)
	}
	if reached := bitten(events); len(reached) != 1 {
		t.Errorf("a caster killed at the first cell of its shape still bit %d holders, "+
			"want 1: the cells after it are not walked", len(reached))
	}
	answered := replies(events)
	if len(answered) != 1 {
		t.Errorf("%d holders answered a unit the first reply killed, want one", len(answered))
	}
	// And nothing at all reaches it after its died line, reply or status.
	died := -1
	for i, event := range events {
		if event.Kind == battle.Died && event.Actor == "f" {
			died = i
		}
	}
	if died < 0 {
		t.Fatal("the attacker is dead and the log never said so")
	}
	for _, event := range events[died+1:] {
		if event.Target == "f" {
			t.Errorf("%s reached the attacker after its died line", event.Kind)
		}
	}
}

// TestASkillThatDrawsNoBloodIsNotAnswered is what "costs whatever bit into it"
// means, and the case has to be a skill that *reached* the holder.
//
// A skill aimed elsewhere proves nothing — it was never going to be answered,
// whatever the rule is. anthem lands on the holder, crosses the midline, applies
// a status and has no power at all, so the only thing separating it from a
// strike is that it drew nothing.
func TestASkillThatDrawsNoBloodIsNotAnswered(t *testing.T) {
	fight := answering(t, "spiked", "", []string{"jab"}, []string{"anthem"})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "jab") {
		t.Fatal("the holder did not get its turn")
	}
	fight.Drain()
	if !take(t, fight, "anthem") {
		t.Fatal("the attacker did not get its turn")
	}
	events := fight.Drain()
	// The skill really did reach the holder, or this is the aimed-elsewhere case
	// wearing a different name.
	reached := false
	for _, event := range events {
		if event.Kind == battle.StatusApplied && event.Target == "a" {
			reached = true
		}
	}
	if !reached {
		t.Fatal("the skill never touched the holder, so this measures nothing")
	}
	if answered := replies(events); len(answered) != 0 {
		t.Errorf("a skill that drew no blood was answered: %+v", answered)
	}
}

// TestAGatedTraitAnswersOnlyWhileItsGateHolds is the gate meaning what it says
// on the one job it had not been asked to do yet.
func TestAGatedTraitAnswersOnlyWhileItsGateHolds(t *testing.T) {
	fight := answering(t, "cornered_spikes", "", []string{"jab"}, []string{"strike"})
	fight.Begin()
	fight.Drain()
	holder := unitByID(t, fight, "a")

	if !take(t, fight, "jab") {
		t.Fatal("the holder did not get its turn")
	}
	fight.Drain()
	if !take(t, fight, "strike") {
		t.Fatal("the attacker did not get its turn")
	}
	if holder.HP <= fight.MaxHP(holder)/2 {
		t.Fatalf("the holder is already past its gate at %d of %d, so this measures nothing",
			holder.HP, fight.MaxHP(holder))
	}
	if answered := replies(fight.Drain()); len(answered) != 0 {
		t.Errorf("a trait gated shut answered anyway: %+v", answered)
	}

	// Hurt it past the line, then attack it again.
	for round := 0; round < 40 && holder.HP > fight.MaxHP(holder)/2; round++ {
		if !step(t, fight) {
			break
		}
		fight.Drain()
	}
	if holder.HP > fight.MaxHP(holder)/2 {
		t.Fatalf("the holder never crossed its gate; it is at %d of %d", holder.HP, fight.MaxHP(holder))
	}
	seen := 0
	for round := 0; round < 6 && seen == 0; round++ {
		if !step(t, fight) {
			break
		}
		seen += len(replies(fight.Drain()))
	}
	if seen == 0 {
		t.Error("the trait is past its gate and still answers nothing")
	}
}

// TestAReplysStatusGoesThroughTheSameResistances is what "not a second damage
// path" buys: a reply is refused, rolled and logged by the code every other
// application uses.
func TestAReplysStatusGoesThroughTheSameResistances(t *testing.T) {
	fight := answering(t, "caustic", "clean_blood", []string{"jab"}, []string{"strike"})
	fight.Begin()
	fight.Drain()
	if !take(t, fight, "jab") {
		t.Fatal("the holder did not get its turn")
	}
	fight.Drain()
	if !take(t, fight, "strike") {
		t.Fatal("the attacker did not get its turn")
	}
	events := fight.Drain()
	attacker := unitByID(t, fight, "f")
	if attacker.Statuses.Has("poison") {
		t.Error("a unit immune to poison was poisoned by a reply")
	}
	refused := find(events, battle.StatusResisted)
	found := false
	for _, event := range refused {
		if event.Passive == "caustic" && event.Refused > 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("the refusal does not name the trait it refused: %+v", refused)
	}
}
