package seed_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/passive"
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
