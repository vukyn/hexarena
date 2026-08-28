package seed_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/seed"
)

func mustPassives(t *testing.T) *passive.Book {
	t.Helper()
	book, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load shipped passives: %v", err)
	}
	return book
}

// TestEveryShippedTraitHasAFlavourClause is the same rule shipped skills obey,
// and a trait needs it more than a skill does.
//
// A skill's derived opening still names what it hits and for how much, so a
// skill without a clause reads flat rather than empty. A trait's derived lines
// say only what it does to a number — "always carries cứng đòn", "refuses bỏng
// outright" — and never what it is, so a trait without one arrives with nothing
// to hang the mechanism on. The trait's own authored name is the fact that most
// wants saying, and until the clause existed it was rendered nowhere in the
// sentences at all.
//
// A trait still being authored may have none, exactly as a skill may. What may
// not happen is one shipping that way.
func TestEveryShippedTraitHasAFlavourClause(t *testing.T) {
	for _, held := range mustPassives(t).All() {
		if strings.TrimSpace(held.Flavour) == "" {
			t.Errorf("%q ships with no flavour clause, so its description opens on a derived line and never says what the trait is",
				held.ID)
		}
	}
}

// TestATraitFlavourNamesNoBody is the skill rule with its escape hatch removed.
//
// A skill free for anybody to carry may not say "mai", and a restricted one may,
// because its restriction guarantees the body: `ingrain` names roots and only a
// plant may take it. **A trait has no restriction mechanism at all** — no
// element, no archetype, no species, no character — so nothing can ever
// guarantee the body a clause names, and the ban is unconditional. There is no
// version of this a future field could relax; the field would have to be built
// first.
//
// The same hand-written list as the skills, read the same blunt way: the whole
// clause, with no attempt to tell whose shell is being described. Rewording
// costs a few words and a miss costs a sentence that reads as nonsense on
// somebody's screen.
func TestATraitFlavourNamesNoBody(t *testing.T) {
	for _, held := range mustPassives(t).All() {
		lowered := strings.ToLower(held.Flavour)
		for word, what := range bodyWords {
			if !strings.Contains(lowered, word) {
				continue
			}
			t.Errorf("%q is a trait anything may carry and its flavour says %q, which is %s: reword the clause, because there is no restriction to guarantee it",
				held.ID, word, what)
		}
	}
}

// TestATraitFlavourSpellsOutNoNumber closes the half of the digit ban a
// character check cannot see, exactly as the skills' own test does.
//
// passive.ParseBook refuses a digit, which protects the guarantee that no
// authored figure can go stale. It is a check on characters, and "hai lớp" walks
// straight past it while saying what "2 lớp" would have said — and a trait grants
// stacks, so the mistake is available here too.
func TestATraitFlavourSpellsOutNoNumber(t *testing.T) {
	for _, held := range mustPassives(t).All() {
		for _, word := range strings.Fields(strings.ToLower(held.Flavour)) {
			word = strings.Trim(word, ",.;:")
			if !slices.Contains(countWords, word) {
				continue
			}
			t.Errorf("%q spells out the number %q in its flavour; every figure in a description is derived, and a written one says the same thing twice until it stops being true",
				held.ID, word)
		}
	}
}

// TestNoShippedShareIsUnderOnePercent is an authoring rule rather than a
// defence, and it is stated here because it was decided rather than discovered.
//
// **No description will ever tune a figure by less than a percent.** A share
// that small is one nobody can feel across a battle, so it is not a tuning
// anybody would author on purpose — and a description of one is worse than
// useless, because `i18n.share` rounds to a whole percent and anything under
// five parts per thousand comes out as "0%", which reads as a feature that does
// not work. This is what makes that rounding safe rather than lossy: the rule
// keeps the data out of the range where rounding could lie, instead of the
// renderer carrying a decimal place to survive data nobody will write.
//
// The floor is a **percent**, not five parts per thousand, and the gap is
// deliberate. Half of it is rounding — 6 parts per thousand rounds to 1% and is
// legal — but a value in that gap is a value somebody typed a zero too many
// into, and there is no reason to allow it while the rule is "nothing under a
// percent".
//
// It walks the shipped books only. A unit test's fixture is free to use any
// number it likes: this is a rule about what is *authored*, and a fixture is not
// authored content.
func TestNoShippedShareIsUnderOnePercent(t *testing.T) {
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load shipped statuses: %v", err)
	}
	// A percent, in the parts-per-thousand the whole engine counts in.
	const floor = scale.Base / 100
	check := func(owner, what string, permille int) {
		t.Helper()
		// The magnitude, because a resistance may now be negative: a
		// vulnerability of -5 is as unsayable as a resistance of 5, and
		// comparing the value rather than its size would wave every negative
		// through while telling the author to "raise it".
		if permille < 0 {
			permille = -permille
		}
		if permille != 0 && permille < floor {
			t.Errorf("%s sets %s to %d parts per thousand, under the one percent a description can say; raise it or drop the field",
				owner, what, permille)
		}
	}
	for _, held := range mustPassives(t).All() {
		if held.Replies.Answers() {
			check(held.ID, "a reply", held.Replies.Power)
			for _, application := range held.Replies.Applies {
				check(held.ID, "a reply's chance", application.Chance)
			}
		}
		for _, application := range held.Applies {
			check(held.ID, "a rider's chance", application.Chance)
		}
		for _, resistance := range held.Resists {
			check(held.ID, "a resistance", resistance.Amount)
		}
		check(held.ID, "a drain", held.Drains)
		for _, raise := range held.Amplifies {
			check(held.ID, "an amplified effect", raise.Effect)
			check(held.ID, "an amplified chance", raise.Chance)
		}
		if held.While != nil {
			check(held.ID, "a gate", held.While.BelowHealth)
		}
	}
	for _, kind := range statuses.Kinds() {
		check(kind.ID, "a tick", kind.TickPower)
		for _, term := range kind.Modifiers {
			amount := int(term.Amount)
			if amount < 0 {
				amount = -amount
			}
			check(kind.ID, "a modifier", amount)
		}
	}
	// The skills too, since every share a skill declares is read out of the same
	// renderer and nothing about the rule is particular to a trait.
	for _, current := range mustSkills(t).Skills() {
		check(current.ID, "a power", current.Power)
		check(current.ID, "an accuracy", current.Accuracy)
		check(current.ID, "a pierce", current.Pierce)
		check(current.ID, "a critical chance", current.Crit)
		check(current.ID, "a restore", current.Restores)
		check(current.ID, "a drain", current.Drains)
		for _, application := range current.Applies {
			check(current.ID, "an application's chance", application.Chance)
		}
		for _, application := range current.SelfApplies {
			check(current.ID, "a self-application's chance", application.Chance)
		}
		if current.Requires != nil {
			check(current.ID, "a bonus power", current.Requires.BonusPower)
			check(current.ID, "a health threshold", current.Requires.BelowHealth)
		}
	}
}
