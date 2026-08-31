package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// screen is which of the four views is in front.
type screen int

const (
	screenMenu screen = iota
	screenBrowse
	screenNew
	screenOrigins
	screenSkills
	screenStatuses
	screenPassives
	screenElements
	screenSpecies
	screenCheck
	// screenPreview is raised from the browser rather than the menu, because it
	// draws one character's art and the browser is where a character is chosen.
	// Appended rather than slotted in beside screenBrowse: nothing serialises
	// these, but the menu is built from the order they are declared in.
	screenPreview
	// screenSpar is raised from the check screen, which is where a character is
	// under a cursor and has just been said to be sound. It owns its own level,
	// unlike the preview, because the level it asks about is its own question
	// rather than one somebody else was already walking.
	screenSpar
	// screenBlurb is raised from the skill listing for the same reason
	// screenPreview is raised from the browser: it describes the skill under a
	// cursor, and the listing is where a skill is chosen.
	screenBlurb
	// screenChart is raised from the elements listing, for the reason screenBlurb
	// is raised from the skills one: it is the same subject read the other way
	// round — the shape of the chart rather than one element's place in it — and
	// at the floor the window will not hold both.
	screenChart
	// screenBuilds is the seventh listing and is reached from the menu, like the
	// other references and unlike the screens above it: a build belongs to a
	// character but the question it answers — which directions are written down —
	// is one a reader has before they have chosen a character to look at.
	//
	// Appended for the reason screenPreview was: nothing serialises these, but the
	// menu is built from the order they are declared in.
	screenBuilds
	// screenSquads is the first screen that writes the author's own file rather
	// than the game's: every other file here is data somebody wrote for the
	// game, and squads.json is a side built to be fought with. It ships like the
	// rest of them — a squad saved here reaches a battle at the next build — so
	// what is different is who it is for, not whether it travels. Appended for the reason
	// screenPreview was — nothing serialises these, but the menu is built from
	// the order they are declared in.
	screenSquads
	// screenFight is on the menu and is also raised from the squad catalogue
	// with f. It can be both because it carries both of its sides itself rather
	// than reading one off the catalogue's cursor: the catalogue's f only seeds
	// the home index, so arriving that way still fights the squad that was
	// pointed at, and arriving from the menu has a subject without one.
	screenFight
	// screenPlay is raised from the fight, for the reason the fight is raised
	// from the catalogue: that is where a pairing is already chosen, and a
	// battle wants two squads before it wants anything else.
	screenPlay
)

// The smallest window the screens fit in — 120x24, declared in
// internal/screen with the measurement that asked for it, because
// screen.Context.UsableWidth is what spends it and a second client draws the
// same screens against the same floor.
//
// Named here as well because a hundred and fifty rows in this package measure
// themselves against it, and an alias is one declaration rather than two.
const (
	minWidth  = draw.MinWidth
	minHeight = draw.MinHeight
)

// detailLabelWidth is the column every detail pane's row name sits in, measured
// in internal/screen from the widest name being drawn in the language in front.
//
// Kept as a function of the model so the nineteen rows that ask for it read
// unchanged.
func detailLabelWidth(m model) int { return m.ctx().DetailLabelWidth() }

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
	style draw.Palette

	width, height int
	screen        screen
	menu          int

	// raisedFrom is where a Back goes: the screen that raised whatever is in
	// front, remembered by the client because a screen may not name one.
	//
	// One slot, and that is all the six screens converted to draw.Action need —
	// they raise one deep and no further (the elements listing raises the chart,
	// the traits listing raises the statuses reference, and neither of those two
	// raises anything at all), so there is no stack to keep and nothing to pop.
	//
	// ⚠️ Two facts make this exactly what it replaces, a `from screen` field on
	// the statuses screen. screenMenu is the **zero value** of screen, so an
	// unwritten slot already means the menu, which is where esc went from all
	// four of the screens that never raise; and it is cleared **as it is used**,
	// in navigate, because the old field was cleared in the screen's own esc
	// rather than by enter — a reader who came through the menu after coming
	// through a trait must not inherit the trait.
	raisedFrom screen

	browse   browseScreen
	form     formScreen
	origins  originsScreen
	skills   skillsScreen
	statuses statusesScreen
	passives passivesScreen
	elements elementsScreen
	species  speciesScreen
	builds   buildsScreen
	squad    squadScreen
	fight    fightScreen
	play     playScreen
	// chart holds nothing: it is drawn from the library every time and has no
	// cursor. It is a field anyway so that the screen is dispatched to like every
	// other one, rather than being a special case in three switches.
	chart   chartScreen
	check   checkScreen
	preview previewScreen
	spar    sparScreen
	blurb   blurbScreen

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
		lib:      lib,
		lang:     lang,
		style:    style,
		browse:   newBrowseScreen(lib),
		form:     newFormScreen(lib),
		origins:  newOriginsScreen(lib),
		skills:   newSkillsScreen(lib),
		statuses: newStatusesScreen(lib),
		passives: newPassivesScreen(lib),
		species:  newSpeciesScreen(lib),
		builds:   newBuildsScreen(lib),
		squad:    newSquadScreen(lib),
		fight:    newFightScreen(),
		play:     newPlayScreen(),
		check:    newCheckScreen(lib),
		preview:  newPreviewScreen(),
		spar:     newSparScreen(),
	}
}

