// Package tui renders a battle for a terminal.
//
// Everything here is a pure function returning a string. Nothing reads input,
// writes output or holds state, which is what makes the rendering testable and
// keeps the client a renderer rather than a second copy of the rules.
//
// The event lines are built from the event alone, never from the battle. That
// constraint is deliberate: if a line cannot be drawn from the log, the log is
// missing something, and a renderer that reaches into the engine to fill the gap
// is one that will drift from it.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// Tags assigns each unit a two character label, stable in roster order, so the
// board can name a cell in the space a hex has.
func Tags(units []*battle.Unit) map[string]string {
	counts := map[hex.Side]int{}
	letters := map[hex.Side]string{hex.SideAlly: "A", hex.SideEnemy: "E"}
	out := make(map[string]string, len(units))
	for _, unit := range units {
		counts[unit.Side]++
		out[unit.ID] = fmt.Sprintf("%s%d", letters[unit.Side], counts[unit.Side])
	}
	return out
}

// Board draws the battlefield, labelling each occupied cell with its unit's tag.
func Board(fight *battle.Battle, tags map[string]string) string {
	occupied := make(map[hex.Offset]string, len(fight.Units()))
	for _, unit := range fight.Units() {
		if unit.Dead {
			continue
		}
		occupied[unit.Cell] = tags[unit.ID]
	}
	header := " c0  c1  c2  c3  c4  c5\n BK  MD  FR  FR  MD  BK\n"
	return header + hex.Render(func(cell hex.Offset) string { return occupied[cell] })
}

// Roster lists every unit with its health, tempo and active effects.
//
// The language is a parameter for the reason Detail's is: the column headings are
// words rather than ids, and a heading nobody can read is a column nobody can
// use. The rest of the table is ids and figures, which is why this was English
// for as long as it was.
//
// ⚠️ **The heading used to carry a rule as well as a label** — that an effect
// with no countdown beside it is permanent — and it does not any more. The rule
// itself has not gone: it is stated on the statuses catalogue, which is the
// screen a reader goes to when they want to know what an effect is and how long
// it lasts, and which both clients carry on their menu. → screen.StatusesScreen,
// i18n.StatusesNoCountdown.
//
// This is the narrow table, and it is what the program promises to draw: every
// window at all is at least MinWidth, and this fits that. RosterWide is the same
// table with three more columns and is only offered where there is room for it.
func Roster(lang i18n.Lang, fight *battle.Battle, tags map[string]string) string {
	return rosterTable(lang, fight, tags, false, nil)
}

// RosterWide is Roster with the three columns a player needs to decide who to
// hit first: the unit's element, its attack and its defence.
//
// ⚠️ **It is an addition and never a replacement.** The narrow table is what
// ships in a window at the floor, unchanged to the byte, because the three
// columns cost more room than the floor has: nothing here is cut to make space
// for them, so a reader of a narrow terminal loses nothing they had. What decides
// between the two is the window, and RosterIsWide is the whole of that rule.
//
// The three sit **after** the tempo and before the effects, so the narrow row is
// this row's own prefix: the columns a reader already knows stay where they were
// and the new ones arrive in a block, rather than the whole table shifting.
//
// The ink is how the element column is coloured, and a nil one draws it plain.
// → ElementInk for why this package takes a function rather than a palette.
func RosterWide(lang i18n.Lang, fight *battle.Battle, tags map[string]string, ink ElementInk) string {
	return rosterTable(lang, fight, tags, true, ink)
}

// RosterFor draws whichever of the two a window of this width has room for.
//
// It exists so that a caller holding a width does not write the comparison out
// itself — one declaration of the rule, as RosterIsWide's comment says.
func RosterFor(lang i18n.Lang, fight *battle.Battle, tags map[string]string, width int, ink ElementInk) string {
	return rosterTable(lang, fight, tags, RosterIsWide(width), ink)
}

// rosterTable is the one walk over the units, drawing whichever row shape it is
// asked for.
//
// One function rather than two, because everything except the format string and
// the fields fed into it is the same question asked twice — which unit, what its
// health bar says, what its effects come to — and two walks would be two places
// for "a dead unit reads as fallen" to be answered differently.
func rosterTable(lang i18n.Lang, fight *battle.Battle, tags map[string]string, wide bool, ink ElementInk) string {
	var b strings.Builder
	b.WriteString(rosterHeadingFor(lang, wide) + "\n")
	for _, unit := range fight.Units() {
		state := HealthBar(unit.HP, fight.MaxHP(unit))
		if unit.Dead {
			state = "fallen"
		}
		stats := fight.Stats(unit)
		if wide {
			// A fallen unit keeps its three new columns rather than blanking
			// them with its health bar: an element and a stat line are what the
			// unit is, and a reader looking back at who was on the board wants
			// them more than a reader watching it stand there does.
			fmt.Fprintf(&b, rosterWideRow,
				unitColumn(tags[unit.ID], unit.Name), state,
				stats[progression.Speed], elementCell(unit.Affinity, ink),
				stats[progression.Attack], stats[progression.Defense], Effects(unit))
			continue
		}
		fmt.Fprintf(&b, rosterRow,
			unitColumn(tags[unit.ID], unit.Name), state,
			stats[progression.Speed], Effects(unit))
	}
	return trimLines(b.String())
}

// rosterRow is the shape of one line of that table, written down once so the
// tests can measure the row rather than restate it.
//
// The first column used to be two — a tag padded to five cells and a name padded
// to twenty-one — and merging them is what bought this table the room it needed:
// twenty-six cells held a tag of two and a name of at most thirteen, so ten of
// them were never drawn on any row in any golden.
const rosterRow = "%-17s%-26s%4d   %s\n"

// rosterWideRow is the same row with the element, the attack and the defence
// between the tempo and the effects.
//
// Read against rosterRow it is that string with `%-8s%4d %4d   ` spliced in:
// the element column and its gap, the two stat columns with one cell between
// them, and the same three-cell gap the effects already stood behind. Everything
// before the splice is byte-identical, which is what makes the narrow row this
// one's prefix rather than a different table.
const rosterWideRow = "%-17s%-26s%4d   %-8s%4d %4d   %s\n"

