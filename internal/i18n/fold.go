package i18n

import (
	"strings"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/skill"
)

// Vietnamese on a terminal is a matching problem before it is a rendering one.
// An author hunting for "phi diệp" has an ASCII keyboard in front of them and,
// very often, no Vietnamese input method at all — so a filter that only matched
// the letters as they were authored would leave every translated name
// unsearchable and reduce the whole feature to its id half. Fold is what closes
// that: it takes a letter back to the ASCII one underneath it, so "diep" finds
// "diệp" and "DIEP" finds it too.
//
// # Why a table rather than a normaliser
//
// The obvious implementation is NFD followed by dropping every combining mark,
// which needs golang.org/x/text — an *indirect* dependency of this module
// today. It would not have been enough on its own: **đ is not a d with a mark
// on it**, it is its own letter with its own codepoint, so the normalising
// version needs a hand-written entry for it beside the general rule. Once one
// entry is hand-written, an explicit table is the smaller thing to read and the
// only thing to check, and a dependency stays where it is.
//
// # The table has to be COMPLETE, unlike every gloss table in this package
//
// A gloss is allowed to miss: a skill with no name prints its id, which is this
// package's declared normal. A missing fold does not degrade like that — the
// row simply stops being findable, with nothing on screen saying it was ever
// there. So the completeness is measured rather than assumed, and it is
// measured against the **data**: TestEveryLetterAShippedNameUsesCanBeFolded
// walks the shipped books and this package's own catalog and fails on a letter
// that folds to nothing, so a name authored later in a letter the table lacks
// is a red test rather than a row nobody can reach.
//
// foldGroups is that table: one ASCII base, and every Vietnamese letter that
// folds to it. Lower case only — the upper-case half is derived, because two
// hand-written halves are two places for one letter to go missing.
var foldGroups = []struct {
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

// foldable is foldGroups indexed for lookup, both cases.
//
// It is built once rather than in an init function so that the table and the
// map it becomes sit next to each other, and it is never ranged over — a map
// range would randomise the order, and while that rule is internal/core's, this
// is not the place to teach anybody the exception.
var foldable = buildFoldTable()

func buildFoldTable() map[rune]rune {
	table := make(map[rune]rune)
	for _, group := range foldGroups {
		for _, letter := range group.accented {
			table[letter] = group.base
			table[unicode.ToUpper(letter)] = unicode.ToUpper(group.base)
		}
	}
	return table
}

// FoldLetter is one letter with its diacritic taken off, and the letter itself
// when it has none.
//
// Case is kept, which is what makes this the half of the fold that is only
// about the alphabet: Fold is what a match actually uses and lower-cases as
// well. Split in two because the two halves fail differently — a letter missing
// from the table is a hole in the alphabet, while a case that does not fold is
// a bug in one line of Fold.
func FoldLetter(letter rune) rune {
	if base, known := foldable[letter]; known {
		return base
	}
	return letter
}

// Fold is a string reduced to what somebody types looking for it: every letter
// at its ASCII base, in lower case.
func Fold(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	for _, letter := range text {
		out.WriteRune(unicode.ToLower(FoldLetter(letter)))
	}
	return out.String()
}

// Matches reports whether a typed query is found in any of the fields offered,
// ignoring case and ignoring diacritics.
//
// **An empty query matches everything**, which is the same shape the two
// categorical filters in the client already have: browseScreen.filter and
// pickState.filter both index their groups from one and keep nought for the
// filter that hides nothing. A typed filter has no index to spare, so the empty
// string is that value, and it is answered here rather than at each caller so
// that "nothing typed hides nothing" cannot be decided twice.
//
// The fields are variadic because a row is usually findable by more than one of
// its columns, and it stays generic on purpose: the pickers narrow by a *group*
// today, and if one is ever given a typed filter as well, this is the function
// it would ask rather than a second reading of the same idea. Nothing has
// adopted it yet, so this is the one caller plus MatchesSkill below.
func Matches(query string, fields ...string) bool {
	wanted := Fold(strings.TrimSpace(query))
	if wanted == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(Fold(field), wanted) {
			return true
		}
	}
	return false
}

// MatchesSkill reports whether a typed query names a skill: by its id, or by
// the Vietnamese name it is called by.
//
// ⚠️ **It is not a method on Lang, and that is a decision rather than an
// oversight.** Name is a field on skill.Skill — data, authored once, in the one
// language the data is written in — so the rows a query finds are the same rows
// whichever language the screen is in, even though the English listing draws no
// name column at all. Reading the name off the language in front instead would
// mean ctrl+l silently changed which rows a standing query had found, from a
// chord that works on every screen and is documented as keeping everything
// typed; a query narrowing to five rows and then to two because somebody
// compared the two languages is exactly the mutation-behind-the-author's-back
// the form's element-and-kit rule refuses.
//
// The cost is stated: an English reader can be handed a row whose id does not
// hold what they typed, because what it matched was a name that screen does not
// show. The filter row says how many of the book are being shown, and the name
// is one keystroke away on ?, which is the same trade the listing's dropped
// gloss column already takes.
func MatchesSkill(query string, carried skill.Skill) bool {
	return Matches(query, carried.ID, Vi.SkillName(carried))
}
