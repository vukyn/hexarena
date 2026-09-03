package screen

import (
	"strings"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// BlurbScreen shows what the thing under the cursor behind it does, in the
// sentences a player reads rather than in the figures an author types.
//
// It closes the loop the authoring tool was missing: every other row on the form
// says what a field *is*, and none of them said what the skill would sound like
// once somebody had to decide whether to use it. An author tuning a bonus from
// 1000 to 700 could see the damage move and not that "doubles" had become
// "amplifies a bit".
//
// # Three screens raise it, and it is still one screen
//
// The skill listing raises it for the skill under its cursor. The cast browser
// raises it for the traits the character under *its* cursor carries at the level
// it is sitting on — which is the same failure one layer over: an author moving
// virulence from 300 to 200 watched the table's "+30%" change and never read the
// sentence, because DescribePassive had exactly one caller and it was the battle
// prompt.
//
// The played battle raises it for the option under its cursor, which is the same
// question a third time and the one place it is asked by a *player* rather than
// by an author: the compact line beside each option is what four of them can be
// compared on, and this is the long form of whichever one is in front. It shares
// the listing's rendering exactly — SkillLines — so a description read while
// choosing and the same description read while authoring cannot differ.
//
// ⚠️ Nothing here touches the battle. The played battle is the one screen
// holding something its client's model does not copy, so raising this and
// leaving it again must step no turn: the option is named on the way in, the
// skill is looked up in the library, and going back puts the previous screen up
// rather than re-entering it — which would rebuild the battle from its seed and
// throw the played half away.
//
// A second screen would have been a second copy of the framing, the footer and
// the escape. So this one branches on what it was handed instead.
//
// # It is handed its subject rather than reaching for one
//
// ⚠️ **This used to read the three screens that raise it** — the browser's level
// eight times, the skill listing's cursor five, the browser's cursor five, and
// the played battle four ways — which is a screen that could not live here and
// could not be drawn by a client whose screens are different ones. The raiser
// pushes a Subject and this describes it: the id of the skill or the character,
// where it sat in the list it came out of, and the level a character is being
// read at.
//
// It keeps **no cursor and no level of its own**, and the push is what keeps that
// true rather than a second copy of it: every key that moves the listing behind
// re-pushes, so the two cannot disagree about what is in front and walking either
// here walks it there. A description of a character the author is not looking at
// is worse than none.
//
// ⚠️ **Where the raiser came from is deliberately not here.** Going back is the
// client's question — its own screens are the only answer to it — so a client
// carries that beside this value rather than in it. See Action.Back for why a
// screen in this package may not name its own way out.
type BlurbScreen struct {
	// Subject is what is being described, handed over by whichever screen raised
	// it. The zero value describes nothing, and draws the same line an empty
	// listing does.
	Subject Subject
	// Scroll is how far down the trait sentences have been walked, and it is
	// **not** the cursor this screen refuses to keep.
	//
	// The difference is what the two can disagree about. A cursor of its own
	// could point at a different character than the browser behind it, so the
	// screen would describe one thing and the screen behind would show another.
	// A scroll offset selects nothing: it is which lines of the answer are
	// visible, and the answer is still the browser's. Every key that changes
	// *what* is being described resets it, so it cannot survive into a shorter
	// answer and leave a reader looking at nothing.
	//
	// Five traits at the level cap wrap to more lines than a 120-by-24 window
	// holds, which is the floor rather than an unusual case.
	// Letting the frame cut it would mean the one screen built for reading a
	// trait cannot finish reading one.
	Scroll int
}

// View is the subject described, in whichever reading its kind asks for.
func (b BlurbScreen) View(c Context) (string, string) {
	switch b.Subject.Kind {
	case CharacterSubject:
		return b.viewTraits(c)
	case SkillSubject:
		return b.viewSkill(c)
	}
	// NoSubject, and anything this describer has not been taught: there is
	// nothing to describe, said the way an empty listing says it. Reachable only
	// by drawing the screen without raising it, which a client's own applier
	// makes impossible and a hand-built state does not.
	return "  " + c.Text(i18n.NoneCatalogued) + "\n", c.Text(i18n.BlurbFooter)
}

// viewSkill is one skill described under its name, with where it sits in the
// list it came out of.
//
// ⚠️ **One reading for the listing and for the battle's option list**, which used
// to be two functions over two screens' state. The two differ in which skill and
// which list; the heading, the position, the marked sentences and the footer are
// the same answer, and a second copy is a second thing free to decide a status
// name is not worth marking.
//
// The skill comes out of the library rather than out of the battle, because a
// battle carries a unit's resolved kit and the sentences are about the declared
// skill — and the id is what the raiser handed over, so the lookup is the same
// one either way round.
//
// An empty list is an ordinary answer rather than an error: a turn nobody is
// being asked about is a turn with nothing to describe, and both keys that raise
// this refuse the empty case anyway.
func (b BlurbScreen) viewSkill(c Context) (string, string) {
	footer := c.Text(i18n.BlurbFooter)
	if b.Subject.Of == 0 {
		return "  " + c.Text(i18n.NoneCatalogued) + "\n", footer
	}
	declared, err := c.Lib.Skills().Lookup(b.Subject.ID)
	if err != nil {
		return "  " + c.Style.Bad.Render(c.Lang.Error(err)) + "\n", footer
	}
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Lang.GlossedSkill(declared)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.ChoicePosition, b.Subject.At, b.Subject.Of)) + "\n\n")
	for _, line := range SkillLines(c, declared) {
		out.WriteString(line + "\n")
	}
	return out.String(), footer
}

