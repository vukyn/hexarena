package battle_test

import "testing"

// TestAScarceSkillIsNotSpentOnWhatACommonOneBuys is the whole of what "holding a
// skill for a later turn" comes to in a rating one turn deep.
//
// Damage is clamped at a target's remaining health — a finishing blow is not rated
// above one that would kill twice over — so against a unit standing at a sliver
// the heaviest skill in a kit and the filler beside it are worth **exactly** the
// same. Before this the tie went to whichever came first in the kit, so `clout`
// was burnt on ten points of health and cooled down for three turns for it while
// `jab`, which is always there, was the option that would have done it.
//
// `clout` is deliberately listed first, because kit order is what the tie-break
// replaces: a version of this test with the cheap skill first passes either way.
func TestAScarceSkillIsNotSpentOnWhatACommonOneBuys(t *testing.T) {
	fight := squad(t, []string{"clout", "jab"}, []string{"lob"}, []string{"jab"},
		0, 0, 10)
	if choice := chosen(t, fight); choice.Skill != "jab" {
		t.Errorf("Suggest picked %q against a unit at ten health, want jab: two "+
			"options worth the same are not the same to spend", choice.Skill)
	}
}

// TestASummonIsSpentOnTheSameTermsAsAnything is the branch a mutation caught,
// and it caught it because nothing shipped summons.
//
// A summoning skill is rated before the shape — it has no power of its own, so it
// would otherwise be the fallback it used to be — and that makes it the one place
// in Suggest where the comparison could quietly have been written a second time.
// Two casts that put the same body on the same cell are worth the same, so the one
// that will be available again is the one to spend.
func TestASummonIsSpentOnTheSameTermsAsAnything(t *testing.T) {
	fight := squad(t, []string{"slow_copy", "copy"}, []string{"lob"},
		[]string{"jab"}, 0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "copy" {
		t.Errorf("Suggest picked %q, want copy: the two calls up the same body, "+
			"and one of them cools down for three turns", choice.Skill)
	}
}

// TestTheCheaperSkillDoesNotWinOnValue is the other half, and it is the half that
// keeps the tie-break a tie-break.
//
// A cooldown says nothing about what a skill is worth, so it may only decide
// between options already worth the same. Against a healthy unit `clout` deals
// more than `jab` and is the answer, cooldown and all — a rating that discounted
// scarcity would be pricing turns it cannot see, which is the mistake tempo
// already had to be corrected for.
func TestTheCheaperSkillDoesNotWinOnValue(t *testing.T) {
	fight := squad(t, []string{"clout", "jab"}, []string{"lob"}, []string{"jab"},
		0, 0, 0)
	if choice := chosen(t, fight); choice.Skill != "clout" {
		t.Errorf("Suggest picked %q against a healthy unit, want clout: the "+
			"tie-break may not outrank what an option is worth", choice.Skill)
	}
}
