package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// screen is which of the four views is in front.
type screen int

const (
	screenMenu screen = iota
	screenBrowse
	screenNew
	screenOrigins
	screenSkills
	screenCheck
)

// The smallest window the screens fit in.
//
// Below this the program says so rather than drawing a layout that overlaps
// itself: a corrupted screen looks like a bug in the tool, and the author is
// left unable to tell whether the character they are looking at is the one they
// typed.
//
// The width was 72 while this client was English only. Vietnamese runs a fifth
// to a third longer for the same sentence, and the busiest footers — six chords
// on the form, the level and filter keys on the browser — landed just past it
// once the language chord was added, so the floor moved to the other number a
// terminal has always had. TestEveryWordingFitsTheMinimumWidth measures the
// catalog against this constant in both languages, so it cannot rot quietly.
const (
	minWidth  = 80
	minHeight = 24
)

// detailLabels is every row name a detail pane draws, which is what its label
// column is measured from. The level row is not in it because it is named after
// a number; detailLabelWidth measures that one separately.
var detailLabels = []i18n.Key{
	i18n.LabelFrom, i18n.LabelPlaystyle, i18n.LabelElement, i18n.LabelKit,
	i18n.LabelArt, i18n.LabelStages, i18n.LabelBiography, i18n.LabelEffectiveHP,
	i18n.LabelNote,
}

// detailLabelWidth is the column every detail pane's row name sits in,
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
func detailLabelWidth(m model) int {
	widest := 0
	for _, key := range detailLabels {
		if width := lipgloss.Width(m.text(key)); width > widest {
			widest = width
		}
	}
	// The level row is named after a number, and the widest one is the cap.
	if width := lipgloss.Width(m.text(i18n.LabelAtLevel, progression.LevelCap)); width > widest {
		widest = width
	}
	return widest + 1
}

// model is the whole program: a library, the language, the screen in front, and
// the four screens' own state.
//
// It knows no rules. Every question it can answer about a character — is this
// id free, can this affinity carry this kit, what does this stat line spend of
// the budget, is this art really there — is asked of internal/forge, which is
// the same package cmd/hexforge asks. That is the point of having two
// front-ends at all: they must be incapable of disagreeing.
//
// It knows no wording either. Every sentence comes from internal/i18n, keyed by
// a constant, so a screen cannot hold a line that exists in one language only.
type model struct {
	lib   *forge.Library
	lang  i18n.Lang
	style palette

	width, height int
	screen        screen
	menu          int

	browse  browseScreen
	form    formScreen
	origins originsScreen
	skills  skillsScreen
	check   checkScreen

	// picker holds the multi-select while it is open, over whichever screen
	// raised it. It lives here rather than on a screen because two screens raise
	// one — the kit on the new-character form, the three allowlists on the new
	// skill form — and a picker owned by a screen would be two pickers to keep
	// in step.
	picker *pickState

	// guard holds the unsaved-changes question while it is being asked. A form
	// with edits in it is the one thing in this program that a stray Escape can
	// destroy, so leaving one asks first.
	guard *guardState
}

// guardState is a pending "are you sure" and what to do if the answer is yes.
//
// The question is a key rather than a sentence so that switching language with
// one pending redraws it, instead of leaving the last language's words on the
// screen until the question is answered.
type guardState struct {
	question i18n.Key
	confirm  func(model) model
}

func newModel(lib *forge.Library, lang i18n.Lang) model {
	style := newPalette()
	return model{
		lib:     lib,
		lang:    lang,
		style:   style,
		browse:  newBrowseScreen(lib),
		form:    newFormScreen(lib),
		origins: newOriginsScreen(lib),
		skills:  newSkillsScreen(lib),
		check:   newCheckScreen(lib),
	}
}

func (m model) Init() tea.Cmd { return nil }

// text is one line in the language in front. Every screen goes through it.
func (m model) text(key i18n.Key, args ...any) string { return m.lang.Say(key, args...) }

