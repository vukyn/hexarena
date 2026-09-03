package main

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// screen is which view is in front.
//
// ⚠️ **It is this client's own enum and nothing in internal/screen may name
// one**, which is the whole reason a screen there answers a keystroke with a
// draw.Action instead of writing a view: the authoring tool's menu is a
// different menu and its `screen` a different enum, so a screen that named an
// entry of either could only ever live in the client it was written in.
type screen int

const (
	screenMenu screen = iota
	// The seven catalogues, in the order the menu offers them. Every one of them
	// is a screen internal/screen owns, drawn here exactly as the authoring tool
	// draws it — with the three that can author offering none of it.
	screenCast
	screenSkills
	screenElements
	screenTraits
	screenSpecies
	screenWorks
	screenSquads
	// screenBattle is on the menu and is also raised from the squad catalogue
	// with `f`. See pairing.go for which two squads it opens on and why that is
	// the seam the network work replaces.
	screenBattle
	// screenStatuses is raised from the traits listing with `?`, which is where a
	// trait names a status. It is not on the menu: the question it answers is
	// "what is this thing the trait just mentioned", and a reader has that
	// question after reading a trait rather than before.
	screenStatuses
	// screenChart is raised from the elements listing with `g`, for the reason
	// screenStatuses is raised from the traits one: it is the same subject read
	// the other way round — the shape of the chart rather than one element's
	// place in it — and at the floor the window will not hold both.
	screenChart
	// screenBlurb is the description screen, raised from three places with `?`:
	// the cast browser (a character's traits), the skill listing (a skill) and
	// the battle (the option under the cursor). One screen branching on the kind
	// of subject it was handed.
	screenBlurb
	// screenPreview is the art a character shows at the level being walked,
	// raised from the browser with `p`.
	screenPreview
	// The three lobby screens, which are this client's own and are drawn by
	// nothing in internal/screen. → lobby.go for why they live here.
	//
	// screenJoin is reached from the menu — a ninth entry — rather than with a
	// key on the squad catalogue, even though a squad is under a cursor there:
	// that would need a draw.Target both clients must map and cmd/hexforge-tui
	// could only decline, and the squad chooser belongs beside the code and the
	// password it is submitted with anyway.
	screenJoin
	// screenWaiting is the room joined and the second seat still empty.
	screenWaiting
	// screenResult is the match's end, and it is the one screen that draws a
	// wire.Closure — the ending a mirror cannot compute for itself.
	screenResult
	// screenCount is how many views this client has, and it exists so a test can
	// walk them.
	//
	// ⚠️ **TODO.md records five separate occasions where a screen slipped out of
	// the authoring tool's sweep and silently lost its width, translation and
	// leak tests.** A sweep is a map somebody remembered to write in; this is the
	// count a walk can be held against, so a screen added below without an entry
	// in the sweep — or without a written reason for staying out of it — is a red
	// test rather than a screen nothing measures.
	// TestEveryScreenThisClientDrawsIsSwept is that walk.
	screenCount
)

// The smallest window the screens fit in — 120x24, declared in internal/screen
// with the measurement that asked for it, because screen.Context.UsableWidth is
// what spends it and both clients draw the same screens against the same floor.
const (
	minWidth  = draw.MinWidth
	minHeight = draw.MinHeight
)

// menuItem is one entry of the top-level view.
type menuItem struct {
	label  i18n.Key
	detail i18n.Key
	target screen
}

// menuItems is the seven catalogues a player reads, and a battle.
//
// ⚠️ **Most of the wordings are the authoring tool's own and three are not**,
// and which three is the whole shape of this client: a listing of the cast is a
// listing of the cast, but "add or edit one" is not something this menu can
// offer, so the skills, works and squads entries carry their own detail lines.
// A wording naming a key this client ignores is the failure readonly_test.go
// exists to measure, and a menu detail is as much a promise as a footer is.
//
// ⚠️ **The build catalogue is deliberately not here.** It is the eighth listing
// internal/screen owns and a reader would want it; it is left out because this
// client's menu is the seven the step that built it asked for, and adding an
// eighth is a decision about what a game client offers rather than a line of
// wiring. Nothing else in this package mentions draw.BuildsScreen, so it is a
// gap rather than a half-finished screen — see TODO.md.
var menuItems = []menuItem{
	{i18n.MenuCast, i18n.MenuCastDetail, screenCast},
	{i18n.MenuSkills, i18n.GameMenuSkillsDetail, screenSkills},
	{i18n.MenuElements, i18n.MenuElementsDetail, screenElements},
	{i18n.MenuPassives, i18n.MenuPassivesDetail, screenTraits},
	{i18n.MenuSpecies, i18n.MenuSpeciesDetail, screenSpecies},
	{i18n.MenuOrigins, i18n.GameMenuWorksDetail, screenWorks},
	{i18n.MenuSquads, i18n.GameMenuSquadsDetail, screenSquads},
	{i18n.GameMenuBattle, i18n.GameMenuBattleDetail, screenBattle},
	{i18n.GameMenuJoin, i18n.GameMenuJoinDetail, screenJoin},
}

