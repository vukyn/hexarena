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

// The six reference screens live in internal/screen now, because a second
// full-screen client draws the same ones. They are named here under the
// spellings this package already used, for the reason minWidth is: an alias is
// one declaration rather than two, and the model's own fields and this client's
// fixtures go on reading as they read.
//
// ⚠️ They are aliases and not wrappers. A wrapper would be a second place a
// cursor could live, and the whole point of the move is that there is one.
type (
	// browseScreen is the cast listing. ⚠️ A **plain alias**, confirmed rather
	// than assumed: it holds a filter, a cursor and a level and not one field of
	// this client's own — no `from`, nothing of the `screen` enum — so the embed
	// blurbScreen needs would be a struct wrapping nothing.
	browseScreen   = draw.BrowseScreen
	chartScreen    = draw.ChartScreen
	elementsScreen = draw.ElementsScreen
	statusesScreen = draw.StatusesScreen
	speciesScreen  = draw.SpeciesScreen
	buildsScreen   = draw.BuildsScreen
	passivesScreen = draw.PassivesScreen
	// buildRow is one line of the build catalogue, named here because this
	// client's own width fixture builds a catalogue state by hand.
	buildRow = draw.BuildRow
	// previewScreen is the art preview, which moved with the describers. It
	// carries nothing this client owns, so it is an alias like the six above.
	previewScreen = draw.PreviewScreen
	// squadsScreen is the squad builder — the catalogue, one squad under edit
	// and one member of it. ⚠️ A **plain alias** like the browser and the skill
	// listing: it names no view of this client's, because esc is a draw.Back,
	// its two questions are draw.Ask actions carrying their own subject, its two
	// pickers are draw.Pick actions, and the fight it raises with `f` is a
	// draw.Raise at draw.Fight — a Target this package maps to a screen the
	// builder itself may not name.
	squadsScreen = draw.SquadsScreen
	// skillsScreen is the skill book and the form over it. ⚠️ A **plain alias**
	// like the browser: everything it holds is its own — the listing's cursor,
	// the typed filter, the form's fields and its six destinations — and it
	// names no view of this client's, because esc, `?`, the discard question and
	// the six pickers all say what they want in a draw.Action now.
	skillsScreen = draw.SkillsScreen
	// originsScreen is the works catalogue and the add-a-work form over it. ⚠️ A
	// **plain alias** like the skill listing: it was the cleanest of the seven
	// moves — no cursor of another screen's, no raise, no cross-screen read — so
	// what it named of this client was one way back, now a draw.Back, and one
	// question, now a draw.Ask about nothing.
	originsScreen = draw.OriginsScreen
	// playScreen is the battle fought by hand. ⚠️ A **plain alias** like the
	// rest: esc is a draw.Back and `?` is a draw.Raise at draw.Blurb, and the two
	// squads it fights — the one thing it used to read off another screen — are
	// handed to draw.PlayScreen.Open by whoever owns the fight.
	playScreen = draw.PlayScreen
	// blurbScreen is the description screen. ⚠️ A **plain alias now**, and it was
	// the last one that was not: it carried a `from screen` — this binary's own
	// enum, so it could not travel with the describer — for as long as one of its
	// three raisers still wrote m.screen itself. All three return a draw.Raise at
	// draw.Blurb through navigate, so m.raisedFrom already records who raised it
	// and the field had no job left. See updateBlurb for what replaced its three
	// readings.
	blurbScreen = draw.BlurbScreen
)

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
	// ⚠️ Two facts make this exactly what it replaces, a `from screen` field on
	// the statuses screen. screenMenu is the **zero value** of screen, so an
	// unwritten slot already means the menu, which is where esc went from all
	// four of the screens that never raise; and it is cleared **as it is used**,
	// in goBack, because the old field was cleared in the screen's own esc rather
	// than by enter — a reader who came through the menu after coming through a
	// trait must not inherit the trait.
	raisedFrom screen

	// raisedOver is the way back that raisedFrom displaced, and it is what turns
	// one slot into a two-step answer.
	//
	// ⚠️ **The played battle is the first screen in this client that is both
	// raised and a raiser**: the fight opens it and it raises the description
	// screen with `?`. One slot cannot hold both — measured, the moment that
	// screen's `?` started going through navigate: read a description, come back,
	// leave the battle, and esc landed on the **menu** instead of on the fight,
	// because the raise had overwritten the only record there was.
	//
	// ⚠️ **TestAWayBackSurvivesTheScreenItRaised is what holds this**, and it had
	// to be written for it: every other way-back test in this client walks a
	// chain one raise deep, which a single slot answers perfectly. The defect
	// needs the raise **in between** — `fight → p → ? → esc → esc` — so
	// collapsing the push below to `m.raisedFrom = from` leaves the whole client
	// suite green except that one test. A field whose only reader is a hand
	// trace is a field the next person simplifies.
	//
	// ⚠️ **Two is ENOUGH FOR TODAY and is not sufficient by design — and the
	// reason is not the one it looks like.** The longest chain here is catalogue
	// → fight → battle → description, which is **three** pushes, so the third
	// one displaces the catalogue and nothing puts it back. That is sound only
	// because the fight answers esc by writing `screenSquads` **itself** rather
	// than by following a way back, so the displaced entry is one nothing ever
	// reads. Both of those are facts about the screens this client happens to
	// have, not properties of the scheme: convert the fight's esc to a draw.Back,
	// or give a screen that is already two deep a raise of its own, and the third
	// entry starts being read and two slots silently answer with the wrong door.
	// The last leg of that same test is what measures it — it walks out to the
	// catalogue, so the day the fight starts popping the test fails instead of
	// the reader ending up on the menu. A real `[]screen` is the answer then; it
	// was passed over now only because two named fields left all twelve existing
	// raisedFrom sites and every everyScreen entry untouched.
	raisedOver screen

	browse   browseScreen
	form     formScreen
	origins  originsScreen
	skills   skillsScreen
	statuses statusesScreen
	passives passivesScreen
	elements elementsScreen
	species  speciesScreen
	builds   buildsScreen
	squad    squadsScreen
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
	// raised it. It lives here rather than on a screen because three screens
	// raise ten between them — the kit and the species on the new-character
	// form, five allowlists and the inflicts field on the skill form, both
	// halves of a loadout on the squad builder — and a picker owned by a screen
	// would be ten pickers to keep in step.
	//
	// ⚠️ **A pointer, and the pointer is the presence flag.** model.key and
	// model.view both branch on it not being nil, and esc and enter close the
	// list by writing nil, which is what draw.PickState.Update hands back. A
	// value here would need absence carried beside it — the rule logFollow and
	// atb.Queue.Pending are both written under — and (*draw.PickState).Toggle
	// and NextFilter mutate in place, so a value would change that too.
	picker *pickState

	// guard holds the unsaved-changes question while it is being asked. A form
	// with edits in it is the one thing in this program that a stray Escape can
	// destroy, so leaving one asks first.
	guard *guardState
}