// rosterElementRoom is the widest element cell that column can hold.
//
// Seven is a dual affinity as ElementCode writes one — two three-letter codes
// and the slash between them — and that is the bound rather than an observation,
// because element.Dual admits no third half. A single element is three. The
// eighth cell in the format is the gap after the column, not part of it.
//
// ⚠️ **It used to be fifteen, for `electric/ground` spelled out**, and the eight
// cells the codes give back are eight the wide table no longer has to find a
// window for: the row's ceiling and the threshold derived from it both move on
// their own, because both are read off the format above.
const rosterElementRoom = 7

// rosterElementCell is the whole of that column: the code and the gap after it.
//
// ⚠️ **elementCell pads to this rather than leaving the padding to the format,
// and it has to.** Go's `%-8s` pads by counting runes, and an inked cell carries
// escape sequences that are runes a terminal never draws — so a coloured cell
// would be counted as far past the column and padded by nothing, and every row
// carrying one would sit a few cells left of every row that did not. The verb
// stays in the format anyway: it is what the bound tests measure the column off,
// and it still pads a plain cell that arrived short.
const rosterElementCell = rosterElementRoom + 1

// elementCodeJoiner is what stands between the two halves of a dual affinity,
// and it is the slash Affinity.String already writes: the codes are shorter
// spellings of the same thing, so they are joined the same way.
const elementCodeJoiner = "/"

// elementCodeLength is how many letters of an element's id a code keeps.
//
// ⚠️ **Three, because two do not separate the book.** `grass` and `ground` share
// their first two letters, so a two-letter scheme has to invent a spelling for
// one of them — and an invented code is no longer something a reader can read
// back to an id. Three is the shortest length at which every declared element is
// its own prefix, which is what TestEveryElementCodeIsItsOwnIdShortened holds.
const elementCodeLength = 3

// ElementCode is how the roster's element column names one element.
//
// ⚠️ **It is an id shortened, never a word translated**, and that is the whole
// reason it may be the same in both languages. internal/i18n's own doc comment
// keeps element ids as they are in Vietnamese and in English, because they are
// what an author types and what the data files store; the gloss table beside it
// puts the Vietnamese *next to* the id — `grass/electric <cỏ/điện>` — rather than
// instead of it, precisely so the id stays readable. A code is the same id with
// its tail cut off, so `gra` is still `grass` to a reader of either language,
// where a code derived from a translation would name a different word on each.
//
// The column drew the names in full until this, and they cost fifteen cells to
// say what seven now say. What the colour adds is emphasis and never meaning:
// the goldens are recorded under NO_COLOR, so the code alone has to tell every
// element apart — which TestEveryElementDrawsItsOwnCodeWithNoColourAtAll holds.
//
// A value the enum does not have is drawn whole rather than shortened. Nothing
// in the book can produce one, and the alternative is worse than an overlong
// cell: `element(12)` cut to three letters is `ele`, which is electric's code,
// so the unreadable case would quietly name a real element.
func ElementCode(member element.Element) string {
	name := member.String()
	if !member.Valid() || utf8.RuneCountInString(name) <= elementCodeLength {
		return name
	}
	return string([]rune(name)[:elementCodeLength])
}

// ElementInk draws one element's code in whatever ink its caller has.
//
// ⚠️ **A function rather than a palette, because this package may not have
// one.** internal/tui renders a battle as text and knows nothing about styles,
// terminals or lipgloss; the screen layer owns the palette and the one table of
// element colours in the program. Handing that table in as a *table* would make
// this package depend on the styling library, and re-deriving the colours here
// would be a second copy of a table whose entries were chosen with reasons. So
// the caller keeps the ink and this package keeps the table — the same division
// Detail already makes with a language it cannot know.
//
// ⚠️ **The ink is applied to the code alone and never to the padding**, which is
// what elementCell relies on to measure the column: the plain width of a cell has
// to be knowable without asking what the ink did to it.
type ElementInk func(member element.Element, code string) string

// elementCell is one unit's element column, padded to its full width.
//
// Each half of a dual affinity is inked on its own, because the colour is about
// the element rather than about the row: a `grass/electric` unit is half green
// and half yellow, and inking the pair in the primary's colour would say
// something untrue about the other half.
//
// The padding is counted off the codes rather than off the drawn cell for the
// reason rosterElementCell gives — an escape sequence is runes nobody sees — and
// a cell already at or past its width is left alone, so an element the enum does
// not have pushes its own row right exactly as an over-long name does.
func elementCell(affinity element.Affinity, ink ElementInk) string {
	var drawn strings.Builder
	plain := 0
	for index, member := range affinity.Elements() {
		if index > 0 {
			drawn.WriteString(elementCodeJoiner)
			plain += utf8.RuneCountInString(elementCodeJoiner)
		}
		code := ElementCode(member)
		plain += utf8.RuneCountInString(code)
		if ink != nil {
			code = ink(member, code)
		}
		drawn.WriteString(code)
	}
	if pad := rosterElementCell - plain; pad > 0 {
		drawn.WriteString(strings.Repeat(" ", pad))
	}
	return drawn.String()
}

// rosterStatRoom is the room the attack and defence columns each have.
//
// ⚠️ **There IS a bound on a buffed stat, and it is not the ceiling.**
// modifier.Set.Stat saturates a change towards `ceiling * Headroom / 1000` and
// scale.Saturate never reaches its limit, so the widest figure the engine can
// produce is one below that — 2399 under the shipped books, four digits. The
// progression ceiling alone (800) would have said three, which is why the bound
// is derived from both books rather than from the one that sounds like a limit.
//
// ⚠️ **Four is the bound and not slack.** Go's `%4d` does not clip: a five-digit
// figure draws five cells and pushes the rest of its own row right, exactly as an
// over-long name would. That is held by
// TestEveryStatTheRosterCanDrawFitsItsColumn, which derives the bound from the
// shipped limits and bounds and names the figure that broke it, rather than by
// room nobody wrote down.
const rosterStatRoom = 4