// model is the whole program: a library, the language, the screen in front, and
// each screen's own state.
//
// It knows no rules and no wording. Every fact it draws comes out of
// internal/forge and every sentence out of internal/i18n, which is what makes
// two front-ends over one engine incapable of disagreeing about either.
//
// ⚠️ **Every screen field is a type internal/screen owns**, and none of them is
// wrapped. A wrapper would be a second place a cursor could live, which is
// exactly what the eleven-step extraction was for.
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
	// screenMenu is the zero value, so an unwritten slot already means the menu —
	// which is where esc goes from every screen the menu itself opens.
	raisedFrom screen
	// raisedOver is the way back that raisedFrom displaced, and it is what turns
	// one slot into a two-step answer.
	//
	// ⚠️ **The battle is both raised and a raiser**: the squad catalogue opens it
	// and it raises the description screen with `?`. One slot cannot hold both —
	// squads → battle → description → esc → esc lands on the menu instead of on
	// the catalogue, because the raise overwrote the only record there was. That
	// is the defect #228 measured in the authoring tool, arriving here for free
	// because this client has the same two-deep chain, and
	// TestAWayBackSurvivesTheScreenItRaised is what holds it.
	//
	// ⚠️ **Two is enough for this client and is not sufficient by design.** The
	// longest chain here is catalogue → battle → description, which is two
	// pushes; a screen that is already two deep growing a raise of its own is
	// where a real `[]screen` becomes the answer.
	raisedOver screen

	// taking is which row of the squad catalogue the reader is taking into a
	// battle, and it is the one field this client keeps that neither the
	// catalogue nor the battle screen holds.
	//
	// A row rather than an id, for the reason the authoring tool's fight keeps
	// one: an index into the catalogue is a fact about *this client's* two
	// screens standing next to each other, which a screen in internal/screen may
	// not know. The raise names an id and subject.go turns it into this. See
	// pairing.go for what the row is turned into.
	taking int

	cast     draw.BrowseScreen
	skills   draw.SkillsScreen
	elements draw.ElementsScreen
	traits   draw.PassivesScreen
	species  draw.SpeciesScreen
	works    draw.OriginsScreen
	squads   draw.SquadsScreen
	battle   draw.PlayScreen
	statuses draw.StatusesScreen
	// chart holds nothing: it is drawn from the library every time and has no
	// cursor. It is a field anyway so the screen is dispatched to like every
	// other one rather than being a special case in three switches.
	chart   draw.ChartScreen
	blurb   draw.BlurbScreen
	preview draw.PreviewScreen

	// The three this client owns outright. → lobby.go.
	join    joinScreen
	waiting waitingScreen
	result  resultScreen

	// session is the PvP side: the socket, the goroutine running Play, and the
	// two channels between it and this model.
	//
	// ⚠️ **A pointer, and it is the one field on this model that is not copied
	// with it.** The model is a value handed back on every keystroke, so a
	// session copied with it would be a second set of channels and a second
	// idea of which match is live. It is also the reason draw.PlayScreen's own
	// note about holding a pointer the model does not copy now has a
	// neighbour — and the discipline is the same: the battle behind it is read
	// under a lock and never kept.
	session *session
}

