package i18n_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
)

// The fold, and the one thing it owes that no gloss table owes: completeness.
//
// A gloss is allowed to miss — a skill with no name prints its id, which is this
// package's declared normal — and the failure is visible on screen. A missing
// fold is not visible anywhere: the row simply stops being findable, and nothing
// says it was ever there. So the table is asserted twice over. Once against a
// hand-written enumeration of the Vietnamese alphabet, which is the design record
// and is meant to be read; and once against the **data**, so that a name authored
// later in a letter the table lacks is a red test rather than a row nobody can
// reach.

// vietnameseLetters is every letter Vietnamese writes that is not already ASCII,
// grouped under the ASCII letter it folds to.
//
// Hardcoded on purpose and written out in full rather than composed from a tone
// list: this is the claim being made about the alphabet, and a generated
// expectation would be the implementation's own table with a second name. Seven
// bases, sixty-seven letters — twelve vowels at five tones each, the six
// modified vowels bare, and đ.
var vietnameseLetters = []struct {
	base     rune
	accented string
}{
	{'a', "àáảãạăằắẳẵặâầấẩẫậ"},
	{'d', "đ"},
	{'e', "èéẻẽẹêềếểễệ"},
	{'i', "ìíỉĩị"},
	{'o', "òóỏõọôồốổỗộơờớởỡợ"},
	{'u', "ùúủũụưừứửữự"},
	{'y', "ỳýỷỹỵ"},
}

// TestEveryVietnameseLetterFoldsToItsBase is the enumeration, in both cases.
//
// Upper case matters as much as lower: the query an author types may be shouted,
// and the upper-case half of the table is derived from the lower-case half rather
// than written twice, so nothing but a test says the derivation is right. Đ is
// the entry that is not a mark on a base letter and is therefore the reason this
// is a table at all — see i18n.Fold.
func TestEveryVietnameseLetterFoldsToItsBase(t *testing.T) {
	counted := 0
	for _, group := range vietnameseLetters {
		for _, letter := range group.accented {
			counted++
			if got := i18n.FoldLetter(letter); got != group.base {
				t.Errorf("%q (%U) folds to %q, want %q", letter, letter, got, group.base)
			}
			upper, upperBase := unicode.ToUpper(letter), unicode.ToUpper(group.base)
			if got := i18n.FoldLetter(upper); got != upperBase {
				t.Errorf("%q (%U) folds to %q, want %q", upper, upper, got, upperBase)
			}
			// And the fold a match uses lower-cases as well, from either case.
			if got := i18n.Fold(string(letter) + string(upper)); got != string(group.base)+string(group.base) {
				t.Errorf("Fold(%q) is %q, want %q",
					string(letter)+string(upper), got, string(group.base)+string(group.base))
			}
		}
	}
	// The count is the design record too: an entry lost from the table above would
	// otherwise just mean one fewer letter checked.
	if counted != 67 {
		t.Errorf("the alphabet here holds %d accented letters, want 67", counted)
	}
	// Đ named on its own, because it is the entry the obvious implementation —
	// NFD, then drop every combining mark — silently gets wrong: it is its own
	// letter rather than a d with a mark on it.
	for _, pair := range []struct{ letter, want rune }{{'đ', 'd'}, {'Đ', 'D'}} {
		if got := i18n.FoldLetter(pair.letter); got != pair.want {
			t.Errorf("%q folds to %q, want %q", pair.letter, got, pair.want)
		}
	}
}

// TestALetterWithNoFoldIsUnchanged is the other half of the contract: the table
// is a lookup, not a filter, so anything it does not know comes back as it was.
//
// The punctuation and the arrows are in here because the catalog is full of them
// — "↑/↓ · / lọc" — and a fold that dropped what it did not recognise would take
// a footer apart.
func TestALetterWithNoFoldIsUnchanged(t *testing.T) {
	for _, letter := range []rune{'a', 'z', 'Z', 'D', 'd', '7', '_', '·', '↑', '?', 'ß', 'ж'} {
		if got := i18n.FoldLetter(letter); got != letter {
			t.Errorf("%q (%U) folded to %q, want it left alone", letter, letter, got)
		}
	}
	// Fold still lower-cases, which is the one difference between the two.
	if got := i18n.Fold("Razor_Leaf · ↑/↓"); got != "razor_leaf · ↑/↓" {
		t.Errorf("Fold left %q", got)
	}
}