// menuItem is one entry of the top-level view.
type menuItem struct {
	label  i18n.Key
	detail i18n.Key
	target screen
}

var menuItems = []menuItem{
	{i18n.MenuCast, i18n.MenuCastDetail, screenBrowse},
	{i18n.MenuNewCharacter, i18n.MenuNewCharacterDetail, screenNew},
	{i18n.MenuOrigins, i18n.MenuOriginsDetail, screenOrigins},
	{i18n.MenuSkills, i18n.MenuSkillsDetail, screenSkills},
	{i18n.MenuCheck, i18n.MenuCheckDetail, screenCheck},
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil
	case tea.KeyMsg:
		return m.key(typed)
	}
	return m, nil
}

// key routes one keystroke.
//
// ctrl+c is handled before anything else and from every screen, including with
// a question pending: a program that can trap somebody in a modal is a program
// they have to kill from another terminal. ctrl+l is next and works from
// everywhere for a smaller reason of the same shape — the two languages are
// only worth comparing side by side if the comparison costs nothing, and a
// toggle that needed the form closed first would be a toggle nobody uses
// mid-sentence.
//
// It is a chord rather than a letter because on the form a letter is text: a
// bare l would have to be typed into a field, not swallowed by the program.
// Only m.lang changes, so every field keeps what has been typed into it.
//
// A bare q is a quit everywhere a text field is not focused, because where one
// is, q is a letter.
func (m model) key(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+l":
		m.lang = m.lang.Other()
		return m, nil
	}
	if m.guard != nil {
		return m.answerGuard(message)
	}
	if m.picker != nil {
		return m.picker.update(m, message)
	}
	if m.tooSmall() {
		if message.String() == "q" {
			return m, tea.Quit
		}
		return m, nil
	}
	switch m.screen {
	case screenMenu:
		return m.updateMenu(message)
	case screenBrowse:
		return m.browse.update(m, message)
	case screenNew:
		return m.form.update(m, message)
	case screenOrigins:
		return m.origins.update(m, message)
	case screenSkills:
		return m.skills.update(m, message)
	case screenCheck:
		return m.check.update(m, message)
	}
	return m, nil
}

// answerGuard resolves a pending confirmation. Anything that is not a yes is a
// no, which is the same default hexforge's own confirmation takes.
//
// The y is the same letter in both languages on purpose: it is what the [y/N]
// on screen offers, and what every other confirmation in this repository takes.
func (m model) answerGuard(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	confirm := m.guard.confirm
	switch strings.ToLower(message.String()) {
	case "y":
		m.guard = nil
		return confirm(m), nil
	default:
		m.guard = nil
		return m, nil
	}
}

// ask raises a confirmation. The callback runs only if the answer is yes.
func (m model) ask(question i18n.Key, confirm func(model) model) model {
	m.guard = &guardState{question: question, confirm: confirm}
	return m
}

func (m model) updateMenu(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.menu > 0 {
			m.menu--
		}
	case "down", "j":
		if m.menu < len(menuItems)-1 {
			m.menu++
		}
	case "enter", " ":
		return m.enter(menuItems[m.menu].target), nil
	}
	return m, nil
}

// enter switches screens, giving the one being entered the chance to refresh
// against a library that may have been written to since it was last drawn.
func (m model) enter(target screen) model {
	m.screen = target
	switch target {
	case screenBrowse:
		m.browse = m.browse.refresh(m.lib)
	case screenCheck:
		m.check = m.check.refresh(m.lib)
	case screenOrigins:
		m.origins = m.origins.refresh(m.lib)
	case screenSkills:
		m.skills = m.skills.refresh(m.lib)
	}
	return m
}

func (m model) tooSmall() bool {
	return m.width > 0 && (m.width < minWidth || m.height < minHeight)
}

