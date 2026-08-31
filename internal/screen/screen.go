// Package screen is what every full-screen view in this repository's terminal
// clients needs, and none of what any one of them decides.
//
// Two clients draw the same reference screens — the authoring tool
// (`cmd/hexforge-tui`) and the PvP game client — so what lives here is the
// screens themselves plus everything they are drawn out of: the palette, the
// floor a window has to clear, the row-drawing helpers every pane is built from,
// and the six read-only references (the affinity chart and its rings, the
// elements, statuses, species, builds and traits listings) with their cursors
// and their keystrokes.
//
// What it still excludes is everything a *client* decides. A screen is handed a
// Context, which holds nothing the client owns, and answers a keystroke with an
// Action — what it wants — so it never names a view, a menu entry or a way back:
// see action.go for why that vocabulary is four Kinds and no more. The frame
// around a body, which screen it is showing and where Back goes are the client's,
// and a screen here may not ask.
//
// It reads no environment and takes no decision from one. `Plain` answers
// whether colour would be noise, and the **binary** is what hands it the three
// inputs — reading `os.Getenv` is the binary's business, and a package two
// clients share may not have an opinion about which of them is running.
package screen

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// Context is what a screen draws with: the books, the language, the palette and
// the window.
//
// It is a value, copied per draw, and holds nothing a screen owns — no cursor,
// no mode, no pending question. That is what makes it shareable: a client keeps
// its own model and hands one of these down, so the helpers below cannot be told
// anything about which client is asking.
type Context struct {
	Lib   *forge.Library
	Lang  i18n.Lang
	Style Palette

	Width, Height int
}

// The smallest window the screens fit in.
//
// Below this the program says so rather than drawing a layout that overlaps
// itself: a corrupted screen looks like a bug in the tool, and the author is
// left unable to tell whether the character they are looking at is the one they
// typed.
//
// The width was 72 while this client was English only, and moved to 80 when
// Vietnamese arrived: it runs a fifth to a third longer for the same sentence,
// and the busiest footers — six chords on the form, the level and filter keys on
// the browser — landed just past 72 once the language chord was added. That is
// where 80 came from, and it is not where 120 comes from, so the argument is
// rewritten rather than extended: 80 was the other number a terminal has always
// had, and 120 is a number the measurement asked for.
//
// **The measurement.** Every screen in everyScreen rendered at 200x60 in both
// languages, and the widest line the width sweep actually constrains — free text
// and data columns are exempt, being the two things that have no length the
// program can promise. Of the 92 screen/language pairs, **34 sit at 76–79 cells**
// of the 79 the old floor left, 29 more at 70–75, 17 at 60–69 and 12 below 60. A
// third of the client is pressed flat against the ceiling, and several of those
// lines landing on 78 and 79 *exactly* is the fingerprint of wording trimmed to
// fit rather than wording that happened to end there.
//
// **What is pinned is almost entirely footers** — rows of key chords, e.g.
// `space pick · ↑/↓ move · ? describes · enter done · esc back · ctrl+l tiếng
// Việt` at exactly 79 — plus the menu's detail column and the skill form's damage
// reading. ⚠️ **A footer cannot be given room any other way.** It is catalog
// wording, so the prose/data split does not reach it: measuring one against the
// window instead of the floor would cut it again on an 80-column terminal, which
// is the failure this sweep exists to prevent. The floor is therefore the only
// lever for this class, which is why widening the data cells bought it nothing —
// measured across #173 and #175, **35 pairs were packed against the ceiling
// before those two changes and 34 after**.
//
// ⚠️ **The cost, stated rather than buried: a terminal narrower than 120 no
// longer draws this program at all.** That is a real loss, accepted because the
// alternative was trimming the wording a third time. It is also the smaller half
// of the tool: `hexforge` needs no room and does everything this front-end does,
// which is what viewTooSmall already points at.
//
// TestEveryWordingFitsTheMinimumWidth measures the catalog against this constant
// in both languages, so it cannot rot quietly. ⚠️ Raising the floor **loosens**
// that sweep, and deliberately: the promise changed, so every existing line
// passing trivially is the new promise being kept rather than a test going
// vacuous. What it does not loosen is the vertical budgets — prose wraps at this
// number, so a screen reserving rows for a wrapped block has to **measure** the
// wrap rather than count it, which is what speciesRoom and passivesRoom do.
const (
	MinWidth  = 120
	MinHeight = 24
)

