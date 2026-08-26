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

// TestAReplyAnswersAUseOfASkillRatherThanAStrike is one of the four rules, and
// the one with a number attached.
//
// If a reply fired per strike, a trait's worth would scale with somebody else's
// strike count — a three-strike skill would cost three times what a single one
// does, for a reason written on neither skill and readable from neither.
func TestAReplyAnswersAUseOfASkillRatherThanAStrike(t *testing.T) {
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
	if answered := replies(events); len(answered) != 1 {
		t.Errorf("a %d-strike skill drew %d replies, want one for the use of it",
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
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
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
			attacker.HP, attacker.MaxHP())
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
// An area skill can bite two holders at once, and the first reply may kill the
// unit that cast it. Whatever is left to answer is answering a dead unit: the
// log would carry damage against something whose died line is already written,
// and the difference between letting that happen and not is one hit or several.
func TestNobodyAnswersACorpse(t *testing.T) {
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 90),
			Skills: []string{"lob"}, Passives: []string{"spiked"}},
		{ID: "b", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(4800, 800, 400, 80),
			Skills: []string{"lob"}, Passives: []string{"spiked"}},
		{ID: "f", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(200, 800, 100, 120),
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
				t.Fatalf("act quake: %v", err)
			}
			acted = true
			break
		}
		break
	}
	if !acted {
		t.Fatal("the attacker could not use its area skill")
	}
	events := fight.Drain()

	bitten := map[string]bool{}
	for _, event := range events {
		if event.Kind == battle.Damaged && event.Passive == "" && event.Amount > 0 {
			bitten[event.Target] = true
		}
	}
	if len(bitten) < 2 {
		t.Fatalf("the area skill bit %d holders, so there is no second reply to withhold", len(bitten))
	}
	attacker := unitByID(t, fight, "f")
	if !attacker.Dead {
		t.Fatalf("the attacker survived at %d, so no reply ever killed it", attacker.HP)
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
	if holder.HP <= holder.MaxHP()/2 {
		t.Fatalf("the holder is already past its gate at %d of %d, so this measures nothing",
			holder.HP, holder.MaxHP())
	}
	if answered := replies(fight.Drain()); len(answered) != 0 {
		t.Errorf("a trait gated shut answered anyway: %+v", answered)
	}

	// Hurt it past the line, then attack it again.
	for round := 0; round < 40 && holder.HP > holder.MaxHP()/2; round++ {
		if !step(t, fight) {
			break
		}
		fight.Drain()
	}
	if holder.HP > holder.MaxHP()/2 {
		t.Fatalf("the holder never crossed its gate; it is at %d of %d", holder.HP, holder.MaxHP())
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