// guardState is a pending "are you sure": the wording, the screen that asked it,
// and what it is about.
//
// The question is a key rather than a sentence so that switching language with
// one pending redraws it, instead of leaving the last language's words on the
// screen until the question is answered.
//
// ⚠️ **It used to hold a `func(model) model`**, and that closure is why the two
// screens that carry one — the skill listing and the squad builder — could not
// move into internal/screen: a callback naming `model` names this client, so a
// screen holding one is a screen written for the client it was written in. The
// three fields below are the same mechanism as data, and the shape they take is
// the one this client already applies everywhere else — a state change the
// screen makes to itself, and a draw.Action the client applies for it.
type guardState struct {
	question i18n.Key

	// asked is the screen the question belongs to, and it is what confirmedBy
	// turns back into that screen's Confirmed. A screen rather than a second
	// enum: `screen` already names every view this client has, and a parallel
	// vocabulary for the four of them that ask would be two names for one idea.
	asked screen

	// about is what the question is about, and this client never reads it: it
	// travels from the screen that asked to that screen's own Confirmed, which
	// is the only thing that knows what the value means.
	//
	// ⚠️ **An `any`, and it is draw.Action.About's own type rather than a
	// vocabulary of this client's.** It used to be a guardSubject declared here,
	// because the one screen that asks about anything — the squad builder — was
	// in this package; that screen is in internal/screen now and its two
	// questions are told apart by a draw.SquadsAsk, so the carrier had to be
	// something a screen in that package can fill in. Three of the five confirms
	// still name nothing at all: a form throwing its own draft away is about the
	// screen that asked, and nil is what they pass.
	about any
}

