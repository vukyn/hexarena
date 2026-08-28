package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/testfixture"
)

func mustSkills(t *testing.T) *skill.Book {
	t.Helper()
	book, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load shipped skills: %v", err)
	}
	return book
}

// benchSkills parses the nineteen-skill bench, which is what a mechanism test
// wants: every element, every shape, both conditions, multi-strike, area splash,
// a cleanse, a dispel and a shield.
func benchSkills(t *testing.T) *skill.Book {
	t.Helper()
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load shapes: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	book, err := skill.ParseBook([]byte(`{"skills":`+testfixture.Skills+`}`),
		skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the bench skills: %v", err)
	}
	return book
}

// TestTheBenchCoversTheMechanics keeps the bench honest: if a mechanic has no
// skill exercising it, nothing downstream is really covered.
//
// It reads the bench rather than the shipped book, because the shipped book is
// the cast's now and holds whatever the authored characters need -- four grass
// skills today. Asking that to cover every mechanism would either be a false
// pass or a standing obstacle to authoring.
func TestTheBenchCoversTheMechanics(t *testing.T) {
	book := benchSkills(t)
	skills := book.Skills()
	if len(skills) < 12 {
		t.Fatalf("the book holds %d skills, want a usable seed set", len(skills))
	}
	var (
		multiStrike, area, amplifier, detonate bool
		cleanse, dispel, shield, selfBuff      bool
		guaranteed, speedScaled, longRange     bool
		gradient                               bool
	)
	elements := make(map[element.Element]bool)
	sides := make(map[skill.Side]bool)
	for _, current := range skills {
		elements[current.Element] = true
		sides[current.Target] = true
		if current.StrikeCount() > 1 {
			multiStrike = true
		}
		if current.Pattern != "single" {
			area = true
		}
		if current.Requires != nil {
			amplifier = true
			if current.Requires.Consume {
				detonate = true
			}
		}
		if current.SelfGradient != nil {
			gradient = true
		}
		if current.Strips != nil {
			harmful := false
			for _, category := range current.Strips.Categories {
				if category.Harmful() {
					harmful = true
				}
			}
			if harmful {
				cleanse = true
			} else {
				dispel = true
			}
		}
		for _, application := range append(current.Applies, current.SelfApplies...) {
			if application.Status == "block" {
				shield = true
			}
		}
		if current.Target == skill.Self && len(current.SelfApplies) > 0 {
			selfBuff = true
		}
		if current.Guaranteed() {
			guaranteed = true
		}
		if current.Scaling.Stat == progression.Speed {
			speedScaled = true
			if current.Scaling.Source != combat.BaseStat {
				t.Errorf("skill %q scales off the current speed, which compounds with the turn economy",
					current.ID)
			}
		}
		// Three ranks is the whole of a side, so reaching the back line is a
		// range of three now rather than the four it took when reach was
		// measured in cells.
		if current.Range >= hex.FormationCols {
			longRange = true
		}
	}
	for _, testCase := range []struct {
		name    string
		covered bool
	}{
		{"a multi strike skill", multiStrike},
		{"an area skill", area},
		{"a conditional amplifier", amplifier},
		{"a detonate", detonate},
		{"a gradient off the caster's own health", gradient},
		{"a cleanse", cleanse},
		{"a dispel", dispel},
		{"a shield", shield},
		{"a self buff", selfBuff},
		{"a skill that cannot miss", guaranteed},
		{"a speed scaled skill", speedScaled},
		{"a skill reaching the back line", longRange},
	} {
		if !testCase.covered {
			t.Errorf("the seed set has no %s", testCase.name)
		}
	}
	if len(elements) < 6 {
		t.Errorf("the seed set uses %d elements, want a spread across the chart", len(elements))
	}
	for _, side := range []skill.Side{skill.Enemy, skill.Ally, skill.Self} {
		if !sides[side] {
			t.Errorf("no shipped skill targets %s", side)
		}
	}
}

// TestEverySkillsReachIsUsable stops a skill declaring a range or a shape that
// can never do what it says on the board it is played on.
func TestEverySkillsReachIsUsable(t *testing.T) {
	book, patterns := mustSkills(t), mustBook(t)
	for _, current := range book.Skills() {
		if current.Target == skill.Self {
			continue
		}
		shape, err := patterns.Lookup(current.Pattern)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		reachable, widest := false, 0
		for _, from := range hex.SideCells(hex.SideAlly) {
			for _, to := range hex.Cells() {
				if from.DistanceTo(to) > current.Range {
					continue
				}
				wantSide := hex.SideEnemy
				if current.Target == skill.Ally {
					wantSide = hex.SideAlly
				}
				if to.Side() != wantSide {
					continue
				}
				reachable = true
				if covered := len(shape.Targets(to)); covered > widest {
					widest = covered
				}
			}
		}
		if !reachable {
			t.Errorf("skill %q at range %d can never reach a %s cell", current.ID, current.Range, current.Target)
		}
		if widest != shape.MaxTargets() {
			t.Errorf("skill %q uses the %q shape but can only ever cover %d of its %d cells",
				current.ID, shape.Name, widest, shape.MaxTargets())
		}
	}
}