func (m model) View() string {
	if m.width == 0 {
		// bubbletea sends the first window size a moment after start. Drawing a
		// guessed layout here and redrawing it immediately is a visible flash.
		return programName + ": " + m.text(i18n.MeasuringTerminal) + "\n"
	}
	if m.tooSmall() {
		return m.viewTooSmall()
	}
	var body, footer string
	switch m.screen {
	case screenMenu:
		body, footer = m.viewMenu(), m.text(i18n.MenuFooter)
	case screenBrowse:
		body, footer = m.browse.view(m)
	case screenNew:
		body, footer = m.form.view(m)
	case screenOrigins:
		body, footer = m.origins.view(m)
	case screenSkills:
		body, footer = m.skills.view(m)
	case screenCheck:
		body, footer = m.check.view(m)
	}
	// The picker is drawn over whichever screen raised it, for the same reason
	// it is a sub-screen at all: a list of nineteen does not fit beside a form.
	if m.picker != nil {
		body, footer = m.picker.view(m)
	}
	if m.guard != nil {
		footer = m.text(i18n.ConfirmFooter, m.text(m.guard.question))
	}
	return m.frame(body, footer)
}

// viewTooSmall is what a window that cannot hold a screen gets instead of a
// mangled one.
//
// It names both sizes, because "too small" without a target is a message nobody
// can act on, and it points at the other front-end, which needs no room at all
// and can do everything this one can. It is drawn in the language in front,
// which is why it is a catalog entry and not a fallback in English: the person
// who cannot read the screen is exactly the person who needs this line.
func (m model) viewTooSmall() string {
	lines := strings.Split(
		m.text(i18n.TerminalTooSmall, minWidth, minHeight, m.width, m.height), "\n")
	clip := lipgloss.NewStyle().MaxWidth(m.width)
	for i, line := range lines {
		lines[i] = clip.Render(line)
	}
	return strings.Join(lines, "\n")
}

// frame puts the header and the footer around a screen's body and pads the
// whole thing to the window's height, so a shorter screen does not leave the
// previous one's tail on display.
func (m model) frame(body, footer string) string {
	header := m.style.title.Render(programName) + m.style.dim.Render("  "+m.lib.Dir())
	lines := []string{header, ""}
	lines = append(lines, strings.Split(body, "\n")...)

	// Two lines for the header, one blank before the footer, one for the
	// footer. Anything past that is cut rather than allowed to push the footer
	// off the bottom: the footer is where the keys are, and a screen whose keys
	// have scrolled away is a screen nobody can leave.
	room := m.height - 2
	if len(lines) > room {
		lines = lines[:room]
		lines[room-1] = m.style.dim.Render(m.text(i18n.Truncated))
	}
	for len(lines) < room {
		lines = append(lines, "")
	}
	lines = append(lines, m.style.footer.Render(footer))

	// Clip every line to the window rather than letting a long one wrap.
	// A biography or a filesystem path is free text of any length, and a
	// wrapped line pushes everything below it down by one — which moves the
	// footer off the bottom and makes the screen disagree with the line count
	// above. MaxWidth cuts without cutting an escape sequence in half.
	clip := lipgloss.NewStyle().MaxWidth(m.width)
	for i, line := range lines {
		lines[i] = clip.Render(line)
	}
	return strings.Join(lines, "\n")
}

// menuLabelWidth is the column the menu's own labels sit in, measured for the
// same reason the detail panes' column is: "danh sách nhân vật" is 18 cells
// against "cast" at 4, so one number for both languages either cuts one or
// wastes a fifth of the row in the other.
func menuLabelWidth(m model) int {
	widest := 0
	for _, item := range menuItems {
		if width := lipgloss.Width(m.text(item.label)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

func (m model) viewMenu() string {
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.MenuHeading)) + "\n\n")
	width := menuLabelWidth(m)
	for i, item := range menuItems {
		marker := "  "
		// Padded before it is styled, not after: a style is escape codes, and
		// fmt would count those toward the column.
		label := pad(m.text(item.label), width)
		if i == m.menu {
			// The marker is the selection. The style only agrees with it.
			marker = "> "
			label = m.style.selected.Render(label)
		}
		out.WriteString(marker + label + " " + m.style.dim.Render(m.text(item.detail)) + "\n")
	}
	out.WriteString("\n" + m.style.dim.Render(m.text(i18n.MenuNote)))
	return out.String()
}

