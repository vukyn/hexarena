package screen

import (
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// # The form the read-only views read a character in
//
// Three screens describe a character at a level and none of them fields one: the
// cast browser's detail pane, the art preview and the traits blurb. All three
// used to ask progression.Furthest for the form, which is what a caller passes
// when nobody is choosing — and on a line that **forks** there is no single
// furthest, so progression.Line.StageAt refused:
//
//	level 46 reaches [Poliwrath Politoed], which are alternatives: name the one being fielded
//
// ⚠️ **The refusal is right and is not what these functions work around.** Taking
// whichever arm the file lists last would hand a reader one form's stat line,
// one form's picture and one form's traits with nothing on the screen saying
// which — the silent wrong answer StageAt exists to refuse. What was missing was
// somewhere for the reader to *say* which arm, so a forking character was a dead
// end in every read-only view; pokemon.poliwag, the one shipped fork, could not
// be previewed at all at any level from 32 up.
//
// So these views get a choice, in the shape the squad builder already uses for
// the same question: a value stepped through with a key, drawn on screen as
// `< name >` so what is in force is read rather than assumed. The builder keeps
// its choice on the placement, because a placement is a thing that gets saved;
// these keep it on the cast browser, because the browser is the one screen with
// a cursor and the other two are handed a Subject by it.
//
// # Why the arms come from FurthestAt rather than StagesAt
//
// A placement may deliberately field an *earlier* form — a level-46 unit fielded
// as Poliwhirl is a legal squad — so cast.Character.StagesAt offers every stage
// the level has reached, and that is what SquadsScreen.StageChoices is built on.
// These screens are asking a narrower question: which form does this level
// resolve to. cast.Character.FurthestAt answers exactly that, and answering it
// with StagesAt would put three names in front of every linear character in the
// cast, which is a chooser on twelve characters to serve one.
//
// That is also what keeps a line that does not fork **byte for byte what it
// was**: FurthestAt hands back one stage, so ChosenForm gives back
// progression.Furthest, every Resolve is the call it always made, and FormRow
// draws nothing at all.

// FormArms is the grown ends of a character's line at a level: one form on a
// line that does not fork, one per arm on a line that does.
//
// A level outside the line, or a character whose line covers nothing there, is
// no arms rather than an error: every caller here is drawing a screen, and the
// refusal a reader needs to see is the one Resolve gives with the level in it,
// not a second one from a chooser.
func FormArms(character cast.Character, level int) []progression.Stage {
	arms, err := character.FurthestAt(level)
	if err != nil {
		return nil
	}
	return arms
}

// ChosenForm is the form these views resolve a character in: progression.Furthest
// on a line that does not fork, and otherwise the arm chosen with the form key.
//
// ⚠️ **It settles the choice on every read rather than storing a settled one.**
// The cursor and the level both move under a chosen name — walking to another
// character, or below the level the fork opens at, leaves a name that is not on
// the list any more — and a stale name reaches Resolve as a form the line does
// not answer to. Deriving it means the two can never disagree; SquadsScreen
// settles the same hazard by writing the field back, which it has to do because
// its value is saved and this one is not.
//
// The fallback is the **first** arm, which is a pick — and the reason it is not
// the silent pick StageAt refuses is that FormRow draws it. A chooser has to open
// on something; what it may not do is open on something unnamed.
func ChosenForm(character cast.Character, level int, chosen string) string {
	arms := FormArms(character, level)
	if len(arms) < 2 {
		return progression.Furthest
	}
	for _, arm := range arms {
		if arm.Name == chosen {
			return chosen
		}
	}
	return arms[0].Name
}

// NextForm is the arm after the one in force, wrapping round, and is what the
// form key writes back.
//
// A character with nothing to choose hands the choice back untouched rather than
// clearing it: the key is live on every character because the screens are one
// screen, and a reader who has chosen Politoed, walked past a linear character
// and walked back should find Politoed still in front.
func NextForm(character cast.Character, level int, chosen string) string {
	arms := FormArms(character, level)
	if len(arms) < 2 {
		return chosen
	}
	settled := ChosenForm(character, level, chosen)
	for index, arm := range arms {
		if arm.Name == settled {
			return arms[(index+1)%len(arms)].Name
		}
	}
	return arms[0].Name
}

// FormRow is the line naming which arm is in force, ready to write into a body:
// indented, styled and newline-terminated, or **empty** when the character has
// no fork at this level.
//
// Empty rather than a row saying "one form" is the whole of the promise a linear
// line is unchanged, so every caller writes the result unconditionally and lets
// this decide.
//
// It is one self-contained line in all three views rather than a labelled row in
// the browser and a bare one in the other two. The detail pane's label column is
// for facts about the character, and this is a control: the same reason
// SquadsScreen draws its held-back note as an unaligned line under the rows it
// belongs to.
func FormRow(c Context, character cast.Character, level int, chosen string) string {
	arms := FormArms(character, level)
	if len(arms) < 2 {
		return ""
	}
	settled := ChosenForm(character, level, chosen)
	at := 1
	for index, arm := range arms {
		if arm.Name == settled {
			at = index + 1
		}
	}
	return "  " + c.Style.Label.Render(
		c.Text(i18n.FormChoice, settled, at, len(arms))) + "\n"
}