// TestEveryLetterAShippedNameUsesCanBeFolded is the half driven from the data
// rather than from the enumeration above, and it is what makes the table's
// completeness a property of the repository instead of a claim about the alphabet.
//
// The scope is every Vietnamese word this client can put on screen: every wording
// in both catalogs, every shipped skill's authored name, and every name the gloss
// tables answer with for an id the shipped books declare. A non-ASCII letter in
// any of them that folds to something that is not an ASCII letter is a letter the
// filter cannot match, so authoring a name with one has to fail here.
func TestEveryLetterAShippedNameUsesCanBeFolded(t *testing.T) {
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load the shipped skills: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	passives, err := seed.PassiveBook()
	if err != nil {
		t.Fatalf("load the shipped passives: %v", err)
	}

	var named []string
	for _, key := range i18n.Keys() {
		for _, lang := range i18n.Langs() {
			named = append(named, lang.Text(key))
		}
	}
	for _, declared := range skills.Skills() {
		named = append(named, declared.Name, i18n.Vi.SkillName(declared))
	}
	for _, kind := range statuses.Kinds() {
		named = append(named, i18n.Vi.Gloss(kind.ID))
	}
	for _, held := range passives.All() {
		named = append(named, i18n.Vi.PassiveName(held))
	}
	if len(named) < int(i18n.Keys()[len(i18n.Keys())-1]) {
		t.Fatalf("only %d strings collected, so this measures almost nothing", len(named))
	}

	// The letters that actually turned up, so a run that collected nothing
	// Vietnamese cannot pass by being empty.
	folded := 0
	for _, text := range named {
		for _, letter := range text {
			if letter < unicode.MaxASCII || !unicode.IsLetter(letter) {
				continue
			}
			base := i18n.FoldLetter(letter)
			if base >= unicode.MaxASCII || !unicode.IsLetter(base) {
				t.Errorf("%q (%U) in %q folds to %q, which is not an ASCII letter — "+
					"add it to foldGroups", letter, letter, text, base)
				continue
			}
			folded++
		}
	}
	if folded == 0 {
		t.Fatal("no accented letter was reached, so the table's completeness was not measured")
	}
	t.Logf("folded %d accented letters across %d strings", folded, len(named))
}

// TestAQueryMatchesIgnoringCaseAndDiacritics is the predicate the listing asks,
// including the answer that makes an empty query hide nothing.
func TestAQueryMatchesIgnoringCaseAndDiacritics(t *testing.T) {
	for _, test := range []struct {
		query  string
		fields []string
		want   bool
	}{
		{"", []string{"anything"}, true},
		{"   ", []string{"anything"}, true},
		{"diep", []string{"razor_leaf", "phi diệp"}, true},
		{"DIEP", []string{"razor_leaf", "phi diệp"}, true},
		{"diệp", []string{"razor_leaf", "phi diệp"}, true},
		{"razor", []string{"razor_leaf", "phi diệp"}, true},
		{"RAZOR_LEAF", []string{"razor_leaf", "phi diệp"}, true},
		{"phi d", []string{"razor_leaf", "phi diệp"}, true},
		{"doc", []string{"sludge_bomb", "độc bùn"}, true},
		{"ĐỘC", []string{"sludge_bomb", "độc bùn"}, true},
		{"zzz", []string{"razor_leaf", "phi diệp"}, false},
		{"diep", []string{"razor_leaf", ""}, false},
		{"diep", nil, false},
	} {
		if got := i18n.Matches(test.query, test.fields...); got != test.want {
			t.Errorf("Matches(%q, %q) is %v, want %v",
				test.query, test.fields, got, test.want)
		}
	}

	// And the skill-shaped reading of it, which is the one the listing uses. It
	// asks nothing about the language in front — a name is a field on the skill —
	// so both languages find the same row.
	carried := skill.Skill{ID: "razor_leaf", Name: "phi diệp"}
	if !i18n.MatchesSkill("diep", carried) {
		t.Error("a shipped name is not found with its marks left off")
	}
	if !i18n.MatchesSkill("razor", carried) {
		t.Error("a skill is not found by its id")
	}
	if i18n.MatchesSkill(strings.ToUpper("zzz"), carried) {
		t.Error("a skill is found by a query nothing holds")
	}
	// A skill with no authored name still matches on its id, and does not match on
	// the empty name — which is what an English screen would be reading if the
	// language decided this, and is exactly why it does not.
	bare := skill.Skill{ID: "strike"}
	if !i18n.MatchesSkill("strike", bare) {
		t.Error("an unnamed skill is not found by its id")
	}
	if i18n.MatchesSkill("diep", bare) {
		t.Error("an unnamed skill matched a name it does not have")
	}
}