// rosterNameRoom is the longest name that column can hold.
//
// The seventeen cells above are 2 + 1 + 13 + 1: the tag, the gap after it, this,
// and the one cell that keeps the health bar's `[` off the end of a name.
// ⚠️ Thirteen is exactly what the longest name in the data needs and no more,
// which is headroom the old twenty-one had and this does not — so the bound is
// held by `TestEveryNameTheRosterCanDrawFitsItsColumn` rather than by slack
// nobody wrote down. That test also derives the column width back out of the
// format string, so the two cannot part company.
const rosterNameRoom = 13

// rosterTagRoom is what unitColumn spends before the name: the tag and its gap.
const rosterTagRoom = 3

// rosterHeading draws the column headings over the columns rosterRow draws its
// fields in.
//
// ⚠️ **`hp` and `spd` are not asked of the catalog, and that is a rule rather
// than an omission.** internal/i18n's own doc comment keeps the six stat labels
// — hp atk def spd acc ddg — as they are in both languages, because they are
// what an author types and what the data files store; a translated one is a
// value nobody can match back to the file they are editing. Nothing else on this
// row is an id, so the rest is worded.
//
// ⚠️ **The heading over the effects column used to carry a rule as well as a
// label, and no longer does.** A permanent effect draws no countdown at all, so
// "no countdown means permanent" is a convention the row never states — and a
// convention nobody states is a fact a reader has to guess. It was said here,
// over the column it is about; it is now said on the **statuses catalogue**,
// which is the screen a reader opens to ask what an effect does and how long it
// lasts, and which both clients carry on their menu. The rule went to the
// vocabulary rather than to the table: it is read once and remembered, where this
// heading repeated it over every battle, in every window, at every width.
// → i18n.StatusesNoCountdown, screen.StatusesScreen.View.
//
// The first heading names both halves of the column it stands over, in the order
// they are drawn, because they are one column now — a heading for the whole of
// it rather than a label sitting on each part's first cell.
func rosterHeading(lang i18n.Lang) string {
	return fmt.Sprintf(rosterHeadingRow,
		lang.Text(i18n.RosterHeadingUnit), "hp", "spd",
		lang.Text(i18n.RosterHeadingEffects))
}

// rosterWideHeading is the same heading with the three columns named.
//
// ⚠️ **`atk` and `def` are untranslated for the reason `hp` and `spd` are**, and
// it is the same rule rather than a second one: internal/i18n's doc comment keeps
// the six stat labels — hp atk def spd acc ddg — as they are in both languages,
// because they are what an author types and what the data files store. A
// translated `atk` is a value nobody can match back to the file they are
// editing.
//
// The element **heading** is a word and is worded; the element **values** under
// it are ids and are not. That is not a split in the rule, it is the rule: a
// heading names a column to a reader and a cell names a thing in a data file.
// → ElementCode.
func rosterWideHeading(lang i18n.Lang) string {
	return fmt.Sprintf(rosterWideHeadingRow,
		lang.Text(i18n.RosterHeadingUnit), "hp", "spd",
		lang.Text(i18n.RosterHeadingElement), "atk", "def",
		lang.Text(i18n.RosterHeadingEffects))
}

// rosterHeadingFor is the heading over whichever row shape is being drawn.
func rosterHeadingFor(lang i18n.Lang, wide bool) string {
	if wide {
		return rosterWideHeading(lang)
	}
	return rosterHeading(lang)
}

// rosterHeadingRow is rosterRow with one verb changed: the speed is a
// left-aligned word here and a right-aligned number there, which is the only way
// a heading and the figures under it can differ and still be the same table.
//
// ⚠️ The two must put their columns in the same cells, and
// TestTheRosterHeadingStandsOverTheColumnsItNames measures that off both format
// strings rather than trusting this comment. Writing the heading out by hand is
// what used to leave `effects` one cell left of the effects.
const rosterHeadingRow = "%-17s%-26s%-4s   %s"

// rosterWideHeadingRow is rosterWideRow with the same one verb changed, three
// times over: the tempo, the attack and the defence are left-aligned words here
// and right-aligned numbers there.
const rosterWideHeadingRow = "%-17s%-26s%-4s   %-8s%-4s %-4s   %s"

// RosterIsWide reports whether a window this wide has room for the wide table.
//
// ⚠️ **This is the one declaration of that rule**, and every caller asks it
// rather than comparing a width to a number of its own. A second comparison is a
// second copy of an arithmetic that moves whenever a column does.
//
// ⚠️ **It must be asked at DRAW time, not when a battle is read.** A live screen
// takes its reading when a turn arrives, which on a ninety-second allowance can
// be a long while before the next one; a player who resizes in between would
// otherwise keep the layout chosen for a window they no longer have. That is why
// internal/screen renders both tables into its reading and picks here.
func RosterIsWide(width int) bool { return width >= rosterWideWidth }

// rosterWideWidth is the narrowest window RosterWide may be drawn in.
//
// ⚠️ **Derived from the format string, never typed.** The wide row's ceiling is
// measurable the same way the narrow one's is — every column padded to its own
// width, the effects column full — and a literal beside it would be a second copy
// of an arithmetic that changes whenever a column does. That is the objection
// that made the matchup marks read their thresholds off the element chart rather
// than restate them.
//
// TestTheWideRosterThresholdIsReadOffItsFormat holds the derivation, in the
// repository's AST-walking style: it reads this declaration back out of the
// source and fails on a literal, because no test comparing two numbers can tell
// a derivation from a constant that happens to agree with it today.
var rosterWideWidth = rosterWidth(rosterWideRow,
	"", "", 0, strings.Repeat("e", rosterElementRoom), 0, 0,
	strings.Repeat("e", effectsRoom))