func newModel(lib *forge.Library, lang i18n.Lang, sess *session) model {
	style := newPalette()
	// Two of these screens are built from a Context rather than from a library
	// alone, because they dress text fields and internal/screen may not read the
	// terminal — so the answer comes off the Palette it is handed. This client
	// never opens either field: the skill form and the add-a-work form are
	// reached through keys it does not offer. They are built the same way anyway,
	// because a screen half-constructed is a screen that draws differently from
	// the one the other client draws.
	//
	// ⚠️ **Authoring is not set, and that is the whole of this client's
	// read-only-ness.** draw.Context.Authoring is nought here and in ctx below,
	// which is the reading a client that cannot write falls into without
	// declaring anything — so there is nothing to forget. See the field's own
	// comment for why the safe half is the zero.
	ctx := draw.Context{Lib: lib, Lang: lang, Style: style}
	return model{
		lib:      lib,
		lang:     lang,
		style:    style,
		cast:     draw.NewBrowseScreen(lib),
		skills:   draw.NewSkillsScreen(ctx),
		traits:   draw.NewPassivesScreen(lib),
		species:  draw.NewSpeciesScreen(lib),
		works:    draw.NewOriginsScreen(ctx),
		squads:   draw.NewSquadsScreen(ctx),
		battle:   draw.NewPlayScreen(),
		statuses: draw.NewStatusesScreen(lib),
		preview:  draw.NewPreviewScreen(),
		join:     newJoinScreen(),
		session:  sess,
	}
}

func (m model) Init() tea.Cmd { return nil }

// ctx is what a screen draws with: the books, the language, the palette and the
// window, and nothing this model owns.
//
// Built per call rather than kept as a field: the model is a value copied on
// every keystroke, so a stored copy would be a second place the window size
// lives.
func (m model) ctx() draw.Context {
	return draw.Context{
		Lib:   m.lib,
		Lang:  m.lang,
		Style: m.style,
		Width: m.width, Height: m.height,
	}
}

// text is one line in the language in front. Every screen goes through it.
func (m model) text(key i18n.Key, args ...any) string { return m.lang.Say(key, args...) }

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.key(typed)
	case matchJoinedMsg:
		// The tick is armed with the match rather than with the first turn: a
		// countdown that started on the turn it first drew would be a clock that
		// only runs once somebody has already waited. → clockTick.
		return m.joined(typed.client), clockTick()
	case matchFailedMsg:
		m.join = m.join.Failed(typed.err)
		// The dial is over however it went, so the session is disarmed here
		// rather than left holding a cancel nobody will call.
		m.session.leave()
		return m, nil
	case matchSteppedMsg, matchAskingMsg:
		// ⚠️ **Both are redraw triggers carrying nothing** and both come to the
		// same thing: read the mirror and re-attach. matchAskingMsg is not a
		// separate arm because "it is your turn" is not a fact this model keeps
		// — socket.Mirror.Asking is the only derivation there is, and it is read
		// under the lock like everything else.
		return m.stepped(), nil
	case clockTickMsg:
		// ⚠️ **The same redraw, and the re-arm is what makes it a clock.** A tick
		// is one Cmd, so a tick that did not ask for the next one would move the
		// countdown exactly once; and re-arming only while a match is live is
		// what stops the process ticking for ever after the reader has left one.
		if !m.session.live() {
			return m, nil
		}
		return m.stepped(), clockTick()
	case matchEndedMsg:
		return m.ended(), nil
	}
	return m, nil
}

// joined is the dial having got past the gate: the session takes the client and
// starts its loop, and the reader lands on the waiting screen.
//
// The welcome is read here rather than waited for, and that is measured: it
// arrives during the handshake, before Play exists, so it is on the mirror the
// moment Dial returns and no step has to happen for the room's format to be
// drawable. → socket's TestSteppedIsCalledForEveryMessageAndHoldsNoLock.
func (m model) joined(client *socket.Client) model {
	m.session.begin(client)
	m.waiting.Code, m.waiting.Seat = client.Code(), client.Seat()
	m.waiting.Welcome, m.waiting.Seated = client.Mirror().Welcome()
	m.join.Dialling, m.join.At = false, ""
	m.screen = screenWaiting
	return m
}

// stepped re-reads the mirror and points the battle screen at whatever it found.
//
// ⚠️ **Everything about the battle happens inside the callback**, which is the
// whole discipline this client is under: session.read runs it under the mirror's
// read lock, and a *battle.Battle handed out of it would be a pointer into a
// battle the Play goroutine is stepping.
func (m model) stepped() model {
	m.session.read(func(sight socket.Sight) {
		if sight.Fight != nil {
			m.battle = m.battle.Attach(m.ctx(), liveOf(sight, m.session.countdown(sight)))
			if m.screen == screenWaiting {
				m.screen = screenBattle
			}
		}
		if sight.Over {
			m.result = resultOf(sight)
		}
	})
	return m
}