func newModel(lib *forge.Library, lang i18n.Lang) model {
	style := newPalette()
	// The skill form dresses its own text fields, and internal/screen may not
	// read the terminal — so that screen is built from a Context rather than
	// from a library alone, which is where the answer already lives
	// (draw.Palette.Plain). The window is not filled in because nothing a
	// constructor does measures one.
	ctx := draw.Context{Lib: lib, Lang: lang, Style: style, Authoring: true}
	return model{
		lib:      lib,
		lang:     lang,
		style:    style,
		browse:   draw.NewBrowseScreen(lib),
		form:     newFormScreen(lib),
		origins:  draw.NewOriginsScreen(ctx),
		skills:   draw.NewSkillsScreen(ctx),
		statuses: draw.NewStatusesScreen(lib),
		passives: draw.NewPassivesScreen(lib),
		species:  draw.NewSpeciesScreen(lib),
		builds:   draw.NewBuildsScreen(lib),
		squad:    draw.NewSquadsScreen(ctx),
		fight:    newFightScreen(),
		play:     draw.NewPlayScreen(),
		check:    newCheckScreen(lib),
		preview:  draw.NewPreviewScreen(),
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
		// ⚠️ **This client is the one that authors, so it is the one that says
		// so.** draw.Context.Authoring is nought for a read-only client, and
		// nought is the reading a forgotten declaration falls into — which is why
		// the declaration lives on the side that has a suite pressing every one
		// of those keys by name. Dropping it here takes `a` and `e` off the skill
		// listing, `a` off the works catalogue and `n`, `enter` and `d` off the
		// squad catalogue, and reddens the tests that press them rather than
		// going quiet.
		Authoring: true,
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
		return m.answerPicker(message)
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
		browse, action := m.browse.Update(m.ctx(), message)
		m.browse = browse
		return m.navigate(screenBrowse, action)
	case screenNew:
		return m.form.update(m, message)
	case screenOrigins:
		origins, action, command := m.origins.Update(m.ctx(), message)
		m.origins = origins
		return m.navigateWith(screenOrigins, action, command)
	case screenSkills:
		skills, action, command := m.skills.Update(m.ctx(), message)
		m.skills = skills
		return m.navigateWith(screenSkills, action, command)
	// ⚠️ Two mechanisms sit side by side in this switch, and that is deliberate
	// rather than half-finished. The screens that have moved into internal/screen
	// — the cast browser and the works catalogue above, and the six below —
	// return a draw.Action, what they want, and this client decides what it
	// means, which is what lets them stop naming entries of an enum only this
	// binary has. Every other branch still writes m.screen itself and will be
	// converted as it moves.
	case screenStatuses:
		statuses, action := m.statuses.Update(m.ctx(), message)
		m.statuses = statuses
		return m.navigate(screenStatuses, action)
	case screenPassives:
		passives, action := m.passives.Update(m.ctx(), message)
		m.passives = passives
		return m.navigate(screenPassives, action)
	case screenElements:
		elements, action := m.elements.Update(m.ctx(), message)
		m.elements = elements
		return m.navigate(screenElements, action)
	case screenSpecies:
		species, action := m.species.Update(m.ctx(), message)
		m.species = species
		return m.navigate(screenSpecies, action)
	case screenBuilds:
		builds, action := m.builds.Update(m.ctx(), message)
		m.builds = builds
		return m.navigate(screenBuilds, action)
	case screenSquads:
		squad, action, command := m.squad.Update(m.ctx(), message)
		m.squad = squad
		return m.navigateWith(screenSquads, action, command)
	case screenFight:
		return m.fight.update(m, message)
	case screenPlay:
		play, action := m.play.Update(m.ctx(), message)
		m.play = play
		return m.navigate(screenPlay, action)
	case screenCheck:
		return m.check.update(m, message)
	case screenPreview:
		return m.updatePreview(message)
	case screenSpar:
		return m.spar.update(m, message)
	case screenBlurb:
		return m.updateBlurb(message)
	case screenChart:
		chart, action := m.chart.Update(m.ctx(), message)
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
	// The two describers. Both were listed here **before** anything returned a
	// Raise naming either, which is what put them under
	// TestEveryRaiseTargetNamesAScreenInThisClient ahead of their raisers rather
	// than after. All four raisers have arrived since: the cast browser raises
	// both, and the skill listing and the played battle raise the blurb.
	draw.Blurb:   screenBlurb,
	draw.Preview: screenPreview,
	// The squad catalogue raises this one and the fight itself has not moved,
	// which is what a Target is for: a screen in internal/screen may not name a
	// view of a client it was not written for, so it asks for one and this map
	// answers.
	draw.Fight: screenFight,
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
		return m.goBack(), nil
	case draw.Raise:
		return m.raise(from, action)
	case draw.Ask:
		// The asking screen is the one that asked, which is what `from` already
		// is, so nothing has to be read off m.screen — and what the question is
		// about travels with it, opaque, straight back to that screen's own
		// Confirmed. This client never opens it: the squad builder's two
		// questions are told apart by a draw.SquadsAsk and the other three are
		// about nothing.
		return m.ask(action.Question, from, action.About), nil
	case draw.Pick:
		// The screen built the list; the client owns it while it is up. Raise
		// settles the defaults a literal did not fill in, exactly as it does for
		// the four raise sites still in this package.
		return m.pick(action.Picker), nil
	case draw.Answer:
		// ⚠️ **Nothing this client draws can produce one, and the reason is one
		// field.** An Answer is a decision taken on a battle the screen does
		// *not* drive — draw.PlayScreen.Live — and that mode belongs to a PvP
		// match, which is cmd/hexarena-tui's. This client draws PlayScreen in
		// local mode only, where every turn goes through the engine on the way
		// in and there is nothing left to hand a client.
		//
		// So the honest answer is to do nothing rather than to invent somewhere
		// to send it: a decision with no socket behind it would be a keystroke
		// this client claimed to have carried out. It has an arm at all because
		// draw.KindCount is what stops a kind arriving unnamed, and a swallowed
		// Answer is the quietest failure in that list — it looks exactly like a
		// turn nobody has resolved yet.
		return m, nil
	}
	// draw.Stay, which is every keystroke a screen handled without leaving.
	return m, nil
}