// rosterWidth is the window a roster layout needs: the widest line its format
// can draw, plus the one cell every line in this program leaves empty.
//
// The empty cell is not decoration — a line filling the last column wraps on some
// terminals, which is why internal/screen measures its wordings against one less
// than the floor. Counted in runes, because Go's fmt pads a string verb by runes
// and a Vietnamese heading is fewer cells than it is bytes.
func rosterWidth(format string, fields ...any) int {
	return utf8.RuneCountInString(fmt.Sprintf(strings.TrimSuffix(format, "\n"), fields...)) + 1
}

// unitColumn draws the tag and the name in the one column they now share.
//
// The tag keeps the first cells and its own width, because it is how the log
// cross-references a unit — `A2 uses pummel` has to find its row at a glance,
// and a tag hunted for inside a name is a tag that has stopped working. A
// summoned unit arrives after Tags was taken and reaches this with no tag at
// all; the gap it leaves is kept rather than closed up, so such a row still
// reads as a unit with no tag rather than as one whose tag went missing.
func unitColumn(tag, name string) string {
	return fmt.Sprintf("%-2s %s", tag, name)
}

// HealthBar draws a health figure as a bar and a count.
func HealthBar(current, max int64) string {
	const width = 10
	if max <= 0 {
		return "-"
	}
	if current < 0 {
		current = 0
	}
	filled := int(current * width / max)
	// A unit that is alive at all keeps one mark, so an almost dead unit does
	// not read as a dead one.
	if filled == 0 && current > 0 {
		filled = 1
	}
	return fmt.Sprintf("[%s%s] %5d/%-5d",
		strings.Repeat("#", filled), strings.Repeat(".", width-filled), current, max)
}

// Effects summarises a unit's timed effects.
func Effects(unit *battle.Unit) string {
	snapshot := unit.Statuses.Snapshot()
	if len(snapshot) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(snapshot))
	for _, entry := range snapshot {
		part := entry.ID
		if entry.Stacks > 1 {
			part += fmt.Sprintf(" x%d", entry.Stacks)
		}
		// A permanent status has no countdown to print, and printing its zero
		// would read as "about to run out" for the one thing that never does.
		//
		// ⚠️ **It does not print ` (always)` either, and the absence IS the
		// notation.** Across the goldens a permanent entry outnumbers a timed one
		// 1123 to 192, so spelling the common case out cost nine cells over a
		// thousand times while the rare case already identifies itself — a timed
		// entry carries its own `(2t)`. What that trade owes a reader is a place
		// the convention is stated, and that is the **statuses catalogue**: it
		// stood over this column for two changes and moved to the screen that
		// exists to explain what an effect is. → i18n.StatusesNoCountdown.
		if !entry.Permanent {
			part += fmt.Sprintf(" (%dt)", entry.Remaining)
		}
		parts = append(parts, part)
	}
	return elided(parts, effectsRoom)
}

// effectsRoom is the room this column is allowed at the width the program
// promises to draw in.
//
// ⚠️ **It used to be everything the row had left, and it deliberately is not any
// more.** The row spends 50 cells before it — the merged tag-and-name column,
// the health bar, the speed and the gaps between them, all fixed by rosterRow —
// and the floor is 120 with one cell held back, so 69 are left. This column
// keeps the 60 it already had and the other 9 stay unspent, because a cap that
// is re-derived from whatever is left is a cap that immediately spends every
// cell the columns beside it give up: merging two columns would have bought the
// table nothing at all, since `elided` would simply have drawn one more effect.
//
// ⚠️ **The cheaper thing dropping ` (always)` bought is not width, it is entry
// cost.** A permanent entry fell from `phalanx x2 (always)` to `phalanx x2`, so
// the same 60 cells now hold roughly twice as many of them — which is what makes
// this column affordable to cut when the element, attack and defence columns
// arrive and need more than 9.
const effectsRoom = 60

// elided joins what fits and counts what did not.
//
// ⚠️ **The column had no bound at all until a squad could hold three composition
// bonuses at once.** One unit carrying `phalanx x2 (always)`, `tidewell x2
// (always)`, `kinship x2 (always)` and a poison draws 124 cells into a row that
// has 60, and `TestEveryWordingFitsTheMinimumWidth` is what said so. Bonuses
// stack by design and a per-element table is eight more of them, so the row was
// always going to meet this — the third bonus is simply where it did.
//
// ⚠️ That reading is kept as the reading it was, in the wording of the day. A
// permanent entry no longer draws ` (always)`, so those same four effects are 27
// cells cheaper — 97 against the 60, still over it, which is the point: making
// entries cheaper moved where the bound bites rather than removing it.
//
// The count is kept rather than the text truncated mid-word, because a reader who
// can see that two effects are hidden knows to open the unit; one who sees
// `kinsh…` knows only that something is broken. It is the same trade every
// listing in this repository makes about an ellipsis.
func elided(parts []string, room int) string {
	if len(parts) == 0 {
		return "-"
	}
	kept, width := 0, 0
	for _, part := range parts {
		// The separator is paid for by every part after the first, which is what
		// makes this arithmetic and not an estimate.
		next := len(part)
		if kept > 0 {
			next += len(", ")
		}
		// The marker has to fit too, or eliding would overflow the row it exists
		// to keep inside one. Reserved only while something is actually left over.
		reserve := 0
		if kept+1 < len(parts) {
			reserve = len(" +9")
		}
		if width+next+reserve > room {
			break
		}
		width += next
		kept++
	}
	// Nothing fits, which a row this narrow can reach: say how many there are
	// rather than drawing a blank that reads as "no effects".
	if kept == 0 {
		return fmt.Sprintf("+%d", len(parts))
	}
	drawn := strings.Join(parts[:kept], ", ")
	if kept < len(parts) {
		drawn += fmt.Sprintf(" +%d", len(parts)-kept)
	}
	return drawn
}

// Order shows the units due to act next.
func Order(queue *atb.Queue, tags map[string]string, count int) string {
	turns := queue.Preview(count)
	if len(turns) == 0 {
		return "next: -"
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		label := tags[turn.ID]
		if label == "" {
			label = turn.ID
		}
		parts = append(parts, label)
	}
	return "next: " + strings.Join(parts, " ")
}

