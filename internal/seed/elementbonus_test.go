package seed_test

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// The per-element table replaces one bonus that paid a squad for sharing
// *anything* with a table that says what each tribe is FOR. What it buys is a
// reason to build one way rather than another; what it costs is a rung per
// element that nobody can reach unless the cast can field it.
//
// ⚠️ This file is the reachability half, and it is checked BY PROPERTY rather
// than by a list. A hand-kept table of "which elements have enough carriers" goes
// stale the day a character is authored, and it goes stale silently — a bonus
// nobody can trigger loads, draws and never fires, which is the one failure
// composition.Axis exists to prevent one level up.
//
// ⚠️ It now holds a second property, on a different axis: DISTINCTNESS. Two
// bonuses that come to the same thing under different words are the failure
// DAT-002 decision 5 forbids, and it is not a reachability question at all — the
// table can be perfectly reachable and still pay a squad twice for one effect.
// TestNoTwoBonusesGrantTheSameEffect at the foot of this file is that half, with
// the one shipped collision named as an exception and pinned both ways. So: the
// tests above ask whether every rung can FIRE, and that one asks whether what
// they fire is worth telling apart.
//
// ⚠️ **"Can fire" here means the CAST could field the rung, and nothing in this
// file asks whether anything that SHIPS reaches one.** That is
// bonusboard_test.go, which walks the whole book against the five squads and the
// two roster halves and reports the answer — DAT-010. Today it is "none of the
// ten, on any of the seven", which no test in this file can see.

// carriersByElement counts the shipped cast by the elements they carry, which is
// the ceiling on every rung a per-element bonus can declare.
//
// The inert element is skipped for the reason Awards skips it: sharing the
// element with no matchup is sharing the absence of one, and no bonus may name it
// — parse refuses that outright.
func carriersByElement(t *testing.T) map[element.Element]int {
	t.Helper()
	book, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	chart, err := seed.ElementChart()
	if err != nil {
		t.Fatalf("load the chart: %v", err)
	}
	counted := make(map[element.Element]int)
	for _, character := range book.All() {
		for _, member := range character.Element.Elements() {
			if isInert(chart, member) {
				continue
			}
			counted[member]++
		}
	}
	return counted
}

func isInert(chart *element.Chart, member element.Element) bool {
	for _, one := range chart.Inert() {
		if one == member {
			return true
		}
	}
	return false
}

// TestEveryElementBonusRungIsReachable is the guard the roadmap asked for five
// times over: a table declared with rungs the cast cannot field would ship with
// its top half dead and nothing would say so.
func TestEveryElementBonusRungIsReachable(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	carriers := carriersByElement(t)
	var checked int
	for _, held := range bonuses.All() {
		if held.Axis != composition.AxisElement || held.Value == "" {
			continue
		}
		member, err := element.Parse(held.Value)
		if err != nil {
			t.Fatalf("%s counts %q, which is no element: %v", held.ID, held.Value, err)
		}
		fieldable := carriers[member]
		if fieldable > hex.MaxTeamSize {
			fieldable = hex.MaxTeamSize
		}
		checked++
		for _, rung := range held.Rungs {
			if rung.At > fieldable {
				t.Errorf("%s declares a rung at %d and the cast fields %d %s carriers: "+
					"a rung nobody can reach loads, draws and never fires",
					held.ID, rung.At, carriers[member], member)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no bonus names an element, so this measures nothing")
	}
}

// TestEveryFieldableElementHasABonus is the other direction, and it is what stops
// the table being finished by forgetting.
//
// An element two of the cast carry is an element a player can build a tribe
// around. If the table has no entry for it, that player gets nothing for a squad
// that satisfies every rule the design has.
//
// ⚠️ This used to REPORT rather than fail, because `same_element` shipped and
// covered every gap — a blanket bonus paying whichever element a side happened to
// share. That escape hatch is gone with it: the eighth element got its own bonus,
// the blanket was retired, and an uncovered element is now a squad that is paid
// nothing rather than a squad paid something generic.
func TestEveryFieldableElementHasABonus(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	named := make(map[string]bool)
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value != "" {
			named[held.Value] = true
		}
	}
	for member, count := range carriersByElement(t) {
		if count < composition.MinimumRung || named[member.String()] {
			continue
		}
		t.Errorf("%s has %d carriers and no bonus of its own: a squad built around it "+
			"is paid nothing", member, count)
	}
}

// TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier is the gap the
// retirement opens, pinned so that it cannot grow.
//
// ⚠️ **`same_element` was not covering nothing.** `carriersByElement` counts the
// CAST, and TestEveryFieldableElementHasABonus reads "one carrier" as "no squad
// can field a tribe of it" — which is true of a DRAFTED squad, because the pool
// is exclusive and a drafted side is six or ten different characters by
// construction, and false of a SAVED one: `internal/draft` records both, and a
// saved squad may field the same character twice. So a saved squad of two Lapras
// reaches rung 2 on ice, and after the retirement it is paid **nothing**, where
// the blanket used to pay it.
//
// That is accepted rather than fixed, and the reason is that fixing it means
// inventing two effects for two tribes only a doubled-up squad can field — a
// bonus authored for a case nobody builds towards, which is the opposite of the
// rule the table was built under. What may not happen is the set growing quietly,
// so it is named here: exactly the elements with one carrier, and no others. A
// third element losing coverage fails, and so does a second light or ice
// character shipping without a bonus following it.
func TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	named := make(map[string]bool)
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value != "" {
			named[held.Value] = true
		}
	}
	for member, count := range carriersByElement(t) {
		switch {
		case named[member.String()] && count < composition.MinimumRung:
			t.Errorf("%s has a bonus of its own and only %d carrier(s): a rung it takes "+
				"a doubled-up saved squad to reach is not what the table was built for",
				member, count)
		case !named[member.String()] && count >= composition.MinimumRung:
			t.Errorf("%s has %d carriers and no bonus of its own", member, count)
		}
	}
}

