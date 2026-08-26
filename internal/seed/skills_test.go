package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
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
		if current.Range >= 4 {
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

// TestADetonateIsWorthLessThanItsBreakEven is the pricing rule for a burst that
// consumes a status: it may beat leaving the ticks alone, but not by so much that
// applying the status and immediately detonating it is the only line worth
// playing.
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
		// What the ticks would have been worth if left alone, plus what a plain
		// attack would have dealt with the same turn.
		perTick := rules.Damage(attackerAttack, referenceDefense, kind.TickPower, neutralAffinity)
		forgone := perTick * int64(current.Requires.MinStacks) * int64(kind.Duration)
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
	got := skillReport(mustSkills(t), mustStatuses(t), mustBook(t), mustRules(t))
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

func skillReport(book *skill.Book, statuses *status.Book, patterns *pattern.Book, rules combat.Rules) string {
	var b strings.Builder
	fmt.Fprintf(&b, "damage is against %d defence at %d attack, a neutral matchup and no accuracy stat\n",
		referenceDefense, attackerAttack)
	b.WriteString("a cooldown counts the caster's own turns, the same unit statuses are timed in\n\n")

	b.WriteString("== skills ==\n")
	// The pierce column earns its width the moment one skill has a value: the
	// damage figure beside it is measured against the armour that skill leaves
	// standing, so without the column the table holds two skills of the same
	// power and different damage and cannot say why.
	b.WriteString("skill           element   tgt    rng  shape          power  hits  total   acc   cd   prc   damage\n")
	for _, current := range book.Skills() {
		damage := int64(0)
		if current.Power > 0 {
			damage = rules.Total(combat.Hit{
				Scaling: attackerAttack, Multiplier: current.Power, Strikes: current.StrikeCount(),
				Affinity: neutralAffinity, Defense: referenceDefense, Pierce: current.Pierce,
			})
		}
		fmt.Fprintf(&b, "%-16s%-10s%-6s%4d  %-14s%5d%6d%7d%6d%5d%6d%9d\n",
			current.ID, current.Element, current.Target, current.Range, current.Pattern,
			current.Power, current.StrikeCount(), current.TotalPower(),
			current.Accuracy, current.Cooldown, current.Pierce, damage)
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
	// than listed with three dashes: "free to anybody" is the common case and
	// the majority of the book, so printing it would bury the four rows that
	// are the point. The header says so instead.
	b.WriteString("\n== who may carry, a skill absent here is free to anybody ==\n")
	b.WriteString("skill           elements   archetypes        characters          species\n")
	for _, current := range book.Skills() {
		if current.Restrict == nil {
			continue
		}
		fmt.Fprintf(&b, "%-16s%-11s%-18s%-20s%s\n", current.ID,
			allowlist(current.Restrict.ElementNames()),
			allowlist(current.Restrict.Archetypes),
			allowlist(current.Restrict.Characters),
			allowlist(current.Restrict.SpeciesNames()))
	}

	// The "needs" column is a sentence rather than a status id because a
	// condition stopped being one thing to name: it may read a status, or how
	// hurt the target is, or both, and a column headed "status" with a health
	// share under it would be the report lying about what it measured.
	b.WriteString("\n== conditional amplifiers ==\n")
	b.WriteString("skill           needs                       stacks   power   amplified   damage   amplified   gain\n")
	for _, current := range book.Skills() {
		if current.Requires == nil {
			continue
		}
		against := current.Requires.Satisfying()
		plain := rules.Damage(attackerAttack, referenceDefense, current.Power, neutralAffinity)
		amplified := rules.Damage(attackerAttack, referenceDefense,
			current.PowerAgainst(against), neutralAffinity)
		note := ""
		if current.Requires.Consume {
			note = " (consumes)"
		}
		// A stack count belongs to a status, so a condition that names none
		// prints a dash rather than the 0 the field happens to hold.
		stacks := "-"
		if current.Requires.ReadsStatus() {
			stacks = strconv.Itoa(current.Requires.MinStacks)
		}
		fmt.Fprintf(&b, "%-16s%-28s%7s%8d%12d%9d%12d%7s%s\n",
			current.ID, conditionReads(current.Requires), stacks,
			current.Power, current.PowerAgainst(against),
			plain, amplified, ratio(amplified, plain), note)
	}

	b.WriteString("\n== what a detonate gives up ==\n")
	b.WriteString("skill           status   ticks forgone   a plain attack   alternative   burst    ratio\n")
	for _, current := range book.Skills() {
		if current.Requires == nil || !current.Requires.Consume {
			continue
		}
		kind, err := statuses.Lookup(current.Requires.Status)
		if err != nil {
			continue
		}
		perTick := rules.Damage(attackerAttack, referenceDefense, kind.TickPower, neutralAffinity)
		forgone := perTick * int64(current.Requires.MinStacks) * int64(kind.Duration)
		plain := rules.Damage(attackerAttack, referenceDefense, 1000, neutralAffinity)
		burst := rules.Damage(attackerAttack, referenceDefense,
			current.PowerAgainst(skill.Carrying(kind.MaxStacks)), neutralAffinity)
		fmt.Fprintf(&b, "%-16s%-9s%15d%17d%14d%8d%9s\n",
			current.ID, kind.ID, forgone, plain, forgone+plain, burst, ratio(burst, forgone+plain))
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
// to survive a save it was not part of. Four blocks are in that position today —
// requires, strips, scaling and self_applies — and the shipped set uses all
// four, which is what makes this test worth more than the fixture version of it.
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
	for _, axis := range []string{"elements", "archetypes", "characters", "species"} {
		if len(axes[axis]) > 0 {
			named = append(named, axis+" "+allowlist(axes[axis]))
		}
	}
	if len(named) == 0 {
		return "nothing"
	}
	return strings.Join(named, ", ")
}