// navigateWith is navigate for a screen that hands back a command of its own as
// well as an action.
//
// ⚠️ **Only a screen with a text field on it needs this, and it is why
// draw.SkillsScreen.Update has three returns rather than two.** A bubbles
// textinput answers an Update with a command — the cursor's blink — and dropping
// it leaves the field with no cursor, which is the same fact draw.PickResult
// carries a Cmd for. It is not on draw.Action, deliberately: an Action is a
// comparable value a screen returns and a test writes out as a literal, and a
// func field would take that away from every screen to serve one.
//
// The two commands cannot both be filled — a Quit is the listing's own q, which
// no field is focused for, and a field's blink comes back with the zero action —
// and where they somehow are, the navigation wins, because a program asked to
// end outranks a cursor.
func (m model) navigateWith(from screen, action draw.Action, command tea.Cmd) (tea.Model, tea.Cmd) {
	next, navigated := m.navigate(from, action)
	if navigated != nil {
		return next, navigated
	}
	return next, command
}

// raise opens the screen a target names, about whatever the raiser named.
//
// ⚠️ It declines rather than half-arriving. A subject the raised screen cannot
// land on leaves the reader where they are — the trait naming a status the book
// has lost is a trait already printing a bare id, and a cursor moved to whatever
// sorted next would answer a question nobody asked. A target with no entry in
// raiseTargets declines for the opposite reason: it is a bug rather than a state,
// and the test above is what says so out loud.
//
// ⚠️ **The subject goes through the general applier**, which is the debt #203
// left. What stood here was `if action.Focus != ""` over a focus helper that
// answered only the statuses screen and returned not-found for every other
// target — so a raise carrying a subject anywhere else declined the whole trip in
// silence, and nothing counted the cases the way TargetCount counts these. The
// applier is a map over screen.SubjectKindCount now, so it can be walked.
//
// Direct rather than through enter, which is what the two raises it replaces
// did: neither screen refreshes on the way in, and routing them through enter
// now would be a behaviour change wearing a tidy-up's clothes.
func (m model) raise(from screen, action draw.Action) (tea.Model, tea.Cmd) {
	target, known := raiseTargets[action.Target]
	if !known {
		return m, nil
	}
	handed, landed := m.applySubject(action.Subject)
	if !landed {
		return m, nil
	}
	m = handed
	// ⚠️ A description is opened at its own top, and this is the one place that
	// says so now.
	//
	// It used to be one raiser's own business — the played battle zeroed the
	// scroll in its `?` handler and the other two did not, which was a latent
	// disagreement rather than a decision: a reader who had scrolled a trait
	// description and then asked about a battle option would have opened the
	// second one at the first one's offset. Raising is what starts a new reading,
	// so the reset belongs to the raise.
	if target == screenBlurb {
		m.blurb.Scroll = 0
	}
	m = m.raisedBy(from)
	// ⚠️ **One target goes through enter and the other four do not**, which is a
	// difference rather than an inconsistency to tidy away. enter refreshes the
	// screen being entered, and three of the other four are raised *about*
	// something — a statuses reference refreshed after applySubject has moved its
	// cursor is a cursor put back where it started. The fight holds a cache of
	// runs keyed on the pairing and the seed count, so arriving with the last
	// visit's cache is a screen reporting a squad that has since been edited;
	// that is what its own refresh is for, and it is what `f` from the catalogue
	// has always done.
	if target == screenFight {
		return m.enter(target), nil
	}
	m.screen = target
	return m, nil
}