// TestNoBlanketElementBonusShips is the retirement itself, held as a property.
//
// The per-element table and a blanket that pays for sharing *anything* are two
// answers to one question, and both paying at once is what the table was built to
// replace — a tribe would collect its own bonus and the generic one on top, so
// what a player is really choosing between is every element plus a constant.
//
// It names no id, because the thing being refused is the SHAPE. A second blanket
// under another name would be the same design arriving by another door, and a
// test naming `same_element` would let it through.
func TestNoBlanketElementBonusShips(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	for _, held := range bonuses.All() {
		if held.Axis == composition.AxisElement && held.Value == "" {
			t.Errorf("bonus %q counts the element axis and names no element: it pays every "+
				"tribe on top of the bonus that tribe already has", held.ID)
		}
	}
}

// acceptedEffectCollision is the one collision the table ships with, named and
// justified. DAT-008.
//
// `ground_root` grants `bedrock` and `same_column` grants `phalanx`, and the two
// kinds are byte-identical: defence +100‰, up to two stacks, permanent. Derive
// it rather than trusting this comment —
//
//	jq -r '.kinds[]|select(.id=="bedrock" or .id=="phalanx")' internal/seed/data/statuses.json
//
// ⚠️ **It is NOT fixed by merging them.** They are earned on different axes and
// they STACK (DAT-002 decision 4), so one status under two names would halve what
// a stacked single-element column gets — a balance change wearing a tidy-up,
// which is the repair the item refuses by name. Re-pointing either needs engine
// code the data cannot reach: the effect vocabulary a composition grant can
// legally carry has nine slots and the shipped table uses all nine, so a genuinely
// new bonus effect needs Go under internal/core and, in two cases, a written-down
// refusal reversed. That derivation is in DAT-008.
//
// Named here rather than logged, which is the other way the two tests in
// areaboard_test.go go, and the difference is what is being held. Those log a
// MEASUREMENT whose right value is undecided; this holds a RULE already settled —
// DAT-002 decision 5, author's call 2026-09-04 — with exactly one known
// violation, and the item asks for an assertion in as many words ("the next
// collision fails"). A t.Logf cannot fail.
//
// The pin runs BOTH ways, which is the property a log cannot have: a THIRD bonus
// arriving with a fourth copy of defence fails, and the day the collision is
// resolved this exception fails too, so it cannot outlive what it excuses. That
// is the shape TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier above
// already uses for the other accepted gap in this table.
var acceptedEffectCollision = []string{"ground_root", "same_column"}