func (m model) Init() tea.Cmd { return nil }

// ctx is what a screen draws with, in the shape internal/screen understands: the
// books, the language, the palette and the window, and nothing this model owns.
//
// Every helper below forwards through it, so the drawing rules have one
// declaration and the ~200 rows that call m.text / m.label / m.wrapped read
// unchanged. Built per call rather than kept as a field: the model is a value
// copied on every keystroke, so a stored copy would be a second place the window
// size lives.
func (m model) ctx() draw.Context {
	return draw.Context{
		Lib:   m.lib,
		Lang:  m.lang,
		Style: m.style,
		Width: m.width, Height: m.height,
	}
}

// text is one line in the language in front. Every screen goes through it.
func (m model) text(key i18n.Key, args ...any) string { return m.ctx().Text(key, args...) }

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
	{i18n.MenuStatuses, i18n.MenuStatusesDetail, screenStatuses},
	{i18n.MenuPassives, i18n.MenuPassivesDetail, screenPassives},
	{i18n.MenuElements, i18n.MenuElementsDetail, screenElements},
	{i18n.MenuSpecies, i18n.MenuSpeciesDetail, screenSpecies},
	{i18n.MenuBuilds, i18n.MenuBuildsDetail, screenBuilds},
	{i18n.MenuSquads, i18n.MenuSquadsDetail, screenSquads},
	{i18n.MenuFight, i18n.MenuFightDetail, screenFight},
	{i18n.MenuCheck, i18n.MenuCheckDetail, screenCheck},
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil
	case tea.KeyPressMsg:
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
func (m model) key(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	// ⚠️ Two mechanisms sit side by side in this switch, and that is deliberate
	// rather than half-finished. The six screens below are the ones being moved
	// into internal/screen: their updates return a draw.Action — what they want —
	// and this client decides what it means, which is what lets them stop naming
	// entries of an enum only this binary has. Every other branch still writes
	// m.screen itself and will be converted as it moves.
	case screenStatuses:
		statuses, action := m.statuses.update(m.ctx(), message)
		m.statuses = statuses
		return m.navigate(screenStatuses, action)
	case screenPassives:
		passives, action := m.passives.update(m.ctx(), message)
		m.passives = passives
		return m.navigate(screenPassives, action)
	case screenElements:
		elements, action := m.elements.update(m.ctx(), message)
		m.elements = elements
		return m.navigate(screenElements, action)
	case screenSpecies:
		species, action := m.species.update(m.ctx(), message)
		m.species = species
		return m.navigate(screenSpecies, action)
	case screenBuilds:
		builds, action := m.builds.update(m.ctx(), message)
		m.builds = builds
		return m.navigate(screenBuilds, action)
	case screenSquads:
		return m.squad.update(m, message)
	case screenFight:
		return m.fight.update(m, message)
	case screenPlay:
		return m.play.update(m, message)
	case screenCheck:
		return m.check.update(m, message)
	case screenPreview:
		return m.preview.update(m, message)
	case screenSpar:
		return m.spar.update(m, message)
	case screenBlurb:
		return m.blurb.update(m, message)
	case screenChart:
		chart, action := m.chart.update(m.ctx(), message)
		m.chart = chart
		return m.navigate(screenChart, action)
	}
	return m, nil
}

// raiseTargets is what a draw.Target means to this client.
//
// A map keyed by target and read by key, never ranged over into anything that
// reaches a screen — the same discipline internal/core holds about map order, one
// layer up.
//
// ⚠️ It has to be **total**: a target with no entry here makes a raise silently
// do nothing, which is the shape of defect this repository has recorded five
// times as a screen slipping out of everyScreen.
// TestEveryRaiseTargetNamesAScreenInThisClient walks screen.TargetCount rather
// than this map, so a target added over there fails here instead of going
// quiet.
var raiseTargets = map[draw.Target]screen{
	draw.Chart:    screenChart,
	draw.Statuses: screenStatuses,
}