// label draws a "name  value" row the way every detail pane in this program
// does, so the panes line up with each other.
func (m model) label(name, format string, args ...any) string {
	return m.labelAt(name, detailLabelWidth(m), format, args...)
}

// continued draws a row that carries on from the one above it: no name of its
// own, and its value under the same column as that row's.
//
// The kit's Vietnamese names are what this exists for. Five skills glossed
// inline would be five brackets on one row, which does not fit in 80 columns,
// so they go underneath in the same order instead — and they only line up with
// the ids above them if they are placed by the same measurement.
func (m model) continued(format string, args ...any) string {
	return m.labelAt("", detailLabelWidth(m), format, args...)
}

// wrapped is label for a value with no bound on its length: it fills the row and
// carries on underneath, aligned with where the value started.
//
// Clipping was wrong for these in two ways at once. It cut at minWidth, which is
// the *floor* a window has to clear rather than a ceiling on what one may use, so
// a hundred-cell terminal was being told it had seventy-nine. And a kit of nine
// ids or a paragraph of biography is longer than any terminal, so widening alone
// would not have saved it — the tail has to go somewhere, and a row below is
// where a reader looks for it.
//
// The rows this draws are variable in number, so it belongs only on a pane that
// can afford that. The form counts its rows to scroll them and must not use it.
func (m model) wrapped(name string, width int, value string) string {
	room := m.usableWidth() - 2 - width - 1
	if room < 8 {
		// Narrower than this and wrapping makes a column of syllables; the
		// clip is the lesser evil.
		return m.labelAt(name, width, "%s", clip(value, max(room, 1)))
	}
	lines := wrapWords(value, room)
	var out strings.Builder
	out.WriteString(m.labelAt(name, width, "%s", lines[0]))
	for _, line := range lines[1:] {
		out.WriteString(m.labelAt("", width, "%s", line))
	}
	return out.String()
}

// usableWidth is what a row may spend: the window when there is one, and the
// floor before the first size message arrives.
func (m model) usableWidth() int {
	if m.width < minWidth {
		return minWidth
	}
	return m.width
}

// wrapWords breaks text on spaces, never mid-word, and never returns nothing.
//
// A word longer than the room gets its own line and overflows it rather than
// being cut: an id is a name, and half a name is worse than a line that runs on
// and gets clipped by the frame.
func wrapWords(text string, room int) []string {
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

// labelAt is label in a caller-chosen column.
//
// The new-character form sizes its own column from the widest field name it is
// drawing, which differs per language ("archetype" is 9 cells, "mẫu vai trò" is
// 11), so its summary rows have to be told that width rather than assume the
// detail panes' one. They used to assume it, and the budget and carry lines sat
// a cell out of line with the stats directly above them — the wrong way in
// English and the wrong way in Vietnamese, in opposite directions.
func (m model) labelAt(name string, width int, format string, args ...any) string {
	return fmt.Sprintf("  %s %s\n",
		m.style.label.Render(pad(name, width)), fmt.Sprintf(format, args...))
}

// pad widens a string to a column.
//
// fmt's own %-*s counts runes, which is the right unit here — every letter this
// client draws, Vietnamese included, is one terminal cell wide, and
// TestEveryWordingIsOneCellPerLetter is what keeps that true. What fmt cannot
// do is ignore a style's escape codes, so padding happens before styling
// everywhere in this package.
func pad(text string, width int) string {
	return fmt.Sprintf("%-*s", width, text)
}
