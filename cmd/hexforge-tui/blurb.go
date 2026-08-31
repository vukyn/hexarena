package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// blurbScreen shows what the thing under the cursor behind it does, in the
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
// the listing's rendering exactly — skillLines — so a description read while
// choosing and the same description read while authoring cannot differ.
//
// ⚠️ Nothing here touches the battle. playScreen is the one screen holding
// something the model does not copy, so raising this and leaving it again must
// step no turn: the option is read, the skill is looked up in the library, and
// esc puts m.screen back rather than going through model.enter — which would
// rebuild the battle from its seed and throw the played half away.
//
// A second screen would have been a second copy of the framing, the footer and
// the esc, differing only in which describer it called. So this one branches on
// `from` instead, which is the single field it keeps — and `from` is not a
// cursor, it is which screen is behind, a question esc had to answer anyway and
// used to answer with a constant because there was only ever one answer.
//
// Like previewScreen it keeps **no cursor and no level**. Both are read off the
// screen it was raised from, so the two cannot disagree about what is in front,
// and walking either here walks it there. A description of a character the author
// is not looking at is worse than none.
//
// # Why a screen rather than a block under the form
//
// The form has nineteen fields and an 120-by-24 window shows
// thirteen of them; the comments in skills.go record fighting for a single row
// twice. A description is three lines and wraps to more, so putting it under the
// form would cost a quarter of the fields for something read occasionally. A
// screen costs nothing until it is asked for.
type blurbScreen struct {
	// from is the screen that raised it, and where esc goes back to.
	from screen
	// scroll is how far down the trait sentences have been walked, and it is
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
	scroll int
}

func (b blurbScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "?":
		m.screen = b.from
		return m, nil
	}
	// The cursor moves from here too, so an author reading one description can
	// read the next without going back and forth. It walks the raising screen's
	// own cursor, which is the one thing this screen borrows — and on the
	// browser it borrows the level as well, because a trait comes in at a level
	// and walking it is how that is seen.
	if b.from == screenPlay {
		// The option behind moves from here too, the way the listing's cursor
		// does, so four options can be read one after another without going back
		// and forth. It goes through playScreen.move, which is the one place that
		// knows an unavailable option is stepped over.
		//
		// While aiming they do nothing: the skill is settled and the cell is
		// what is being chosen, so walking the options would change what is
		// described out from under a decision already half taken.
		if m.play.pending != nil && !m.play.aiming {
			switch message.String() {
			case "up", "k":
				m.play = m.play.move(-1)
			case "down", "j":
				m.play = m.play.move(1)
			}
		}
		return m, nil
	}
	if b.from == screenBrowse {
		rows := len(m.browse.rows())
		switch message.String() {
		// [ and ] alias the page keys, here and at the other two sites that
		// scroll, because a compact keyboard has neither PgUp nor PgDn and the
		// trait sentences are exactly what a window this size cannot finish. It
		// is safe to take a printable key here: this branch feeds no text field,
		// which is what makes the alias free rather than a letter stolen from
		// something being typed.
		case "pgdown", "]":
			b.scroll++
			m.blurb = b
			return m, nil
		case "pgup", "[":
			b.scroll = max(b.scroll-1, 0)
			m.blurb = b
			return m, nil
		case "up", "k":
			m.browse.cursor = clamp(m.browse.cursor-1, 0, rows-1)
		case "down", "j":
			m.browse.cursor = clamp(m.browse.cursor+1, 0, rows-1)
		case "left", "h":
			m.browse.level = clamp(m.browse.level-1, 1, progression.LevelCap)
		case "right", "l":
			m.browse.level = clamp(m.browse.level+1, 1, progression.LevelCap)
		case "home":
			m.browse.level = 1
		case "end":
			m.browse.level = progression.LevelCap
		}
		// Anything that changed which character or which level is in front
		// changed the answer, so the offset into the old one means nothing.
		b.scroll = 0
		m.blurb = b
		return m, nil
	}
	// Through the listing's own visible rows, not its book: the cursor counts what
	// the filter left, so stepping it against the full book would walk past the
	// end of the narrowed listing and describe a skill the screen behind is not
	// pointing at. skillsScreen.rows is the one funnel — see it for why.
	rows := len(m.skills.rows())
	switch message.String() {
	case "up", "k":
		m.skills.cursor = clamp(m.skills.cursor-1, 0, rows-1)
	case "down", "j":
		m.skills.cursor = clamp(m.skills.cursor+1, 0, rows-1)
	}
	return m, nil
}