// forgoneBy is what a detonate throws away: the status it consumes, priced in
// the damage it would have been worth over the rest of its life if left alone.
//
// ⚠️ There are two currencies, and a detonate has to be priced in the one it
// actually spends. A damage-over-time status is worth its remaining **ticks**,
// which land whatever the attacker does next. A stat debuff ticks for nothing
// and is worth the **extra damage its holder takes from ordinary attacks** while
// it is up, so `expose` is priced by hitting the lowered defence and the real
// one with the same plain attack and charging the difference for every turn the
// debuff had left.
//
// The second reading did not exist until a detonate wanted a status that was not
// a damage-over-time, and its absence was not a missing feature but a wrong
// answer: TickPower is nought for a stat debuff, so the old arithmetic priced
// consuming one at **nothing at all** and would have waved through a burst of any
// size. `dragon_drive` is the skill that found it.
//
// ok is false for a status this cannot price, rather than nought, because nought
// is indistinguishable from "gives up nothing" and that is exactly the lie this
// is here to stop.
func forgoneBy(t *testing.T, kind status.Kind, stacks int, rules combat.Rules) (int64, bool) {
	t.Helper()
	turns := int64(kind.Duration)
	if kind.TickPower > 0 {
		perTick := rules.Damage(attackerAttack, referenceDefense, kind.TickPower, neutralAffinity)
		return perTick * int64(stacks) * turns, true
	}
	carried := status.Set{}
	for range stacks {
		carried.Apply(kind, 0)
	}
	var reference progression.Values
	reference[progression.Defense] = referenceDefense
	lowered := carried.Modifiers().Stats(reference, mustCeilings(t), mustBounds(t))
	if lowered[progression.Defense] == referenceDefense {
		return 0, false
	}
	plain := rules.Damage(attackerAttack, referenceDefense, 1000, neutralAffinity)
	against := rules.Damage(attackerAttack, lowered[progression.Defense], 1000, neutralAffinity)
	return (against - plain) * turns, true
}

// TestADetonateIsWorthLessThanItsBreakEven is the pricing rule for a burst that
// consumes a status: it may beat leaving the status alone, but not by so much
// that applying it and immediately detonating it is the only line worth playing.
func TestADetonateIsWorthLessThanItsBreakEven(t *testing.T) {
	book, statuses, rules := mustSkills(t), mustStatuses(t), mustRules(t)
	for _, current := range book.Skills() {
		if current.Requires == nil || !current.Requires.Consume {
			continue
		}
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil {
			t.Errorf("skill %q: %v", current.ID, err)
			continue
		}
		burst := rules.Damage(attackerAttack, referenceDefense,
			current.PowerAgainst(skill.Carrying(kind.MaxStacks)), neutralAffinity)
		// What the status would have been worth if left alone, plus what a plain
		// attack would have dealt with the same turn.
		forgone, priced := forgoneBy(t, kind, current.Requires.MinStacks, rules)
		if !priced {
			t.Errorf("skill %q consumes %q, which has neither ticks nor a defence term, so what "+
				"detonating gives up cannot be priced and the burst would be bounded by nothing",
				current.ID, kind.ID)
			continue
		}
		plain := rules.Damage(attackerAttack, referenceDefense, 1000, neutralAffinity)
		alternative := forgone + plain
		if burst <= alternative {
			t.Errorf("skill %q bursts for %d against an alternative worth %d, so detonating is never right",
				current.ID, burst, alternative)
		}
		if burst > alternative*2 {
			t.Errorf("skill %q bursts for %d against an alternative worth %d, over twice as good",
				current.ID, burst, alternative)
		}
	}
}