// ended is Play having returned: the reader lands on the result, with whatever
// the loop returned beside the standing.
//
// ⚠️ **It reads the mirror one last time rather than trusting what the last step
// left**, because the ending itself arrives as a message: the wire.Closed a
// departure sends is taken in by Receive, and the step that carried it is the
// one immediately before Play returns.
func (m model) ended() model {
	m.session.read(func(sight socket.Sight) { m.result = resultOf(sight) })
	if err, done := m.session.outcome(); done {
		m.result.Err = err
	}
	m.screen = screenResult
	return m
}

// liveOf is a mirror's reading turned into what the battle screen needs of it.
//
// ⚠️ **The refusal is carried as a NAME**, which is what keeps internal/wire out
// of internal/screen: the screen draws it through i18n.Lang.Refusal, whose whole
// signature exists so that a shared drawing package never has to know the
// protocol. The **latest** one, because a refusal does not end a connection —
// several can arrive in a match, and what a player needs is the one that just
// happened.
// ⚠️ **The countdown is a parameter rather than something read here**, and that
// is the same division: this turns a reading into a drawing, and what a clock
// says is not on the reading. → clock.go, which is where every clock in this
// package is.
func liveOf(sight socket.Sight, clock draw.PlayClock) draw.PlayLive {
	live := draw.PlayLive{
		Fight:  sight.Fight,
		Asking: sight.Asking,
		Side:   sight.Side,
		Seed:   sight.Seed,
		Clock:  clock,
	}
	if len(sight.Refusals) > 0 {
		live.Refusal = sight.Refusals[len(sight.Refusals)-1].String()
	}
	return live
}

// resultOf is a mirror's reading turned into the result screen.
//
// ⚠️ **The fought list is COPIED out of the sight.** Mirror.Read's own rule is
// that nothing it hands over may outlive the call, and this screen keeps what it
// is given until the reader leaves it.
func resultOf(sight socket.Sight) resultScreen {
	result := resultScreen{Fought: slices.Clone(sight.Fought)}
	if sight.Closure.Closes() {
		result.Closure = sight.Closure.String()
	}
	return result
}

// key routes one keystroke.
//
// ctrl+c is handled before anything else and from every screen: a program that
// can trap somebody is a program they have to kill from another terminal. ctrl+l
// is next and works from everywhere for a smaller reason of the same shape — the
// two languages are only worth comparing side by side if the comparison costs
// nothing.
//
// ⚠️ **It is a chord rather than a letter for a reason this client does not
// have and keeps anyway.** In the authoring tool a bare `l` is text on a form;
// here no field is ever focused, so a letter would do. The two clients toggling
// on different keys would be two answers to one question, and the keys are named
// in a catalog both of them read.
func (m model) key(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+l":
		m.lang = m.lang.Other()
		return m, nil
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
	case screenCast:
		next, action := m.cast.Update(m.ctx(), message)
		m.cast = next
		return m.navigate(screenCast, action)
	case screenSkills:
		next, action, command := m.skills.Update(m.ctx(), message)
		m.skills = next
		return m.navigateWith(screenSkills, action, command)
	case screenElements:
		next, action := m.elements.Update(m.ctx(), message)
		m.elements = next
		return m.navigate(screenElements, action)
	case screenTraits:
		next, action := m.traits.Update(m.ctx(), message)
		m.traits = next
		return m.navigate(screenTraits, action)
	case screenSpecies:
		next, action := m.species.Update(m.ctx(), message)
		m.species = next
		return m.navigate(screenSpecies, action)
	case screenWorks:
		next, action, command := m.works.Update(m.ctx(), message)
		m.works = next
		return m.navigateWith(screenWorks, action, command)
	case screenSquads:
		next, action, command := m.squads.Update(m.ctx(), message)
		m.squads = next
		return m.navigateWith(screenSquads, action, command)
	case screenBattle:
		next, action := m.battle.Update(m.ctx(), message)
		m.battle = next
		return m.navigate(screenBattle, action)
	case screenStatuses:
		next, action := m.statuses.Update(m.ctx(), message)
		m.statuses = next
		return m.navigate(screenStatuses, action)
	case screenChart:
		next, action := m.chart.Update(m.ctx(), message)
		m.chart = next
		return m.navigate(screenChart, action)
	case screenBlurb:
		return m.updateBlurb(message)
	case screenPreview:
		return m.updatePreview(message)
	case screenJoin:
		next, action, command := m.join.Update(m.ctx(), message)
		dial := m.dialling(next)
		m.join = next
		if dial != nil {
			// ⚠️ The dial replaces the field's own blink, and that is right
			// rather than a loss: a screen with a network round trip in flight
			// takes no keystrokes, so there is no cursor to keep alive.
			return m, dial
		}
		return m.navigateWith(screenJoin, action, command)
	case screenWaiting:
		return m.updateWaiting(message)
	case screenResult:
		return m.updateResult(message)
	}
	return m, nil
}