// Menu lists what the acting unit may do. Unavailable options are shown with
// their reason rather than hidden, because a player deciding what to do needs to
// know a skill exists and is two turns away.
func Menu(fight *battle.Battle, prompt *battle.Prompt, tags map[string]string) string {
	var b strings.Builder
	books := fight.Books()
	for index, option := range prompt.Options {
		declared, err := books.Skills.Lookup(option.Skill)
		if err != nil {
			fmt.Fprintf(&b, " %d) %-16s unknown\n", index+1, option.Skill)
			continue
		}
		label := fmt.Sprintf(" %d)", index+1)
		if !option.Available() {
			label = "  -"
		}
		fmt.Fprintf(&b, "%s %-16s%-10s rng %-3d %-13s pow %-6d acc %-5d",
			label, declared.ID, declared.Element, declared.Range,
			declared.Pattern, declared.TotalPower(), declared.Accuracy)
		if declared.StrikeCount() > 1 {
			fmt.Fprintf(&b, " x%d", declared.StrikeCount())
		}
		if declared.Cooldown > 0 {
			fmt.Fprintf(&b, " cd%d", declared.Cooldown)
		}
		if extra := Extras(declared); extra != "" {
			fmt.Fprintf(&b, "  %s", extra)
		}
		if !option.Available() {
			// ⚠️ **The engine's English sentence on purpose, not an oversight.**
			// screen.OptionRefusal says the same four facts in the reader's own
			// language, off battle.Block and the counts beside it — and it needs a
			// Lang to do it. This function is handed an event and a book and
			// nothing else, which is the property that lets a replay be drawn with
			// no library at all, so there is no language here to ask. Every other
			// column on this line is a bare id or a number for the same reason.
			fmt.Fprintf(&b, "  <%s>", option.Reason)
		}
		b.WriteString("\n")
	}
	_ = tags
	return trimLines(b.String())
}

// trimLines drops the padding a column layout leaves at the end of a line, so
// what is printed has no invisible tail.
func trimLines(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// Extras summarises what a skill does beyond damage.
func Extras(declared skill.Skill) string {
	parts := make([]string, 0, 4)
	for _, application := range declared.Applies {
		parts = append(parts, fmt.Sprintf("%s %d%%", application.Status, application.Chance/10))
	}
	for _, application := range declared.SelfApplies {
		parts = append(parts, fmt.Sprintf("self %s x%d", application.Status, application.Stacks))
	}
	if declared.Requires != nil {
		verb := "needs"
		if declared.Requires.Consume {
			verb = "eats"
		}
		// A condition reads a status, or how hurt the target is, or both, so the
		// clause is assembled rather than formatted in one go: the old single
		// line printed the status field unconditionally and would render a
		// health-only condition as "needs  x0", which reads as a bug in the
		// skill rather than as a skill this line cannot describe.
		clauses := make([]string, 0, 2)
		if declared.Requires.ReadsStatus() {
			clauses = append(clauses, fmt.Sprintf("%s x%d",
				declared.Requires.Status, declared.Requires.MinStacks))
		}
		if declared.Requires.ReadsHealth() {
			clauses = append(clauses, fmt.Sprintf("health <=%d%%", declared.Requires.BelowHealth/10))
		}
		parts = append(parts, fmt.Sprintf("%s %s for +%d",
			verb, strings.Join(clauses, " and "), declared.Requires.BonusPower))
	}
	if declared.Strips != nil {
		names := make([]string, 0, len(declared.Strips.Categories))
		for _, category := range declared.Strips.Categories {
			names = append(names, category.String())
		}
		sort.Strings(names)
		parts = append(parts, fmt.Sprintf("strips %s x%d", strings.Join(names, "/"), declared.Strips.Stacks))
	}
	return strings.Join(parts, ", ")
}

// Aims lists the cells a chosen skill may be pointed at, with who is standing
// there and what the shape would catch.
//
// caster is the half of the board the unit about to cast stands on, and it is
// the frame the shape's steps are read in — not the side the aim lands on. It is
// a parameter because this draws for whoever is being asked: a shape spreads
// away from its caster, so drawing an enemy's aim in the ally's frame would list
// cells the resolution does not touch. → package pattern's doc, TODO.md ENG-012.
func Aims(fight *battle.Battle, option battle.Option, tags map[string]string,
	caster hex.Side) string {
	var b strings.Builder
	books := fight.Books()
	declared, err := books.Skills.Lookup(option.Skill)
	if err != nil {
		return "unknown skill"
	}
	shape, err := books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return "unknown shape"
	}
	occupied := make(map[hex.Offset]*battle.Unit, len(fight.Units()))
	for _, unit := range fight.Units() {
		if !unit.Dead {
			occupied[unit.Cell] = unit
		}
	}
	for index, aim := range option.Aims {
		caught := make([]string, 0, shape.MaxTargets())
		for _, cell := range shape.Targets(aim, caster) {
			unit, standing := occupied[cell]
			if !standing {
				continue
			}
			caught = append(caught, fmt.Sprintf("%s %s", tags[unit.ID], unit.Name))
		}
		fmt.Fprintf(&b, " %d) %-6s%s\n", index+1, aim, strings.Join(caught, ", "))
	}
	return trimLines(b.String())
}