// navigate applies what a screen asked for.
//
// from is the screen that asked, which is the whole of what a Raise has to
// record: Back is the answer to "how did I get here", and only the client can
// know it.
func (m model) navigate(from screen, action draw.Action) (tea.Model, tea.Cmd) {
	switch action.Kind {
	case draw.Quit:
		// This client ends there. A game client with a battle half played is
		// exactly why the screen handed back an action instead of the command.
		return m, tea.Quit
	case draw.Back:
		m.screen = m.raisedFrom
		// Forgotten as it is used: the next visit through the menu must not
		// inherit this one's way back.
		m.raisedFrom = screenMenu
		return m, nil
	case draw.Raise:
		return m.raise(from, action)
	}
	// draw.Stay, which is every keystroke a screen handled without leaving.
	return m, nil
}

// raise opens the screen a target names, landing it on the id the raiser asked
// for.
//
// ⚠️ It declines rather than half-arriving. A focus the raised screen cannot find
// leaves the reader where they are — the trait naming a status the book has lost
// is a trait already printing a bare id, and a cursor moved to whatever sorted
// next would answer a question nobody asked. A target with no entry in
// raiseTargets declines for the opposite reason: it is a bug rather than a state,
// and the test above is what says so out loud.
//
// Direct rather than through enter, which is what the two raises it replaces
// did: neither screen refreshes on the way in, and routing them through enter
// now would be a behaviour change wearing a tidy-up's clothes.
func (m model) raise(from screen, action draw.Action) (tea.Model, tea.Cmd) {
	target, known := raiseTargets[action.Target]
	if !known {
		return m, nil
	}
	if action.Focus != "" {
		focused, found := m.focus(target, action.Focus)
		if !found {
			return m, nil
		}
		m = focused
	}
	m.raisedFrom = from
	m.screen = target
	return m, nil
}

// focus lands a screen's cursor on a named id and reports whether it found one.
//
// The statuses reference is the only screen that answers today, because it is
// the only one anything raises with an id. A focus aimed anywhere else is not
// silently dropped — it reports not found, and raise declines the whole trip,
// so the mismatch shows as a reader who did not move rather than as a screen
// that opened somewhere arbitrary.
func (m model) focus(target screen, id string) (model, bool) {
	if target != screenStatuses {
		return m, false
	}
	statuses, found := m.statuses.focus(id)
	if !found {
		return m, false
	}
	m.statuses = statuses
	return m, true
}

// answerGuard resolves a pending confirmation. Anything that is not a yes is a
// no, which is the same default hexforge's own confirmation takes.
//
// The y is the same letter in both languages on purpose: it is what the [y/N]
// on screen offers, and what every other confirmation in this repository takes.
func (m model) answerGuard(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) updateMenu(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "enter", "space":
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
	case screenStatuses:
		m.statuses = m.statuses.refresh(m.lib)
	case screenPassives:
		m.passives = m.passives.refresh(m.lib)
	case screenSpecies:
		m.species = m.species.refresh(m.lib)
	case screenBuilds:
		m.builds = m.builds.refresh(m.lib)
	case screenSquads:
		m.squad = m.squad.refresh(m.lib)
	case screenFight:
		// The fight draws both of its sides out of the catalogue's list, so it
		// reads that list on the way in the way every other listing screen here
		// reads its own. It is the same reading rather than a second copy — a
		// second copy is a second thing to keep in step — which is why this
		// refreshes another screen's state and not one of its own.
		//
		// ⚠️ Nothing today can make that list stale between the model being
		// built and the fight being entered: newSquadScreen refreshes at
		// construction, and the only two writes (save and delete) are the
		// catalogue's own keys, which refresh after them. So no test
		// discriminates this line, and a mutation deleting it passes the whole
		// suite. It is here so the fight owns its subject rather than depending
		// on somebody having visited the catalogue first, and it must not be
		// read as a guard against a state that has been seen.
		m.squad = m.squad.refresh(m.lib)
		m.fight = m.fight.refresh()
	case screenPlay:
		// A battle is built on the way in rather than lazily in the view: the
		// screen holds a pointer, and building one while drawing would be a
		// redraw with a side effect.
		m.play = m.play.begin(m)
	case screenSpar:
		m.spar = m.spar.refresh()
	}
	return m
}

func (m model) tooSmall() bool {
	return m.width > 0 && (m.width < minWidth || m.height < minHeight)
}