func (b blurbScreen) view(m model) (string, string) {
	switch b.from {
	case screenBrowse:
		return b.viewTraits(m)
	case screenPlay:
		return b.viewOption(m)
	}
	// The rows the listing has left rather than the book, so the position beside
	// the name — "3 / 5" — counts the same list the marker was moving through.
	skills := m.skills.rows()
	if len(skills) == 0 {
		return "  " + m.text(i18n.NoneCatalogued) + "\n", m.text(i18n.BlurbFooter)
	}
	at := clamp(m.skills.cursor, 0, len(skills)-1)
	return viewSkill(m, skills[at], at+1, len(skills))
}

// viewOption is the skill under the battle's own cursor.
//
// It reads m.play rather than keeping a cursor, which is the rule this screen is
// built on: a cursor of its own could point at a different option than the
// battle behind it, and then the description and the list would disagree about
// what is being chosen. The skill comes out of the library rather than out of the
// battle, because a battle carries a unit's resolved kit and the sentences are
// about the declared skill.
//
// Nothing pending and no options are both ordinary answers rather than errors: a
// turn nobody is being asked about is a turn with nothing to describe, and the
// key that raises this refuses the empty case anyway.
func (b blurbScreen) viewOption(m model) (string, string) {
	pending := m.play.pending
	if pending == nil || len(pending.Options) == 0 {
		return "  " + m.text(i18n.NoneCatalogued) + "\n", m.text(i18n.BlurbFooter)
	}
	at := clamp(m.play.option, 0, len(pending.Options)-1)
	declared, err := m.lib.Skills().Lookup(pending.Options[at].Skill)
	if err != nil {
		return "  " + m.style.bad.Render(m.lang.Error(err)) + "\n", m.text(i18n.BlurbFooter)
	}
	return viewSkill(m, declared, at+1, len(pending.Options))
}

// viewSkill is one skill described under its name, with where it sits in the
// list it came out of.
//
// Shared by the listing and the battle rather than written twice, for the reason
// the whole screen is one screen: the two differ in which skill and which list,
// and everything else about them — the heading, the position, the marked
// sentences, the footer — is the same answer. A second copy would be a second
// thing free to decide a status name is not worth marking.
func viewSkill(m model, declared skill.Skill, at, of int) (string, string) {
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.lang.GlossedSkill(declared)) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition, at, of)) + "\n\n")
	for _, line := range skillLines(m, declared) {
		out.WriteString(line + "\n")
	}
	return out.String(), m.text(i18n.BlurbFooter)
}

// skillLines is what a skill does, one line per sentence and already marked.
//
// A function rather than the second copy it was about to become: the picker
// reads the skill under its own cursor and this screen reads the one under the
// listing's, which is the same paragraph asked for from two places. A second
// copy of the marking is a second thing that can decide a name is worth
// pointing out — and the comment above this screen already refused to write a
// second copy of a framing for the same reason.
func skillLines(m model, declared skill.Skill) []string {
	// The status names these sentences will use, marked where they are printed,
	// exactly as the trait listing marks its own -- the two screens are read one
	// after the other, and a name that is bold on one and plain on the other reads
	// as a difference between the effects rather than between the screens. A miss
	// in the glossary drops out here rather than marking a bare id.
	names := make([]string, 0, 4)
	for _, id := range i18n.StatusesInSkill(declared) {
		if name := m.lang.Gloss(id); name != "" {
			names = append(names, name)
		}
	}
	out := make([]string, 0, 4)
	for _, line := range strings.Split(m.lang.Describe(declared, m.lib.Patterns()), "\n") {
		out = append(out, "  "+marked(line, names, func(word string) string {
			return m.style.emphasis.Render(word)
		}))
	}
	return out
}