// Line renders one event, using nothing but the event.
//
// glosses is data id -> display name, for the skill, status and trait ids the
// line prints — i18n.Lang.LogGlosses is what builds one. It is a caller-supplied
// map for exactly the reason tags and Summary's names are: the event lines are
// built from the event alone, so this package may not be handed books, a battle
// or a language, and a name is a fact about the reader rather than about what
// happened. **A nil or empty map reproduces the line byte for byte**, which is
// both what English is (a data id is shown as the data writes it) and what a
// replay drawn without books is, and it is the property the goldens hold.
//
// Every occurrence is glossed rather than the first mention: a log is read in
// pieces, scrolled and frozen at a frame, so "the first mention" is a row a
// reader may never have on screen.
func Line(event battle.Event, tags, glosses map[string]string) string {
	tag := func(id string) string {
		if id == "" {
			return ""
		}
		if label, ok := tags[id]; ok {
			return label
		}
		return id
	}
	// gloss is the id with its name beside it, and the bare id when there is
	// none. The bracket comes from i18n rather than being spelled here, so the log
	// and every screen that names a data id agree on the punctuation.
	gloss := func(id string) string {
		if id == "" {
			return ""
		}
		return i18n.GlossBracket(id, glosses[id])
	}
	head := fmt.Sprintf("  %-3s", tag(event.Actor))
	switch event.Kind {
	case battle.Started:
		name := event.Name
		if name == "" {
			name = event.Actor
		}
		return fmt.Sprintf("  %-3s %s enters at %s on the %s side with %d health, %s",
			tag(event.Actor), name, event.Cell, event.Side, event.Amount, event.Note)
	case battle.TurnBegan:
		return fmt.Sprintf("\n  %-3s turn %d", tag(event.Actor), event.Turn)
	case battle.StatusTicked:
		return head + fmt.Sprintf(" takes %d from %s x%d", event.Amount, gloss(event.Status), event.Stacks)
	case battle.Healed:
		// A regeneration names itself; a drain or a restoring skill has no
		// status to name and says only how much came back.
		if event.Status != "" {
			return head + fmt.Sprintf(" heals %d from %s x%d%s", event.Amount,
				gloss(event.Status), event.Stacks, reducedNote(event.Reduced))
		}
		// A drain says the share it took, because the amount alone cannot be
		// reproduced from the skill any more: a trait may drain as well, so the
		// number on screen is not the skill's own figure applied to the damage.
		if event.Drained > 0 {
			return head + fmt.Sprintf(" drains %d, %d%% of what it dealt, %d hp left%s",
				event.Amount, event.Drained/10, event.Remaining, reducedNote(event.Reduced))
		}
		return head + fmt.Sprintf(" heals %d, %d hp left%s", event.Amount, event.Remaining,
			reducedNote(event.Reduced))
	case battle.StatusExpired:
		return head + fmt.Sprintf(" %s wears off", gloss(event.Status))
	case battle.SpeedChanged:
		return head + fmt.Sprintf(" speed %d to %d", event.Before, event.Amount)
	case battle.TurnSkipped:
		return head + fmt.Sprintf(" loses the turn (%s)", event.Note)
	case battle.SkillUsed:
		return head + fmt.Sprintf(" uses %s at %s%s", gloss(event.Skill), event.Cell,
			gradientNote(event.Gradient))
	case battle.Amplified:
		return head + fmt.Sprintf("  %s amplified by %s x%d, power x%s",
			gloss(event.Skill), gloss(event.Status), event.Stacks, multiple(event.Power))
	case battle.Spread:
		return head + fmt.Sprintf("  %s jumps off %s carrying %s",
			gloss(event.Skill), tag(event.Target), gloss(event.Status))
	case battle.StatusConsumed:
		// The stacks left where a consume took only some. Nought is both "took
		// the lot" and "there were none left", which are the same fact from the
		// reader's side, so the clause is dropped rather than printing a zero
		// that says nothing.
		if event.Remaining > 0 {
			return head + fmt.Sprintf("  consumes %s x%d off %s, giving up %d, %d left",
				gloss(event.Status), event.Stacks, tag(event.Target), event.Amount, event.Remaining)
		}
		return head + fmt.Sprintf("  consumes %s x%d off %s, giving up %d",
			gloss(event.Status), event.Stacks, tag(event.Target), event.Amount)
	case battle.Missed:
		return head + fmt.Sprintf("  misses %s (%d%%)", tag(event.Target), event.Chance/10)
	case battle.Blocked:
		return head + fmt.Sprintf("  is blocked by %s, %d charges left", tag(event.Target), event.Remaining)
	case battle.Paid:
		// Beside the guard's line because both are health moving for a reason a
		// strike does not explain, and worded so it cannot be read as damage:
		// this is the caster handing something over, not somebody taking it.
		return head + fmt.Sprintf(" pays %d for %s, %d hp left",
			event.Amount, gloss(event.Skill), event.Remaining)
	case battle.Split:
		// Worded so it cannot be read as the price above it. A payment leaves the
		// caster the same creature and a heal puts it back; this says the maximum
		// itself moved, and the figure after it is what the caster is now — not
		// what it has left, which is the number `pays` prints and the one a reader
		// would otherwise assume.
		return head + fmt.Sprintf("  gives %d of itself to %s, %d hp at most now",
			event.Amount, tag(event.Target), event.Remaining)
	case battle.Absorbed:
		// A line of its own beside the block above, and the two words are chosen
		// to be unmistakable: a blocked strike is stopped and this one is eaten.
		// The figure after it is what the barrier has left rather than a charge
		// count, which is the whole difference between the two guards said in the
		// unit each is measured in.
		return head + fmt.Sprintf("  %d soaked by %s, %d left in the barrier",
			event.Amount, tag(event.Target), event.Remaining)
	case battle.Damaged:
		// A reply is damage, and the only thing that separates it from a strike
		// in the log is that a trait rather than a skill is named — so it is the
		// same case, worded so a reader can tell that this happened on somebody
		// else's turn. Reading it as damage-with-no-skill would work and would
		// be a rule nobody wrote down.
		if event.Passive != "" {
			return head + fmt.Sprintf("  answers %s with %s for %d%s, %d left",
				tag(event.Target), gloss(event.Passive), event.Amount,
				affinityNote(event.Multiplier), event.Remaining)
		}
		return head + fmt.Sprintf("  hits %s for %d%s%s%s, %d left",
			tag(event.Target), event.Amount, affinityNote(event.Multiplier),
			pierceNote(event.Pierce), criticalNote(event.Critical), event.Remaining)
	case battle.StatusApplied:
		note := ""
		if event.Note != "" {
			note = ", " + event.Note
		}
		// The trait is named where there is one, for the same reason the damage
		// above names it: a status arriving on the attacker's own turn, from the
		// unit it just hit, has nothing else in the log to account for it.
		source := ""
		if event.Passive != "" {
			source = " (" + gloss(event.Passive) + ")"
		}
		// A vulnerability shows on the line where the status lands, because that
		// is the case it exists to cause — the refusal arm below never runs for a
		// target that made itself easier to hit and still got lucky.
		if event.Refused < 0 {
			source += fmt.Sprintf(" (%d%% invited)", -event.Refused/10)
		}
		return head + fmt.Sprintf("  %s x%d on %s, now %d%s%s",
			gloss(event.Status), event.Stacks, tag(event.Target), event.Remaining, note, source)
	case battle.StatusResisted:
		// Two different things end an application, and the kind is called
		// status_resisted for both: the roll failed, or the target's traits
		// refused it. Saying which is the whole reason the event carries the
		// share refused — a reader given only "resists" cannot tell a piece of
		// luck from a property of the unit.
		switch {
		case event.Refused >= 1000:
			return head + fmt.Sprintf("  %s is immune to %s", tag(event.Target), gloss(event.Status))
		case event.Refused > 0:
			return head + fmt.Sprintf("  %s shrugs off %s (%d%% chance, %d%% refused)",
				tag(event.Target), gloss(event.Status), event.Chance/10, event.Refused/10)
		case event.Refused < 0:
			// A vulnerability, which is a refusal of a negative share. Without
			// this line the roll below prints a chance higher than the skill's
			// own and nothing on screen says why — and explaining its own figures
			// is the whole job of this renderer.
			return head + fmt.Sprintf("  %s is wide open to %s (%d%% chance, %d%% invited)",
				tag(event.Target), gloss(event.Status), event.Chance/10, -event.Refused/10)
		default:
			return head + fmt.Sprintf("  %s resists %s (%d%%)",
				tag(event.Target), gloss(event.Status), event.Chance/10)
		}
	case battle.StatusStripped:
		return head + fmt.Sprintf("  strips %d off %s", event.Stacks, tag(event.Target))
	case battle.PassiveHeld:
		// A trait and the permanent status it put on, in one line, because
		// either half alone leaves the reader guessing: the trait's name says
		// nothing about what it does, and the status appearing on its own has
		// nothing to account for it.
		return head + fmt.Sprintf("  holds %s: %s x%d", gloss(event.Passive), gloss(event.Status), event.Stacks)
	case battle.BonusHeld:
		// What the squad shared, how many shared it, and what that paid for. The
		// count is on the line rather than left to a reader to count off the
		// board, because the board shows who is standing and not which rung was
		// reached — and the two differ the moment a bonus is sharers-only.
		return head + fmt.Sprintf("  %d of %s: %s x%d (%s)",
			event.Count, gloss(event.Shared), gloss(event.Status), event.Stacks, gloss(event.Bonus))
	case battle.PassiveReleased:
		// The same line the other way round. A gated trait letting go takes a
		// visible number down with it, so it reads beside the heal that caused
		// it rather than being left to the reader to infer from the damage.
		return head + fmt.Sprintf("  lets go of %s: %s x%d", gloss(event.Passive), gloss(event.Status), event.Stacks)
	case battle.Died:
		return head + fmt.Sprintf(" falls at %s", event.Cell)
	case battle.Summoned:
		// The new unit rather than the caster, and its health with it: this is
		// the line that introduces somebody, so it carries what a started line
		// carries — a reader meeting a name for the first time needs the same
		// facts whether the roster placed it or a skill did.
		return head + fmt.Sprintf("  %s calls up %s at %s, %d hp",
			gloss(event.Skill), event.Target, event.Cell, event.Amount)
	case battle.Left:
		// Not "falls". A copy running out of turns is not a unit being beaten,
		// and the note says which of the two reasons it was.
		return head + fmt.Sprintf(" leaves at %s (%s)", event.Cell, event.Note)
	case battle.Ended:
		// Every ending is drawn from the outcome rather than from the winner,
		// because three of the four have no winner to name and one of them —
		// a stalemate — leaves units standing on both sides. Drawing that as
		// "nobody is left" would be the log telling the reader something false.
		switch event.Outcome {
		case battle.Victory:
			return fmt.Sprintf("\n  the %s side wins", event.Side)
		case battle.Annihilation:
			return "\n  a draw: nobody is left standing"
		case battle.Stalemate:
			return "\n  a draw: nobody can reach anyone"
		default:
			return "\n  the battle ends"
		}
	default:
		return head + " " + event.Kind.String()
	}
}