// dialling is the command a join screen asked for by turning Dialling on, and
// nil the rest of the time.
//
// ⚠️ **The screen says which room and the client owns the socket**, which is the
// same division every raise in this program is under: a screen names what it
// wants and asks nobody how it is done. It is read as a *transition* — off, then
// on — so a redraw while a dial is in flight does not open a second one.
func (m model) dialling(next joinScreen) tea.Cmd {
	if m.join.Dialling || !next.Dialling {
		return nil
	}
	squad, have := next.Chosen()
	if !have {
		return nil
	}
	// ⚠️ **The mirror is built from the EMBEDDED books, whatever --data says.**
	// wire.Version's digest is over the embedded files, so a client that fought
	// on an edited directory would pass a gate promising the two peers simulate
	// the same battle and then fight a different one — a divergence on turn one,
	// reported correctly and confusingly. Refusing to pass the digest's own
	// promise on to the battle is the whole reason the digest exists. The join
	// screen says so when the two really differ. → i18n.JoinDataEdited.
	books, err := seed.Books()
	if err != nil {
		m.join.Err = err
		return nil
	}
	version, err := wire.Local(buildString())
	if err != nil {
		m.join.Err = err
		return nil
	}
	return m.session.dial(next.At, wire.Hello{
		Version:  version,
		Squad:    squad,
		Password: wire.Password(next.Password.Value()),
	}, books)
}

// updateWaiting and updateResult are the two lobby screens with no keys of their
// own beyond leaving.
//
// esc on the waiting screen **leaves the room**, which costs nothing and is not
// asked about: nobody forfeits, so a confirmation would be this client inventing
// a cost the design refused. → session.leave.
func (m model) updateWaiting(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if message.String() == "esc" {
		return m.leaveMatch(), nil
	}
	return m, nil
}

func (m model) updateResult(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if message.String() == "esc" {
		// The match is already over, so this is only tidying: the session is
		// disarmed and the battle screen goes back to being a local one.
		return m.leaveMatch(), nil
	}
	return m, nil
}

// leaveMatch ends whatever match is joined and puts the reader back on the menu.
//
// The battle screen is replaced rather than merely marked, because a live
// PlayScreen holds the mirror's battle: keeping it would leave this client
// drawing a board nobody is stepping any more.
func (m model) leaveMatch() model {
	m.session.leave()
	m.battle = draw.NewPlayScreen()
	m.screen = screenMenu
	m.raisedFrom, m.raisedOver = screenMenu, screenMenu
	return m
}

// raiseTargets is what a draw.Target means to this client.
//
// A map keyed by target and read by key, never ranged over into anything that
// reaches a screen — the same discipline internal/core holds about map order,
// one layer up.
//
// ⚠️ It has to be **total**: a target with no entry makes a raise silently do
// nothing, which is the shape of defect TODO.md records five times as a screen
// slipping out of a sweep.
// TestEveryRaiseTargetNamesAScreenInThisClient walks screen.TargetCount rather
// than this map, so a target added over there fails here instead of going quiet.
//
// ⚠️ **draw.Fight is the entry the two clients answer differently**, and that is
// what a Target is for rather than a hole in the scheme: in the authoring tool
// it means "pick a second squad and measure the pairing", and here it means
// "take this squad into a battle". See pairing.go.
var raiseTargets = map[draw.Target]screen{
	draw.Chart:    screenChart,
	draw.Statuses: screenStatuses,
	draw.Blurb:    screenBlurb,
	draw.Preview:  screenPreview,
	draw.Fight:    screenBattle,
}

