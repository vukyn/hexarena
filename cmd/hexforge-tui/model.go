package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/forge"
)

// screen is which of the four views is in front.
type screen int

const (
	screenMenu screen = iota
	screenBrowse
	screenNew
	screenOrigins
	screenCheck
)

// The smallest window the forms fit in. Below this the program says so rather
// than drawing a layout that overlaps itself: a corrupted screen looks like a
// bug in the tool, and the author is left unable to tell whether the character
// they are looking at is the one they typed.
const (
	minWidth  = 72
	minHeight = 24
)

// model is the whole program: a library, the screen in front, and the four
// screens' own state.
//
// It knows no rules. Every question it can answer about a character — is this
// id free, can this affinity carry this kit, what does this stat line spend of
// the budget, is this art really there — is asked of internal/forge, which is
// the same package cmd/hexforge asks. That is the point of having two
// front-ends at all: they must be incapable of disagreeing.
type model struct {
	lib   *forge.Library
	style palette

	width, height int
	screen        screen
	menu          int

	browse  browseScreen
	form    formScreen
	origins originsScreen
	check   checkScreen

	// guard holds the unsaved-changes question while it is being asked. A form
	// with edits in it is the one thing in this program that a stray Escape can
	// destroy, so leaving one asks first.
	guard *guardState
}

// guardState is a pending "are you sure" and what to do if the answer is yes.
type guardState struct {
	question string
	confirm  func(model) model
}

func newModel(lib *forge.Library) model {
	style := newPalette()
	return model{
		lib:     lib,
		style:   style,
		browse:  newBrowseScreen(lib),
		form:    newFormScreen(lib),
		origins: newOriginsScreen(lib),
		check:   newCheckScreen(lib),
	}
}

func (m model) Init() tea.Cmd { return nil }

// menuItem is one entry of the top-level view.
type menuItem struct {
	label  string
	detail string
	target screen
}

var menuItems = []menuItem{
	{"cast", "browse the authored characters and resolve them at any level", screenBrowse},
	{"new character", "author one, with the budget and the carry check live", screenNew},
	{"origins", "the works the cast is borrowed from, and add one", screenOrigins},
	{"check", "verify the art is really there and the budget is kept", screenCheck},
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
// they have to kill from another terminal. A bare q is a quit everywhere a text
// field is not focused, because where one is, q is a letter.
func (m model) key(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if message.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.guard != nil {
		return m.answerGuard(message)
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
	case screenCheck:
		return m.check.update(m, message)
	}
	return m, nil
}

// answerGuard resolves a pending confirmation. Anything that is not a yes is a
// no, which is the same default hexforge's own confirmation takes.
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
func (m model) ask(question string, confirm func(model) model) model {
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
		return "hexforge-tui: measuring the terminal…\n"
	}
	if m.tooSmall() {
		return m.viewTooSmall()
	}
	var body, footer string
	switch m.screen {
	case screenMenu:
		body, footer = m.viewMenu(), "↑/↓ move · enter open · q quit"
	case screenBrowse:
		body, footer = m.browse.view(m)
	case screenNew:
		body, footer = m.form.view(m)
	case screenOrigins:
		body, footer = m.origins.view(m)
	case screenCheck:
		body, footer = m.check.view(m)
	}
	if m.guard != nil {
		footer = m.guard.question + " [y/N] · ctrl+c quit"
	}
	return m.frame(body, footer)
}

// viewTooSmall is what a window that cannot hold a form gets instead of a
// mangled one.
//
// It names both sizes, because "too small" without a target is a message
// nobody can act on, and it points at the other front-end, which needs no room
// at all and can do everything this one can.
func (m model) viewTooSmall() string {
	// Every line is kept short on purpose. This is the one screen that is drawn
	// into a window known to be small, so it has to read in a window smaller
	// still — and it is clipped like any other, rather than wrapping.
	lines := strings.Split(fmt.Sprintf(`terminal too small

needs at least %dx%d
this window is %dx%d

Make it bigger, or use
hexforge instead: same
cast, same checks, and
it fits any terminal.

q or ctrl+c to quit`, minWidth, minHeight, m.width, m.height), "\n")
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
	header := m.style.title.Render("hexforge") + m.style.dim.Render("  "+m.lib.Dir())
	lines := []string{header, ""}
	lines = append(lines, strings.Split(body, "\n")...)

	// Two lines for the header, one blank before the footer, one for the
	// footer. Anything past that is cut rather than allowed to push the footer
	// off the bottom: the footer is where the keys are, and a screen whose keys
	// have scrolled away is a screen nobody can leave.
	room := m.height - 2
	if len(lines) > room {
		lines = lines[:room]
		lines[room-1] = m.style.dim.Render("… cut off; a taller window shows the rest")
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

func (m model) viewMenu() string {
	var out strings.Builder
	out.WriteString(m.style.heading.Render("what would you like to do?") + "\n\n")
	for i, item := range menuItems {
		marker := "  "
		label := item.label
		if i == m.menu {
			// The marker is the selection. The style only agrees with it.
			marker = "> "
			label = m.style.selected.Render(label)
		}
		out.WriteString(fmt.Sprintf("%s%-16s %s\n", marker, label, m.style.dim.Render(item.detail)))
	}
	out.WriteString("\n" + m.style.dim.Render(
		"Everything written here goes through the same checks as hexforge, and the\n"+
			"game boots from the embedded copy — rebuild before an edit reaches a battle."))
	return out.String()
}

// label draws a "name  value" row the way every detail pane in this program
// does, so the panes line up with each other.
func (m model) label(name, format string, args ...any) string {
	return fmt.Sprintf("  %s %s\n",
		m.style.label.Render(fmt.Sprintf("%-11s", name)), fmt.Sprintf(format, args...))
}