// detailLabels is every row name a detail pane draws, which is what its label
// column is measured from. The level row is not in it because it is named after
// a number; DetailLabelWidth measures that one separately.
var detailLabels = []i18n.Key{
	i18n.LabelFrom, i18n.LabelPlaystyle, i18n.LabelElement, i18n.LabelKit,
	i18n.LabelArt, i18n.LabelStages, i18n.LabelBiography, i18n.LabelEffectiveHP,
	i18n.LabelNote, i18n.LabelIntent,
}

// DetailLabelWidth is the column every detail pane's row name sits in,
// measured from the widest name being drawn in the language in front. The extra
// column is the gap, so the widest label is still followed by a space.
//
// It was a constant 11, and that is what went wrong: a constant is one number
// for two languages, so it is only ever right for both by luck. The luck ran
// out on two labels at once — "effective hp" is 12 cells and "nguồn tham khảo"
// is 15 — and a label past its column pushes that one row's value right of
// every other row's, which is the same misalignment the form's summary rows had
// before formLabelWidth measured itself. Measure, do not raise the constant:
// raising it would waste the difference in whichever language is shorter, and
// leave the next reworded label to break it again.
func (c Context) DetailLabelWidth() int {
	widest := 0
	for _, key := range detailLabels {
		if width := lipgloss.Width(c.Text(key)); width > widest {
			widest = width
		}
	}
	// The level row is named after a number, and the widest one is the cap.
	if width := lipgloss.Width(c.Text(i18n.LabelAtLevel, progression.LevelCap)); width > widest {
		widest = width
	}
	return widest + 1
}

// Text is one line in the language in front. Every screen goes through it.
func (c Context) Text(key i18n.Key, args ...any) string { return c.Lang.Say(key, args...) }

// Label draws a "name  value" row the way every detail pane in this program
// does, so the panes line up with each other.
func (c Context) Label(name, format string, args ...any) string {
	return c.LabelAt(name, c.DetailLabelWidth(), format, args...)
}

// Continued draws a row that carries on from the one above it: no name of its
// own, and its value under the same column as that row's.
//
// The kit's Vietnamese names are what this exists for. Five skills glossed
// inline would be five brackets on one row, which does not fit the floor,
// so they go underneath in the same order instead — and they only line up with
// the ids above them if they are placed by the same measurement.
func (c Context) Continued(format string, args ...any) string {
	return c.LabelAt("", c.DetailLabelWidth(), format, args...)
}

// Wrapped is Label for a value with no bound on its length: it fills the row and
// carries on underneath, aligned with where the value started.
//
// Clipping was wrong for these in two ways at once. It cut at MinWidth, which is
// the *floor* a window has to clear rather than a ceiling on what one may use, so
// a hundred-cell terminal was being told it had seventy-nine. And a kit of nine
// ids or a paragraph of biography is longer than any terminal, so widening alone
// would not have saved it — the tail has to go somewhere, and a row below is
// where a reader looks for it.
//
// The rows this draws are variable in number, so it belongs only on a pane that
// can afford that. The form counts its rows to scroll them and must not use it.
func (c Context) Wrapped(name string, width int, value string) string {
	return c.WrappedIn(name, width, lipgloss.NewStyle(), value)
}

// WrappedIn is Wrapped with a style, applied one line at a time.
//
// One line at a time matters. Styling the whole block instead treats it as a box:
// lipgloss pads every line out to the width of the widest and swallows the
// trailing newline, so the row after it was appended to the end of a field of
// spaces and disappeared. The dim reading lost its dimness and the art row lost
// its existence, from one Render around the wrong thing.
//
// ⚠️ **The `- 2 - width - 1` is one cell short of what every other row spends,
// and it is moved here unchanged on purpose.** It lets a wrapped value fill the
// window's final column — the one column every other row leaves empty, because a
// line filling it wraps on some terminals. `TODO.md` records it as left alone
// deliberately: fixing it changes what fits, on every pane that wraps.
func (c Context) WrappedIn(name string, width int, style lipgloss.Style, value string) string {
	room := c.UsableWidth() - 2 - width - 1
	if room < 8 {
		// Narrower than this and wrapping makes a column of syllables; the
		// clip is the lesser evil.
		return c.LabelAt(name, width, "%s", style.Render(Clip(value, max(room, 1))))
	}
	lines := WrapWords(value, room)
	var out strings.Builder
	out.WriteString(c.LabelAt(name, width, "%s", style.Render(lines[0])))
	for _, line := range lines[1:] {
		out.WriteString(c.LabelAt("", width, "%s", style.Render(line)))
	}
	return out.String()
}