// viewTraits is what the character under the browser's cursor carries, at the
// level the browser is sitting on.
//
// The level is read rather than assumed for the reason the detail pane behind it
// reads it: a trait comes in at a level, so "what is this unit carrying" has no
// answer without one — and a screen that described every declared trait would be
// describing traits the character does not have yet.
//
// The form is progression.Furthest, which is what the detail pane resolves with.
// Two screens asking the same question have to ask it the same way, or walking
// from one to the other changes the answer for a reason nothing on either says.
func (b blurbScreen) viewTraits(m model) (string, string) {
	footer := m.text(i18n.BlurbTraitsFooter)
	rows := m.browse.rows()
	if len(rows) == 0 {
		return "  " + m.text(i18n.BrowseNothingHere) + "\n", footer
	}
	character := rows[clamp(m.browse.cursor, 0, len(rows)-1)]
	held := m.lib.KitPassives(character.PassivesAt(m.browse.level, progression.Furthest))

	var out strings.Builder
	out.WriteString(m.style.heading.Render(character.Name) + "  " +
		m.style.dim.Render(m.text(i18n.LabelAtLevel, m.browse.level)) + "\n\n")
	if len(held) == 0 {
		// A trait the character has not learned yet is the common case at a low
		// level, so this is a normal answer rather than an empty screen: the same
		// sentence DescribePassive gives for a trait that holds nothing, because
		// "carries no traits" is the same fact from either end.
		return out.String() + "  " + m.text(i18n.BlurbTraitNone), footer
	}

	body := traitLines(m, held)
	room := traitRoom(m)
	// Clamped here rather than where it is incremented, because the key that
	// moves it does not know how long the answer is: the answer is built from
	// the browser's cursor and level, and both can have moved since.
	scroll := clamp(b.scroll, 0, max(len(body)-room, 0))
	for _, line := range body[scroll:min(scroll+room, len(body))] {
		out.WriteString(line + "\n")
	}
	if len(body) > room {
		out.WriteString(m.style.dim.Render("  " + m.text(i18n.BlurbMore,
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
func traitLines(m model, held []passive.Passive) []string {
	out := make([]string, 0, 6*len(held))
	for index, one := range held {
		if index > 0 {
			out = append(out, "")
		}
		out = append(out, "  "+m.style.label.Render(m.lang.GlossedPassive(one)))
		out = append(out, traitSentences(m, one)...)
	}
	return out
}

// traitSentences is one trait's description, wrapped and indented under
// whatever named it above.
//
// Split off the loop above rather than duplicated into the picker, which reads
// a trait under a cursor of its own and carries the name in its heading instead
// of in the body: what the two share is the measure and the indent, and those
// are exactly the parts a second copy would be free to disagree about.
func traitSentences(m model, one passive.Passive) []string {
	// Wrapped to the floor rather than to the window, which is the opposite of
	// what m.wrapped does and is right for a different reason. Those rows carry
	// authored free text -- a biography, a kit of nine ids -- which has to go
	// somewhere and gets whatever width there is. These are the program's own
	// prose, and prose has a measure: a sentence run across a two-hundred-column
	// terminal is a line a reader loses their place in, and it is also a line
	// TestEveryWordingFitsTheMinimumWidth measures against the floor.
	room := minWidth - 1 - traitIndent
	out := make([]string, 0, 6)
	for _, sentence := range strings.Split(m.lang.DescribePassive(one), "\n") {
		for _, line := range wrapWords(sentence, max(room, 8)) {
			out = append(out, strings.Repeat(" ", traitIndent)+line)
		}
	}
	return out
}

// traitIndent is how far the sentences sit under the trait they belong to. Four
// rather than two, so a wrapped line is still visibly part of the trait above it
// and not the start of the next one.
const traitIndent = 4

// traitRoom is how many lines of sentences fit: the window, less the two the
// heading takes and the one the position line does.
//
// The picker's reading state shares it rather than counting again. The two spend
// their lines on the same four things in the same order, so two counts would be
// two answers to one question — and the one that was wrong would be the one that
// let frame cut the line saying there is more to read.
//
// The position line is counted whether or not it is drawn. Counting it only when
// it appears would make the answer one line taller the moment it fits, which is
// the shape of loop that flickers between two layouts on a window exactly at the
// boundary.
func traitRoom(m model) int {
	room := m.height - 4 - 3
	if room < 3 {
		return 3
	}
	return room
}