// fingerprint is what a grant COMES TO on the board: every field of the kind
// except the two that are labels.
//
// Three decisions here, each a branch somebody will later want to simplify.
//
// ⚠️ **%#v over a hand-written field list.** status.Kind grows: PierceShare,
// WardShare, HealShare and PoolPower all arrived after the type existed, and a
// hand-written list would have gone on comparing the fields it was written for
// while a new effect slipped past it — the "two windows do not cover a growing
// list" shape this repository has broken three times. The price of %#v is the
// mirror risk, and it is real: a future field that is a LABEL rather than an
// effect has to be zeroed below, or two effect-identical statuses stop colliding
// because one of them has a display name. Name and ID are zeroed for exactly
// that reason (a name is what internal/i18n draws, not what the board feels).
//
// ⚠️ **Modifier order is normalised.** Two statuses carrying the same terms in
// either order are the same effect. No shipped status carries two modifiers at
// all, so this branch is latent on the shipped data — a fixture hiding a branch,
// which this repository has paid for five times — and
// TestTwoKindsWithTheSameTermsInEitherOrderFingerprintAlike exercises it on
// hand-built kinds so the sort is not carried unexecuted. An empty slice is
// flattened to a nil one for the same reason: `"modifiers": []` and no modifiers
// key are the same effect and %#v spells them differently.
//
// ⚠️ **Stacks are deliberately EXCLUDED**, and the caller never passes them in.
// Decision 5 forbids two bonuses "coming to the same thing", not coming to the
// same amount. Keying on the grant's stack count as well would let the shipped
// collision be dissolved by editing `bedrock` from one stack to three — a tidy-up
// wearing a fix, which is precisely what DAT-008 refuses. What this holds is the
// KIND of effect, never its size.
func fingerprint(kind status.Kind) string {
	kind.ID, kind.Name = "", ""
	if len(kind.Modifiers) == 0 {
		kind.Modifiers = nil
	} else {
		kind.Modifiers = slices.Clone(kind.Modifiers)
		slices.SortFunc(kind.Modifiers, func(a, b modifier.Modifier) int {
			return cmp.Or(
				cmp.Compare(a.Target, b.Target),
				cmp.Compare(a.Mode, b.Mode),
				cmp.Compare(a.Amount, b.Amount),
			)
		})
	}
	return fmt.Sprintf("%#v", kind)
}