// View draws the screen, and is the one place the alternate screen is asked
// for: bubbletea v2 puts that on the view rather than on a program option, so
// it is a property of what is being drawn rather than of how the program was
// started.
func (m model) View() tea.View {
	view := tea.NewView(m.screenContent())
	view.AltScreen = true
	return view
}

func (m model) screenContent() string {
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
	case screenStatuses:
		body, footer = m.statuses.view(m)
	case screenPassives:
		body, footer = m.passives.view(m)
	case screenElements:
		body, footer = m.elements.view(m)
	case screenSpecies:
		body, footer = m.species.view(m)
	case screenBuilds:
		body, footer = m.builds.view(m)
	case screenSquads:
		body, footer = m.squad.view(m)
	case screenFight:
		body, footer = m.fight.view(m)
	case screenPlay:
		body, footer = m.play.view(m)
	case screenCheck:
		body, footer = m.check.view(m)
	case screenPreview:
		body, footer = m.preview.view(m)
	case screenSpar:
		body, footer = m.spar.view(m)
	case screenBlurb:
		body, footer = m.blurb.view(m)
	case screenChart:
		body, footer = m.chart.view(m)
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
	// Cut through clip for the reason frame does, and this is the one screen
	// where a cut is close to certain: it is only ever drawn in a window already
	// too narrow for anything else, so the sentence saying so is the sentence
	// most likely to lose its tail.
	for i, line := range lines {
		lines[i] = clip(line, m.width)
	}
	return strings.Join(lines, "\n")
}