// raisedBy records a way back, keeping the one it displaces.
//
// ⚠️ It is a push and not an assignment because of raisedOver's own note: a
// screen that was itself raised keeps its door while it raises another.
func (m model) raisedBy(from screen) model {
	m.raisedOver, m.raisedFrom = m.raisedFrom, from
	return m
}

// goBack follows the way back and forgets it, which is what every Back in this
// client comes to.
//
// Forgotten as it is used: the next visit through the menu must not inherit this
// one's way back. What was under it moves up, so the screen being returned to
// still has the door it arrived through.
func (m model) goBack() model {
	m.screen = m.raisedFrom
	m.raisedFrom, m.raisedOver = m.raisedOver, screenMenu
	return m
}

// confirmedBy is what a confirmed guard means to this client: which screen's
// Confirmed answers the question that screen asked.
//
// A map keyed by the asking screen and read by key, never ranged over into
// anything that reaches a screen — the same discipline raiseTargets and
// `subjects` already hold one field over.
//
// ⚠️ It has to be **total over guardAskers**: a screen that can ask and has no
// entry here swallows a confirmed `y` in silence — the question comes down, the
// reader believes they answered it, and nothing happens. That is the same shape
// as a target with no screen and a subject kind with no applier, and
// TestEveryScreenThatAsksAnswersItsOwnQuestion walks the declared list rather
// than this map for the reason those two walk their counts.
var confirmedBy = map[screen]func(model, any) (model, draw.Action){
	screenNew:     confirmForm,
	screenOrigins: confirmOrigins,
	screenSkills:  confirmSkills,
	screenSquads:  confirmSquads,
}

// guardAskers is every screen that raises a guard, written down rather than
// derived.
//
// ⚠️ It is the declared count the dispatch above is held against, and it is a
// list rather than a `screenCount` because most screens never ask: a walk over
// every view would demand a Confirmed from the chart and the menu. Adding a
// question to a screen means adding it here, which is the act that puts the
// screen under the totality test instead of leaving it to be noticed by a reader
// pressing `y` and getting nothing.
var guardAskers = [...]screen{screenNew, screenOrigins, screenSkills, screenSquads}

// confirmForm, confirmOrigins, confirmSkills and confirmSquads are the four
// adapters between the dispatch above and the screens' own Confirmed.
//
// Each is the same three lines — hand the screen the context and what the
// question was about, put what comes back on the model, and give the client the
// action — and they are written out rather than generated because a screen's
// field is named on the model and Go has nowhere else to say which one.
func confirmForm(m model, about any) (model, draw.Action) {
	form, action := m.form.Confirmed(m.ctx(), about)
	m.form = form
	return m, action
}

func confirmOrigins(m model, about any) (model, draw.Action) {
	origins, action := m.origins.Confirmed(m.ctx(), about)
	m.origins = origins
	return m, action
}

func confirmSkills(m model, about any) (model, draw.Action) {
	skills, action := m.skills.Confirmed(m.ctx(), about)
	m.skills = skills
	return m, action
}

func confirmSquads(m model, about any) (model, draw.Action) {
	squad, action := m.squad.Confirmed(m.ctx(), about)
	m.squad = squad
	return m, action
}