// TestNoTwoBonusesGrantTheSameEffect is DAT-002 decision 5 held as a property:
// each bonus must do something no other bonus does, because two bonuses that come
// to the same thing under different words are the "two callers wording one
// choice" mistake at the level of a feature instead of a string. A player holding
// both cannot tell them apart on the board — the effects column reads
// `phalanx x2 (always), bedrock x2 (always)` and both lines mean *more defence*.
//
// It is the effect-level twin of internal/i18n.LogGlossCollisions, which refuses
// two KINDS claiming one id. This one refuses two BONUSES claiming one effect,
// and it borrows that function's discipline of sorting before it emits: an order
// that reaches an output may not come out of a map.
//
// ⚠️ **Scoped to the bonus book, not to statuses.json**, and that is not an
// accident of convenience. The same grouping over all forty-three shipped kinds
// finds a second group — `swelter == verdure == moisture`, all
// `{"category":"reserve","max_stacks":999,"duration":5}` — which is NOT a
// collision under decision 5: reserve counters are told apart by which skill
// spends them and no bonus grants any of them. A guard written over the status
// book would be red on shipped data for a reason the rule says nothing about.
//
// ⚠️ **What it does not hold, said plainly.** It catches the byte-identical case
// and nothing softer. A bonus granting defence +100‰ AND dodge +100‰ fingerprints
// differently while still reading to a player as "tougher", and no test can
// adjudicate that. Decision 5's "state what no shipped bonus already does" stays
// a sentence a PR has to write; this is the floor under it, not a replacement.
//
// The set is derived from the data, never from a written-down list of ids, for
// the reason every other property in this file is: a list goes stale the day a
// bonus is authored, and it goes stale silently.
func TestNoTwoBonusesGrantTheSameEffect(t *testing.T) {
	bonuses, err := seed.BonusBook()
	if err != nil {
		t.Fatalf("load the bonuses: %v", err)
	}
	statuses := mustStatuses(t)

	claimed := map[string][]string{}
	// carriers is what each effect is CALLED, kept only so a failure can name the
	// statuses rather than print a Go-syntax struct at somebody.
	carriers := map[string][]string{}
	var granted int
	for _, held := range bonuses.All() {
		for _, rung := range held.Rungs {
			for _, grant := range rung.Grants {
				kind, err := statuses.Lookup(grant.Status)
				if err != nil {
					t.Fatalf("bonus %q at rung %d grants %q: %v", held.ID, rung.At, grant.Status, err)
				}
				granted++
				effect := fingerprint(kind)
				if !slices.Contains(claimed[effect], held.ID) {
					claimed[effect] = append(claimed[effect], held.ID)
				}
				if !slices.Contains(carriers[effect], kind.ID) {
					carriers[effect] = append(carriers[effect], kind.ID)
				}
			}
		}
	}
	// Fail loudly when the walk finds nothing, the guard both tests in
	// areaboard_test.go carry: a green run that resolved no grant at all has held
	// nothing down.
	if granted == 0 {
		t.Fatal("no bonus grants any status, so this test measured nothing: " +
			"the bonus book has no ladder to walk")
	}

	accepted := false
	for _, effect := range slices.Sorted(maps.Keys(claimed)) {
		group := slices.Clone(claimed[effect])
		if len(group) < 2 {
			continue
		}
		slices.Sort(group)
		if slices.Equal(group, acceptedEffectCollision) {
			accepted = true
			continue
		}
		t.Errorf("bonuses %s all resolve to one effect (granted as %s): decision 5 asks each "+
			"bonus to do something no other bonus does, and a player holding both is shown two "+
			"lines that mean the same thing",
			strings.Join(group, ", "), strings.Join(carriers[effect], " / "))
	}
	if !accepted {
		t.Errorf("%s no longer resolve to the same effect: DAT-008 is closed, "+
			"so delete this exception rather than leaving it to excuse nothing",
			strings.Join(acceptedEffectCollision, " and "))
	}
}

// TestTwoKindsWithTheSameTermsInEitherOrderFingerprintAlike exercises the branch
// of fingerprint no shipped data can reach.
//
// No status in statuses.json carries two modifiers, so the sort inside
// fingerprint runs over a list of at most one term on every real input and its
// comparator is never asked a question. An assertion carried by data that cannot
// exercise it is a fixture hiding a branch, which is why the kinds here are built
// by hand rather than loaded.
//
// The second half is the control that makes the first half a measurement: two
// kinds whose terms genuinely differ must NOT fingerprint alike. Without it a
// fingerprint that returned a constant would pass the ordering arm.
func TestTwoKindsWithTheSameTermsInEitherOrderFingerprintAlike(t *testing.T) {
	tougher := modifier.Modifier{Target: modifier.Defense, Mode: modifier.Percent, Amount: 100}
	faster := modifier.Modifier{Target: modifier.Speed, Mode: modifier.Percent, Amount: 50}

	oneOrder := status.Kind{
		ID: "first", Name: "một", Category: status.Buff, MaxStacks: 2, Permanent: true,
		Modifiers: []modifier.Modifier{tougher, faster},
	}
	theOther := status.Kind{
		ID: "second", Name: "hai", Category: status.Buff, MaxStacks: 2, Permanent: true,
		Modifiers: []modifier.Modifier{faster, tougher},
	}
	if fingerprint(oneOrder) != fingerprint(theOther) {
		t.Errorf("two kinds carrying %v and %v in opposite order fingerprint differently: "+
			"a bonus could dodge the distinctness guard by writing its terms the other way up",
			tougher, faster)
	}

	weaker := status.Kind{
		ID: "third", Name: "ba", Category: status.Buff, MaxStacks: 2, Permanent: true,
		Modifiers: []modifier.Modifier{tougher, {Target: modifier.Speed, Mode: modifier.Percent, Amount: 500}},
	}
	if fingerprint(oneOrder) == fingerprint(weaker) {
		t.Error("a kind whose speed term is ten times another's fingerprints the same: " +
			"fingerprint is not reading the amounts, so the ordering arm above measured nothing")
	}
}
