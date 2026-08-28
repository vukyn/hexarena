package battle_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

// provoking is a board with one attacker and two identical enemies, so that the
// only thing a case changes is which of the two answers back.
//
// The two are the same in every respect a rating reads — the same stats, the same
// health, the same kit — which is what makes the *aim* Suggest picks a reading of
// one term and nothing else. "first" stands in the cell that comes first in the
// option's aim list, so a rating charging neither of them takes it on the tie.
func provoking(t *testing.T, allySkills []string, allyHealth int64,
	firstTraits, secondTraits []string) *battle.Battle {
	t.Helper()
	fight, err := battle.New(books(t), 7, []battle.Roster{
		{ID: "a", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 120),
			Skills: allySkills},
		{ID: "first", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 2},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"strike"}, Passives: firstTraits},
		{ID: "second", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 0},
			Affinity: single("neutral"), Stats: stats(3000, 800, 400, 10),
			Skills: []string{"strike"}, Passives: secondTraits},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	atHealth(t, fight, "a", allyHealth)
	return fight
}

// aimedAt names the unit standing where a choice points, so a failure reads as a
// unit rather than as a pair of coordinates.
func aimedAt(t *testing.T, fight *battle.Battle, choice battle.Choice) string {
	t.Helper()
	for _, id := range []string{"a", "first", "second"} {
		unit, known := fight.Unit(id)
		if known && unit.Cell == choice.Aim {
			return id
		}
	}
	return choice.Aim.String()
}

// TestATargetWithNoReplyIsChargedNothing is the control every case below is read
// against, and it is a test rather than a comment because the whole method here is
// "the aim moved".
//
// Two identical enemies and neither answers: the rating has nothing to tell them
// apart, so the tie falls to the first cell offered. A term that charged something
// to a target holding no reply at all would show up right here.
func TestATargetWithNoReplyIsChargedNothing(t *testing.T) {
	fight := provoking(t, []string{"strike"}, 0, nil, nil)
	if aim := aimedAt(t, fight, chosen(t, fight)); aim != "first" {
		t.Errorf("with neither enemy answering, Suggest aimed at %s rather than taking "+
			"the tie on the first cell offered: something is charging a target that "+
			"holds no reply", aim)
	}
}

// TestAnAttackIsChargedForTheReplyItProvokes is the whole of the change.
//
// The same skill against two enemies that differ in one thing: one of them
// answers. price.go did not mention passive.Replies at all before this, so both
// aims came out at the same figure and the tie went to the first cell — the
// opponent walked into the thorns for want of a term that could see them.
func TestAnAttackIsChargedForTheReplyItProvokes(t *testing.T) {
	fight := provoking(t, []string{"strike"}, 0, []string{"spiked"}, nil)
	if aim := aimedAt(t, fight, chosen(t, fight)); aim != "second" {
		t.Errorf("Suggest aimed at %s with a reply waiting there and an identical enemy "+
			"beside it that answers nothing: the reply is not being charged", aim)
	}
}

// TestAReplyRepelsRatherThanAttracts is the sign, asserted in both directions.
//
// ⚠️ It is worth a test of its own because a sign error here does not look like a
// bug. rate *subtracts* what comes back, so an addition would make an opponent
// prefer the unit that answers — and "the opponent hunts the venom_blood holder"
// reads as a plausible strategy rather than as arithmetic pointing the wrong way.
// The control above pins the tie to the first cell, so a rating that added the
// term would leave the aim on "first" in the first half and move it *to* "first"
// in the second; only a cost moves it away from the answering unit both times.
func TestAReplyRepelsRatherThanAttracts(t *testing.T) {
	onFirst := provoking(t, []string{"strike"}, 0, []string{"spiked"}, nil)
	if aim := aimedAt(t, onFirst, chosen(t, onFirst)); aim != "second" {
		t.Errorf("with the reply on the first cell, Suggest aimed at %s, want second", aim)
	}
	onSecond := provoking(t, []string{"strike"}, 0, nil, []string{"spiked"})
	if aim := aimedAt(t, onSecond, chosen(t, onSecond)); aim != "first" {
		t.Errorf("with the reply on the second cell, Suggest aimed at %s, want first: "+
			"the term is adding what it should subtract, so an answer reads as an "+
			"attraction", aim)
	}
}