// answerGuard resolves a pending confirmation. Anything that is not a yes is a
// no, which is the same default hexforge's own confirmation takes.
//
// The y is the same letter in both languages on purpose: it is what the [y/N]
// on screen offers, and what every other confirmation in this repository takes.
//
// ⚠️ The question is taken off the model **before** anything runs, and what the
// confirm needs travels as an argument. A confirm reading m.guard would be a
// confirm that has to be run while the question it answers is still pending,
// which is an ordering nothing states and one keystroke handler could break.
func (m model) answerGuard(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	guard := *m.guard
	m.guard = nil
	if strings.ToLower(message.String()) != "y" {
		return m, nil
	}
	confirm, known := confirmedBy[guard.asked]
	if !known {
		return m, nil
	}
	answered, action := confirm(m, guard.about)
	// Both halves, applied the way every converted screen's keystroke already is:
	// the screen changed itself, and the client does whatever leaving it costs.
	// One of the five navigates — discarding a half-written character goes back
	// to the menu — and it says so with a draw.Back rather than by writing
	// m.screen from inside a screen's own file.
	return answered.navigate(guard.asked, action)
}

// pickLanding is what one destination means to this client: whose screen it
// belongs to, and the adapter that hands the answer there.
//
// The screen is carried rather than read off m.screen, for the reason `ask`
// takes one: a picker is drawn over whatever raised it, so m.screen happens to
// be the right answer today and would be a coincidence the dispatch rested on.
// It is what navigate is given as the "from", so the day a pick raises another
// screen — which is 5d's question and not this step's — the trip already knows
// where it came from.
type pickLanding struct {
	on   screen
	land func(model, any, pickAnswer) (model, draw.Action)
}

// pickedInto is where each finished pick goes.
//
// A map keyed by the destination and read by key, never ranged over into
// anything that reaches a screen — the discipline raiseTargets, `subjects` and
// confirmedBy already hold.
//
// ⚠️ It has to be **total over pickDestCount**: a destination with no entry here
// swallows a finished pick in silence — the list closes, the reader believes
// they chose, and the field is unchanged. That is the confirmedBy failure again
// and it is easier to make, because there are ten of these across three screens
// rather than four screens with one apiece.
// TestEveryPickDestinationLandsSomewhere walks the count rather than this map,
// for the reason the other three walks do.
//
// ⚠️ Three adapters against ten entries is not a table wanting collapsing. The
// key is the destination because a *destination* is what may go unhandled;
// keying it by screen would make the five allowlists one entry and put the
// question of which field back inside the screen, where nothing counts it.
//
// ⚠️ **The key is an `any` because there are three vocabularies now, and this is
// the one place entitled to know all of them.** Six of the ten destinations
// followed the skill form into internal/screen as draw.SkillsPick and two
// followed the squad builder as draw.SquadsPick; the two that name a screen
// still in this package are pickDest values. PickState carries any of them as
// the `any` it always was, so the map that turns one back into a landing takes
// the same type. TestEveryPickDestinationLandsSomewhere walks **all three**
// counts, which is what stops any of them growing an entry in silence.
var pickedInto = map[any]pickLanding{
	pickIntoKit:             {on: screenNew, land: landOnForm},
	pickIntoSpecies:         {on: screenNew, land: landOnForm},
	draw.SkillsPickElements: {on: screenSkills, land: landOnSkills},
	draw.SkillsPickRoles:    {on: screenSkills, land: landOnSkills},
	draw.SkillsPickWorlds:   {on: screenSkills, land: landOnSkills},
	draw.SkillsPickKinds:    {on: screenSkills, land: landOnSkills},
	draw.SkillsPickWho:      {on: screenSkills, land: landOnSkills},
	draw.SkillsPickInflicts: {on: screenSkills, land: landOnSkills},
	draw.SquadsPickKit:      {on: screenSquads, land: landOnSquads},
	draw.SquadsPickTrait:    {on: screenSquads, land: landOnSquads},
}

// landOnForm, landOnSkills and landOnSquads are the three adapters between the
// dispatch above and the screens' own Picked.
//
// Each is the same three lines — hand the screen the context, the destination
// and the answer, put what comes back on the model, and give the client the
// action — and they are written out rather than generated because a screen's
// field is named on the model and Go has nowhere else to say which one. Exactly
// the shape confirmForm and its three siblings already have.
func landOnForm(m model, into any, answer pickAnswer) (model, draw.Action) {
	form, action := m.form.Picked(m.ctx(), into, answer)
	m.form = form
	return m, action
}