// navigate applies what a screen asked for.
//
// from is the screen that asked, which is the whole of what a Raise has to
// record: Back is the answer to "how did I get here", and only the client can
// know it.
func (m model) navigate(from screen, action draw.Action) (tea.Model, tea.Cmd) {
	switch action.Kind {
	case draw.Quit:
		// ⚠️ **This client ends there, and the day a match was a thing two
		// people were in the middle of has come without this line changing.**
		// The comment here used to predict the opposite — "this is the one line
		// that changes" — and it was wrong for a reason worth keeping rather
		// than deleting: **leaving mid-match costs nothing by design.** Nobody
		// forfeits (→ README.md § Nobody forfeits: "a player who is losing can
		// leave at no cost… the enforcement is social"), a departure announces
		// and ends the match as abandoned, and neither seat is charged with
		// anything. A confirmation would be this client inventing a cost the
		// design deliberately refused.
		//
		// What a quit mid-match does owe is that the Play goroutine stops, and
		// that is not this line's job either: run's own `defer sess.leave()`
		// fires however program.Run returns, which is what makes it a guarantee
		// rather than a thing to remember here.
		//
		// A hot-seat battle is likewise a pure function of its seed and the
		// decisions taken, so abandoning one costs nothing `ctrl+s` could not
		// have written out.
		return m, tea.Quit
	case draw.Back:
		// ⚠️ **esc out of a live battle leaves the MATCH rather than going back
		// one screen**, and that is the one place a Back means something this
		// client had to decide: the screen behind a match is the room it was
		// joined from, which no longer exists. Leaving costs nothing and is not
		// asked about — nobody forfeits — so this is the whole of it.
		if from == screenBattle && m.battle.Live {
			return m.leaveMatch(), nil
		}
		return m.goBack(), nil
	case draw.Raise:
		return m.raise(from, action)
	case draw.Answer:
		// The one keystroke a battle is about, on its way to the chooser Play is
		// blocked on. It **never blocks**: nobody asking means the answer is
		// dropped, which is right, because there is no turn for it to be about.
		// → session.answer.
		m.session.answer(action.Answer)
		return m, nil
	case draw.Ask, draw.Pick:
		// ⚠️ **Nothing this client draws can ask for either, and that is measured
		// rather than assumed.** Both are the authoring half of the vocabulary: an
		// Ask is what a form puts to a reader before it throws a draft away, and a
		// Pick is the list a form fills a field from. Every screen that raises one
		// does it from a mode this client cannot enter, because the keys that open
		// those modes are the ones draw.Context.Authoring turns off.
		//
		// So the honest answer is to do nothing rather than to keep a guard field
		// and a picker field that no keystroke can reach — a modal nobody can open
		// is a modal nobody maintains, and it would draw over the screen in front
		// the first time something did reach it.
		// TestNoScreenInThisClientAsksOrPicks presses every key each of those
		// screens answers to and asserts neither kind ever comes back, which is the
		// claim this arm rests on; TestEveryActionKindIsAppliedByThisClient is what
		// stops a seventh kind arriving here unnamed.
		return m, nil
	}
	// draw.Stay, which is every keystroke a screen handled without leaving.
	return m, nil
}

// navigateWith is navigate for a screen that hands back a command of its own as
// well as an action.
//
// ⚠️ **Only a screen with a text field on it needs this**, which is why three
// of this client's screens have three-return Updates: a bubbles textinput
// answers an Update with the cursor's blink, and dropping it leaves the field
// with no cursor. This client focuses none of those fields — they are all on
// forms it does not open — so the command is always nil today. It is carried
// anyway, because the signature is the screen's and a client that dropped half
// of what a screen hands back would be a client the next field breaks.
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
// land on leaves the reader where they are — a trait naming a status the book
// has lost is a trait already printing a bare id, and a cursor moved to whatever
// sorted next would answer a question nobody asked. A target with no entry in
// raiseTargets declines for the opposite reason: it is a bug rather than a
// state, and the test above is what says so out loud.
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
	// A description is opened at its own top, and this is the one place that says
	// so: raising is what starts a new reading, so a reader who had scrolled a
	// trait's sentences and then asked about a battle option would otherwise open
	// the second one at the first one's offset.
	if target == screenBlurb {
		m.blurb.Scroll = 0
	}
	m = m.raisedBy(from)
	// ⚠️ **The battle goes through enter and the other four do not.** enter is
	// what builds the battle from the pairing, and a battle is built on the way in
	// rather than lazily in the view — the screen holds a pointer, and building
	// one while drawing would be a redraw with a side effect. The other four are
	// raised *about* something, and three of them have had a cursor moved by
	// applySubject, which a refresh would put back where it started.
	if target == screenBattle {
		return m.enter(target), nil
	}
	m.screen = target
	return m, nil
}