// TestAMultiStrikeSkillIsChargedOneAnswerAndNotThree holds the reading of the
// resolving function that this term is built on.
//
// Battle.answer runs after the whole skill loop, once per holder, and says why in
// its own comment: a reply answers a *use* of a skill rather than a strike, so a
// trait's worth cannot scale with somebody else's strike count. So triple and lob
// are charged the same answer, and the heavier skill stays the better one.
//
// Charged per strike instead, triple would pay three times: on this board that
// takes it from 411 to −102, which is not merely worse than lob but *declined*
// outright, and the rating would take the lighter skill for a difference the
// engine never resolves.
func TestAMultiStrikeSkillIsChargedOneAnswerAndNotThree(t *testing.T) {
	bare := provoking(t, []string{"triple", "lob"}, 0, nil, nil)
	if choice := chosen(t, bare); choice.Skill != "triple" {
		t.Fatalf("against a bare target Suggest opened with %q, want triple: the premise "+
			"of this case is that the multi-strike skill is the better one", choice.Skill)
	}
	answered := provoking(t, []string{"triple", "lob"}, 0, []string{"spiked"}, []string{"spiked"})
	if choice := chosen(t, answered); choice.Skill != "triple" {
		t.Errorf("against a replying target Suggest picked %q, want triple: a reply is "+
			"charged once per cast, so three strikes cost one answer and the order of "+
			"the two skills cannot move", choice.Skill)
	}
}

// TestAGatedReplyIsChargedOnlyWhileItIsInForce, because answer reads inForce
// before it lets a trait answer and a shut gate answers nothing.
//
// cornered_spikes replies only while its holder is at or under half health. At
// full health the rating must charge nothing for it and take the tie on the first
// cell; hurt past the line it must charge and aim elsewhere.
func TestAGatedReplyIsChargedOnlyWhileItIsInForce(t *testing.T) {
	standing := provoking(t, []string{"strike"}, 0, []string{"cornered_spikes"}, nil)
	if aim := aimedAt(t, standing, chosen(t, standing)); aim != "first" {
		t.Errorf("Suggest aimed at %s away from a gated reply whose gate is shut: a "+
			"trait that cannot answer is being charged as though it could", aim)
	}

	hurt := provoking(t, []string{"strike"}, 0, []string{"cornered_spikes"}, nil)
	atHealth(t, hurt, "first", 1000)
	if aim := aimedAt(t, hurt, chosen(t, hurt)); aim != "second" {
		t.Errorf("Suggest aimed at %s with the gate open: a reply in force is not being "+
			"charged", aim)
	}
}

// TestAReplyThatWouldFinishTheCasterCostsItsTurnsToo is the dead-attacker rule,
// and it is the one worth getting the right way round.
//
// Battle.reply gives its damage no exemption for arriving out of turn, so an
// answer may kill — and a rating that could not see that would walk its own unit
// into one. The charge is the health it has left plus the turns it will now never
// take, at the same killHorizon friendlyFire charges for an ally it kills, which
// takes the attack well below nought — and since an option priced below nought
// is declined rather than merely deprioritised, the unit passes instead of
// walking into it.
//
// The healthy half is the other side of the same claim, on the identical board:
// the same answer against a caster that would survive it costs 171 out of 342 and
// the attack is still taken. So what the second half tests is the lethal branch
// and not a term that has simply swamped the rating.
//
// Both enemies answer, so declining is a decision about the skill rather than a
// choice of the quieter target — which is the case the other tests here make.
func TestAReplyThatWouldFinishTheCasterCostsItsTurnsToo(t *testing.T) {
	healthy := provoking(t, []string{"strike"}, 0, []string{"spiked"}, []string{"spiked"})
	if choice := chosen(t, healthy); choice.Skill != "strike" {
		t.Fatalf("a healthy caster picked %q against a reply it would survive, want "+
			"strike", choice.Skill)
	}

	dying := provoking(t, []string{"strike"}, 100, []string{"spiked"}, []string{"spiked"})
	prompt, err := dying.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if choice, offered := dying.Suggest(prompt); offered {
		t.Errorf("a caster standing at 100 took %q into an answer that would finish it: "+
			"the reply's damage is not being charged against what the caster has left, "+
			"or the turns it loses with its own death are not being charged at all",
			choice.Skill)
	}
}