func landOnSkills(m model, into any, answer pickAnswer) (model, draw.Action) {
	skills, action := m.skills.Picked(m.ctx(), into, answer)
	m.skills = skills
	return m, action
}

func landOnSquads(m model, into any, answer pickAnswer) (model, draw.Action) {
	squad, action := m.squad.Picked(m.ctx(), into, answer)
	m.squad = squad
	return m, action
}

// answerPicker hands one keystroke to the picker in front and does whatever it
// asks for.
//
// Three answers, and the pair the picker hands back says which: the list is
// still up, or it came down with nothing, or it came down with an answer for a
// destination. The picker is written back **before** the answer is landed, for
// the reason the guard's is: a landing that ran while the list it is closing was
// still in front would be a screen drawn over by a picker that is finished with.
//
// The command is the picker's own — the chance field's cursor — and is the one
// thing a screen in internal/screen hands back that is not data. See
// draw.PickResult.
func (m model) answerPicker(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	picker, result := m.picker.Update(m.ctx(), message)
	m.picker = picker
	if !result.Answered {
		return m, result.Cmd
	}
	return m.answerPick(result.Into, result.Answer)
}

// answerPick hands a closed picker's answer to the screen the destination names.
//
// Both halves are applied the way every converted keystroke already is: the
// screen changed itself, and the client does whatever leaving it costs. None of
// the ten leaves today — a pick fills in a field and the reader is put back in
// front of the form they were filling — so all ten hand back the zero action,
// and the pair is (screen, action) anyway because that is the shape Update and
// Confirmed have.
//
// ⚠️ **The destination arrives as an `any` and is read here, which is the one
// place entitled to.** draw.PickState carries it and never looks at it — a
// destination names a field of one screen, and only the client in front knows
// which of its screens that is — so the lookup is a map read rather than a type
// assertion now: there are two destination vocabularies (this client's pickDest
// and the skill form's own draw.SkillsPick, which followed that screen), and a
// value in neither is a picker raised by somebody else, which lands nowhere for
// the same reason pickNowhere does.
func (m model) answerPick(carried any, answer pickAnswer) (tea.Model, tea.Cmd) {
	landing, known := pickedInto[carried]
	if !known {
		return m, nil
	}
	picked, action := landing.land(m, carried, answer)
	return picked.navigate(landing.on, action)
}