func TestSkillBookGolden(t *testing.T) {
	got := skillReport(t, mustSkills(t), mustStatuses(t), mustBook(t), mustRules(t))
	path := filepath.Join("testdata", "skills.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("skill report differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// allowlist is one column of a restriction: the names it admits, or a dash for
// a list that is empty and therefore admits everybody.
func allowlist(names []string) string {
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, " ")
}

// conditionReads is what a skill's condition asks about its target, written out.
//
// Both halves when it asks both, because "and" is the rule: a condition naming a
// status and a threshold holds only when the target satisfies each, and a report
// listing one of them would make a narrow skill read as a broad one.
func conditionReads(condition *skill.Condition) string {
	parts := make([]string, 0, 2)
	if condition.ReadsStatus() {
		parts = append(parts, condition.Status)
	}
	if condition.ReadsHealth() {
		parts = append(parts, fmt.Sprintf("health <=%d per mille", condition.BelowHealth))
	}
	return strings.Join(parts, " and ")
}

func skillReport(t *testing.T, book *skill.Book, statuses *status.Book, patterns *pattern.Book, rules combat.Rules) string {
	var b strings.Builder
	fmt.Fprintf(&b, "damage is against %d defence at %d attack, a neutral matchup and no accuracy stat\n",
		referenceDefense, attackerAttack)
	b.WriteString("a cooldown counts the caster's own turns, the same unit statuses are timed in\n\n")

	b.WriteString("== skills ==\n")
	// The pierce column earns its width the moment one skill has a value: the
	// damage figure beside it is measured against the armour that skill leaves
	// standing, so without the column the table holds two skills of the same
	// power and different damage and cannot say why.
	// The crit column is here while every entry in it is nought, and that is the
	// point: a property the design record cannot show is a property nobody
	// compares two skills by, so the column arrives with the mechanic rather
	// than with the first skill to use it.
	b.WriteString("skill           element   tgt    rng  shape          power  hits  total   acc   cd   prc  crit   damage\n")
	for _, current := range book.Skills() {
		damage := int64(0)
		if current.Power > 0 {
			damage = rules.Total(combat.Hit{
				Scaling: attackerAttack, Multiplier: current.Power, Strikes: current.StrikeCount(),
				Affinity: neutralAffinity, Defense: referenceDefense, Pierce: current.Pierce,
			})
		}
		fmt.Fprintf(&b, "%-16s%-10s%-6s%4d  %-14s%5d%6d%7d%6d%5d%6d%6d%9d\n",
			current.ID, current.Element, current.Target, current.Range, current.Pattern,
			current.Power, current.StrikeCount(), current.TotalPower(),
			current.Accuracy, current.Cooldown, current.Pierce, current.Crit, damage)
	}

	b.WriteString("\n== statuses inflicted, chance is fixed and no stat moves it ==\n")
	b.WriteString("skill           status           stacks   chance   accuracy   lands per cast\n")
	for _, current := range book.Skills() {
		for _, application := range current.Applies {
			landed := current.Accuracy * application.Chance / combat.PermilleBase
			fmt.Fprintf(&b, "%-16s%-17s%7d%9d%11d%17d\n",
				current.ID, application.Status, application.Stacks,
				application.Chance, current.Accuracy, landed)
		}
		for _, application := range current.SelfApplies {
			fmt.Fprintf(&b, "%-16s%-17s%7d%9d%11d%17s\n",
				current.ID, application.Status+" (self)", application.Stacks,
				application.Chance, current.Accuracy, "-")
		}
	}

	// A restriction changes who may play a skill at all, which is a louder fact
	// about the shipped book than any number beside it, and it was the one rule
	// this report could not state. The unrestricted skills are left out rather
	// than listed with dashes: the header says what their absence means.
	//
	// ⚠️ That reading flipped once and the report did not notice. "Free to
	// anybody" was the common case and the majority of the book, and it stopped
	// being either when every skill was given the work it comes out of, so the
	// listed rows are now the book and the absent ones the exception. A column
	// missing from this header is a restriction the design record cannot show:
	// origins arrived and thirty-two rows appeared here with nothing but dashes
	// across them until it was added.
	b.WriteString("\n== who may carry, a skill absent here is free to anybody ==\n")
	b.WriteString("skill           elements   archetypes        characters          species   origins\n")
	for _, current := range book.Skills() {
		if current.Restrict == nil {
			continue
		}
		fmt.Fprintf(&b, "%-16s%-11s%-18s%-20s%-10s%s\n", current.ID,
			allowlist(current.Restrict.ElementNames()),
			allowlist(current.Restrict.Archetypes),
			allowlist(current.Restrict.Characters),
			allowlist(current.Restrict.SpeciesNames()),
			allowlist(current.Restrict.OriginNames()))
	}

	// The "needs" column is a sentence rather than a status id because a
	// condition stopped being one thing to name: it may read a status, or how
	// hurt the target is, or both, and a column headed "status" with a health
	// share under it would be the report lying about what it measured.
	// Both conditions, because a table that read only one of them would be the
	// report telling an author their skill has no amplifier while the engine
	// amplifies it. The "whose" column is what a reader needs to tell "while the
	// target is poisoned" from "while I am cornered" -- the two read the same in
	// the needs column and mean opposite things.
	b.WriteString("\n== conditional amplifiers ==\n")
	b.WriteString("skill           whose    needs                       stacks   power   amplified   damage   amplified   gain\n")
	for _, current := range book.Skills() {
		for _, side := range []struct {
			whose     string
			condition *skill.Condition
		}{
			{"target", current.Requires},
			{"caster", current.SelfRequires},
		} {
			if side.condition == nil {
				continue
			}
			plain := rules.Damage(attackerAttack, referenceDefense, current.Power, neutralAffinity)
			amplified := rules.Damage(attackerAttack, referenceDefense,
				current.Power+side.condition.BonusPower, neutralAffinity)
			note := ""
			if side.condition.Consume {
				note = " (consumes)"
			}
			// A stack count belongs to a status, so a condition that names none
			// prints a dash rather than the 0 the field happens to hold.
			stacks := "-"
			if side.condition.ReadsStatus() {
				stacks = strconv.Itoa(side.condition.MinStacks)
			}
			fmt.Fprintf(&b, "%-16s%-9s%-28s%7s%8d%12d%9d%12d%7s%s\n",
				current.ID, side.whose, conditionReads(side.condition), stacks,
				current.Power, current.Power+side.condition.BonusPower,
				plain, amplified, ratio(amplified, plain), note)
		}
	}

	// A section of its own rather than a row in the amplifiers above, because the
	// column that table is built around is "needs", and a gradient needs nothing.
	// It is always on and always partly on, which is the whole difference between
	// the two features — folding it in would give it a blank in the one column
	// that says what a conditional skill is conditional on.
	b.WriteString("\n== a gradient off the caster's own health ==\n")
	b.WriteString("what the caster's own wounds add, at the bottom of the bar\n")
	b.WriteString("skill           at empty   power   at the bottom   damage   at the bottom    gain\n")
	for _, current := range book.Skills() {
		if current.SelfGradient == nil {
			continue
		}
		bottom := current.Power * (scale.Base + current.SelfGradient.AtEmpty) / scale.Base
		plain := rules.Damage(attackerAttack, referenceDefense, current.Power, neutralAffinity)
		hurt := rules.Damage(attackerAttack, referenceDefense, bottom, neutralAffinity)
		fmt.Fprintf(&b, "%-16s%9d%8d%16d%9d%16d%8s\n",
			current.ID, current.SelfGradient.AtEmpty, current.Power, bottom, plain, hurt,
			ratio(hurt, plain))
	}

	b.WriteString("\n== what a detonate gives up ==\n")
	b.WriteString("the ticks of a damage-over-time, or the extra damage a stat debuff was letting through\n")
	b.WriteString("skill           status   spends     forgone   a plain attack   alternative   burst    ratio\n")
	for _, current := range book.Skills() {
		if current.Requires == nil || !current.Requires.Consume {
			continue
		}
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil {
			continue
		}
		forgone, priced := forgoneBy(t, kind, current.Requires.MinStacks, rules)
		spends := "ticks"
		if kind.TickPower == 0 {
			spends = "defence"
		}
		if !priced {
			spends = "unpriced"
		}
		plain := rules.Damage(attackerAttack, referenceDefense, 1000, neutralAffinity)
		burst := rules.Damage(attackerAttack, referenceDefense,
			current.PowerAgainst(skill.Carrying(kind.MaxStacks)), neutralAffinity)
		fmt.Fprintf(&b, "%-16s%-9s%-9s%9d%17d%14d%8d%9s\n",
			current.ID, kind.ID, spends, forgone, plain, forgone+plain, burst, ratio(burst, forgone+plain))
	}

	b.WriteString("\n== cleanses and dispels ==\n")
	b.WriteString("skill           categories                    stacks\n")
	for _, current := range book.Skills() {
		if current.Strips == nil {
			continue
		}
		names := make([]string, 0, len(current.Strips.Categories))
		for _, category := range current.Strips.Categories {
			names = append(names, category.String())
		}
		fmt.Fprintf(&b, "%-16s%-30s%6d\n", current.ID, strings.Join(names, ", "), current.Strips.Stacks)
	}

	b.WriteString("\n== timed effects ==\n")
	b.WriteString("status      category      stacks   turns   tick power   tick   over its life   modifiers\n")
	for _, kind := range statuses.Kinds() {
		tick, life := int64(0), int64(0)
		if kind.TickPower > 0 {
			tick = rules.Damage(attackerAttack, referenceDefense, kind.TickPower, neutralAffinity)
			// One stack applied per turn until the cap, then the remaining
			// duration at full stacks.
			for turn := 1; turn <= kind.MaxStacks+kind.Duration-1; turn++ {
				stacks := turn
				if stacks > kind.MaxStacks {
					stacks = kind.MaxStacks
				}
				life += tick * int64(stacks)
			}
		}
		terms := make([]string, 0, len(kind.Modifiers))
		for _, term := range kind.Modifiers {
			terms = append(terms, term.String())
		}
		joined := strings.Join(terms, ", ")
		if joined == "" {
			joined = "-"
		}
		fmt.Fprintf(&b, "%-12s%-14s%7d%8d%13d%7d%16d   %s\n",
			kind.ID, kind.Category, kind.MaxStacks, kind.Duration,
			kind.TickPower, tick, life, joined)
	}

	b.WriteString("\n== area shapes a skill can choose ==\n")
	b.WriteString("shape          width   best aim   worst aim\n")
	for _, shape := range patterns.Patterns() {
		best, worst := 0, 99
		for _, centre := range hex.SideCells(hex.SideEnemy) {
			covered := len(shape.Targets(centre))
			if covered > best {
				best = covered
			}
			if covered < worst {
				worst = covered
			}
		}
		fmt.Fprintf(&b, "%-15s%6d%11d%12d\n", shape.Name, shape.MaxTargets(), best, worst)
	}
	return b.String()
}

// TestTheShippedSkillBookSurvivesBeingWritten is the round trip on the real
// data rather than on a fixture.
//
// It matters because cmd/hexforge now writes this file: it rewrites the whole
// book on every addition, so a field the authoring form does not ask about has
// to survive a save it was not part of. Six blocks are in that position today —
// requires, self_requires, self_gradient, strips, scaling and summons — and the
// shipped set uses every one, which is what makes this test worth more than the
// fixture version of it.
func TestTheShippedSkillBookSurvivesBeingWritten(t *testing.T) {
	book := mustSkills(t)
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("patterns: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	raw, err := book.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, err := skill.ParseBook(raw, skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("the written book does not load: %v", err)
	}
	if !reflect.DeepEqual(reparsed.Skills(), book.Skills()) {
		t.Error("writing the shipped skill book and reading it back changed it")
	}
	// Every block the form does not author is still there afterwards, named
	// rather than counted, so a skill losing one is a failure that says which.
	for _, current := range reparsed.Skills() {
		before, err := book.Lookup(current.ID)
		if err != nil {
			t.Fatalf("lookup %q: %v", current.ID, err)
		}
		switch {
		case (current.Requires == nil) != (before.Requires == nil):
			t.Errorf("%q lost or gained its condition", current.ID)
		case (current.SelfRequires == nil) != (before.SelfRequires == nil):
			t.Errorf("%q lost or gained the condition it reads about itself", current.ID)
		case (current.SelfGradient == nil) != (before.SelfGradient == nil):
			t.Errorf("%q lost or gained its gradient", current.ID)
		case (current.Strips == nil) != (before.Strips == nil):
			t.Errorf("%q lost or gained its cleanse", current.ID)
		case current.Scaling != before.Scaling:
			t.Errorf("%q came back scaling off %v, want %v", current.ID, current.Scaling, before.Scaling)
		case len(current.SelfApplies) != len(before.SelfApplies):
			t.Errorf("%q came back with %d self applications, want %d",
				current.ID, len(current.SelfApplies), len(before.SelfApplies))
		}
	}
}

// TestABodyBoundSkillIsRestricted is the rule a name can break without any
// parser noticing: a skill whose name describes a *body* rather than an effect
// may not be free for anyone to carry.
//
// "withdraw" was the one that showed it. Squirtle pulls into its shell, so the
// name read perfectly on the character it was written for and would have read
// as nonsense the first time a Machop took it -- and nothing refused it,
// because the engine checks elements and allowlists and has no opinion about
// anatomy. The two ways out are different rules, not a preference: a skill
// whose *effect* is general gets a name that is general too, which is why
// "withdraw" is now glossed as a stance rather than as a shell, and a skill
// whose identity is the point keeps its name and carries a restriction.
//
// The list is hand-written on purpose, the way the archetype design table is.
// There is no property that separates "cắm rễ" from "xoáy nước" -- somebody has
// to read the name and decide -- so this is the place that decision is recorded,
// and a shipped skill added to the list without a restriction fails here.
//
// Each row names the *axis* rather than only asking for some restriction, and
// that is the half of this test that took two attempts. "Anybody may carry it"
// is one failure; "the wrong list keeps it" is a different one that reads as a
// pass, and the difference is the whole subject here -- the lineage pair sat on
// a character allowlist for exactly as long as nobody could say which list it
// should have been on.
//
// Why the two axes differ, since the four rows are two of each:
//
//   - species, for a lineage. Being a dragon outlives the character that first
//     had it, so a character allowlist said "only this one may carry it" when it
//     meant "only a dragon may".
//   - species, for the two that mean a body too. `ingrain` and `synthesis` read
//     as roots and photosynthesis, so a plant is what they mean and grass was
//     only ever a proxy for it -- a grass-element construct with no roots could
//     take both, and nothing said no. They were kept on the proxy because
//     cast.resolveArchetype refuses a species-restricted skill in a preset, and
//     moving them cost the blighter kit two entries. That is the right way round:
//     a preset losing a skill is a smaller thing than modelling a body as a
//     fighting style, and scorcher had already paid the same price for the two
//     lineage skills.
func TestABodyBoundSkillIsRestricted(t *testing.T) {
	bound := map[string]struct{ why, axis string }{
		"ingrain":      {"roots, so only something that grows may take it", "species"},
		"synthesis":    {"photosynthesis, so only something that grows may take it", "species"},
		"dragon_rage":  {"a lineage, not a technique", "species"},
		"dragon_dance": {"a lineage, not a technique", "species"},
	}
	book := mustSkills(t)
	for id, expected := range bound {
		carried, err := book.Lookup(id)
		if err != nil {
			t.Errorf("the list names %q, which the shipped book does not hold: %v", id, err)
			continue
		}
		if carried.Restrict == nil {
			t.Errorf("%q names %s, but anybody may carry it", id, expected.why)
			continue
		}
		axes := map[string][]string{
			"elements":   carried.Restrict.ElementNames(),
			"archetypes": carried.Restrict.Archetypes,
			"characters": carried.Restrict.Characters,
			"species":    carried.Restrict.SpeciesNames(),
			"origins":    carried.Restrict.OriginNames(),
		}
		if len(axes[expected.axis]) == 0 {
			t.Errorf("%q names %s and should be kept by its %s, but that list is empty; it is kept by %s",
				id, expected.why, expected.axis, allowlistedAxes(axes))
		}
	}
}

// allowlistedAxes names which lists of a restriction hold anything, so a
// refusal above can say what the skill is kept by instead of only what it is
// not.
func allowlistedAxes(axes map[string][]string) string {
	// Ranged in a fixed order rather than over the map: this reaches a failure
	// message, and Go randomises map order.
	named := make([]string, 0, len(axes))
	for _, axis := range []string{"elements", "archetypes", "characters", "species", "origins"} {
		if len(axes[axis]) > 0 {
			named = append(named, axis+" "+allowlist(axes[axis]))
		}
	}
	if len(named) == 0 {
		return "nothing"
	}
	return strings.Join(named, ", ")
}

// TestEveryShippedSkillHasAFlavourClause is the wording rule, held the way the
// archetype gloss rule is: a shipped skill reaches a player, and a player reading
// "Đánh 1 mục tiêu đối phương, 110% công" is being handed the data rather than
// told what happened.
//
// The field itself is optional and stays that way — a skill being authored in the
// tool has no clause yet and should not be refused for it. What may not happen is
// a skill *shipping* without one.
func TestEveryShippedSkillHasAFlavourClause(t *testing.T) {
	for _, current := range mustSkills(t).Skills() {
		if strings.TrimSpace(current.Flavour) == "" {
			t.Errorf("%q ships with no flavour clause, so its description opens with the derived one",
				current.ID)
		}
	}
}

// bodyWords are the words a skill free for anybody to carry may not use about
// itself, and each one is a judgement rather than a rule a machine could derive.
//
// The list is hand-written for the same reason TestABodyBoundSkillIsRestricted's
// is: nothing separates "mai" from "nước" except somebody reading it and deciding
// whether every creature has one. Teeth and mouths are deliberately absent — a
// mouth is near enough universal that "há miệng phun lửa" reads on anything, and
// that judgement was already made when fire_fang and bite were left unrestricted.
var bodyWords = map[string]string{
	"mai":  "a shell",
	"vỏ":   "a shell",
	"cánh": "wings",
	"chân": "legs",
	"rễ":   "roots",
	"vảy":  "scales",
	"sừng": "horns",
	"lông": "fur or feathers",
}

// TestAFreeSkillsFlavourNamesNoBodyItMayNotHave is the hole the flavour clause
// opened in a rule that already existed.
//
// `withdraw` was renamed from "thu mai" to "thủ thế" precisely because anybody may
// carry it and a shell-less creature reading "thu mai" is nonsense — and then the
// first flavour clause written for it said "rụt hết vào trong mai", which is the
// same defect through a field the name test does not look at. A restricted skill
// is free to name whatever its restriction guarantees: `ingrain` may say roots,
// because only a plant may take it.
//
// It reads the whole clause and cannot tell whose shell is being described, so a
// body word about something *else* trips it too — `leech_seed` said the seed took
// root in its target and was refused for it. That bluntness is the right way
// round: rewording costs a few words, and a miss costs a sentence that reads as
// nonsense on somebody's screen.
func TestAFreeSkillsFlavourNamesNoBodyItMayNotHave(t *testing.T) {
	for _, current := range mustSkills(t).Skills() {
		if current.Restrict != nil {
			continue
		}
		lowered := strings.ToLower(current.Flavour)
		for word, what := range bodyWords {
			if !strings.Contains(lowered, word) {
				continue
			}
			t.Errorf("%q is free for anybody to carry and its flavour says %q, which is %s: restrict the skill or reword the clause",
				current.ID, word, what)
		}
	}
}

// countWords are the numbers a flavour clause may not spell out.
//
// skill.ParseBook already refuses a digit, and that is the check that protects
// the guarantee — but it is a check on *characters*, and "hai nhát" walks past it
// while saying exactly what "2 nhát" would have. The first clause written for
// fire_fang did precisely that, and the sentence read "Ngoạm hai nhát, ... 2 nhát,
// mỗi nhát 60% công": the authored half and the derived half saying the same thing
// twice, one of which would stop being true if the strike count moved.
//
// "một" is deliberately absent. It is the indefinite article far more often than
// it is a count — "một cột lửa", "một loạt lá" — so banning it would cost every
// clause its natural opening to catch a mistake nobody makes: a skill of one
// strike is the default, and nobody writes it out.
var countWords = []string{"hai", "ba", "bốn", "năm", "sáu", "bảy", "tám", "chín", "mười"}

// TestAFlavourClauseSpellsOutNoNumber closes the half of the digit ban that a
// character check cannot see.
func TestAFlavourClauseSpellsOutNoNumber(t *testing.T) {
	for _, current := range mustSkills(t).Skills() {
		for _, word := range strings.Fields(strings.ToLower(current.Flavour)) {
			word = strings.Trim(word, ",.;:")
			if !slices.Contains(countWords, word) {
				continue
			}
			t.Errorf("%q spells out the number %q in its flavour; every figure in a description is derived, and a written one says the same thing twice until it stops being true",
				current.ID, word)
		}
	}
}

// volleyWords are the words that describe several things leaving the hand one
// after another, and a skill that lands once may not use them.
//
// This is the half of the number ban that neither of the two checks above can
// see: skill.ParseBook refuses a digit, TestAFlavourClauseSpellsOutNoNumber
// refuses a spelled one, and "một nhúm phi tiêu" walks past both while promising
// exactly what "2 nhát" would have. The shipped clause for kunai did precisely
// that — a handful of blades thrown, above a derived half that struck once — and
// it read as the wrong weapon on top of reading as the wrong number, because a
// handful thrown at once is the shuriken standing next to it in the same kit.
//
// "chùm" is deliberately absent, on the same judgement that kept teeth out of
// bodyWords. A cluster of bubbles leaves as one puff and lands as one hit, which
// is what `bubble` says and what `bubble` does; these four are things released
// separately, and a skill that releases them separately strikes more than once.
var volleyWords = map[string]string{
	"nhúm":  "a handful",
	"loạt":  "a volley",
	"tràng": "a salvo",
	"mớ":    "a bunch",
}

// TestASingleStrikesFlavourDescribesNoVolley holds it over the shipped book.
//
// It fires on nought strikes as well as one: a skill that puts a status on
// somebody or calls somebody up does not throw a volley of anything either.
func TestASingleStrikesFlavourDescribesNoVolley(t *testing.T) {
	for _, current := range mustSkills(t).Skills() {
		if current.Strikes > 1 {
			continue
		}
		// Whole words, not substrings. Every one of these is a short syllable
		// that sits inside ordinary Vietnamese: "mớ" is inside **mới**, which is
		// as common a word as there is, and matching it as a substring rejected
		// three clauses that promise nothing. The sibling number ban already
		// splits into words for exactly this reason; this one did not, and the
		// gap only showed once a clause happened to say "mới".
		//
		// Unlike bodyWords, which is deliberately blunt — a shell named anywhere
		// in a clause is worth a second look whoever it belongs to — a volley
		// word means nothing except as its own word.
		for _, spoken := range strings.Fields(strings.ToLower(current.Flavour)) {
			spoken = strings.Trim(spoken, ",.;:—")
			what, promised := volleyWords[spoken]
			if !promised {
				continue
			}
			t.Errorf("%q lands once and its flavour says %q, which is %s: a clause that promises several throws above a derived half that strikes once says the count twice and gets it wrong both times",
				current.ID, spoken, what)
		}
	}
}

// casterWords are the ways a summon's flavour can reach for whoever called it up,
// and a summoning skill may not use any of them.
//
// A summon is the only thing in this book with a second party in its sentence, so
// it is the only place the temptation arises — and it is wrong in both directions
// at once. Where the summon is a share of its caster the comparison is *derived*,
// and printing it in the authored half is fire_fang's "hai nhát" again in another
// field. Where the summon is a fixed stat line the comparison cannot be checked at
// all: this layer holds the skill and not whoever carries it, so a clause about
// the caster is a claim about somebody nobody here has met.
//
// Both shipped Naruto summons did it and both were wrong. The clone said its
// copies carried "một phần sức của bản gốc", which is the share the sentence now
// prints as a figure. The toad was called "to hơn cả người gọi" and is not: its
// stat line has less health and less attack than the ninja who calls it, and only
// more defence — the one comparison the clause could have made was the one it did
// not.
var casterWords = []string{"người gọi", "kẻ gọi", "bản gốc", "chủ nhân"}

// TestASummonsFlavourClaimsNothingAboutItsCaster holds it over the shipped book.
func TestASummonsFlavourClaimsNothingAboutItsCaster(t *testing.T) {
	for _, current := range mustSkills(t).Skills() {
		if !current.Summons.Summons() {
			continue
		}
		lowered := strings.ToLower(current.Flavour)
		for _, word := range casterWords {
			if !strings.Contains(lowered, word) {
				continue
			}
			t.Errorf("%q summons and its flavour says %q: a share of the caster is already printed as a figure, and a fixed stat line cannot be compared to somebody this layer has never seen",
				current.ID, word)
		}
	}
}

// sharedPool is every shipped skill that deliberately belongs to no work, with
// the reason it is loose.
//
// A named list rather than a rule about names, because "does this word belong to
// one fiction" is a judgement and not a pattern: `bite` and `withdraw` are
// Pokémon moves by provenance and ordinary English by sound, so a ninja doing
// either reads as a ninja doing either. What matters is that widening the pool
// costs a line here and an argument for it, rather than happening by omission.
var sharedPool = map[string]string{
	"smokescreen": "throwing smoke to be harder to hit is not one fiction's idea",
	"bite":        "a mouth is a mouth",
	"withdraw":    "guarding rather than striking, which anything with a guard can do",
	"rapid_spin":  "spinning out of a hold, on the same terms",
	"rally":       "encouragement belongs to nobody",
	"taunt":       "drawing an attack onto yourself is a tactic, not a technique",
	"wide_guard":  "standing in front of somebody is the same tactic pointed the other way",
}

// TestEverySkillSaysWhichWorkItIsFrom is the shipped half of the origin gate.
//
// The gate itself is an allowlist, so it does nothing by default: a Naruto skill
// authored without one is carried by a Pokémon, silently, and the book still
// loads. That is the failure this catches, and it catches it by default —
// omitting the list fails, and the only way to pass without one is to say here
// why the skill belongs to nobody.
//
// ⚠️ It is not the pool being small that makes the rule work, it is the pool
// being *written down*. A version of this that exempted, say, every neutral
// skill would have exempted `rasengan` on a technicality.
func TestEverySkillSaysWhichWorkItIsFrom(t *testing.T) {
	origins := mustOrigins(t)
	for _, carried := range mustSkills(t).Skills() {
		named := carried.Restrict.OriginNames()
		why, loose := sharedPool[carried.ID]
		switch {
		case loose && len(named) > 0:
			t.Errorf("%s is in the shared pool (%s) and is also kept for %s; it cannot be both",
				carried.ID, why, strings.Join(named, " or "))
		case loose:
			continue
		case len(named) == 0:
			t.Errorf("%s names no work, so anybody out of any of them may carry it; give it restrict.origins, or add it to sharedPool with the reason it belongs to nobody",
				carried.ID)
		}
		for _, id := range named {
			if _, known := origins.Get(id); !known {
				t.Errorf("%s is kept for %q, which the origin catalog does not declare", carried.ID, id)
			}
		}
	}
	for id := range sharedPool {
		if _, err := mustSkills(t).Lookup(id); err != nil {
			t.Errorf("the shared pool names %q, which the book does not hold: %v", id, err)
		}
	}
}