// multiple spells a parts-per-thousand power as the multiplier it actually is,
// because the raw figure is the only number on the line a reader has to divide
// by a thousand before it means anything. A power of 3500 reads as x3.5 and a
// power of 1000 as x1; trailing zeroes are trimmed so the common figures stay
// short, and no number is rounded, because a thousandth is the smallest a power
// can be and the log must stay reproducible against the rules.
//
// ASCII x rather than a multiplication sign: an ambiguous-width glyph is
// measured as one cell and drawn as two in enough terminals to overlap the
// column beside it.
func multiple(permille int) string {
	sign := ""
	if permille < 0 {
		sign, permille = "-", -permille
	}
	whole, fraction := permille/scale.Base, permille%scale.Base
	if fraction == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	return fmt.Sprintf("%s%d.%s", sign, whole,
		strings.TrimRight(fmt.Sprintf("%03d", fraction), "0"))
}

// affinityNote spells out an elemental multiplier, and says nothing when the
// matchup is neutral.
func affinityNote(multiplier int) string {
	switch {
	case multiplier == 0 || multiplier == 1000:
		return ""
	case multiplier >= 2000:
		return " (doubly weak!)"
	case multiplier > 1000:
		return " (weak)"
	case multiplier <= 500:
		return " (doubly resisted)"
	default:
		return " (resisted)"
	}
}