// ask raises a confirmation, naming the screen that asked and what about.
//
// The screen is passed rather than read off m.screen: a question is a fact about
// who raised it, and taking it from whatever happens to be in front would make
// the dispatch depend on a coincidence that holds today.
func (m model) ask(question i18n.Key, asked screen, about any) model {
	m.guard = &guardState{question: question, asked: asked, about: about}
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
		m.browse = m.browse.Refresh(m.lib)
	case screenCheck:
		m.check = m.check.refresh(m.lib)
	case screenOrigins:
		m.origins = m.origins.Refresh(m.ctx())
	case screenSkills:
		m.skills = m.skills.Refresh(m.ctx())
	case screenStatuses:
		m.statuses = m.statuses.Refresh(m.lib)
	case screenPassives:
		m.passives = m.passives.Refresh(m.lib)
	case screenSpecies:
		m.species = m.species.Refresh(m.lib)
	case screenBuilds:
		m.builds = m.builds.Refresh(m.lib)
	case screenSquads:
		m.squad = m.squad.Refresh(m.ctx())
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
		m.squad = m.squad.Refresh(m.ctx())
		m.fight = m.fight.refresh()
	case screenPlay:
		// A battle is built on the way in rather than lazily in the view: the
		// screen holds a pointer, and building one while drawing would be a
		// redraw with a side effect.
		//
		// ⚠️ **The pairing is handed over here**, which is the one thing that
		// screen used to reach for: it asked the fight which two squads it was
		// between, and the fight is a screen of this client's that has not moved.
		// Which two squads a battle is between is a parameter of opening it, so
		// the client that owns both screens answers once and passes the answer in.
		//
		// The bool is dropped rather than branched on: an empty catalogue hands
		// back two squads with nobody in them, which is exactly what Open refuses,
		// and it says so on the screen. Branching here would put that refusal in
		// two places.
		home, away, _ := m.fight.sides(m)
		m.play = m.play.Open(m.ctx(), home, away)
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
		body, footer = m.browse.View(m.ctx())
	case screenNew:
		body, footer = m.form.view(m)
	case screenOrigins:
		body, footer = m.origins.View(m.ctx())
	case screenSkills:
		body, footer = m.skills.View(m.ctx())
	case screenStatuses:
		body, footer = m.statuses.View(m.ctx())
	case screenPassives:
		body, footer = m.passives.View(m.ctx())
	case screenElements:
		body, footer = m.elements.View(m.ctx())
	case screenSpecies:
		body, footer = m.species.View(m.ctx())
	case screenBuilds:
		body, footer = m.builds.View(m.ctx())
	case screenSquads:
		body, footer = m.squad.View(m.ctx())
	case screenFight:
		body, footer = m.fight.view(m)
	case screenPlay:
		body, footer = m.play.View(m.ctx())
	case screenCheck:
		body, footer = m.check.view(m)
	case screenPreview:
		body, footer = m.preview.View(m.ctx())
	case screenSpar:
		body, footer = m.spar.view(m)
	case screenBlurb:
		body, footer = m.blurb.View(m.ctx())
	case screenChart:
		body, footer = m.chart.View(m.ctx())
	}
	// The picker is drawn over whichever screen raised it, for the same reason
	// it is a sub-screen at all: a list of nineteen does not fit beside a form.
	if m.picker != nil {
		body, footer = m.picker.View(m.ctx())
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

// wrapped is label for a value with no bound on its length: it fills the row and
// carries on underneath, aligned with where the value started.
//
// The rows this draws are variable in number, so it belongs only on a pane that
// can afford that. The form counts its rows to scroll them and must not use it.
func (m model) wrapped(name string, width int, value string) string {
	return m.ctx().Wrapped(name, width, value)
}

// usableWidth is what a row may spend: the window when there is one, and the
// floor before the first size message arrives.
func (m model) usableWidth() int { return m.ctx().UsableWidth() }

// fieldValueRoom is what a form row has left for the one part of it that has no
// length of its own — the ids in an allowlist, the kit and the species on the
// character form — once the marker, the label column, the fixed part of the
// value and the two-space gap before whatever follows have been paid for.
//
// One declaration for **both forms**, in internal/screen beside Pad, LabelAt and
// UsableWidth, which is the rest of what a row is made of. It moved there when
// the skill form did: the character form has not, and a copy on each side of the
// package boundary is the second copy this function exists to have stopped.
func fieldValueRoom(width, labelWidth, spent int) int {
	return draw.FieldValueRoom(width, labelWidth, spent)
}

// wrapWords breaks text on spaces, never mid-word, and never returns nothing.
func wrapWords(text string, room int) []string { return draw.WrapWords(text, room) }

// clamp keeps an index or a level inside its range, and returns the low bound
// when the range is empty.
func clamp(value, low, high int) int { return draw.Clamp(value, low, high) }

// clip shortens a line to a number of cells and says that it did, keeping the
// front, which is where the id, the label and the first half of a sentence are.
// It is the one cutting rule the whole client goes through, frame included.
func clip(text string, room int) string { return draw.Clip(text, room) }

// ⚠️ **Four forwarders went with the squad builder, and one is left.** The rule
// this file follows for pad, clip and clamp is that a forwarder exists for the
// *production* call sites it spares: a body in internal/screen, a call site here
// reading as it read. `window` and `traitSentences` had exactly one caller each
// and it was that screen; `skillLines` and `traitIndent` had lost theirs when the
// multi-select moved and were being kept alive by tests alone, which is a
// declaration for nobody. All four are draw.Window, draw.TraitSentences,
// draw.SkillLines and draw.TraitIndent at the handful of sites that still ask.

// budgetLine is the joint health-and-defence bound drawn as a meter and as
// numbers. It moved with the cast browser, which is the pane it was written for,
// and is forwarded here because the new-character form draws the same row and
// has not moved — the one reading of the three that still has a production
// caller in this package.
func budgetLine(m model, budget forge.Budget) string {
	return draw.BudgetLine(m.ctx(), budget)
}

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