// TestNothingButTheReplyTermReadsATraitsAnswer is the double-counting proof, and
// it is by consequence rather than by assertion.
//
// A skill of no power bites nobody, so Act never reaches answer and no reply is
// provoked — which means the reply term returns nought for it by its own first
// guard. If any *other* term in the rating read passive.Replies — a threat, a
// best strike, a kill horizon — the two enemies would stop being identical to it
// and the aim would move. It does not, so the reply reaches the rating through
// exactly one term and cannot be charged twice.
func TestNothingButTheReplyTermReadsATraitsAnswer(t *testing.T) {
	bare := provoking(t, []string{"taint"}, 0, nil, nil)
	want := aimedAt(t, bare, chosen(t, bare))
	answered := provoking(t, []string{"taint"}, 0, []string{"spiked", "caustic"}, nil)
	if got := aimedAt(t, answered, chosen(t, answered)); got != want {
		t.Errorf("a skill of no power aimed at %s once a target could answer and at %s "+
			"when it could not: a reply is reaching the rating through some term other "+
			"than the one that prices it, and would therefore be charged twice",
			got, want)
	}
}

// TestRatingAnAnswerMutatesNothing is TestARatingMutatesNothingAndDrawsNothing on
// a board that has replies on it, because the reply term is the newest thing in
// the file to read a unit that is not the caster.
//
// It reads the holder's stats, the caster's defence and a trait's gate, and it
// walks a shape — every one of those is a read, and none of them may become a
// write. A reply also *drains*, which is a heal on somebody the rating is only
// looking at, so a term that reached the resolving path rather than the pricing
// one would show up as a moved board here and nowhere else.
func TestRatingAnAnswerMutatesNothing(t *testing.T) {
	fight := provoking(t, []string{"strike", "triple", "brace"}, 900,
		[]string{"spiked", "caustic"}, []string{"barbed"})
	carrying(t, fight, "first", "poison", 2)
	carrying(t, fight, "second", "weaken", 1)

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	before := describeAnswering(fight)
	first, ok := fight.Suggest(prompt)
	if !ok {
		t.Fatal("Suggest offered nothing at all")
	}
	for range 500 {
		again, ok := fight.Suggest(prompt)
		if !ok || again != first {
			t.Fatalf("Suggest answered %v then %v (offered: %v)", first, again, ok)
		}
	}
	if after := describeAnswering(fight); after != before {
		t.Errorf("rating a board with replies on it moved the board:\nbefore %s\nafter  %s",
			before, after)
	}
}

// describeAnswering is describeBoard for this file's three units.
func describeAnswering(fight *battle.Battle) string {
	var out strings.Builder
	for _, id := range []string{"a", "first", "second"} {
		unit, known := fight.Unit(id)
		if !known {
			continue
		}
		out.WriteString(id)
		out.WriteString(":")
		out.WriteString(strings.Repeat("|", int(unit.HP%7)))
		for _, held := range unit.Statuses.Snapshot() {
			out.WriteString(" ")
			out.WriteString(held.ID)
			out.WriteString(string(rune('0' + held.Stacks)))
			out.WriteString("/")
			out.WriteString(string(rune('0' + held.Remaining)))
		}
		out.WriteString("; ")
	}
	return out.String()
}