// SkillLines is what a skill does, one line per sentence and already marked.
//
// A function rather than the second copy it was about to become: the authoring
// tool's kit picker reads the skill under its own cursor and this screen reads
// the one it was handed, which is the same paragraph asked for from two places.
// A second copy of the marking is a second thing that can decide a name is worth
// pointing out — and the comment above this screen already refused to write a
// second copy of a framing for the same reason.
//
// Exported because that second caller is in another package now. It takes a
// Context and a declared skill and reads nothing else, so it is as much this
// package's as the screen that draws it.
func SkillLines(c Context, declared skill.Skill) []string {
	// The status names these sentences will use, marked where they are printed,
	// exactly as the trait listing marks its own -- the two screens are read one
	// after the other, and a name that is bold on one and plain on the other reads
	// as a difference between the effects rather than between the screens. A miss
	// in the glossary drops out here rather than marking a bare id.
	names := make([]string, 0, 4)
	for _, id := range i18n.StatusesInSkill(declared) {
		if name := c.Lang.Gloss(id); name != "" {
			names = append(names, name)
		}
	}
	out := make([]string, 0, 4)
	for _, line := range strings.Split(c.Lang.Describe(declared, c.Lib.Patterns()), "\n") {
		out = append(out, "  "+Marked(line, names, func(word string) string {
			return c.Style.Emphasis.Render(word)
		}))
	}
	return out
}

// viewTraits is what the character the raiser named carries, at the level it
// named.
//
// The level is read rather than assumed for the reason the detail pane behind it
// reads it: a trait comes in at a level, so "what is this unit carrying" has no
// answer without one — and a screen that described every declared trait would be
// describing traits the character does not have yet.
//
// The form is the raiser's, which is what the detail pane resolves with. Two
// screens asking the same question have to ask it the same way, or walking from
// one to the other changes the answer for a reason nothing on either says.
//
// ⚠️ It used to be progression.Furthest here and there, which is the same answer
// for eleven of the twelve shipped characters and **no answer at all** for the
// twelfth: a line that forks reaches two grown forms at one level, and
// cast.Character.form falls back to the empty stage when it cannot name one — so
// this screen drew the traits of neither arm and said nothing about it, which is
// the quiet half of the same defect the art preview showed as a refusal.
func (b BlurbScreen) viewTraits(c Context) (string, string) {
	footer := c.Text(i18n.BlurbTraitsFooter)
	if b.Subject.Of == 0 {
		return "  " + c.Text(i18n.BrowseNothingHere) + "\n", footer
	}
	character, known := c.Lib.Characters().Get(b.Subject.ID)
	if !known {
		return "  " + c.Text(i18n.BrowseNothingHere) + "\n", footer
	}
	held := c.Lib.KitPassives(character.PassivesAt(b.Subject.Level, b.Subject.Stage))
	// Which arm the sentences below belong to. Empty on a line that does not
	// fork, so a linear character's screen is exactly what it was.
	form := FormRow(c, character, b.Subject.Level, b.Subject.Stage)

	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(character.Name) + "  " +
		c.Style.Dim.Render(c.Text(i18n.LabelAtLevel, b.Subject.Level)) + "\n")
	// Under the heading and above the blank, so the character, the level and the
	// form read as one block and the sentences still start after a gap. An empty
	// form row leaves exactly the two newlines the heading used to write.
	out.WriteString(form + "\n")
	if len(held) == 0 {
		// A trait the character has not learned yet is the common case at a low
		// level, so this is a normal answer rather than an empty screen: the same
		// sentence DescribePassive gives for a trait that holds nothing, because
		// "carries no traits" is the same fact from either end.
		return out.String() + "  " + c.Text(i18n.BlurbTraitNone), footer
	}

	body := traitLines(c, held)
	room := TraitRoom(c)
	// The fork row is paid for out of the sentences' room rather than out of
	// TraitRoom, which the authoring tool's trait picker also reads and which
	// draws no such row. Counted only when it is drawn — unlike the position
	// line above it, which is counted either way — because whether it appears
	// turns on the character rather than on the window, so there is no window
	// height it can flicker at.
	if form != "" {
		room = max(room-1, 3)
	}
	// Clamped here rather than where it is incremented, because the key that
	// moves it does not know how long the answer is: the answer is built from
	// the subject the raiser last pushed, and that can have moved since.
	scroll := Clamp(b.Scroll, 0, max(len(body)-room, 0))
	for _, line := range body[scroll:min(scroll+room, len(body))] {
		out.WriteString(line + "\n")
	}
	if len(body) > room {
		out.WriteString(c.Style.Dim.Render("  " + c.Text(i18n.BlurbMore,
			min(scroll+room, len(body)), len(body))))
	}
	// No trailing newline. The frame splits this on newlines and pads what is
	// left, so a trailing one is a blank line that counts against the room --
	// and the frame cuts from the bottom, so what that blank costs is the line
	// saying there is more to read.
	return strings.TrimRight(out.String(), "\n"), footer
}