// UsableWidth is what a row may spend: the window when there is one, and the
// floor before the first size message arrives.
func (c Context) UsableWidth() int {
	if c.Width < MinWidth {
		return MinWidth
	}
	return c.Width
}

// LabelAt is Label in a caller-chosen column.
//
// The new-character form sizes its own column from the widest field name it is
// drawing, which differs per language ("archetype" is 9 cells, "mẫu vai trò" is
// 11), so its summary rows have to be told that width rather than assume the
// detail panes' one. They used to assume it, and the budget and carry lines sat
// a cell out of line with the stats directly above them — the wrong way in
// English and the wrong way in Vietnamese, in opposite directions.
func (c Context) LabelAt(name string, width int, format string, args ...any) string {
	return fmt.Sprintf("  %s %s\n",
		c.Style.Label.Render(Pad(name, width)), fmt.Sprintf(format, args...))
}

// Pad widens a string to a column.
//
// fmt's own %-*s counts runes, which is the right unit here — every letter this
// client draws, Vietnamese included, is one terminal cell wide, and
// TestEveryWordingIsOneCellPerLetter is what keeps that true. What fmt cannot
// do is ignore a style's escape codes, so padding happens before styling
// everywhere in this package.
func Pad(text string, width int) string {
	return fmt.Sprintf("%-*s", width, text)
}

// WrapWords breaks text on spaces, never mid-word, and never returns nothing.
//
// A word longer than the room gets its own line and overflows it rather than
// being cut: an id is a name, and half a name is worse than a line that runs on
// and gets clipped by the frame.
func WrapWords(text string, room int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	lines, current := []string{}, words[0]
	for _, word := range words[1:] {
		if lipgloss.Width(current)+1+lipgloss.Width(word) <= room {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	return append(lines, current)
}

// Ellipsis marks a line that did not fit. One rune rather than three dots, so
// that its cell count is its character count, which is the unit every column in
// these clients is measured in.
const Ellipsis = "…"

// Clip shortens a line to a number of cells and **says that it did**, keeping
// the front, which is where the id, the label and the first half of a sentence
// are.
//
// Shortened rather than wrapped, for the reason frame clips: a wrapped row
// pushes every row under it down by one, which is how the footer leaves the
// bottom of the screen.
//
// ⚠️ **Escape-aware and marking at once, which is a pair neither of the two
// tools this replaced could manage.** `lipgloss`'s `MaxWidth` steps over an
// escape sequence correctly and cuts **silently**. This function's own previous
// body appended the mark but sliced `[]rune`, and on a styled line that peels
// the terminating `\x1b[m` off the end one rune at a time — measured, not
// argued: a bold red ten-cell line cut to nine came back
// `"\x1b[1;31mabcdefgh…"`, the right width, the right letters, and **no reset**,
// so the colour bled down the rest of the screen. Every caller then passed
// unstyled text, so nothing showed it; frame's lines are styled, so making frame
// call the old body would have shipped the bleed on the first cut header.
// `ansi.Truncate` is measured in cells, steps over escape sequences rather than
// through them, and re-closes whatever the cut left open.
//
// ⚠️ **The mark is only added when the line is genuinely longer than the room.**
// A line that fills the room exactly comes back byte for byte unchanged — the
// early return below is that promise written down, and it is the whole
// off-by-one risk of marking at all: an ellipsis on a line that fitted claims a
// tail that was never there and spends a cell of real content to claim it.
// `ansi.Truncate` makes the same distinction on its own; the early return keeps
// it from mattering which of the two is doing so, and keeps an uncut line
// identical to what `MaxWidth` produced before this change.
//
// The mark itself is `Ellipsis`, the one already declared for the art row — a
// second declaration would be a second thing to keep in step, and the whole
// client is measured in cells that are also characters.
//
// It lives here rather than beside its first caller for the reason `Pad` does.
// It began as the picker's private helper for one refusal row and is now the one
// rule the whole client cuts by, so a second copy of "shorten and mark" is
// exactly what having it beside `Pad` and `LabelAt` exists to stop.
func Clip(text string, room int) string {
	if room < 1 {
		return ""
	}
	if lipgloss.Width(text) <= room {
		return text
	}
	return ansi.Truncate(text, room, Ellipsis)
}