// pierceNote says how much of the target's armour a hit went through, and says
// nothing at all when it went through none.
//
// It is on the line because a reader who cannot see it cannot account for the
// damage: the same attacker, power and multiplier against the same defender
// produce a different figure, and a log a reader cannot reproduce is the log
// lying. The share rather than the defence it left, because the share is the
// skill's own property and the defence is the defender's.
// gradientNote is what a hurt caster added to its own skill, and it is worded as
// a share rather than a power because the share is the part a reader cannot
// work out: the power on the event is the book's figure, and the strike that
// follows is larger than it with nothing else to say why.
func gradientNote(gradient int) string {
	if gradient <= 0 {
		return ""
	}
	return fmt.Sprintf(" (hurt, +%d%%)", gradient/10)
}

// reducedNote is what the healed unit's own statuses took off the number in
// front of it, and says nothing at all when they took nothing.
//
// It is on the line for the reason pierceNote and gradientNote are, and it is the
// least optional of the three: a heal of 244 off a skill the book prints as 900
// leaves a reader with no figure on the screen or in the data that agrees with
// it. The share rather than the amount lost, because the share is the property of
// the statuses on the unit and the amount is a consequence of whatever was aimed
// at it.
func reducedNote(reduced int) string {
	if reduced <= 0 {
		return ""
	}
	return fmt.Sprintf(" (healing cut %d%%)", reduced/10)
}

func pierceNote(pierce int) string {
	if pierce <= 0 {
		return ""
	}
	if pierce >= 1000 {
		return " (straight through the armour)"
	}
	return fmt.Sprintf(" (through %d%% of the armour)", pierce/10)
}

// criticalNote marks a strike that landed critically. Like pierceNote it names
// no figure: what a critical is worth is one constant the rules hold, and this
// renderer reads events rather than the rules.
func criticalNote(critical bool) string {
	if !critical {
		return ""
	}
	return " (critical)"
}

// Log renders a run of events. glosses is Line's, and a nil one renders exactly
// what this function rendered before there was a third parameter.
func Log(events []battle.Event, tags, glosses map[string]string) string {
	if len(events) == 0 {
		return ""
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, Line(event, tags, glosses))
	}
	return strings.Join(lines, "\n")
}

// Tally is what one unit did and had done to it, counted from the event log.
type Tally struct {
	ID     string
	Name   string
	Side   hex.Side
	Turns  int
	Dealt  int64
	Taken  int64
	Ticked int64
	Healed int64
	Hits   int
	Misses int
	Walled int
	Kills  int
	Fell   bool
}

// Tallies reads a battle's whole log and counts what each unit did.
//
// It is built from the events alone, with no access to the battle, which is the
// same constraint the event lines are held to. Anything a summary needs that the
// log cannot supply is something the log is missing.
func Tallies(events []battle.Event) []Tally {
	order := make([]string, 0, 10)
	byID := make(map[string]*Tally, 10)
	// A kill is credited to whoever last hurt the unit that fell, which the log
	// carries even though no event says "killed by".
	lastHarm := make(map[string]string, 10)

	touch := func(id string) *Tally {
		if id == "" {
			return nil
		}
		if existing, ok := byID[id]; ok {
			return existing
		}
		byID[id] = &Tally{ID: id, Name: id}
		order = append(order, id)
		return byID[id]
	}

	for _, event := range events {
		actor, target := touch(event.Actor), touch(event.Target)
		switch event.Kind {
		case battle.Started:
			actor.Side = event.Side
		case battle.TurnBegan:
			actor.Turns++
		case battle.StatusTicked:
			actor.Ticked += event.Amount
			actor.Taken += event.Amount
		case battle.Healed:
			actor.Healed += event.Amount
		case battle.Damaged:
			actor.Dealt += event.Amount
			actor.Hits++
			if target != nil {
				target.Taken += event.Amount
				lastHarm[target.ID] = actor.ID
			}
		case battle.Missed:
			actor.Misses++
		case battle.Blocked:
			if target != nil {
				target.Walled++
			}
		case battle.Died:
			actor.Fell = true
			if killer, known := lastHarm[actor.ID]; known && killer != actor.ID {
				if credited := byID[killer]; credited != nil {
					credited.Kills++
				}
			}
		}
	}

	out := make([]Tally, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

// Summary renders the tallies as a table.
func Summary(events []battle.Event, tags map[string]string, names map[string]string) string {
	tallies := Tallies(events)
	if len(tallies) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("tag  unit                 turns    dealt    taken   effects   hits   miss  walled  kills\n")
	for _, tally := range tallies {
		name := names[tally.ID]
		if name == "" {
			name = tally.Name
		}
		if tally.Fell {
			name += " (fell)"
		}
		fmt.Fprintf(&b, "%-5s%-21s%6d%9d%9d%10d%7d%7d%8d%7d\n",
			tags[tally.ID], name, tally.Turns, tally.Dealt, tally.Taken,
			tally.Ticked, tally.Hits, tally.Misses, tally.Walled, tally.Kills)
	}
	return trimLines(b.String())
}

// Names maps unit ids to their display names, for a summary rendered from a log
// that carries ids rather than names.
func Names(units []*battle.Unit) map[string]string {
	out := make(map[string]string, len(units))
	for _, unit := range units {
		out[unit.ID] = unit.Name
	}
	return out
}

// NamesFromLog reads the display names out of a log's opening records, so a
// saved battle renders with the names it was fought under.
func NamesFromLog(events []battle.Event) map[string]string {
	out := make(map[string]string, 10)
	for _, event := range events {
		if event.Kind == battle.Started && event.Actor != "" && event.Name != "" {
			out[event.Actor] = event.Name
		}
	}
	return out
}

// TagsFromLog assigns tags using only the log's opening records, so a saved
// battle can be rendered without the roster it was fought with. It is the same
// constraint the event lines hold to, applied to the labels.
func TagsFromLog(events []battle.Event) map[string]string {
	counts := map[hex.Side]int{}
	letters := map[hex.Side]string{hex.SideAlly: "A", hex.SideEnemy: "E"}
	out := make(map[string]string, 10)
	for _, event := range events {
		if event.Kind != battle.Started || event.Actor == "" {
			continue
		}
		if _, already := out[event.Actor]; already {
			continue
		}
		counts[event.Side]++
		out[event.Actor] = fmt.Sprintf("%s%d", letters[event.Side], counts[event.Side])
	}
	return out
}