// traitLines is every line the trait sentences take, already wrapped.
//
// Wrapped rather than clipped, and that is not a preference: the derived reply
// sentence is seventy-six cells before its indent, so the floor cut it mid-word
// — "…3% khả nă" — which reads as the tool being broken rather than as a
// terminal being narrow. Every other pane that carries a sentence wraps for the
// same reason.
func traitLines(c Context, held []passive.Passive) []string {
	out := make([]string, 0, 6*len(held))
	for index, one := range held {
		if index > 0 {
			out = append(out, "")
		}
		out = append(out, "  "+c.Style.Label.Render(c.Lang.GlossedPassive(one)))
		out = append(out, TraitSentences(c, one)...)
	}
	return out
}

// TraitSentences is one trait's description, wrapped and indented under
// whatever named it above.
//
// Split off the loop above rather than duplicated into the authoring tool's
// picker, which reads a trait under a cursor of its own and carries the name in
// its heading instead of in the body: what the two share is the measure and the
// indent, and those are exactly the parts a second copy would be free to
// disagree about. Exported for that caller, which is in another package now.
func TraitSentences(c Context, one passive.Passive) []string {
	// Wrapped to the floor rather than to the window, which is the opposite of
	// what Context.Wrapped does and is right for a different reason. Those rows
	// carry authored free text -- a biography, a kit of nine ids -- which has to
	// go somewhere and gets whatever width there is. These are the program's own
	// prose, and prose has a measure: a sentence run across a two-hundred-column
	// terminal is a line a reader loses their place in, and it is also a line
	// TestEveryWordingFitsTheMinimumWidth measures against the floor.
	room := MinWidth - 1 - TraitIndent
	out := make([]string, 0, 6)
	for _, sentence := range strings.Split(c.Lang.DescribePassive(one), "\n") {
		for _, line := range WrapWords(sentence, max(room, 8)) {
			out = append(out, strings.Repeat(" ", TraitIndent)+line)
		}
	}
	return out
}

// TraitRoom is how many lines of sentences fit: the window, less the two the
// heading takes and the one the position line does.
//
// The authoring tool's picker shares it rather than counting again, which is why
// it is exported. The two spend their lines on the same four things in the same
// order, so two counts would be two answers to one question — and the one that
// was wrong would be the one that let the frame cut the line saying there is
// more to read.
//
// The position line is counted whether or not it is drawn. Counting it only when
// it appears would make the answer one line taller the moment it fits, which is
// the shape of loop that flickers between two layouts on a window exactly at the
// boundary.
func TraitRoom(c Context) int {
	room := c.Height - 4 - 3
	if room < 3 {
		return 3
	}
	return room
}