// frame puts the header and the footer around a screen's body and pads the
// whole thing to the window's height, so a shorter screen does not leave the
// previous one's tail on display.
func (m model) frame(body, footer string) string {
	header := m.style.Title.Render(programName) + m.style.Dim.Render("  "+m.lib.Dir())
	lines := []string{header, ""}
	lines = append(lines, strings.Split(body, "\n")...)

	// Two lines for the header, one blank before the footer, one for the
	// footer. Anything past that is cut rather than allowed to push the footer
	// off the bottom: the footer is where the keys are, and a screen whose keys
	// have scrolled away is a screen nobody can leave.
	room := m.height - 2
	if len(lines) > room {
		lines = lines[:room]
		lines[room-1] = m.style.Dim.Render(m.text(i18n.Truncated))
	}
	for len(lines) < room {
		lines = append(lines, "")
	}
	lines = append(lines, m.style.Footer.Render(footer))

	// Clip every line to the window rather than letting a long one wrap.
	// A biography or a filesystem path is free text of any length, and a
	// wrapped line pushes everything below it down by one — which moves the
	// footer off the bottom and makes the screen disagree with the line count
	// above.
	//
	// ⚠️ **And the cut says so**, which it did not. This used to be
	// `lipgloss.MaxWidth`, which cuts safely and cuts **silently** — a sentence
	// arrived a few cells short with nothing on the screen saying a tail had been
	// taken off, which is worse than one that says so, because a reader cannot
	// tell a truncated explanation from a complete one. It is the horizontal twin
	// of the `Truncated` marker eleven lines above: the vertical cut has said so
	// since it was written and the horizontal one had not.
	//
	// clip is escape-aware and marking at once, which neither tool this replaced
	// managed alone — see its own comment. The mark is a cell of the window
	// rather than a cell past it, so a marked line is exactly as wide as the
	// unmarked cut would have been and the row arithmetic above is untouched.
	//
	// **Every line, and that is a measurement rather than a shrug.** At the 120
	// floor, over every screen and state `everyScreen` registers in both
	// languages, what still reaches this cut is the header naming the library
	// directory, the check screen's summary line (which also names it), and the
	// form's archetype row (a preset id and its whole kit) — 122, 128 and 131
	// cells. At 160 and at 200, nothing. All of it is **text**: a path, a
	// sentence, a list of ids. **No drawing can reach here** — `tui.Board` is
	// nineteen cells wide against a floor of 120, and the preview's art is
	// `usableWidth() - 2` by construction — which is why marking every line
	// cannot paste an ellipsis onto the end of a picture.
	// `TestNoDrawingIsEverWideEnoughToBeMarked` is what says so if a wider
	// drawing is ever added, since an ellipsis on a sentence and an ellipsis on a
	// hex board are not the same kind of help.
	for i, line := range lines {
		lines[i] = clip(line, m.width)
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
	out.WriteString(m.style.Heading.Render(m.text(i18n.MenuHeading)) + "\n\n")
	width := menuLabelWidth(m)
	for i, item := range menuItems {
		marker := "  "
		// Padded before it is styled, not after: a style is escape codes, and
		// fmt would count those toward the column.
		label := pad(m.text(item.label), width)
		if i == m.menu {
			// The marker is the selection. The style only agrees with it.
			marker = "> "
			label = m.style.Selected.Render(label)
		}
		out.WriteString(marker + label + " " + m.style.Dim.Render(m.text(item.detail)) + "\n")
	}
	out.WriteString("\n" + m.style.Dim.Render(m.text(i18n.MenuNote)))
	return out.String()
}

// The drawing helpers are one-line forwarders into internal/screen, which is
// where their bodies live: a second full-screen client draws the same reference
// screens, and a row is not something either of them gets to decide for itself.
// The implementation moved; the call sites did not.

// label draws a "name  value" row the way every detail pane in this program
// does, so the panes line up with each other.
func (m model) label(name, format string, args ...any) string {
	return m.ctx().Label(name, format, args...)
}

// continued draws a row that carries on from the one above it: no name of its
// own, and its value under the same column as that row's.
func (m model) continued(format string, args ...any) string {
	return m.ctx().Continued(format, args...)
}

// wrapped is label for a value with no bound on its length: it fills the row and
// carries on underneath, aligned with where the value started.
//
// The rows this draws are variable in number, so it belongs only on a pane that
// can afford that. The form counts its rows to scroll them and must not use it.
func (m model) wrapped(name string, width int, value string) string {
	return m.ctx().Wrapped(name, width, value)
}

// wrappedIn is wrapped with a style, applied one line at a time.
func (m model) wrappedIn(name string, width int, style lipgloss.Style, value string) string {
	return m.ctx().WrappedIn(name, width, style, value)
}

// usableWidth is what a row may spend: the window when there is one, and the
// floor before the first size message arrives.
func (m model) usableWidth() int { return m.ctx().UsableWidth() }

// fieldValueRoom is what a form row has left for the one part of it that has no
// length of its own — the chances beside the inflicts field, the ids in an
// allowlist, the kit and the species on the character form — once the marker,
// the label column, the fixed part of the value and the two-space gap before
// whatever follows have been paid for.
//
// One declaration for **both forms**, because the skill form and the character
// form draw the identical row — `marker + pad(label, width) + " " + value` — and
// each had written the arithmetic out for itself. All four copies were one cell
// over, in the same direction, and the two that were found first were fixed
// while the two in form.go were not; a second copy is a second thing to fix
// twice. The window's last column is left empty for the reason frame leaves it:
// a line filling a terminal's final cell wraps on some of them, and one wrapped
// line pushes the footer off the bottom.
//
// ⚠️ It lives here rather than beside either caller for that reason too. In
// skills.go it read as the skill form's arithmetic, which is what let form.go
// write its own; here it sits with pad, labelAt and usableWidth, which are the
// rest of what a row is made of.
//
// The width is handed in rather than read off minWidth, and that is what keeps
// the single declaration honest: a row that spends this on **data** — the
// chances, an allowlist of ids, a chosen kit — passes m.usableWidth(), which is
// the window when there is one, while a row spending it on wording would pass
// minWidth. All four callers today are data and all four pass the window, but
// picking a side inside the function would make the next caller either wrong or
// a second copy of the arithmetic, which is exactly what this exists to have
// stopped.
func fieldValueRoom(width, labelWidth, spent int) int {
	const marker, gap = 2, 2
	return width - 1 - marker - labelWidth - 1 - spent - gap
}

// wrapWords breaks text on spaces, never mid-word, and never returns nothing.
func wrapWords(text string, room int) []string { return draw.WrapWords(text, room) }

// clip shortens a line to a number of cells and says that it did, keeping the
// front, which is where the id, the label and the first half of a sentence are.
// It is the one cutting rule the whole client goes through, frame included.
func clip(text string, room int) string { return draw.Clip(text, room) }

// labelAt is label in a caller-chosen column.
//
// The new-character form sizes its own column from the widest field name it is
// drawing, which differs per language, so its summary rows have to be told that
// width rather than assume the detail panes' one.
func (m model) labelAt(name string, width int, format string, args ...any) string {
	return m.ctx().LabelAt(name, width, format, args...)
}

// pad widens a string to a column, counting runes, which is the right unit here
// — every letter this client draws, Vietnamese included, is one terminal cell
// wide, and TestEveryWordingIsOneCellPerLetter is what keeps that true.
func pad(text string, width int) string { return draw.Pad(text, width) }