// raisedBy records a way back, keeping the one it displaces.
//
// It is a push and not an assignment because of raisedOver's own note: a screen
// that was itself raised keeps its door while it raises another.
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
		return m.enterUnlessInAMatch(menuItems[m.menu].target), nil
	}
	return m, nil
}

// enter switches screens, giving the one being entered the chance to refresh
// against a library that may have been written to since it was last drawn.
//
// ⚠️ **Nothing this client runs writes to those books**, so a refresh here is
// not guarding against its own edits: it is guarding against the authoring tool
// having been run beside it, which is the ordinary way somebody plays a squad
// they have just built. That is the same reading the authoring tool's own
// refresh takes and it costs a slice copy.
func (m model) enter(target screen) model {
	m.screen = target
	switch target {
	case screenCast:
		m.cast = m.cast.Refresh(m.lib)
	case screenSkills:
		m.skills = m.skills.Refresh(m.ctx())
	case screenTraits:
		m.traits = m.traits.Refresh(m.lib)
	case screenSpecies:
		m.species = m.species.Refresh(m.lib)
	case screenWorks:
		m.works = m.works.Refresh(m.ctx())
	case screenSquads:
		m.squads = m.squads.Refresh(m.ctx())
	case screenBattle:
		// The catalogue is re-read before the pairing is taken off it, because the
		// pairing is read out of that list rather than off this screen — the same
		// reading every other listing here does on the way in, rather than a
		// second copy of it. A reader arriving from the menu without having
		// visited the catalogue has to be handed a list all the same.
		m.squads = m.squads.Refresh(m.ctx())
		home, away := m.pairing()
		m.battle = m.battle.Open(m.ctx(), home, away)
	case screenJoin:
		// The catalogue is re-read for the reason the battle re-reads it: the
		// authoring tool is the thing that writes squads.json and the ordinary
		// way somebody plays a side is to have just built it.
		m.squads = m.squads.Refresh(m.ctx())
		m.join = m.join.Refresh(m.ctx(), m.squads.Saved)
	}
	return m
}

// enter is asked before either of the two screens a match owns, because a match
// is a thing two people are in the middle of.
//
// ⚠️ **This is one of the two risks of drawing a live battle and a hot-seat one
// on ONE model field.** A player cannot be in both at once, so one
// draw.PlayScreen serves both — but the menu still offers a battle, and
// `enter(screenBattle)` from the menu while a match is live would Open a
// hot-seat game over the top of the mirror's: the reader would be playing
// themselves while a person on another machine waited for a turn. The join
// entry is refused for the mirror image of the same reason — a second Dial
// would orphan the first socket. Both go to the match instead, which is where
// the reader was trying to get back to anyway.
func (m model) enterUnlessInAMatch(target screen) model {
	if !m.session.live() || (target != screenBattle && target != screenJoin) {
		return m.enter(target)
	}
	if m.battle.Live && m.battle.Fight != nil {
		m.screen = screenBattle
		return m
	}
	m.screen = screenWaiting
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

// parts is the body and the footer of whatever is in front, which is the pair
// every screen in internal/screen answers with.
//
// It is separate from screenContent because the two halves are two different
// promises and one of them is measured on its own: a footer names the keys, and
// a footer naming a key this client ignores is the failure readonly_test.go
// exists to catch. Framed together they are one string with the footer's own
// line no longer identifiable.
func (m model) parts() (body, footer string) {
	switch m.screen {
	case screenMenu:
		return m.viewMenu(), m.text(i18n.MenuFooter)
	case screenCast:
		return m.cast.View(m.ctx())
	case screenSkills:
		return m.skills.View(m.ctx())
	case screenElements:
		return m.elements.View(m.ctx())
	case screenTraits:
		return m.traits.View(m.ctx())
	case screenSpecies:
		return m.species.View(m.ctx())
	case screenWorks:
		return m.works.View(m.ctx())
	case screenSquads:
		return m.squads.View(m.ctx())
	case screenBattle:
		return m.battle.View(m.ctx())
	case screenStatuses:
		return m.statuses.View(m.ctx())
	case screenChart:
		return m.chart.View(m.ctx())
	case screenBlurb:
		return m.blurb.View(m.ctx())
	case screenPreview:
		return m.preview.View(m.ctx())
	case screenJoin:
		return m.join.View(m.ctx())
	case screenWaiting:
		return m.waiting.View(m.ctx())
	case screenResult:
		return m.result.View(m.ctx())
	}
	return "", ""
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
	body, footer := m.parts()
	return m.frame(body, footer)
}

// viewTooSmall is what a window that cannot hold a screen gets instead of a
// mangled one.
//
// It names both sizes, because "too small" without a target is a message nobody
// can act on, and it points at the **other** front-end: cmd/hexarena needs no
// room at all and plays the same battles. It is drawn in the language in front,
// which is why it is a catalog entry rather than a fallback in English — the
// person who cannot read the screen is exactly the person who needs this line.
func (m model) viewTooSmall() string {
	lines := strings.Split(
		m.text(i18n.GameTerminalTooSmall, minWidth, minHeight, m.width, m.height), "\n")
	// Cut through clip for the reason frame does, and this is the one screen
	// where a cut is close to certain: it is only ever drawn in a window already
	// too narrow for anything else.
	for i, line := range lines {
		lines[i] = draw.Clip(line, m.width)
	}
	return strings.Join(lines, "\n")
}

// frame puts the header and the footer around a screen's body and pads the
// whole thing to the window's height, so a shorter screen does not leave the
// previous one's tail on display.
//
// ⚠️ **It is cmd/hexforge-tui's frame, deliberately the same arithmetic.** Every
// Room helper in internal/screen budgets `Height - 4` against exactly this
// shape — two rows for the header pair, one blank and one footer — so a client
// framing its screens differently would be a client whose screens budget for
// somebody else's frame. That is a mirror rather than a declaration, which is
// why both clients' goldens record the framed result.
func (m model) frame(body, footer string) string {
	header := m.style.Title.Render(programName) + m.style.Dim.Render("  "+m.lib.Dir())
	lines := []string{header, ""}
	lines = append(lines, strings.Split(body, "\n")...)

	// Two lines for the header, one blank before the footer, one for the footer.
	// Anything past that is cut rather than allowed to push the footer off the
	// bottom: the footer is where the keys are, and a screen whose keys have
	// scrolled away is a screen nobody can leave.
	room := m.height - 2
	if len(lines) > room {
		lines = lines[:room]
		lines[room-1] = m.style.Dim.Render(m.text(i18n.Truncated))
	}
	for len(lines) < room {
		lines = append(lines, "")
	}
	lines = append(lines, m.style.Footer.Render(footer))

	// Clip every line to the window rather than letting a long one wrap. A
	// biography or a filesystem path is free text of any length, and a wrapped
	// line pushes everything below it down by one — which moves the footer off
	// the bottom and makes the screen disagree with the line count above. The cut
	// **says so**, which is what draw.Clip is for: a truncated sentence that does
	// not admit it is worse than one that does, because a reader cannot tell it
	// from a complete one.
	for i, line := range lines {
		lines[i] = draw.Clip(line, m.width)
	}
	return strings.Join(lines, "\n")
}

// menuLabelWidth is the column the menu's own labels sit in, measured rather
// than declared: "danh sách nhân vật" is 18 cells against "cast" at 4, so one
// number for both languages either cuts one or wastes a fifth of the row in the
// other.
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
		// Padded before it is styled, not after: a style is escape codes, and fmt
		// would count those toward the column.
		label := draw.Pad(m.text(item.label), width)
		if i == m.menu {
			// The marker is the selection. The style only agrees with it.
			marker = "> "
			label = m.style.Selected.Render(label)
		}
		out.WriteString(marker + label + " " + m.style.Dim.Render(m.text(item.detail)) + "\n")
	}
	out.WriteString("\n" + m.style.Dim.Render(m.text(i18n.GameMenuNote)))
	return out.String()
}
